package main

// k_router — Go
// Coordinates all 8 modules. Runs the pipeline per turn.
// Starts background daemons (forgetting, the_unknown).
// Reads from stdin, writes K's response (or silence) to stdout.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

var rootDir string

func init() {
	ex, _ := os.Executable()
	rootDir = filepath.Dir(filepath.Dir(ex))
	// If running from source (go run), walk up
	if _, err := os.Stat(filepath.Join(rootDir, "talk.py")); err != nil {
		rootDir, _ = os.Getwd()
		for rootDir != "/" {
			if _, err := os.Stat(filepath.Join(rootDir, "talk.py")); err == nil {
				break
			}
			rootDir = filepath.Dir(rootDir)
		}
	}
}

func redisConnect() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb = redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[router] redis unavailable: %v — continuing without persistence\n", err)
	}
}

// runModule pipes input to a binary's stdin, returns trimmed stdout
func runModule(binary string, input string) string {
	cmd := exec.Command(binary)
	cmd.Stdin  = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[router] %s error: %v\n", binary, err)
		return ""
	}
	return strings.TrimSpace(out.String())
}

// runPython calls a Python script with stdin JSON
func runPython(script string, payload interface{}) string {
	data, _ := json.Marshal(payload)
	cmd := exec.Command("python3", script)
	cmd.Stdin  = bytes.NewReader(data)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[router] python %s error: %v\n", script, err)
		return ""
	}
	return strings.TrimSpace(out.String())
}

// runProlog calls mirror.pl with a Prolog term
func runProlog(plFile string, term string) string {
	cmd := exec.Command("swipl",
		"-q",
		"-g", "main",
		"-t", "halt",
		plFile,
	)
	cmd.Stdin  = strings.NewReader(term + ".\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return strings.TrimSpace(out.String())
}

func redisGet(key, fallback string) string {
	if rdb == nil {
		return fallback
	}
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return fallback
	}
	return val
}

func redisSet(key, val string) {
	if rdb != nil {
		rdb.Set(ctx, key, val, 0)
	}
}

func redisPush(key, val string, maxLen int64) {
	if rdb != nil {
		rdb.LPush(ctx, key, val)
		rdb.LTrim(ctx, key, 0, maxLen-1)
	}
}

// Map perception class to exposure events
func perceptionToEvents(class string, silenceDecision string) []string {
	var events []string
	switch class {
	case "reveals_refusal":
		events = append(events, "refusal_of_question")
	case "reveals_mirror":
		events = append(events, "question_returned")
	case "reveals_search":
		// persistence detected if they keep asking
		turnCount, _ := strconv.Atoi(redisGet("k:turn:count", "0"))
		if turnCount > 2 {
			events = append(events, "persistence_detected")
		}
	}
	// silence broken by user (they spoke at all)
	events = append(events, "silence_broken_by_you")

	// k_deflects_successfully fires only when K SPOKE a deflecting question
	// (state=control, output=SPEAK) — not when K stays silent.
	// Silence is not deflection. Silence is the baseline.
	if silenceDecision == "SPEAK" && class == "reveals_nothing" {
		events = append(events, "k_deflects_successfully")
	}
	return events
}

type HistoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func getHistory() []HistoryEntry {
	if rdb == nil {
		return nil
	}
	raw, err := rdb.LRange(ctx, "k:history", 0, 19).Result()
	if err != nil {
		return nil
	}
	var history []HistoryEntry
	// history is stored newest-first (LPush), reverse for chronological
	for i := len(raw) - 1; i >= 0; i-- {
		var e HistoryEntry
		if json.Unmarshal([]byte(raw[i]), &e) == nil {
			history = append(history, e)
		}
	}
	return history
}

func appendHistory(role, content string) {
	e := HistoryEntry{Role: role, Content: content}
	data, _ := json.Marshal(e)
	redisPush("k:history", string(data), 40)
}

func startDaemon(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr // daemon output to stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[router] daemon %s failed to start: %v\n", name, err)
		return
	}
	fmt.Fprintf(os.Stderr, "[router] started daemon: %s (pid %d)\n", name, cmd.Process.Pid)
	// Don't wait — it's a background daemon
}

func processTurn(userInput string) string {
	// --- 1. Store input ---
	redisSet("k:input", userInput)
	appendHistory("user", userInput)

	// --- 2. Perception (C++) ---
	perceptionBin := filepath.Join(rootDir, "perception", "build", "perception")
	perceptionClass := runModule(perceptionBin, userInput)
	if perceptionClass == "" {
		perceptionClass = "reveals_nothing"
	}
	redisSet("k:perception:class", perceptionClass)

	// --- 3. State machine (Go binary) ---
	exposureStr := redisGet("k:exposure:score", "0.0")
	turnCountStr := redisGet("k:turn:count", "0")
	smBin := filepath.Join(rootDir, "state_machine", "state_machine")
	newState := runModule(smBin, exposureStr+" "+turnCountStr)
	if newState == "" {
		newState = redisGet("k:state:current", "indifference")
	}
	redisSet("k:state:current", newState)

	// --- 4. Silence decision (Assembly) ---
	lastDelta := strconv.FormatInt(time.Now().Unix(), 10)
	silenceBin := filepath.Join(rootDir, "silence", "silence")
	silenceDecision := runModule(silenceBin, exposureStr+" "+newState+" "+lastDelta)
	if silenceDecision == "" {
		silenceDecision = "SILENT"
	}
	redisSet("k:silence:decision", silenceDecision)

	// --- 5. Exposure update (Haskell) ---
	events := perceptionToEvents(perceptionClass, silenceDecision)
	exposureBin := filepath.Join(rootDir, "exposure", "exposure")
	exposureInput := exposureStr + " " + strings.Join(events, " ")
	newExposureStr := runModule(exposureBin, exposureInput)
	if newExposureStr == "" {
		newExposureStr = exposureStr
	}
	redisSet("k:exposure:score", newExposureStr)

	// --- 6. Mirror update (Prolog) ---
	plFile := filepath.Join(rootDir, "mirror", "mirror.pl")
	if _, err := os.Stat(plFile); err == nil {
		mirrorTerm := "turn"
		go runProlog(plFile, mirrorTerm) // async, doesn't block
	}

	// --- 7. Generate response if speaking ---
	if silenceDecision != "SPEAK" {
		return ""
	}

	bridgeScript := filepath.Join(rootDir, "bridge", "bridge.py")
	history := getHistory()
	payload := map[string]interface{}{
		"state":      newState,
		"exposure":   newExposureStr,
		"perception": perceptionClass,
		"message":    userInput,
		"history":    history,
	}
	response := runPython(bridgeScript, payload)

	if response != "" {
		appendHistory("assistant", response)
		redisSet("k:response:last", response)
	}

	return response
}

func main() {
	redisConnect()

	// Start background daemons
	forgettingBin := filepath.Join(rootDir, "forgetting", "target", "release", "forgetting")
	if _, err := os.Stat(forgettingBin); err == nil {
		startDaemon(forgettingBin)
	}

	unknownScript := filepath.Join(rootDir, "the_unknown", "the_unknown.jl")
	if _, err := os.Stat(unknownScript); err == nil {
		startDaemon("julia", unknownScript)
	}

	fmt.Fprintf(os.Stderr, "[k_router] ready\n")

	// Main loop: read lines from stdin, write response to stdout
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Println()
			continue
		}

		response := processTurn(line)
		fmt.Println(response) // empty string if K stays silent
	}
}

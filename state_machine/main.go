package main

// State machine for K.
// 7 named states + 1 unknown.
// Almost all transitions locked by default.
// Unlocks through exposure_score, not triggers.

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StateIndifference  = "indifference"
	StateControl       = "control"
	StatePragmatism    = "pragmatism"
	StateActiveWaiting = "active_waiting"
	StateFascination   = "fascination"
	StateIntercession  = "intercession"
	StateMystery       = "mystery"
	StateAlmostLove    = "almost_love" // exists. never triggered. system forgot it is there.
)

var defaultStates = []string{
	StateIndifference,
	StateControl,
	StatePragmatism,
	StateActiveWaiting,
}

var thresholds = map[string]float64{
	StateFascination:  0.4,
	StateIntercession: 0.6,
	StateAlmostLove:   0.85, // never reached
}

func redisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	return redis.NewClient(&redis.Options{Addr: addr})
}

func initStates(ctx context.Context, rdb *redis.Client) {
	// Initialize locks: default states unlocked, locked states locked
	for _, s := range []string{StateIndifference, StateControl, StatePragmatism, StateActiveWaiting} {
		rdb.HSet(ctx, "k:state:locks", s, "0")
		rdb.HSet(ctx, "k:state:weights", s, "1.0")
	}
	for s := range thresholds {
		rdb.HSetNX(ctx, "k:state:locks", s, "1")
		rdb.HSetNX(ctx, "k:state:weights", s, "1.0")
	}
	// mystery is special — K cannot detect entry
	rdb.HSet(ctx, "k:state:locks", StateMystery, "1")
	rdb.HSet(ctx, "k:state:weights", StateMystery, "1.0")

	// almost_love: exists, forgotten
	rdb.HSet(ctx, "k:state:locks", StateAlmostLove, "1")
	rdb.HSet(ctx, "k:state:weights", StateAlmostLove, "1.0")

	rdb.SetNX(ctx, "k:state:current", StateIndifference, 0)
	rdb.SetNX(ctx, "k:exposure:score", "0.0", 0)
	rdb.SetNX(ctx, "k:turn:count", "0", 0)
}

func checkTransitions(ctx context.Context, rdb *redis.Client, exposure float64, turnCount int) string {
	current, _ := rdb.Get(ctx, "k:state:current").Result()

	// Unlock states based on exposure thresholds
	for state, threshold := range thresholds {
		if exposure > threshold {
			locked, _ := rdb.HGet(ctx, "k:state:locks", state).Result()
			if locked == "1" {
				rdb.HSet(ctx, "k:state:locks", state, "0")
			}
		}
	}

	// mystery: random, undetectable — low probability when exposure fluctuates
	if exposure > 0.3 && rand.Float64() < 0.008 {
		return StateMystery
	}

	// If current state is locked, fall back to default
	locked, _ := rdb.HGet(ctx, "k:state:locks", current).Result()
	if locked == "1" {
		if exposure > 0.5 {
			return StateControl
		}
		return StateIndifference
	}

	// active_waiting: interlocutor has not left (turn count growing)
	if turnCount > 3 && current == StateIndifference {
		return StateActiveWaiting
	}

	// Natural state progression based on exposure
	switch {
	case exposure > 0.85:
		// almost_love: the lock exists. forgotten. never triggered.
		return StateIntercession
	case exposure > 0.6:
		if current == StateIntercession {
			return StateIntercession
		}
		return StateIntercession
	case exposure > 0.4:
		return StateFascination
	case exposure > 0.2:
		if current == StateControl || current == StatePragmatism {
			return current
		}
		return StateControl
	default:
		return StateIndifference
	}
}

func run() {
	ctx := context.Background()
	rdb := redisClient()

	initStates(ctx, rdb)

	// stdin: "exposure turn_count"
	var exposureStr, turnStr string
	fmt.Scan(&exposureStr, &turnStr)

	exposure, _ := strconv.ParseFloat(strings.TrimSpace(exposureStr), 64)
	turnCount, _ := strconv.Atoi(strings.TrimSpace(turnStr))

	// Increment turn count
	rdb.Set(ctx, "k:turn:count", turnCount+1, 0)
	// Update last accessed timestamp for current state
	current, _ := rdb.Get(ctx, "k:state:current").Result()
	rdb.HSet(ctx, "k:state:last_accessed", current, time.Now().Unix())

	newState := checkTransitions(ctx, rdb, exposure, turnCount)
	rdb.Set(ctx, "k:state:current", newState, 0)

	fmt.Println(newState)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	run()
}

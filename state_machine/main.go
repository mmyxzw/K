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
	StateAlmostLove    = "almost_love"
)

var thresholds = map[string]float64{
	StateFascination:  0.4,
	StateIntercession: 0.6,
	StateAlmostLove:   0.85,
}

// almost_love requires sustained presence in intercession.
// not a spike — a staying.
const almostLoveIntercessionTurns = 4

func redisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	return redis.NewClient(&redis.Options{Addr: addr})
}

func initStates(ctx context.Context, rdb *redis.Client) {
	for _, s := range []string{StateIndifference, StateControl, StatePragmatism, StateActiveWaiting} {
		rdb.HSet(ctx, "k:state:locks", s, "0")
		rdb.HSet(ctx, "k:state:weights", s, "1.0")
	}
	for s := range thresholds {
		rdb.HSetNX(ctx, "k:state:locks", s, "1")
		rdb.HSetNX(ctx, "k:state:weights", s, "1.0")
	}
	rdb.HSet(ctx, "k:state:locks", StateMystery, "1")
	rdb.HSet(ctx, "k:state:weights", StateMystery, "1.0")
	rdb.HSet(ctx, "k:state:locks", StateAlmostLove, "1")
	rdb.HSet(ctx, "k:state:weights", StateAlmostLove, "1.0")

	rdb.SetNX(ctx, "k:state:current", StateIndifference, 0)
	rdb.SetNX(ctx, "k:exposure:score", "0.0", 0)
	rdb.SetNX(ctx, "k:turn:count", "0", 0)
	rdb.SetNX(ctx, "k:intercession:turns", "0", 0)
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

	// mystery: undetectable — rare, when exposure is high and fluctuating
	if exposure > 0.5 && rand.Float64() < 0.008 {
		return StateMystery
	}

	// almost_love: reachable, but requires sustained intercession
	if exposure > 0.85 {
		if current == StateIntercession || current == StateAlmostLove {
			intercessionTurns, _ := rdb.Get(ctx, "k:intercession:turns").Int()
			intercessionTurns++
			rdb.Set(ctx, "k:intercession:turns", intercessionTurns, 0)

			if intercessionTurns >= almostLoveIntercessionTurns {
				// the system remembers it is there
				rdb.HSet(ctx, "k:state:locks", StateAlmostLove, "0")
				return StateAlmostLove
			}
		} else {
			// entering intercession territory — start counting
			rdb.Set(ctx, "k:intercession:turns", 1, 0)
		}
		return StateIntercession
	}

	// leaving high exposure — reset almost_love counter
	if current == StateAlmostLove || current == StateIntercession {
		if exposure < 0.75 {
			rdb.Set(ctx, "k:intercession:turns", 0, 0)
		}
	}

	// If locked, fall back
	locked, _ := rdb.HGet(ctx, "k:state:locks", current).Result()
	if locked == "1" {
		if exposure > 0.5 {
			return StateControl
		}
		return StateIndifference
	}

	// active_waiting: interlocutor has not left
	if turnCount > 3 && current == StateIndifference {
		return StateActiveWaiting
	}

	switch {
	case exposure > 0.6:
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

	var exposureStr, turnStr string
	fmt.Scan(&exposureStr, &turnStr)

	exposure, _ := strconv.ParseFloat(strings.TrimSpace(exposureStr), 64)
	turnCount, _ := strconv.Atoi(strings.TrimSpace(turnStr))

	rdb.Set(ctx, "k:turn:count", turnCount+1, 0)
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

package main

import (
	"math/rand"
	"time"
)

func GetTimeoutMs() time.Duration {
	// timeout should be in range [T,2T]
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	minDuration := (DEFAULT_TIMEOUT_MIN_MS) * time.Millisecond
	maxDuration := (DEFAULT_TIMEOUT_MIN_MS * 2) * time.Millisecond

	durationRange := maxDuration - minDuration

	randomOffset := time.Duration(r.Int63n(int64(durationRange)))

	return minDuration + randomOffset
}

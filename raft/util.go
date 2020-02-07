package raft

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"time"
)

// randomTimeout returns a value in range [val, 2*val)
func randomTimeout(val time.Duration) <-chan time.Time {
	delta := time.Duration(rand.Int63()) % val
	return time.After(val + delta)
}

// min returns the minimum.
func min(a, b uint64) uint64 {
	if a <= b {
		return a
	}
	return b
}

// max returns the maximum
func max(a, b uint64) uint64 {
	if a >= b {
		return a
	}
	return b
}

func generateUUID() string {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16])
}

// asyncNotify is used to do an async channel send to
// a list of channels. This will not block.
func asyncNotify(chans []chan struct{}) {
	for _, ch := range chans {
		asyncNotifyCh(ch)
	}
}

// asyncNotifyCh is used to do an async channel send
// to a single channel without blocking.
func asyncNotifyCh(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
package raft

import "time"

// Config stores the user defined configuration for a raft node
type Config struct {
	// ElectionTimeout stores the timeout duration for contesting elections
	ElectionTimeout time.Duration

	// HeartBeatTimeout stores the timeout duration for sending out the
	// heartbeats by a leader
	HeartbeatTimeout time.Duration
}

// DefaultConfig returns default configuration for a raft node
func DefaultConfig() *Config {
	return &Config{
		ElectionTimeout:  150 * time.Millisecond,
		HeartbeatTimeout: 100 * time.Millisecond,
	}
}

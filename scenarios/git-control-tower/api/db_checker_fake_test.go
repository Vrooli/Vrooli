package main

import (
	"context"
	"fmt"
)

// FakeDBChecker implements DBChecker without needing a real database.
// Use this in tests to exercise health check logic quickly and safely.
//
// The fake simulates database connectivity state:
//   - Configured: whether a database is configured at all
//   - Connected: whether the database responds to pings
//   - PingError: error to return from Ping (for testing error paths)
type FakeDBChecker struct {
	Configured bool  // Whether the database is "configured"
	Connected  bool  // Whether pings succeed
	PingError  error // Error to return from Ping (overrides Connected)

	// Call tracking for verification
	PingCalls int
}

// NewFakeDBChecker creates a FakeDBChecker with sensible defaults.
// By default, simulates a configured and connected database.
func NewFakeDBChecker() *FakeDBChecker {
	return &FakeDBChecker{
		Configured: true,
		Connected:  true,
	}
}

// Ping simulates a database ping.
func (f *FakeDBChecker) Ping(ctx context.Context) error {
	f.PingCalls++

	if f.PingError != nil {
		return f.PingError
	}

	if !f.Connected {
		return fmt.Errorf("connection refused")
	}

	return nil
}

// IsConfigured returns whether the database is configured.
func (f *FakeDBChecker) IsConfigured() bool {
	return f.Configured
}

// --- Test helpers (builder pattern) ---

// WithUnconfigured simulates a missing database configuration.
func (f *FakeDBChecker) WithUnconfigured() *FakeDBChecker {
	f.Configured = false
	return f
}

// WithDisconnected simulates a database that's unreachable.
func (f *FakeDBChecker) WithDisconnected() *FakeDBChecker {
	f.Connected = false
	return f
}

// WithPingError sets a specific error for Ping to return.
func (f *FakeDBChecker) WithPingError(err error) *FakeDBChecker {
	f.PingError = err
	return f
}

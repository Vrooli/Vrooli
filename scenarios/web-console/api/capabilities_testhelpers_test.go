package main

import (
	"context"
	"web-console/internal/capabilities"
)

// fakeChecker is the in-package-main test double for capabilities.Checker,
// shared by tests that need to drive the capabilities registry against
// scripted statuses. The real fakeChecker that powers the registry's own
// tests lives in internal/capabilities/registry_test.go alongside the
// types it exercises.
type fakeChecker struct {
	status  capabilities.Status
	message string
	calls   int
}

func (f *fakeChecker) Check(_ context.Context) (capabilities.Status, string) {
	f.calls++
	return f.status, f.message
}

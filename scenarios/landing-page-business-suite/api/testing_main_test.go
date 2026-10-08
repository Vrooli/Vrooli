package main

import (
	"os"
	"sync"
	"testing"
)

var testContainerShutdownOnce sync.Once

func TestMain(m *testing.M) {
	// Tests inject configuration-shaped credential values through the test
	// process only. Production resolution is authority-only.
	_ = os.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "1")
	code := m.Run()
	if testContainerCleanup != nil {
		testContainerShutdownOnce.Do(func() {
			testContainerCleanup()
		})
	}
	os.Exit(code)
}

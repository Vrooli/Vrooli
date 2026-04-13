package main

import (
	"os"
	"sync"
	"testing"
)

var testContainerShutdownOnce sync.Once

func TestMain(m *testing.M) {
	code := m.Run()
	if testContainerCleanup != nil {
		testContainerShutdownOnce.Do(func() {
			testContainerCleanup()
		})
	}
	os.Exit(code)
}

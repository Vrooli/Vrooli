package queue

import (
	"os"
	"testing"
)

// TestMain applies a global timing scale to keep queue tests fast and deterministic.
func TestMain(m *testing.M) {
	reset := SetTimingScaleForTests(0.01)
	code := m.Run()
	reset()
	os.Exit(code)
}

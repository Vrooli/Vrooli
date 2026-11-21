package engine

import (
	"context"
	"strings"
	"testing"
)

// [REQ:DBM-TEST-ISOLATION] A real adapter refuses before even resolving a binary.
func TestProductionEngineCannotExecuteInTests(t *testing.T) {
	_, err := (ExecRunner{Binary: "must-never-be-executed"}).Run(context.Background(), "snapshot", "restore")
	if err == nil || !strings.Contains(err.Error(), "disabled in tests") {
		t.Fatalf("isolation failed: %v", err)
	}
}

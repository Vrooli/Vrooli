package sources

import (
	"context"
	"strings"
	"testing"
)

func TestProductionSourceCannotExecuteInTests(t *testing.T) {
	_, err := (ExecRunner{}).Run(context.Background(), "must-never-be-executed")
	if err == nil || !strings.Contains(err.Error(), "disabled in tests") {
		t.Fatalf("isolation failed: %v", err)
	}
}

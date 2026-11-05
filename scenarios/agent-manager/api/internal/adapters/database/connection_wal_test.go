package database

import (
	"strings"
	"testing"
)

func TestWithWALLimitCapsTheLeftoverWAL(t *testing.T) {
	dsn := withWALLimit("file:/tmp/agent-manager.db?_pragma=journal_mode(WAL)")
	if !strings.HasSuffix(dsn, "&_pragma=journal_size_limit(67108864)") {
		t.Fatalf("dsn=%s", dsn)
	}
	if !strings.HasPrefix(dsn, "file:/tmp/agent-manager.db?_pragma=journal_mode(WAL)&") {
		t.Fatalf("existing pragmas were not preserved: %s", dsn)
	}
}

package isolation

import (
	"strings"
	"testing"
	"time"
)

func TestShouldRetainFromEnv(t *testing.T) {
	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "1")
	if !ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to honor environment override")
	}

	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "0")
	if ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to be false when env is not 1")
	}
}

func TestNewManagerAppliesDefaultTimeout(t *testing.T) {
	manager := NewManager(Config{})
	if manager.cfg.Timeout != 90*time.Second {
		t.Fatalf("expected default timeout of 90s, got %s", manager.cfg.Timeout)
	}
}

func TestBuildDBNameSanitizesAndTruncates(t *testing.T) {
	name := buildDBName("My Scenario!", strings.Repeat("run-id-", 20))
	if !strings.HasPrefix(name, "tg_pb_") {
		t.Fatalf("expected database name prefix, got %q", name)
	}
	if len(name) > 60 {
		t.Fatalf("expected database name to be truncated to 60 chars, got %d", len(name))
	}
	if strings.ContainsAny(name, " !") {
		t.Fatalf("expected database name to be sanitized, got %q", name)
	}
}

func TestMergeAndSanitize(t *testing.T) {
	merged := merge(map[string]string{"A": "1", "B": "2"}, map[string]string{"B": "override", "C": "3"})
	if merged["A"] != "1" || merged["B"] != "override" || merged["C"] != "3" {
		t.Fatalf("unexpected merged env map: %#v", merged)
	}

	if got := sanitize(" Test Genie/Playbooks "); got != "test_genie_playbooks" {
		t.Fatalf("expected sanitize to normalize punctuation and whitespace, got %q", got)
	}
}

package collectors

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProcessCollector_NoDoubleFork asserts that one Collect cycle forks the
// zombie and high-thread queries at most once each (previously getProcessHealth
// re-ran both, doubling the forks), and that critical-process presence uses a
// single process-table scan rather than one pgrep per name.
func TestProcessCollector_NoDoubleFork(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process collector shells out only on linux")
	}

	var mu sync.Mutex
	counts := map[string]int{}
	pgrepCount := 0

	orig := commandOutput
	defer func() { commandOutput = orig }()
	commandOutput = func(_ context.Context, _ time.Duration, name string, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		full := name + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(full, "grep ' Z'"):
			counts["zombie"]++
		case strings.Contains(full, "--sort=-nlwp"):
			counts["highthread"]++
		case strings.Contains(full, "ps -e --no-headers"):
			counts["total"]++
		case strings.Contains(full, "ps -eo comm"):
			counts["commscan"]++
		case strings.Contains(full, "pgrep"):
			pgrepCount++
		}
		return []byte(""), nil
	}

	c := NewProcessCollector()
	if _, err := c.Collect(context.Background()); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if counts["zombie"] != 1 {
		t.Errorf("zombie query forked %d times, want exactly 1 (no double fork)", counts["zombie"])
	}
	if counts["highthread"] != 1 {
		t.Errorf("high-thread query forked %d times, want exactly 1 (no double fork)", counts["highthread"])
	}
	if pgrepCount != 0 {
		t.Errorf("pgrep forked %d times, want 0 (replaced by single ps scan)", pgrepCount)
	}
	if counts["commscan"] != 1 {
		t.Errorf("critical-process comm scan ran %d times, want exactly 1", counts["commscan"])
	}
}

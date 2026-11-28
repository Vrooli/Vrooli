package strategy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestBenchmarkManifestDoesNotPinActiveStrategy(t *testing.T) {
	flag := cliManifestFlag(t, "strategy", "benchmark", "strategy")
	if flag.Default != "" {
		t.Fatalf("benchmark --strategy manifest default %q bypasses the reported serving strategy", flag.Default)
	}
}

func TestStrategyCompareManifestMatchesAPIConstraints(t *testing.T) {
	for _, command := range []struct{ group, name string }{
		{group: "strategy", name: "compare"},
		{group: "evals", name: "compare-strategies"},
	} {
		apply := cliManifestFlag(t, command.group, command.name, "apply")
		if !strings.Contains(apply.Description, "active strategy is never changed") {
			t.Errorf("%s %s --apply description hides that it does not mutate serving strategy: %q", command.group, command.name, apply.Description)
		}
		strategies := cliManifestFlag(t, command.group, command.name, "strategies")
		if command.name == "compare-strategies" {
			strategies = cliManifestFlag(t, command.group, command.name, "strategy_names")
		}
		if !strings.Contains(strategies.Description, "at least three including the active strategy") {
			t.Errorf("%s %s does not state the API's active-arm/three-arm constraint: %q", command.group, command.name, strategies.Description)
		}
	}
}

type manifestFlag struct {
	Name        string `json:"name"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

type manifestCommand struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Flags       []manifestFlag `json:"flags"`
}

type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
}

type testCLIManifest struct {
	Groups []manifestGroup `json:"groups"`
}

func cliManifestFlag(t *testing.T, groupName, commandName, flagName string) manifestFlag {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	manifestPath := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "manifest.json"))
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read CLI manifest: %v", err)
	}
	var manifest testCLIManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode CLI manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		if group.Name != groupName {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != commandName {
				continue
			}
			for _, flag := range command.Flags {
				if flag.Name == flagName {
					return flag
				}
			}
		}
	}
	t.Fatalf("CLI manifest flag not found: %s %s --%s", groupName, commandName, flagName)
	return manifestFlag{}
}

func TestBenchmarkStrategyNameUsesReportedActiveStrategy(t *testing.T) {
	got, err := benchmarkStrategyName("", "lexical-fallback")
	if err != nil || got != "lexical-fallback" {
		t.Fatalf("benchmarkStrategyName() = %q, %v; want current active strategy lexical-fallback", got, err)
	}
	got, err = benchmarkStrategyName("lexical-fallback", "lexical-fallback")
	if err != nil || got != "lexical-fallback" {
		t.Fatalf("explicit active strategy = %q, %v", got, err)
	}
	if _, err := benchmarkStrategyName("semantic-cross-encoder", "lexical-fallback"); err == nil {
		t.Fatal("candidate strategy must use guarded compare, not the active-only benchmark")
	}
	if _, err := benchmarkStrategyName("", ""); err == nil {
		t.Fatal("empty API active strategy must fail closed")
	}
}

func TestExpectedRankUsesFirstExpectedID(t *testing.T) {
	hits := []*routingv1.SearchHit{{Id: "other"}, {Id: "wanted"}, {Id: "later"}}
	if got := expectedRank([]string{"wanted", "later"}, hits); got != 2 {
		t.Fatalf("expectedRank = %d, want 2", got)
	}
	if got := expectedRank([]string{"missing"}, hits); got != 0 {
		t.Fatalf("missing expectedRank = %d, want 0", got)
	}
}

func TestPercentileUsesNearestRank(t *testing.T) {
	values := []int64{40, 10, 30, 20}
	if got := percentile(values, 50); got != 20 {
		t.Fatalf("p50 = %d, want 20", got)
	}
	if got := percentile(values, 95); got != 40 {
		t.Fatalf("p95 = %d, want 40", got)
	}
	if got := percentile(nil, 95); got != 0 {
		t.Fatalf("empty percentile = %d, want 0", got)
	}
}

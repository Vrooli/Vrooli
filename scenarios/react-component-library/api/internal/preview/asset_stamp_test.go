package preview

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedStampContract is the same file the Vite plugin's test suite reads.
// Two implementations of the stamp transformer exist for good reasons — the
// production UI build runs inside the Vite graph, preview bundles do not — but
// they must agree on which node carries the marker. A case that only one side
// satisfies is a case where the preview harness and the production oracle
// measured different things while both reporting success.
type sharedStampContract struct {
	Cases []struct {
		Name             string   `json:"name"`
		Source           string   `json:"source"`
		SourcePath       string   `json:"sourcePath"`
		Asset            string   `json:"asset"`
		Version          string   `json:"version"`
		ComponentName    string   `json:"componentName"`
		MustContain      []string `json:"mustContain"`
		MustNotContain   []string `json:"mustNotContain"`
		AssetMarkerCount *int     `json:"assetMarkerCount"`
	} `json:"cases"`
}

func loadSharedStampContract(t *testing.T) sharedStampContract {
	t.Helper()
	path := filepath.Join("..", "..", "..", "contracts", "asset-stamp.fixtures.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read shared stamp contract at %s: %v", path, err)
	}
	var contract sharedStampContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode shared stamp contract: %v", err)
	}
	if len(contract.Cases) == 0 {
		t.Fatal("shared stamp contract has no cases; the anti-divergence guard is vacuous")
	}
	return contract
}

func TestStampPreviewSourceSatisfiesSharedContract(t *testing.T) {
	contract := loadSharedStampContract(t)
	for _, testCase := range contract.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			got := stampPreviewSource(testCase.Source, testCase.SourcePath, testCase.Asset, testCase.Version)
			for _, fragment := range testCase.MustContain {
				if !strings.Contains(got, fragment) {
					t.Errorf("output missing %q\ngot: %s", fragment, got)
				}
			}
			for _, fragment := range testCase.MustNotContain {
				if strings.Contains(got, fragment) {
					t.Errorf("output retained %q\ngot: %s", fragment, got)
				}
			}
			if testCase.AssetMarkerCount != nil {
				if count := strings.Count(got, "data-rcl-asset"); count != *testCase.AssetMarkerCount {
					t.Errorf("marker count = %d, want %d\ngot: %s", count, *testCase.AssetMarkerCount, got)
				}
			}
		})
	}
}

// A dynamic root is emitted as object-literal properties. The marker names
// contain hyphens, so unquoted keys are a syntax error rather than a style
// choice; this pins the quoted form on the Go side too.
func TestStampPreviewSourceQuotesDynamicRootKeys(t *testing.T) {
	source := `import { createElement } from "react"; export function Presence({ Component }) { return createElement(Component, { className: "x" }); }`
	got := stampPreviewSource(source, "library/primitives/Presence/versions/1.0.0/Presence.tsx", "motion.presence", "1.0.0")
	if strings.Contains(got, "data-rcl-asset:") {
		t.Fatalf("dynamic root used an unquoted hyphenated key, which cannot parse: %s", got)
	}
	if !strings.Contains(got, `"data-rcl-asset": "motion.presence"`) {
		t.Fatalf("dynamic root was not stamped with a quoted key: %s", got)
	}
}

func TestStampPreviewSourceIgnoresNonTSXPaths(t *testing.T) {
	source := `export const tokens = { space: 1 };`
	got := stampPreviewSource(source, "library/foundations/Tokens/versions/1.0.0/Tokens.ts", "foundations.tokens", "1.0.0")
	if got != source {
		t.Fatalf("non-tsx source must pass through unchanged, got: %s", got)
	}
}

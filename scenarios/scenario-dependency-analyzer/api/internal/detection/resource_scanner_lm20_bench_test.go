package detection

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// BenchmarkResourceScanBrowserAutomationStudio exercises the real scanner on
// a large maintained scenario tree. It intentionally uses the repository's
// current resource catalog, so the benchmark covers both directory traversal
// and per-file heuristic work.
func BenchmarkResourceScanBrowserAutomationStudio(b *testing.B) {
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		b.Fatal(err)
	}
	resources := discoverAvailableResources(filepath.Join(repoRoot, "resources"))
	scanner := newResourceScanner(&catalogManager{loaded: true, knownResources: resources})
	scenarioPath := os.Getenv("LM20_SCAN_SCENARIO_ROOT")
	if scenarioPath == "" {
		scenarioPath = filepath.Join(repoRoot, "scenarios", "browser-automation-studio")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deps, err := scanner.scanContext(context.Background(), scenarioPath, "browser-automation-studio", nil)
		if err != nil {
			b.Fatal(err)
		}
		if len(deps) == 0 {
			b.Fatal("real scenario scan returned no dependencies")
		}
	}
}

// TestResourceScanMatchesRegexReferenceForPinnedScenario is an opt-in semantic
// oracle for the optimized matcher. It compares the real scanner against the
// prior per-resource regex behavior on the same pinned scenario input.
func TestResourceScanMatchesRegexReferenceForPinnedScenario(t *testing.T) {
	if os.Getenv("LM20_VERIFY_REAL_SCAN") != "1" {
		t.Skip("set LM20_VERIFY_REAL_SCAN=1 to run the real-tree regex comparison")
	}
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	scenarioPath := os.Getenv("LM20_SCAN_SCENARIO_ROOT")
	if scenarioPath == "" {
		scenarioPath = filepath.Join(repoRoot, "scenarios", "browser-automation-studio")
	}
	resources := discoverAvailableResources(filepath.Join(repoRoot, "resources"))
	scanner := newResourceScanner(&catalogManager{loaded: true, knownResources: resources})

	got, err := scanner.scanContext(context.Background(), scenarioPath, "browser-automation-studio", nil)
	if err != nil {
		t.Fatalf("optimized scan: %v", err)
	}
	want, err := regexReferenceScan(context.Background(), scanner, scenarioPath, "browser-automation-studio", resources)
	if err != nil {
		t.Fatalf("regex reference scan: %v", err)
	}
	if gotStable, wantStable := stableDependencies(got), stableDependencies(want); !reflect.DeepEqual(gotStable, wantStable) {
		t.Fatalf("optimized scan changed detected dependencies\noptimized: %#v\nregex reference: %#v", gotStable, wantStable)
	}
}

func regexReferenceScan(ctx context.Context, scanner *resourceScanner, scenarioPath, scenarioName string, resources map[string]struct{}) ([]types.ScenarioDependency, error) {
	results := map[string]types.ScenarioDependency{}
	err := filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(scenarioPath, path)
		if err != nil {
			return err
		}
		if shouldIgnoreDetectionFile(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(entry) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !contains(resourceDetectionExtensions, strings.ToLower(filepath.Ext(path))) {
			return nil
		}
		content, err := readDetectionSource(ctx, path, entry)
		if err != nil {
			return err
		}
		source := string(content)
		scanner.detectResourceCLICommands(source, rel, scenarioName, results, resources)
		asciiSource := isASCII(source)
		lowerSource := ""
		if asciiSource {
			lowerSource = strings.ToLower(source)
		}
		for resourceName := range resources {
			if asciiSource && isASCII(resourceName) && !strings.Contains(lowerSource, normalizeName(resourceName)) {
				continue
			}
			for _, pattern := range resourceHeuristicPatterns(resourceName) {
				if pattern.MatchString(source) {
					scanner.recordDetection(results, resources, scenarioName, resourceName, "heuristic", pattern.String(), rel, "declared-resource")
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	deps := make([]types.ScenarioDependency, 0, len(results))
	for _, dep := range results {
		deps = append(deps, dep)
	}
	return deps, nil
}

func stableDependencies(deps []types.ScenarioDependency) []types.ScenarioDependency {
	for i := range deps {
		deps[i].ID = ""
		deps[i].DiscoveredAt = time.Time{}
		deps[i].LastVerified = time.Time{}
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].DependencyName < deps[j].DependencyName })
	return deps
}

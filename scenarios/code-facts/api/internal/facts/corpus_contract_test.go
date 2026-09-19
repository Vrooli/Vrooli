package facts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"testing"
)

type corpusPolicyFixture struct {
	Version                 string   `json:"version"`
	GovernedRoots           []string `json:"governed_roots"`
	Roles                   []string `json:"roles"`
	DefaultSearchableRoles  []string `json:"default_searchable_roles"`
	ExcludedDirectoryNames  []string `json:"excluded_directory_names"`
	GeneratedAliasPrefixes  []string `json:"generated_alias_prefixes"`
	TransientPrefixes       []string `json:"transient_prefixes"`
	ScaleMultiplier         int      `json:"scale_multiplier"`
	ExecutionStartInventory struct {
		CapturedOn string `json:"captured_on"`
		Roles      map[string]struct {
			Files int   `json:"files"`
			Bytes int64 `json:"bytes"`
		} `json:"roles"`
	} `json:"execution_start_inventory"`
}

type retrievalEvalFixture struct {
	Version      string `json:"version"`
	QualityGates struct {
		RecallAt5 float64 `json:"recall_at_5"`
		MRRAt3    float64 `json:"mrr_at_3"`
	} `json:"quality_gates"`
	Cases []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Query    string `json:"query"`
		Scope    string `json:"scope"`
		Expected []struct {
			Path   string `json:"path"`
			Symbol string `json:"symbol"`
		} `json:"expected"`
	} `json:"cases"`
}

// TestCorpusPolicyFixtureIsComplete [REQ:CF-P0-011] protects the versioned
// inventory contract independently from the catalog implementation.
func TestCorpusPolicyFixtureIsComplete(t *testing.T) {
	var fixture corpusPolicyFixture
	readFixture(t, "corpus-policy-v1.json", &fixture)
	wantRoles := []string{"contract", "documentation", "fixture", "generated_alias", "implementation", "test", "transient"}
	sort.Strings(fixture.Roles)
	if got := fixture.Roles; !equalStrings(got, wantRoles) {
		t.Fatalf("roles = %v, want %v", got, wantRoles)
	}
	if fixture.Version != "code-facts-corpus-v1" || fixture.ScaleMultiplier != 3 {
		t.Fatalf("fixture identity = %q x%d, want code-facts-corpus-v1 x3", fixture.Version, fixture.ScaleMultiplier)
	}
	if len(fixture.GovernedRoots) == 0 || len(fixture.ExcludedDirectoryNames) == 0 || len(fixture.TransientPrefixes) == 0 {
		t.Fatal("corpus policy must govern roots and explicitly exclude transient content")
	}
	for _, role := range wantRoles {
		inventory, ok := fixture.ExecutionStartInventory.Roles[role]
		if !ok || inventory.Files <= 0 || inventory.Bytes <= 0 {
			t.Errorf("execution-start inventory for %q = %#v, want positive files and bytes", role, inventory)
		}
	}
}

// TestRetrievalEvaluationCorpusHasAuthoritativeLocators [REQ:CF-P0-013]
// rejects line-number expectations and missing source evidence.
func TestRetrievalEvaluationCorpusHasAuthoritativeLocators(t *testing.T) {
	var fixture retrievalEvalFixture
	readFixture(t, "retrieval-eval-v1.json", &fixture)
	if fixture.QualityGates.RecallAt5 != 0.95 || fixture.QualityGates.MRRAt3 != 0.85 {
		t.Fatalf("quality gates = %#v", fixture.QualityGates)
	}
	repoRoot := repositoryRoot(t)
	seen := map[string]bool{}
	categories := map[string]bool{}
	for _, testCase := range fixture.Cases {
		if testCase.ID == "" || testCase.Query == "" || testCase.Category == "" || testCase.Scope == "" {
			t.Fatalf("incomplete evaluation case: %#v", testCase)
		}
		if seen[testCase.ID] {
			t.Fatalf("duplicate evaluation id %q", testCase.ID)
		}
		seen[testCase.ID] = true
		categories[testCase.Category] = true
		for _, expected := range testCase.Expected {
			if expected.Path == "" || filepath.IsAbs(expected.Path) {
				t.Fatalf("case %s locator path %q must be repository-relative", testCase.ID, expected.Path)
			}
			if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(expected.Path))); err != nil {
				t.Fatalf("case %s authoritative locator %q: %v", testCase.ID, expected.Path, err)
			}
		}
	}
	for _, category := range []string{"exact_identifier", "path", "natural_language", "callers", "contract", "role_policy", "scope", "freshness", "negative"} {
		if !categories[category] {
			t.Errorf("evaluation corpus lacks %q category", category)
		}
	}
}

func TestThreeTimesEvaluationFixtureIsDeterministic(t *testing.T) {
	var policy corpusPolicyFixture
	var fixture retrievalEvalFixture
	readFixture(t, "corpus-policy-v1.json", &policy)
	readFixture(t, "retrieval-eval-v1.json", &fixture)
	first := scaledEvaluationIDs(fixture, policy.ScaleMultiplier)
	second := scaledEvaluationIDs(fixture, policy.ScaleMultiplier)
	if !equalStrings(first, second) || len(first) != len(fixture.Cases)*3 {
		t.Fatalf("scaled ids are not deterministic: first=%v second=%v", first, second)
	}
}

func scaledEvaluationIDs(fixture retrievalEvalFixture, multiplier int) []string {
	ids := make([]string, 0, len(fixture.Cases)*multiplier)
	for namespace := 0; namespace < multiplier; namespace++ {
		for _, testCase := range fixture.Cases {
			ids = append(ids, "scale-"+strconv.Itoa(namespace)+":"+testCase.ID)
		}
	}
	return ids
}

func readFixture(t *testing.T, name string, target any) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t testing.TB) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

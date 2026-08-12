package routing_test

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"search-hub/internal/routing"
)

// TestSearchHubPolicyIsAgnostic scans the non-test policy source that chooses,
// probes, schedules, validates, and demotes providers. The forbidden set is
// derived from the current fleet and provider descriptors, so adding a
// registrant cannot silently create a new hard-coded exception.
func TestSearchHubPolicyIsAgnostic(t *testing.T) {
	repoRoot := repositoryRoot(t)
	forbidden := fleetIdentifiers(t, repoRoot)
	violations := scanPolicySources(t, filepath.Join(repoRoot, "scenarios", "search-hub", "api", "internal"), forbidden)
	if len(violations) > 0 {
		t.Fatalf("Search Hub policy contains fleet-specific identifiers:\n%s", strings.Join(violations, "\n"))
	}
}

// TestSearchHubPolicyGuardRejectsInjectedIdentifier proves the negative
// direction without modifying production files: an injected scenario token in
// a policy source is rejected by the same scanner used by the clean-tree test.
func TestSearchHubPolicyGuardRejectsInjectedIdentifier(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "policy.go"), []byte("package policy\nconst owner = \"agent-manager\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	violations := scanPolicySources(t, root, map[string]struct{}{"agent-manager": {}})
	if len(violations) != 1 || !strings.Contains(violations[0], "agent-manager") {
		t.Fatalf("injected identifier was not rejected: %v", violations)
	}
}

func TestSyntheticRegistrantUsesGenericRoutingPolicy(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId:    "synthetic-registrant.records",
		ProviderGroup: "synthetic-registrant",
		Type:          "synthetic",
		Description:   "Synthetic records used to prove descriptor-driven routing.",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("synthetic-registrant", "/search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	classifier := &fakeClassifier{result: routing.ClassifyResult{ProviderIDs: []string{provider.GetProviderId()}, Confidence: 0.99}}
	router := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: staticResolver{urls: map[string]string{"synthetic-registrant": "http://synthetic.test"}},
		Doer: routeDoer{byURL: map[string]cannedResponse{
			"http://synthetic.test/search": {status: 200, body: `{"results":[{"id":"synthetic-1","title":"synthetic record","score":0.9}]}`},
		}},
		Classifier:    classifier,
		QueryTimeout:  time.Second,
		RerankTimeout: time.Second,
	})
	resp, err := router.Query(context.Background(), &routingv1.QueryRequest{Query: "find synthetic records"})
	if err != nil {
		t.Fatal(err)
	}
	if !classifier.called {
		t.Fatal("synthetic registrant was not classified")
	}
	if len(resp.GetCorporaSearched()) != 1 || resp.GetCorporaSearched()[0] != provider.GetProviderId() {
		t.Fatalf("synthetic registrant was not routed generically: %v", resp.GetCorporaSearched())
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../scenarios/search-hub/api/internal/routing/agnosticism_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func fleetIdentifiers(t *testing.T, repoRoot string) map[string]struct{} {
	t.Helper()
	ids := map[string]struct{}{}
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "search-hub" {
			continue
		}
		ids[entry.Name()] = struct{}{}
		searchFile := filepath.Join(scenariosRoot, entry.Name(), ".vrooli", "search.json")
		var descriptor struct {
			Providers []struct {
				ProviderID    string `json:"provider_id"`
				ProviderGroup string `json:"provider_group"`
			} `json:"providers"`
		}
		raw, readErr := os.ReadFile(searchFile)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			t.Fatalf("read %s: %v", searchFile, readErr)
		}
		if err := json.Unmarshal(raw, &descriptor); err != nil {
			t.Fatalf("decode %s: %v", searchFile, err)
		}
		for _, provider := range descriptor.Providers {
			if provider.ProviderID != "" {
				ids[provider.ProviderID] = struct{}{}
			}
			if provider.ProviderGroup != "" {
				ids[provider.ProviderGroup] = struct{}{}
			}
		}
	}
	return ids
}

func scanPolicySources(t *testing.T, root string, forbidden map[string]struct{}) []string {
	t.Helper()
	// This is intentionally one explicit fixture exclusion, not a broad test
	// or data glob. The fixture corpus is synthetic by declaration and is the
	// only place where real-looking provider names are expected in policy data.
	excludedFixtureRoots := []string{"routing/testdata/provider_corpus"}
	if len(excludedFixtureRoots) != 1 {
		t.Fatalf("fixture exclusion list changed unexpectedly: %v", excludedFixtureRoots)
	}
	violations := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Base(root) == "internal" {
			first := strings.Split(rel, string(filepath.Separator))[0]
			if first != "." && first != "routing" && first != "evalsched" && first != "validation" {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		for _, excluded := range excludedFixtureRoots {
			if rel == excluded || strings.HasPrefix(rel, excluded+string(filepath.Separator)) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, raw, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			for id := range forbidden {
				if value == id || strings.Contains(value, id) {
					violations = append(violations, fmt.Sprintf("%s contains %q", rel, id))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(violations)
	return violations
}

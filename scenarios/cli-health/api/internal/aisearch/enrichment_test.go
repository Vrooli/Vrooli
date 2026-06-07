package aisearch

import (
	"strings"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

func TestHumanizePath(t *testing.T) {
	cases := map[string]string{
		"vrooli scenario list":                       "vrooli scenario list",
		"maintenance-orchestrator scenario list-all": "maintenance orchestrator scenario list all",
		"go-code-graph rewrite RewriteApply":         "go code graph rewrite rewrite apply",
		"prompt-manager action create":               "prompt manager action create",
	}
	for in, want := range cases {
		if got := humanizePath(in); got != want {
			t.Errorf("humanizePath(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestComposeEnriched_HumanizesAndCleans(t *testing.T) {
	// A help-source record whose description is the verbose --help dump.
	r := CommandRecord{
		Origin:   "maintenance-orchestrator",
		Group:    "scenario",
		Name:     "list-all",
		FullPath: "maintenance-orchestrator scenario list-all",
		Description: "maintenance-orchestrator scenario list-all - List every scenario\n\n" +
			"Usage:\n  maintenance-orchestrator scenario list-all [options]\n\nOptions:\n  --json  emit json",
		Source: "help",
	}
	got := composeCommandEmbeddingTextEnriched(r)

	if !strings.Contains(got, "scenario list all") {
		t.Errorf("enriched text must contain humanized identity 'scenario list all':\n%s", got)
	}
	if !strings.Contains(got, "List every scenario") {
		t.Errorf("enriched text must keep the lead gloss:\n%s", got)
	}
	if strings.Contains(got, "Usage:") || strings.Contains(got, "--json") {
		t.Errorf("enriched text must strip the verbose help dump:\n%s", got)
	}
	if strings.Contains(got, "Source:") || strings.Contains(got, "Binding:") {
		t.Errorf("enriched text must drop machine-only Source/Binding lines:\n%s", got)
	}
}

func TestCleanDescription_ManifestKept(t *testing.T) {
	r := CommandRecord{
		FullPath:    "demo x y",
		Description: "List things quickly",
		Source:      "manifest",
	}
	if got := cleanDescription(r); got != "List things quickly" {
		t.Errorf("manifest description = %q; want unchanged", got)
	}
}

func TestAuthorityDecorator_BoostsCanonicalOrigin(t *testing.T) {
	dec := newAuthorityDecorator()
	hits := []pkg.SearchResult{
		{ID: "1", Score: 0.50, Payload: map[string]any{"origin": "vrooli", "full_path": "vrooli scenario list"}},
		{ID: "2", Score: 0.60, Payload: map[string]any{"origin": "maintenance-orchestrator", "full_path": "maintenance-orchestrator scenario list-all"}},
	}
	dec(hits, pkg.SearchQuery{Query: "list all scenarios"})

	if hits[0].Score <= 0.60 {
		t.Errorf("canonical vrooli hit should be boosted above the sibling: got %.3f", hits[0].Score)
	}
	if hits[1].Score != 0.60 {
		t.Errorf("non-canonical hit must be untouched: got %.3f", hits[1].Score)
	}
}

func TestAuthorityDecorator_SkipsWhenQueryNamesOtherOrigin(t *testing.T) {
	dec := newAuthorityDecorator()
	hits := []pkg.SearchResult{
		{ID: "1", Score: 0.50, Payload: map[string]any{"origin": "vrooli", "full_path": "vrooli scenario list"}},
		{ID: "2", Score: 0.40, Payload: map[string]any{"origin": "swarm-manager", "full_path": "swarm-manager records create"}},
	}
	dec(hits, pkg.SearchQuery{Query: "swarm manager records create"})

	if hits[0].Score != 0.50 {
		t.Errorf("query names another origin (swarm) — vrooli must NOT be boosted: got %.3f", hits[0].Score)
	}
}

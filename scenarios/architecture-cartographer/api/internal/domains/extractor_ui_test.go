package domains

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestUIFeatureExtractor(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir,
		"ui/src/features/graph",
		"ui/src/features/conflicts",
		"ui/src/features/health", // non-feature, filtered
	)
	ext, err := NewUIFeatureExtractor().Extract(context.Background(), dir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if ext.Source != SourceUIFeatures {
		t.Fatalf("source = %q", ext.Source)
	}
	names := make([]string, 0, len(ext.Domains))
	for _, d := range ext.Domains {
		names = append(names, d.Name)
	}
	if !reflect.DeepEqual(names, []string{"conflicts", "graph"}) {
		t.Fatalf("ui features = %v, want [conflicts graph]", names)
	}
}

func TestUIFeatureExtractor_MissingDir(t *testing.T) {
	ext, err := NewUIFeatureExtractor().Extract(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("missing ui dir should not error: %v", err)
	}
	if len(ext.Domains) != 0 {
		t.Fatalf("expected no features, got %d", len(ext.Domains))
	}
}

func TestSourceAdvisory(t *testing.T) {
	if !SourceUIFeatures.Advisory() {
		t.Fatal("UI features must be advisory")
	}
	for _, s := range []Source{SourceDomainsDoc, SourceAPIFolders, SourceCLIGroups, SourceAPIManifest} {
		if s.Advisory() {
			t.Fatalf("%q must not be advisory", s)
		}
	}
}

// TestResolve_AdvisoryNeverAuthority ensures a UI-only declaration set does
// not win authority (the ladder must fall through to ErrNoAuthority).
func TestResolve_AdvisoryNeverAuthority(t *testing.T) {
	exts := []Extraction{
		{Source: SourceDomainsDoc},
		{Source: SourceUIFeatures, Domains: []ExtractedDomain{{Name: "widgets", Paths: []string{"ui/"}}}},
	}
	_, err := Resolve("x", exts, time.Time{})
	if _, ok := err.(ErrNoAuthority); !ok {
		t.Fatalf("UI-only declarations must not win authority; want ErrNoAuthority, got %v", err)
	}
}

func TestExtractorsFor_OrderAndUIAlwaysLast(t *testing.T) {
	exs := ExtractorsFor([]string{"cli_groups", "domains_doc"}, nil)
	got := make([]Source, 0, len(exs))
	for _, e := range exs {
		got = append(got, e.Source())
	}
	// Requested order honored; api_folders dropped (not requested); UI appended last.
	want := []Source{SourceCLIGroups, SourceDomainsDoc, SourceUIFeatures}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestExtractorsFor_EmptyFallsBackToDefault(t *testing.T) {
	exs := ExtractorsFor(nil, nil)
	if len(exs) != 4 {
		t.Fatalf("default ladder should have 4 rungs (doc, folders, cli, ui), got %d", len(exs))
	}
	if exs[len(exs)-1].Source() != SourceUIFeatures {
		t.Fatal("UI rung must be last")
	}
}

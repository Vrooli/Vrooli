package repocontract

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestEnumerateCanonicalTargets(t *testing.T) {
	contract := mustLoadDefault(t, "/home/matthalloran8/Vrooli")
	targets, err := contract.EnumerateTargets("/home/matthalloran8/Vrooli")
	if err != nil {
		t.Fatalf("EnumerateTargets: %v", err)
	}
	counts := map[TargetKind]int{}
	for _, target := range targets {
		counts[target.Kind]++
	}
	t.Logf("target counts: %#v total=%d", counts, len(targets))
	if len(targets) == 0 {
		t.Fatal("expected canonical repository targets")
	}
	for _, kind := range []TargetKind{
		TargetKindScenario, TargetKindTool, TargetKindResource,
		TargetKindPackage, TargetKindSafeguard, TargetKindTeam,
		TargetKindControlPlane, TargetKindDocs, TargetKindProject,
	} {
		if counts[kind] == 0 {
			t.Fatalf("canonical target kind %s has no enumerated targets", kind)
		}
	}
	project, err := contract.Target("/home/matthalloran8/Vrooli", TargetKindProject, "repo")
	if err != nil {
		t.Fatalf("project target: %v", err)
	}
	if project.ID != "repo" || project.Root != "." {
		t.Fatalf("project target = %#v, want id repo/root .", project)
	}
}

func TestTargetIndexLookupUsesLongestRootAndMissesUnknownRoots(t *testing.T) {
	index := NewTargetIndex([]Target{
		{Kind: TargetKindControlPlane, ID: "internal", Root: "internal"},
		{Kind: TargetKindTool, ID: "compiler", Root: "internal/tools/compiler"},
	})
	tests := []struct {
		name string
		path string
		kind TargetKind
		id   string
		ok   bool
	}{
		{name: "longest root", path: "internal/tools/compiler/main.go", kind: TargetKindTool, id: "compiler", ok: true},
		{name: "parent root", path: "internal/other/main.go", kind: TargetKindControlPlane, id: "internal", ok: true},
		{name: "unknown root", path: "resources/postgres/data.db", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := index.Lookup(tt.path)
			if ok != tt.ok {
				t.Fatalf("Lookup(%q) ok = %v, want %v", tt.path, ok, tt.ok)
			}
			if ok && (got.Kind != tt.kind || got.ID != tt.id) {
				t.Fatalf("Lookup(%q) = %#v, want %s:%s", tt.path, got, tt.kind, tt.id)
			}
		})
	}
}

func TestEnumerateTargetsSkipsMissingMarkersAndExcludedRoots(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	doc := validContractDoc(t)
	doc.Targets.Kinds = map[string]TargetSpec{
		"scenario": {Roots: []string{"scenarios/*"}, Marker: ".vrooli/service.json"},
		"tool":     {Roots: []string{"internal/tools/*"}, Marker: "tool.json", Exclude: []string{"internal/tools/excluded"}},
	}
	writeContractFile(t, fixture.Root, doc)
	fixture.WriteScenarioStub(t, "present")
	if err := os.MkdirAll(filepath.Join(fixture.Root, "scenarios", "missing"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"present", "excluded"} {
		if err := os.MkdirAll(filepath.Join(fixture.Root, "internal", "tools", name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fixture.Root, "internal", "tools", name, "tool.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	contract := mustLoadDefault(t, fixture.Root)
	targets, err := contract.EnumerateTargets(fixture.Root)
	if err != nil {
		t.Fatalf("EnumerateTargets() error = %v", err)
	}
	index := NewTargetIndex(targets)
	if target, ok := index.Lookup("scenarios/missing/file.go"); ok {
		t.Fatalf("missing-marker lookup = %#v, want no target", target)
	}
	if target, ok := index.Lookup("internal/tools/excluded/main.go"); ok {
		t.Fatalf("excluded lookup = %#v, want no target", target)
	}
	if target, ok := index.Lookup("internal/tools/present/main.go"); !ok || target.Kind != TargetKindTool || target.ID != "present" {
		t.Fatalf("present lookup = %#v, %v", target, ok)
	}
}

func TestContractTargetIndexRefreshesByContractMtimeAndTTL(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	doc := validContractDoc(t)
	doc.Targets.Kinds = map[string]TargetSpec{
		"scenario": {Roots: []string{"scenarios/*"}, Marker: ".vrooli/service.json"},
	}
	contractPath := writeContractFile(t, fixture.Root, doc)
	fixture.WriteScenarioStub(t, "first")

	oldNow := targetIndexNow
	oldTTL := targetIndexTTL
	t.Cleanup(func() {
		targetIndexNow = oldNow
		targetIndexTTL = oldTTL
		clearTargetIndexCache()
	})
	clearTargetIndexCache()
	now := time.Unix(100, 0)
	targetIndexNow = func() time.Time { return now }
	contract := mustLoadDefault(t, fixture.Root)
	first, err := contract.NewTargetIndex(fixture.Root)
	if err != nil {
		t.Fatalf("first TargetIndex() error = %v", err)
	}
	fixture.WriteScenarioStub(t, "second")
	if _, ok := first.Lookup("scenarios/second/file.go"); ok {
		t.Fatal("first index unexpectedly contains a target added after enumeration")
	}
	cached, err := contract.NewTargetIndex(fixture.Root)
	if err != nil {
		t.Fatalf("cached TargetIndex() error = %v", err)
	}
	if cached != first {
		t.Fatal("TargetIndex() rebuilt before mtime or TTL invalidation")
	}

	info, err := os.Stat(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(contractPath, info.ModTime().Add(time.Second), info.ModTime().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	byMtime, err := contract.NewTargetIndex(fixture.Root)
	if err != nil {
		t.Fatalf("mtime TargetIndex() error = %v", err)
	}
	if byMtime == first {
		t.Fatal("TargetIndex() did not refresh after contract mtime changed")
	}
	if _, ok := byMtime.Lookup("scenarios/second/file.go"); !ok {
		t.Fatal("mtime-refreshed index missed the new target")
	}

	fixture.WriteScenarioStub(t, "third")
	now = now.Add(DefaultTargetIndexTTL + time.Second)
	byTTL, err := contract.NewTargetIndex(fixture.Root)
	if err != nil {
		t.Fatalf("TTL TargetIndex() error = %v", err)
	}
	if byTTL == byMtime {
		t.Fatal("TargetIndex() did not refresh after TTL expiration")
	}
	if _, ok := byTTL.Lookup("scenarios/third/file.go"); !ok {
		t.Fatal("TTL-refreshed index missed the new target")
	}
}

func clearTargetIndexCache() {
	targetIndexMu.Lock()
	defer targetIndexMu.Unlock()
	targetIndexCache = map[string]targetIndexCacheEntry{}
}

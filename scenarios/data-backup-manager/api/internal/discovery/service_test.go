package discovery_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/discovery/mocks"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"
)

func newService(d discovery.Deps) discovery.Service {
	if d.Volumes == nil {
		d.Volumes = &mocks.FakeVolumeScanner{}
	}
	if d.Sources == nil {
		d.Sources = &mocks.FakeTargetSourceScanner{}
	}
	if d.Targets == nil {
		d.Targets = &mocks.FakeTargetCatalog{}
	}
	if d.Destinations == nil {
		d.Destinations = &mocks.FakeDestinationCatalog{}
	}
	if d.Protected == nil {
		d.Protected = &mocks.FakeProtectedPaths{}
	}
	if d.Dismissals == nil {
		d.Dismissals = mocks.NewFakeDismissalStore()
	}
	return discovery.NewService(d)
}

func candidate(owner, name, locator string, kind sources.SourceKind, bytes int64) discovery.TargetCandidate {
	return discovery.TargetCandidate{Owner: owner, Name: name, SourceKind: kind, Locator: locator, ApproxBytes: bytes}
}

func TestListTargetSuggestionsExcludesRegisteredByKeyAndLocator(t *testing.T) {
	svc := newService(discovery.Deps{
		Sources: &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
			candidate("vrooli", "plans", "/home/u/.vrooli/plans", sources.KindFilesystem, 100),
			candidate("vrooli", "state", "/home/u/.vrooli/state", sources.KindFilesystem, 200),
			candidate("vrooli", "config", "/home/u/.vrooli/config", sources.KindFilesystem, 50),
		}},
		Targets: &mocks.FakeTargetCatalog{Targets: []discovery.ExistingTarget{
			// plans excluded by (owner,name); state excluded by locator match.
			{Owner: "vrooli", Name: "plans", Locator: "/somewhere/else"},
			{Owner: "other", Name: "x", Locator: "/home/u/.vrooli/state"},
		}},
	})

	got, err := svc.ListTargetSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "config" {
		t.Fatalf("expected only config suggested, got %+v", got)
	}
	if got[0].ID == "" {
		t.Fatal("expected a stable id on the suggestion")
	}
}

func TestListTargetSuggestionsExcludesDismissed(t *testing.T) {
	configID := discovery.TargetSuggestionIDForTest("/home/u/.vrooli/config")
	svc := newService(discovery.Deps{
		Sources: &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
			candidate("vrooli", "config", "/home/u/.vrooli/config", sources.KindFilesystem, 50),
			candidate("vrooli", "plans", "/home/u/.vrooli/plans", sources.KindFilesystem, 50),
		}},
		Dismissals: mocks.NewFakeDismissalStore(configID),
	})

	got, err := svc.ListTargetSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "plans" {
		t.Fatalf("expected config dismissed and only plans left, got %+v", got)
	}
}

func TestListTargetSuggestionsOrdersPlatformThenLarger(t *testing.T) {
	svc := newService(discovery.Deps{
		Sources: &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
			candidate("scenario-x", "store", "/repo/x/store", sources.KindFilesystem, 9999),
			candidate("vrooli", "small", "/home/u/.vrooli/small", sources.KindFilesystem, 10),
			candidate("vrooli", "big", "/home/u/.vrooli/big", sources.KindFilesystem, 5000),
		}},
	})

	got, err := svc.ListTargetSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Platform owner first (big before small by size), scenario owner last even
	// though it is the largest.
	want := []string{"big", "small", "store"}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d: %+v", len(got), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("order[%d] = %q, want %q (full: %+v)", i, got[i].Name, name, got)
		}
	}
}

func TestListTargetSuggestionsFlowsResourceCandidatesThroughComposite(t *testing.T) {
	// A composite of a well-known scanner (vrooli platform state) and a resource
	// scanner (coding-agent durable data) — the production shape. The service
	// must surface both, place "vrooli" first, and dedup an already-registered
	// resource target.
	wellKnown := &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
		candidate("vrooli", "plans", "/home/u/.vrooli/plans", sources.KindFilesystem, 100),
	}}
	resourceScan := &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
		{Owner: "claude-code", Name: "history", SourceKind: sources.KindFilesystem, Locator: "/home/u/.claude/history.jsonl", ApproxBytes: 9000},
		{Owner: "codex", Name: "sessions", SourceKind: sources.KindFilesystem, Locator: "/home/u/.codex/sessions", ApproxBytes: 5000},
		{Owner: "claude-code", Name: "credentials", SourceKind: sources.KindFilesystem, Locator: "/home/u/.claude/.credentials.json", ApproxBytes: 471, Sensitive: true},
	}}
	svc := newService(discovery.Deps{
		Sources: discovery.NewCompositeScanner(wellKnown, resourceScan),
		Targets: &mocks.FakeTargetCatalog{Targets: []discovery.ExistingTarget{
			// codex/sessions is already registered → must be filtered out.
			{Owner: "codex", Name: "sessions", Locator: "/home/u/.codex/sessions"},
		}},
	})

	got, err := svc.ListTargetSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// vrooli/plans first (platform owner), then resource candidates by size desc.
	want := []string{"plans", "history", "credentials"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %d: %+v", want, len(got), got)
	}
	for i, n := range want {
		if got[i].Name != n {
			t.Fatalf("order[%d] = %q, want %q (full: %+v)", i, got[i].Name, n, got)
		}
	}
	if got[0].Owner != "vrooli" {
		t.Errorf("platform owner must rank first, got %q", got[0].Owner)
	}
	// The sensitive resource candidate keeps its flag through the pipeline.
	var creds discovery.TargetSuggestion
	for _, s := range got {
		if s.Name == "credentials" {
			creds = s
		}
	}
	if !creds.Sensitive {
		t.Error("credentials suggestion must remain flagged sensitive end-to-end")
	}
}

func vol(mount, fstype string, class sysmounts.DriveClass, removable, ro bool, free, total int64) discovery.Volume {
	return discovery.Volume{
		Mountpoint: mount, Filesystem: fstype, Class: class,
		Removable: removable, ReadOnly: ro, FreeBytes: free, TotalBytes: total,
	}
}

func TestListDestinationSuggestionsFiltersAndRanks(t *testing.T) {
	svc := newService(discovery.Deps{
		Volumes: &mocks.FakeVolumeScanner{Volumes: []discovery.Volume{
			vol("/", "ext4", sysmounts.ClassFixed, false, false, 100, 500),             // contains protected → not ok
			vol("/media/u/USB", "vfat", sysmounts.ClassRemovable, true, false, 50, 64), // removable, ok
			vol("/mnt/data", "ext4", sysmounts.ClassFixed, false, false, 900, 1000),    // fixed, ok, large
			vol("/cdrom", "iso9660", sysmounts.ClassRemovable, true, true, 0, 700),     // read-only → excluded
			vol("/mnt/net", "nfs4", sysmounts.ClassNetwork, false, false, 10, 20),      // network, ok
			vol("/mnt/used", "ext4", sysmounts.ClassFixed, false, false, 10, 20),       // already a destination
		}},
		Protected:    &mocks.FakeProtectedPaths{Paths: []string{"/home/u/.vrooli"}},
		Destinations: &mocks.FakeDestinationCatalog{Destinations: []discovery.ExistingDestination{{Location: "/mnt/used"}}},
	})

	got, err := svc.ListDestinationSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// /cdrom (ro) and /mnt/used (existing) excluded → 4 remain.
	if len(got) != 4 {
		t.Fatalf("expected 4 suggestions, got %d: %+v", len(got), got)
	}
	// Ranking: separate-root-ok first; among those removable, then fixed (large),
	// then network; the in-root "/" volume (not ok) ranks last.
	wantOrder := []string{"/media/u/USB", "/mnt/data", "/mnt/net", "/"}
	for i, loc := range wantOrder {
		if got[i].Location != loc {
			t.Fatalf("order[%d] = %q, want %q (full: %+v)", i, got[i].Location, loc, got)
		}
	}
	// "/" overlaps a protected path → flagged not-ok.
	if last := got[len(got)-1]; last.Location != "/" || last.SeparateRootOK {
		t.Fatalf("expected '/' flagged separate_root_ok=false, got %+v", last)
	}
	// Removable carries the convenience flag.
	if !got[0].Removable {
		t.Fatalf("expected first (USB) to be removable, got %+v", got[0])
	}
}

func TestDismissSuggestionPersistsAndFilters(t *testing.T) {
	store := mocks.NewFakeDismissalStore()
	svc := newService(discovery.Deps{
		Sources: &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{
			candidate("vrooli", "plans", "/home/u/.vrooli/plans", sources.KindFilesystem, 10),
		}},
		Dismissals: store,
	})

	first, err := svc.ListTargetSuggestions(context.Background())
	if err != nil || len(first) != 1 {
		t.Fatalf("expected 1 suggestion before dismiss, got %d (%v)", len(first), err)
	}
	ok, err := svc.DismissSuggestion(context.Background(), first[0].ID)
	if err != nil || !ok {
		t.Fatalf("dismiss failed: ok=%v err=%v", ok, err)
	}
	after, err := svc.ListTargetSuggestions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected 0 suggestions after dismiss, got %+v", after)
	}
}

func TestDismissSuggestionRejectsEmptyID(t *testing.T) {
	svc := newService(discovery.Deps{})
	if _, err := svc.DismissSuggestion(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty id")
	}
}

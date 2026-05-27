package discovery_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/discovery/mocks"
	"data-backup-manager/internal/sources"
)

func cand(owner, name string) discovery.TargetCandidate {
	return discovery.TargetCandidate{Owner: owner, Name: name, SourceKind: sources.KindFilesystem, Locator: "/" + owner + "/" + name}
}

func TestCompositeScannerConcatenatesInOrder(t *testing.T) {
	first := &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{cand("vrooli", "plans")}}
	second := &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{cand("claude-code", "history"), cand("codex", "sessions")}}
	composite := discovery.NewCompositeScanner(first, second)

	got, err := composite.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := []string{"plans", "history", "sessions"}
	if len(got) != len(want) {
		t.Fatalf("expected %d candidates, got %d: %+v", len(want), len(got), got)
	}
	for i, n := range want {
		if got[i].Name != n {
			t.Fatalf("order[%d] = %q, want %q", i, got[i].Name, n)
		}
	}
}

func TestCompositeScannerPropagatesFirstError(t *testing.T) {
	boom := errors.New("boom")
	composite := discovery.NewCompositeScanner(
		&mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{cand("vrooli", "plans")}},
		&mocks.FakeTargetSourceScanner{Err: boom},
		&mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{cand("codex", "sessions")}},
	)
	if _, err := composite.Scan(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestCompositeScannerToleratesNilScanner(t *testing.T) {
	composite := discovery.NewCompositeScanner(nil, &mocks.FakeTargetSourceScanner{Candidates: []discovery.TargetCandidate{cand("vrooli", "plans")}})
	got, err := composite.Scan(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("expected 1 candidate and no error, got %d (%v)", len(got), err)
	}
}

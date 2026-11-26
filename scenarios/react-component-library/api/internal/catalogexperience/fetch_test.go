package catalogexperience

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"react-component-library/internal/experience"
)

type stubReader struct {
	snapshot experience.Snapshot
	err      error
	got      experience.Component
}

func TestFetcherUsesRepositoryReader(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "react-component-library"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Fetcher(root)(context.Background(), "react-component-library:Missing", "1.0.0")
	if err != nil || got != nil {
		t.Fatalf("missing exact-version contract should produce no captures, got %v, error %v", got, err)
	}
}

func (r *stubReader) Get(_ context.Context, component experience.Component) (experience.Snapshot, error) {
	r.got = component
	return r.snapshot, r.err
}

func TestFetcherMapsExactVersionEvidence(t *testing.T) {
	reader := &stubReader{snapshot: experience.Snapshot{
		EvidenceStatus: "covered",
		States:         []experience.State{{ID: "ready", ExampleName: "ReadyExample"}},
		Claims:         []experience.Claim{{ID: "a11y", Type: "accessibility"}, {ID: "floor"}},
		Evidence: []experience.Evidence{
			{ClaimID: "a11y", Verdict: "pass", StateID: "ready", ExampleName: "ReadyExample", Viewport: "desktop", CaptureRef: "capture://one", CheckedAt: "2026-01-02"},
			{ClaimID: "a11y", ClaimType: "persisted-type", Verdict: "fail", StateID: "ready", ExampleName: "ReadyExample"},
			{ClaimID: "a11y", Verdict: "fail", StateID: "ready", ExampleName: "OldExample"},
			{ClaimID: "floor", Verdict: "pass", StateID: "inherited"},
			{ClaimID: "unknown", Verdict: "pass", StateID: "unknown"},
		},
	}}
	fetch := fetcher(reader)
	got, err := fetch(context.Background(), "  react-component-library:CardShell  ", "  1.2.3  ")
	if err != nil {
		t.Fatalf("fetch returned error: %v", err)
	}
	if reader.got.LibraryID != "  react-component-library:CardShell  " || reader.got.Slug != "CardShell" || reader.got.Version != "  1.2.3  " {
		t.Fatalf("reader received component %+v", reader.got)
	}
	if len(got) != 4 {
		t.Fatalf("expected current and unknown-state captures, got %+v", got)
	}
	if got[0].ClaimType != "accessibility" || got[0].Verdict != "pass" || got[0].CaptureRef != "capture://one" || got[0].CheckedAt != "2026-01-02" {
		t.Fatalf("legacy capture was not mapped with contract claim type: %+v", got[0])
	}
	if got[1].ClaimType != "persisted-type" {
		t.Fatalf("persisted claim type should take precedence, got %+v", got[1])
	}
	if got[2].ClaimType != "" || got[3].ClaimID != "unknown" {
		t.Fatalf("unknown inherited claims should remain unmapped and unknown states accepted: %+v", got[2:])
	}
}

func TestFetcherSkipsUnavailableEvidenceAndPropagatesReaderErrors(t *testing.T) {
	for _, status := range []string{"unavailable", "not-configured"} {
		reader := &stubReader{snapshot: experience.Snapshot{EvidenceStatus: status, Evidence: []experience.Evidence{{ClaimID: "a"}}}}
		got, err := fetcher(reader)(context.Background(), "component", "1")
		if err != nil || got != nil {
			t.Fatalf("status %q returned captures %v, error %v", status, got, err)
		}
	}
	wantErr := errors.New("reader failure")
	got, err := fetcher(&stubReader{err: wantErr})(context.Background(), "component", "1")
	if !errors.Is(err, wantErr) || got != nil {
		t.Fatalf("reader error was not propagated: captures=%v err=%v", got, err)
	}
}

func TestMatchesCurrentStateRejectsRenamedSpecimens(t *testing.T) {
	snapshot := experience.Snapshot{
		States: []experience.State{{ID: "default", ExampleName: "ready"}},
	}
	if matchesCurrentState(snapshot, experience.Evidence{StateID: "default", ExampleName: "default"}) {
		t.Fatal("renamed specimen should not be accepted as current state evidence")
	}
	if !matchesCurrentState(snapshot, experience.Evidence{StateID: "default", ExampleName: "ready"}) {
		t.Fatal("current specimen should be accepted")
	}
}

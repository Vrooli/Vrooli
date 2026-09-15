package operatorstate_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	sharedstate "github.com/vrooli/vrooli/internal/operatorstate"
)

func newService(t *testing.T) *sharedstate.Service {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "..", ".."))
	return sharedstate.New(sharedstate.Config{
		RepoRoot:   t.TempDir(),
		SchemaPath: filepath.Join(repoRoot, sharedstate.SchemaPath),
		Now:        func() time.Time { return time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC) },
	})
}

// [REQ:ONB-STATE-AUTHORITY]
// [REQ:ONB-STATE-MERGE-PATCH]
func TestServiceEvidenceRoutesChangesThroughOneMergePatchAuthority(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"shared","scenarios":{"alpha":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	updated, err := service.Apply(ctx, []byte(`{"scenarios":{"beta":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if updated.TrustPosture != "shared" || updated.Scenarios["alpha"].Enabled == nil || !*updated.Scenarios["alpha"].Enabled || updated.Scenarios["beta"].Enabled == nil || !*updated.Scenarios["beta"].Enabled {
		t.Fatalf("merge patch lost prior state: %#v", updated)
	}
}

// [REQ:ONB-STATE-SCHEMA-VALIDATED]
func TestServiceEvidenceRejectsInvalidStateWithoutWriting(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"shared"}`)); err != nil {
		t.Fatal(err)
	}
	before, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"invalid"}`)); err == nil {
		t.Fatal("invalid trust posture was accepted")
	}
	after, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.TrustPosture != before.TrustPosture || sharedstate.Revision(after) != sharedstate.Revision(before) {
		t.Fatalf("invalid patch changed stored document: before=%#v after=%#v", before, after)
	}
}

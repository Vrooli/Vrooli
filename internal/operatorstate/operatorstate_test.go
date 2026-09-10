package operatorstate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func testService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	return New(Config{RepoRoot: root, SchemaPath: filepath.Join(repoRoot, SchemaPath), Now: func() time.Time {
		return time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	}}), root
}

func TestApplyPreservesFieldsThisWriterDoesNotModel(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	first, err := service.Apply(ctx, []byte(`{"trust_posture":"shared","core":{"seed":["alpha"],"trusted_base":["alpha"]},"future_permission":{"enabled":true}}`))
	if err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	if first.TrustPosture != "shared" || first.Core == nil {
		t.Fatalf("typed fields were not retained: %#v", first)
	}
	second, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if _, ok := second.RawFields["future_permission"]; !ok {
		t.Fatal("future field was discarded")
	}
	state, err := service.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(state.RawFields["future_permission"]) != `{"enabled":true}` {
		t.Fatalf("future field bytes = %s", state.RawFields["future_permission"])
	}
}

func TestApplyRejectsInvalidPatchWithoutChangingStoredDocument(t *testing.T) {
	service, root := testService(t)
	ctx := context.Background()
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"personal"}`)); err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	path := filepath.Join(root, ".vrooli", StateFile)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	if _, err := service.Apply(ctx, []byte(`{"trust_posture":"not-a-posture"}`)); err == nil || !strings.Contains(err.Error(), "/trust_posture") {
		t.Fatalf("invalid patch error = %v, want JSON path", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid patch changed stored state")
	}
}

func TestHostWorkloadPostureDefaultsAndValidates(t *testing.T) {
	service, _ := testService(t)
	doc, err := service.Load(context.Background())
	if err != nil || doc.HostWorkloadPosture != "vrooli_only" {
		t.Fatalf("default posture = %q, err=%v", doc.HostWorkloadPosture, err)
	}
	if _, err := service.Apply(context.Background(), []byte(`{"host_workload_posture":"operator_machine"}`)); err == nil || !strings.Contains(err.Error(), "/host_workload_posture") {
		t.Fatalf("invalid posture error = %v", err)
	}
}

func TestCapacityPostureDefaultsAndValidates(t *testing.T) {
	service, _ := testService(t)
	doc, err := service.Load(context.Background())
	if err != nil || doc.CapacityPosture != "balanced" {
		t.Fatalf("default capacity posture = %q, err=%v", doc.CapacityPosture, err)
	}
	if _, err := service.Apply(context.Background(), []byte(`{"capacity_posture":"maximum"}`)); err == nil || !strings.Contains(err.Error(), "/capacity_posture") {
		t.Fatalf("invalid capacity posture error = %v", err)
	}
}

func TestAccelerationPreferenceDefaultsFromPostureAndValidates(t *testing.T) {
	tests := []struct {
		name string
		doc  Document
		want string
	}{
		{name: "ordinary posture", doc: Document{CapacityPosture: "balanced"}, want: AccelPreferenceAuto},
		{name: "minimal posture", doc: Document{CapacityPosture: "minimal"}, want: AccelPreferenceForceCPU},
		{name: "explicit gpu", doc: Document{CapacityPosture: "minimal", AccelPreference: AccelPreferencePreferGPU}, want: AccelPreferencePreferGPU},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.doc.EffectiveAccelPreference(); got != tt.want {
				t.Fatalf("EffectiveAccelPreference() = %q, want %q", got, tt.want)
			}
		})
	}

	service, _ := testService(t)
	if _, err := service.Apply(context.Background(), []byte(`{"accel_preference":"fastest"}`)); err == nil || !strings.Contains(err.Error(), "/accel_preference") {
		t.Fatalf("invalid acceleration preference error = %v", err)
	}
}

func TestCapacityChoicesSurviveLedgerWipe(t *testing.T) {
	service, root := testService(t)
	ctx := context.Background()
	if _, err := service.Apply(ctx, []byte(`{"resources":{"ollama":{"capacity":{"rung":"qwen3.5:4b","priority":"interactive","tunables":{"num_parallel":2}}}}}`)); err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(root, ".vrooli", "capacity.db")
	if err := os.WriteFile(ledger, []byte("replaceable observations"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(ledger); err != nil {
		t.Fatal(err)
	}
	doc, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	choice := doc.Resources["ollama"].Capacity
	if choice == nil || choice.Rung != "qwen3.5:4b" || choice.Priority != "interactive" || choice.Tunables["num_parallel"] != float64(2) {
		t.Fatalf("capacity choice after ledger wipe = %#v", choice)
	}
}

func TestDisjointConcurrentPatchesBothLand(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	var group sync.WaitGroup
	errs := make(chan error, 2)
	for _, patch := range []string{`{"scenarios":{"alpha":{"enabled":true}}}`, `{"resources":{"ollama":{"enabled":true}}}`} {
		group.Add(1)
		go func(patch string) {
			defer group.Done()
			_, err := service.Apply(ctx, []byte(patch))
			errs <- err
		}(patch)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent apply: %v", err)
		}
	}
	doc, err := service.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Scenarios["alpha"].Enabled == nil || !*doc.Scenarios["alpha"].Enabled || doc.Resources["ollama"].Enabled == nil || !*doc.Resources["ollama"].Enabled {
		t.Fatalf("disjoint patches did not both land: %#v", doc)
	}
}

func TestApplyAtRevisionRejectsStaleWriterWithoutChangingState(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	initial, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, []byte(`{"resources":{"ollama":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyAtRevision(ctx, Revision(initial), []byte(`{"scenarios":{"alpha":{"auto_restart":true}}}`)); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale apply error = %v, want revision conflict", err)
	}
	current, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.Scenarios["alpha"].AutoRestart != nil {
		t.Fatalf("stale patch changed state: %#v", current.Scenarios["alpha"])
	}
}

func TestSaveDraftIsTargetScopedAndDoesNotChangeEffectiveChoices(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	effective, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := service.SaveDraft(ctx, "target-a", "actor-a", Revision(effective), Revision(effective), "resources", map[string]string{"scenario.alpha": "disabled"})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Scenarios["alpha"].Enabled == nil || !*saved.Scenarios["alpha"].Enabled {
		t.Fatalf("draft changed effective selection: %#v", saved.Scenarios)
	}
	if len(saved.Drafts) != 1 || saved.Drafts[DraftKey("target-a", "actor-a")].Choices["scenario.alpha"] != "disabled" {
		t.Fatalf("draft was not persisted: %#v", saved.Drafts)
	}
	if _, ok := saved.Drafts[DraftKey("target-b", "actor-a")]; ok {
		t.Fatal("draft leaked across target scope")
	}
}

func TestSaveDraftRejectsSecretLikeChoicesAndStaleClients(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	first, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDraft(ctx, "target-a", "actor-a", Revision(first), Revision(first), "scenarios", map[string]string{"api_token": "must-not-persist"}); err == nil {
		t.Fatal("secret-like draft choice was accepted")
	}
	if _, err := service.SaveDraft(ctx, "target-a", "actor-a", "stale", Revision(first), "scenarios", map[string]string{"scenario.alpha": "enabled"}); !errors.Is(err, ErrDraftConflict) {
		t.Fatalf("stale draft error = %v, want draft conflict", err)
	}
}

func TestSaveProfileSessionIsBoundedNonSecretAndRevisionChecked(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	initial, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]json.RawMessage{
		"purposes": json.RawMessage(`["develop-apps"]`),
		"hosting":  json.RawMessage(`"managed-vps"`),
	}
	saved, err := service.SaveProfileSession(ctx, ProfileSession{
		Target: "target-a", Actor: "actor-a", Mode: "guided", ProfileID: "develop-and-publish",
		ProfileVersion: "1.0.0", CatalogRevision: "catalog-r1", BaseRevision: Revision(initial),
		Answers: answers, ManualDecisions: map[string]bool{"optional-tool": false},
		TargetContext: map[string]string{"operation": "prepare-desktop-release"},
	}, Revision(initial))
	if err != nil {
		t.Fatalf("save profile session: %v", err)
	}
	if saved.Session == nil || saved.Session.Profile == nil || saved.Session.Profile.ProfileID != "develop-and-publish" {
		t.Fatalf("profile session was not persisted: %#v", saved.Session)
	}
	loaded, err := service.ProfileSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Answers["purposes"][0] = 'x'
	again, err := service.ProfileSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Answers["purposes"]) != `["develop-apps"]` {
		t.Fatalf("profile session returned mutable answer bytes: %s", again.Answers["purposes"])
	}
	if _, err := service.SaveProfileSession(ctx, ProfileSession{Target: "target-a", Actor: "actor-a", Mode: "guided", BaseRevision: Revision(saved)}, "stale"); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale profile session error = %v, want revision conflict", err)
	}
	if _, err := service.SaveProfileSession(ctx, ProfileSession{Target: "target-a", Actor: "actor-a", Mode: "guided", BaseRevision: Revision(saved), Answers: map[string]json.RawMessage{"api_token": json.RawMessage(`"no"`)}}, Revision(saved)); err == nil {
		t.Fatal("secret-like profile answer was accepted")
	}
}

func TestDiscardDraftLeavesCommittedConfigurationUntouched(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	initial, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDraft(ctx, "target-a", "actor-a", Revision(initial), Revision(initial), "review", map[string]string{"scenario.alpha": "enabled"}); err != nil {
		t.Fatal(err)
	}
	discarded, err := service.DiscardDraft(ctx, "target-a", "actor-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(discarded.Drafts) != 0 || discarded.Scenarios["alpha"].Enabled == nil || !*discarded.Scenarios["alpha"].Enabled {
		t.Fatalf("discard changed committed state: %#v", discarded)
	}
}

func TestPruneDraftsUsesControlledClockAndPreservesUnreadableDrafts(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 1, 0, 0, 0, time.UTC)
	service.cfg.Now = func() time.Time { return now }
	initial, err := service.Apply(ctx, []byte(`{"scenarios":{"alpha":{"enabled":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDraft(ctx, "target-old", "actor-a", Revision(initial), Revision(initial), "scenarios", map[string]string{"scenario.alpha": "disabled"}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * 24 * time.Hour)
	current, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	drafts := current.Drafts
	drafts[DraftKey("target-b", "actor-a")] = Draft{Target: "target-b", Actor: "actor-a", BaseRevision: "r", Revision: "r", UpdatedAt: "not-a-timestamp", Choices: map[string]string{"scenario.alpha": "enabled"}}
	patch, err := json.Marshal(map[string]any{"drafts": drafts})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyAtRevision(ctx, Revision(current), patch); err != nil {
		t.Fatal(err)
	}
	pruned, err := service.PruneDrafts(ctx, 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pruned.Drafts[DraftKey("target-old", "actor-a")]; ok {
		t.Fatalf("expired draft was retained: %#v", pruned.Drafts)
	}
	if _, ok := pruned.Drafts[DraftKey("target-b", "actor-a")]; !ok {
		t.Fatal("unreadable draft was removed")
	}
	if pruned.Scenarios["alpha"].Enabled == nil || !*pruned.Scenarios["alpha"].Enabled {
		t.Fatal("pruning changed committed configuration")
	}
}

func TestOnlyOperatorStatePackageWritesTheOperatorState(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	var writers []string
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		if strings.Contains(source, "operator-state.json") && (strings.Contains(source, "WriteFile") || strings.Contains(source, "os.Create") || strings.Contains(source, "os.OpenFile")) && !strings.HasSuffix(path, filepath.Join("internal", "operatorstate", "operatorstate.go")) {
			writers = append(writers, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan source: %v", err)
	}
	if len(writers) != 0 {
		t.Fatalf("operator-state path has writers outside operatorstate: %v", writers)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "internal", "operatorstate", "operatorstate.go"))
	if err != nil {
		t.Fatalf("read operatorstate writer: %v", err)
	}
	if !strings.Contains(string(data), "storage.WriteFileAtomic") {
		t.Fatal("operatorstate package does not contain the atomic writer")
	}
}

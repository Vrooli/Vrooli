package operatingmode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestResolvedEnvelopeSurvivesSaveReloadAndRoutesNovelFields(t *testing.T) {
	store := testStore(t)
	def := MustDefinition(ModePhasedPlanDrain)
	phase := def.PhaseGraph.Phases["execute"]
	originalPhase := phase
	originalGuards := append([]GuardedTransition(nil), def.PhaseGraph.Guards["execute"]...)
	defer func() {
		def.PhaseGraph.Phases["execute"] = originalPhase
		def.PhaseGraph.Guards["execute"] = originalGuards
	}()
	phase.DeclaredOutput = &DeclaredOutput{
		EnvelopeKey:              resultEnvelopeKey,
		RequiresStructuredResult: true,
		Fields: []OutputField{
			{Name: "novel_flag", Type: "boolean", Required: true},
			{Name: "details", Type: "object", Required: true, Fields: []OutputField{{Name: "label", Type: "string", Required: true}}},
			{Name: "values", Type: "array", Required: true},
		},
		Resolution: defaultResolutionPolicy(),
	}
	phase.OutputContract = PhaseOutputContract{RequiresStructuredResult: true}
	def.PhaseGraph.Phases["execute"] = phase
	def.PhaseGraph.Guards["execute"] = []GuardedTransition{{
		When: Guard{Op: GuardOpEq, Field: "novel_flag", Value: true},
		To:   []Phase{"execute"},
	}}

	round, err := store.CreateRound(RoundEnvelope{ScopeID: "exec-novel", Mode: string(ModePhasedPlanDrain), Phase: "execute"})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	svc := &Service{store: store, clock: store.Clock}
	output := `{"operating_mode_result":{"novel_flag":true,"details":{"label":"preserved"},"values":[1,"two",false]}}`
	resolved, err := svc.applyPhaseResultWithPersistence(context.Background(), def, &round, []resolutionCandidate{{
		Content: output, EventID: "event-novel", Sequence: 42,
	}}, false)
	if err != nil {
		t.Fatalf("applyPhaseResult: %v", err)
	}
	if !resolved.Resolved() {
		t.Fatalf("resolution abstained: %+v", resolved)
	}
	round.Status = RoundStatusCompleted
	if err := store.SaveRound(round); err != nil {
		t.Fatalf("SaveRound: %v", err)
	}

	loaded, err := store.LoadRound("exec-novel", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	lookup := RoundPayload(loaded.Payload).ResultFieldLookup()
	if got, ok := lookup.Lookup("details.label"); !ok || got != "preserved" {
		t.Fatalf("details.label = %#v (present=%v), want preserved", got, ok)
	}
	if got, ok := lookup.Lookup("values"); !ok || len(got.([]any)) != 3 {
		t.Fatalf("values = %#v (present=%v), want 3 entries", got, ok)
	}
	next := nextPhasesForCompletedRound(def, loaded)
	if len(next) != 1 || next[0] != "execute" {
		t.Fatalf("next phases after reload = %v, want [execute]", next)
	}
	record, ok := RoundPayload(loaded.Payload).Resolution()
	if !ok || record.SelectedMessage == nil || record.SelectedMessage.EventID != "event-novel" || record.SelectedMessage.Sequence != 42 {
		t.Fatalf("selected message provenance = %+v (present=%v)", record.SelectedMessage, ok)
	}
}

func TestLegacyRoundWithoutResolvedEnvelopeStillRoutesFromPayloadProjection(t *testing.T) {
	store := testStore(t)
	round, err := store.CreateRound(RoundEnvelope{
		ScopeID: "legacy-round", Mode: string(ModePhasedPlanDrain), Phase: "execute",
		Status: RoundStatusCompleted, Payload: map[string]any{"progress": "continue"},
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	loaded, err := store.LoadRound("legacy-round", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if envelope, ok := payloadEnvelopeMap(loaded.Payload); ok || envelope != nil {
		t.Fatalf("legacy fixture unexpectedly gained canonical envelope: %#v", envelope)
	}
	if got, ok := RoundPayload(loaded.Payload).ResultFieldLookup().Lookup("progress"); !ok || got != "continue" {
		t.Fatalf("legacy progress = %#v (present=%v), want continue", got, ok)
	}
	if next := nextPhasesForCompletedRound(MustDefinition(ModePhasedPlanDrain), loaded); len(next) != 1 || next[0] != "execute" {
		t.Fatalf("legacy next phases = %v, want [execute]", next)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	return &Store{
		InitDir: func(name string) string {
			return filepath.Join(root, "initiatives", name)
		},
		TargetDir: func(kind TargetKind, scopeID string) string {
			return TargetScopeDir(filepath.Join(root, "data"), kind, scopeID)
		},
		Clock: func() time.Time {
			return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		},
	}
}

func TestStore_ModeScopedPaths(t *testing.T) {
	store := testStore(t)

	modeDir, err := store.ModeDir("sandboxing", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("ModeDir: %v", err)
	}
	if filepath.ToSlash(modeDir) == "modes/holistic-loop" {
		t.Fatalf("ModeDir returned relative path: %s", modeDir)
	}
	if got := filepath.ToSlash(modeDir); !hasSuffix(got, "initiatives/sandboxing/modes/holistic-loop") {
		t.Fatalf("ModeDir = %s", got)
	}

	// Plan-target rounds never live under an initiative folder: the store
	// routes them through the target directory resolver.
	roundPath, err := store.RoundPath("exec-1234", ModePhasedPlanDrain, 7)
	if err != nil {
		t.Fatalf("RoundPath: %v", err)
	}
	if got := filepath.ToSlash(roundPath); !hasSuffix(got, "data/mode-targets/plan-manager-plan/exec-1234/modes/phased-plan-drain/rounds/round-007.json") {
		t.Fatalf("RoundPath = %s", got)
	}
}

func TestStore_CreateListLoadRoundPreservesEnvelope(t *testing.T) {
	store := testStore(t)

	created, err := store.CreateRound(RoundEnvelope{
		ScopeID:         "sandboxing",
		Mode:            string(ModePhasedPlanDrain),
		Phase:           "execute",
		RunID:           "run-123",
		Status:          RoundStatusAgentRunning,
		AgentProfileKey: ProfileDeepWork,
		Handoffs: []Handoff{{
			Summary:         "Completed phase 1",
			CompletedPhases: []string{"phase-1"},
			ChangedFiles:    []string{"api/main.go"},
			Tests:           []string{"go test ./..."},
			NextStep:        "Continue with phase 2",
		}},
		ArtifactUpdates: []ArtifactUpdate{{Path: "modes/phased-plan-drain/round-summary.json", ContentType: "application/json"}},
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if created.Round != 1 {
		t.Fatalf("round number = %d, want 1", created.Round)
	}
	if created.RunStrategy != string(RunStrategySequentialHandoff) {
		t.Fatalf("run strategy = %q", created.RunStrategy)
	}

	created.Status = RoundStatusCompleted
	if err := store.SaveRound(created); err != nil {
		t.Fatalf("SaveRound: %v", err)
	}

	second, err := store.CreateRound(RoundEnvelope{
		ScopeID: "sandboxing",
		Mode:    string(ModePhasedPlanDrain),
		Phase:   "execute",
	})
	if err != nil {
		t.Fatalf("CreateRound second: %v", err)
	}
	if second.Round != 2 {
		t.Fatalf("second round = %d, want 2", second.Round)
	}
	if second.AgentProfileKey != ProfileDeepWork {
		t.Fatalf("defaulted profile = %q, want %q", second.AgentProfileKey, ProfileDeepWork)
	}

	rounds, err := store.ListRounds("sandboxing", ModePhasedPlanDrain)
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 2 {
		t.Fatalf("round count = %d, want 2", len(rounds))
	}
	if got := rounds[0].Handoffs[0].CompletedPhases[0]; got != "phase-1" {
		t.Fatalf("handoff not preserved: %q", got)
	}
	if rounds[0].AgentProfileKey != ProfileDeepWork {
		t.Fatalf("profile not preserved: %q", rounds[0].AgentProfileKey)
	}

	loaded, err := store.LoadRound("sandboxing", ModePhasedPlanDrain, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if loaded.Status != RoundStatusCompleted {
		t.Fatalf("loaded status = %q", loaded.Status)
	}
}

func TestStoreCompareAndCreateRoundRejectsStaleReplay(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-compare-reserve" }
	def := MustDefinition(ModePhasedPlanDrain)
	execution, err := store.CreateExecution("plan-compare-reserve", def)
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	proposed, version, err := store.PrepareRound(RoundEnvelope{
		ExecutionID: execution.ExecutionID, DefinitionDigest: execution.DefinitionDigest,
		ScopeID: execution.ScopeID, Mode: execution.Mode, Phase: "execute",
	})
	if err != nil {
		t.Fatalf("PrepareRound: %v", err)
	}
	if proposed.Round != 1 || version == "" {
		t.Fatalf("preflight = round:%d version:%q", proposed.Round, version)
	}
	roundsDir, err := store.ExecutionRoundsDir(execution)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(roundsDir); !os.IsNotExist(err) {
		t.Fatalf("read-only preflight created rounds directory: %v", err)
	}
	type result struct {
		round RoundEnvelope
		err   error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, createErr := store.CompareAndCreateRound(proposed, version)
			results <- result{round: created, err: createErr}
		}()
	}
	wg.Wait()
	close(results)
	succeeded, stale := 0, 0
	for got := range results {
		switch {
		case got.err == nil:
			succeeded++
			if got.round.Round != proposed.Round {
				t.Fatalf("created round = %d, want proposed %d", got.round.Round, proposed.Round)
			}
		case errors.Is(got.err, ErrRoundPreflightStale):
			stale++
		default:
			t.Fatalf("compare-and-reserve error = %v", got.err)
		}
	}
	if succeeded != 1 || stale != 1 {
		t.Fatalf("racing reservations = succeeded:%d stale:%d, want 1/1", succeeded, stale)
	}
	rounds, err := store.ListRounds(execution.ScopeID, def.Mode)
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 || rounds[0].Round != 1 {
		t.Fatalf("stale replay created duplicate rounds: %+v", rounds)
	}
}

func TestStoreCompareAndCreateRoundRejectsExecutionVersionChange(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-version-change" }
	def := MustDefinition(ModePhasedPlanDrain)
	execution, err := store.CreateExecution("plan-version-change", def)
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	proposed, version, err := store.PrepareRound(RoundEnvelope{
		ExecutionID: execution.ExecutionID, DefinitionDigest: execution.DefinitionDigest,
		ScopeID: execution.ScopeID, Mode: execution.Mode, Phase: "execute",
	})
	if err != nil {
		t.Fatalf("PrepareRound: %v", err)
	}
	execution.PromptPolicyMetadata = map[string]any{"policy": "changed-after-preflight"}
	if err := store.SaveExecution(execution); err != nil {
		t.Fatalf("SaveExecution: %v", err)
	}
	if _, err := store.CompareAndCreateRound(proposed, version); !errors.Is(err, ErrRoundPreflightStale) {
		t.Fatalf("execution-version error = %v, want ErrRoundPreflightStale", err)
	}
	rounds, err := store.ListRounds(execution.ScopeID, def.Mode)
	if err != nil || len(rounds) != 0 {
		t.Fatalf("execution-version race rounds = %+v err=%v, want none", rounds, err)
	}
}

func TestStore_ListRoundsRejectsMalformedJSON(t *testing.T) {
	store := testStore(t)
	dir, err := store.RoundsDir("sandboxing", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("RoundsDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "round-001.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := store.ListRounds("sandboxing", ModeHolisticLoop); err == nil {
		t.Fatal("ListRounds accepted malformed JSON")
	}
}

func TestStore_LoadRoundNotFound(t *testing.T) {
	store := testStore(t)
	_, err := store.LoadRound("sandboxing", ModeHolisticLoop, 42)
	if !errors.Is(err, ErrRoundNotFound) {
		t.Fatalf("LoadRound err = %v, want ErrRoundNotFound", err)
	}
}

func hasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}

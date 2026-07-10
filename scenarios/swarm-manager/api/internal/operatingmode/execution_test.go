package operatingmode

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPinDefinitionBundleIsTransitiveImmutableAndDigestStable(t *testing.T) {
	root, err := clonePinnedDefinition(MustDefinition(ModeHolisticLoop))
	if err != nil {
		t.Fatalf("clone root: %v", err)
	}
	child, err := clonePinnedDefinition(MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("clone child: %v", err)
	}
	resolve := func(mode Mode) (Definition, error) {
		if mode == ModePhasedPlanDrain {
			return child, nil
		}
		return Definition{}, errors.New("unexpected mode")
	}
	bundle, digest, err := pinDefinitionBundle(root, resolve)
	if err != nil {
		t.Fatalf("pinDefinitionBundle: %v", err)
	}
	if len(bundle.Definitions) != 2 {
		t.Fatalf("definition count = %d, want parent + delegated child", len(bundle.Definitions))
	}
	if _, err := bundle.Definition(ModePhasedPlanDrain); err != nil {
		t.Fatalf("delegated definition missing: %v", err)
	}

	phase := root.PhaseGraph.Phases["investigate"]
	phase.ActivityPurpose = "mutated-live-registry-value"
	root.PhaseGraph.Phases["investigate"] = phase
	pinned, err := bundle.RootDefinition()
	if err != nil {
		t.Fatalf("RootDefinition: %v", err)
	}
	if pinned.PhaseGraph.Phases["investigate"].ActivityPurpose == "mutated-live-registry-value" {
		t.Fatal("pinned bundle changed after source definition mutation")
	}

	rootWithDifferentExamples, err := clonePinnedDefinition(MustDefinition(ModeHolisticLoop))
	if err != nil {
		t.Fatalf("clone examples root: %v", err)
	}
	rootWithDifferentExamples.ExampleRuns = append(rootWithDifferentExamples.ExampleRuns, ExampleRun{ID: "non-runtime-fixture"})
	_, digestWithDifferentExamples, err := pinDefinitionBundle(rootWithDifferentExamples, resolve)
	if err != nil {
		t.Fatalf("pin bundle with examples: %v", err)
	}
	if digestWithDifferentExamples != digest {
		t.Fatalf("definition digest changed for non-runtime examples: %s != %s", digestWithDifferentExamples, digest)
	}
}

func TestExecutionManifestPersistsDefinitionAndProvenanceSlots(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-001" }
	execution, err := store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	if execution.ExecutionID != "execution-001" || execution.Status != ExecutionStatusActive {
		t.Fatalf("execution = %+v", execution)
	}
	if execution.DefinitionDigest == "" || execution.SchemaVersion != executionManifestSchemaVersion {
		t.Fatalf("manifest provenance = %+v", execution)
	}
	if execution.CompiledInputContract != nil || execution.ValidatedInputSnapshot != nil || execution.ReachablePromptSources != nil {
		t.Fatalf("Phase 4 provenance slots must start empty: %+v", execution)
	}
	path, err := store.executionManifestPath(execution, MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("executionManifestPath: %v", err)
	}
	if got := filepath.ToSlash(path); !hasSuffix(got, "data/mode-targets/plan-manager-plan/plan-123/modes/phased-plan-drain/executions/execution-001/manifest.json") {
		t.Fatalf("manifest path = %s", got)
	}

	execution.Status = ExecutionStatusCompleted
	if err := store.SaveExecution(execution); err != nil {
		t.Fatalf("SaveExecution: %v", err)
	}
	loaded, err := store.LoadExecution("plan-123", ModePhasedPlanDrain, execution.ExecutionID)
	if err != nil {
		t.Fatalf("LoadExecution: %v", err)
	}
	if loaded.Status != ExecutionStatusCompleted || loaded.CompletedAt == "" {
		t.Fatalf("loaded execution = %+v, want completed timestamp", loaded)
	}
	if _, err := loaded.DefinitionBundle.RootDefinition(); err != nil {
		t.Fatalf("loaded pinned definition: %v", err)
	}
	executions, err := store.ListExecutions("plan-123", ModePhasedPlanDrain)
	if err != nil || len(executions) != 1 {
		t.Fatalf("ListExecutions = %+v, %v", executions, err)
	}
}

func TestRunOwnerIndexIsIdempotentAndRejectsDualOwnership(t *testing.T) {
	store := testStore(t)
	ids := []string{"execution-001", "execution-002"}
	store.ExecutionID = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	first, err := store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution first: %v", err)
	}
	second, err := store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution second: %v", err)
	}
	if err := store.IndexRunOwner(first, "run-123", 1); err != nil {
		t.Fatalf("IndexRunOwner first: %v", err)
	}
	if err := store.IndexRunOwner(first, "run-123", 1); err != nil {
		t.Fatalf("IndexRunOwner replay: %v", err)
	}
	owner, err := store.ResolveRunOwner("plan-123", ModePhasedPlanDrain, "run-123")
	if err != nil || owner.ExecutionID != first.ExecutionID || owner.Round != 1 {
		t.Fatalf("ResolveRunOwner = %+v, %v", owner, err)
	}
	if err := store.IndexRunOwner(second, "run-123", 1); !errors.Is(err, ErrRunOwnerAmbiguous) {
		t.Fatalf("dual ownership error = %v, want ErrRunOwnerAmbiguous", err)
	}
	owner, err = store.ResolveRunOwner("plan-123", ModePhasedPlanDrain, "run-123")
	if err != nil || owner.ExecutionID != first.ExecutionID {
		t.Fatalf("ambiguous write changed existing owner: %+v, %v", owner, err)
	}
}

func TestExecutionRoundUsesPinnedLayoutAndRemainsLegacyAddressable(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-001" }
	execution, err := store.CreateExecution("plan-123", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	round, err := store.CreateRound(RoundEnvelope{
		ExecutionID: execution.ExecutionID, DefinitionDigest: execution.DefinitionDigest,
		ScopeID: "plan-123", Mode: string(ModePhasedPlanDrain), Phase: "execute",
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if round.ExecutionID != execution.ExecutionID || round.DefinitionDigest != execution.DefinitionDigest || round.Round != 1 {
		t.Fatalf("round = %+v", round)
	}
	executionPath, err := store.ExecutionRoundPath(execution, round.Round)
	if err != nil {
		t.Fatalf("ExecutionRoundPath: %v", err)
	}
	if _, err := os.Stat(executionPath); err != nil {
		t.Fatalf("execution round missing: %v", err)
	}
	legacyPath, err := store.RoundPath("plan-123", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("RoundPath: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy path unexpectedly written: %v", err)
	}
	loaded, err := store.LoadRound("plan-123", ModePhasedPlanDrain, round.Round)
	if err != nil {
		t.Fatalf("LoadRound compatibility lookup: %v", err)
	}
	if loaded.ExecutionID != execution.ExecutionID {
		t.Fatalf("loaded execution id = %q", loaded.ExecutionID)
	}
	rounds, err := store.ListRounds("plan-123", ModePhasedPlanDrain)
	if err != nil || len(rounds) != 1 || rounds[0].ExecutionID != execution.ExecutionID {
		t.Fatalf("ListRounds = %+v, %v", rounds, err)
	}
}

func TestContinueOrCreateExecutionRejectsMultipleResumableManifests(t *testing.T) {
	store := testStore(t)
	ids := []string{"execution-001", "execution-002", "execution-003"}
	store.ExecutionID = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	def := MustDefinition(ModePhasedPlanDrain)
	first, err := store.ContinueOrCreateExecution("plan-123", def)
	if err != nil {
		t.Fatalf("ContinueOrCreateExecution first: %v", err)
	}
	replayed, err := store.ContinueOrCreateExecution("plan-123", def)
	if err != nil || replayed.ExecutionID != first.ExecutionID {
		t.Fatalf("replayed execution = %+v, %v", replayed, err)
	}
	if _, err := store.CreateExecution("plan-123", def); err != nil {
		t.Fatalf("CreateExecution conflicting fixture: %v", err)
	}
	if _, err := store.ContinueOrCreateExecution("plan-123", def); !errors.Is(err, ErrExecutionAmbiguous) {
		t.Fatalf("multiple resumable error = %v, want ErrExecutionAmbiguous", err)
	}
}

func TestAdoptLegacyExecutionPinsRoundsPreservesBackupAndIndexesRuns(t *testing.T) {
	store := testStore(t)
	def := MustDefinition(ModePhasedPlanDrain)
	legacy, err := store.CreateRound(RoundEnvelope{
		ScopeID: "plan-legacy", Mode: string(def.Mode), Phase: "execute",
		Status: RoundStatusAgentRunning, RunID: "run-legacy-1",
	})
	if err != nil {
		t.Fatalf("CreateRound legacy: %v", err)
	}
	legacyPath, err := store.RoundPath("plan-legacy", def.Mode, legacy.Round)
	if err != nil {
		t.Fatalf("RoundPath legacy: %v", err)
	}
	original, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("read original legacy round: %v", err)
	}

	execution, ambiguous, err := store.AdoptLegacyExecution("plan-legacy", def)
	if err != nil {
		t.Fatalf("AdoptLegacyExecution: %v", err)
	}
	if ambiguous || execution == nil {
		t.Fatalf("adoption = execution:%+v ambiguous:%v", execution, ambiguous)
	}
	if execution.Migration == nil || execution.Migration.RoundCount != 1 || execution.Status != ExecutionStatusActive {
		t.Fatalf("migrated execution = %+v", execution)
	}
	backupPath := filepath.Join(filepath.Dir(filepath.Dir(legacyPath)), "legacy-rounds", execution.ExecutionID, filepath.Base(legacyPath))
	backedUp, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read legacy backup: %v", err)
	}
	if string(backedUp) != string(original) {
		t.Fatal("legacy backup bytes changed during adoption")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("flat legacy round remained active after adoption: %v", err)
	}
	loaded, err := store.LoadRound("plan-legacy", def.Mode, legacy.Round)
	if err != nil {
		t.Fatalf("LoadRound migrated: %v", err)
	}
	if loaded.ExecutionID != execution.ExecutionID || loaded.DefinitionDigest != execution.DefinitionDigest {
		t.Fatalf("migrated round provenance = %+v", loaded)
	}
	owner, err := store.ResolveRunOwner("plan-legacy", def.Mode, legacy.RunID)
	if err != nil || owner.ExecutionID != execution.ExecutionID || owner.Round != legacy.Round {
		t.Fatalf("migrated run owner = %+v, %v", owner, err)
	}
}

func TestAdoptLegacyExecutionLeavesAmbiguousHistoryReadOnly(t *testing.T) {
	store := testStore(t)
	def := MustDefinition(ModePhasedPlanDrain)
	if _, err := store.CreateRound(RoundEnvelope{
		ScopeID: "plan-ambiguous", Mode: string(def.Mode), Phase: "execute", Status: RoundStatusCanceled,
	}); err != nil {
		t.Fatalf("CreateRound canceled legacy: %v", err)
	}
	if _, err := store.CreateRound(RoundEnvelope{
		ScopeID: "plan-ambiguous", Mode: string(def.Mode), Phase: "execute", Status: RoundStatusAgentRunning,
	}); err != nil {
		t.Fatalf("CreateRound later legacy: %v", err)
	}
	execution, ambiguous, err := store.AdoptLegacyExecution("plan-ambiguous", def)
	if err != nil {
		t.Fatalf("AdoptLegacyExecution: %v", err)
	}
	if execution != nil || !ambiguous {
		t.Fatalf("adoption = execution:%+v ambiguous:%v, want read-only ambiguity", execution, ambiguous)
	}
	executions, err := store.ListExecutions("plan-ambiguous", def.Mode)
	if err != nil || len(executions) != 0 {
		t.Fatalf("ListExecutions = %+v, %v", executions, err)
	}
	rounds, err := store.ListRounds("plan-ambiguous", def.Mode)
	if err != nil || len(rounds) != 2 || rounds[0].ExecutionID != "" || rounds[1].ExecutionID != "" {
		t.Fatalf("read-only legacy rounds = %+v, %v", rounds, err)
	}
}

func TestValidateExecutionManifestRejectsDefinitionDigestMismatch(t *testing.T) {
	bundle, digest, err := pinDefinitionBundle(MustDefinition(ModePhasedPlanDrain), DefinitionFor)
	if err != nil {
		t.Fatalf("pinDefinitionBundle: %v", err)
	}
	execution := OperatingModeExecution{
		ExecutionID: "execution-tampered", ScopeID: "plan-1", Mode: string(ModePhasedPlanDrain),
		SchemaVersion: executionManifestSchemaVersion, DefinitionDigest: digest, DefinitionBundle: bundle,
	}
	if err := validateExecutionManifest(execution); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	execution.DefinitionDigest = "sha256:tampered"
	if err := validateExecutionManifest(execution); err == nil {
		t.Fatal("tampered definition digest accepted")
	}
}

package operatingmode

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func callerExecutionDefinition(t *testing.T) Definition {
	t.Helper()
	def, err := clonePinnedDefinition(MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("clone phased-plan-drain: %v", err)
	}
	caller := callerInputContractDefinition()
	def.InputContract = caller.InputContract
	phase := def.PhaseGraph.Phases[def.PhaseGraph.StartPhase]
	phase.Reads = []string{"CALLER_PAYLOAD_JSON", "CALLER_LIMIT"}
	def.PhaseGraph.Phases[def.PhaseGraph.StartPhase] = phase
	return def
}

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
	if len(execution.CompiledInputContract) == 0 || execution.InputContractDigest == "" {
		t.Fatalf("compiled input provenance was not pinned at execution creation: %+v", execution)
	}
	var compiled CompiledInputContract
	if err := json.Unmarshal(execution.CompiledInputContract, &compiled); err != nil {
		t.Fatalf("decode compiled input contract: %v", err)
	}
	if compiled.RootMode != ModePhasedPlanDrain || len(compiled.Modes) != 1 {
		t.Fatalf("compiled input contract = %+v", compiled)
	}
	if string(execution.ValidatedInputSnapshot) != "{}" || execution.InputSnapshotDigest == "" || execution.ReachablePromptSources != nil {
		t.Fatalf("validated empty input snapshot must be pinned while prompt provenance remains pending: %+v", execution)
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

func TestCreateExecutionRejectsInvalidInputContractBeforeFilesystemMutation(t *testing.T) {
	store := testStore(t)
	def, err := clonePinnedDefinition(MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("clone definition: %v", err)
	}
	def.InputContract.Sources[0].Capability = "generic.capability-does-not-exist"
	if _, err := store.CreateExecution("plan-invalid", def); err == nil || !strings.Contains(err.Error(), "unavailable capability") {
		t.Fatalf("CreateExecution error = %v, want unavailable capability", err)
	}
	scopeDir, err := store.scopeDir("plan-invalid", def)
	if err != nil {
		t.Fatalf("scopeDir: %v", err)
	}
	modeDir := filepath.Join(scopeDir, filepath.FromSlash(def.Artifact.Root))
	if _, err := os.Stat(filepath.Join(modeDir, "executions")); !os.IsNotExist(err) {
		t.Fatalf("invalid preflight created an executions directory: %v", err)
	}
}

func TestCreateExecutionWithInputsPersistsValidatedSnapshotBeforeRounds(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-with-inputs" }
	def := callerExecutionDefinition(t)
	execution, err := store.CreateExecutionWithInputs("plan-inputs", def, map[string]any{
		"caller.payload": map[string]any{"message": "preserved"},
		"caller.limit":   4,
	})
	if err != nil {
		t.Fatalf("CreateExecutionWithInputs: %v", err)
	}
	if execution.InputSnapshotDigest == "" || len(execution.ValidatedInputSnapshot) == 0 || len(execution.InputRetentionMetadata) != 2 {
		t.Fatalf("execution input provenance = %+v", execution)
	}
	var snapshot map[string]any
	if err := json.Unmarshal(execution.ValidatedInputSnapshot, &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if payload, ok := snapshot["caller.payload"].(map[string]any); !ok || payload["message"] != "preserved" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	rounds, err := store.ListRounds("plan-inputs", def.Mode)
	if err != nil || len(rounds) != 0 {
		t.Fatalf("rounds after execution preflight = %+v, %v", rounds, err)
	}
	loaded, err := store.LoadExecution("plan-inputs", def.Mode, execution.ExecutionID)
	if err != nil || loaded.InputSnapshotDigest != execution.InputSnapshotDigest {
		t.Fatalf("loaded execution = %+v, %v", loaded, err)
	}
}

func TestCreateExecutionWithInputsRejectsInvalidValuesBeforeFilesystemMutation(t *testing.T) {
	store := testStore(t)
	def := callerExecutionDefinition(t)
	if _, err := store.CreateExecutionWithInputs("plan-invalid-inputs", def, map[string]any{
		"caller.payload": "wrong-type",
	}); err == nil || !strings.Contains(err.Error(), "want object") {
		t.Fatalf("CreateExecutionWithInputs error = %v, want type rejection", err)
	}
	scopeDir, err := store.scopeDir("plan-invalid-inputs", def)
	if err != nil {
		t.Fatalf("scopeDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(scopeDir, filepath.FromSlash(def.Artifact.Root), "executions")); !os.IsNotExist(err) {
		t.Fatalf("invalid caller input created execution state: %v", err)
	}
}

func TestContinueOrCreateExecutionWithInputsIsIdempotentAndRejectsConflict(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-input-replay" }
	def := callerExecutionDefinition(t)
	inputs := map[string]any{"caller.payload": map[string]any{"message": "same"}, "caller.limit": 2}
	first, err := store.ContinueOrCreateExecutionWithInputs("plan-replay", def, inputs)
	if err != nil {
		t.Fatalf("ContinueOrCreateExecutionWithInputs first: %v", err)
	}
	replayed, err := store.ContinueOrCreateExecutionWithInputs("plan-replay", def, map[string]any{
		"caller.limit": float64(2), "caller.payload": map[string]any{"message": "same"},
	})
	if err != nil || replayed.ExecutionID != first.ExecutionID {
		t.Fatalf("replayed execution = %+v, %v", replayed, err)
	}
	if _, err := store.ContinueOrCreateExecutionWithInputs("plan-replay", def, map[string]any{
		"caller.payload": map[string]any{"message": "changed"}, "caller.limit": 2,
	}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflicting replay error = %v", err)
	}
	executions, err := store.ListExecutions("plan-replay", def.Mode)
	if err != nil || len(executions) != 1 {
		t.Fatalf("executions after conflict = %+v, %v", executions, err)
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

func TestValidateExecutionManifestRejectsInputProvenanceDigestMismatch(t *testing.T) {
	store := testStore(t)
	execution, err := store.CreateExecution("plan-input-digests", MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	originalSnapshotDigest := execution.InputSnapshotDigest
	execution.InputSnapshotDigest = "sha256:tampered"
	if err := validateExecutionManifest(execution); err == nil || !strings.Contains(err.Error(), "input snapshot digest") {
		t.Fatalf("snapshot digest error = %v", err)
	}
	execution.InputSnapshotDigest = originalSnapshotDigest
	var compiled map[string]any
	if err := json.Unmarshal(execution.CompiledInputContract, &compiled); err != nil {
		t.Fatalf("decode compiled contract: %v", err)
	}
	compiled["schema_version"] = "tampered"
	execution.CompiledInputContract, _ = json.Marshal(compiled)
	if err := validateExecutionManifest(execution); err == nil || !strings.Contains(err.Error(), "input contract digest") {
		t.Fatalf("contract digest error = %v", err)
	}
}

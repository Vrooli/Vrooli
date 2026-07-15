package agentopsdiag

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

func itemTarget(ref string) *apipb.AgentOpsTargetSelector {
	return &apipb.AgentOpsTargetSelector{Kind: domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_BACKLOG_ITEM, Id: ref}
}

// putOverride is the write-path helper the override tests share.
func putOverride(t *testing.T, svc *Service, owner *apipb.AgentOpsTargetSelector, op, version, mode string) (*apipb.AgentOpsPutBindingOverrideResponse, error) {
	t.Helper()
	resp, err := svc.PutBindingOverride(context.Background(), connect.NewRequest(&apipb.AgentOpsPutBindingOverrideRequest{
		Owner: owner, Operation: op, OperationVersion: version, Mode: mode, ModeRevision: rev,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// --- ListOperationCatalog -------------------------------------------------

func TestListOperationCatalog(t *testing.T) {
	svc, _ := buildService(t)
	resp, err := svc.ListOperationCatalog(context.Background(), connect.NewRequest(&apipb.AgentOpsListOperationCatalogRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetEntries()) != 1 {
		t.Fatalf("expected the 1 authored contract, got %d", len(resp.Msg.GetEntries()))
	}
	e := resp.Msg.GetEntries()[0]
	if e.GetContract().GetId() != "review-round" || e.GetContract().GetVersion() != "1.0.0" {
		t.Fatalf("contract identity = %s@%s", e.GetContract().GetId(), e.GetContract().GetVersion())
	}
	if e.GetRevision() == "" {
		t.Fatal("catalog entry must pin a content revision")
	}
	if len(e.GetContract().GetOutcomes()) == 0 || len(e.GetContract().GetResultFields()) == 0 {
		t.Fatalf("contract must project outcomes + result fields: %+v", e.GetContract())
	}
	// review-round requires provides-review-artifacts, which backlog-item and
	// initiative both provide.
	if len(e.GetCompatibleTargets()) != 2 {
		t.Fatalf("compatible targets = %v", e.GetCompatibleTargets())
	}
}

// --- ListCompatibleModes ----------------------------------------------------

func TestListCompatibleModesVerdicts(t *testing.T) {
	svc, _ := buildService(t)
	resp, err := svc.ListCompatibleModes(context.Background(), connect.NewRequest(&apipb.AgentOpsListCompatibleModesRequest{
		Target: initTarget("ship-x"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	byMode := map[string]*apipb.AgentOpsCompatibleMode{}
	for _, m := range resp.Msg.GetModes() {
		byMode[m.GetMode()] = m
	}
	if len(byMode) != 2 {
		t.Fatalf("expected both authored modes with verdicts, got %v", byMode)
	}
	// The initiative-target mode can run review-round on an initiative...
	syn := byMode["synthetic-loop"]
	if syn.GetModeDigest() == "" || syn.GetModeRevision() != syn.GetModeDigest() {
		t.Fatalf("mode digest/revision must be pinned: %+v", syn)
	}
	if len(syn.GetVerdicts()) != 1 || !syn.GetVerdicts()[0].GetCompatible() {
		t.Fatalf("synthetic-loop must be compatible with review-round on initiative: %+v", syn.GetVerdicts())
	}
	// ...the item-target mode cannot, and the verdict says why.
	item := byMode["item-loop"]
	if item.GetVerdicts()[0].GetCompatible() || item.GetVerdicts()[0].GetReason() == "" {
		t.Fatalf("item-loop must be incompatible with a reason: %+v", item.GetVerdicts())
	}
}

func TestListCompatibleModesUnknownOperationFilter(t *testing.T) {
	svc, _ := buildService(t)
	_, err := svc.ListCompatibleModes(context.Background(), connect.NewRequest(&apipb.AgentOpsListCompatibleModesRequest{
		Target: initTarget("ship-x"), Operation: "no-such-op",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown operation filter must be InvalidArgument, got %v", err)
	}
}

func TestListCompatibleModesWithoutRegistryFailsClosed(t *testing.T) {
	svc, _ := buildService(t)
	svc.modeDefs, svc.checker = nil, nil
	_, err := svc.ListCompatibleModes(context.Background(), connect.NewRequest(&apipb.AgentOpsListCompatibleModesRequest{
		Target: initTarget("ship-x"),
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing mode registry must be FailedPrecondition, got %v", err)
	}
}

// --- GetResolvedBindings ----------------------------------------------------

func TestGetResolvedBindingsInitiativeOverrideWins(t *testing.T) {
	svc, _ := buildService(t)
	// Author an initiative override for review-round; the system default
	// (synthetic-loop) becomes the inherited-but-overridden contribution.
	if _, err := putOverride(t, svc, initTarget("ship-x"), "review-round", "", "synthetic-loop"); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.GetResolvedBindings(context.Background(), connect.NewRequest(&apipb.AgentOpsGetResolvedBindingsRequest{
		Target: initTarget("ship-x"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetOperations()) != 1 {
		t.Fatalf("expected 1 compatible operation, got %d", len(resp.Msg.GetOperations()))
	}
	row := resp.Msg.GetOperations()[0]
	if !row.GetResolved() || row.GetBinding().GetLayer() != domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE {
		t.Fatalf("initiative override must win: %+v", row)
	}
	if row.GetPolicyId() != "initiative-default" {
		t.Fatalf("policy must be pinned: %+v", row)
	}
	if len(row.GetContributions()) != 2 {
		t.Fatalf("expected system-default + initiative-override contributions, got %+v", row.GetContributions())
	}
	for _, c := range row.GetContributions() {
		isOverride := c.GetBinding().GetLayer() == domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE
		if c.GetWinning() != isOverride {
			t.Fatalf("winning flag must mark exactly the override layer: %+v", row.GetContributions())
		}
	}
}

func TestGetResolvedBindingsItemInheritsInitiativeOverride(t *testing.T) {
	svc, _ := buildService(t)
	// The initiative pins the item-target mode for review-round; item fix/it-1
	// (a ship-x member via InitiativeOfItem) must inherit it.
	if _, err := putOverride(t, svc, initTarget("ship-x"), "review-round", "", "item-loop"); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.GetResolvedBindings(context.Background(), connect.NewRequest(&apipb.AgentOpsGetResolvedBindingsRequest{
		Target: itemTarget("fix/it-1"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	row := resp.Msg.GetOperations()[0]
	if !row.GetResolved() || row.GetBinding().GetMode() != "item-loop" ||
		row.GetBinding().GetLayer() != domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE {
		t.Fatalf("item must inherit the initiative override: %+v", row)
	}
	// An unowned item sees only the system default (whose initiative-target
	// mode is incompatible with a backlog-item target -> typed failure).
	other, err := svc.GetResolvedBindings(context.Background(), connect.NewRequest(&apipb.AgentOpsGetResolvedBindingsRequest{
		Target: itemTarget("fix/loner"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := other.Msg.GetOperations()[0]; got.GetResolved() || got.GetError() != "incompatible-mode" {
		t.Fatalf("unowned item must fail closed as a typed per-operation result: %+v", got)
	}
}

// --- ListBindingOverrides / Put / Delete ------------------------------------

func TestPutListDeleteBindingOverride(t *testing.T) {
	svc, _ := buildService(t)
	owner := initTarget("ship-x")

	put, err := putOverride(t, svc, owner, "review-round", "", "synthetic-loop")
	if err != nil {
		t.Fatal(err)
	}
	if put.GetFile() != "review-round.json" || put.GetRevision() == "" {
		t.Fatalf("put must report deterministic file + revision: %+v", put)
	}
	if put.GetStored().GetLayer() != domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE ||
		put.GetStored().GetOwner().GetId() != "ship-x" {
		t.Fatalf("layer/owner must be derived server-side: %+v", put.GetStored())
	}

	// Idempotent replace: same operation+version writes the same file.
	again, err := putOverride(t, svc, owner, "review-round", "", "synthetic-loop")
	if err != nil {
		t.Fatal(err)
	}
	if again.GetFile() != put.GetFile() {
		t.Fatalf("replace must keep the deterministic filename: %q vs %q", again.GetFile(), put.GetFile())
	}

	list, err := svc.ListBindingOverrides(context.Background(), connect.NewRequest(&apipb.AgentOpsListBindingOverridesRequest{Owner: owner}))
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Msg.GetOverrides()) != 1 {
		t.Fatalf("expected exactly one stored override, got %+v", list.Msg.GetOverrides())
	}
	doc := list.Msg.GetOverrides()[0]
	if doc.GetFile() != "review-round.json" || doc.GetRevision() == "" || doc.GetUpdatedAt() == "" {
		t.Fatalf("listing must carry file-level provenance: %+v", doc)
	}

	del, err := svc.DeleteBindingOverride(context.Background(), connect.NewRequest(&apipb.AgentOpsDeleteBindingOverrideRequest{
		Owner: owner, Operation: "review-round",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !del.Msg.GetFound() {
		t.Fatal("delete of an existing override must report found=true")
	}
	// Deleting again is found=false, never an error.
	del2, err := svc.DeleteBindingOverride(context.Background(), connect.NewRequest(&apipb.AgentOpsDeleteBindingOverrideRequest{
		Owner: owner, Operation: "review-round",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if del2.Msg.GetFound() {
		t.Fatal("delete of an absent override must report found=false")
	}
}

func TestPutBindingOverrideFailClosed(t *testing.T) {
	svc, _ := buildService(t)
	owner := initTarget("ship-x")

	// Unknown operation.
	if _, err := putOverride(t, svc, owner, "no-such-op", "", "synthetic-loop"); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unknown operation must be FailedPrecondition, got %v", err)
	}
	// Declared operation, unauthored version pin.
	if _, err := putOverride(t, svc, owner, "review-round", "9.9.9", "synthetic-loop"); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unauthored version pin must be FailedPrecondition, got %v", err)
	}
	// Unregistered mode.
	if _, err := putOverride(t, svc, owner, "review-round", "", "ghost-mode"); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unregistered mode must be FailedPrecondition, got %v", err)
	}
	// Mode incompatible with every kind the override can apply to: an item
	// owner's override can only ever govern backlog-item targets, which the
	// initiative-target mode cannot serve.
	if _, err := putOverride(t, svc, itemTarget("fix/it-1"), "review-round", "", "synthetic-loop"); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("incompatible mode write must be FailedPrecondition, got %v", err)
	}
	// Owner kind that owns no override layer.
	planTarget := &apipb.AgentOpsTargetSelector{Kind: domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_PLAN_EXECUTION, Id: "p"}
	if _, err := putOverride(t, svc, planTarget, "review-round", "", "synthetic-loop"); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("plan-execution owner must be InvalidArgument, got %v", err)
	}
	// No mode registry -> no writes, fail closed.
	svc.checker = nil
	if _, err := putOverride(t, svc, owner, "review-round", "", "synthetic-loop"); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("write without a mode registry must be FailedPrecondition, got %v", err)
	}
}

func TestPutBindingOverrideRejectsAmbiguity(t *testing.T) {
	svc, _ := buildService(t)
	owner := initTarget("ship-x")
	if _, err := putOverride(t, svc, owner, "review-round", "", "synthetic-loop"); err != nil {
		t.Fatal(err)
	}
	// A second override for the same operation under a different version key
	// would make every resolution ambiguous — refuse it.
	_, err := putOverride(t, svc, owner, "review-round", "1.0.0", "synthetic-loop")
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("ambiguity-introducing put must be FailedPrecondition, got %v", err)
	}
}

// --- GetWorkflowProjection ----------------------------------------------------

func TestGetWorkflowProjection(t *testing.T) {
	svc, loc := buildService(t)
	repo := opsrunner.NewWorkflowRepo(loc)
	execStore := opsrunner.NewExecutionStore(loc)
	kind, id := agentops.TargetInitiative, "ship-x"

	prov := func(execSuffix string) agentops.ExecutionProvenance {
		return agentops.ExecutionProvenance{
			Kind: "agentops-execution-provenance", Operation: agentops.OpReviewRound, OperationVersion: "1.0.0",
			Binding: agentops.ProvenanceBinding{Layer: agentops.LayerInitiativeOverride, OwnerKind: "initiative", OwnerID: id},
			Mode:    "synthetic-loop", ModeRevision: rev,
			CompiledModeDigest: digestOf(t, `{"m":1}`), PromptCatalogRevision: "pc-1", PromptCatalogDigest: digestOf(t, `{"p":1}`),
			Target: agentops.ProvenanceTarget{Kind: kind, ID: id}, CallerInputDigest: digestOf(t, `{}`),
			PolicyRevision: "pol-1", WorkflowInstanceID: "wf-initiative-ship-x" + execSuffix,
		}
	}

	w, _ := repo.CreateOrLoad(kind, id)
	next := w
	next.State = agentops.WorkflowRunning
	next.Operations = []agentops.OperationExecutionRecord{
		{Operation: agentops.OpReviewRound, ExecutionID: "exec-1", IdempotencyKey: "k1", ProvenanceDigest: mustDigestOf(t, prov("")), State: "failed", Outcome: "failed", RunID: "run-1"},
		{Operation: agentops.OpReviewRound, ExecutionID: "exec-2", IdempotencyKey: "k2", ProvenanceDigest: mustDigestOf(t, prov("")), State: "running", RunID: "run-2"},
	}
	next.Decisions = []agentops.HumanDecision{{Decision: "retry-review", Actor: "operator", AtVersion: 1, Note: "flaky evidence"}}
	next.Timers = []agentops.ScheduledIntent{{Intent: "auto-advance", Action: agentops.ActionOpenReview, NotBefore: "2026-07-15T00:00:00Z"}}
	next.LegalActions = []agentops.ActionName{agentops.ActionOpenReview}
	next.IdempotencyKeys = []string{"k1", "k2"}
	next.Version = 1
	if err := repo.Commit(0, next); err != nil {
		t.Fatal(err)
	}
	// Only the first attempt has a persisted snapshot; the second row must
	// degrade (snapshot_found=false), not fail the projection.
	if err := execStore.SaveExecution(kind, id, "exec-1", opsrunner.ExecutionSnapshot{
		Provenance: prov(""), CompiledMode: json.RawMessage(`{"m":1}`), PromptCatalog: json.RawMessage(`{"p":1}`),
		EffectiveInputs: json.RawMessage(`{}`), Outcome: "failed", RecordedAt: "2026-07-14T10:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := svc.GetWorkflowProjection(context.Background(), connect.NewRequest(&apipb.AgentOpsGetWorkflowProjectionRequest{Target: initTarget(id)}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetFound() {
		t.Fatal("workflow must be found")
	}
	wf := resp.Msg.GetWorkflow()
	if len(wf.GetDecisions()) != 1 || wf.GetDecisions()[0].GetDecision() != "retry-review" {
		t.Fatalf("decisions must be projected (mappers previously dropped them): %+v", wf.GetDecisions())
	}
	if len(wf.GetTimers()) != 1 || len(wf.GetLegalActions()) != 1 {
		t.Fatalf("timers + legal actions must be projected: %+v", wf)
	}
	if resp.Msg.GetPolicyId() != "initiative-default" {
		t.Fatalf("domain policy must be pinned: %q", resp.Msg.GetPolicyId())
	}
	ops := resp.Msg.GetOperations()
	if len(ops) != 2 {
		t.Fatalf("expected 2 operation projections, got %d", len(ops))
	}
	first, second := ops[0], ops[1]
	if !first.GetSnapshotFound() || first.GetMode() != "synthetic-loop" ||
		first.GetBindingLayer() != domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE ||
		first.GetRecordedAt() == "" || first.GetRunId() != "run-1" {
		t.Fatalf("first attempt must be snapshot-enriched: %+v", first)
	}
	if first.GetAttempt() != 1 || second.GetAttempt() != 2 || second.GetPriorExecutionId() != "exec-1" {
		t.Fatalf("retry linkage broken: attempt1=%d attempt2=%d prior=%q", first.GetAttempt(), second.GetAttempt(), second.GetPriorExecutionId())
	}
	if second.GetSnapshotFound() || second.GetMode() != "" {
		t.Fatalf("missing snapshot must degrade the row, not invent data: %+v", second)
	}
}

// --- ListExecutionHistory -----------------------------------------------------

func TestListExecutionHistoryNewestFirstWithLimit(t *testing.T) {
	svc, loc := buildService(t)
	execStore := opsrunner.NewExecutionStore(loc)
	kind, id := agentops.TargetInitiative, "ship-x"
	prov := agentops.ExecutionProvenance{
		Kind: "agentops-execution-provenance", Operation: agentops.OpReviewRound, OperationVersion: "1.0.0",
		Binding: agentops.ProvenanceBinding{Layer: agentops.LayerSystemDefault, OwnerKind: "system", OwnerID: "system"},
		Mode:    "synthetic-loop", ModeRevision: rev,
		CompiledModeDigest: digestOf(t, `{"m":1}`), PromptCatalogRevision: "pc-1", PromptCatalogDigest: digestOf(t, `{"p":1}`),
		Target: agentops.ProvenanceTarget{Kind: kind, ID: id}, CallerInputDigest: digestOf(t, `{}`),
		PolicyRevision: "pol-1", WorkflowInstanceID: "wf-initiative-ship-x",
	}
	save := func(execID, at string) {
		t.Helper()
		if err := execStore.SaveExecution(kind, id, execID, opsrunner.ExecutionSnapshot{
			Provenance: prov, CompiledMode: json.RawMessage(`{"m":1}`), PromptCatalog: json.RawMessage(`{"p":1}`),
			EffectiveInputs: json.RawMessage(`{}`), Outcome: "accepted", RecordedAt: at,
		}); err != nil {
			t.Fatal(err)
		}
	}
	save("exec-old", "2026-07-13T10:00:00Z")
	save("exec-new", "2026-07-14T10:00:00Z")

	resp, err := svc.ListExecutionHistory(context.Background(), connect.NewRequest(&apipb.AgentOpsListExecutionHistoryRequest{Target: initTarget(id)}))
	if err != nil {
		t.Fatal(err)
	}
	got := resp.Msg.GetExecutions()
	if len(got) != 2 || got[0].GetExecutionId() != "exec-new" || got[1].GetExecutionId() != "exec-old" {
		t.Fatalf("history must be newest-first: %+v", got)
	}
	if !got[0].GetReproducible() || got[0].GetMode() != "synthetic-loop" || got[0].GetCompiledModeDigest() == "" {
		t.Fatalf("summary must carry digests + reproducibility: %+v", got[0])
	}

	limited, err := svc.ListExecutionHistory(context.Background(), connect.NewRequest(&apipb.AgentOpsListExecutionHistoryRequest{Target: initTarget(id), Limit: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Msg.GetExecutions()) != 1 || limited.Msg.GetExecutions()[0].GetExecutionId() != "exec-new" {
		t.Fatalf("limit must keep the newest: %+v", limited.Msg.GetExecutions())
	}
}

// --- GetMigrationStatus ---------------------------------------------------------

func TestGetMigrationStatus(t *testing.T) {
	svc, _ := buildService(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "migration-status.json")
	svc.WithMigrationStatusPath(path)

	// Absent document -> not-started, never an error.
	resp, err := svc.GetMigrationStatus(context.Background(), connect.NewRequest(&apipb.AgentOpsGetMigrationStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetDocumentFound() || resp.Msg.GetState() != "not-started" {
		t.Fatalf("absent document must be not-started: %+v", resp.Msg)
	}

	doc := opsrunner.MigrationStatus{
		Kind: "agentops-migration-status", SchemaVersion: "1.0.0", State: opsrunner.MigrationStaged,
		Epoch: 2, StagedCount: 41, QuarantinedCount: 1,
		StartedAt: "2026-07-14T08:00:00Z", UpdatedAt: "2026-07-14T09:00:00Z",
		ReportPath: "docs/operations/migration/epoch-2.json",
	}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	resp, err = svc.GetMigrationStatus(context.Background(), connect.NewRequest(&apipb.AgentOpsGetMigrationStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetDocumentFound() || resp.Msg.GetState() != "staged" || resp.Msg.GetEpoch() != 2 ||
		resp.Msg.GetStagedCount() != 41 || resp.Msg.GetQuarantinedCount() != 1 ||
		resp.Msg.GetReportPath() == "" {
		t.Fatalf("staged status must project fully: %+v", resp.Msg)
	}

	// A present-but-invalid document fails closed rather than reading as
	// not-started.
	if err := os.WriteFile(path, []byte(`{"state":"weird"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetMigrationStatus(context.Background(), connect.NewRequest(&apipb.AgentOpsGetMigrationStatusRequest{})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("invalid status document must be a typed error, got %v", err)
	}
}

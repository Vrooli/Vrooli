package agentopsdiag

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

const rev = "sha256:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func buildService(t *testing.T) (*Service, opsrunner.FSLocator) {
	t.Helper()
	catalogDir := t.TempDir()
	write := func(rel string, v any) {
		p := filepath.Join(catalogDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(v, "", "  ")
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var review agentops.OperationContract
	for _, oc := range agentops.SeedOperationContracts() {
		if oc.ID == agentops.OpReviewRound {
			review = oc
		}
	}
	write(filepath.Join(opscatalog.DirOperationContracts, "review-round.json"), review)
	// The fixture binding pins an exact operation_version, exactly like every
	// shipped bindings/*.json document. A version-agnostic fixture here once
	// masked a live bug: diagnostics resolved with an empty version, which
	// skips version-pinned system defaults and reported no-binding for every
	// operation at rest (finding 80cb2437).
	write(filepath.Join(opscatalog.DirBindings, "review.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpReviewRound, OperationVersion: "1.0.0",
		Layer: agentops.LayerSystemDefault, Mode: "synthetic-loop", ModeRevision: rev,
	})
	write(filepath.Join(opscatalog.DirPolicy, "initiative.json"), agentops.TransitionPolicy{
		Kind: "agentops-transition-policy", ID: "initiative-default", Version: "1.0.0", DomainKind: "initiative",
		Transitions: []agentops.PolicyTransition{{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionOpenReview, ToState: "awaiting-decision"}},
	})
	catalog, err := opscatalog.Load(catalogDir)
	if err != nil {
		t.Fatal(err)
	}
	storeRoot := t.TempDir()
	loc := opsrunner.FSLocator{
		InitiativeDir:  func(name string) (string, error) { return filepath.Join(storeRoot, "initiatives", name), nil },
		BacklogItemDir: func(kind, name string) (string, error) { return filepath.Join(storeRoot, "backlog", kind, name), nil },
		ScenarioDir:    func(name string) (string, error) { return filepath.Join(storeRoot, "scenarios", name), nil },
	}
	repo := opsrunner.NewWorkflowRepo(loc)
	execStore := opsrunner.NewExecutionStore(loc)

	// Mode registry + checker, wired exactly as production does (the same
	// LivePreparer construction is both the mode registry answer and the
	// resolver's checker): one initiative-target mode and one item-target mode.
	defs := map[string]operatingmode.Definition{
		"synthetic-loop": {Mode: "synthetic-loop", Target: operatingmode.TargetPolicy{Kind: operatingmode.TargetInitiative}},
		"item-loop":      {Mode: "item-loop", Target: operatingmode.TargetPolicy{Kind: operatingmode.TargetBacklogItem}},
	}
	checker := opsrunner.NewLivePreparer(catalog, defs)

	// Item fix/it-1 belongs to initiative ship-x; every other item is unowned.
	overrides := opsrunner.NewFSOverrideStore(loc)
	overrides.InitiativeOfItem = func(itemRef string) (string, error) {
		if itemRef == "fix/it-1" {
			return "ship-x", nil
		}
		return "", nil
	}

	resolver := opsrunner.NewBindingResolver(catalog, overrides, checker)
	resolver.InitiativeOfItem = overrides.InitiativeOfItem
	svc := NewService(catalog, resolver, repo, execStore).
		WithModes(defs, checker).
		WithOverrideAdmin(overrides, opsrunner.NewOverrideWriter(loc))
	return svc, loc
}

func initTarget(id string) *apipb.AgentOpsTargetSelector {
	return &apipb.AgentOpsTargetSelector{Kind: domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_INITIATIVE, Id: id}
}

func TestResolveBinding(t *testing.T) {
	svc, _ := buildService(t)
	resp, err := svc.ResolveBinding(context.Background(), connect.NewRequest(&apipb.AgentOpsResolveBindingRequest{
		Target: initTarget("ship-x"), Operation: "review-round",
	}))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resp.Msg.GetResolved().GetMode() != "synthetic-loop" || resp.Msg.GetResolved().GetModeRevision() != rev {
		t.Fatalf("resolved = %+v", resp.Msg.GetResolved())
	}
	if resp.Msg.GetPolicyId() != "initiative-default" {
		t.Fatalf("policy id = %q", resp.Msg.GetPolicyId())
	}
}

func TestResolveBindingNoBindingIsNotFound(t *testing.T) {
	svc, _ := buildService(t)
	_, err := svc.ResolveBinding(context.Background(), connect.NewRequest(&apipb.AgentOpsResolveBindingRequest{
		Target: initTarget("x"), Operation: "workshop-round", // no binding authored
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("absent binding must map to NotFound, got %v", err)
	}
}

func TestValidateInvocationIncompatible(t *testing.T) {
	svc, _ := buildService(t)
	resp, err := svc.ValidateInvocation(context.Background(), connect.NewRequest(&apipb.AgentOpsValidateInvocationRequest{
		Target:    &apipb.AgentOpsTargetSelector{Kind: domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_PLAN_EXECUTION, Id: "p"},
		Operation: "review-round",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetOperationDeclared() || resp.Msg.GetTargetCompatible() {
		t.Fatalf("plan-execution must be incompatible with review-round: %+v", resp.Msg)
	}
	if len(resp.Msg.GetMissingCapabilities()) == 0 {
		t.Fatalf("expected missing capabilities to be reported")
	}
}

func TestInspectWorkflowAndExecution(t *testing.T) {
	svc, loc := buildService(t)
	repo := opsrunner.NewWorkflowRepo(loc)
	execStore := opsrunner.NewExecutionStore(loc)
	kind, id := agentops.TargetInitiative, "ship-x"

	// Seed a running workflow with one operation record.
	w, _ := repo.CreateOrLoad(kind, id)
	prov := agentops.ExecutionProvenance{
		Kind: "agentops-execution-provenance", Operation: agentops.OpReviewRound, OperationVersion: "1.0.0",
		Binding: agentops.ProvenanceBinding{Layer: agentops.LayerSystemDefault, OwnerKind: "system", OwnerID: "system"},
		Mode:    "synthetic-loop", ModeRevision: rev,
		CompiledModeDigest: digestOf(t, `{"m":1}`), PromptCatalogRevision: "pc-1", PromptCatalogDigest: digestOf(t, `{"p":1}`),
		Target: agentops.ProvenanceTarget{Kind: kind, ID: id}, CallerInputDigest: digestOf(t, `{}`),
		PolicyRevision: "pol-1", WorkflowInstanceID: w.InstanceID,
	}
	next := w
	next.State = agentops.WorkflowRunning
	next.Operations = []agentops.OperationExecutionRecord{{
		Operation: agentops.OpReviewRound, ExecutionID: "exec-1", IdempotencyKey: "k1",
		ProvenanceDigest: mustDigestOf(t, prov), State: "completed", Outcome: "accepted",
	}}
	next.IdempotencyKeys = []string{"k1"}
	next.Version = 1
	if err := repo.Commit(0, next); err != nil {
		t.Fatal(err)
	}
	if err := execStore.SaveExecution(kind, id, "exec-1", opsrunner.ExecutionSnapshot{
		Provenance: prov, CompiledMode: json.RawMessage(`{"m":1}`), PromptCatalog: json.RawMessage(`{"p":1}`),
		EffectiveInputs: json.RawMessage(`{}`), Outcome: "accepted",
	}); err != nil {
		t.Fatal(err)
	}

	wresp, err := svc.InspectWorkflow(context.Background(), connect.NewRequest(&apipb.AgentOpsInspectWorkflowRequest{Target: initTarget(id)}))
	if err != nil {
		t.Fatal(err)
	}
	if !wresp.Msg.GetFound() || wresp.Msg.GetWorkflow().GetState() != domainpb.AgentOpsWorkflowState_AGENT_OPS_WORKFLOW_STATE_RUNNING {
		t.Fatalf("workflow inspect = %+v", wresp.Msg)
	}
	if len(wresp.Msg.GetWorkflow().GetOperations()) != 1 {
		t.Fatalf("expected 1 operation record")
	}

	eresp, err := svc.InspectExecution(context.Background(), connect.NewRequest(&apipb.AgentOpsInspectExecutionRequest{Target: initTarget(id), ExecutionId: "exec-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !eresp.Msg.GetFound() || !eresp.Msg.GetReproducible() {
		t.Fatalf("execution inspect must be found + reproducible: %+v", eresp.Msg)
	}
	if eresp.Msg.GetProvenance().GetMode() != "synthetic-loop" {
		t.Fatalf("provenance mode = %q", eresp.Msg.GetProvenance().GetMode())
	}
}

func digestOf(t *testing.T, s string) string {
	t.Helper()
	d, err := agentops.CanonicalDigest([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustDigestOf(t *testing.T, v any) string {
	t.Helper()
	d, err := agentops.DigestOf(v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d, "sha256:") {
		t.Fatalf("bad digest %q", d)
	}
	return d
}

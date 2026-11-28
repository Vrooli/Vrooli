package coresetpermissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type fakeState struct {
	doc   operatorstate.Document
	reads int
	err   error
}

func boolPtr(value bool) *bool { return &value }

func (s *fakeState) Get(context.Context) (operatorstate.Document, error) {
	s.reads++
	if s.err != nil {
		return operatorstate.Document{}, s.err
	}
	return s.doc, nil
}

// fakeCAS exercises provider mapping and retry behavior only. It is not
// evidence that production has the owner transaction.
type fakeCAS struct {
	state     *fakeState
	receipts  map[string]StateReceipt
	mutations map[string]StateMutation
	now       time.Time
	calls     int
}

func (w *fakeCAS) CompareAndSetCore(_ context.Context, m StateMutation) (StateReceipt, error) {
	w.calls++
	if prior, ok := w.receipts[m.RequestID]; ok {
		old := w.mutations[m.RequestID]
		if old.ActorID == m.ActorID && old.Target == m.Target && old.ExpectedRevision == m.ExpectedRevision && old.RequestDigest == m.RequestDigest && old.AuthorityDigest == m.AuthorityDigest {
			return prior, nil
		}
		return StateReceipt{}, ErrRequestIdentityConflict{RequestID: m.RequestID}
	}
	current := operatorstate.Revision(w.state.doc)
	if current != m.ExpectedRevision {
		return StateReceipt{}, ErrRevisionConflict{Expected: m.ExpectedRevision, Current: current}
	}
	if err := m.Authority.Validate(); err != nil {
		return StateReceipt{}, err
	}
	newRevision := fmt.Sprintf("r%d", len(w.receipts)+2)
	w.state.doc.Core = &operatorstate.CoreSet{Seed: append([]string(nil), m.Authority.Seed...), TrustedBase: append([]string(nil), m.Authority.TrustedBase...), Important: append([]coreset.ComponentID(nil), m.Authority.Important...), CoreMembers: append([]coreset.ComponentID(nil), m.Authority.CoreMembers...)}
	w.state.doc.UpdatedAt = newRevision
	r := StateReceipt{RequestID: m.RequestID, ActorID: m.ActorID, Target: m.Target, Subject: WriteSubject, ExpectedRevision: m.ExpectedRevision, NewRevision: newRevision, RequestDigest: m.RequestDigest, AuthorityDigest: m.AuthorityDigest, Outcome: "updated", CreatedAt: w.now}
	w.receipts[m.RequestID], w.mutations[m.RequestID] = r, m
	return r, nil
}

func fixture(writable bool) (*Provider, *fakeState, *fakeCAS, context.Context) {
	state := &fakeState{doc: operatorstate.Document{Version: "v1", UpdatedAt: "r1", Scenarios: map[string]operatorstate.ScenarioChoice{"custom-app": {Enabled: boolPtr(true)}, "different-app": {Enabled: boolPtr(true)}}, Core: &operatorstate.CoreSet{Seed: []string{"web-console", "custom-app"}, TrustedBase: []string{"web-console"}}}}
	var writer *fakeCAS
	if writable {
		writer = &fakeCAS{state: state, receipts: map[string]StateReceipt{}, mutations: map[string]StateMutation{}, now: time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC)}
	}
	p := NewProvider(state)
	if writer != nil {
		p = NewProvider(state, writer)
	}
	p.WithOptions(WithMutationAuthorizer(func(context.Context) error { return nil }), WithClock(func() time.Time { return time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC) }))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true})
	return p, state, writer, ctx
}

func TestListAndReadAreLocalAndCanonical(t *testing.T) {
	p, state, _, ctx := fixture(false)
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 {
		t.Fatalf("descriptors=%+v", ds)
	}
	d := ds[0]
	if d.Owner != Owner || d.PrincipalID != "operator-1" || d.SubjectKind != SubjectKind || d.SubjectID != "local" || d.TargetBound || d.Target != (permissions.TargetBinding{}) || d.CanWrite {
		t.Fatalf("descriptor=%+v", d)
	}
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	var authority coreset.Authority
	if err = json.Unmarshal(got.Value, &authority); err != nil {
		t.Fatal(err)
	}
	if got.Revision != "r1" || !contains(authority.Seed, "web-console") || !contains(authority.TrustedBase, "prompt-manager") || state.reads < 2 {
		t.Fatalf("read=%+v authority=%+v reads=%d", got, authority, state.reads)
	}
}

func TestActorSubjectAndRemoteTargetMismatchRefuse(t *testing.T) {
	p, _, writer, ctx := fixture(true)
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := ds[0]
	req := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "request-1", Value: []byte(`{"seed":["custom-app"],"trusted_base":[]}`)}
	remote := req
	remote.Target.MachineID = "machine-1"
	remote.Target.NodeID = "node-1"
	remote.Target.EnrollmentGeneration = 1
	if _, err = p.Authorize(ctx, d, remote); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("remote target=%v", err)
	}
	bad := d
	bad.SubjectID = "other"
	if _, err = p.Authorize(ctx, bad, req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("subject=%v", err)
	}
	wrongActor := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "other", Verified: true})
	if _, err = p.Authorize(wrongActor, d, req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("actor=%v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("invalid bindings reached owner: calls=%d", writer.calls)
	}
}

func TestInvalidReadPermitAndOwnerFailure(t *testing.T) {
	p, state, _, ctx := fixture(false)
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	req := permissions.ReadRequest{ActorID: "operator-1", Owner: Owner}
	reads := state.reads
	if _, err = p.Read(ctx, ds[0], permissions.Permit{}, req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("permit=%v", err)
	}
	if state.reads != reads {
		t.Fatal("invalid permit reached owner")
	}
	state.err = errors.New("owner API unavailable")
	if _, err = p.List(ctx); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("owner error=%v", err)
	}
}

func TestWritesUnavailableWithoutProductionReceiptAdapter(t *testing.T) {
	p, _, _, ctx := fixture(false)
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	req := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "request-1", Value: []byte(`{"seed":[],"trusted_base":[]}`)}
	if _, err = p.Authorize(ctx, ds[0], req); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("authorize=%v", err)
	}
	if _, err = p.Write(ctx, ds[0], permissions.Permit{}, req); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("write=%v", err)
	}
}

func TestOwnerCASExactReplayChangedRequestAndStaleRevision(t *testing.T) {
	p, _, writer, ctx := fixture(true)
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := ds[0]
	req := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "request-1", Value: []byte(`{"seed":["custom-app"],"trusted_base":[]}`)}
	first, err := r.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := r.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first != replay {
		t.Fatalf("replay=%+v first=%+v", replay, first)
	}
	changed := req
	changed.Value = []byte(`{"seed":["different-app"],"trusted_base":[]}`)
	if _, err = r.Write(ctx, d.ID, changed); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed payload=%v", err)
	}
	actorB := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-b", Verified: true})
	changedActor := req
	changedActor.ActorID = "operator-b"
	if _, err = r.Write(actorB, d.ID, changedActor); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed actor=%v", err)
	}
	stale := req
	stale.IdempotencyKey = "stale-request"
	stale.ExpectedRevision = "stale"
	if _, err = r.Write(ctx, d.ID, stale); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale revision=%v", err)
	}
	if writer.calls != 5 {
		t.Fatalf("CAS owner calls=%d", writer.calls)
	}
	if strings.Contains(fmt.Sprintf("%+v", first), "custom-app") {
		t.Fatal("audit receipt leaked authority values")
	}
}

func TestTierWriteRequiresSelectedEntriesAndPersistsBothTiers(t *testing.T) {
	p, state, writer, ctx := fixture(true)
	p.WithInventory(func(kind, name string) error {
		if (kind == "scenario" && name == "monitor-app") || (kind == "resource" && name == "redis") {
			return nil
		}
		return fmt.Errorf("unknown %s %q", kind, name)
	})
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	state.doc.Scenarios = map[string]operatorstate.ScenarioChoice{"monitor-app": {Enabled: boolPtr(false)}}
	state.doc.Resources = map[string]operatorstate.EnabledChoice{"redis": {Enabled: boolPtr(true)}}
	invalid := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "unselected-tier", Value: []byte(`{"seed":["web-console","custom-app"],"trusted_base":["web-console"],"important":[{"kind":"scenario","name":"monitor-app"}]}`)}
	if _, err := registry.Write(ctx, ds[0].ID, invalid); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unselected important write error=%v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("unselected tier reached persistence: calls=%d", writer.calls)
	}

	state.doc.Scenarios["monitor-app"] = operatorstate.ScenarioChoice{Enabled: boolPtr(true)}
	unknown := invalid
	unknown.IdempotencyKey = "unknown-tier"
	unknown.Value = []byte(`{"seed":["web-console","custom-app"],"trusted_base":["web-console"],"important":[{"kind":"scenario","name":"ghost"}]}`)
	state.doc.Scenarios["ghost"] = operatorstate.ScenarioChoice{Enabled: boolPtr(true)}
	if _, err := registry.Write(ctx, ds[0].ID, unknown); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown selected tier error=%v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("unknown tier reached persistence: calls=%d", writer.calls)
	}
	valid := invalid
	valid.IdempotencyKey = "selected-tiers"
	valid.Value = []byte(`{"seed":["web-console","custom-app"],"trusted_base":["web-console"],"important":[{"kind":"scenario","name":"monitor-app"}],"members":[{"kind":"resource","name":"redis"}]}`)
	if _, err := registry.Write(ctx, ds[0].ID, valid); err != nil {
		t.Fatalf("selected tier write: %v", err)
	}
	if state.doc.Core == nil || len(state.doc.Core.Important) != 1 || len(state.doc.Core.CoreMembers) != 1 {
		t.Fatalf("durable tier state=%+v", state.doc.Core)
	}
}

func TestOwnerAdapterUsesAtomicOperatorStateReceiptAndSurvivesRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	service := ownerServiceForTest(t, statePath)
	p := NewOwnerProvider(service)
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:write"}})
	ds, err := registry.Inventory(ctx)
	if err != nil || len(ds) != 1 || !ds[0].CanWrite {
		t.Fatalf("inventory=%+v err=%v", ds, err)
	}
	initial, err := registry.Read(ctx, PermissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	value := []byte(`{"seed":["web-console"],"trusted_base":["web-console"]}`)
	req := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: initial.Revision, IdempotencyKey: "core-set-owner-request", Value: value}
	first, err := registry.Write(ctx, PermissionID, req)
	if err != nil || first.ResultRevision == initial.Revision || first.RequestDigest == "" || first.ActorID != "operator-a" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	restarted := NewOwnerProvider(ownerServiceForTest(t, statePath))
	restartedRegistry, err := permissions.NewRegistry(restarted)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := restartedRegistry.Write(ctx, PermissionID, req)
	if err != nil || replayed != first {
		t.Fatalf("replay=%+v err=%v first=%+v", replayed, err, first)
	}
	changed := req
	changed.Value = []byte(`{"seed":["web-console"],"trusted_base":[]}`)
	if _, err = restartedRegistry.Write(ctx, PermissionID, changed); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed request=%v", err)
	}
	stale := req
	stale.IdempotencyKey = "stale-core-request"
	stale.ExpectedRevision = "stale"
	if _, err = restartedRegistry.Write(ctx, PermissionID, stale); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale revision=%v", err)
	}
	current, err := restartedRegistry.Read(ctx, PermissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(current.Value), "permission_receipts") || strings.Contains(fmt.Sprintf("%+v", first), "custom-app") {
		t.Fatal("state receipt or authority value leaked into receipt/read metadata")
	}
}

func TestOwnerAdapterCanClearTierWithExplicitEmptyArray(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	service := ownerServiceForTest(t, statePath)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:write"}})
	initial, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Patch(ctx, []byte(`{"scenarios":{"monitor-app":{"enabled":true}}}`), []string{"scenarios.monitor-app.enabled"}, operatorstate.Revision(initial)); err != nil {
		t.Fatalf("select monitor scenario: %v", err)
	}
	p := NewOwnerProvider(service).WithInventory(func(kind, name string) error {
		if kind == "scenario" && name == "monitor-app" {
			return nil
		}
		return fmt.Errorf("unknown %s %q", kind, name)
	})
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	read, err := registry.Read(ctx, PermissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	set := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: read.Revision, IdempotencyKey: "set-important", Value: []byte(`{"seed":["agent-manager","prompt-manager","web-console","vrooli-autoheal"],"trusted_base":["agent-manager","prompt-manager","web-console","vrooli-autoheal"],"important":[{"kind":"scenario","name":"monitor-app"}],"members":[]}`)}
	setReceipt, err := registry.Write(ctx, PermissionID, set)
	if err != nil {
		t.Fatalf("set important tier: %v", err)
	}
	clear := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: setReceipt.ResultRevision, IdempotencyKey: "clear-important", Value: []byte(`{"seed":["agent-manager","prompt-manager","web-console","vrooli-autoheal"],"trusted_base":["agent-manager","prompt-manager","web-console","vrooli-autoheal"],"important":[],"members":[]}`)}
	if _, err := registry.Write(ctx, PermissionID, clear); err != nil {
		t.Fatalf("clear important tier: %v", err)
	}
	current, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.Core == nil || len(current.Core.Important) != 0 || len(current.Core.CoreMembers) != 0 {
		t.Fatalf("explicit tier clear was not persisted: %+v", current.Core)
	}
}

func ownerServiceForTest(t *testing.T, statePath string) operatorstateapi.Service {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	store := operatorstate.New(operatorstate.Config{RepoRoot: t.TempDir(), SchemaPath: filepath.Join(repoRoot, operatorstate.SchemaPath), StatePath: func(context.Context) (string, error) { return statePath, nil }})
	return operatorstateapi.New(store, nil)
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

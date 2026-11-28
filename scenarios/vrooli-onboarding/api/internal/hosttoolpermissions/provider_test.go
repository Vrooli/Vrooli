package hosttoolpermissions

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"errors"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"os"
	"path/filepath"
	"time"
)

func fixture(t *testing.T) (*Provider, context.Context, *string) {
	t.Helper()
	rev := "manifest-rev-1"
	reader := &testStateReader{doc: operatorstate.Document{UpdatedAt: "state-r1"}}
	p := New(func(context.Context) (string, []Tool, error) {
		return rev, []Tool{{Name: "node", Scenario: "web-console", Required: true, Reason: "build assets", Privilege: "user", SchemaDigest: "sha256:abc"}}, nil
	}, reader, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "op-1", Verified: true})
	return p, ctx, &rev
}

type testStateReader struct{ doc operatorstate.Document }

func (r *testStateReader) Get(context.Context) (operatorstate.Document, error) { return r.doc, nil }

type testStateWriter struct {
	reader    *testStateReader
	receipts  map[string]StateReceipt
	mutations map[string]StateMutation
	failure   error
}

func (w *testStateWriter) CompareAndSetHostTool(_ context.Context, m StateMutation) (StateReceipt, error) {
	if w.failure != nil {
		return StateReceipt{}, w.failure
	}
	if prior, ok := w.receipts[m.RequestID]; ok {
		if w.mutations[m.RequestID] != m {
			return StateReceipt{}, errors.New("request identity conflict")
		}
		return prior, nil
	}
	if operatorstate.Revision(w.reader.doc) != m.ExpectedRevision {
		return StateReceipt{}, operatorstate.ErrRevisionConflict
	}
	r := StateReceipt{RequestID: m.RequestID, ActorID: m.ActorID, Target: LocalTarget, Subject: SubjectKind + ":" + m.Scenario + "/" + m.Name, ExpectedRevision: m.ExpectedRevision, NewRevision: "state-r2", RequestDigest: m.RequestDigest, Outcome: "updated", At: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)}
	w.receipts[m.RequestID] = r
	w.mutations[m.RequestID] = m
	w.reader.doc.UpdatedAt = r.NewRevision
	w.reader.doc.HostTools = map[string]operatorstate.OptInChoice{m.Name: {OptedIn: &m.OptedIn}}
	return r, nil
}

func TestReadOnlyInventoryThroughRegistryAndRejectWrite(t *testing.T) {
	p, ctx, _ := fixture(t)
	r, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := r.Inventory(ctx)
	if err != nil || len(ds) != 1 {
		t.Fatalf("inventory=%v err=%v", ds, err)
	}
	d := ds[0]
	if d.CanWrite || d.Scope != PermissionScope || d.PrincipalID != "op-1" || d.Source != Source {
		t.Fatalf("descriptor not fully bound/read-only: %+v", d)
	}
	got, err := r.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "op-1", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	var tool Tool
	if err := json.Unmarshal(got.Value, &tool); err != nil || tool.Name != "node" || got.Revision != "state-r1" {
		t.Fatalf("read=%+v tool=%+v err=%v", got, tool, err)
	}
	_, err = r.Write(ctx, d.ID, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: got.Revision, IdempotencyKey: "install-1", Value: []byte(`{"opted_in":true}`)})
	if err == nil {
		t.Fatal("write should be rejected")
	}
}

func TestStaleDescriptorAndWrongTargetRefused(t *testing.T) {
	p, ctx, rev := fixture(t)
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := ds[0]
	*rev = "manifest-rev-2"
	_, err = p.AuthorizeRead(ctx, d, permissions.ReadRequest{ActorID: "op-1", Owner: Owner})
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale descriptor accepted: %v", err)
	}
	*rev = "manifest-rev-1"
	_, err = p.AuthorizeRead(ctx, d, permissions.ReadRequest{ActorID: "op-1", Owner: Owner, Target: permissions.TargetBinding{MachineID: "m", NodeID: "n", EnrollmentGeneration: 1}})
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("target-bound request accepted: %v", err)
	}
}

func TestInvalidInventoryAndUnverifiedActorFailClosed(t *testing.T) {
	p := New(func(context.Context) (string, []Tool, error) { return "", nil, nil }, &testStateReader{doc: operatorstate.Document{UpdatedAt: "r1"}}, nil)
	_, err := p.List(context.Background())
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unverified actor accepted: %v", err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "op", Verified: true})
	_, err = p.List(ctx)
	if connect.CodeOf(err) != connect.CodeDataLoss {
		t.Fatalf("empty revision accepted: %v", err)
	}
}

func TestOptInWriteUsesCASReceiptAndExactReplay(t *testing.T) {
	p, ctx, _ := fixture(t)
	reader := p.reader.(*testStateReader)
	writer := &testStateWriter{reader: reader, receipts: map[string]StateReceipt{}, mutations: map[string]StateMutation{}}
	p.writer = writer
	p.authorizeMutation = func(context.Context) error { return nil }
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := registry.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := ds[0]
	if !d.CanWrite {
		t.Fatal("opt-in provider should expose write when owner CAS is available")
	}
	request := permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "state-r1", IdempotencyKey: "stable-request-1", Value: []byte(`{"opted_in":true}`)}
	first, err := registry.Write(ctx, d.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.Write(ctx, d.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.ResultRevision != second.ResultRevision || writerCalls(writer) != 1 {
		t.Fatalf("replay mismatch first=%+v second=%+v calls=%d", first, second, writerCalls(writer))
	}
	read, err := registry.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "op-1", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	var projected Tool
	if err = json.Unmarshal(read.Value, &projected); err != nil || !projected.OptedIn {
		t.Fatalf("opt-in state not projected: %+v err=%v", projected, err)
	}
}

func writerCalls(w *testStateWriter) int { return len(w.receipts) }

func TestOptInRejectsUnsafePayloadStaleRevisionAndOwnerFailure(t *testing.T) {
	p, ctx, _ := fixture(t)
	reader := p.reader.(*testStateReader)
	writer := &testStateWriter{reader: reader, receipts: map[string]StateReceipt{}, mutations: map[string]StateMutation{}}
	p.writer = writer
	p.authorizeMutation = func(context.Context) error { return nil }
	d := mustDescriptor(t, p, ctx)
	for _, payload := range []string{`{"install":true}`, `{"opted_in":"yes"}`, `{"opted_in":true,"config":{}}`} {
		_, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "state-r1", IdempotencyKey: "req", Value: []byte(payload)})
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("unsafe payload %s got %v", payload, err)
		}
	}
	_, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "old-rev", IdempotencyKey: "req", Value: []byte(`{"opted_in":true}`)})
	if err != nil {
		t.Fatalf("authorization should defer owner CAS revision check: %v", err)
	}
	_, err = p.Write(ctx, d, permissions.Permit{}, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "old-rev", IdempotencyKey: "req", Value: []byte(`{"opted_in":true}`)})
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("invalid permit accepted: %v", err)
	}
	writer.failure = errors.New("owner store unavailable")
	decision, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "state-r1", IdempotencyKey: "req-2", Value: []byte(`{"opted_in":true}`)})
	if err != nil || !decision.Allowed {
		t.Fatalf("authorize=%+v err=%v", decision, err)
	}
	// The shared registry mints the opaque permit and the failed owner call must
	// surface without a fabricated receipt.
	registry, _ := permissions.NewRegistry(p)
	_, err = registry.Write(ctx, d.ID, permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: "state-r1", IdempotencyKey: "req-2", Value: []byte(`{"opted_in":true}`)})
	if err == nil {
		t.Fatal("owner failure was hidden")
	}
}

func mustDescriptor(t *testing.T, p *Provider, ctx context.Context) permissions.Descriptor {
	t.Helper()
	ds, err := p.List(ctx)
	if err != nil || len(ds) != 1 {
		t.Fatalf("list=%v err=%v", ds, err)
	}
	return ds[0]
}

func TestOwnerAdapterPatchWithReceiptAndStaleRevision(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	store := operatorstate.New(operatorstate.Config{RepoRoot: filepath.Dir(statePath), StatePath: func(context.Context) (string, error) { return statePath, nil }})
	service := operatorstateapi.New(store, nil)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "op-1", Verified: true, Scopes: []string{PermissionScope}})
	p := New(func(context.Context) (string, []Tool, error) {
		return "manifest-rev", []Tool{{Name: "node", Scenario: "web-console", Reason: "build", Privilege: "user", SchemaDigest: "sha256:abc"}}, nil
	}, service, NewOwnerWriter(service))
	p.authorizeMutation = func(context.Context) error { return nil }
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := mustDescriptor(t, p, ctx)
	req := permissions.WriteRequest{ActorID: "op-1", Owner: Owner, ExpectedRevision: operatorstate.Revision(initial), IdempotencyKey: "owner-adapter-replay", Value: []byte(`{"opted_in":true}`)}
	first, err := registry.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.ResultRevision != second.ResultRevision {
		t.Fatalf("owner receipt replay changed: first=%+v second=%+v", first, second)
	}
	doc, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	choice := doc.HostTools["node"]
	if choice.OptedIn == nil || !*choice.OptedIn {
		t.Fatalf("owner state missing opt-in: %+v", doc.HostTools)
	}
	if len(doc.PermissionReceipts) != 1 {
		t.Fatalf("receipt not durable in owner state: %+v", doc.PermissionReceipts)
	}
	if doc.Completion != nil {
		t.Fatalf("opt-in CAS was incorrectly treated as host configuration completion: %+v", doc.Completion)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(statePath), "apply-runs")); !os.IsNotExist(err) {
		t.Fatalf("opt-in CAS created or mutated an apply run; err=%v", err)
	}
	stale := req
	stale.IdempotencyKey = "stale-request"
	stale.ExpectedRevision = operatorstate.Revision(initial)
	_, err = registry.Write(ctx, d.ID, stale)
	if err == nil {
		t.Fatal("stale revision write unexpectedly succeeded")
	}
	if got := len(doc.PermissionReceipts); got != 1 {
		t.Fatalf("stale mutation added receipt: %d", got)
	}
}

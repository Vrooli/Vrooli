package safeguardpermissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type testReader struct {
	doc   operatorstate.Document
	calls int
}

func (r *testReader) Get(context.Context) (operatorstate.Document, error) {
	r.calls++
	return r.doc, nil
}

type testWriter struct {
	reader   *testReader
	requests map[string]Mutation
	receipts map[string]Receipt
	calls    int
}

func (w *testWriter) CompareAndSetSafeguard(_ context.Context, m Mutation) (Receipt, error) {
	w.calls++
	if prior, ok := w.receipts[m.RequestID]; ok {
		old := w.requests[m.RequestID]
		if old.ActorID == m.ActorID && old.Name == m.Name && old.RequestDigest == m.RequestDigest && old.ExpectedRevision == m.ExpectedRevision {
			return prior, nil
		}
		return Receipt{}, fmt.Errorf("request id conflict")
	}
	if operatorstate.Revision(w.reader.doc) != m.ExpectedRevision {
		return Receipt{}, operatorstate.ErrRevisionConflict
	}
	r := Receipt{RequestID: m.RequestID, ActorID: m.ActorID, Target: LocalTarget, Subject: SubjectKind + ":" + m.Name, ExpectedRevision: m.ExpectedRevision, NewRevision: "r2", RequestDigest: m.RequestDigest, Outcome: "updated", At: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)}
	w.receipts[m.RequestID] = r
	w.requests[m.RequestID] = m
	w.reader.doc.UpdatedAt = "r2"
	return r, nil
}
func testFixture(t *testing.T) (*Provider, *testReader, *testWriter, context.Context) {
	t.Helper()
	secret := "private-secret-material"
	r := &testReader{doc: operatorstate.Document{Version: "v1", UpdatedAt: "r1", HostSafeguards: map[string]operatorstate.OptInChoice{"firewall": {Config: map[string]any{"token": secret}}}}}
	w := &testWriter{reader: r, requests: map[string]Mutation{}, receipts: map[string]Receipt{}}
	p := New(r, w, func(context.Context) ([]Safeguard, error) {
		return []Safeguard{{Name: "firewall", SchemaDigest: "schema-a", ConfigSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"mode": map[string]any{"type": "string", "enum": []any{"guard", "observe"}}, "token": map[string]any{"type": "string"}}}, ConfigFields: []string{"mode", "token"}, SecretFields: []string{"token"}}}, nil
	})
	p.authorizeMutation = func(context.Context) error { return nil }
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	return p, r, w, ctx
}
func TestListReadWriteAndDurableReplayContract(t *testing.T) {
	p, r, w, ctx := testFixture(t)
	ds, err := p.List(ctx)
	if err != nil || len(ds) != 1 {
		t.Fatalf("list=%v err=%v", ds, err)
	}
	d := ds[0]
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	read, err := registry.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err = json.Unmarshal(read.Value, &value); err != nil {
		t.Fatal(err)
	}
	if value["opted_in"] != false {
		t.Fatalf("unexpected value %v", value)
	}
	if strings.Contains(string(read.Value), "private-secret-material") || value["config"].(map[string]any)["token"] != nil {
		t.Fatalf("secret field was not redacted: %s", read.Value)
	}
	payload := []byte(`{"opted_in":true}`)
	req := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "request-1", Value: payload}
	first, err := registry.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate provider/service recreation while retaining owner receipts.
	p2 := New(r, w, p.inventory)
	p2.authorizeMutation = func(context.Context) error { return nil }
	registry2, err := permissions.NewRegistry(p2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry2.Write(ctx, d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || w.calls != 2 {
		t.Fatalf("replay=%+v/%+v calls=%d", first, second, w.calls)
	}
	changed := req
	changed.Value = []byte(`{"opted_in":false}`)
	if _, err = registry2.Write(ctx, d.ID, changed); err == nil {
		t.Fatal("changed payload replay was accepted")
	}
}
func TestRefusesRemoteActorStaleAndUnsupportedBeforeWrite(t *testing.T) {
	p, _, w, ctx := testFixture(t)
	ds, _ := p.List(ctx)
	d := ds[0]
	if _, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, Target: permissions.TargetBinding{NodeID: "remote", MachineID: "machine", EnrollmentGeneration: 1}, ExpectedRevision: "r1", IdempotencyKey: "x", Value: []byte(`{"opted_in":true}`)}); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("remote target error=%v", err)
	}
	stale := d
	stale.SchemaVersion = SchemaVersion + "/old"
	if _, err := p.Authorize(ctx, stale, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "x", Value: []byte(`{"opted_in":true}`)}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale error=%v", err)
	}
	if _, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "other", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "x", Value: []byte(`{"opted_in":true}`)}); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("actor mismatch error=%v", err)
	}
	p.authorizeMutation = func(context.Context) error {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("operator policy denied"))
	}
	if _, err := p.Authorize(ctx, d, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "x", Value: []byte(`{"opted_in":true}`)}); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("policy denial error=%v", err)
	}
	if w.calls != 0 {
		t.Fatalf("refusal reached writer %d times", w.calls)
	}
}
func TestReceiptsAndErrorsDoNotContainConfigValues(t *testing.T) {
	p, _, writer, ctx := testFixture(t)
	ds, _ := p.List(ctx)
	d := ds[0]
	registry, _ := permissions.NewRegistry(p)
	for _, raw := range []string{`{"config":{"token":"super-secret-value"}}`, `{"config":{"unknown":"value"}}`, `{"config_key":"unknown","value":"value"}`, `{"config":{"mode":"guard"},"unexpected":true}`, `{"config":{"mode":"guard"},"config_key":"mode","value":"guard"}`, `{"config":{"mode":true}}`, `{"config":{"mode":"unsupported"}}`} {
		if _, err := registry.Write(ctx, d.ID, permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: "r1", IdempotencyKey: "secret-free", Value: []byte(raw)}); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("unsupported config write %s error=%v", raw, err)
		}
	}
	if writer.calls != 0 {
		t.Fatalf("unsupported or secret configuration reached owner writer %d times", writer.calls)
	}
}

func TestMutationDigestBindsManifestSchemaRevision(t *testing.T) {
	value := []byte(`{"config":{"target":"collector.invalid"}}`)
	one := boundMutationDigest(SchemaVersion+"/schema-a", value)
	two := boundMutationDigest(SchemaVersion+"/schema-b", value)
	if one == two {
		t.Fatal("same config payload under changed manifest schema shared a durable request digest")
	}
}

func TestOwnerAdapterPersistsReplayReceiptAcrossServiceRecreation(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	service := ownerServiceForTest(t, statePath)
	if _, err := service.Get(ctx); err != nil {
		t.Fatal(err)
	}
	inventory := func(context.Context) ([]Safeguard, error) {
		return []Safeguard{{Name: "firewall", SchemaDigest: "schema-a", ConfigSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"mode": map[string]any{"type": "string", "enum": []any{"guard", "observe"}}, "token": map[string]any{"type": "string"}}}, ConfigFields: []string{"mode", "token"}, SecretFields: []string{"token"}}}, nil
	}
	provider := New(service, NewOwnerWriter(service), inventory)
	provider.authorizeMutation = func(context.Context) error { return nil }
	registry, err := permissions.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	read, err := registry.Read(ctx, PermissionID("firewall"), permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"config":{"mode":"guard"}}`)
	request := permissions.WriteRequest{ActorID: "operator-a", Owner: Owner, ExpectedRevision: read.Revision, IdempotencyKey: "durable-safeguard-1", Value: payload}
	first, err := registry.Write(ctx, PermissionID("firewall"), request)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%+v", first), `"mode":"guard"`) {
		t.Fatal("receipt contains a safeguard value")
	}
	secondService := ownerServiceForTest(t, statePath)
	second := New(secondService, NewOwnerWriter(secondService), inventory)
	second.authorizeMutation = func(context.Context) error { return nil }
	restarted, err := permissions.NewRegistry(second)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := restarted.Write(ctx, PermissionID("firewall"), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != first.ID || replayed.ResultRevision != first.ResultRevision {
		t.Fatalf("replay changed receipt: first=%+v replayed=%+v", first, replayed)
	}
	changed := request
	changed.Value = []byte(`{"opted_in":false}`)
	if _, err := restarted.Write(ctx, PermissionID("firewall"), changed); !errors.Is(err, operatorstate.ErrMutationReceiptConflict) {
		t.Fatalf("changed replay error=%v", err)
	}
	current, err := restarted.Read(ctx, PermissionID("firewall"), permissions.ReadRequest{ActorID: "operator-a", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(current.Value), "top-secret") {
		t.Fatalf("provider read exposed schema secret: %s", current.Value)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("owner state file was not written: %v", err)
	}
	state, err := secondService.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.Completion != nil {
		t.Fatalf("configuration CAS was incorrectly treated as host apply completion: %+v", state.Completion)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(statePath), "apply-runs")); !os.IsNotExist(err) {
		t.Fatalf("configuration CAS created or mutated an apply run; err=%v", err)
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

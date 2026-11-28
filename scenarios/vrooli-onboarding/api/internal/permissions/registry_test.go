package permissions

import (
	"context"
	"errors"
	"testing"
	"time"
)

type unavailableProvider struct{ owner string }

func (p unavailableProvider) ProviderID() string { return "unavailable." + p.owner }
func (p unavailableProvider) Owner() string      { return p.owner }
func (unavailableProvider) List(context.Context) ([]Descriptor, error) {
	return nil, errors.New("owner temporarily unavailable")
}
func (unavailableProvider) AuthorizeRead(context.Context, Descriptor, ReadRequest) (Decision, error) {
	return Decision{}, errors.New("unavailable")
}
func (unavailableProvider) Read(context.Context, Descriptor, Permit, ReadRequest) (ReadResult, error) {
	return ReadResult{}, errors.New("unavailable")
}
func (unavailableProvider) Authorize(context.Context, Descriptor, WriteRequest) (Decision, error) {
	return Decision{}, errors.New("unavailable")
}
func (unavailableProvider) Write(context.Context, Descriptor, Permit, WriteRequest) (Receipt, error) {
	return Receipt{}, errors.New("unavailable")
}

type provider struct {
	descriptor      Descriptor
	reads           int
	readAuth        int
	authorizes      int
	writes          int
	readDecision    Decision
	decision        Decision
	receipt         Receipt
	readValue       []byte
	permitValid     bool
	readDecisionSet bool
	decisionSet     bool
}

func (p *provider) ProviderID() string { return "bridge.registry" }
func (p *provider) Owner() string      { return p.descriptor.Owner }
func (p *provider) List(context.Context) ([]Descriptor, error) {
	return []Descriptor{p.descriptor}, nil
}
func (p *provider) AuthorizeRead(_ context.Context, _ Descriptor, _ ReadRequest) (Decision, error) {
	p.readAuth++
	if p.readDecisionSet {
		return p.readDecision, nil
	}
	return Decision{Allowed: true}, nil
}
func (p *provider) Read(_ context.Context, _ Descriptor, permit Permit, req ReadRequest) (ReadResult, error) {
	p.reads++
	p.permitValid = permit.ValidForRead(p.descriptor.ID, req)
	return ReadResult{PermissionID: p.descriptor.ID, Owner: p.descriptor.Owner, Target: req.Target, Revision: "r4", Value: p.readValue, ObservedAt: time.Unix(1, 0)}, nil
}
func (p *provider) Authorize(_ context.Context, _ Descriptor, _ WriteRequest) (Decision, error) {
	p.authorizes++
	if p.decisionSet {
		return p.decision, nil
	}
	return Decision{Allowed: true}, nil
}
func (p *provider) Write(_ context.Context, _ Descriptor, permit Permit, req WriteRequest) (Receipt, error) {
	p.permitValid = permit.ValidForWrite(p.descriptor.ID, req)
	if !p.permitValid {
		return Receipt{}, context.Canceled
	}
	p.writes++
	if p.receipt.ID != "" {
		return p.receipt, nil
	}
	return Receipt{ID: "audit-1", PermissionID: p.descriptor.ID, Owner: p.descriptor.Owner, ActorID: req.ActorID, Target: req.Target, ExpectedRevision: req.ExpectedRevision, ResultRevision: "r5", IdempotencyKey: req.IdempotencyKey, RequestDigest: digestValue(req.Value), DecisionRef: permit.DecisionRef(), Outcome: "updated", At: time.Unix(2, 0)}, nil
}

func fixtureDescriptor() Descriptor {
	return Descriptor{
		Version: ContractVersion, ID: "bridge/node-scopes", Owner: "vrooli-bridge",
		PrincipalKind: "human-operator", PrincipalID: "operator-1", Scope: "vrooli-bridge:write", SubjectKind: "node",
		SubjectID: "node-7", TargetBound: true, Target: TargetBinding{MachineID: "machine-7", NodeID: "node-7", EnrollmentGeneration: 4},
		Source: "Bridge NodeRegistryService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator",
	}
}

func TestRegistryRoutesReadAndWriteThroughOwnerAndReturnsBoundReceipt(t *testing.T) {
	p := &provider{descriptor: fixtureDescriptor()}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := r.Inventory(context.Background()); err != nil || len(got) != 1 || got[0].ID != p.descriptor.ID {
		t.Fatalf("inventory = %+v, err=%v", got, err)
	}
	target := p.descriptor.Target
	read, err := r.Read(context.Background(), p.descriptor.ID, ReadRequest{ActorID: "operator-1", Owner: p.descriptor.Owner, Target: target})
	if err != nil || read.Revision != "r4" || p.reads != 1 || p.readAuth != 1 || !p.permitValid {
		t.Fatalf("read = %+v, err=%v calls=%d auth=%d", read, err, p.reads, p.readAuth)
	}
	receipt, err := r.Write(context.Background(), p.descriptor.ID, WriteRequest{ActorID: "operator-1", Owner: p.descriptor.Owner, Target: target, ExpectedRevision: "r4", IdempotencyKey: "request-1"})
	if err != nil || receipt.ResultRevision != "r5" || p.authorizes != 1 || p.writes != 1 || !p.permitValid {
		t.Fatalf("receipt = %+v, err=%v authorize=%d write=%d", receipt, err, p.authorizes, p.writes)
	}
}

func TestOwnerFilteredInventorySkipsUnrelatedUnavailableOwners(t *testing.T) {
	p := &provider{descriptor: fixtureDescriptor()}
	r, err := NewRegistry(p, unavailableProvider{owner: "notification-hub"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Inventory(context.Background()); err == nil {
		t.Fatal("aggregate inventory must fail closed when one owner is unavailable")
	}
	got, err := r.InventoryForOwner(context.Background(), "vrooli-bridge")
	if err != nil || len(got) != 1 || got[0].Owner != "vrooli-bridge" {
		t.Fatalf("filtered inventory = %+v, err=%v", got, err)
	}
	if _, err := r.InventoryForOwner(context.Background(), "unknown-owner"); err == nil {
		t.Fatal("unknown owner filter must be rejected")
	}
}

func TestProviderFilteredInventoryDoesNotProbeSiblingProviderForSameOwner(t *testing.T) {
	p := &provider{descriptor: fixtureDescriptor()}
	r, err := NewRegistry(p, unavailableProvider{owner: "vrooli-bridge"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.InventoryForProvider(context.Background(), p.ProviderID())
	if err != nil || len(got) != 1 || got[0].ID != p.descriptor.ID {
		t.Fatalf("provider-filtered inventory = %+v, err=%v", got, err)
	}
	if _, err := r.InventoryForProvider(context.Background(), "missing.provider"); err == nil {
		t.Fatal("unknown provider filter must fail closed")
	}
}

func TestDescriptorRequiresExplicitTargetSelectorWhenTargetBound(t *testing.T) {
	d := fixtureDescriptor()
	d.Target = TargetBinding{}
	if err := d.Validate(); err == nil {
		t.Fatal("target-bound descriptor without a target selector must be rejected")
	}
}

func TestRegistryRejectsUnknownCrossOwnerAndStaleTargetBeforeProvider(t *testing.T) {
	p := &provider{descriptor: fixtureDescriptor()}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	good := ReadRequest{ActorID: "operator-1", Owner: p.descriptor.Owner, Target: p.descriptor.Target}
	if _, err := r.Read(context.Background(), "unknown", good); err == nil {
		t.Fatal("unknown permission must fail closed")
	}
	crossOwner := good
	crossOwner.Owner = "other-owner"
	if _, err := r.Read(context.Background(), p.descriptor.ID, crossOwner); err == nil {
		t.Fatal("cross-owner read must fail closed")
	}
	stale := good
	stale.Target.EnrollmentGeneration++
	if _, err := r.Read(context.Background(), p.descriptor.ID, stale); err == nil {
		t.Fatal("stale enrollment binding must fail closed")
	}
	if p.reads != 0 || p.readAuth != 0 || p.authorizes != 0 || p.writes != 0 {
		t.Fatalf("refusals reached provider: %+v", p)
	}
}

func TestAvailableOwnerPermissionStillWorksWhenUnrelatedProviderIsUnavailable(t *testing.T) {
	p := &provider{descriptor: fixtureDescriptor()}
	r, err := NewRegistry(p, unavailableProvider{owner: "vrooli-bridge"})
	if err != nil {
		t.Fatal(err)
	}
	request := ReadRequest{ActorID: "operator-1", Owner: p.descriptor.Owner, Target: p.descriptor.Target}
	if _, err := r.Read(context.Background(), p.descriptor.ID, request); err != nil {
		t.Fatalf("unrelated unavailable owner blocked read: %v", err)
	}
	if _, err := r.Read(context.Background(), "unknown", request); err == nil {
		t.Fatal("unknown permission must report unavailable provider inventory rather than claim unknown")
	}
}

func TestRegistryRejectsMismatchedOwnerAuthorizationOrReceipt(t *testing.T) {
	d := fixtureDescriptor()
	p := &provider{descriptor: d, decision: Decision{Allowed: false}, decisionSet: true}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	req := WriteRequest{ActorID: "operator-1", Owner: d.Owner, Target: d.Target, ExpectedRevision: "r4", IdempotencyKey: "request-1"}
	if _, err := r.Write(context.Background(), d.ID, req); err == nil || p.writes != 0 {
		t.Fatalf("owner authorization mismatch should refuse before write, err=%v writes=%d", err, p.writes)
	}
	p.decision = Decision{Allowed: true, DecisionRef: "decision-1"}
	p.receipt = Receipt{ID: "audit-2", PermissionID: d.ID, Owner: d.Owner, ActorID: "operator-1", Target: d.Target, ExpectedRevision: "wrong", ResultRevision: "r5", IdempotencyKey: req.IdempotencyKey, RequestDigest: digestValue(req.Value), DecisionRef: "decision-1", Outcome: "updated", At: time.Now()}
	if _, err := r.Write(context.Background(), d.ID, req); err == nil {
		t.Fatal("mismatched owner receipt must be rejected")
	}
}

func TestRegistryRequiresOwnerAuthorizationBeforeRead(t *testing.T) {
	d := fixtureDescriptor()
	p := &provider{descriptor: d, readDecision: Decision{Allowed: false}, readDecisionSet: true}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Read(context.Background(), d.ID, ReadRequest{ActorID: "operator-1", Owner: d.Owner, Target: d.Target})
	if err == nil || p.reads != 0 {
		t.Fatalf("mismatched read authorization should refuse before source read, err=%v reads=%d", err, p.reads)
	}
}

func TestRegistryBindsPrincipalAndNeverReturnsSecretValues(t *testing.T) {
	d := fixtureDescriptor()
	p := &provider{descriptor: d}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	wrongPrincipal := ReadRequest{ActorID: "operator-2", Owner: d.Owner, Target: d.Target}
	if _, err := r.Read(context.Background(), d.ID, wrongPrincipal); err == nil || p.readAuth != 0 {
		t.Fatalf("principal mismatch should refuse before owner call, err=%v auth=%d", err, p.readAuth)
	}
	d.Sensitivity = "secret"
	p.descriptor = d
	p.readValue = []byte("credential material")
	r, err = NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Read(context.Background(), d.ID, ReadRequest{ActorID: "operator-1", Owner: d.Owner, Target: d.Target})
	if err == nil {
		t.Fatal("secret permission read must not return plaintext")
	}
}

func TestWritePermitBindsPayloadAndIdempotencyKey(t *testing.T) {
	d := fixtureDescriptor()
	p := &provider{descriptor: d}
	r, err := NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	req := WriteRequest{ActorID: "operator-1", Owner: d.Owner, Target: d.Target, ExpectedRevision: "r4", IdempotencyKey: "request-1", Value: []byte(`{"scope":"vrooli:read"}`)}
	receipt, err := r.Write(context.Background(), d.ID, req)
	if err != nil || receipt.RequestDigest != digestValue(req.Value) {
		t.Fatalf("receipt = %+v, err=%v", receipt, err)
	}
	if !p.permitValid {
		t.Fatal("provider received a valid request-bound permit")
	}
	permit := Permit{permissionID: d.ID, owner: d.Owner, actorID: req.ActorID, target: req.Target, expectedRevision: req.ExpectedRevision, idempotencyKey: req.IdempotencyKey, valueDigest: digestValue(req.Value), operation: OperationWrite}
	changedKey := req
	changedKey.IdempotencyKey = "replay-key"
	if permit.ValidForWrite(d.ID, changedKey) {
		t.Fatal("permit accepted an idempotency-key substitution")
	}
}

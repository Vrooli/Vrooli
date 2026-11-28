package credentialpermissions

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type fakeTargets struct {
	targets     []Target
	current     Target
	err         error
	lists, gets int
}

func TestActionDescriptorsRejectUnboundRevisionAndUnrepresentableGeneration(t *testing.T) {
	p, _, _, ctx := fixture()
	entries, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d := descriptorByID(entries, revokeActionID("grant-1"))
	d.SchemaVersion = ActionSchema + "/revoke#source=3;expected=1"
	if err := validateActionDescriptor(d, "revoke", "grant-1"); err == nil {
		t.Fatal("revoke descriptor with unrelated expected revision was accepted")
	}
	if err := validateTarget(permissions.TargetBinding{MachineID: "m", NodeID: "n", EnrollmentGeneration: math.MaxUint64}); err == nil {
		t.Fatal("generation outside Bridge int64 contract was accepted")
	}
}

func (f *fakeTargets) CurrentTargets(context.Context, string) ([]Target, error) {
	f.lists++
	return append([]Target(nil), f.targets...), f.err
}
func (f *fakeTargets) CurrentTarget(_ context.Context, _ string, want permissions.TargetBinding) (Target, error) {
	f.gets++
	if f.err != nil {
		return Target{}, f.err
	}
	if f.current.Binding != want {
		return Target{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("target generation changed"))
	}
	return f.current, nil
}

type fakeGrants struct {
	byNode                                          map[string][]GrantMetadata
	err                                             error
	calls                                           int
	onGet                                           func()
	createCalls, revokeCalls                        int
	createResult, revokeResult                      MutationReceipt
	createInput                                     CreateMutation
	createRequestID, revokeRequestID, revokeGrantID string
	revokeRevision                                  int64
}

func (f *fakeGrants) ListGrantMetadata(_ context.Context, _ string, target permissions.TargetBinding) ([]GrantMetadata, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return append([]GrantMetadata(nil), f.byNode[target.NodeID]...), nil
}
func (f *fakeGrants) GetGrantMetadata(_ context.Context, _ string, target permissions.TargetBinding, id string) (GrantMetadata, error) {
	f.calls++
	if f.onGet != nil {
		f.onGet()
	}
	if f.err != nil {
		return GrantMetadata{}, f.err
	}
	for _, grant := range f.byNode[target.NodeID] {
		if grant.ID == id {
			return grant, nil
		}
	}
	return GrantMetadata{}, connect.NewError(connect.CodeNotFound, errors.New("missing"))
}

func (f *fakeGrants) CreateGrantWithReceipt(_ context.Context, _ string, _ permissions.TargetBinding, requestID string, input CreateMutation) (MutationReceipt, error) {
	f.createCalls++
	f.createRequestID, f.createInput = requestID, input
	return f.createResult, f.err
}
func (f *fakeGrants) RevokeGrantWithReceipt(_ context.Context, _ string, _ permissions.TargetBinding, id string, revision int64, requestID string) (MutationReceipt, error) {
	f.revokeCalls++
	f.revokeGrantID, f.revokeRevision, f.revokeRequestID = id, revision, requestID
	return f.revokeResult, f.err
}

func fixture() (*Provider, *fakeTargets, *fakeGrants, context.Context) {
	binding := permissions.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 4}
	grant := GrantMetadata{ID: "grant-1", NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 3, AckedGeneration: 3, Version: "3", GrantedAt: time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC), ReceiptAt: time.Date(2026, 11, 20, 8, 1, 0, 0, time.UTC), ReceiptAccepted: true}
	targets := &fakeTargets{targets: []Target{{Binding: binding}}, current: Target{Binding: binding}}
	grants := &fakeGrants{byNode: map[string][]GrantMetadata{"node-1": {grant}}}
	p := NewProvider(targets, grants, func(context.Context) (string, error) { return "owner-session", nil }, WithOwnerSessionProvider(func(context.Context) (OwnerSession, error) {
		return OwnerSession{Token: "owner-session", OwnerID: "operator-1"}, nil
	}))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true})
	return p, targets, grants, ctx
}

func descriptorByID(entries []permissions.Descriptor, id string) permissions.Descriptor {
	for _, entry := range entries {
		if entry.ID == id {
			return entry
		}
	}
	return permissions.Descriptor{}
}

func TestListReadExposeOnlyBoundOpaqueGrantMetadata(t *testing.T) {
	p, targets, grants, ctx := fixture()
	reg, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := reg.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("inventory=%+v", entries)
	}
	var d permissions.Descriptor
	for _, entry := range entries {
		if entry.ID == "bridge/credential-grant/grant-1" {
			d = entry
		}
	}
	if d.ID != "bridge/credential-grant/grant-1" || d.Owner != Owner || d.SubjectID != "grant-1" || d.SubjectKind != SubjectKind || !d.TargetBound || d.Target.NodeID != "node-1" || d.Target.EnrollmentGeneration != 4 || !d.CanRead || d.CanWrite {
		t.Fatalf("descriptor=%+v", d)
	}
	result, err := reg.Read(ctx, d.ID, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: d.Target})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sealed_value", "secret_value", "plaintext", "receipt_reason", "purge_receipt_reason", "owner-session"} {
		if strings.Contains(string(result.Value), forbidden) {
			t.Fatalf("read leaked %q: %s", forbidden, result.Value)
		}
	}
	if !strings.Contains(string(result.Value), `"status":"delivered"`) || result.Revision == "" || targets.gets != 1 || grants.calls != 3 {
		t.Fatalf("read=%+v reads=%d grant calls=%d", result, targets.gets, grants.calls)
	}
}

func TestRevokedGrantIsVisibleAsStatusButCannotBeRead(t *testing.T) {
	p, _, grants, ctx := fixture()
	g := grants.byNode["node-1"][0]
	g.RevokedAt = time.Now().UTC()
	g.Version = "4"
	grants.byNode["node-1"] = []GrantMetadata{g}
	reg, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	d, err := reg.Inventory(ctx)
	if err != nil || len(d) != 3 {
		t.Fatalf("inventory=%+v err=%v", d, err)
	}
	readDescriptor := descriptorByID(d, permissionID("grant-1"))
	if readDescriptor.ID == "" {
		t.Fatalf("read descriptor missing: %+v", d)
	}
	if _, err = reg.Read(ctx, readDescriptor.ID, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: readDescriptor.Target}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("revoked read=%v", err)
	}
}

func TestReadRefusesMetadataRevisionChangedAfterInventory(t *testing.T) {
	p, _, grants, ctx := fixture()
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	grants.onGet = func() {
		grant := grants.byNode["node-1"][0]
		grant.Generation++
		grant.AckedGeneration++
		grant.Version = "4"
		grants.byNode["node-1"][0] = grant
	}
	if _, err = registry.Read(ctx, permissionID("grant-1"), permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: permissions.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 4}}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("changed source revision read=%v", err)
	}
}

func TestWrongActorAndRemoteTargetRefuseBeforeMetadataCalls(t *testing.T) {
	p, targets, grants, ctx := fixture()
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	read := descriptorByID(ds, permissionID("grant-1"))
	before := grants.calls
	wrong := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-2", Verified: true})
	if _, err = p.AuthorizeRead(wrong, read, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: read.Target}); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("actor mismatch=%v", err)
	}
	remote := read.Target
	remote.MachineID = "machine-2"
	if _, err = p.AuthorizeRead(ctx, read, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: remote}); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("target mismatch=%v", err)
	}
	if grants.calls != before || targets.gets != 0 {
		t.Fatalf("bad binding caused bridge access: grants=%d gets=%d", grants.calls, targets.gets)
	}
}

func TestChangedEnrollmentAndMalformedInventoryFailClosed(t *testing.T) {
	p, targets, grants, ctx := fixture()
	ds, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	read := descriptorByID(ds, permissionID("grant-1"))
	targets.current.Binding.EnrollmentGeneration++
	if _, err = p.AuthorizeRead(ctx, read, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: read.Target}); err != nil {
		t.Fatalf("authorization is local binding only: %v", err)
	}
	reg, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reg.Read(ctx, read.ID, permissions.ReadRequest{ActorID: "operator-1", Owner: Owner, Target: read.Target}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("changed generation=%v", err)
	}
	if grants.calls != 2 {
		t.Fatalf("stale target reached grant read: calls=%d", grants.calls)
	}
	targets.current = Target{Binding: read.Target}
	grants.byNode["node-1"][0].NodeID = "node-other"
	if _, err = reg.Inventory(ctx); connect.CodeOf(err) != connect.CodeDataLoss {
		t.Fatalf("cross-target grant=%v", err)
	}
}

func TestMissingOwnerTokenAndOwnerFailureAreSanitized(t *testing.T) {
	p, _, grants, ctx := fixture()
	p.token = func(context.Context) (string, error) { return "", nil }
	if _, err := p.List(ctx); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("missing token=%v", err)
	}
	if grants.calls != 0 {
		t.Fatalf("missing token reached Bridge: %d", grants.calls)
	}
	p, _, _, ctx = fixture()
	p.grants.(*fakeGrants).err = errors.New("secret-value-must-not-leak")
	if _, err := p.List(ctx); connect.CodeOf(err) != connect.CodeUnavailable || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("owner error=%v", err)
	}
}

func TestCredentialGrantRegistryWritesUseBridgeReceiptsAndReplayRequestBinding(t *testing.T) {
	p, _, grants, ctx := fixture()
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := registry.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	revoke := descriptorByID(entries, revokeActionID("grant-1"))
	if revoke.ID == "" || revoke.CanRead || !revoke.CanWrite {
		t.Fatalf("revoke descriptor=%+v", revoke)
	}
	req := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, Target: revoke.Target, ExpectedRevision: "3", IdempotencyKey: "request-revoke-1", Value: []byte(`{"operation":"revoke"}`)}
	grants.revokeResult = MutationReceipt{ID: "credential-grant-mutation:request-revoke-1", Operation: revokeGrantSubject, RequestID: req.IdempotencyKey, ActorID: req.ActorID, NodeID: req.Target.NodeID, EnrollmentGeneration: int64(req.Target.EnrollmentGeneration), Subject: revokeGrantSubject + ":grant-1", ExpectedRevision: 3, ResultRevision: 4, PayloadDigest: bridgeMutationDigest(req, "revoke", "grant-1", actionPayload{Operation: "revoke"}), Outcome: "revoked", DeliveryState: "pending", At: time.Now().UTC(), Grant: GrantMetadata{ID: "grant-1", NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 3, AckedGeneration: 3, Version: "4", RevokedAt: time.Now().UTC()}}
	first, err := registry.Write(ctx, revoke.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ResultRevision != "4" || first.OwnerState != "pending" || grants.revokeCalls != 1 || grants.revokeRequestID != req.IdempotencyKey || grants.revokeRevision != 3 || grants.revokeGrantID != "grant-1" {
		t.Fatalf("receipt=%+v calls=%d grant=%s revision=%d request=%s", first, grants.revokeCalls, grants.revokeGrantID, grants.revokeRevision, grants.revokeRequestID)
	}
	second, err := registry.Write(ctx, revoke.ID, req)
	if err != nil || second.ID != first.ID || second.RequestDigest != first.RequestDigest || grants.revokeCalls != 2 {
		t.Fatalf("same request replay receipt=%+v err=%v calls=%d", second, err, grants.revokeCalls)
	}
	grant := grants.byNode["node-1"][0]
	grant.RevokedAt, grant.Version = time.Now().UTC(), "4"
	grants.byNode["node-1"][0] = grant
	replayedAfterLostResponse, err := registry.Write(ctx, revoke.ID, req)
	if err != nil || replayedAfterLostResponse.ID != first.ID || replayedAfterLostResponse.ResultRevision != "4" || grants.revokeCalls != 3 {
		t.Fatalf("revocation receipt replay after state change=%+v err=%v calls=%d", replayedAfterLostResponse, err, grants.revokeCalls)
	}

	create := descriptorByID(entries, createActionID(targetKey(revoke.Target)))
	createReq := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, Target: create.Target, ExpectedRevision: "0", IdempotencyKey: "request-create-1", Value: []byte(`{"operation":"create","logical_id":"service/new","field":"token","class":"user_prompt","retention":"ephemeral","generation":1}`)}
	input := CreateMutation{LogicalID: "service/new", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1}
	grants.createResult = MutationReceipt{ID: "credential-grant-mutation:request-create-1", Operation: createGrantSubject, RequestID: createReq.IdempotencyKey, ActorID: createReq.ActorID, NodeID: createReq.Target.NodeID, EnrollmentGeneration: int64(createReq.Target.EnrollmentGeneration), Subject: createGrantSubject, ExpectedRevision: 0, ResultRevision: 1, PayloadDigest: bridgeMutationDigest(createReq, "create", create.SubjectID, actionPayload{Operation: "create", LogicalID: input.LogicalID, Field: input.Field, Class: input.Class, Retention: input.Retention, Generation: input.Generation}), Outcome: "created", At: time.Now().UTC(), Grant: GrantMetadata{ID: "grant-new", NodeID: "node-1", LogicalID: input.LogicalID, Field: input.Field, Class: input.Class, Retention: input.Retention, Generation: 1, Version: "1"}}
	created, err := registry.Write(ctx, create.ID, createReq)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResultRevision != "1" || grants.createCalls != 1 || grants.createInput != input || grants.createRequestID != createReq.IdempotencyKey {
		t.Fatalf("create receipt=%+v calls=%d input=%+v request=%s", created, grants.createCalls, grants.createInput, grants.createRequestID)
	}
}

func TestCredentialGrantMutationsRejectUnknownPayloadCrossActorAndStaleTargetBeforeOwnerWrite(t *testing.T) {
	p, targets, grants, ctx := fixture()
	entries, err := p.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	action := descriptorByID(entries, revokeActionID("grant-1"))
	base := permissions.WriteRequest{ActorID: "operator-1", Owner: Owner, Target: action.Target, ExpectedRevision: "3", IdempotencyKey: "request-1", Value: []byte(`{"operation":"revoke"}`)}
	for name, value := range map[string][]byte{
		"private value":            []byte(`{"operation":"revoke","secret":"do-not-forward"}`),
		"duplicate key":            []byte(`{"operation":"revoke","operation":"create"}`),
		"ambiguous trailing value": []byte(`{"operation":"revoke"} {}`),
	} {
		t.Run(name, func(t *testing.T) {
			req := base
			req.Value = value
			if _, err := p.Authorize(ctx, action, req); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("bad payload error=%v", err)
			}
		})
	}
	if grants.revokeCalls != 0 {
		t.Fatalf("invalid payload reached owner write %d times", grants.revokeCalls)
	}
	wrong := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-2", Verified: true})
	if _, err := p.Authorize(wrong, action, base); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross actor=%v", err)
	}
	if grants.revokeCalls != 0 {
		t.Fatalf("cross actor reached owner write %d times", grants.revokeCalls)
	}
	staleSource := base
	staleSource.ExpectedRevision = "2"
	if _, err := p.Authorize(ctx, action, staleSource); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale source revision=%v", err)
	}
	if grants.revokeCalls != 0 {
		t.Fatalf("stale source revision reached owner write %d times", grants.revokeCalls)
	}
	targets.current.Binding.EnrollmentGeneration++
	if _, err := p.Authorize(ctx, action, base); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale target=%v", err)
	}
	if grants.revokeCalls != 0 {
		t.Fatalf("stale target reached owner write %d times", grants.revokeCalls)
	}

	_, _, _, mismatchCtx := fixture()
	p.ownerSession = func(context.Context) (OwnerSession, error) {
		return OwnerSession{Token: "token-for-operator-2", OwnerID: "operator-2"}, nil
	}
	if _, err := p.Authorize(mismatchCtx, action, base); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Bridge owner identity mismatch=%v", err)
	}
	if grants.revokeCalls != 0 {
		t.Fatalf("mismatched Bridge owner reached owner write %d times", grants.revokeCalls)
	}
}

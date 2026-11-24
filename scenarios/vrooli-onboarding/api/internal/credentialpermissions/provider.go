// Package credentialpermissions projects Bridge-owned sealed credential grant
// metadata into onboarding's typed permission registry. It has no credential
// value, provisioning, or local persistence path.
package credentialpermissions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID         = "bridge.sealed-credential-grants"
	Owner              = "vrooli-bridge"
	SubjectKind        = "sealed-credential-grant"
	SchemaVersion      = "bridge/credential-grant-metadata/v1"
	Source             = "Bridge side-effect-free credential grant metadata API"
	ActionPrefix       = "bridge/credential-grant-action/"
	ActionSource       = "Bridge durable credential grant mutation receipt API"
	ActionSchema       = "bridge/credential-grant-mutation/v1"
	createGrantSubject = "credential-grant.create"
	revokeGrantSubject = "credential-grant.revoke"
)

type TokenProvider func(context.Context) (string, error)

// Target is a live Bridge machine/node binding. Sources must return only active
// machine bindings with their current node and enrollment generation.
type Target struct{ Binding permissions.TargetBinding }

// TargetReader is a narrow owner-backed read surface. It must not infer a
// target from an empty selector or return revoked/stale machine bindings.
type TargetReader interface {
	CurrentTargets(context.Context, string) ([]Target, error)
	CurrentTarget(context.Context, string, permissions.TargetBinding) (Target, error)
}

// GrantMetadata deliberately omits every credential value, sealed payload,
// raw receipt reason and purge reason. The Bridge implementation must be
// side-effect-free; the existing ListGrants RPC is not suitable because its
// handler also pushes sealed values to agents.
type GrantMetadata struct {
	ID              string
	NodeID          string
	LogicalID       string
	Field           string
	Class           string
	Retention       string
	Generation      int64
	AckedGeneration int64
	GrantedAt       time.Time
	RevokedAt       time.Time
	ReceiptAt       time.Time
	ReceiptAccepted bool
	PurgeState      string
	PurgeReceiptAt  time.Time
	PurgeAccepted   bool
	Version         string
}

type GrantMetadataClient interface {
	ListGrantMetadata(context.Context, string, permissions.TargetBinding) ([]GrantMetadata, error)
	GetGrantMetadata(context.Context, string, permissions.TargetBinding, string) (GrantMetadata, error)
	CreateGrantWithReceipt(context.Context, string, permissions.TargetBinding, string, CreateMutation) (MutationReceipt, error)
	RevokeGrantWithReceipt(context.Context, string, permissions.TargetBinding, string, int64, string) (MutationReceipt, error)
}

type CreateMutation struct {
	LogicalID, Field, Class, Retention string
	Generation                         int64
}

type MutationReceipt struct {
	ID, Operation, RequestID, ActorID, NodeID, Subject     string
	EnrollmentGeneration, ExpectedRevision, ResultRevision int64
	PayloadDigest, Outcome, DeliveryState                  string
	At                                                     time.Time
	Grant                                                  GrantMetadata
}

type OwnerSession struct{ Token, OwnerID string }
type OwnerSessionProvider func(context.Context) (OwnerSession, error)

type Provider struct {
	targets      TargetReader
	grants       GrantMetadataClient
	token        TokenProvider
	ownerSession OwnerSessionProvider
	now          func() time.Time
}

type Option func(*Provider)

func WithClock(now func() time.Time) Option { return func(p *Provider) { p.now = now } }
func WithOwnerSessionProvider(session OwnerSessionProvider) Option {
	return func(p *Provider) { p.ownerSession = session }
}

func NewProvider(targets TargetReader, grants GrantMetadataClient, token TokenProvider, opts ...Option) *Provider {
	p := &Provider{targets: targets, grants: grants, token: token, now: func() time.Time { return time.Now().UTC() }}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

var _ permissions.Provider = (*Provider)(nil)

func (*Provider) ProviderID() string { return ProviderID }
func (*Provider) Owner() string      { return Owner }

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	ctx, token, err := p.ownerContext(ctx)
	if err != nil {
		return nil, err
	}
	targets, err := p.targets.CurrentTargets(ctx, token)
	if err != nil {
		return nil, bridgeError("list current targets", err)
	}
	out := make([]permissions.Descriptor, 0)
	for _, target := range targets {
		if err := validateTarget(target.Binding); err != nil {
			return nil, connect.NewError(connect.CodeDataLoss, err)
		}
		grants, err := p.grants.ListGrantMetadata(ctx, token, target.Binding)
		if err != nil {
			return nil, bridgeError("list credential grant metadata", err)
		}
		for _, grant := range grants {
			if err := validateGrant(grant, target.Binding); err != nil {
				return nil, connect.NewError(connect.CodeDataLoss, err)
			}
			out = append(out, descriptor(actor, target.Binding, grant))
			if action, ok := revokeDescriptor(actor, target.Binding, grant); ok {
				out = append(out, action)
			}
		}
		out = append(out, createDescriptor(actor, target.Binding))
	}
	return out, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err := validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target, d.SubjectID)}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("credential grant read permit mismatch"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.ReadResult{}, err
	}
	ctx, token, err := p.ownerContext(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	target, err := p.targets.CurrentTarget(ctx, token, req.Target)
	if err != nil {
		return permissions.ReadResult{}, bridgeError("get current target", err)
	}
	if target.Binding != req.Target {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge target generation or current node changed"))
	}
	grant, err := p.grants.GetGrantMetadata(ctx, token, req.Target, d.SubjectID)
	if err != nil {
		return permissions.ReadResult{}, bridgeError("read credential grant metadata", err)
	}
	if err = validateGrant(grant, req.Target); err != nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodeDataLoss, err)
	}
	if grant.ID != d.SubjectID {
		return permissions.ReadResult{}, connect.NewError(connect.CodeNotFound, errors.New("credential grant metadata not found for target"))
	}
	if d.SchemaVersion != SchemaVersion+"#"+revision(grant) {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential grant metadata changed; refresh the permission inventory"))
	}
	if !grant.RevokedAt.IsZero() {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential grant is revoked"))
	}
	value, marshalErr := json.Marshal(safeView(grant))
	if marshalErr != nil {
		return permissions.ReadResult{}, marshalErr
	}
	sum := sha256.Sum256(value)
	return permissions.ReadResult{PermissionID: d.ID, Owner: Owner, Target: req.Target, Revision: revision(grant), Value: value, ValueDigest: hex.EncodeToString(sum[:]), ObservedAt: p.now().UTC()}, nil
}

// Mutation descriptors are separate from the read-only metadata descriptors.
// The provider forwards mutations only to Bridge's durable receipt APIs; it
// never stores grant state or secret values locally.
func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	op, subject, ok := decodeActionID(d.ID)
	if !ok || req.Owner != Owner || req.ActorID != actor.Subject || d.PrincipalID != actor.Subject || d.PrincipalKind != string(actor.Kind) || d.Owner != Owner || d.Version != permissions.ContractVersion || !d.TargetBound || d.Target != req.Target || d.Source != ActionSource || d.Scope != "vrooli-bridge:write" || d.Sensitivity != "operator" || d.CanRead || !d.CanWrite {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("credential grant mutation actor, owner, source, or target binding mismatch"))
	}
	if err := validateActionDescriptor(d, op, subject); err != nil {
		return permissions.Decision{}, err
	}
	_, err = parseMutation(req.Value, op)
	if err != nil {
		return permissions.Decision{}, err
	}
	if req.ExpectedRevision != actionExpectedRevision(d, op) {
		return permissions.Decision{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential grant source revision changed"))
	}
	session, err := p.currentOwnerSession(ctx, actor.Subject)
	if err != nil {
		return permissions.Decision{}, err
	}
	if _, err := p.currentTarget(ctx, session.Token, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target, subject+":"+req.IdempotencyKey)}, nil
}
func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("credential grant mutation permit mismatch"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Receipt{}, err
	}
	op, subject, ok := decodeActionID(d.ID)
	if !ok {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("credential grant action is unavailable"))
	}
	if req.Owner != Owner || req.ActorID != actor.Subject || d.PrincipalID != actor.Subject || d.PrincipalKind != string(actor.Kind) || d.Owner != Owner || d.Version != permissions.ContractVersion || !d.TargetBound || d.Target != req.Target || d.Source != ActionSource || d.Scope != "vrooli-bridge:write" || d.Sensitivity != "operator" || d.CanRead || !d.CanWrite {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("credential grant mutation actor, owner, source, or target binding mismatch"))
	}
	if err := validateActionDescriptor(d, op, subject); err != nil {
		return permissions.Receipt{}, err
	}
	mutation, err := parseMutation(req.Value, op)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if req.ExpectedRevision != actionExpectedRevision(d, op) {
		return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential grant source revision changed"))
	}
	session, err := p.currentOwnerSession(ctx, actor.Subject)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if _, err := p.currentTarget(ctx, session.Token, req.Target); err != nil {
		return permissions.Receipt{}, err
	}
	var ownerReceipt MutationReceipt
	switch op {
	case "create":
		ownerReceipt, err = p.grants.CreateGrantWithReceipt(ctx, session.Token, req.Target, req.IdempotencyKey, CreateMutation{LogicalID: mutation.LogicalID, Field: mutation.Field, Class: mutation.Class, Retention: mutation.Retention, Generation: mutation.Generation})
	case "revoke":
		revision, parseErr := strconv.ParseInt(req.ExpectedRevision, 10, 64)
		if parseErr != nil || revision <= 0 {
			return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential grant revision is invalid"))
		}
		ownerReceipt, err = p.grants.RevokeGrantWithReceipt(ctx, session.Token, req.Target, subject, revision, req.IdempotencyKey)
	default:
		return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("credential grant mutation is unsupported"))
	}
	if err != nil {
		return permissions.Receipt{}, bridgeError("credential grant mutation", err)
	}
	if err := validateMutationReceipt(ownerReceipt, actor.Subject, req, d, op, subject); err != nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, err)
	}
	digest := sha256.Sum256(req.Value)
	return permissions.Receipt{ID: ownerReceipt.ID, PermissionID: d.ID, Owner: Owner, ActorID: actor.Subject, Target: req.Target, ExpectedRevision: req.ExpectedRevision, ResultRevision: strconv.FormatInt(ownerReceipt.ResultRevision, 10), IdempotencyKey: req.IdempotencyKey, RequestDigest: hex.EncodeToString(digest[:]), DecisionRef: permit.DecisionRef(), Outcome: ownerReceipt.Outcome, OwnerState: ownerReceipt.DeliveryState, At: ownerReceipt.At.UTC()}, nil
}

type SafeGrant struct {
	ID              string     `json:"id"`
	LogicalID       string     `json:"logical_id"`
	Field           string     `json:"field"`
	Class           string     `json:"class"`
	Retention       string     `json:"retention"`
	Generation      int64      `json:"generation"`
	AckedGeneration int64      `json:"acked_generation"`
	Status          string     `json:"status"`
	GrantedAt       *time.Time `json:"granted_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	ReceiptAt       *time.Time `json:"receipt_at,omitempty"`
	ReceiptAccepted bool       `json:"receipt_accepted"`
	PurgeState      string     `json:"purge_state,omitempty"`
	PurgeReceiptAt  *time.Time `json:"purge_receipt_at,omitempty"`
	PurgeAccepted   bool       `json:"purge_accepted"`
}

func safeView(g GrantMetadata) SafeGrant {
	status := "awaiting_receipt"
	if g.AckedGeneration >= g.Generation && g.ReceiptAccepted {
		status = "delivered"
	}
	if !g.RevokedAt.IsZero() {
		status = "revoked"
	}
	return SafeGrant{ID: g.ID, LogicalID: g.LogicalID, Field: g.Field, Class: g.Class, Retention: g.Retention, Generation: g.Generation, AckedGeneration: g.AckedGeneration, Status: status, GrantedAt: nonzeroTime(g.GrantedAt), RevokedAt: nonzeroTime(g.RevokedAt), ReceiptAt: nonzeroTime(g.ReceiptAt), ReceiptAccepted: g.ReceiptAccepted, PurgeState: g.PurgeState, PurgeReceiptAt: nonzeroTime(g.PurgeReceiptAt), PurgeAccepted: g.PurgeAccepted}
}

func nonzeroTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func descriptor(actor identity.Principal, target permissions.TargetBinding, g GrantMetadata) permissions.Descriptor {
	return permissions.Descriptor{Version: permissions.ContractVersion, ID: permissionID(g.ID), Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-bridge:read", SubjectKind: SubjectKind, SubjectID: g.ID, TargetBound: true, Target: target, Source: Source, SchemaVersion: SchemaVersion + "#" + revision(g), CanRead: true, CanWrite: false, Sensitivity: "operator"}
}

func createDescriptor(actor identity.Principal, target permissions.TargetBinding) permissions.Descriptor {
	key := targetKey(target)
	return permissions.Descriptor{Version: permissions.ContractVersion, ID: createActionID(key), Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-bridge:write", SubjectKind: "credential-grant-create", SubjectID: key, TargetBound: true, Target: target, Source: ActionSource, SchemaVersion: ActionSchema + "/create#0", CanWrite: true, Sensitivity: "operator"}
}

func revokeDescriptor(actor identity.Principal, target permissions.TargetBinding, g GrantMetadata) (permissions.Descriptor, bool) {
	version, err := strconv.ParseInt(g.Version, 10, 64)
	if err != nil || version <= 0 || (!g.RevokedAt.IsZero() && version <= 1) {
		return permissions.Descriptor{}, false
	}
	expectedRevision := version
	if !g.RevokedAt.IsZero() {
		expectedRevision--
	}
	return permissions.Descriptor{Version: permissions.ContractVersion, ID: revokeActionID(g.ID), Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-bridge:write", SubjectKind: "credential-grant-revoke", SubjectID: g.ID, TargetBound: true, Target: target, Source: ActionSource, SchemaVersion: fmt.Sprintf("%s/revoke#source=%d;expected=%d", ActionSchema, version, expectedRevision), CanWrite: true, Sensitivity: "operator"}, true
}

func targetKey(t permissions.TargetBinding) string {
	value := fmt.Sprintf("%s\x00%s\x00%d", t.MachineID, t.NodeID, t.EnrollmentGeneration)
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}
func createActionID(key string) string { return ActionPrefix + "create/" + key }
func revokeActionID(id string) string {
	return ActionPrefix + "revoke/" + base64.RawURLEncoding.EncodeToString([]byte(id))
}
func decodeActionID(id string) (string, string, bool) {
	value, ok := strings.CutPrefix(id, ActionPrefix)
	if !ok {
		return "", "", false
	}
	op, encoded, ok := strings.Cut(value, "/")
	if !ok || (op != "create" && op != "revoke") || encoded == "" {
		return "", "", false
	}
	subjectBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(subjectBytes) == 0 {
		return "", "", false
	}
	if op == "create" {
		return op, encoded, true
	}
	return op, string(subjectBytes), true
}

type actionPayload struct {
	Operation  string `json:"operation"`
	LogicalID  string `json:"logical_id,omitempty"`
	Field      string `json:"field,omitempty"`
	Class      string `json:"class,omitempty"`
	Retention  string `json:"retention,omitempty"`
	Generation int64  `json:"generation,omitempty"`
}

func parseMutation(raw []byte, operation string) (actionPayload, error) {
	var value actionPayload
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload must be an object"))
	}
	keys := map[string]bool{}
	for decoder.More() {
		keyToken, keyErr := decoder.Token()
		key, ok := keyToken.(string)
		if keyErr != nil || !ok || keys[key] {
			return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload has duplicate or invalid fields"))
		}
		keys[key] = true
		var field json.RawMessage
		if err := decoder.Decode(&field); err != nil {
			return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload is invalid"))
		}
	}
	if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload is invalid"))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload contains trailing data"))
	}
	if operation == "revoke" && (len(keys) != 1 || !keys["operation"]) {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("revoke accepts only an operation field"))
	}
	if operation == "create" {
		for _, key := range []string{"operation", "logical_id", "field", "class", "retention", "generation"} {
			if !keys[key] {
				return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("create requires complete metadata fields"))
			}
		}
		if len(keys) != 6 {
			return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("create payload contains unsupported fields"))
		}
	}
	if operation != "create" && operation != "revoke" {
		return actionPayload{}, connect.NewError(connect.CodeUnimplemented, errors.New("credential grant mutation is unsupported"))
	}
	strict := json.NewDecoder(strings.NewReader(string(raw)))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&value); err != nil {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action payload is invalid"))
	}
	if value.Operation != operation {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant action operation mismatch"))
	}
	if operation == "revoke" {
		if value.LogicalID != "" || value.Field != "" || value.Class != "" || value.Retention != "" || value.Generation != 0 {
			return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("revoke accepts no metadata fields"))
		}
		return value, nil
	}
	if value.LogicalID == "" || strings.TrimSpace(value.LogicalID) != value.LogicalID || value.Field == "" || strings.TrimSpace(value.Field) != value.Field || value.Generation != 1 || (value.Class != "user_prompt" && value.Class != "remote_fetch" && value.Class != "infrastructure") || (value.Retention != "durable" && value.Retention != "ephemeral") || (value.Class == "infrastructure" && value.Retention == "durable") {
		return actionPayload{}, connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant create metadata is unsupported"))
	}
	return value, nil
}

func validateActionDescriptor(d permissions.Descriptor, operation, subject string) error {
	if err := validateTarget(d.Target); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if operation == "create" {
		if d.ID != createActionID(subject) || subject != targetKey(d.Target) || d.SubjectID != subject || d.SubjectKind != "credential-grant-create" || d.SchemaVersion != ActionSchema+"/create#0" {
			return connect.NewError(connect.CodePermissionDenied, errors.New("credential grant create permission descriptor mismatch"))
		}
		return nil
	}
	if operation == "revoke" {
		source, expected, ok := parseRevokeRevision(d.SchemaVersion)
		if !ok || source <= 0 || expected <= 0 || (expected != source && expected != source-1) || d.ID != revokeActionID(subject) || d.SubjectID != subject || d.SubjectKind != "credential-grant-revoke" {
			return connect.NewError(connect.CodePermissionDenied, errors.New("credential grant revoke permission descriptor mismatch"))
		}
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, errors.New("credential grant action is unsupported"))
}

func actionExpectedRevision(d permissions.Descriptor, operation string) string {
	if operation == "create" {
		return "0"
	}
	_, value, ok := parseRevokeRevision(d.SchemaVersion)
	if !ok {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func parseRevokeRevision(schema string) (source, expected int64, ok bool) {
	value, found := strings.CutPrefix(schema, ActionSchema+"/revoke#source=")
	if !found {
		return 0, 0, false
	}
	sourceText, expectedText, found := strings.Cut(value, ";expected=")
	if !found {
		return 0, 0, false
	}
	source, err1 := strconv.ParseInt(sourceText, 10, 64)
	expected, err2 := strconv.ParseInt(expectedText, 10, 64)
	return source, expected, err1 == nil && err2 == nil
}

func (p *Provider) currentOwnerSession(ctx context.Context, actorID string) (OwnerSession, error) {
	if p.ownerSession == nil {
		return OwnerSession{}, connect.NewError(connect.CodeUnavailable, errors.New("Bridge owner session identity is unavailable"))
	}
	session, err := p.ownerSession(ctx)
	if err != nil || strings.TrimSpace(session.Token) == "" {
		return OwnerSession{}, connect.NewError(connect.CodeUnauthenticated, errors.New("Bridge owner session is unavailable"))
	}
	if session.OwnerID == "" || session.OwnerID != actorID {
		return OwnerSession{}, connect.NewError(connect.CodePermissionDenied, errors.New("onboarding principal does not match Bridge owner session"))
	}
	return session, nil
}

func (p *Provider) currentTarget(ctx context.Context, token string, target permissions.TargetBinding) (Target, error) {
	current, err := p.targets.CurrentTarget(ctx, token, target)
	if err != nil {
		return Target{}, bridgeError("current target validation", err)
	}
	if current.Binding != target {
		return Target{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge target generation or current node changed"))
	}
	return current, nil
}

func validateMutationReceipt(receipt MutationReceipt, actorID string, req permissions.WriteRequest, d permissions.Descriptor, operation, subject string) error {
	mutation, err := parseMutation(req.Value, operation)
	if err != nil {
		return err
	}
	wantSubject, wantOutcome, wantOperation := createGrantSubject, "created", createGrantSubject
	wantRevision := int64(0)
	if operation == "revoke" {
		wantRevision, err = strconv.ParseInt(req.ExpectedRevision, 10, 64)
		if err != nil {
			return err
		}
		wantSubject, wantOutcome, wantOperation = revokeGrantSubject+":"+subject, "revoked", revokeGrantSubject
	}
	if receipt.ID != "credential-grant-mutation:"+req.IdempotencyKey || receipt.Operation != wantOperation || receipt.RequestID != req.IdempotencyKey || receipt.ActorID != actorID || receipt.NodeID != req.Target.NodeID || receipt.EnrollmentGeneration != int64(req.Target.EnrollmentGeneration) || receipt.Subject != wantSubject || receipt.ExpectedRevision != wantRevision || receipt.ResultRevision != wantRevision+1 || receipt.PayloadDigest != bridgeMutationDigest(req, operation, subject, mutation) || receipt.Outcome != wantOutcome || receipt.At.IsZero() || receipt.Grant.NodeID != req.Target.NodeID || receipt.Grant.ID == "" {
		return errors.New("Bridge returned an incomplete or mismatched credential mutation receipt")
	}
	if operation == "revoke" && (receipt.Grant.ID != subject || receipt.Grant.RevokedAt.IsZero() || receipt.Grant.Version != strconv.FormatInt(receipt.ResultRevision, 10)) {
		return errors.New("Bridge revoke receipt does not bind the current grant revision")
	}
	if operation == "create" && (receipt.Grant.LogicalID != mutation.LogicalID || receipt.Grant.Field != mutation.Field || receipt.Grant.Class != mutation.Class || receipt.Grant.Retention != mutation.Retention || receipt.Grant.Generation != mutation.Generation || receipt.Grant.Version != strconv.FormatInt(receipt.ResultRevision, 10)) {
		return errors.New("Bridge create receipt does not bind the requested grant metadata")
	}
	return nil
}

func bridgeMutationDigest(req permissions.WriteRequest, operation, subject string, mutation actionPayload) string {
	var value any
	if operation == "create" {
		value = struct {
			Operation, NodeID, LogicalID, Field, Class, Retention string
			EnrollmentGeneration, Generation, ExpectedRevision    int64
		}{"create", req.Target.NodeID, mutation.LogicalID, mutation.Field, mutation.Class, mutation.Retention, int64(req.Target.EnrollmentGeneration), mutation.Generation, 0}
	} else {
		revision, _ := strconv.ParseInt(req.ExpectedRevision, 10, 64)
		value = struct {
			Operation, ID, NodeID                  string
			EnrollmentGeneration, ExpectedRevision int64
		}{"revoke", subject, req.Target.NodeID, int64(req.Target.EnrollmentGeneration), revision}
	}
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func validateBinding(actor identity.Principal, d permissions.Descriptor, actorID, owner string, target permissions.TargetBinding) error {
	if err := validateTarget(target); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if d.Version != permissions.ContractVersion || d.ID != permissionID(d.SubjectID) || d.Owner != Owner || owner != Owner || actor.Subject != actorID || d.PrincipalID != actorID || d.PrincipalKind != string(actor.Kind) || d.Scope != "vrooli-bridge:read" || !d.TargetBound || d.Target != target || d.SubjectKind != SubjectKind || strings.TrimSpace(d.SubjectID) == "" || d.Source != Source || !strings.HasPrefix(d.SchemaVersion, SchemaVersion+"#") || strings.TrimPrefix(d.SchemaVersion, SchemaVersion+"#") == "" || !d.CanRead || d.CanWrite || d.Sensitivity != "operator" {
		return connect.NewError(connect.CodePermissionDenied, errors.New("credential grant actor, owner, permission, subject, or target mismatch"))
	}
	return nil
}
func validateTarget(t permissions.TargetBinding) error {
	if strings.TrimSpace(t.MachineID) == "" || strings.TrimSpace(t.NodeID) == "" || strings.TrimSpace(t.MachineID) != t.MachineID || strings.TrimSpace(t.NodeID) != t.NodeID || t.EnrollmentGeneration == 0 || t.EnrollmentGeneration > math.MaxInt64 {
		return errors.New("Bridge machine, current node, and enrollment generation are required")
	}
	return nil
}
func validateGrant(g GrantMetadata, t permissions.TargetBinding) error {
	if g.ID == "" || strings.TrimSpace(g.ID) != g.ID || g.NodeID != t.NodeID || g.LogicalID == "" || g.Field == "" || g.Generation <= 0 || g.AckedGeneration < 0 || g.AckedGeneration > g.Generation {
		return fmt.Errorf("Bridge returned invalid or cross-target credential grant metadata")
	}
	switch g.Class {
	case "infrastructure", "per_install_generated", "user_prompt", "remote_fetch":
	default:
		return errors.New("Bridge returned an unsupported credential class")
	}
	switch g.Retention {
	case "durable", "ephemeral":
	default:
		return errors.New("Bridge returned an unsupported retention mode")
	}
	return nil
}
func (p *Provider) ownerContext(ctx context.Context) (context.Context, string, error) {
	if p.targets == nil || p.grants == nil || p.token == nil {
		return nil, "", connect.NewError(connect.CodeUnavailable, errors.New("Bridge credential metadata provider is unavailable"))
	}
	token, err := p.token(ctx)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("Bridge owner session is unavailable"))
	}
	if strings.TrimSpace(token) == "" {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("Bridge owner session is unavailable"))
	}
	return ctx, token, nil
}
func verifiedActor(ctx context.Context) (identity.Principal, error) {
	a, ok := identity.PrincipalFromContext(ctx)
	if !ok || !a.IsHuman() {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human operator required"))
	}
	return a, nil
}
func permissionID(id string) string { return "bridge/credential-grant/" + id }
func revision(g GrantMetadata) string {
	if g.Version != "" {
		return g.Version
	}
	revoked := "active"
	if !g.RevokedAt.IsZero() {
		revoked = "revoked:" + g.RevokedAt.UTC().Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("%d:%d:%t:%s", g.Generation, g.AckedGeneration, g.ReceiptAccepted, revoked)
}
func decisionRef(actor string, target permissions.TargetBinding, id string) string {
	return fmt.Sprintf("bridge-credential:%s:%s:%s:%d", actor, target.NodeID, id, target.EnrollmentGeneration)
}
func bridgeError(op string, err error) error {
	code := connect.CodeUnavailable
	if c := connect.CodeOf(err); c != connect.CodeUnknown {
		code = c
	}
	return connect.NewError(code, fmt.Errorf("Bridge %s request failed", op))
}

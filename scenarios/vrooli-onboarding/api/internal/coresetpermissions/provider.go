// Package coresetpermissions exposes the operator-owned seed/trusted-base
// authority through onboarding's typed permission registry. Storage and CAS
// remain owned by operatorstateapi; the read-only provider never keeps a local
// copy, and writes are enabled only when an owner supplies durable CAS receipts.
package coresetpermissions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID    = "onboarding.core-authority"
	Owner         = "vrooli-onboarding"
	PermissionID  = "onboarding/core-authority"
	SubjectKind   = "operator-state-document"
	SubjectID     = "local"
	LocalTarget   = "local"
	SchemaVersion = "operator-state/core-authority/v2"
	Source        = "api-core/coreset + owner OperatorState API"
	WriteSubject  = "operator-state-document:local"
)

// StateReader is satisfied by operatorstateapi.Service.Get. It is deliberately
// only the owning API's read surface; this provider has no storage dependency.
type StateReader interface {
	Get(context.Context) (operatorstate.Document, error)
}

// TargetResolver returns the currently admitted machine/node/generation for
// the authenticated operator. It must fail closed when the selection is
// absent, stale, revoked, or no longer current.
// StateMutation is the owner-side atomic request contract. An implementation
// must compare expected_revision, apply only the core authority field through
// the OperatorState owner, and durably store/replay the identity-bound receipt
// in the same transaction. It must not update a private copy or replace closure
// digest state.
type StateMutation struct {
	ActorID          string
	Target           string
	ExpectedRevision string
	RequestID        string
	RequestDigest    string
	AuthorityDigest  string
	Authority        coreset.Authority
}

type StateReceipt struct {
	RequestID        string
	ActorID          string
	Target           string
	Subject          string
	ExpectedRevision string
	NewRevision      string
	RequestDigest    string
	AuthorityDigest  string
	Outcome          string
	CreatedAt        time.Time
}

type ErrRevisionConflict struct{ Expected, Current string }

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("operator-state core revision changed: expected %q, current %q", e.Expected, e.Current)
}
func (e ErrRevisionConflict) RevisionConflict() bool { return true }

type ErrRequestIdentityConflict struct{ RequestID string }

func (e ErrRequestIdentityConflict) Error() string {
	return fmt.Sprintf("operator-state request %q was reused with different authority", e.RequestID)
}

// StateWriter is intentionally separate from StateReader. Production writes
// are enabled only through NewOwnerProvider, which adapts the owner API's
// atomic request-ID receipt transaction to this narrow core-only contract.
type StateWriter interface {
	CompareAndSetCore(context.Context, StateMutation) (StateReceipt, error)
}

type MutationAuthorizer func(context.Context) error
type Option func(*Provider)

func WithMutationAuthorizer(authorize MutationAuthorizer) Option {
	return func(p *Provider) { p.authorizeMutation = authorize }
}
func WithClock(now func() time.Time) Option { return func(p *Provider) { p.now = now } }

type Provider struct {
	reader            StateReader
	writer            StateWriter
	validateInventory func(kind, name string) error
	authorizeMutation MutationAuthorizer
	now               func() time.Time
}

func (p *Provider) WithInventory(validate func(kind, name string) error) *Provider {
	p.validateInventory = validate
	return p
}

func NewProvider(reader StateReader, writers ...StateWriter) *Provider {
	p := &Provider{reader: reader, authorizeMutation: authz.RequireMutation, now: func() time.Time { return time.Now().UTC() }}
	if len(writers) > 0 {
		p.writer = writers[0]
	}
	return p
}
func (p *Provider) WithOptions(opts ...Option) *Provider {
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
	if p.reader == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("core authority provider is unavailable"))
	}
	if _, _, err = p.readAuthority(ctx); err != nil {
		return nil, err
	}
	d := descriptor(actor, p.writer != nil)
	return []permissions.Descriptor{d}, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target)}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("core authority read permit mismatch"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.ReadResult{}, err
	}
	authority, revision, err := p.readAuthority(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	value, err := json.Marshal(authority)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	digest := sha256.Sum256(value)
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Target: req.Target, Revision: revision, Value: value, ValueDigest: hex.EncodeToString(digest[:]), ObservedAt: p.now().UTC()}, nil
}

func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	if !d.CanWrite || p.writer == nil {
		return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("core authority writes require an owner-side atomic CAS and durable receipt adapter"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Decision{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Decision{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected revision and request id are required"))
	}
	if _, err = decodeAuthority(req.Value); err != nil {
		return permissions.Decision{}, err
	}
	if p.authorizeMutation == nil {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("core authority mutation authorization is unavailable"))
	}
	if err = p.authorizeMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: decisionRef(actor.Subject, req.Target) + ":mutation"}, nil
}

func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !d.CanWrite || p.writer == nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("core authority writes require an owner-side atomic CAS and durable receipt adapter"))
	}
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, connect.NewError(connect.CodePermissionDenied, errors.New("core authority write permit mismatch"))
	}
	actor, err := verifiedActor(ctx)
	if err != nil {
		return permissions.Receipt{}, err
	}
	if err = validateBinding(actor, d, req.ActorID, req.Owner, req.Target); err != nil {
		return permissions.Receipt{}, err
	}
	if strings.TrimSpace(req.ExpectedRevision) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("expected revision and request id are required"))
	}
	authority, err := decodeAuthority(req.Value)
	if err != nil {
		return permissions.Receipt{}, err
	}
	current, err := p.reader.Get(ctx)
	if err != nil {
		return permissions.Receipt{}, mapStateError(err)
	}
	if err := validateTierSelection(authority, current); err != nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	tierEntries := append(append([]coreset.ComponentID(nil), authority.CoreMembers...), authority.Important...)
	if len(tierEntries) > 0 && p.validateInventory == nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeUnavailable, errors.New("canonical scenario/resource inventory is unavailable"))
	}
	for _, member := range tierEntries {
		if err := p.validateInventory(member.Kind, member.Name); err != nil {
			return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	requestDigest := digest(req.Value)
	authorityBytes, err := json.Marshal(authority)
	if err != nil {
		return permissions.Receipt{}, err
	}
	authorityDigest := "sha256:" + digest(authorityBytes)
	mutation := StateMutation{ActorID: actor.Subject, Target: LocalTarget, ExpectedRevision: req.ExpectedRevision, RequestID: req.IdempotencyKey, RequestDigest: requestDigest, AuthorityDigest: authorityDigest, Authority: authority}
	stateReceipt, err := p.writer.CompareAndSetCore(ctx, mutation)
	if err != nil {
		return permissions.Receipt{}, mapStateError(err)
	}
	if stateReceipt.RequestID != req.IdempotencyKey || stateReceipt.ActorID != actor.Subject || stateReceipt.Target != LocalTarget || stateReceipt.Subject != WriteSubject || stateReceipt.ExpectedRevision != req.ExpectedRevision || stateReceipt.NewRevision == "" || stateReceipt.NewRevision == req.ExpectedRevision || stateReceipt.RequestDigest != requestDigest || stateReceipt.AuthorityDigest != authorityDigest || stateReceipt.Outcome == "" || stateReceipt.CreatedAt.IsZero() {
		return permissions.Receipt{}, connect.NewError(connect.CodeDataLoss, errors.New("OperatorState returned an unbound core authority receipt"))
	}
	return permissions.Receipt{ID: "operator-state-core:" + stateReceipt.RequestID, PermissionID: d.ID, Owner: d.Owner, ActorID: stateReceipt.ActorID, Target: req.Target, ExpectedRevision: stateReceipt.ExpectedRevision, ResultRevision: stateReceipt.NewRevision, IdempotencyKey: stateReceipt.RequestID, RequestDigest: requestDigest, DecisionRef: permit.DecisionRef(), Outcome: stateReceipt.Outcome, At: stateReceipt.CreatedAt.UTC()}, nil
}

func validateTierSelection(authority coreset.Authority, current operatorstate.Document) error {
	currentSeed := map[string]struct{}{}
	if current.Core != nil {
		for _, name := range current.Core.Seed {
			currentSeed[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}
	}
	requiredCore := map[string]struct{}{}
	for _, name := range coreset.RequiredOperationalScenarios {
		requiredCore[name] = struct{}{}
	}
	for _, name := range authority.Seed {
		name = strings.ToLower(strings.TrimSpace(name))
		if _, existed := currentSeed[name]; existed {
			continue
		}
		if _, required := requiredCore[name]; required {
			continue
		}
		choice, ok := current.Scenarios[name]
		if !ok || choice.Enabled == nil || !*choice.Enabled {
			return fmt.Errorf("scenario %q must be selected before it can enter core.seed", name)
		}
	}
	for _, member := range append(append([]coreset.ComponentID(nil), authority.CoreMembers...), authority.Important...) {
		switch member.Kind {
		case coreset.MemberKindScenario:
			if containsName(authority.Seed, member.Name) {
				continue
			} // core is stronger and subsumes important.
			choice, ok := current.Scenarios[member.Name]
			if !ok || choice.Enabled == nil || !*choice.Enabled {
				return fmt.Errorf("scenario %q must be selected before it can receive an autoheal tier", member.Name)
			}
		case coreset.MemberKindResource:
			choice, ok := current.Resources[member.Name]
			if !ok || choice.Enabled == nil || !*choice.Enabled {
				return fmt.Errorf("resource %q must be selected before it can receive an autoheal tier", member.Name)
			}
		}
	}
	return nil
}

func containsName(values []string, name string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), name) {
			return true
		}
	}
	return false
}

func (p *Provider) readAuthority(ctx context.Context) (coreset.Authority, string, error) {
	if p.reader == nil {
		return coreset.Authority{}, "", connect.NewError(connect.CodeUnavailable, errors.New("OperatorState reader is unavailable"))
	}
	doc, err := p.reader.Get(ctx)
	if err != nil {
		return coreset.Authority{}, "", mapStateError(err)
	}
	authority := coreset.Authority{}
	if doc.Core != nil {
		authority = coreset.Authority{Seed: append([]string(nil), doc.Core.Seed...), TrustedBase: append([]string(nil), doc.Core.TrustedBase...), Important: append([]coreset.ComponentID(nil), doc.Core.Important...), CoreMembers: append([]coreset.ComponentID(nil), doc.Core.CoreMembers...)}
	}
	authority = coreset.NormalizeOperationalAuthority(authority)
	if err = authority.Validate(); err != nil {
		return coreset.Authority{}, "", connect.NewError(connect.CodeDataLoss, fmt.Errorf("stored core authority is invalid: %w", err))
	}
	revision := operatorstate.Revision(doc)
	if strings.TrimSpace(revision) == "" {
		return coreset.Authority{}, "", connect.NewError(connect.CodeDataLoss, errors.New("OperatorState has no revision"))
	}
	return authority, revision, nil
}

func decodeAuthority(raw []byte) (coreset.Authority, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return coreset.Authority{}, connect.NewError(connect.CodeInvalidArgument, errors.New("core authority payload must be a JSON object"))
	}
	var authority coreset.Authority
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&authority); err != nil {
		return coreset.Authority{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode core authority: %w", err))
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return coreset.Authority{}, connect.NewError(connect.CodeInvalidArgument, errors.New("core authority payload contains trailing JSON"))
		}
		return coreset.Authority{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode trailing core authority data: %w", err))
	}
	authority = coreset.NormalizeOperationalAuthority(authority)
	if err := authority.Validate(); err != nil {
		return coreset.Authority{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return authority, nil
}

func descriptor(actor identity.Principal, writable bool) permissions.Descriptor {
	return permissions.Descriptor{Version: permissions.ContractVersion, ID: PermissionID, Owner: Owner, PrincipalKind: string(actor.Kind), PrincipalID: actor.Subject, Scope: "vrooli-onboarding:read", SubjectKind: SubjectKind, SubjectID: SubjectID, TargetBound: false, Source: Source, SchemaVersion: SchemaVersion, CanRead: true, CanWrite: writable, Sensitivity: "operator"}
}
func validateBinding(actor identity.Principal, d permissions.Descriptor, actorID, owner string, target permissions.TargetBinding) error {
	if d.Version != permissions.ContractVersion || d.ID != PermissionID || d.Owner != Owner || owner != Owner || actor.Subject != actorID || d.PrincipalID != actorID || d.PrincipalKind != string(actor.Kind) || d.Scope != "vrooli-onboarding:read" || d.TargetBound || d.Target != (permissions.TargetBinding{}) || target != (permissions.TargetBinding{}) || d.SubjectKind != SubjectKind || d.SubjectID != SubjectID || d.Source != Source || d.SchemaVersion != SchemaVersion || d.Sensitivity != "operator" || !d.CanRead {
		return connect.NewError(connect.CodePermissionDenied, errors.New("core authority actor, owner, permission, subject, or target mismatch"))
	}
	return nil
}
func verifiedActor(ctx context.Context) (identity.Principal, error) {
	actor, ok := identity.PrincipalFromContext(ctx)
	if !ok || !actor.IsHuman() {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified human operator required"))
	}
	return actor, nil
}
func decisionRef(actor string, _ permissions.TargetBinding) string {
	return "operator-state-core:" + actor + ":" + LocalTarget
}
func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func mapStateError(err error) error {
	if err == nil {
		return nil
	}
	var ce *connect.Error
	if errors.As(err, &ce) {
		return err
	}
	if errors.Is(err, operatorstate.ErrRevisionConflict) {
		return connect.NewError(connect.CodeAborted, errors.New("operator-state revision changed"))
	}
	if errors.Is(err, operatorstate.ErrMutationReceiptConflict) {
		return connect.NewError(connect.CodeAlreadyExists, errors.New("operator-state request ID is bound to another mutation"))
	}
	var conflict ErrRevisionConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeAborted, err)
	}
	var reused ErrRequestIdentityConflict
	if errors.As(err, &reused) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	return connect.NewError(connect.CodeUnavailable, err)
}

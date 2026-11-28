package operatorstateapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const permissionID = "onboarding/operator-state"

// PermissionPatch is the owner-defined payload for the shared permission
// registry. The update mask preserves the existing field-scoped write boundary.
type PermissionPatch struct {
	State      json.RawMessage `json:"state"`
	UpdateMask []string        `json:"update_mask"`
}

type permissionProvider struct{ service Service }

func (s Service) PatchWithReceipt(ctx context.Context, stateJSON []byte, updateMask []string, expectedRevision string, binding operatorstate.MutationReceiptBinding) (operatorstate.MutationReceipt, error) {
	if s.store == nil {
		return operatorstate.MutationReceipt{}, errors.New("operator state store is not configured")
	}
	patch, err := maskedPatch(stateJSON, updateMask)
	if err != nil {
		return operatorstate.MutationReceipt{}, err
	}
	patch, err = normalizeCorePatch(patch)
	if err != nil {
		return operatorstate.MutationReceipt{}, err
	}
	_, receipt, _, err := s.store.ApplyAtRevisionWithReceipt(ctx, expectedRevision, patch, s.validate, binding)
	return receipt, err
}

func (s Service) FindMutationReceipt(ctx context.Context, owner, permission, requestID string) (operatorstate.MutationReceipt, bool, error) {
	if s.store == nil {
		return operatorstate.MutationReceipt{}, false, errors.New("operator state store is not configured")
	}
	return s.store.FindMutationReceipt(ctx, owner, permission, requestID)
}

func (s Service) PermissionRegistry() (*permissions.Registry, error) {
	providers := make([]permissions.Provider, 0, len(s.providers)+1)
	providers = append(providers, permissionProvider{service: s})
	providers = append(providers, s.providers...)
	return permissions.NewRegistry(providers...)
}

func (permissionProvider) ProviderID() string                 { return "onboarding.operator-state" }
func (permissionProvider) Owner() string                      { return "vrooli-onboarding" }
func (permissionProvider) MatchesPermissionID(id string) bool { return id == permissionID }

func (p permissionProvider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || principal.Subject == "" {
		return nil, errors.New("verified onboarding principal required")
	}
	return []permissions.Descriptor{{
		Version: permissions.ContractVersion, ID: permissionID, Owner: p.Owner(),
		PrincipalKind: string(principal.Kind), PrincipalID: principal.Subject,
		Scope: "vrooli-onboarding:write", SubjectKind: "operator-state-document", SubjectID: "local",
		Source: "internal/operatorstate + operatorstateapi", SchemaVersion: "operator-state/v1",
		CanRead: true, CanWrite: true, Sensitivity: "operator",
	}}, nil
}

func (p permissionProvider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || principal.Subject != req.ActorID || d.PrincipalID != req.ActorID {
		return permissions.Decision{}, errors.New("verified actor does not match operator-state read")
	}
	return permissions.Decision{Allowed: true, DecisionRef: "identity:" + principal.Subject}, nil
}

func (p permissionProvider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, errors.New("operator-state read permit mismatch")
	}
	doc, err := p.service.Get(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	// Enrollment session mode is owned by api-core/operatorsession and is
	// exposed through its field permission. Do not leak a legacy copy retained
	// in operator-state's unknown-field map through the generic document view.
	delete(doc.RawFields, "session_mode")
	data, err := MarshalDocument(doc)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	revision := operatorstate.Revision(doc)
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Revision: revision, Value: data, ObservedAt: time.Now().UTC()}, nil
}

func (p permissionProvider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || principal.Subject != req.ActorID || d.PrincipalID != req.ActorID {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("verified actor does not match operator-state write"))
	}
	if err := authz.RequireMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "authz:" + principal.Subject + ":vrooli-onboarding:write"}, nil
}

func (p permissionProvider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, errors.New("operator-state write permit mismatch")
	}
	var patch PermissionPatch
	if err := json.Unmarshal(req.Value, &patch); err != nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode operator-state permission patch: %w", err))
	}
	if len(patch.State) == 0 || len(patch.UpdateMask) == 0 {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("operator-state permission patch requires state and update_mask"))
	}
	for _, path := range patch.UpdateMask {
		parts := strings.Split(strings.TrimSpace(path), ".")
		if len(parts) > 0 {
			switch snakeCase(strings.TrimSpace(parts[0])) {
			case "host_safeguards":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("host safeguards must be changed through their typed permission"))
			case "host_tools":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("host tool opt-ins must be changed through their typed permission"))
			case "trust_posture":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("trust posture is owner-managed and cannot be changed through onboarding permissions"))
			case "update_control":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("update control must be changed through its typed permission"))
			case "session_mode":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("session mode is enrollment-owned and cannot be changed through generic operator-state permissions"))
			case "core":
				return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("core authority must be changed through its typed permission"))
			}
		}
	}
	requestDigest := digestPermissionRequest(req.Value)
	stored, err := p.service.PatchWithReceipt(ctx, patch.State, patch.UpdateMask, req.ExpectedRevision, operatorstate.MutationReceiptBinding{Owner: d.Owner, PermissionID: d.ID, ActorID: req.ActorID, Target: "local", Subject: d.SubjectKind + ":" + d.SubjectID, RequestID: req.IdempotencyKey, ExpectedRevision: req.ExpectedRevision, RequestDigest: requestDigest, DecisionRef: permit.DecisionRef()})
	if err != nil {
		return permissions.Receipt{}, err
	}
	return permissions.Receipt{ID: stored.ID, PermissionID: stored.PermissionID, Owner: stored.Owner, ActorID: stored.ActorID, Target: req.Target, ExpectedRevision: stored.ExpectedRevision, ResultRevision: stored.ResultRevision, IdempotencyKey: stored.RequestID, RequestDigest: stored.RequestDigest, DecisionRef: stored.DecisionRef, Outcome: stored.Outcome, At: stored.At}, nil
}

func digestPermissionRequest(value []byte) string {
	digest := sha256.Sum256(value)
	return fmt.Sprintf("%x", digest)
}

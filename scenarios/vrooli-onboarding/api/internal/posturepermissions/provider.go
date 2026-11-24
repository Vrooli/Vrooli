// Package posturepermissions exposes the per-field install posture contracts
// through onboarding's common permission registry. Operator state remains the
// owner for update_control; api-core/operatorsession remains the owner for
// enrollment metadata.
package posturepermissions

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	updateControlID = "onboarding/install-posture/update-control"
	sessionModeID   = "onboarding/install-posture/session-mode"
	trustPostureID  = "onboarding/install-posture/trust-posture"
)

type Provider struct {
	kind       string
	state      operatorstateapi.Service
	enrollment operatorsession.EnrollmentMetadataReader
}

func NewUpdateControl(state operatorstateapi.Service) *Provider {
	return &Provider{kind: updateControlID, state: state}
}

func NewSessionMode(enrollment operatorsession.EnrollmentMetadataReader) *Provider {
	return &Provider{kind: sessionModeID, enrollment: enrollment}
}

func NewTrustPosture(state operatorstateapi.Service) *Provider {
	return &Provider{kind: trustPostureID, state: state}
}

func (p *Provider) ProviderID() string { return "onboarding.install-posture." + p.kind }
func (p *Provider) Owner() string {
	if p.kind == sessionModeID {
		return "api-core/operatorsession"
	}
	return "vrooli-onboarding"
}

func (p *Provider) MatchesPermissionID(id string) bool {
	return id == updateControlID || id == sessionModeID || id == trustPostureID
}

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || strings.TrimSpace(principal.Subject) == "" {
		return nil, errors.New("verified onboarding principal required")
	}
	if p.kind == sessionModeID {
		if p.enrollment == nil {
			return nil, nil
		}
		enrollment, err := p.enrollment.ReadEnrollment()
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read operator enrollment metadata: %w", err)
		}
		if enrollment.OperatorID != principal.Subject {
			return nil, nil
		}
	}
	descriptor := permissions.Descriptor{
		Version: permissions.ContractVersion, Owner: p.Owner(), PrincipalKind: string(principal.Kind), PrincipalID: principal.Subject,
		CanRead: true, Sensitivity: "operator",
	}
	if p.kind == updateControlID {
		descriptor.ID = updateControlID
		descriptor.Scope = "operator-state:update-control"
		descriptor.SubjectKind = "operator-state-field"
		descriptor.SubjectID = "update_control"
		descriptor.Source = "internal/operatorstate"
		descriptor.SchemaVersion = "operator-state/update-control/v1"
		descriptor.CanWrite = true
	} else if p.kind == sessionModeID {
		descriptor.ID = sessionModeID
		descriptor.Scope = "operator-session:read"
		descriptor.SubjectKind = "operator-session-field"
		descriptor.SubjectID = "session_mode"
		descriptor.Source = "packages/api-core/operatorsession.EnrollmentMetadataReader"
		descriptor.SchemaVersion = "operator-session/enrollment/v1"
	} else if p.kind == trustPostureID {
		descriptor.ID = trustPostureID
		descriptor.Scope = "operator-state:trust-posture:read"
		descriptor.SubjectKind = "operator-state-field"
		descriptor.SubjectID = "trust_posture"
		descriptor.Source = "internal/operatorstate + packages/api-core/trustposture.Parse"
		descriptor.SchemaVersion = "operator-state/trust-posture/v1"
	} else {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("unknown posture provider"))
	}
	return []permissions.Descriptor{descriptor}, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || principal.Subject != req.ActorID || d.PrincipalID != req.ActorID {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("verified actor does not match posture read"))
	}
	return permissions.Decision{Allowed: true, DecisionRef: "identity:" + principal.Subject}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, errors.New("posture read permit mismatch")
	}
	var value any
	var revision string
	switch d.ID {
	case updateControlID:
		doc, err := p.state.Get(ctx)
		if err != nil {
			return permissions.ReadResult{}, err
		}
		value = struct {
			Value     string   `json:"value"`
			Supported []string `json:"supported_values"`
		}{Value: doc.EffectiveUpdateControl(), Supported: []string{"observe", "guard"}}
		revision = operatorstate.Revision(doc)
	case sessionModeID:
		if p.enrollment == nil {
			return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("operator enrollment metadata is unavailable"))
		}
		enrollment, err := p.enrollment.ReadEnrollment()
		if err != nil {
			return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read operator enrollment metadata: %w", err))
		}
		if enrollment.OperatorID != req.ActorID {
			return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("enrollment owner does not match verified actor"))
		}
		value = struct {
			Value     operatorsession.Mode   `json:"value"`
			Supported []operatorsession.Mode `json:"supported_values"`
		}{Value: enrollment.Mode, Supported: []operatorsession.Mode{operatorsession.ModePersonal, operatorsession.ModeShared, operatorsession.ModeHosted}}
		encoded, _ := json.Marshal(enrollment)
		digest := sha256.Sum256(encoded)
		revision = fmt.Sprintf("sha256:%x", digest[:])
	case trustPostureID:
		doc, err := p.state.Get(ctx)
		if err != nil {
			return permissions.ReadResult{}, err
		}
		posture := trustposture.Posture(doc.TrustPosture)
		if posture == "" {
			posture = trustposture.Personal
		}
		if _, err := trustposture.DefaultsFor(posture); err != nil {
			return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("operator trust posture is invalid: %w", err))
		}
		value = struct {
			Value     trustposture.Posture   `json:"value"`
			Supported []trustposture.Posture `json:"supported_values"`
		}{Value: posture, Supported: []trustposture.Posture{trustposture.Personal, trustposture.Shared, trustposture.Hosted}}
		revision = operatorstate.Revision(doc)
	default:
		return permissions.ReadResult{}, connect.NewError(connect.CodeNotFound, errors.New("unknown install posture field"))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Revision: revision, Value: data, ObservedAt: time.Now().UTC()}, nil
}

func (p *Provider) Authorize(ctx context.Context, d permissions.Descriptor, req permissions.WriteRequest) (permissions.Decision, error) {
	if d.ID != updateControlID {
		return permissions.Decision{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("session mode and trust posture are owner-managed and read-only through onboarding"))
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || principal.Subject != req.ActorID || d.PrincipalID != req.ActorID {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("verified actor does not match update-control write"))
	}
	if err := authz.RequireMutation(ctx); err != nil {
		return permissions.Decision{}, err
	}
	return permissions.Decision{Allowed: true, DecisionRef: "authz:" + principal.Subject + ":operator-state:update-control"}, nil
}

func (p *Provider) Write(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.WriteRequest) (permissions.Receipt, error) {
	if !permit.ValidForWrite(d.ID, req) {
		return permissions.Receipt{}, errors.New("update-control write permit mismatch")
	}
	if d.ID != updateControlID {
		return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("session mode and trust posture are owner-managed and read-only through onboarding"))
	}
	var patch struct {
		Value string `json:"value"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(req.Value)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patch); err != nil {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("decode update-control value: %w", err))
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return permissions.Receipt{}, connect.NewError(connect.CodeInvalidArgument, errors.New("update-control value must contain exactly one JSON object"))
	}
	if patch.Value != "observe" && patch.Value != "guard" {
		return permissions.Receipt{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("update_control supports only observe or guard; own is reserved"))
	}
	stateJSON, err := json.Marshal(map[string]string{"update_control": patch.Value})
	if err != nil {
		return permissions.Receipt{}, err
	}
	stored, err := p.state.PatchWithReceipt(ctx, stateJSON, []string{"update_control"}, req.ExpectedRevision, operatorstate.MutationReceiptBinding{
		Owner: d.Owner, PermissionID: d.ID, ActorID: req.ActorID, Target: "local", Subject: d.SubjectKind + ":" + d.SubjectID,
		RequestID: req.IdempotencyKey, ExpectedRevision: req.ExpectedRevision,
		RequestDigest: fmt.Sprintf("%x", sha256.Sum256(req.Value)), DecisionRef: permit.DecisionRef(),
	})
	if err != nil {
		return permissions.Receipt{}, err
	}
	return permissions.Receipt{ID: stored.ID, PermissionID: stored.PermissionID, Owner: stored.Owner, ActorID: stored.ActorID, Target: req.Target,
		ExpectedRevision: stored.ExpectedRevision, ResultRevision: stored.ResultRevision, IdempotencyKey: stored.RequestID,
		RequestDigest: stored.RequestDigest, DecisionRef: stored.DecisionRef, Outcome: stored.Outcome, At: stored.At}, nil
}

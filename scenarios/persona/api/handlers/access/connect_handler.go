package access

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "persona/internal/access"
	"persona/internal/personas"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ActAs(ctx context.Context, req *connect.Request[accessv1.ActAsRequest]) (*connect.Response[accessv1.ActAsResponse], error) {
	session, err := h.service.ActAs(ctx, req.Msg.GetPersonaId(), req.Header().Get(cliutil.HeaderAgentIdentityToken), req.Msg.GetAction())
	if err != nil {
		return nil, accessError(err)
	}
	return connect.NewResponse(&accessv1.ActAsResponse{PersonaId: session.PersonaID, RunId: session.RunID, AccountSubject: session.AccountSubject, AuthorisingHuman: session.AuthorisingHuman, GrantedAt: timestamppb.New(session.GrantedAt)}), nil
}

func (h *connectHandler) ResolvePersona(ctx context.Context, req *connect.Request[accessv1.ResolvePersonaRequest]) (*connect.Response[accessv1.ResolvePersonaResponse], error) {
	resolution, err := h.service.ResolvePersona(ctx, req.Msg.GetPersonaId(), req.Header().Get(cliutil.HeaderAgentIdentityToken), req.Msg.GetFields())
	if err != nil {
		return nil, accessError(err)
	}
	return connect.NewResponse(&accessv1.ResolvePersonaResponse{Persona: &accessv1.PersonaResolution{PersonaId: resolution.PersonaID, Kind: toPersonaKind(resolution.Kind), DisplayName: resolution.DisplayName, LegalSubjectId: resolution.LegalSubjectID, ControlledEmail: resolution.ControlledEmail, AddressIds: resolution.AddressIDs, ReturnedFields: resolution.ReturnedFields}}), nil
}

func (h *connectHandler) CreateGrant(ctx context.Context, req *connect.Request[accessv1.CreateGrantRequest]) (*connect.Response[accessv1.CreateGrantResponse], error) {
	grant, err := h.service.CreateGrant(ctx, domain.GrantInput{PersonaID: req.Msg.GetPersonaId(), HumanSubject: req.Msg.GetHumanSubject(), Level: fromGrantLevel(req.Msg.GetLevel()), Source: req.Msg.GetSource()})
	if err != nil {
		return nil, accessError(err)
	}
	return connect.NewResponse(&accessv1.CreateGrantResponse{Grant: toProtoGrant(grant)}), nil
}

func (h *connectHandler) ListGrants(ctx context.Context, req *connect.Request[accessv1.ListGrantsRequest]) (*connect.Response[accessv1.ListGrantsResponse], error) {
	grants, err := h.service.ListGrants(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, accessError(err)
	}
	out := &accessv1.ListGrantsResponse{Grants: make([]*accessv1.PersonaGrant, 0, len(grants))}
	for _, grant := range grants {
		out.Grants = append(out.Grants, toProtoGrant(grant))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RemoveGrant(ctx context.Context, req *connect.Request[accessv1.RemoveGrantRequest]) (*connect.Response[accessv1.RemoveGrantResponse], error) {
	if err := h.service.RemoveGrant(ctx, req.Msg.GetGrantId()); err != nil {
		return nil, accessError(err)
	}
	return connect.NewResponse(&accessv1.RemoveGrantResponse{}), nil
}

func (h *connectHandler) IssueAttestation(ctx context.Context, req *connect.Request[accessv1.IssueAttestationRequest]) (*connect.Response[accessv1.IssueAttestationResponse], error) {
	requested := time.Time{}
	if req.Msg.GetExpiresAtUnix() != 0 {
		requested = time.Unix(req.Msg.GetExpiresAtUnix(), 0).UTC()
	}
	attestation, err := h.service.IssueAttestation(ctx, req.Msg.GetPersonaId(), req.Header().Get(cliutil.HeaderAgentIdentityToken), req.Msg.GetAudience(), requested)
	if err != nil {
		return nil, accessError(err)
	}
	return connect.NewResponse(&accessv1.IssueAttestationResponse{Attestation: &accessv1.IdentityAttestation{Issuer: attestation.Issuer, Audience: attestation.Audience, LegalPerson: attestation.LegalPerson, PersonaId: attestation.PersonaID, AccountSubject: attestation.AccountSubject, RunId: attestation.RunID, IssuedAtUnix: attestation.IssuedAt.Unix(), ExpiresAtUnix: attestation.ExpiresAt.Unix(), ClaimFormat: attestation.ClaimFormat, Signature: attestation.Signature, KeyId: attestation.KeyID}}), nil
}

func accessError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, domain.ErrMissingPersona), errors.Is(err, domain.ErrInvalidIdentity), errors.Is(err, domain.ErrPersonaBinding), errors.Is(err, domain.ErrScopeMissing), errors.Is(err, domain.ErrProposeOnly), errors.Is(err, domain.ErrAttestationExpiry):
		code = connect.CodePermissionDenied
	case errors.Is(err, domain.ErrAttestationSigner):
		code = connect.CodeFailedPrecondition
	case errors.Is(err, domain.ErrGrantMissing):
		code = connect.CodePermissionDenied
	case errors.Is(err, domain.ErrAuthorityUnreachable):
		code = connect.CodeUnavailable
	}
	if errors.Is(err, domain.ErrMissingPersona) {
		code = connect.CodeInvalidArgument
	}
	return connect.NewError(code, err)
}

func toPersonaKind(kind personas.Kind) accessv1.ResolvedPersonaKind {
	if kind == personas.KindPersonal {
		return accessv1.ResolvedPersonaKind_RESOLVED_PERSONA_KIND_PERSONAL
	}
	if kind == personas.KindBusiness {
		return accessv1.ResolvedPersonaKind_RESOLVED_PERSONA_KIND_BUSINESS
	}
	return accessv1.ResolvedPersonaKind_RESOLVED_PERSONA_KIND_UNSPECIFIED
}

func fromGrantLevel(level accessv1.GrantLevel) domain.GrantLevel {
	if level == accessv1.GrantLevel_GRANT_LEVEL_ACT {
		return domain.GrantAct
	}
	if level == accessv1.GrantLevel_GRANT_LEVEL_PROPOSE {
		return domain.GrantPropose
	}
	return ""
}

func toProtoGrant(grant domain.Grant) *accessv1.PersonaGrant {
	return &accessv1.PersonaGrant{Id: grant.ID, PersonaId: grant.PersonaID, HumanSubject: grant.HumanSubject, Level: toGrantLevel(grant.Level), Source: grant.Source, CreatedAt: timestamppb.New(grant.CreatedAt), UpdatedAt: timestamppb.New(grant.UpdatedAt)}
}

func toGrantLevel(level domain.GrantLevel) accessv1.GrantLevel {
	if level == domain.GrantAct {
		return accessv1.GrantLevel_GRANT_LEVEL_ACT
	}
	return accessv1.GrantLevel_GRANT_LEVEL_PROPOSE
}

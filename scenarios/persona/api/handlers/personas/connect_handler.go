package personas

import (
	"context"
	"errors"

	domain "persona/internal/personas"

	"connectrpc.com/connect"
	personasv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/personas"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) CreatePersona(ctx context.Context, req *connect.Request[personasv1.CreatePersonaRequest]) (*connect.Response[personasv1.CreatePersonaResponse], error) {
	p, err := h.service.Create(ctx, domain.CreateInput{Kind: fromKind(req.Msg.GetKind()), LegalBasis: domain.LegalBasis{SubjectID: req.Msg.GetLegalBasis().GetSubjectId(), SubjectName: req.Msg.GetLegalBasis().GetSubjectName(), BasisType: req.Msg.GetLegalBasis().GetBasisType()}, DisplayName: req.Msg.GetDisplayName(), Identifiers: fromIdentifiers(req.Msg.GetIdentifiers())})
	if err != nil {
		return nil, personaError(err)
	}
	return connect.NewResponse(&personasv1.CreatePersonaResponse{Persona: toProto(p)}), nil
}

func (h *connectHandler) GetPersona(ctx context.Context, req *connect.Request[personasv1.GetPersonaRequest]) (*connect.Response[personasv1.GetPersonaResponse], error) {
	p, err := h.service.Get(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, personaError(err)
	}
	return connect.NewResponse(&personasv1.GetPersonaResponse{Persona: toProto(p)}), nil
}

func (h *connectHandler) ListPersonas(ctx context.Context, req *connect.Request[personasv1.ListPersonasRequest]) (*connect.Response[personasv1.ListPersonasResponse], error) {
	items, err := h.service.List(ctx, req.Msg.GetIncludeArchived(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, personaError(err)
	}
	result := &personasv1.ListPersonasResponse{Personas: make([]*personasv1.Persona, 0, len(items))}
	for _, p := range items {
		result.Personas = append(result.Personas, toProto(p))
	}
	return connect.NewResponse(result), nil
}

func (h *connectHandler) ArchivePersona(ctx context.Context, req *connect.Request[personasv1.ArchivePersonaRequest]) (*connect.Response[personasv1.ArchivePersonaResponse], error) {
	p, err := h.service.Archive(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, personaError(err)
	}
	return connect.NewResponse(&personasv1.ArchivePersonaResponse{Persona: toProto(p)}), nil
}

func (h *connectHandler) CheckHealth(ctx context.Context, req *connect.Request[personasv1.CheckHealthRequest]) (*connect.Response[personasv1.CheckHealthResponse], error) {
	findings, err := h.service.CheckHealth(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, personaError(err)
	}
	result := make([]*personasv1.HealthFinding, 0, len(findings))
	for _, finding := range findings {
		result = append(result, &personasv1.HealthFinding{Code: finding.Code, Message: finding.Message, Blocking: finding.Blocking})
	}
	return connect.NewResponse(&personasv1.CheckHealthResponse{Findings: result}), nil
}

func personaError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, domain.ErrMissingID) || errors.Is(err, domain.ErrMissingLegal) || errors.Is(err, domain.ErrInvalidKind) || errors.Is(err, domain.ErrMissingIdentity) || errors.Is(err, domain.ErrInvalidIdentity) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, domain.ErrNotFound) {
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}

func fromKind(k personasv1.PersonaKind) domain.Kind {
	if k == personasv1.PersonaKind_PERSONA_KIND_PERSONAL {
		return domain.KindPersonal
	}
	if k == personasv1.PersonaKind_PERSONA_KIND_BUSINESS {
		return domain.KindBusiness
	}
	return ""
}

func toKind(k domain.Kind) personasv1.PersonaKind {
	if k == domain.KindPersonal {
		return personasv1.PersonaKind_PERSONA_KIND_PERSONAL
	}
	return personasv1.PersonaKind_PERSONA_KIND_BUSINESS
}

func fromIdentifiers(in []*personasv1.Identifier) []domain.Identifier {
	out := make([]domain.Identifier, 0, len(in))
	for _, item := range in {
		if item != nil {
			out = append(out, domain.Identifier{Type: item.GetType(), Value: item.GetValue()})
		}
	}
	return out
}

func toProto(p domain.Persona) *personasv1.Persona {
	identifiers := make([]*personasv1.Identifier, 0, len(p.Identifiers))
	for _, item := range p.Identifiers {
		identifiers = append(identifiers, &personasv1.Identifier{Type: item.Type, Value: item.Value})
	}
	out := &personasv1.Persona{Id: p.ID, Kind: toKind(p.Kind), LegalBasis: &personasv1.LegalBasis{SubjectId: p.LegalBasis.SubjectID, SubjectName: p.LegalBasis.SubjectName, BasisType: p.LegalBasis.BasisType}, DisplayName: p.DisplayName, Identifiers: identifiers, Status: toStatus(p.Status), CreatedAt: timestamppb.New(p.CreatedAt)}
	if p.ArchivedAt != nil {
		out.ArchivedAt = timestamppb.New(*p.ArchivedAt)
	}
	return out
}

func toStatus(s domain.Status) personasv1.PersonaStatus {
	if s == domain.StatusArchived {
		return personasv1.PersonaStatus_PERSONA_STATUS_ARCHIVED
	}
	return personasv1.PersonaStatus_PERSONA_STATUS_ACTIVE
}

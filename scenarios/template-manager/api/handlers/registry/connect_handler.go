package registry

import (
	"context"
	"errors"
	"log"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	"connectrpc.com/connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Repository catalog.Repository
	Logger     *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListTemplates(ctx context.Context, req *connect.Request[registryv1.ListTemplatesRequest]) (*connect.Response[registryv1.ListTemplatesResponse], error) {
	records, err := h.deps.Repository.ListTemplates(ctx, protoKindToDomain(req.Msg.Kind))
	if err != nil {
		h.deps.Logger.Printf("registry.ListTemplates: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("list templates"))
	}
	resp := &registryv1.ListTemplatesResponse{Templates: make([]*registryv1.TemplateRecord, 0, len(records))}
	for _, record := range records {
		resp.Templates = append(resp.Templates, templateToProto(record))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetTemplate(ctx context.Context, req *connect.Request[registryv1.GetTemplateRequest]) (*connect.Response[registryv1.GetTemplateResponse], error) {
	record, err := h.deps.Repository.GetTemplate(ctx, req.Msg.Id)
	if err != nil {
		var notFound catalog.ErrNotFound
		if errors.As(err, &notFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("registry.GetTemplate(%q): %v", req.Msg.Id, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("get template"))
	}
	return connect.NewResponse(&registryv1.GetTemplateResponse{Template: templateToProto(record)}), nil
}

func protoKindToDomain(kind registryv1.TemplateKind) catalog.TemplateKind {
	switch kind {
	case registryv1.TemplateKind_TEMPLATE_KIND_SCENARIO:
		return catalog.KindScenario
	case registryv1.TemplateKind_TEMPLATE_KIND_DESIGN:
		return catalog.KindDesign
	case registryv1.TemplateKind_TEMPLATE_KIND_RESOURCE:
		return catalog.KindResource
	default:
		return ""
	}
}

func domainKindToProto(kind catalog.TemplateKind) registryv1.TemplateKind {
	switch kind {
	case catalog.KindScenario:
		return registryv1.TemplateKind_TEMPLATE_KIND_SCENARIO
	case catalog.KindDesign:
		return registryv1.TemplateKind_TEMPLATE_KIND_DESIGN
	case catalog.KindResource:
		return registryv1.TemplateKind_TEMPLATE_KIND_RESOURCE
	default:
		return registryv1.TemplateKind_TEMPLATE_KIND_UNSPECIFIED
	}
}

func templateToProto(record catalog.TemplateRecord) *registryv1.TemplateRecord {
	return &registryv1.TemplateRecord{
		Id:           record.ID,
		Kind:         domainKindToProto(record.Kind),
		DisplayName:  record.DisplayName,
		Version:      record.Version,
		ManifestPath: record.ManifestPath,
		SourcePath:   record.SourcePath,
		Tags:         append([]string(nil), record.Tags...),
		Status:       record.Status,
		UpdatedAt:    timestamppb.New(record.UpdatedAt),
		VersionLag: &registryv1.TemplateVersionLag{
			CurrentVersion: record.CurrentVersion,
			LatestVersion:  record.LatestVersion,
			LagCount:       record.LagCount,
		},
	}
}

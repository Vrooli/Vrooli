package content

import (
	"context"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalcontent "landing-page-react-vite-api/internal/content"
)

// Deps wires the content Connect handler.
type Deps struct {
	Service *internalcontent.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the ContentService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetPublicSections(ctx context.Context, req *connect.Request[landingv1.GetPublicSectionsRequest]) (*connect.Response[landingv1.SectionsResponse], error) {
	sections, err := h.deps.Service.GetPublicSections(req.Msg.VariantId)
	if err != nil {
		h.deps.Logger.Printf("content.GetPublicSections: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.SectionsResponse{Sections: sectionsToProto(sections)}), nil
}

func (h *connectHandler) GetSections(ctx context.Context, req *connect.Request[landingv1.GetSectionsRequest]) (*connect.Response[landingv1.SectionsResponse], error) {
	sections, err := h.deps.Service.GetSections(req.Msg.VariantId)
	if err != nil {
		h.deps.Logger.Printf("content.GetSections: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.SectionsResponse{Sections: sectionsToProto(sections)}), nil
}

func (h *connectHandler) GetSection(ctx context.Context, req *connect.Request[landingv1.GetSectionRequest]) (*connect.Response[landingv1.SectionResponse], error) {
	section, err := h.deps.Service.GetSection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&landingv1.SectionResponse{Section: sectionToProto(section)}), nil
}

func (h *connectHandler) CreateSection(ctx context.Context, req *connect.Request[landingv1.CreateSectionRequest]) (*connect.Response[landingv1.SectionResponse], error) {
	m := req.Msg
	enabled := true
	if m.Enabled != nil {
		enabled = *m.Enabled
	}
	created, err := h.deps.Service.CreateSection(internalcontent.Section{
		VariantID:   m.VariantId,
		SectionType: m.SectionType,
		Content:     structToMap(m.Content),
		Order:       int(m.Order),
		Enabled:     enabled,
	})
	if err != nil {
		h.deps.Logger.Printf("content.CreateSection: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.SectionResponse{Section: sectionToProto(created)}), nil
}

func (h *connectHandler) UpdateSection(ctx context.Context, req *connect.Request[landingv1.UpdateSectionRequest]) (*connect.Response[landingv1.SectionResponse], error) {
	if err := h.deps.Service.UpdateSection(req.Msg.Id, structToMap(req.Msg.Content)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	section, err := h.deps.Service.GetSection(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.SectionResponse{Section: sectionToProto(section)}), nil
}

func (h *connectHandler) DeleteSection(ctx context.Context, req *connect.Request[landingv1.DeleteSectionRequest]) (*connect.Response[landingv1.DeleteSectionResponse], error) {
	if err := h.deps.Service.DeleteSection(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&landingv1.DeleteSectionResponse{Deleted: true}), nil
}

func sectionsToProto(sections []internalcontent.Section) []*landingv1.ContentSection {
	out := make([]*landingv1.ContentSection, 0, len(sections))
	for i := range sections {
		out = append(out, sectionToProto(&sections[i]))
	}
	return out
}

func sectionToProto(s *internalcontent.Section) *landingv1.ContentSection {
	return &landingv1.ContentSection{
		Id:          s.ID,
		VariantId:   s.VariantID,
		SectionType: s.SectionType,
		Content:     mapToStruct(s.Content),
		Order:       int32(s.Order),
		Enabled:     s.Enabled,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}

func mapToStruct(m map[string]interface{}) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

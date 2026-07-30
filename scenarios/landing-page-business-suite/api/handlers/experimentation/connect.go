package variant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	"landing-page-business-suite-api/internal/contracts"
	"landing-page-business-suite-api/internal/experimentation"
)

// variantConnectHandler is the single generated-contract transport for the
// JSON-backed experiment configuration. It deliberately uses ConfigStorer so
// no second persistence model can diverge from the public landing experience.
type variantConnectHandler struct{ store experimentation.ConfigStorer }

func newVariantConnectHandler(store experimentation.ConfigStorer) *variantConnectHandler {
	return &variantConnectHandler{store: store}
}

func (h *variantConnectHandler) SelectVariant(_ context.Context, _ *connect.Request[lpbsv1.SelectVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	active := activeVariantSnapshots(h.store.ListVariants())
	if len(active) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no active variants available"))
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(experimentation.SelectWeightedRandomVariant(active), false)}), nil
}

func (h *variantConnectHandler) GetPublicVariant(_ context.Context, request *connect.Request[lpbsv1.GetPublicVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	if experimentation.NormalizeVariantStatus(snapshot.Variant.Status) != "active" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("variant is not publicly available"))
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(snapshot, false)}), nil
}

func (h *variantConnectHandler) GetVariant(_ context.Context, request *connect.Request[lpbsv1.GetVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(snapshot, true)}), nil
}

func (h *variantConnectHandler) ListVariants(_ context.Context, request *connect.Request[lpbsv1.ListVariantsRequest]) (*connect.Response[lpbsv1.ListVariantsResponse], error) {
	filter := strings.TrimSpace(request.Msg.GetStatusFilter())
	if filter != "" && filter != "active" && filter != "archived" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("status_filter must be active or archived"))
	}
	result := &lpbsv1.ListVariantsResponse{}
	for _, snapshot := range h.store.ListVariants() {
		if filter == "" || experimentation.NormalizeVariantStatus(snapshot.Variant.Status) == filter {
			result.Variants = append(result.Variants, variantProto(snapshot, false))
		}
	}
	return connect.NewResponse(result), nil
}

func (h *variantConnectHandler) CreateVariant(_ context.Context, request *connect.Request[lpbsv1.CreateVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	m := request.Msg
	slug, name := strings.TrimSpace(m.GetSlug()), strings.TrimSpace(m.GetName())
	if slug == "" || name == "" || len(m.GetAxes()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("slug, name, and axes are required"))
	}
	if _, err := h.store.GetVariant(slug); err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("variant %q already exists", slug))
	}
	sections := cloneControlSections(h.store)
	snapshot := &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: slug, Name: name, Description: m.GetDescription(), Weight: int(m.GetWeight()),
		Status: "active", Axes: cloneAxes(m.GetAxes()), HeaderConfig: experimentation.DefaultLandingHeaderConfig(name),
	}, Sections: sections}
	if err := h.store.SaveVariant(slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("save variant: %w", err))
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(snapshot, false)}), nil
}

func (h *variantConnectHandler) UpdateVariant(_ context.Context, request *connect.Request[lpbsv1.UpdateVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	m := request.Msg
	if m.Name != nil {
		snapshot.Variant.Name = strings.TrimSpace(*m.Name)
	}
	if m.Description != nil {
		snapshot.Variant.Description = *m.Description
	}
	if m.Weight != nil {
		snapshot.Variant.Weight = int(*m.Weight)
	}
	if m.Axes != nil {
		snapshot.Variant.Axes = cloneAxes(m.Axes.Values)
	}
	if m.HeaderConfig != nil {
		snapshot.Variant.HeaderConfig = experimentation.NormalizeLandingHeaderConfig(headerFromProto(m.HeaderConfig), snapshot.Variant.Name)
	}
	if m.SeoConfig != nil {
		snapshot.Variant.SEOConfig = seoProtoToRaw(m.SeoConfig)
	}
	if strings.TrimSpace(snapshot.Variant.Name) == "" || len(snapshot.Variant.Axes) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name and axes are required"))
	}
	if err := h.store.SaveVariant(snapshot.Variant.Slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("save variant: %w", err))
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(snapshot, false)}), nil
}

func (h *variantConnectHandler) ArchiveVariant(_ context.Context, request *connect.Request[lpbsv1.ArchiveVariantRequest]) (*connect.Response[lpbsv1.VariantResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	snapshot.Variant.Status = "archived"
	if err := h.store.SaveVariant(snapshot.Variant.Slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("archive variant: %w", err))
	}
	return connect.NewResponse(&lpbsv1.VariantResponse{Variant: variantProto(snapshot, false)}), nil
}

func (h *variantConnectHandler) DeleteVariant(_ context.Context, request *connect.Request[lpbsv1.DeleteVariantRequest]) (*connect.Response[lpbsv1.DeleteVariantResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	if err := h.store.DeleteVariant(snapshot.Variant.Slug); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete variant: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteVariantResponse{Deleted: true}), nil
}

func (h *variantConnectHandler) ExportVariantSnapshot(_ context.Context, request *connect.Request[lpbsv1.ExportVariantSnapshotRequest]) (*connect.Response[lpbsv1.ExportVariantSnapshotResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.ExportVariantSnapshotResponse{Snapshot: snapshotProto(snapshot)}), nil
}

func (h *variantConnectHandler) ImportVariantSnapshot(_ context.Context, request *connect.Request[lpbsv1.ImportVariantSnapshotRequest]) (*connect.Response[lpbsv1.ImportVariantSnapshotResponse], error) {
	slug := strings.TrimSpace(request.Msg.GetSlug())
	if slug == "" || request.Msg.GetSnapshot() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("slug and snapshot are required"))
	}
	snapshot, err := snapshotFromProto(request.Msg.GetSnapshot())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if snapshot.Variant.Slug != slug {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot slug must match request slug"))
	}
	if err := h.store.SaveVariant(slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("save snapshot: %w", err))
	}
	return connect.NewResponse(&lpbsv1.ImportVariantSnapshotResponse{Snapshot: snapshotProto(snapshot)}), nil
}

func (h *variantConnectHandler) SyncVariantSnapshots(_ context.Context, _ *connect.Request[lpbsv1.SyncVariantSnapshotsRequest]) (*connect.Response[lpbsv1.SyncVariantSnapshotsResponse], error) {
	if err := h.store.LoadAll(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("reload variant snapshots: %w", err))
	}
	return connect.NewResponse(&lpbsv1.SyncVariantSnapshotsResponse{Count: boundedProtoInt32(h.store.VariantCount())}), nil
}

func (h *variantConnectHandler) GetVariantSections(_ context.Context, request *connect.Request[lpbsv1.GetVariantSectionsRequest]) (*connect.Response[lpbsv1.VariantSectionsResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	result := &lpbsv1.VariantSectionsResponse{Sections: make([]*sharedv1.ContentSection, 0, len(snapshot.Sections))}
	for _, section := range snapshot.Sections {
		result.Sections = append(result.Sections, sectionProto(section))
	}
	return connect.NewResponse(result), nil
}

func (h *variantConnectHandler) GetVariantSection(_ context.Context, request *connect.Request[lpbsv1.GetVariantSectionRequest]) (*connect.Response[lpbsv1.VariantSectionResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	section, _, err := variantSection(snapshot, request.Msg.GetSectionKey())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&lpbsv1.VariantSectionResponse{Section: sectionProto(*section)}), nil
}

func (h *variantConnectHandler) CreateVariantSection(_ context.Context, request *connect.Request[lpbsv1.CreateVariantSectionRequest]) (*connect.Response[lpbsv1.VariantSectionResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	section, err := sectionFromProto(request.Msg.GetSection(), len(snapshot.Sections))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	snapshot.Sections = append(snapshot.Sections, section)
	if err := h.store.SaveVariant(snapshot.Variant.Slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create variant section: %w", err))
	}
	return connect.NewResponse(&lpbsv1.VariantSectionResponse{Section: sectionProto(snapshot.Sections[len(snapshot.Sections)-1])}), nil
}

func (h *variantConnectHandler) UpdateVariantSection(_ context.Context, request *connect.Request[lpbsv1.UpdateVariantSectionRequest]) (*connect.Response[lpbsv1.VariantSectionResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	section, _, err := variantSection(snapshot, request.Msg.GetSectionKey())
	if err != nil {
		return nil, err
	}
	if request.Msg.SectionType != nil {
		section.SectionType = strings.TrimSpace(request.Msg.GetSectionType())
	}
	if section.SectionType == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("section_type is required"))
	}
	if request.Msg.Content != nil {
		content, err := json.Marshal(request.Msg.GetContent().AsMap())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode section content: %w", err))
		}
		section.Content = content
	}
	if request.Msg.Order != nil {
		section.Order = int(request.Msg.GetOrder())
	}
	if request.Msg.Enabled != nil {
		section.Enabled = request.Msg.GetEnabled()
	}
	if err := h.store.SaveVariant(snapshot.Variant.Slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update variant section: %w", err))
	}
	return connect.NewResponse(&lpbsv1.VariantSectionResponse{Section: sectionProto(*section)}), nil
}

func (h *variantConnectHandler) DeleteVariantSection(_ context.Context, request *connect.Request[lpbsv1.DeleteVariantSectionRequest]) (*connect.Response[lpbsv1.DeleteVariantSectionResponse], error) {
	snapshot, err := h.variant(request.Msg.GetSlug())
	if err != nil {
		return nil, err
	}
	_, index, err := variantSection(snapshot, request.Msg.GetSectionKey())
	if err != nil {
		return nil, err
	}
	snapshot.Sections = append(snapshot.Sections[:index], snapshot.Sections[index+1:]...)
	if err := h.store.SaveVariant(snapshot.Variant.Slug, snapshot); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete variant section: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteVariantSectionResponse{Deleted: true}), nil
}

func variantSection(snapshot *experimentation.VariantSnapshot, key string) (*experimentation.VariantSection, int, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, -1, connect.NewError(connect.CodeInvalidArgument, errors.New("section_key is required"))
	}
	for index := range snapshot.Sections {
		if snapshot.Sections[index].Key == key {
			return &snapshot.Sections[index], index, nil
		}
	}
	return nil, -1, connect.NewError(connect.CodeNotFound, fmt.Errorf("section %q not found", key))
}

func (h *variantConnectHandler) variant(slug string) (*experimentation.VariantSnapshot, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("variant slug required"))
	}
	snapshot, err := h.store.GetVariant(slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("variant %q not found", slug))
	}
	return snapshot, nil
}

func activeVariantSnapshots(snapshots []*experimentation.VariantSnapshot) []*experimentation.VariantSnapshot {
	active := make([]*experimentation.VariantSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if experimentation.NormalizeVariantStatus(snapshot.Variant.Status) == "active" {
			active = append(active, snapshot)
		}
	}
	return active
}

func cloneControlSections(store experimentation.ConfigStoreReader) []experimentation.VariantSection {
	control, err := store.GetVariant("control")
	if err != nil {
		return []experimentation.VariantSection{}
	}
	sections := make([]experimentation.VariantSection, len(control.Sections))
	for i, section := range control.Sections {
		sections[i] = section
		sections[i].Content = append(json.RawMessage(nil), section.Content...)
	}
	return sections
}

func cloneAxes(axes map[string]string) map[string]string {
	result := make(map[string]string, len(axes))
	for key, value := range axes {
		result[key] = value
	}
	return result
}

func variantProto(snapshot *experimentation.VariantSnapshot, includeSEO bool) *lpbsv1.Variant {
	result := &lpbsv1.Variant{Slug: snapshot.Variant.Slug, Name: snapshot.Variant.Name, Description: snapshot.Variant.Description, Weight: boundedProtoInt32(snapshot.Variant.Weight), Status: experimentation.NormalizeVariantStatus(snapshot.Variant.Status), Axes: cloneAxes(snapshot.Variant.Axes)}
	header, err := HeaderProto(snapshot.Variant.HeaderConfig)
	if err == nil {
		result.HeaderConfig = header
	}
	if includeSEO {
		result.SeoConfig = seoRawToProto(snapshot.Variant.SEOConfig)
	}
	return result
}

func snapshotProto(snapshot *experimentation.VariantSnapshot) *lpbsv1.VariantSnapshot {
	result := &lpbsv1.VariantSnapshot{Slug: snapshot.Variant.Slug, Name: snapshot.Variant.Name, Description: snapshot.Variant.Description, Weight: boundedProtoInt32(snapshot.Variant.Weight), Status: experimentation.NormalizeVariantStatus(snapshot.Variant.Status), Axes: cloneAxes(snapshot.Variant.Axes)}
	header, err := HeaderProto(snapshot.Variant.HeaderConfig)
	if err == nil {
		result.HeaderConfig = header
	}
	result.SeoConfig = seoRawToProto(snapshot.Variant.SEOConfig)
	for _, section := range snapshot.Sections {
		result.Sections = append(result.Sections, sectionProto(section))
	}
	return result
}

func snapshotFromProto(input *lpbsv1.VariantSnapshot) (*experimentation.VariantSnapshot, error) {
	slug, name := strings.TrimSpace(input.GetSlug()), strings.TrimSpace(input.GetName())
	if slug == "" || name == "" || len(input.GetAxes()) == 0 {
		return nil, errors.New("snapshot slug, name, and axes are required")
	}
	sections := make([]experimentation.VariantSection, 0, len(input.GetSections()))
	for index, section := range input.GetSections() {
		converted, err := sectionFromProto(section, index)
		if err != nil {
			return nil, err
		}
		sections = append(sections, converted)
	}
	header := experimentation.DefaultLandingHeaderConfig(name)
	if input.GetHeaderConfig() != nil {
		header = experimentation.NormalizeLandingHeaderConfig(headerFromProto(input.GetHeaderConfig()), name)
	}
	return &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{Slug: slug, Name: name, Description: input.GetDescription(), Weight: int(input.GetWeight()), Status: experimentation.NormalizeVariantStatus(input.GetStatus()), Axes: cloneAxes(input.GetAxes()), HeaderConfig: header, SEOConfig: seoProtoToRaw(input.GetSeoConfig())}, Sections: sections}, nil
}

func sectionProto(section experimentation.VariantSection) *sharedv1.ContentSection {
	content := &structpb.Struct{}
	var values map[string]any
	if len(section.Content) > 0 && json.Unmarshal(section.Content, &values) == nil {
		content, _ = structpb.NewStruct(values)
	}
	return &sharedv1.ContentSection{Key: section.Key, SectionType: section.SectionType, Content: content, Order: boundedProtoInt32(section.Order), Enabled: section.Enabled}
}

// boundedProtoInt32 preserves the protobuf wire range when a persisted JSON
// configuration was authored outside the UI's normal validation path.
func boundedProtoInt32(value int) int32 {
	const max = int(^uint32(0) >> 1)
	const min = -max - 1
	if value > max {
		return int32(max)
	}
	if value < min {
		return int32(min)
	}
	return int32(value)
}

func sectionFromProto(section *sharedv1.ContentSection, index int) (experimentation.VariantSection, error) {
	if section == nil || strings.TrimSpace(section.GetSectionType()) == "" {
		return experimentation.VariantSection{}, errors.New("snapshot section type is required")
	}
	content, err := json.Marshal(section.GetContent().AsMap())
	if err != nil {
		return experimentation.VariantSection{}, fmt.Errorf("encode section content: %w", err)
	}
	order := int(section.GetOrder())
	if order <= 0 {
		order = index + 1
	}
	return experimentation.VariantSection{Key: strings.TrimSpace(section.GetKey()), SectionType: section.GetSectionType(), Content: content, Order: order, Enabled: section.GetEnabled()}, nil
}

func seoRawToProto(raw json.RawMessage) *sharedv1.VariantSEOConfig {
	if len(raw) == 0 {
		return nil
	}
	var config contracts.VariantSEOConfig
	if json.Unmarshal(raw, &config) != nil {
		return nil
	}
	result := &sharedv1.VariantSEOConfig{Title: config.Title, Description: config.Description, OgTitle: config.OGTitle, OgDescription: config.OGDescription, OgImageUrl: config.OGImageURL, TwitterCard: config.TwitterCard, CanonicalPath: config.CanonicalPath, Noindex: config.NoIndex}
	if len(config.StructuredData) > 0 {
		result.StructuredData, _ = structpb.NewStruct(config.StructuredData)
	}
	return result
}

func seoProtoToRaw(config *sharedv1.VariantSEOConfig) json.RawMessage {
	if config == nil {
		return nil
	}
	value := seoConfigFromProto(config)
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func seoConfigFromProto(config *sharedv1.VariantSEOConfig) contracts.VariantSEOConfig {
	if config == nil {
		return contracts.VariantSEOConfig{}
	}
	result := contracts.VariantSEOConfig{Title: config.GetTitle(), Description: config.GetDescription(), OGTitle: config.GetOgTitle(), OGDescription: config.GetOgDescription(), OGImageURL: config.GetOgImageUrl(), TwitterCard: config.GetTwitterCard(), CanonicalPath: config.GetCanonicalPath(), NoIndex: config.GetNoindex()}
	if structuredData := config.GetStructuredData(); structuredData != nil {
		result.StructuredData = structuredData.AsMap()
	}
	return result
}

func headerFromProto(header *sharedv1.LandingHeaderConfig) *experimentation.LandingHeaderConfig {
	if header == nil {
		return nil
	}
	result := &experimentation.LandingHeaderConfig{}
	if branding := header.GetBranding(); branding != nil {
		result.Branding = experimentation.HeaderBrandingConfig{Mode: branding.GetMode(), Label: branding.GetLabel(), Subtitle: branding.GetSubtitle(), MobilePreference: branding.GetMobilePreference()}
	}
	if nav := header.GetNav(); nav != nil {
		for _, link := range nav.GetLinks() {
			result.Nav.Links = append(result.Nav.Links, headerLinkFromProto(link))
		}
	}
	if ctas := header.GetCtas(); ctas != nil {
		if primary := ctas.GetPrimary(); primary != nil {
			result.Ctas.Primary = experimentation.HeaderCTAConfig{Mode: primary.GetMode(), Label: primary.GetLabel(), Href: primary.GetHref(), Variant: primary.GetVariant()}
		}
		if secondary := ctas.GetSecondary(); secondary != nil {
			result.Ctas.Secondary = experimentation.HeaderCTAConfig{Mode: secondary.GetMode(), Label: secondary.GetLabel(), Href: secondary.GetHref(), Variant: secondary.GetVariant()}
		}
	}
	if behavior := header.GetBehavior(); behavior != nil {
		result.Behavior = experimentation.HeaderBehaviorConfig{Sticky: behavior.GetSticky(), HideOnScroll: behavior.GetHideOnScroll()}
	}
	return result
}

func headerLinkFromProto(link *sharedv1.HeaderNavLink) experimentation.HeaderNavLink {
	result := experimentation.HeaderNavLink{ID: link.GetId(), Type: link.GetType(), Label: link.GetLabel(), SectionType: link.GetSectionType(), Anchor: link.GetAnchor(), Href: link.GetHref()}
	if visible := link.GetVisibleOn(); visible != nil {
		result.VisibleOn = experimentation.HeaderVisibilityConfig{Desktop: visible.GetDesktop(), Mobile: visible.GetMobile()}
	}
	if sectionID := link.SectionId; sectionID != nil {
		value := int(*sectionID)
		result.SectionID = &value
	}
	for _, child := range link.GetChildren() {
		result.Children = append(result.Children, headerLinkFromProto(child))
	}
	return result
}

// registerVariantConnectRoutes preserves public reads while enforcing the
// existing administrator boundary for every mutation and administrative read.
func RegisterConnectRoutes(router *mux.Router, store experimentation.ConfigStorer, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	handler := newVariantConnectHandler(store)
	_, generated := lpbsconnect.NewVariantServiceHandler(handler)
	_, sectionGenerated := lpbsconnect.NewVariantSectionServiceHandler(handler)
	router.Handle(lpbsconnect.VariantServiceSelectVariantProcedure, generated).Methods("POST")
	router.Handle(lpbsconnect.VariantServiceGetPublicVariantProcedure, generated).Methods("POST")
	for _, procedure := range []string{lpbsconnect.VariantServiceGetVariantProcedure, lpbsconnect.VariantServiceListVariantsProcedure, lpbsconnect.VariantServiceCreateVariantProcedure, lpbsconnect.VariantServiceUpdateVariantProcedure, lpbsconnect.VariantServiceArchiveVariantProcedure, lpbsconnect.VariantServiceDeleteVariantProcedure, lpbsconnect.VariantServiceExportVariantSnapshotProcedure, lpbsconnect.VariantServiceImportVariantSnapshotProcedure, lpbsconnect.VariantServiceSyncVariantSnapshotsProcedure} {
		router.Handle(procedure, requireAdmin(generated.ServeHTTP)).Methods("POST")
	}
	for _, procedure := range []string{lpbsconnect.VariantSectionServiceGetVariantSectionsProcedure, lpbsconnect.VariantSectionServiceGetVariantSectionProcedure, lpbsconnect.VariantSectionServiceCreateVariantSectionProcedure, lpbsconnect.VariantSectionServiceUpdateVariantSectionProcedure, lpbsconnect.VariantSectionServiceDeleteVariantSectionProcedure} {
		router.Handle(procedure, requireAdmin(sectionGenerated.ServeHTTP)).Methods("POST")
	}
}

var (
	_ lpbsconnect.VariantServiceHandler        = (*variantConnectHandler)(nil)
	_ lpbsconnect.VariantSectionServiceHandler = (*variantConnectHandler)(nil)
)

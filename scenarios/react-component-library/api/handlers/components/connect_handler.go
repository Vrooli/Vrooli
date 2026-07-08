package components

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/components"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

// Deps wires the seams the Connect components handler needs. Service
// handles read paths; Repo is required by the indexer (DeleteMissing
// and Upsert sit on the repository surface, deliberately not on the
// service — those are walker-internal concerns, not application
// policy).
type Deps struct {
	Service    components.Service
	Repo       components.Repository
	SourceRoot string
	Logger     *log.Logger
	// IndexObserver is the optional post-upsert seam wired by main.go
	// to drive cross-domain consumers (currently the deps service's
	// SyncForComponent — req 10). Nil = no observer; the indexer
	// behaves exactly as before.
	IndexObserver components.UpsertObserver
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

func (h *connectHandler) ListComponents(ctx context.Context, req *connect.Request[componentsv1.ListComponentsRequest]) (*connect.Response[componentsv1.ListComponentsResponse], error) {
	out, err := h.deps.Service.List(ctx, components.SearchQuery{
		Match:    req.Msg.Match,
		Tag:      req.Msg.Tag,
		Tags:     append([]string(nil), req.Msg.Tags...),
		Category: req.Msg.Category,
		StyleID:  req.Msg.StyleId,
		Affinity: req.Msg.Affinity,
		Limit:    int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("components.ListComponents: %v", err)
		return nil, components.ToConnectError(err)
	}
	resp := &componentsv1.ListComponentsResponse{
		Components: make([]*componentsv1.Component, 0, len(out)),
	}
	for _, c := range out {
		resp.Components = append(resp.Components, domainToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetComponent(ctx context.Context, req *connect.Request[componentsv1.GetComponentRequest]) (*connect.Response[componentsv1.GetComponentResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.GetComponent(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.GetComponentResponse{Component: domainToProto(got)}), nil
}

func (h *connectHandler) GetComponentByLibraryId(ctx context.Context, req *connect.Request[componentsv1.GetComponentByLibraryIdRequest]) (*connect.Response[componentsv1.GetComponentByLibraryIdResponse], error) {
	got, err := h.deps.Service.GetByLibraryID(ctx, req.Msg.LibraryId)
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.GetComponentByLibraryId(%q): %v", req.Msg.LibraryId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.GetComponentByLibraryIdResponse{Component: domainToProto(got)}), nil
}

func (h *connectHandler) GetComponentContent(ctx context.Context, req *connect.Request[componentsv1.GetComponentContentRequest]) (*connect.Response[componentsv1.GetComponentContentResponse], error) {
	content, err := h.deps.Service.GetContent(ctx, req.Msg.Id)
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.GetComponentContent(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.GetComponentContentResponse{
		Content:    content.Body,
		SourcePath: content.SourcePath,
		Sha256:     content.SHA256,
	}), nil
}

func (h *connectHandler) InitializeComponent(ctx context.Context, req *connect.Request[componentsv1.InitializeComponentRequest]) (*connect.Response[componentsv1.InitializeComponentResponse], error) {
	out, err := h.deps.Service.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID:      req.Msg.LibraryId,
		Slug:           req.Msg.Slug,
		DisplayName:    req.Msg.DisplayName,
		Description:    req.Msg.Description,
		Tags:           append([]string(nil), req.Msg.Tags...),
		InitialVersion: req.Msg.InitialVersion,
		FileName:       req.Msg.FileName,
		InitialSource:  req.Msg.InitialSource,
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.InitializeComponent(%q): %v", req.Msg.LibraryId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.InitializeComponentResponse{
		Component:    domainToProto(out.Component),
		ManifestPath: out.ManifestPath,
		SourcePath:   out.SourcePath,
	}), nil
}

func (h *connectHandler) CreateComponentVersion(ctx context.Context, req *connect.Request[componentsv1.CreateComponentVersionRequest]) (*connect.Response[componentsv1.CreateComponentVersionResponse], error) {
	out, err := h.deps.Service.CreateComponentVersion(ctx, components.CreateComponentVersionInput{
		ComponentID: req.Msg.ComponentId,
		Version:     req.Msg.Version,
		FromVersion: req.Msg.FromVersion,
		Intent:      protoIntentToDomain(req.Msg.Intent),
		FileName:    req.Msg.FileName,
		Source:      req.Msg.Source,
		ChangelogMD: req.Msg.ChangelogMd,
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.CreateComponentVersion(%q, %q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.CreateComponentVersionResponse{
		Component:  domainToProto(out.Component),
		Version:    versionToProto(out.Version),
		SourcePath: out.SourcePath,
	}), nil
}

func (h *connectHandler) UpdateComponentManifest(ctx context.Context, req *connect.Request[componentsv1.UpdateComponentManifestRequest]) (*connect.Response[componentsv1.UpdateComponentManifestResponse], error) {
	out, err := h.deps.Service.UpdateComponentManifest(ctx, components.UpdateComponentManifestInput{
		ComponentID:        req.Msg.ComponentId,
		DisplayName:        req.Msg.DisplayName,
		Description:        req.Msg.Description,
		Tags:               append([]string(nil), req.Msg.Tags...),
		LatestVersion:      req.Msg.LatestVersion,
		DraftVersion:       req.Msg.DraftVersion,
		DeprecatedVersions: append([]string(nil), req.Msg.DeprecatedVersions...),
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.UpdateComponentManifest(%q): %v", req.Msg.ComponentId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.UpdateComponentManifestResponse{Component: domainToProto(out)}), nil
}

func (h *connectHandler) ListComponentVersions(ctx context.Context, req *connect.Request[componentsv1.ListComponentVersionsRequest]) (*connect.Response[componentsv1.ListComponentVersionsResponse], error) {
	rows, err := h.deps.Service.ListVersions(ctx, req.Msg.ComponentId, int(req.Msg.Limit))
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.ListComponentVersions(%q): %v", req.Msg.ComponentId, err)
		}
		return nil, connectErr
	}
	resp := &componentsv1.ListComponentVersionsResponse{Versions: make([]*componentsv1.ComponentVersion, 0, len(rows))}
	for _, v := range rows {
		resp.Versions = append(resp.Versions, versionToProto(v))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListDesignStyles(ctx context.Context, req *connect.Request[componentsv1.ListDesignStylesRequest]) (*connect.Response[componentsv1.ListDesignStylesResponse], error) {
	rows, err := h.deps.Service.ListDesignStyles(ctx)
	if err != nil {
		h.deps.Logger.Printf("components.ListDesignStyles: %v", err)
		return nil, components.ToConnectError(err)
	}
	resp := &componentsv1.ListDesignStylesResponse{Styles: make([]*componentsv1.DesignStyle, 0, len(rows))}
	for _, style := range rows {
		resp.Styles = append(resp.Styles, &componentsv1.DesignStyle{
			Id:       style.ID,
			Name:     style.Name,
			Tags:     append([]string(nil), style.Tags...),
			Supports: append([]string(nil), style.Supports...),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateStyleFit(ctx context.Context, req *connect.Request[componentsv1.ValidateStyleFitRequest]) (*connect.Response[componentsv1.ValidateStyleFitResponse], error) {
	verdict, err := h.deps.Service.ValidateStyleFit(ctx, req.Msg.ComponentId, req.Msg.Version, req.Msg.Scenario)
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.ValidateStyleFit(%q, %q, %q): %v", req.Msg.ComponentId, req.Msg.Version, req.Msg.Scenario, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(styleFitVerdictToProto(verdict)), nil
}

func (h *connectHandler) GetComponentVersionContent(ctx context.Context, req *connect.Request[componentsv1.GetComponentVersionContentRequest]) (*connect.Response[componentsv1.GetComponentVersionContentResponse], error) {
	v, err := h.deps.Service.GetVersion(ctx, req.Msg.ComponentId, req.Msg.Version)
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.GetComponentVersionContent(%q, %q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.GetComponentVersionContentResponse{
		Version: versionToProto(v),
		Content: v.Content,
	}), nil
}

func (h *connectHandler) UpdateComponentContent(ctx context.Context, req *connect.Request[componentsv1.UpdateComponentContentRequest]) (*connect.Response[componentsv1.UpdateComponentContentResponse], error) {
	content, err := h.deps.Service.UpdateContent(ctx, req.Msg.Id, components.WriteContentInput{
		Body:           req.Msg.Content,
		ExpectedSHA256: req.Msg.ExpectedSha256,
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.UpdateComponentContent(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&componentsv1.UpdateComponentContentResponse{
		Sha256:     content.SHA256,
		SourcePath: content.SourcePath,
	}), nil
}

func (h *connectHandler) IndexComponents(ctx context.Context, _ *connect.Request[componentsv1.IndexComponentsRequest]) (*connect.Response[componentsv1.IndexComponentsResponse], error) {
	idx := components.NewIndexer(h.deps.Repo, h.deps.SourceRoot, nil)
	if h.deps.IndexObserver != nil {
		idx.SetUpsertObserver(h.deps.IndexObserver)
	}
	res, err := idx.Run(ctx)
	if err != nil {
		h.deps.Logger.Printf("components.IndexComponents: %v", err)
		return nil, components.ToConnectError(err)
	}
	errs := make([]string, 0, len(res.Errors))
	for _, e := range res.Errors {
		errs = append(errs, e.Error())
	}
	for _, finding := range res.Findings {
		errs = append(errs, formatIndexFinding(finding))
	}
	return connect.NewResponse(&componentsv1.IndexComponentsResponse{
		Scanned:    int32(res.Scanned),
		Indexed:    int32(res.Indexed),
		Skipped:    int32(res.Skipped),
		Deleted:    int32(res.Deleted),
		Errors:     errs,
		LibraryIds: append([]string(nil), res.LibraryIDs...),
	}), nil
}

func formatIndexFinding(finding components.IndexFinding) string {
	if finding.Detail != "" {
		return fmt.Sprintf("finding:%s:%s:%s", finding.Kind, finding.SourcePath, finding.Detail)
	}
	return fmt.Sprintf("finding:%s:%s:%s expected %q got %q", finding.Kind, finding.SourcePath, finding.Field, finding.Expected, finding.Actual)
}

func protoIntentToDomain(intent componentsv1.ComponentVersionIntent) components.VersionIntent {
	switch intent {
	case componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_DRAFT:
		return components.VersionIntentDraft
	case componentsv1.ComponentVersionIntent_COMPONENT_VERSION_INTENT_RELEASE:
		return components.VersionIntentRelease
	default:
		return ""
	}
}

package components

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"react-component-library/internal/components"
	"react-component-library/internal/experience"
	previewdomain "react-component-library/internal/preview"
	"react-component-library/internal/versionledger"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

// Deps wires the seams the Connect components handler needs. Service
// handles read paths; Repo is required by the indexer (DeleteMissing
// and Upsert sit on the repository surface, deliberately not on the
// service — those are walker-internal concerns, not application
// policy).
type Deps struct {
	Service    components.Service
	Authoring  components.AuthoringService
	Repo       components.Repository
	SourceRoot string
	Logger     *log.Logger
	// IndexObserver is the optional post-upsert seam wired by main.go
	// to drive cross-domain consumers (currently the deps service's
	// SyncForComponent — req 10). Nil = no observer; the indexer
	// behaves exactly as before.
	IndexObserver    components.UpsertObserver
	ExperienceReader experience.Reader
	VersionLedger    *versionledger.Repository
	Preview          previewdomain.Service
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
		Match:     req.Msg.Match,
		Tag:       req.Msg.Tag,
		Tags:      append([]string(nil), req.Msg.Tags...),
		Category:  req.Msg.Category,
		StyleID:   req.Msg.StyleId,
		Affinity:  req.Msg.Affinity,
		AssetKind: protoAssetKindToDomain(req.Msg.AssetKind),
		Limit:     int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("components.ListComponents: %v", err)
		return nil, components.ToConnectError(err)
	}
	resp := &componentsv1.ListComponentsResponse{
		Components: make([]*componentsv1.Component, 0, len(out)),
	}
	index, indexErr := h.catalogIndex()
	if indexErr != nil {
		// Not fatal: the implementation registry is still authoritative for
		// everything except placement. Log it rather than failing the list, but
		// do log it — a silently unenriched list is what made every asset render
		// under "Other / Rung 0" without anything reporting a fault.
		h.deps.Logger.Printf("components.ListComponents: catalog projection unavailable: %v", indexErr)
	}
	for _, c := range out {
		h.enrichCatalogProjection(index, &c)
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
	index, indexErr := h.catalogIndex()
	if indexErr != nil {
		h.deps.Logger.Printf("components.GetComponent(%q): catalog projection unavailable: %v", req.Msg.Id, indexErr)
	}
	h.enrichCatalogProjection(index, &got)
	resp := &componentsv1.GetComponentResponse{Component: domainToProto(got)}
	if req.Msg.GetIncludeExperience() {
		if h.deps.ExperienceReader == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component experience reader is not configured"))
		}
		snapshot, err := h.deps.ExperienceReader.Get(ctx, experience.Component{ID: got.ID, LibraryID: got.LibraryID, Slug: got.Slug, Version: got.Version})
		if err != nil {
			h.deps.Logger.Printf("components.GetComponent(%q) experience: %v", req.Msg.GetId(), err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read component experience: %w", err))
		}
		resp.Experience = experienceToProto(snapshot)
	}
	return connect.NewResponse(resp), nil
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
	content, err := h.deps.Service.GetContentAt(ctx, req.Msg.Id, req.Msg.Path)
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
		LibraryID:        req.Msg.LibraryId,
		Slug:             req.Msg.Slug,
		DisplayName:      req.Msg.DisplayName,
		Description:      req.Msg.Description,
		Tags:             append([]string(nil), req.Msg.Tags...),
		InitialVersion:   req.Msg.InitialVersion,
		FileName:         req.Msg.FileName,
		InitialSource:    req.Msg.InitialSource,
		ScaffoldExamples: true,
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

func (h *connectHandler) IngestComponent(ctx context.Context, req *connect.Request[componentsv1.IngestComponentRequest]) (*connect.Response[componentsv1.IngestComponentResponse], error) {
	out, err := h.deps.Service.IngestComponent(ctx, components.IngestComponentInput{
		Scenario:               req.Msg.Scenario,
		SourceFile:             req.Msg.SourceFile,
		SourceFiles:            append([]string(nil), req.Msg.SourceFiles...),
		Version:                req.Msg.Version,
		Slug:                   req.Msg.Slug,
		DisplayName:            req.Msg.DisplayName,
		Description:            req.Msg.Description,
		Tags:                   append([]string(nil), req.Msg.Tags...),
		Slot:                   req.Msg.Slot,
		AcceptBehaviorLoss:     req.Msg.AcceptBehaviorLoss,
		ExperienceContractPath: req.Msg.ExperienceContractPath,
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.IngestComponent(%q, %q): %v", req.Msg.Scenario, req.Msg.SourceFile, err)
		}
		return nil, connectErr
	}
	findings := make([]*componentsv1.IngestFinding, 0, len(out.Findings))
	for _, finding := range out.Findings {
		findings = append(findings, &componentsv1.IngestFinding{Code: finding.Code, Message: finding.Message, SourceFile: finding.SourceFile})
	}
	return connect.NewResponse(&componentsv1.IngestComponentResponse{
		Component:     domainToProto(out.Component),
		ManifestPath:  out.ManifestPath,
		SourcePath:    out.SourcePath,
		DraftVersion:  out.DraftVersion,
		Findings:      findings,
		ParityReport:  parityReportToProto(out.ParityReport),
		ChecklistPath: out.ChecklistPath,
	}), nil
}

func (h *connectHandler) BeginComponentVersion(ctx context.Context, req *connect.Request[componentsv1.BeginComponentVersionRequest]) (*connect.Response[componentsv1.BeginComponentVersionResponse], error) {
	if h.deps.Authoring == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component authoring service is not configured"))
	}
	out, err := h.deps.Authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{
		Component: req.Msg.GetComponent(),
		Bump:      req.Msg.GetBump(),
		Version:   req.Msg.GetVersion(),
	})
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	return connect.NewResponse(&componentsv1.BeginComponentVersionResponse{
		Component:     domainToProto(out.Component),
		Version:       versionToProto(out.Version),
		SourcePath:    out.SourcePath,
		ArtifactPaths: append([]string(nil), out.ArtifactPaths...),
		PreviewPath:   authoringPreviewPath(out.Component.LibraryID, out.Version.Version),
	}), nil
}

func (h *connectHandler) CheckComponentVersion(ctx context.Context, req *connect.Request[componentsv1.CheckComponentVersionRequest]) (*connect.Response[componentsv1.CheckComponentVersionResponse], error) {
	if h.deps.Authoring == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component authoring service is not configured"))
	}
	out, err := h.deps.Authoring.CheckComponentVersion(ctx, req.Msg.GetComponent(), req.Msg.GetVersion())
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	if h.deps.Preview == nil {
		out.Passed = false
		out.Checks = append(out.Checks, components.ComponentVersionCheck{Stage: "preview", Verdict: "failed", Message: "preview compiler is not configured", Remediation: "restart the scenario with the preview service enabled"})
	} else if _, err := h.deps.Preview.GetBundleVersion(ctx, out.Component.ID, out.Version); err != nil {
		out.Passed = false
		out.Checks = append(out.Checks, components.ComponentVersionCheck{Stage: "preview", Verdict: "failed", Message: err.Error(), Remediation: "fix the component or harness bundle before publishing"})
	} else {
		out.Checks = append(out.Checks, components.ComponentVersionCheck{Stage: "preview", Verdict: "passed", Message: "component and story harness bundled successfully"})
	}
	checks := make([]*componentsv1.ComponentVersionCheck, 0, len(out.Checks))
	for _, check := range out.Checks {
		checks = append(checks, &componentsv1.ComponentVersionCheck{Stage: check.Stage, Verdict: check.Verdict, Message: check.Message, Remediation: check.Remediation})
	}
	return connect.NewResponse(&componentsv1.CheckComponentVersionResponse{
		Component:   domainToProto(out.Component),
		Version:     out.Version,
		Passed:      out.Passed,
		Checks:      checks,
		PreviewPath: authoringPreviewPath(out.Component.LibraryID, out.Version),
	}), nil
}

func (h *connectHandler) PublishComponentVersion(ctx context.Context, req *connect.Request[componentsv1.PublishComponentVersionRequest]) (*connect.Response[componentsv1.PublishComponentVersionResponse], error) {
	if h.deps.Authoring == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component authoring service is not configured"))
	}
	checked, err := h.deps.Authoring.CheckComponentVersion(ctx, req.Msg.GetComponent(), req.Msg.GetDraftVersion())
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	if !checked.Passed {
		return nil, components.ToConnectError(components.ErrVersionCheckFailed{LibraryID: checked.Component.LibraryID, Version: checked.Version, Checks: checked.Checks})
	}
	if h.deps.Preview == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("preview compiler is not configured"))
	}
	if _, err := h.deps.Preview.GetBundleVersion(ctx, checked.Component.ID, checked.Version); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("preview check failed for %s@%s: %w", checked.Component.LibraryID, checked.Version, err))
	}
	out, err := h.deps.Authoring.PublishComponentVersion(ctx, components.PublishComponentVersionInput{
		Component:               req.Msg.GetComponent(),
		DraftVersion:            checked.Version,
		Version:                 req.Msg.GetVersion(),
		ChangelogMD:             req.Msg.GetChangelogMd(),
		AcknowledgeParityWaiver: req.Msg.GetAcknowledgeParityWaiver(),
	})
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	return connect.NewResponse(&componentsv1.PublishComponentVersionResponse{
		Component:     domainToProto(out.Component),
		Version:       versionToProto(out.Version),
		SourcePath:    out.SourcePath,
		ArtifactPaths: append([]string(nil), out.ArtifactPaths...),
		PreviewPath:   authoringPreviewPath(out.Component.LibraryID, out.Version.Version),
	}), nil
}

func authoringPreviewPath(libraryID, version string) string {
	return "/preview/" + url.PathEscape(libraryID) + "/harness.html?version=" + url.QueryEscape(version)
}

func (h *connectHandler) CreateComponentVersion(ctx context.Context, req *connect.Request[componentsv1.CreateComponentVersionRequest]) (*connect.Response[componentsv1.CreateComponentVersionResponse], error) {
	out, err := h.deps.Service.CreateComponentVersion(ctx, components.CreateComponentVersionInput{
		ComponentID:             req.Msg.ComponentId,
		Version:                 req.Msg.Version,
		FromVersion:             req.Msg.FromVersion,
		Intent:                  protoIntentToDomain(req.Msg.Intent),
		FileName:                req.Msg.FileName,
		Source:                  req.Msg.Source,
		ChangelogMD:             req.Msg.ChangelogMd,
		AcknowledgeParityWaiver: req.Msg.AcknowledgeParityWaiver,
		ScaffoldExamples:        true,
		ParityReport:            parityReportFromProto(req.Msg.ParityReport),
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
	content := v.Content
	if path := req.Msg.Path; path != "" {
		found := false
		for _, file := range v.Files {
			if file.Path == path {
				content, found = file.Content, true
				break
			}
		}
		if !found {
			return nil, components.ToConnectError(components.ErrComponentNotFound{IDOrLibraryID: req.Msg.ComponentId + "@" + req.Msg.Version + "/" + path})
		}
	}
	return connect.NewResponse(&componentsv1.GetComponentVersionContentResponse{
		Version: versionToProto(v),
		Content: content,
	}), nil
}

func (h *connectHandler) ListComponentStories(ctx context.Context, req *connect.Request[componentsv1.ListComponentStoriesRequest]) (*connect.Response[componentsv1.ListComponentStoriesResponse], error) {
	rows, err := h.deps.Service.ListStories(ctx, components.StoryQuery{
		ComponentID: req.Msg.ComponentId,
		Version:     req.Msg.Version,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		connectErr := components.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("components.ListComponentStories(%q, %q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	resp := &componentsv1.ListComponentStoriesResponse{Stories: make([]*componentsv1.ComponentStory, 0, len(rows))}
	for _, story := range rows {
		resp.Stories = append(resp.Stories, storyToProto(story))
	}
	return connect.NewResponse(resp), nil
}

// ListPreviewFrames exposes catalog-owned frame candidates and their
// compatibility diagnostics. The UI may render compatible candidates as a
// temporary experiment; this RPC never edits story.json.
func (h *connectHandler) ListPreviewFrames(ctx context.Context, req *connect.Request[componentsv1.ListPreviewFramesRequest]) (*connect.Response[componentsv1.ListPreviewFramesResponse], error) {
	if strings.TrimSpace(req.Msg.ComponentId) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("component_id required"))
	}
	subject, err := h.deps.Service.Get(ctx, req.Msg.ComponentId)
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	assets, err := components.CatalogFrameAssetsFromDir(filepath.Join(filepath.Dir(h.deps.SourceRoot), "catalog"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load frame catalog: %w", err))
	}
	implementations, err := h.deps.Service.List(ctx, components.SearchQuery{Limit: 10000})
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	capability := subjectCapability(subject.CatalogID)
	out := &componentsv1.ListPreviewFramesResponse{}
	for _, asset := range assets {
		if asset.Kind == "fixture" || len(asset.Regions) == 0 {
			continue
		}
		regions := previewRegionsForSubject(asset, capability)
		for _, selectedRegion := range regions {
			candidate := &componentsv1.PreviewFrameCandidate{Asset: asset.ID, Region: selectedRegion, Label: asset.ID, Compatible: true}
			if !containsString(asset.Targets, "react-vite") {
				candidate.Compatible = false
				candidate.DiagnosticCode = "unsupported_target"
				candidate.Diagnostic = "frame does not declare react-vite support"
			}
			if !canonicalPreviewFrame(asset.ID) && candidate.Compatible {
				candidate.Compatible = false
				candidate.DiagnosticCode = "not_canonical_frame"
				candidate.Diagnostic = "region-bearing catalog asset is not approved as a canonical Preview frame"
			}
			matches := findCatalogImplementations(implementations, asset.ID)
			if len(matches) == 0 && candidate.Compatible {
				candidate.Compatible = false
				candidate.DiagnosticCode = "implementation_missing"
				candidate.Diagnostic = "catalog frame has no indexed react-vite implementation"
			} else if len(matches) > 1 && candidate.Compatible {
				candidate.Compatible = false
				candidate.DiagnosticCode = "implementation_ambiguous"
				candidate.Diagnostic = "catalog frame resolves to multiple indexed implementations"
			} else if len(matches) == 1 {
				candidate.Version = matches[0].LatestVersion
			}
			if candidate.Compatible {
				if required := asset.RegionCapabilities[selectedRegion]; required != "" {
					candidate.Capability = required
					if capability != required {
						candidate.Compatible = false
						candidate.DiagnosticCode = "subject_capability_mismatch"
						candidate.Diagnostic = fmt.Sprintf("subject exposes %q; region requires %q", capability, required)
					}
				}
			}
			for _, expected := range asset.Expects {
				// Adapter ports are supplied by the Preview runtime. Only a
				// catalog-declared data-source port requires a fixture asset.
				if expected.Capability != "data-source" {
					continue
				}
				fixture := findFixture(assets, expected)
				if fixture == "" {
					candidate.Compatible = false
					candidate.DiagnosticCode = "fixture_unavailable"
					candidate.Diagnostic = fmt.Sprintf("no fixture satisfies %q", expected.Capability)
				} else {
					candidate.Fixture = fixture
				}
			}
			out.Candidates = append(out.Candidates, candidate)
		}
	}
	return connect.NewResponse(out), nil
}

// PersistPreviewFrame promotes a temporary frame experiment into a new draft
// story contract. The explicit draft boundary is important: Preview controls
// must never mutate released library bytes in place.
func (h *connectHandler) PersistPreviewFrame(ctx context.Context, req *connect.Request[componentsv1.PersistPreviewFrameRequest]) (*connect.Response[componentsv1.PersistPreviewFrameResponse], error) {
	if strings.TrimSpace(req.Msg.ComponentId) == "" || strings.TrimSpace(req.Msg.StoryId) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("component_id and story_id are required"))
	}
	if strings.TrimSpace(req.Msg.Asset) == "" || strings.TrimSpace(req.Msg.FrameVersion) == "" || strings.TrimSpace(req.Msg.Region) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("asset, frame_version, and region are required"))
	}
	subject, err := h.deps.Service.Get(ctx, req.Msg.ComponentId)
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	if err := h.validatePreviewFramePersistence(ctx, subject, req.Msg.Asset, req.Msg.FrameVersion, req.Msg.Region, req.Msg.Capability); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	requestedVersion := strings.TrimSpace(req.Msg.Version)
	if requestedVersion != "" && requestedVersion != subject.LatestVersion && requestedVersion != subject.DraftVersion {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("frame persistence only targets the latest release or active draft; %q is not current", requestedVersion))
	}
	if h.deps.Authoring == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component authoring service is not configured"))
	}
	targetVersion := subject.DraftVersion
	if targetVersion == "" {
		// Validate the story before creating a draft. An invalid story id must
		// not leave an otherwise empty authoring draft behind.
		currentStories, listErr := h.deps.Service.ListStories(ctx, components.StoryQuery{ComponentID: subject.ID, Version: subject.LatestVersion, Limit: 1})
		if listErr != nil {
			return nil, components.ToConnectError(listErr)
		}
		storyExists := false
		for _, story := range currentStories {
			contract, _ := components.ParseStoryContract([]byte(story.ContractJSON))
			if contract == nil {
				continue
			}
			for _, definition := range contract.Stories {
				if definition.ID == req.Msg.StoryId {
					storyExists = true
					break
				}
			}
		}
		if !storyExists {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("story %q not found in %s@%s", req.Msg.StoryId, subject.LibraryID, subject.LatestVersion))
		}
		begun, beginErr := h.deps.Authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{
			Component: subject.LibraryID,
			Bump:      "patch",
		})
		if beginErr != nil {
			return nil, components.ToConnectError(beginErr)
		}
		targetVersion = begun.Version.Version
	}
	stories, err := h.deps.Service.ListStories(ctx, components.StoryQuery{ComponentID: subject.ID, Version: targetVersion, Limit: 1})
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	if len(stories) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("story contract not found for %s@%s", subject.LibraryID, targetVersion))
	}
	contract, diagnostics := components.ParseStoryContract([]byte(stories[0].ContractJSON))
	if len(components.StoryContractErrors(diagnostics)) > 0 || contract == nil {
		parts := make([]string, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			parts = append(parts, diagnostic.Error())
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("story contract is not persistable: %s", strings.Join(parts, "; ")))
	}
	found := false
	for i := range contract.Stories {
		if contract.Stories[i].ID != req.Msg.StoryId {
			continue
		}
		contract.Stories[i].Frame = &components.StoryFrame{
			Asset: req.Msg.Asset, Version: req.Msg.FrameVersion, Region: req.Msg.Region,
			Capability: req.Msg.Capability, Fixture: req.Msg.Fixture,
		}
		found = true
		break
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("story %q not found in %s@%s", req.Msg.StoryId, subject.LibraryID, targetVersion))
	}
	raw, err := json.Marshal(contract)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode story contract: %w", err))
	}
	writer, ok := h.deps.Service.(interface {
		UpdateVersionContentAt(context.Context, string, string, string, components.WriteContentInput) (components.Content, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("versioned story authoring is not configured"))
	}
	written, err := writer.UpdateVersionContentAt(ctx, subject.ID, targetVersion, "story.json", components.WriteContentInput{Body: string(raw)})
	if err != nil {
		return nil, components.ToConnectError(err)
	}
	return connect.NewResponse(&componentsv1.PersistPreviewFrameResponse{
		ComponentId: subject.ID, Version: targetVersion, StoryId: req.Msg.StoryId,
		StoryJson: written.Body, SourcePath: written.SourcePath,
	}), nil
}

func (h *connectHandler) validatePreviewFramePersistence(ctx context.Context, subject components.Component, assetID, version, region, capability string) error {
	assets, err := components.CatalogFrameAssetsFromDir(filepath.Join(filepath.Dir(h.deps.SourceRoot), "catalog"))
	if err != nil {
		return fmt.Errorf("load frame catalog: %w", err)
	}
	var frame components.CatalogFrameAsset
	found := false
	for _, candidate := range assets {
		if candidate.ID == assetID {
			frame, found = candidate, true
			break
		}
	}
	if !found || !canonicalPreviewFrame(assetID) {
		return fmt.Errorf("frame %q is not a canonical Preview frame", assetID)
	}
	if !containsString(frame.Targets, "react-vite") || !containsString(frame.Regions, region) {
		return fmt.Errorf("frame %q does not support target react-vite and region %q", assetID, region)
	}
	required := frame.RegionCapabilities[region]
	if required != "" && required != capability {
		return fmt.Errorf("frame region %q requires capability %q", region, required)
	}
	if capability != "" && subjectCapability(subject.CatalogID) != capability {
		return fmt.Errorf("subject capability %q does not match frame capability %q", subjectCapability(subject.CatalogID), capability)
	}
	matches := func() []components.Component {
		items, listErr := h.deps.Service.List(ctx, components.SearchQuery{Limit: 10000})
		if listErr != nil {
			return nil
		}
		return findCatalogImplementations(items, assetID)
	}()
	if len(matches) == 0 {
		return fmt.Errorf("frame %q has no indexed implementation", assetID)
	}
	if len(matches) > 1 {
		return fmt.Errorf("frame %q resolves to multiple indexed implementations", assetID)
	}
	if _, err := h.deps.Service.GetVersion(ctx, matches[0].ID, version); err != nil {
		return fmt.Errorf("frame %q version %q is unavailable: %w", assetID, version, err)
	}
	return nil
}

func subjectCapability(catalogID string) string {
	if domain, _, ok := strings.Cut(catalogID, "."); ok && domain == "navigation" {
		return "navigation"
	}
	return ""
}

func findCatalogImplementations(items []components.Component, catalogID string) []components.Component {
	matches := make([]components.Component, 0, 1)
	for _, item := range items {
		if item.CatalogID == catalogID && item.AssetKind != components.AssetKindHook {
			matches = append(matches, item)
		}
	}
	return matches
}

func containsString(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}

func canonicalPreviewFrame(assetID string) bool {
	return assetID == "navigation.page" || assetID == "navigation.app-shell"
}

func previewRegionsForSubject(asset components.CatalogFrameAsset, capability string) []string {
	if capability == "navigation" {
		if containsString(asset.Regions, "navigation") {
			return []string{"navigation"}
		}
	}
	if containsString(asset.Regions, "content") {
		return []string{"content"}
	}
	if len(asset.Regions) > 0 {
		return []string{asset.Regions[0]}
	}
	return nil
}

func findFixture(assets []components.CatalogFrameAsset, expected components.CatalogFramePort) string {
	for _, asset := range assets {
		if asset.Kind == "fixture" && asset.FixtureSatisfies != nil && asset.FixtureSatisfies.Capability == expected.Capability {
			return asset.ID
		}
	}
	return ""
}

func (h *connectHandler) UpdateComponentContent(ctx context.Context, req *connect.Request[componentsv1.UpdateComponentContentRequest]) (*connect.Response[componentsv1.UpdateComponentContentResponse], error) {
	content, err := h.deps.Service.UpdateContentAt(ctx, req.Msg.Id, req.Msg.Path, components.WriteContentInput{
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
	if h.deps.VersionLedger != nil {
		if err := h.deps.VersionLedger.Rebuild(ctx); err != nil {
			h.deps.Logger.Printf("components.IndexComponents: rebuild version ledger: %v", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
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

package components

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"react-component-library/internal/components"
	previewdomain "react-component-library/internal/preview"
	"react-component-library/internal/retirement"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

// AuthoringDeps is the narrow port set needed to create and edit canonical
// component source. Registry projections, experience reads, and indexing-only
// observers are deliberately outside this handler's dependency surface.
type AuthoringDeps struct {
	Service    components.Service
	Authoring  components.AuthoringService
	Repo       components.Repository
	SourceRoot string
	Logger     *log.Logger
	Preview    previewdomain.Service
}

type authoringConnectHandler struct{ deps AuthoringDeps }

func NewAuthoringConnectHandler(d AuthoringDeps) *authoringConnectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &authoringConnectHandler{deps: d}
}

func (h *authoringConnectHandler) RetireComponent(ctx context.Context, req *connect.Request[componentsv1.RetireComponentRequest]) (*connect.Response[componentsv1.RetireComponentResponse], error) {
	libraryID := strings.TrimSpace(req.Msg.LibraryId)
	if libraryID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("library_id is required"))
	}
	if h.deps.Repo == nil || filepath.Base(h.deps.SourceRoot) != "library" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("retirement requires the canonical library repository"))
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(h.deps.SourceRoot)))
	var out retirement.Result
	var err error
	if req.Msg.Apply {
		out, err = retirement.Retire(ctx, root, h.deps.Repo, libraryID)
	} else {
		out.LibraryID = libraryID
		var release func()
		ctx, release, err = components.AcquireLibraryMutation(ctx, h.deps.SourceRoot)
		if err == nil {
			defer release()
			var c components.Component
			c, err = h.deps.Repo.GetByLibraryID(ctx, libraryID)
			if err == nil {
				out.ComponentID = c.ID
				out.Preflight, err = retirement.Preflight(ctx, root, c)
			}
		}
	}
	msg := &componentsv1.RetireComponentResponse{
		LibraryId: out.LibraryID, ComponentId: out.ComponentID,
		PreflightCompleted: out.Preflight.Completed, Ready: out.Preflight.Ready(),
		RequiredBy: out.Preflight.RequiredBy, SuggestedBy: out.Preflight.SuggestedBy, SourceReferences: out.Preflight.SourceReferences,
		SnapshotPath: out.Archive.SnapshotPath, SourceArchivePath: out.Archive.SourceArchivePath, CatalogArchivePath: out.CatalogArchivePath, Retired: out.Retired,
	}
	if err != nil {
		msg.Error = err.Error()
	}
	return connect.NewResponse(msg), nil
}

func (h *authoringConnectHandler) GetComponentContent(ctx context.Context, req *connect.Request[componentsv1.GetComponentContentRequest]) (*connect.Response[componentsv1.GetComponentContentResponse], error) {
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

func (h *authoringConnectHandler) InitializeComponent(ctx context.Context, req *connect.Request[componentsv1.InitializeComponentRequest]) (*connect.Response[componentsv1.InitializeComponentResponse], error) {
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

func (h *authoringConnectHandler) IngestComponent(ctx context.Context, req *connect.Request[componentsv1.IngestComponentRequest]) (*connect.Response[componentsv1.IngestComponentResponse], error) {
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

func (h *authoringConnectHandler) BeginComponentVersion(ctx context.Context, req *connect.Request[componentsv1.BeginComponentVersionRequest]) (*connect.Response[componentsv1.BeginComponentVersionResponse], error) {
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

func (h *authoringConnectHandler) CheckComponentVersion(ctx context.Context, req *connect.Request[componentsv1.CheckComponentVersionRequest]) (*connect.Response[componentsv1.CheckComponentVersionResponse], error) {
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

func (h *authoringConnectHandler) PublishComponentVersion(ctx context.Context, req *connect.Request[componentsv1.PublishComponentVersionRequest]) (*connect.Response[componentsv1.PublishComponentVersionResponse], error) {
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

func (h *authoringConnectHandler) CreateComponentVersion(ctx context.Context, req *connect.Request[componentsv1.CreateComponentVersionRequest]) (*connect.Response[componentsv1.CreateComponentVersionResponse], error) {
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

func (h *authoringConnectHandler) UpdateComponentManifest(ctx context.Context, req *connect.Request[componentsv1.UpdateComponentManifestRequest]) (*connect.Response[componentsv1.UpdateComponentManifestResponse], error) {
	if req.Msg.ClearRetiredMajorAliases && len(req.Msg.RetiredMajorAliases) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot set and clear retired major aliases together"))
	}
	var retiredMajorAliases []string
	if req.Msg.ClearRetiredMajorAliases {
		retiredMajorAliases = []string{}
	} else if len(req.Msg.RetiredMajorAliases) > 0 {
		retiredMajorAliases = append([]string(nil), req.Msg.RetiredMajorAliases...)
	}
	dependencies := make([]components.AssetDependency, 0, len(req.Msg.Dependencies))
	for _, dependency := range req.Msg.Dependencies {
		if dependency == nil {
			continue
		}
		dependencies = append(dependencies, components.AssetDependency{LibraryID: dependency.LibraryId, Version: dependency.Version})
	}
	out, err := h.deps.Service.UpdateComponentManifest(ctx, components.UpdateComponentManifestInput{
		ComponentID:                    req.Msg.ComponentId,
		RetiredMajorAliases:            retiredMajorAliases,
		DisplayName:                    req.Msg.DisplayName,
		Description:                    req.Msg.Description,
		Tags:                           append([]string(nil), req.Msg.Tags...),
		LatestVersion:                  req.Msg.LatestVersion,
		DraftVersion:                   req.Msg.DraftVersion,
		DeprecatedVersions:             append([]string(nil), req.Msg.DeprecatedVersions...),
		CatalogID:                      req.Msg.CatalogId,
		Entry:                          req.Msg.Entry,
		ReplacedBy:                     append([]string(nil), req.Msg.ReplacedBy...),
		ClearSupplementalJustification: req.Msg.ClearSupplementalJustification,
		ClearCatalogID:                 req.Msg.ClearCatalogId,
		Dependencies:                   dependencies,
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

func (h *authoringConnectHandler) UpdateComponentContent(ctx context.Context, req *connect.Request[componentsv1.UpdateComponentContentRequest]) (*connect.Response[componentsv1.UpdateComponentContentResponse], error) {
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

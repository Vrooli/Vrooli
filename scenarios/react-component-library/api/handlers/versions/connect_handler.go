package versions

import (
	"context"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"

	"react-component-library/internal/components"
	"react-component-library/internal/versionledger"
	"react-component-library/internal/versions"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
)

// Deps wires the seams the Connect versions handler needs.
type Deps struct {
	Service      versions.Service
	Logger       *log.Logger
	Ledger       *versionledger.Repository
	Components   components.Service
	Materializer components.Materializer
}

func (h *connectHandler) ListRetireCandidates(ctx context.Context, req *connect.Request[versionsv1.ListRetireCandidatesRequest]) (*connect.Response[versionsv1.ListRetireCandidatesResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, err := h.deps.Ledger.RetireCandidates(ctx, req.Msg.GetComponentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.ListRetireCandidatesResponse{}
	for _, item := range items {
		out.Candidates = append(out.Candidates, &versionsv1.RetireCandidate{ComponentId: item.ComponentID, LibraryId: item.LibraryID, Version: item.Version, Status: item.Status})
	}
	return connect.NewResponse(out), nil
}

func cleanupScopeFromProto(scope *versionsv1.CleanupScope) versionledger.CleanupScope {
	if scope == nil {
		return versionledger.CleanupScope{}
	}
	return versionledger.CleanupScope{ComponentID: scope.GetComponentId(), LibraryID: scope.GetLibraryId(), OlderThanDays: int(scope.GetOlderThanDays())}
}

func cleanupItemToProto(item versionledger.CleanupItem) *versionsv1.CleanupItem {
	references := make([]*versionsv1.VersionReference, 0, len(item.References))
	for _, ref := range item.References {
		references = append(references, &versionsv1.VersionReference{
			Kind: ref.Kind, OwnerLibraryId: ref.OwnerLibraryID, OwnerVersion: ref.OwnerVersion,
			OwnerPath: ref.OwnerPath, ImportSpecifier: ref.ImportSpecifier, Evidence: ref.Evidence,
			OwnerScenario: ref.OwnerScenario, AdoptionId: ref.AdoptionID,
		})
	}
	return &versionsv1.CleanupItem{
		Version:  &versionsv1.RetireCandidate{ComponentId: item.Candidate.ComponentID, LibraryId: item.Candidate.LibraryID, Version: item.Candidate.Version, Status: item.Candidate.Status},
		Eligible: item.Eligible, Reason: item.Reason, AdoptionCount: int32(item.AdoptionCount), DependencyCount: int32(item.DependencyCount), AgeDays: int32(item.AgeDays),
		References: references,
	}
}

func (h *connectHandler) PlanCleanup(ctx context.Context, req *connect.Request[versionsv1.PlanCleanupRequest]) (*connect.Response[versionsv1.PlanCleanupResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, hash, err := h.deps.Ledger.PlanCleanup(ctx, cleanupScopeFromProto(req.Msg.GetScope()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.PlanCleanupResponse{PlanHash: hash}
	for _, item := range items {
		out.Items = append(out.Items, cleanupItemToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CleanupVersions(ctx context.Context, req *connect.Request[versionsv1.CleanupVersionsRequest]) (*connect.Response[versionsv1.CleanupVersionsResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	items, hash, retired, err := h.deps.Ledger.CleanupVersions(ctx, cleanupScopeFromProto(req.Msg.GetScope()), req.Msg.GetPlanHash(), req.Msg.GetConfirm())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &versionsv1.CleanupVersionsResponse{PlanHash: hash, RetiredCount: int32(retired), Applied: req.Msg.GetConfirm()}
	for _, item := range items {
		out.Items = append(out.Items, cleanupItemToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CleanupDraft(ctx context.Context, req *connect.Request[versionsv1.CleanupDraftRequest]) (*connect.Response[versionsv1.CleanupDraftResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	item, err := h.deps.Ledger.CleanupDraft(ctx, req.Msg.GetComponentId(), int(req.Msg.GetOlderThanDays()), req.Msg.GetConfirm())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&versionsv1.CleanupDraftResponse{Item: cleanupItemToProto(item), Applied: req.Msg.GetConfirm() && item.Eligible}), nil
}

func (h *connectHandler) ListVersionLedger(ctx context.Context, req *connect.Request[versionsv1.ListVersionLedgerRequest]) (*connect.Response[versionsv1.ListVersionLedgerResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version ledger is not configured"))
	}
	items, err := h.deps.Ledger.ListWindow(ctx, req.Msg.GetLibraryId(), req.Msg.GetWindow())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.ListVersionLedgerResponse{}
	for _, item := range items {
		out.Rows = append(out.Rows, &versionsv1.VersionLedgerRow{LibraryId: item.LibraryID, Version: item.Version, CreatedAt: item.CreatedAt.Format(time.RFC3339Nano), ReleasedAt: item.ReleasedAt.Format(time.RFC3339Nano), RetiredAt: item.RetiredAt.Format(time.RFC3339Nano), LifecycleState: item.LifecycleState, GatePassCount: int32(item.GatePassCount), GateFailCount: int32(item.GateFailCount), TestRuns: int32(item.TestRuns), TestPassRate: item.TestPassRate, AdoptionCurrent: int32(item.AdoptionCurrent), AdoptionPeak: int32(item.AdoptionPeak), FileCount: int32(item.FileCount), LinesOfCode: int32(item.LinesOfCode), DependencyCount: int32(item.DependencyCount), Presence: item.Presence})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) transition(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest], state string) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version lifecycle is not configured"))
	}
	item, err := h.deps.Ledger.Transition(ctx, req.Msg.GetComponentId(), req.Msg.GetVersion(), state, req.Msg.GetConfirm(), req.Msg.GetPlanHash())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&versionsv1.VersionLifecycleResponse{Version: &versionsv1.RetireCandidate{ComponentId: item.ComponentID, LibraryId: item.LibraryID, Version: item.Version, Status: item.Status}, LifecycleState: state}), nil
}

func (h *connectHandler) DeprecateVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "deprecated")
}

func (h *connectHandler) ArchiveVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "archived")
}

func (h *connectHandler) RetireVersion(ctx context.Context, req *connect.Request[versionsv1.VersionLifecycleRequest]) (*connect.Response[versionsv1.VersionLifecycleResponse], error) {
	return h.transition(ctx, req, "retired")
}

func (h *connectHandler) MaterializeVersion(ctx context.Context, req *connect.Request[versionsv1.MaterializeVersionRequest]) (*connect.Response[versionsv1.MaterializeVersionResponse], error) {
	if h.deps.Components == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component materializer is not configured"))
	}
	if !req.Msg.GetAll() && (req.Msg.GetComponentId() == "" || req.Msg.GetVersion() == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("component_id and version are required unless all=true"))
	}
	var targets []struct{ componentID, libraryID, version string }
	if req.Msg.GetAll() {
		assets, err := h.deps.Components.List(ctx, components.SearchQuery{Limit: 2000})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		for _, asset := range assets {
			versions, err := h.deps.Components.ListVersions(ctx, asset.ID, 2000)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			for _, version := range versions {
				targets = append(targets, struct{ componentID, libraryID, version string }{asset.ID, asset.LibraryID, version.Version})
			}
		}
	} else {
		asset, err := h.deps.Components.Get(ctx, req.Msg.GetComponentId())
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		targets = append(targets, struct{ componentID, libraryID, version string }{asset.ID, asset.LibraryID, req.Msg.GetVersion()})
	}
	materializer := h.deps.Materializer
	if materializer == nil {
		var ok bool
		materializer, ok = h.deps.Components.(components.Materializer)
		if !ok {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component materializer is not configured"))
		}
	}
	out := &versionsv1.MaterializeVersionResponse{}
	for _, target := range targets {
		result, err := materializer.EnsureMaterialized(ctx, target.componentID, target.version, req.Msg.GetInto())
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		v, err := h.deps.Components.GetVersion(ctx, target.componentID, target.version)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Versions = append(out.Versions, &versionsv1.MaterializedVersion{ComponentId: target.componentID, LibraryId: target.libraryID, Version: target.version, Directory: result.Directory, FilesWritten: int32(result.FilesWritten), AlreadyPresent: result.AlreadyPresent || v.Presence == "materialized"})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ReconcilePresence(ctx context.Context, req *connect.Request[versionsv1.ReconcilePresenceRequest]) (*connect.Response[versionsv1.ReconcilePresenceResponse], error) {
	if h.deps.Components == nil || h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("presence reconciliation is not configured"))
	}
	materializer := h.deps.Materializer
	if materializer == nil {
		var ok bool
		materializer, ok = h.deps.Components.(components.Materializer)
		if !ok {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component materializer is not configured"))
		}
	}
	assets, err := h.deps.Components.List(ctx, components.SearchQuery{Limit: 2000})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	allCandidates, err := h.deps.Ledger.RetireCandidates(ctx, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	candidatesByComponent := make(map[string]map[string]versionledger.Candidate)
	for _, candidate := range allCandidates {
		byVersion := candidatesByComponent[candidate.ComponentID]
		if byVersion == nil {
			byVersion = make(map[string]versionledger.Candidate)
			candidatesByComponent[candidate.ComponentID] = byVersion
		}
		byVersion[candidate.Version] = candidate
	}
	response := &versionsv1.ReconcilePresenceResponse{Applied: req.Msg.GetApply()}
	for _, asset := range assets {
		if req.Msg.GetComponentId() != "" && req.Msg.GetComponentId() != asset.ID && req.Msg.GetComponentId() != asset.LibraryID {
			continue
		}
		candidateByVersion := candidatesByComponent[asset.ID]
		rows, err := h.deps.Components.ListVersions(ctx, asset.ID, 2000)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		for _, row := range rows {
			candidate := &versionsv1.RetireCandidate{ComponentId: asset.ID, LibraryId: asset.LibraryID, Version: row.Version, Status: string(row.Status)}
			if _, safe := candidateByVersion[row.Version]; safe && row.Presence != "evicted" {
				response.Evict = append(response.Evict, candidate)
				if req.Msg.GetApply() {
					items, planHash, planErr := h.deps.Ledger.PlanCleanup(ctx, versionledger.CleanupScope{ComponentID: asset.ID})
					if planErr != nil {
						return nil, connect.NewError(connect.CodeInternal, planErr)
					}
					if _, transitionErr := h.deps.Ledger.Transition(ctx, asset.ID, row.Version, "archived", true, planHash); transitionErr != nil {
						_ = items
						return nil, connect.NewError(connect.CodeFailedPrecondition, transitionErr)
					}
				}
				continue
			}
			if _, safe := candidateByVersion[row.Version]; !safe && row.Presence == "evicted" {
				response.Materialize = append(response.Materialize, candidate)
				if req.Msg.GetApply() {
					if _, materializeErr := materializer.EnsureMaterialized(ctx, asset.ID, row.Version, ""); materializeErr != nil {
						return nil, connect.NewError(connect.CodeFailedPrecondition, materializeErr)
					}
				}
				continue
			}
			response.Unchanged = append(response.Unchanged, candidate)
		}
	}
	return connect.NewResponse(response), nil
}

func archiveResponse(summary versionledger.ArchiveSummary) *versionsv1.ArchiveResponse {
	counts := make(map[string]int32, len(summary.RowCounts))
	for table, count := range summary.RowCounts {
		counts[table] = int32(count)
	}
	return &versionsv1.ArchiveResponse{Path: summary.Path, SchemaVersion: int32(summary.SchemaVersion), RowCounts: counts, Checksum: summary.Checksum}
}

func (h *connectHandler) ExportArchive(ctx context.Context, req *connect.Request[versionsv1.ArchiveRequest]) (*connect.Response[versionsv1.ArchiveResponse], error) {
	if h.deps.Ledger == nil || req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("archive path is required"))
	}
	summary, err := h.deps.Ledger.ExportArchive(ctx, req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(archiveResponse(summary)), nil
}

func (h *connectHandler) ImportArchive(ctx context.Context, req *connect.Request[versionsv1.ImportArchiveRequest]) (*connect.Response[versionsv1.ArchiveResponse], error) {
	if h.deps.Ledger == nil || req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("archive path is required"))
	}
	summary, err := h.deps.Ledger.ImportArchive(ctx, req.Msg.GetPath(), req.Msg.GetOverwrite())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(archiveResponse(summary)), nil
}

func (h *connectHandler) Doctor(ctx context.Context, _ *connect.Request[versionsv1.DoctorRequest]) (*connect.Response[versionsv1.DoctorResponse], error) {
	if h.deps.Ledger == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version ledger is not configured"))
	}
	issues, err := h.deps.Ledger.Doctor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &versionsv1.DoctorResponse{}
	for _, issue := range issues {
		out.Issues = append(out.Issues, &versionsv1.DoctorIssue{LibraryId: issue.LibraryID, Version: issue.Version, Path: issue.Path, ExpectedSha256: issue.Expected, ActualSha256: issue.Actual, Reason: issue.Reason})
	}
	return connect.NewResponse(out), nil
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

func (h *connectHandler) ListVersions(ctx context.Context, req *connect.Request[versionsv1.ListVersionsRequest]) (*connect.Response[versionsv1.ListVersionsResponse], error) {
	if req.Msg.GetAll() {
		if h.deps.Components == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("catalog is not configured for all-version listing"))
		}
		assets, err := h.deps.Components.List(ctx, components.SearchQuery{Limit: 2000})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out := &versionsv1.ListVersionsResponse{}
		limit := int(req.Msg.GetLimit())
		if limit <= 0 {
			limit = 2000
		}
		for _, asset := range assets {
			rows, err := h.deps.Components.ListVersions(ctx, asset.ID, limit)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			for _, row := range rows {
				out.Versions = append(out.Versions, versionToProto(versions.Version{
					ID: row.ID, ComponentID: row.ComponentID, LibraryID: row.LibraryID, Version: row.Version,
					Status: string(row.Status), SourcePath: row.SourcePath, ContentSHA256: row.ContentSHA256,
					ChangelogMD: row.ChangelogMD, RecordedAt: row.CreatedAt, CreatedAt: row.CreatedAt,
					ReleasedAt: row.ReleasedAt, RequiredTokens: row.RequiredTokens, RequiredTokenPatterns: row.RequiredTokenPatterns,
					Presence: row.Presence,
				}, false))
				if len(out.Versions) >= limit {
					return connect.NewResponse(out), nil
				}
			}
		}
		return connect.NewResponse(out), nil
	}
	out, err := h.deps.Service.List(ctx, versions.ListQuery{
		ComponentID: req.Msg.ComponentId,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("versions.ListVersions: %v", err)
		return nil, mapErr(err)
	}
	resp := &versionsv1.ListVersionsResponse{Versions: make([]*versionsv1.Version, 0, len(out))}
	for _, v := range out {
		resp.Versions = append(resp.Versions, versionToProto(v, false))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetVersion(ctx context.Context, req *connect.Request[versionsv1.GetVersionRequest]) (*connect.Response[versionsv1.GetVersionResponse], error) {
	v, err := h.deps.Service.Get(ctx, req.Msg.ComponentId, req.Msg.Version)
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.GetVersion(%q,%q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	resp := &versionsv1.GetVersionResponse{Version: versionToProto(v, false)}
	if req.Msg.IncludeContent {
		resp.Content = v.Content
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DiffVersions(ctx context.Context, req *connect.Request[versionsv1.DiffVersionsRequest]) (*connect.Response[versionsv1.DiffVersionsResponse], error) {
	result, err := h.deps.Service.Diff(ctx, versions.DiffInput{
		ComponentID: req.Msg.ComponentId,
		From:        req.Msg.From,
		To:          req.Msg.To,
	})
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.DiffVersions: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(diffToProto(result)), nil
}

func mapErr(err error) *connect.Error {
	if mapped := versions.ToConnectError(err); mapped != nil {
		return mapped
	}
	return connect.NewError(connect.CodeInternal, err)
}

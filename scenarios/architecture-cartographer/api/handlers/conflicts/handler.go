// Package conflicts is the Connect-RPC surface for the conflicts
// domain. Orchestrates the Detector registry by fetching the graph
// snapshot via graph.Service and the manifest via manifest.Service,
// then delegating to conflicts.Service.
package conflicts

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"

	"connectrpc.com/connect"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deps wires the seams the conflicts handler needs.
type Deps struct {
	Conflicts conflicts.Service
	Graph     graph.Service
	Manifest  manifest.Service
}

// Handler implements conflicts_v1connect.ConflictsServiceHandler.
type Handler struct {
	conflicts_v1connect.UnimplementedConflictsServiceHandler
	deps Deps
}

// NewHandler constructs the Connect handler.
func NewHandler(d Deps) *Handler { return &Handler{deps: d} }

var _ conflicts_v1connect.ConflictsServiceHandler = (*Handler)(nil)

func (h *Handler) DetectConflicts(ctx context.Context, req *connect.Request[conflictsv1.DetectConflictsRequest]) (*connect.Response[conflictsv1.DetectConflictsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	// Load the snapshot. v0.1 ALWAYS extracts a fresh one when
	// snapshot_id is not supplied — production users provide a fresh
	// id from `arch-cart graph extract`. Fetching the latest is
	// deferred to a later phase to avoid coupling on a "latest"
	// repository method that doesn't exist yet.
	var snap graph.GraphSnapshot
	if id := strings.TrimSpace(req.Msg.GetSnapshotId()); id != "" {
		s, err := h.deps.Graph.GetSnapshot(ctx, id)
		if err != nil {
			return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
		}
		snap = s
	} else {
		s, _, err := h.deps.Graph.ExtractGraph(ctx, graph.ExtractGraphInput{Scenario: scenario, IdempotencyKey: req.Msg.GetIdempotencyKey()})
		if err != nil {
			return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
		}
		snap = s
	}
	m, err := h.deps.Manifest.GetManifest(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(manifest.ErrorToConnectCode(err), err)
	}

	out, err := h.deps.Conflicts.DetectConflicts(ctx, conflicts.DetectOrchestrationInput{
		Scenario:       scenario,
		Snapshot:       snap,
		Manifest:       m,
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	resp := &conflictsv1.DetectConflictsResponse{}
	for _, c := range out {
		resp.Conflicts = append(resp.Conflicts, conflictToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) ListConflicts(ctx context.Context, req *connect.Request[conflictsv1.ListConflictsRequest]) (*connect.Response[conflictsv1.ListConflictsResponse], error) {
	filter := conflicts.ListConflictsFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		Types:    append([]string(nil), req.Msg.GetTypes()...),
		PageSize: int(req.Msg.GetPageSize()),
	}
	for _, s := range req.Msg.GetStatuses() {
		filter.Statuses = append(filter.Statuses, protoToStatus(s))
	}
	page, err := h.deps.Conflicts.ListConflicts(ctx, filter)
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	out := &conflictsv1.ListConflictsResponse{}
	for _, c := range page.Conflicts {
		out.Conflicts = append(out.Conflicts, conflictToProto(c))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetConflict(ctx context.Context, req *connect.Request[conflictsv1.GetConflictRequest]) (*connect.Response[conflictsv1.GetConflictResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	c, err := h.deps.Conflicts.GetConflict(ctx, id)
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&conflictsv1.GetConflictResponse{Conflict: conflictToProto(c)}), nil
}

func (h *Handler) AssignConflict(ctx context.Context, req *connect.Request[conflictsv1.AssignConflictRequest]) (*connect.Response[conflictsv1.AssignConflictResponse], error) {
	c, dry, err := h.deps.Conflicts.AssignConflict(ctx, req.Msg.GetId(), req.Msg.GetDomain(), req.Msg.GetNote(), req.Msg.GetDryRun() || dryRunHeaderFromHTTP(req.Header()))
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&conflictsv1.AssignConflictResponse{Conflict: conflictToProto(c), DryRun: dry}), nil
}

func (h *Handler) ResolveConflict(ctx context.Context, req *connect.Request[conflictsv1.ResolveConflictRequest]) (*connect.Response[conflictsv1.ResolveConflictResponse], error) {
	c, dry, applyDeferred, err := h.deps.Conflicts.ResolveConflict(ctx, req.Msg.GetId(), req.Msg.GetNote(), req.Msg.GetForce(), req.Msg.GetDryRun() || dryRunHeaderFromHTTP(req.Header()))
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&conflictsv1.ResolveConflictResponse{Conflict: conflictToProto(c), DryRun: dry, ApplyDeferred: applyDeferred}), nil
}

func (h *Handler) ReopenConflict(ctx context.Context, req *connect.Request[conflictsv1.ReopenConflictRequest]) (*connect.Response[conflictsv1.ReopenConflictResponse], error) {
	c, dry, err := h.deps.Conflicts.ReopenConflict(ctx, req.Msg.GetId(), req.Msg.GetNote(), req.Msg.GetDryRun() || dryRunHeaderFromHTTP(req.Header()))
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&conflictsv1.ReopenConflictResponse{Conflict: conflictToProto(c), DryRun: dry}), nil
}

func (h *Handler) ValidateConflicts(ctx context.Context, req *connect.Request[conflictsv1.ValidateConflictsRequest]) (*connect.Response[conflictsv1.ValidateConflictsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	out, clean, err := h.deps.Conflicts.ValidateConflicts(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(conflicts.ErrorToConnectCode(err), err)
	}
	resp := &conflictsv1.ValidateConflictsResponse{Clean: clean}
	for _, c := range out {
		resp.Conflicts = append(resp.Conflicts, conflictToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) ListDetectors(ctx context.Context, _ *connect.Request[conflictsv1.ListDetectorsRequest]) (*connect.Response[conflictsv1.ListDetectorsResponse], error) {
	descs := h.deps.Conflicts.ListDetectors(ctx)
	out := &conflictsv1.ListDetectorsResponse{}
	for _, d := range descs {
		out.Detectors = append(out.Detectors, &conflictsv1.DetectorDescriptor{
			Name:        d.Name,
			Description: d.Description,
			Stability:   d.Stability,
			EmitsTypes:  append([]string(nil), d.EmitsTypes...),
		})
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ListResolvers(ctx context.Context, _ *connect.Request[conflictsv1.ListResolversRequest]) (*connect.Response[conflictsv1.ListResolversResponse], error) {
	descs := h.deps.Conflicts.ListResolvers(ctx)
	out := &conflictsv1.ListResolversResponse{}
	for _, d := range descs {
		desc := &conflictsv1.ResolverDescriptor{
			Name:          d.Name,
			Description:   d.Description,
			Stability:     d.Stability,
			RequiresApply: d.RequiresApply,
		}
		for _, k := range d.HandlesKinds {
			desc.HandlesKinds = append(desc.HandlesKinds, fixKindToProto(k))
		}
		out.Resolvers = append(out.Resolvers, desc)
	}
	return connect.NewResponse(out), nil
}

// -------------------------- proto<->domain --------------------------

// dryRunHeaderFromHTTP inspects the request header for X-Dry-Run: true.
func dryRunHeaderFromHTTP(h interface{ Get(string) string }) bool {
	return h.Get("X-Dry-Run") == "true"
}

func conflictToProto(c conflicts.Conflict) *conflictsv1.Conflict {
	out := &conflictsv1.Conflict{
		Id:             c.ID,
		Scenario:       c.Scenario,
		Detector:       c.Detector,
		Type:           c.Type,
		Subtype:        c.Subtype,
		Severity:       severityToProto(c.Severity),
		Locations:      append([]string(nil), c.Locations...),
		Domains:        append([]string(nil), c.Domains...),
		Status:         statusToProto(c.Status),
		AssignedDomain: c.AssignedDomain,
		ResolutionNote: c.ResolutionNote,
		SnapshotId:     c.SnapshotID,
	}
	if !c.DetectedAt.IsZero() {
		out.DetectedAt = timestamppb.New(c.DetectedAt)
	}
	if !c.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(c.UpdatedAt)
	}
	for _, e := range c.Evidence {
		out.Evidence = append(out.Evidence, &conflictsv1.ConflictEvidence{
			Kind:    e.Kind,
			Summary: e.Summary,
			Locator: e.Locator,
			Payload: append([]byte(nil), e.Payload...),
		})
	}
	for _, f := range c.SuggestedFixes {
		out.SuggestedFixes = append(out.SuggestedFixes, &conflictsv1.Fix{
			Id:         f.ID,
			Kind:       fixKindToProto(f.Kind),
			Resolver:   f.Resolver,
			Summary:    f.Summary,
			Payload:    append([]byte(nil), f.Payload...),
			Confidence: f.Confidence,
		})
	}
	return out
}

func severityToProto(s conflicts.Severity) conflictsv1.Severity {
	switch s {
	case conflicts.SeverityInfo:
		return conflictsv1.Severity_SEVERITY_INFO
	case conflicts.SeverityWarn:
		return conflictsv1.Severity_SEVERITY_WARN
	case conflicts.SeverityError:
		return conflictsv1.Severity_SEVERITY_ERROR
	case conflicts.SeverityBlocker:
		return conflictsv1.Severity_SEVERITY_BLOCKER
	default:
		return conflictsv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func statusToProto(s conflicts.ResolutionStatus) conflictsv1.ResolutionStatus {
	switch s {
	case conflicts.ResolutionStatusDetected:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_DETECTED
	case conflicts.ResolutionStatusAssigned:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_ASSIGNED
	case conflicts.ResolutionStatusSplit:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_SPLIT
	case conflicts.ResolutionStatusResolved:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_RESOLVED
	case conflicts.ResolutionStatusValidated:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_VALIDATED
	case conflicts.ResolutionStatusCommitted:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_COMMITTED
	case conflicts.ResolutionStatusForceResolved:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_FORCE_RESOLVED
	default:
		return conflictsv1.ResolutionStatus_RESOLUTION_STATUS_UNSPECIFIED
	}
}

func protoToStatus(s conflictsv1.ResolutionStatus) conflicts.ResolutionStatus {
	switch s {
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_DETECTED:
		return conflicts.ResolutionStatusDetected
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_ASSIGNED:
		return conflicts.ResolutionStatusAssigned
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_SPLIT:
		return conflicts.ResolutionStatusSplit
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_RESOLVED:
		return conflicts.ResolutionStatusResolved
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_VALIDATED:
		return conflicts.ResolutionStatusValidated
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_COMMITTED:
		return conflicts.ResolutionStatusCommitted
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_FORCE_RESOLVED:
		return conflicts.ResolutionStatusForceResolved
	default:
		return conflicts.ResolutionStatusDetected
	}
}

func fixKindToProto(k conflicts.FixKind) conflictsv1.FixKind {
	switch k {
	case conflicts.FixKindMoveFile:
		return conflictsv1.FixKind_FIX_KIND_MOVE_FILE
	case conflicts.FixKindReassignDomain:
		return conflictsv1.FixKind_FIX_KIND_REASSIGN_DOMAIN
	case conflicts.FixKindBreakCycle:
		return conflictsv1.FixKind_FIX_KIND_BREAK_CYCLE
	case conflicts.FixKindAddDependency:
		return conflictsv1.FixKind_FIX_KIND_ADD_DEPENDENCY
	case conflicts.FixKindAddTransitional:
		return conflictsv1.FixKind_FIX_KIND_ADD_TRANSITIONAL
	default:
		return conflictsv1.FixKind_FIX_KIND_UNSPECIFIED
	}
}

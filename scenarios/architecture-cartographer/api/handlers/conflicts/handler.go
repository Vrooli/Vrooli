// Package conflicts is the Connect-RPC surface for the conflicts
// domain. Orchestrates the Detector registry by fetching the graph
// snapshot via graph.Service and the derived domain map via
// domains.Service, then delegating to conflicts.Service.
package conflicts

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/suppressions"

	"connectrpc.com/connect"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deps wires the seams the conflicts handler needs.
type Deps struct {
	Conflicts    conflicts.Service
	Graph        graph.Service
	Domains      domains.Service
	Signals      signals.Service
	Suppressions suppressions.Provider
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
	dmap, err := h.deps.Domains.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(domains.ErrorToConnectCode(err), err)
	}

	// Active in-repo suppression markers (best-effort: a scan failure must
	// not block detection — findings are simply reported unsuppressed).
	var markers []suppressions.Marker
	if h.deps.Suppressions != nil {
		if ms, sErr := h.deps.Suppressions.Active(ctx, scenario); sErr == nil {
			markers = ms
		}
	}

	var verdictProvider conflicts.VerdictProvider
	if h.deps.Signals != nil {
		verdictProvider = NewSignalsVerdictAdapter(h.deps.Signals)
	}
	out, err := h.deps.Conflicts.DetectConflicts(ctx, conflicts.DetectOrchestrationInput{
		Scenario:        scenario,
		Snapshot:        snap,
		DomainMap:       dmap,
		IdempotencyKey:  req.Msg.GetIdempotencyKey(),
		Suppressions:    markers,
		VerdictProvider: verdictProvider,
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
			Name:         d.Name,
			Description:  d.Description,
			Stability:    d.Stability,
			EmitsTypes:   append([]string(nil), d.EmitsTypes...),
			FindingClass: findingClassToProto(d.Class),
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

func conflictToProto(c conflicts.Conflict) *sharedv1.Conflict {
	out := &sharedv1.Conflict{
		Id:                c.ID,
		StableId:          c.StableID,
		InstanceId:        c.InstanceID,
		Scenario:          c.Scenario,
		Detector:          c.Detector,
		Type:              c.Type,
		Subtype:           c.Subtype,
		Severity:          severityToProto(c.Severity),
		FindingClass:      findingClassToProto(c.FindingClass),
		Locations:         append([]string(nil), c.Locations...),
		Domains:           append([]string(nil), c.Domains...),
		SnapshotId:        c.SnapshotID,
		Suppressed:        c.Suppressed,
		SuppressionReason: c.SuppressionReason,
	}
	if !c.DetectedAt.IsZero() {
		out.DetectedAt = timestamppb.New(c.DetectedAt)
	}
	if !c.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(c.UpdatedAt)
	}
	for _, e := range c.Evidence {
		out.Evidence = append(out.Evidence, &sharedv1.ConflictEvidence{
			Kind:    e.Kind,
			Summary: e.Summary,
			Locator: e.Locator,
			Payload: append([]byte(nil), e.Payload...),
		})
	}
	for _, f := range c.SuggestedFixes {
		out.SuggestedFixes = append(out.SuggestedFixes, &sharedv1.Fix{
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

func findingClassToProto(c conflicts.FindingClass) sharedv1.FindingClass {
	switch c {
	case conflicts.FindingClassDeterministic:
		return sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC
	case conflicts.FindingClassHeuristic:
		return sharedv1.FindingClass_FINDING_CLASS_HEURISTIC
	default:
		return sharedv1.FindingClass_FINDING_CLASS_UNSPECIFIED
	}
}

func severityToProto(s conflicts.Severity) sharedv1.Severity {
	switch s {
	case conflicts.SeverityInfo:
		return sharedv1.Severity_SEVERITY_INFO
	case conflicts.SeverityWarn:
		return sharedv1.Severity_SEVERITY_WARN
	case conflicts.SeverityError:
		return sharedv1.Severity_SEVERITY_ERROR
	case conflicts.SeverityBlocker:
		return sharedv1.Severity_SEVERITY_BLOCKER
	default:
		return sharedv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func fixKindToProto(k conflicts.FixKind) sharedv1.FixKind {
	switch k {
	case conflicts.FixKindMoveFile:
		return sharedv1.FixKind_FIX_KIND_MOVE_FILE
	case conflicts.FixKindReassignDomain:
		return sharedv1.FixKind_FIX_KIND_REASSIGN_DOMAIN
	case conflicts.FixKindBreakCycle:
		return sharedv1.FixKind_FIX_KIND_BREAK_CYCLE
	case conflicts.FixKindAddDependency:
		return sharedv1.FixKind_FIX_KIND_ADD_DEPENDENCY
	case conflicts.FixKindAddTransitional:
		return sharedv1.FixKind_FIX_KIND_ADD_TRANSITIONAL
	default:
		return sharedv1.FixKind_FIX_KIND_UNSPECIFIED
	}
}

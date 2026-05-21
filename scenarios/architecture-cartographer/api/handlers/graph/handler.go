// Package graph is the Connect-RPC surface for the graph domain.
// Translates between the proto wire types and the domain types in
// internal/graph; applies the error-mapping policy; honours X-Dry-Run
// for ClearGraphSnapshots.
package graph

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"architecture-cartographer/internal/graph"

	"connectrpc.com/connect"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements graph_v1connect.GraphServiceHandler.
type Handler struct {
	graph_v1connect.UnimplementedGraphServiceHandler
	svc graph.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc graph.Service) *Handler { return &Handler{svc: svc} }

var _ graph_v1connect.GraphServiceHandler = (*Handler)(nil)

func (h *Handler) ExtractGraph(ctx context.Context, req *connect.Request[graphv1.ExtractGraphRequest]) (*connect.Response[graphv1.ExtractGraphResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	in := graph.ExtractGraphInput{
		Scenario:       scenario,
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	}
	for _, l := range req.Msg.GetLanguages() {
		in.Languages = append(in.Languages, protoToLanguage(l))
	}
	snap, fromCache, err := h.svc.ExtractGraph(ctx, in)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.ExtractGraphResponse{
		Snapshot:  snapshotToProto(snap),
		FromCache: fromCache,
	}), nil
}

func (h *Handler) GetGraphSnapshot(ctx context.Context, req *connect.Request[graphv1.GetGraphSnapshotRequest]) (*connect.Response[graphv1.GetGraphSnapshotResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	snap, err := h.svc.GetSnapshot(ctx, id)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.GetGraphSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

func (h *Handler) ListGraphSnapshots(ctx context.Context, req *connect.Request[graphv1.ListGraphSnapshotsRequest]) (*connect.Response[graphv1.ListGraphSnapshotsResponse], error) {
	filter := graph.ListSnapshotsFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		PageSize: int(req.Msg.GetPageSize()),
	}
	page, err := h.svc.ListSnapshots(ctx, filter)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	out := &graphv1.ListGraphSnapshotsResponse{}
	for _, s := range page.Snapshots {
		out.Snapshots = append(out.Snapshots, snapshotToProto(s))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ClearGraphSnapshots(ctx context.Context, req *connect.Request[graphv1.ClearGraphSnapshotsRequest]) (*connect.Response[graphv1.ClearGraphSnapshotsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	dryRun := req.Msg.GetDryRun() || req.Header().Get("X-Dry-Run") == "true"
	deleted, dry, err := h.svc.ClearSnapshots(ctx, scenario, dryRun)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.ClearGraphSnapshotsResponse{
		Deleted: int32(deleted),
		DryRun:  dry,
	}), nil
}

func (h *Handler) ExportGraph(ctx context.Context, req *connect.Request[graphv1.ExportGraphRequest]) (*connect.Response[graphv1.ExportGraphResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	snap, err := h.svc.GetSnapshot(ctx, id)
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&graphv1.ExportGraphResponse{
		Payload:     payload,
		ContentType: "application/json",
	}), nil
}

// -------------------------- proto<->domain --------------------------

func protoToLanguage(l graphv1.Language) graph.Language {
	switch l {
	case graphv1.Language_LANGUAGE_GO:
		return graph.LanguageGo
	case graphv1.Language_LANGUAGE_TYPESCRIPT:
		return graph.LanguageTypeScript
	default:
		return graph.LanguageUnspecified
	}
}

func languageToProto(l graph.Language) graphv1.Language {
	switch l {
	case graph.LanguageGo:
		return graphv1.Language_LANGUAGE_GO
	case graph.LanguageTypeScript:
		return graphv1.Language_LANGUAGE_TYPESCRIPT
	default:
		return graphv1.Language_LANGUAGE_UNSPECIFIED
	}
}

func snapshotToProto(s graph.GraphSnapshot) *graphv1.GraphSnapshot {
	out := &graphv1.GraphSnapshot{
		Id:           s.ID,
		Scenario:     s.Scenario,
		ContentHash:  s.ContentHash,
		ExtractionMs: s.ExtractionMS,
	}
	if !s.ExtractedAt.IsZero() {
		out.ExtractedAt = timestamppb.New(s.ExtractedAt)
	}
	for _, l := range s.Languages {
		out.Languages = append(out.Languages, languageToProto(l))
	}
	for _, f := range s.Files {
		out.Files = append(out.Files, &graphv1.FileNode{
			Id:        f.ID,
			Path:      f.Path,
			PackageId: f.PackageID,
			Language:  languageToProto(f.Language),
			Lines:     int32(f.Lines),
			IsTest:    f.IsTest,
		})
	}
	for _, p := range s.Packages {
		out.Packages = append(out.Packages, &graphv1.PackageNode{
			Id:         p.ID,
			ImportPath: p.ImportPath,
			Directory:  p.Directory,
			Language:   languageToProto(p.Language),
			Internal:   p.Internal,
		})
	}
	for _, sym := range s.Symbols {
		out.Symbols = append(out.Symbols, &graphv1.SymbolNode{
			Id:        sym.ID,
			Name:      sym.Name,
			PackageId: sym.PackageID,
			FileId:    sym.FileID,
			Kind:      sym.Kind,
			Exported:  sym.Exported,
		})
	}
	for _, e := range s.Imports {
		out.Imports = append(out.Imports, &graphv1.ImportEdge{
			From:        e.From,
			ToPackageId: e.ToPackageID,
			SymbolIds:   append([]string(nil), e.SymbolIDs...),
			TestOnly:    e.TestOnly,
		})
	}
	return out
}

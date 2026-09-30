package scenarios

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/scenarios"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
)

// ArtifactsService is the subset of *artifacts.Service the scenarios
// connect handler invokes for the streaming Generate / Clear RPCs.
type ArtifactsService interface {
	GenerateForScenarioStream(ctx context.Context, root string, onProgress func(flowID string, report artifacts.Report, err error) error) error
	ClearForScenario(root string) ([]artifacts.ClearResult, error)
}

// StreamDeps bundles the artifacts service alongside the scenarios
// service so the connect handler can fan a single GenerateScenario call
// out into per-flow stream messages.
type StreamDeps struct {
	Scenarios Service
	Artifacts ArtifactsService
	Logger    *log.Logger
}

type streamHandler struct {
	*connectHandler
	stream StreamDeps
}

// NewStreamHandler wraps a connectHandler with the streaming RPCs that
// need an artifacts service. The combined handler implements the full
// ScenariosServiceHandler interface.
func NewStreamHandler(d StreamDeps) *streamHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &streamHandler{
		connectHandler: NewConnectHandler(Deps{Service: d.Scenarios, Logger: d.Logger}),
		stream:         d,
	}
}

func (h *streamHandler) GenerateScenarioArtifacts(ctx context.Context, req *connect.Request[scenariosv1.GenerateScenarioArtifactsRequest], stream *connect.ServerStream[scenariosv1.GenerateScenarioArtifactsResponse]) error {
	id := req.Msg.GetScenarioId()
	detail, err := h.stream.Scenarios.Detail(id)
	if err != nil {
		return scenarios.ToConnectError(err)
	}
	streamErr := h.stream.Artifacts.GenerateForScenarioStream(ctx, detail.Path, func(flowID string, report artifacts.Report, perFlowErr error) error {
		msg := &scenariosv1.GenerateScenarioArtifactsResponse{FlowId: flowID}
		if perFlowErr != nil {
			msg.ErrorMessage = perFlowErr.Error()
		} else {
			msg.Report = artifactReportToProto(report)
		}
		return stream.Send(msg)
	})
	if streamErr != nil {
		if errors.Is(streamErr, context.Canceled) {
			return connect.NewError(connect.CodeCanceled, streamErr)
		}
		h.stream.Logger.Printf("scenarios.GenerateScenarioArtifacts(%q): %v", id, streamErr)
		return scenarios.ToConnectError(streamErr)
	}
	return nil
}

func (h *streamHandler) ClearScenarioArtifacts(_ context.Context, req *connect.Request[scenariosv1.ClearScenarioArtifactsRequest]) (*connect.Response[scenariosv1.ClearScenarioArtifactsResponse], error) {
	id := req.Msg.GetScenarioId()
	detail, err := h.stream.Scenarios.Detail(id)
	if err != nil {
		return nil, scenarios.ToConnectError(err)
	}
	results, err := h.stream.Artifacts.ClearForScenario(detail.Path)
	if err != nil {
		h.stream.Logger.Printf("scenarios.ClearScenarioArtifacts(%q): %v", id, err)
		return nil, artifacts.ToConnectError(err)
	}
	out := &scenariosv1.ClearScenarioArtifactsResponse{
		Flows: make([]*artifactsv1.ClearArtifactsResponse, 0, len(results)),
	}
	for _, r := range results {
		out.Flows = append(out.Flows, &artifactsv1.ClearArtifactsResponse{
			FlowId:  r.FlowID,
			Removed: append([]string(nil), r.Removed...),
		})
	}
	return connect.NewResponse(out), nil
}

// artifactReportToProto duplicates the conversion that lives in
// handlers/artifacts so this package has no inward-pointing dep on the
// artifacts handlers package. Kept tight — fewer than 20 lines.
func artifactReportToProto(r artifacts.Report) *artifactsv1.ArtifactReport {
	files := make([]*artifactsv1.ArtifactFile, 0, len(r.Files))
	for _, f := range r.Files {
		files = append(files, &artifactsv1.ArtifactFile{
			Path:   f.Path,
			Exists: f.Exists,
			Size:   f.Size,
			Mtime:  artifactTimeToProto(f.MTime),
		})
	}
	return &artifactsv1.ArtifactReport{
		FlowId:       r.FlowID,
		ScenarioPath: r.ScenarioPath,
		GeneratedDir: r.GeneratedDir,
		Status:       artifactStatusToProto(r.Status),
		Files:        files,
		Missing:      append([]string(nil), r.Missing...),
	}
}

func artifactTimeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func artifactStatusToProto(s artifacts.Status) artifactsv1.ArtifactStatus {
	switch s {
	case artifacts.StatusFresh:
		return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_FRESH
	case artifacts.StatusMissing:
		return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_MISSING
	}
	return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_UNSPECIFIED
}

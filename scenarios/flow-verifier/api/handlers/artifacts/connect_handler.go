package artifacts

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
)

// ScenariosService is the subset of scenarios.Service this handler
// uses to resolve a flow id → scenario root when the request omits an
// explicit root.
type ScenariosService interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
}

// Deps wires the artifacts Connect handler's dependencies.
type Deps struct {
	Service   *artifacts.Service
	Scenarios ScenariosService
	Logger    *log.Logger
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

func (h *connectHandler) GetArtifactStatus(_ context.Context, req *connect.Request[artifactsv1.GetArtifactStatusRequest]) (*connect.Response[artifactsv1.GetArtifactStatusResponse], error) {
	root, err := h.resolveRoot(req.Msg.GetRoot(), req.Msg.GetScenarioId(), req.Msg.GetFlowId())
	if err != nil {
		return nil, h.resolveErr(err, req.Msg.GetFlowId())
	}
	report, err := h.deps.Service.Status(root, req.Msg.GetFlowId())
	if err != nil {
		return nil, h.serviceErr(err)
	}
	return connect.NewResponse(&artifactsv1.GetArtifactStatusResponse{Report: reportToProto(report)}), nil
}

func (h *connectHandler) GenerateArtifacts(ctx context.Context, req *connect.Request[artifactsv1.GenerateArtifactsRequest]) (*connect.Response[artifactsv1.GenerateArtifactsResponse], error) {
	root, err := h.resolveRoot(req.Msg.GetRoot(), req.Msg.GetScenarioId(), req.Msg.GetFlowId())
	if err != nil {
		return nil, h.resolveErr(err, req.Msg.GetFlowId())
	}
	report, err := h.deps.Service.Generate(ctx, root, req.Msg.GetFlowId())
	if err != nil {
		return nil, h.serviceErr(err)
	}
	return connect.NewResponse(&artifactsv1.GenerateArtifactsResponse{Report: reportToProto(report)}), nil
}

func (h *connectHandler) ClearArtifacts(_ context.Context, req *connect.Request[artifactsv1.ClearArtifactsRequest]) (*connect.Response[artifactsv1.ClearArtifactsResponse], error) {
	root, err := h.resolveRoot(req.Msg.GetRoot(), req.Msg.GetScenarioId(), req.Msg.GetFlowId())
	if err != nil {
		return nil, h.resolveErr(err, req.Msg.GetFlowId())
	}
	result, err := h.deps.Service.Clear(root, req.Msg.GetFlowId())
	if err != nil {
		return nil, h.serviceErr(err)
	}
	return connect.NewResponse(&artifactsv1.ClearArtifactsResponse{
		FlowId:  result.FlowID,
		Removed: append([]string(nil), result.Removed...),
	}), nil
}

func (h *connectHandler) resolveRoot(root, scenarioID, flowID string) (string, error) {
	if root != "" {
		return root, nil
	}
	if h.deps.Scenarios == nil {
		return "", errors.New("scenarios service not configured")
	}
	if scenarioID != "" {
		detail, err := h.deps.Scenarios.Detail(scenarioID)
		if err != nil {
			return "", err
		}
		return detail.Path, nil
	}
	all, err := h.deps.Scenarios.List()
	if err != nil {
		return "", err
	}
	for _, scenario := range all {
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := h.deps.Scenarios.Detail(scenario.ID)
		if err != nil {
			continue
		}
		for _, row := range detail.Flows {
			if row.FlowID == flowID {
				return scenario.Path, nil
			}
		}
	}
	return "", artifacts.ErrFlowNotFound
}

func (h *connectHandler) resolveErr(err error, flowID string) error {
	if errors.Is(err, scenarios.ErrScenarioNotFound) || errors.Is(err, artifacts.ErrFlowNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	h.deps.Logger.Printf("artifacts resolve(%q): %v", flowID, err)
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) serviceErr(err error) error {
	mapped := artifacts.ToConnectError(err)
	if connect.CodeOf(mapped) == connect.CodeInternal {
		h.deps.Logger.Printf("artifacts: %v", err)
	}
	return mapped
}

func reportToProto(r artifacts.Report) *artifactsv1.ArtifactReport {
	files := make([]*artifactsv1.ArtifactFile, 0, len(r.Files))
	for _, f := range r.Files {
		files = append(files, &artifactsv1.ArtifactFile{
			Path:   f.Path,
			Exists: f.Exists,
			Size:   f.Size,
			Mtime:  timeToProto(f.MTime),
		})
	}
	return &artifactsv1.ArtifactReport{
		FlowId:       r.FlowID,
		ScenarioPath: r.ScenarioPath,
		GeneratedDir: r.GeneratedDir,
		Status:       statusToProto(r.Status),
		Files:        files,
		Missing:      append([]string(nil), r.Missing...),
	}
}

func timeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func statusToProto(s artifacts.Status) artifactsv1.ArtifactStatus {
	switch s {
	case artifacts.StatusFresh:
		return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_FRESH
	case artifacts.StatusMissing:
		return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_MISSING
	}
	return artifactsv1.ArtifactStatus_ARTIFACT_STATUS_UNSPECIFIED
}

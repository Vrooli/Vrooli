package diagnostics

import (
	"audio-tools/internal/protoint"
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"audio-tools/internal/diagnostics"
	"audio-tools/internal/diagnostics/smokedata"
	"audio-tools/internal/protomap"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Deps.Logger is
// required (logx.Logger); a nil value panics.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("diagnostics.NewConnectHandler requires Deps.Logger")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RunSuite(ctx context.Context, req *connect.Request[diagv1.RunSuiteRequest]) (*connect.Response[diagv1.RunSuiteResponse], error) {
	if h.deps.Orchestrator == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("diagnostics orchestrator not configured"))
	}
	caps, err := capabilitiesFromProto(req.Msg.GetCapabilities())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := h.deps.Orchestrator.RunSuite(ctx, caps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&diagv1.RunSuiteResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) GetLastRun(_ context.Context, _ *connect.Request[diagv1.GetLastRunRequest]) (*connect.Response[diagv1.GetLastRunResponse], error) {
	if h.deps.Orchestrator == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("diagnostics orchestrator not configured"))
	}
	run := h.deps.Orchestrator.Last()
	return connect.NewResponse(&diagv1.GetLastRunResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) ListFixtures(_ context.Context, _ *connect.Request[diagv1.ListFixturesRequest]) (*connect.Response[diagv1.ListFixturesResponse], error) {
	wav := smokedata.SmokeWAV()
	text := smokedata.SmokeText()
	return connect.NewResponse(&diagv1.ListFixturesResponse{
		Fixtures: []*diagv1.Fixture{
			{
				Id: "smoke.wav", Capability: diagv1.Capability_CAPABILITY_STT,
				Description: "1s 16 kHz mono PCM tone (~440 Hz) used by the STT and Transcode steps.",
				SizeBytes:   int64(len(wav)), ContentType: "audio/wav",
			},
			{
				Id: "smoke_text.txt", Capability: diagv1.Capability_CAPABILITY_SUMMARIZE,
				Description: "Canned narrative used by the Summarize and TTS steps.",
				SizeBytes:   int64(len(text)), ContentType: "text/plain",
			},
		},
	}), nil
}

func capabilitiesFromProto(in []diagv1.Capability) ([]diagnostics.Capability, error) {
	out := make([]diagnostics.Capability, 0, len(in))
	for _, c := range in {
		switch c {
		case diagv1.Capability_CAPABILITY_STT:
			out = append(out, diagnostics.CapabilitySTT)
		case diagv1.Capability_CAPABILITY_TTS:
			out = append(out, diagnostics.CapabilityTTS)
		case diagv1.Capability_CAPABILITY_SUMMARIZE:
			out = append(out, diagnostics.CapabilitySummarize)
		case diagv1.Capability_CAPABILITY_TRANSCODE:
			out = append(out, diagnostics.CapabilityTranscode)
		default:
			return nil, fmt.Errorf("unknown capability: %s", c)
		}
	}
	return out, nil
}

func capabilityToProto(c diagnostics.Capability) diagv1.Capability {
	switch c {
	case diagnostics.CapabilitySTT:
		return diagv1.Capability_CAPABILITY_STT
	case diagnostics.CapabilityTTS:
		return diagv1.Capability_CAPABILITY_TTS
	case diagnostics.CapabilitySummarize:
		return diagv1.Capability_CAPABILITY_SUMMARIZE
	case diagnostics.CapabilityTranscode:
		return diagv1.Capability_CAPABILITY_TRANSCODE
	}
	return diagv1.Capability_CAPABILITY_UNSPECIFIED
}

func statusToProto(s diagnostics.Status) diagv1.SuiteOverall_Status {
	switch s {
	case diagnostics.StatusPass:
		return diagv1.SuiteOverall_STATUS_PASS
	case diagnostics.StatusPartial:
		return diagv1.SuiteOverall_STATUS_PARTIAL
	case diagnostics.StatusFail:
		return diagv1.SuiteOverall_STATUS_FAIL
	case diagnostics.StatusNever:
		return diagv1.SuiteOverall_STATUS_NEVER
	}
	return diagv1.SuiteOverall_STATUS_UNSPECIFIED
}

func runToProto(run diagnostics.Run) *diagv1.RunSuiteResult {
	if run.ID == "" {
		// Empty (never-run) sentinel.
		return &diagv1.RunSuiteResult{
			Overall: &diagv1.SuiteOverall{Status: diagv1.SuiteOverall_STATUS_NEVER},
		}
	}
	out := &diagv1.RunSuiteResult{
		RunId:            run.ID,
		StartedAtUnixMs:  run.StartedAt.UnixMilli(),
		FinishedAtUnixMs: run.FinishedAt.UnixMilli(),
		Steps:            make([]*diagv1.SuiteStepResult, 0, len(run.Steps)),
		Overall: &diagv1.SuiteOverall{
			Status:     statusToProto(run.Overall),
			PassCount:  protoint.FromInt(run.PassCount),
			FailCount:  protoint.FromInt(run.FailCount),
			TotalCount: protoint.FromInt(run.TotalCount),
		},
	}
	for _, s := range run.Steps {
		var tier commonv1.ProviderTier
		if s.ProviderTier != "" {
			tier = protomap.ProviderTierToProto(s.ProviderTier)
		}
		out.Steps = append(out.Steps, &diagv1.SuiteStepResult{
			Capability:       capabilityToProto(s.Capability),
			Ok:               s.OK,
			ErrorCode:        s.ErrorCode,
			ErrorMessage:     s.ErrorMessage,
			StartedAtUnixMs:  s.StartedAt.UnixMilli(),
			FinishedAtUnixMs: s.FinishedAt.UnixMilli(),
			ProviderTier:     tier,
			ProviderId:       s.ProviderID,
			ModelId:          s.ModelID,
			LatencyMs:        s.LatencyMs,
			Details:          s.Details,
		})
	}
	return out
}

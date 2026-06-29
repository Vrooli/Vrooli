package eval

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	inteval "audio-tools/internal/eval"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

type connectHandler struct{ deps Deps }

// NewConnectHandler builds the EvalService Connect handler. Deps.Logger and
// Deps.Clock are required seams; nil values panic.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("eval.NewConnectHandler requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("eval.NewConnectHandler requires Deps.Clock")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RunEval(ctx context.Context, req *connect.Request[evalv1.RunEvalRequest]) (*connect.Response[evalv1.RunEvalResponse], error) {
	if h.deps.Corpus == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("eval service not configured (no corpus/database)"))
	}
	if h.deps.NewProvider == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("eval requires a transcription provider (Whisper) — none configured"))
	}

	clips, err := h.loadClips(ctx, req.Msg.GetClipIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(clips) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("corpus is empty — record clips before running an eval"))
	}

	specs, err := h.buildSpecs(req.Msg.GetStrategies())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	opts := inteval.EvalOptions{
		ChunkMs:         int(req.Msg.GetChunkMs()),
		QualityPass:     true,
		RealtimeRepeats: int(req.Msg.GetRealtimeRepeats()),
	}
	report := inteval.RunEval(ctx, clips, specs, opts)
	return connect.NewResponse(&evalv1.RunEvalResponse{Report: reportToProto(report)}), nil
}

func reportToProto(r inteval.EvalReport) *evalv1.EvalReport {
	out := &evalv1.EvalReport{
		QualityMeasured: r.QualityMeasured,
		LatencyMeasured: r.LatencyMeasured,
		PerStrategy:     make([]*evalv1.StrategyReport, 0, len(r.PerStrategy)),
	}
	for _, s := range r.PerStrategy {
		out.PerStrategy = append(out.PerStrategy, strategyReportToProto(s))
	}
	return out
}

func strategyReportToProto(s inteval.StrategyReport) *evalv1.StrategyReport {
	sr := &evalv1.StrategyReport{
		Strategy:                 string(s.Strategy),
		Label:                    s.Label,
		Wer:                      s.WER,
		Substitutions:            int32(s.EditCounts.Substitutions),
		Insertions:               int32(s.EditCounts.Insertions),
		Deletions:                int32(s.EditCounts.Deletions),
		RefWords:                 int32(s.RefWords),
		WhisperCalls:             int32(s.WhisperCalls),
		WhisperAudioSeconds:      s.WhisperAudioSeconds,
		Rtf:                      s.RTF,
		FinalizationLatencyP50Ms: s.FinalizationLatencyP50Ms,
		FinalizationLatencyP95Ms: s.FinalizationLatencyP95Ms,
		PartialRevisions:         int32(s.PartialRevisions),
		PerClip:                  make([]*evalv1.ClipReport, 0, len(s.PerClip)),
	}
	for _, c := range s.PerClip {
		errStr := ""
		if c.Err != nil {
			errStr = c.Err.Error()
		}
		sr.PerClip = append(sr.PerClip, &evalv1.ClipReport{
			ClipId:                   c.ClipID,
			Reference:                c.Reference,
			Hypothesis:               c.Hypothesis,
			Wer:                      c.WER.Rate(),
			WhisperCalls:             int32(c.WhisperCalls),
			WhisperAudioSeconds:      c.WhisperAudioSeconds,
			Rtf:                      c.RTF,
			SegmentCount:             int32(c.SegmentCount),
			PartialRevisions:         int32(c.PartialRevisions),
			FinalizationLatencyP50Ms: c.FinalizationLatencyP50Ms(),
			FinalizationLatencyP95Ms: c.FinalizationLatencyP95Ms(),
			Error:                    errStr,
		})
	}
	return sr
}

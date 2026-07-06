// Package diagnostics runs the full audio-tools capability suite end-to-end.
//
// The orchestrator depends on narrow runner interfaces that the existing
// chain seams (sttchain.Chain, ttschain.Chain, summarizechain.Chain) and
// the internal/audio.TranscodeOpts function already satisfy. This keeps
// the orchestrator independent of transport and provider wiring so unit
// tests can substitute mocks per capability.
package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/clock"
	"audio-tools/internal/diagnostics/smokedata"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/quality"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	"audio-tools/internal/usagereport"

	"github.com/google/uuid"
)

// Capability identifies one suite step. Values are stable wire strings
// shared with the proto enum and the CLI --capability flag.
type Capability string

const (
	CapabilitySTT       Capability = "stt"
	CapabilityTTS       Capability = "tts"
	CapabilitySummarize Capability = "summarize"
	CapabilityTranscode Capability = "transcode"
)

// AllCapabilities is the canonical ordered list the suite runs when the
// caller does not narrow the request.
var AllCapabilities = []Capability{CapabilitySTT, CapabilityTTS, CapabilitySummarize, CapabilityTranscode}

// SttRunner is the seam for the STT step. *sttchain.Chain satisfies it.
type SttRunner interface {
	Execute(ctx context.Context, req sttchain.Request) (*sttchain.Result, error)
}

// TtsRunner is the seam for the TTS step. *ttschain.Chain satisfies it.
type TtsRunner interface {
	Execute(ctx context.Context, req ttschain.Request) (*ttschain.Result, error)
}

// SummaryRunner is the seam for the Summarize step. *summarizechain.Chain satisfies it.
type SummaryRunner interface {
	Execute(ctx context.Context, req summarizechain.Request) (*summarizechain.Result, error)
}

// Transcoder is the seam for the Transcode step. The internal/audio
// package's TranscodeOpts is wrapped to satisfy this in production.
type Transcoder interface {
	Transcode(ctx context.Context, audio []byte, outputFormat string) ([]byte, error)
}

// Status mirrors diagnostics_v1.SuiteOverall.Status using stable
// internal strings so callers that map to other transports do not have
// to import the generated proto.
type Status string

const (
	StatusNever   Status = "never"
	StatusPass    Status = "pass"
	StatusPartial Status = "partial"
	StatusFail    Status = "fail"
)

// StepResult is one capability outcome.
type StepResult struct {
	Capability   Capability
	OK           bool
	ErrorCode    string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
	ProviderTier string
	ProviderID   string
	ModelID      string
	LatencyMs    float64
	Details      map[string]string
}

// Run is the envelope persisted in the last-run store and returned by RunSuite.
type Run struct {
	ID         string
	StartedAt  time.Time
	FinishedAt time.Time
	Steps      []StepResult
	Overall    Status
	PassCount  int
	FailCount  int
	TotalCount int
}

// Deps wires the orchestrator. All four runners are required; tests
// pass mocks. The PerStepTimeout defaults to 30s and the Clock defaults
// to clock.System{}.
type Deps struct {
	STT            SttRunner
	TTS            TtsRunner
	Summarize      SummaryRunner
	Transcode      Transcoder
	Store          LastRunStore
	Clock          clock.Clock
	PerStepTimeout time.Duration
	// Usage, when non-nil, receives one row per executed step tagged
	// operation="diagnostics" so operator probes show up alongside real
	// traffic in the Usage page. Optional — nil disables recording.
	Usage usagereport.Recorder
	// NewRunID is overridable for deterministic tests; defaults to uuid.NewString.
	NewRunID func() string
	// STTConfig supplies the active stream/STT quality levers for the STT
	// readiness probe. nil falls back to documented defaults.
	STTConfig func(context.Context) sttpkg.StreamConfig
	Registry  *sttengine.Registry
}

// Orchestrator runs the suite and records the most recent result.
//
// Concurrent RunSuite calls serialize on runMu so a double-click does
// not stack provider load — the second caller waits for the first to
// finish and sees a fresh run rather than the cached result.
type Orchestrator struct {
	deps  Deps
	runMu sync.Mutex
}

// New returns a configured Orchestrator. Required deps that are nil are
// treated as "capability unavailable" — the corresponding step reports
// not_configured rather than panicking.
func New(deps Deps) *Orchestrator {
	if deps.Clock == nil {
		deps.Clock = clock.System{}
	}
	if deps.PerStepTimeout == 0 {
		deps.PerStepTimeout = 30 * time.Second
	}
	if deps.NewRunID == nil {
		deps.NewRunID = uuid.NewString
	}
	if deps.Store == nil {
		deps.Store = NewLastRunStore(10)
	}
	return &Orchestrator{deps: deps}
}

// Last returns the most recent recorded run, or a zero Run with empty
// ID when the suite has never executed.
func (o *Orchestrator) Last() Run { return o.deps.Store.Latest() }

// RunSuite executes the requested capabilities sequentially under
// per-step deadlines. An empty `capabilities` slice runs all known
// capabilities in AllCapabilities order. Unknown capability names
// return an error.
func (o *Orchestrator) RunSuite(ctx context.Context, capabilities []Capability) (Run, error) {
	caps, err := normalizeCapabilities(capabilities)
	if err != nil {
		return Run{}, err
	}
	o.runMu.Lock()
	defer o.runMu.Unlock()
	return o.runOnce(ctx, caps), nil
}

func (o *Orchestrator) runOnce(ctx context.Context, caps []Capability) Run {
	run := Run{
		ID:        o.deps.NewRunID(),
		StartedAt: o.deps.Clock.Now(),
		Steps:     make([]StepResult, 0, len(caps)),
	}
	for _, c := range caps {
		step := o.runStep(ctx, c)
		run.Steps = append(run.Steps, step)
		o.recordUsage(run.ID, step)
	}
	run.FinishedAt = o.deps.Clock.Now()
	run.TotalCount = len(run.Steps)
	for _, s := range run.Steps {
		if s.OK {
			run.PassCount++
		} else {
			run.FailCount++
		}
	}
	switch {
	case run.PassCount == run.TotalCount:
		run.Overall = StatusPass
	case run.FailCount == run.TotalCount:
		run.Overall = StatusFail
	default:
		run.Overall = StatusPartial
	}
	o.deps.Store.Record(run)
	return run
}

func (o *Orchestrator) runStep(ctx context.Context, c Capability) StepResult {
	stepCtx, cancel := context.WithTimeout(ctx, o.deps.PerStepTimeout)
	defer cancel()
	started := o.deps.Clock.Now()
	switch c {
	case CapabilitySTT:
		return o.runSTT(stepCtx, started)
	case CapabilityTTS:
		return o.runTTS(stepCtx, started)
	case CapabilitySummarize:
		return o.runSummarize(stepCtx, started)
	case CapabilityTranscode:
		return o.runTranscode(stepCtx, started)
	default:
		return StepResult{
			Capability: c, OK: false,
			ErrorCode: "unknown_capability", ErrorMessage: fmt.Sprintf("unknown capability %q", c),
			StartedAt: started, FinishedAt: o.deps.Clock.Now(),
		}
	}
}

func (o *Orchestrator) runSTT(ctx context.Context, started time.Time) StepResult {
	res := StepResult{Capability: CapabilitySTT, StartedAt: started, Details: map[string]string{}}
	if o.deps.STT == nil {
		return finishUnavailable(res, o.deps.Clock.Now())
	}
	cfg := o.sttConfig(ctx)
	out, err := o.deps.STT.Execute(ctx, sttchain.Request{Audio: smokedata.SmokeWAV(), Format: "wav", VADFilter: cfg.VADFilterEnabled})
	res.FinishedAt = o.deps.Clock.Now()
	if err != nil {
		return applyChainErr(res, err)
	}
	decision := quality.New(cfg, o.deps.Registry, nil).ApplyResult(ctx, out, smokedata.SmokeWAV())
	res.OK = true
	res.ProviderTier = string(out.Tier)
	res.ProviderID = out.ProviderID
	res.ModelID = out.ModelID
	res.LatencyMs = float64(out.Latency.Milliseconds())
	res.Details["diagnostic_scope"] = "asr_readiness"
	res.Details["quality_assessed"] = "false"
	res.Details["quality_note"] = "Bundled STT smoke audio verifies the provider path accepts and processes audio; transcript accuracy is measured by the eval harness."
	res.Details["transcript_filtered"] = fmt.Sprintf("%t", decision.Filtered)
	res.Details["filter_reason"] = decision.FilterReason
	res.Details["raw_transcript_length"] = fmt.Sprintf("%d", len(out.Text))
	res.Details["filtered_transcript_length"] = fmt.Sprintf("%d", len(decision.Text))
	res.Details["transcript_len"] = fmt.Sprintf("%d", len(decision.Text))
	res.Details["vad_filter"] = fmt.Sprintf("%t", cfg.VADFilterEnabled)
	if decision.Text != "" {
		res.Details["transcript_preview"] = previewString(decision.Text, 80)
	}
	return res
}

func (o *Orchestrator) sttConfig(ctx context.Context) sttpkg.StreamConfig {
	if o.deps.STTConfig == nil {
		return sttpkg.Defaults()
	}
	return o.deps.STTConfig(ctx)
}

func (o *Orchestrator) runTTS(ctx context.Context, started time.Time) StepResult {
	res := StepResult{Capability: CapabilityTTS, StartedAt: started, Details: map[string]string{}}
	if o.deps.TTS == nil {
		return finishUnavailable(res, o.deps.Clock.Now())
	}
	out, err := o.deps.TTS.Execute(ctx, ttschain.Request{
		Text:  previewString(smokedata.SmokeText(), 120),
		Voice: "voice.feminine.warm", Speed: 1.0, ResponseFormat: "wav",
	})
	res.FinishedAt = o.deps.Clock.Now()
	if err != nil {
		return applyChainErr(res, err)
	}
	res.OK = true
	res.ProviderTier = string(out.Tier)
	res.ProviderID = out.ProviderID
	res.ModelID = out.ModelID
	res.LatencyMs = float64(out.Latency.Milliseconds())
	res.Details["audio_bytes"] = fmt.Sprintf("%d", len(out.Audio))
	res.Details["content_type"] = out.ContentType
	return res
}

func (o *Orchestrator) runSummarize(ctx context.Context, started time.Time) StepResult {
	res := StepResult{Capability: CapabilitySummarize, StartedAt: started, Details: map[string]string{}}
	if o.deps.Summarize == nil {
		return finishUnavailable(res, o.deps.Clock.Now())
	}
	out, err := o.deps.Summarize.Execute(ctx, summarizechain.Request{
		Text: smokedata.SmokeText(), Level: "moderate",
	})
	res.FinishedAt = o.deps.Clock.Now()
	if err != nil {
		return applyChainErr(res, err)
	}
	res.OK = true
	res.ProviderTier = string(out.Tier)
	res.ProviderID = out.ProviderID
	res.ModelID = out.ModelID
	res.LatencyMs = float64(out.Latency.Milliseconds())
	res.Details["summary_len"] = fmt.Sprintf("%d", len(out.Text))
	if out.Text != "" {
		res.Details["summary_preview"] = previewString(out.Text, 80)
	}
	return res
}

func (o *Orchestrator) runTranscode(ctx context.Context, started time.Time) StepResult {
	res := StepResult{Capability: CapabilityTranscode, StartedAt: started, Details: map[string]string{}}
	if o.deps.Transcode == nil {
		return finishUnavailable(res, o.deps.Clock.Now())
	}
	t0 := o.deps.Clock.Now()
	out, err := o.deps.Transcode.Transcode(ctx, smokedata.SmokeWAV(), "wav")
	res.FinishedAt = o.deps.Clock.Now()
	res.LatencyMs = float64(res.FinishedAt.Sub(t0).Milliseconds())
	if err != nil {
		return applyTranscodeErr(res, err)
	}
	res.OK = true
	res.ProviderTier = "local"
	res.ProviderID = "ffmpeg"
	res.Details["output_bytes"] = fmt.Sprintf("%d", len(out))
	return res
}

// recordUsage emits one UsageRow per diagnostic step so operator probes
// are visible in the Usage page alongside real traffic. Rows are tagged
// operation="diagnostics" + capability=<step> so callers can filter.
// Skipped steps (Usage nil, or capability unmapped) are no-ops.
func (o *Orchestrator) recordUsage(runID string, step StepResult) {
	if o.deps.Usage == nil {
		return
	}
	capability := ""
	switch step.Capability {
	case CapabilitySTT:
		capability = "stt"
	case CapabilityTTS:
		capability = "tts"
	case CapabilitySummarize:
		capability = "summarize"
	case CapabilityTranscode:
		capability = "audio"
	default:
		return
	}
	row := store.UsageRow{
		OperationID:  runID + ":" + string(step.Capability),
		EmittedAt:    step.FinishedAt.UTC(),
		Capability:   capability,
		Operation:    "diagnostics",
		ProviderTier: step.ProviderTier,
		ProviderID:   step.ProviderID,
		ModelID:      step.ModelID,
		LatencyMs:    step.LatencyMs,
	}
	if !step.OK {
		if step.ErrorMessage != "" {
			row.Error = step.ErrorMessage
		} else {
			row.Error = step.ErrorCode
		}
	}
	o.deps.Usage.Enqueue(row)
}

func finishUnavailable(res StepResult, now time.Time) StepResult {
	res.FinishedAt = now
	res.OK = false
	res.ErrorCode = "not_configured"
	res.ErrorMessage = string(res.Capability) + " runner not configured"
	return res
}

func applyChainErr(res StepResult, err error) StepResult {
	res.OK = false
	res.ErrorMessage = err.Error()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		res.ErrorCode = "deadline_exceeded"
	// A summarize model role that resolves but is not installed is an
	// operator-fixable precondition, not an opaque "internal" — surface it
	// distinctly so the suite tile can show the actionable remedy. Checked
	// before ErrAllProvidersFailed because the chain wraps both.
	case errors.Is(err, intsumm.ErrSummarizeModelNotInstalled):
		res.ErrorCode = "model_not_installed"
	case errors.Is(err, sttchain.ErrAllProvidersFailed),
		errors.Is(err, ttschain.ErrAllProvidersFailed),
		errors.Is(err, summarizechain.ErrAllProvidersFailed):
		res.ErrorCode = "provider_unavailable"
	case errors.Is(err, sttchain.ErrInsufficientCredits),
		errors.Is(err, ttschain.ErrInsufficientCredits),
		errors.Is(err, summarizechain.ErrInsufficientCredits):
		res.ErrorCode = "insufficient_credits"
	default:
		res.ErrorCode = "internal"
	}
	return res
}

func applyTranscodeErr(res StepResult, err error) StepResult {
	res.OK = false
	res.ErrorMessage = err.Error()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		res.ErrorCode = "deadline_exceeded"
	// ffmpeg missing is a substrate dependency gap (operator action), not a
	// chain failure. Match on the typed sentinel rather than a brittle
	// string compare so a reworded error can't silently fall through.
	case errors.Is(err, intaudio.ErrFFmpegMissing):
		res.ErrorCode = "provider_unavailable"
	// ffmpeg ran but rejected the fixture/input — a real, distinct failure
	// class the suite should not flatten to "internal".
	case errors.Is(err, intaudio.ErrFfmpegExec):
		res.ErrorCode = "invalid_input"
	default:
		res.ErrorCode = "internal"
	}
	return res
}

func normalizeCapabilities(in []Capability) ([]Capability, error) {
	if len(in) == 0 {
		out := make([]Capability, len(AllCapabilities))
		copy(out, AllCapabilities)
		return out, nil
	}
	known := map[Capability]bool{}
	for _, c := range AllCapabilities {
		known[c] = true
	}
	seen := map[Capability]bool{}
	out := make([]Capability, 0, len(in))
	for _, c := range in {
		if !known[c] {
			return nil, fmt.Errorf("unknown capability %q", c)
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out, nil
}

func previewString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

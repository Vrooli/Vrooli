// Speaker configuration handlers (Get/Update + Status). The
// in-process speaker config cell lives here and is the single source
// of truth for the audio-tools instance.
package stt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// loadPersistedSpeakerCfg hydrates the in-process speaker-config cell from the
// persisted row. Best-effort: a missing or corrupt row leaves the defaults.
func loadPersistedSpeakerCfg(ctx context.Context, repo SpeakerConfigRepository, log logx.Logger) {
	raw, ok, err := repo.Get(ctx)
	if err != nil {
		if log != nil {
			log.Printf("speaker-config: load failed, using defaults: %v", err)
		}
		return
	}
	if !ok || raw == "" {
		return
	}
	var d speakerCfgDoc
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		if log != nil {
			log.Printf("speaker-config: corrupt persisted config, using defaults: %v", err)
		}
		return
	}
	speakerCfgMu.Lock()
	speakerCfg = d
	speakerCfgMu.Unlock()
}

// In-process speaker config; the single audio-tools instance owns the cell.
var (
	speakerCfgMu sync.Mutex
	speakerCfg   = defaultSpeakerCfg()
)

// persistSpeakerCfgLocked writes the speaker-config doc to the persistent store
// (when configured) and then commits it to the in-process cell. Persist happens
// BEFORE the commit so an I/O failure leaves the cell and the stored row
// consistent (no half-applied config). Callers MUST hold speakerCfgMu. Shared by
// UpdateSpeakerConfig and the enroll handler so the durability invariant lives
// in one place.
func (h *connectHandler) persistSpeakerCfgLocked(ctx context.Context, d speakerCfgDoc) error {
	if h.deps.SpeakerConfig != nil {
		raw, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("encode speaker config: %w", err)
		}
		if err := h.deps.SpeakerConfig.Set(ctx, string(raw)); err != nil {
			return fmt.Errorf("persist speaker config: %w", err)
		}
	}
	speakerCfg = d
	return nil
}

// appendUnique returns list with v appended only when absent, in a freshly
// allocated slice so callers never mutate a shared backing array.
func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(append([]string{}, list...), v)
}

// speakerVerification adapts pipeline.EvaluateSpeaker (cosine-similarity
// verification against the speaker-verification resource) to the
// egress.SpeakerIsolation seam. It lives in the handler layer — not pipeline —
// because pipeline cannot import egress (egress imports sttchain, sttchain
// imports pipeline; that would cycle). It captures the SpeakerConfig + client
// taken at session start so a mid-session config change does not retune an
// in-flight stream, and holds a per-session SessionSpeakerState so the decision
// accumulates across segments (warm-up + EMA) rather than swinging per segment.
//
// It is stateful (pointer receiver) and is constructed once per session by
// currentSpeakerIsolation. The egress pipeline drives Evaluate from a single
// goroutine (see internal/stt/segmenter runEgress), so the state needs no lock.
type speakerVerification struct {
	cfg    sttpipeline.SpeakerConfig
	client *sttpipeline.SpeakerClient
	state  *sttpipeline.SessionSpeakerState

	// logger, when non-nil, emits a one-line diagnostic per segment so
	// operators can see whether the gate is actually receiving evidence
	// (Applied/Sufficient) and what session score it's accumulating. Set by
	// currentSpeakerIsolation; tests leave it nil.
	logger logx.Logger
}

// newSpeakerVerification builds the per-session stateful isolation adapter.
func newSpeakerVerification(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient) *speakerVerification {
	return &speakerVerification{
		cfg:    cfg,
		client: client,
		state:  sttpipeline.NewSessionSpeakerState(cfg),
	}
}

// NewSpeakerIsolationFromConfig builds the production speaker-verification
// egress adapter from an explicit per-session config. Experiment workers use
// this to exercise the same adapter as live STT without reading or mutating the
// live speaker-config cell.
func NewSpeakerIsolationFromConfig(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient, logger logx.Logger) egress.SpeakerIsolation {
	if !cfg.Enabled || cfg.Mode == "off" {
		return nil
	}
	v := newSpeakerVerification(cfg, client)
	v.logger = logger
	return v
}

func (s *speakerVerification) Evaluate(ctx context.Context, audio []byte) egress.SpeakerVerdict {
	d := sttpipeline.EvaluateSpeaker(ctx, s.cfg, s.client, audio)
	allowed, smoothed, reason := s.state.Observe(d)

	score := smoothed
	if !s.state.HasEvidence() {
		// No applied segment yet (warm-up / undetermined): surface this
		// segment's own score for display rather than a still-zero EMA.
		score = d.Score
	}
	v := egress.SpeakerVerdict{Allowed: allowed, Score: score, Threshold: s.cfg.Threshold}
	if !allowed {
		if reason == "" {
			reason = sttpipeline.FormatSpeakerDecisionError(d)
		}
		v.Reason = reason
	}
	// Allowed but verification never matched a profile => let through under
	// FallbackWithoutVerification (resource down / no enrolled profile).
	if allowed && d.Enabled && !d.Applied {
		v.FallbackUsed = true
	}

	if s.logger != nil {
		// One line per segment evaluation. Reads at a glance whether the gate
		// has evidence to work with: a sticky "applied=false sufficient=false"
		// means the resource is judging every segment as too-short-voiced and
		// the gate is functionally inert; "applied=true" with a smoothed score
		// reveals what cutoff would actually fire.
		s.logger.Printf(
			"speaker-verify: allowed=%t applied=%t sufficient=%t voiced=%.2fs raw=%.3f smoothed=%.3f thr=%.2f mode=%s reason=%q",
			allowed, d.Applied, d.Sufficient, d.VoicedSeconds, d.Score, score, s.cfg.Threshold, s.cfg.Mode, v.Reason,
		)
	}
	return v
}

// currentSpeakerIsolation builds the per-session audio-domain isolation from
// the live speaker-config cell + the resource client. Returns nil when speaker
// isolation is disabled or off, so the Segmenter omits the audio-domain egress
// stage entirely. Read once at session start — the stage never retunes mid-
// session (mirrors how StreamConfig is snapshotted per session).
func currentSpeakerIsolation(d Deps) egress.SpeakerIsolation {
	speakerCfgMu.Lock()
	doc := speakerCfg
	speakerCfgMu.Unlock()
	if !doc.Enabled || doc.Mode == "off" {
		return nil
	}
	cfg := sttpipeline.SpeakerConfig{
		Enabled:                     doc.Enabled,
		ProfileIDs:                  doc.ProfileIDs,
		Threshold:                   doc.Threshold,
		Mode:                        doc.Mode,
		RejectBehavior:              doc.RejectBehavior,
		FallbackWithoutVerification: doc.FallbackWithoutVerification,
		ExtractionEnabled:           doc.ExtractionEnabled,
		MinDecisionSeconds:          doc.MinDecisionSeconds,
		ScoreSmoothing:              doc.ScoreSmoothing,
	}
	return NewSpeakerIsolationFromConfig(cfg, d.SpeakerResource, d.Logger)
}

// speakerExtraction adapts the speaker-verification resource's /v1/extract
// endpoint to the ingress.TargetExtractor seam. Like speakerVerification it
// lives in the handler layer — pipeline cannot implement an ingress interface
// without the ingress→sttchain→pipeline cycle. It isolates the enrolled
// speaker's voice from a window of canonical PCM; on no-match or any failure it
// returns the input unchanged. Extraction ISOLATES audio (ingress); it never
// blocks a segment — the egress verification gate is what blocks text.
type speakerExtraction struct {
	cfg    sttpipeline.SpeakerConfig
	client *sttpipeline.SpeakerClient
}

func (s speakerExtraction) Extract(ctx context.Context, pcm []byte) ([]byte, error) {
	if s.client == nil || len(s.cfg.ProfileIDs) == 0 {
		return pcm, nil
	}
	// The resource decodes by container sniffing and cannot read headerless
	// PCM, so wrap the window in a WAV header (mirrors pipeline.EvaluateSpeaker).
	// The resource returns cleaned canonical PCM (s16le/16kHz/mono) ready to
	// re-enter the ingress stream with no further decode.
	wav := audioformat.WAVFromCanonicalPCM(pcm)
	var best []byte
	var bestScore float64
	found := false
	for _, profileID := range s.cfg.ProfileIDs {
		res, err := s.client.Extract(ctx, wav, profileID, true)
		if err != nil {
			continue
		}
		if res.Matched {
			return res.Audio, nil
		}
		if !found || res.Score > bestScore {
			bestScore = res.Score
			best = res.Audio
			found = true
		}
	}
	if !found || len(best) == 0 {
		return pcm, nil
	}
	return best, nil
}

// NewSpeakerExtractionFromConfig builds the production target-speaker
// extraction ingress adapter from an explicit per-session config. Experiment
// workers use this path for hermetic speaker-dimension runs.
func NewSpeakerExtractionFromConfig(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient) ingress.TargetExtractor {
	if !cfg.Enabled || cfg.Mode == "off" || !cfg.ExtractionEnabled || len(cfg.ProfileIDs) == 0 {
		return nil
	}
	if client == nil {
		return nil
	}
	return speakerExtraction{
		cfg: sttpipeline.SpeakerConfig{
			ProfileIDs: cfg.ProfileIDs,
			Threshold:  cfg.Threshold,
			Mode:       cfg.Mode,
		},
		client: client,
	}
}

// currentSpeakerExtraction builds the per-session ingress extractor from the
// live speaker-config cell + the resource client. Returns nil (the Segmenter
// then omits the ingress extraction stage) unless extraction is explicitly
// enabled AND a profile is bound. Read once at session start — mirrors
// currentSpeakerIsolation, and is independent of it (extraction is ingress,
// verification is egress; they may both run).
func currentSpeakerExtraction(d Deps) ingress.TargetExtractor {
	speakerCfgMu.Lock()
	doc := speakerCfg
	speakerCfgMu.Unlock()
	if !doc.Enabled || doc.Mode == "off" || !doc.ExtractionEnabled || len(doc.ProfileIDs) == 0 {
		return nil
	}
	return NewSpeakerExtractionFromConfig(sttpipeline.SpeakerConfig{
		Enabled:           doc.Enabled,
		ProfileIDs:        doc.ProfileIDs,
		Threshold:         doc.Threshold,
		Mode:              doc.Mode,
		ExtractionEnabled: doc.ExtractionEnabled,
	}, d.SpeakerResource)
}

func (h *connectHandler) GetSpeakerConfig(_ context.Context, _ *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	speakerCfgMu.Lock()
	d := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.GetSpeakerConfigResponse{Config: d.toProto()}), nil
}

var speakerConfigAllowedPaths = map[string]struct{}{
	"enabled":                       {},
	"profile_ids":                   {},
	"threshold":                     {},
	"mode":                          {},
	"reject_behavior":               {},
	"fallback_without_verification": {},
	"extraction_enabled":            {},
	"min_decision_seconds":          {},
	"score_smoothing":               {},
}

func (h *connectHandler) UpdateSpeakerConfig(ctx context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	m := req.Msg
	mask := m.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask required"))
	}
	if bad := protomap.MaskPathsOutsideAllowed(mask, speakerConfigAllowedPaths); len(bad) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown update_mask paths: %v", bad))
	}
	cfg := m.GetConfig()
	speakerCfgMu.Lock()
	defer speakerCfgMu.Unlock()
	d := speakerCfg
	if protomap.MaskHas(mask, "enabled") {
		d.Enabled = cfg.GetEnabled()
	}
	if protomap.MaskHas(mask, "profile_ids") {
		d.ProfileIDs = append([]string{}, cfg.GetProfileIds()...)
	}
	if protomap.MaskHas(mask, "threshold") {
		d.Threshold = cfg.GetThreshold()
	}
	if protomap.MaskHas(mask, "mode") {
		d.Mode = protomap.SpeakerModeFromProto(cfg.GetMode())
	}
	if protomap.MaskHas(mask, "reject_behavior") {
		d.RejectBehavior = protomap.RejectBehaviorFromProto(cfg.GetRejectBehavior())
	}
	if protomap.MaskHas(mask, "fallback_without_verification") {
		d.FallbackWithoutVerification = cfg.GetFallbackWithoutVerification()
	}
	if protomap.MaskHas(mask, "extraction_enabled") {
		d.ExtractionEnabled = cfg.GetExtractionEnabled()
	}
	if protomap.MaskHas(mask, "min_decision_seconds") {
		d.MinDecisionSeconds = cfg.GetMinDecisionSeconds()
	}
	if protomap.MaskHas(mask, "score_smoothing") {
		d.ScoreSmoothing = cfg.GetScoreSmoothing()
	}
	// Persist BEFORE committing to the cell so an I/O failure leaves the
	// in-memory state and the stored row consistent (no half-applied config).
	if err := h.persistSpeakerCfgLocked(ctx, d); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: d.toProto()}), nil
}

func (h *connectHandler) GetSpeakerStatus(ctx context.Context, _ *connect.Request[sttv1.GetSpeakerStatusRequest]) (*connect.Response[sttv1.GetSpeakerStatusResponse], error) {
	speakerCfgMu.Lock()
	cfg := speakerCfg
	speakerCfgMu.Unlock()

	var profiles []*sttv1.SpeakerProfile
	if h.deps.Speaker != nil {
		rows, err := h.deps.Speaker.List(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		for _, p := range rows {
			profiles = append(profiles, speakerProfileToProto(p))
		}
	}
	st := &sttv1.SpeakerStatus{
		Config:            cfg.toProto(),
		Capability:        "available",
		CapabilityLabel:   "Speaker store",
		ResourceReady:     true,
		ProfileConfigured: len(cfg.ProfileIDs) > 0,
		ProfileExists:     len(profiles) > 0,
		ProfileCount:      int32(len(profiles)),
		Profiles:          profiles,
		CheckedAt:         protomap.TimeToProto(h.deps.Clock.Now().UTC()),
	}
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{Status: st}), nil
}

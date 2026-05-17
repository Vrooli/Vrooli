// Speaker configuration handlers (Get/Update + Status). The
// in-process speaker config cell lives here and is the single source
// of truth for the audio-tools instance.
package stt

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// In-process speaker config; the single audio-tools instance owns the cell.
var (
	speakerCfgMu sync.Mutex
	speakerCfg   = defaultSpeakerCfg()
)

func (h *connectHandler) GetSpeakerConfig(_ context.Context, _ *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	speakerCfgMu.Lock()
	d := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.GetSpeakerConfigResponse{Config: d.toProto()}), nil
}

func (h *connectHandler) UpdateSpeakerConfig(_ context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	m := req.Msg
	speakerCfgMu.Lock()
	d := speakerCfg
	if m.GetHasEnabled() {
		d.Enabled = m.GetEnabled()
	}
	if m.GetHasProfileIds() {
		d.ProfileIDs = append([]string{}, m.GetProfileIds()...)
	}
	if m.GetHasThreshold() {
		d.Threshold = m.GetThreshold()
	}
	if m.GetHasMode() {
		d.Mode = m.GetMode()
	}
	if m.GetHasRejectBehavior() {
		d.RejectBehavior = m.GetRejectBehavior()
	}
	if m.GetHasFallbackWithoutVerification() {
		d.FallbackWithoutVerification = m.GetFallbackWithoutVerification()
	}
	if m.GetHasExtractionEnabled() {
		d.ExtractionEnabled = m.GetExtractionEnabled()
	}
	speakerCfg = d
	speakerCfgMu.Unlock()
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
			profiles = append(profiles, &sttv1.SpeakerProfile{
				Id:           p.ID,
				DisplayName:  p.Name,
				CreatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
				UpdatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
				EmbeddingDim: int32(len(p.Embedding)),
			})
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
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{Status: st}), nil
}

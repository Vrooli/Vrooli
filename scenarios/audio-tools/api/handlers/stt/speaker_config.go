// Speaker configuration handlers (Get/Update + Status). The
// in-process speaker config cell lives here and is the single source
// of truth for the audio-tools instance.
package stt

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"audio-tools/internal/protomap"

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

var speakerConfigAllowedPaths = map[string]struct{}{
	"enabled":                       {},
	"profile_ids":                   {},
	"threshold":                     {},
	"mode":                          {},
	"reject_behavior":               {},
	"fallback_without_verification": {},
	"extraction_enabled":            {},
}

func (h *connectHandler) UpdateSpeakerConfig(_ context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
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
				CreatedAt:    protomap.TimeToProto(p.CreatedAt),
				UpdatedAt:    protomap.TimeToProto(p.CreatedAt),
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
		CheckedAt:         protomap.TimeToProto(h.deps.Clock.Now().UTC()),
	}
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{Status: st}), nil
}

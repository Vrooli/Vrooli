// Speaker administration handlers (config, status, profile list/enroll/
// remove/delete) backed by the speaker store and a small in-memory
// speaker-config cell.
package stt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/store"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// speakerCfgDoc is the JSON view of SpeakerConfig.
type speakerCfgDoc struct {
	Enabled                     bool     `json:"enabled"`
	ProfileIDs                  []string `json:"profile_ids"`
	Threshold                   float64  `json:"threshold"`
	Mode                        string   `json:"mode"`
	RejectBehavior              string   `json:"reject_behavior"`
	FallbackWithoutVerification bool     `json:"fallback_without_verification"`
	ExtractionEnabled           bool     `json:"extraction_enabled"`
}

func defaultSpeakerCfg() speakerCfgDoc {
	return speakerCfgDoc{
		Enabled: false, ProfileIDs: []string{}, Threshold: 0.7,
		Mode: "off", RejectBehavior: "drop",
	}
}

func (d speakerCfgDoc) toProto() *sttv1.SpeakerConfig {
	return &sttv1.SpeakerConfig{
		Enabled: d.Enabled, ProfileIds: d.ProfileIDs, Threshold: d.Threshold,
		Mode: d.Mode, RejectBehavior: d.RejectBehavior,
		FallbackWithoutVerification: d.FallbackWithoutVerification,
		ExtractionEnabled:           d.ExtractionEnabled,
	}
}

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

func (h *connectHandler) ListSpeakerProfiles(ctx context.Context, _ *connect.Request[sttv1.ListSpeakerProfilesRequest]) (*connect.Response[sttv1.ListSpeakerProfilesResponse], error) {
	if h.deps.Speaker == nil {
		return connect.NewResponse(&sttv1.ListSpeakerProfilesResponse{}), nil
	}
	rows, err := h.deps.Speaker.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*sttv1.SpeakerProfile, 0, len(rows))
	for _, p := range rows {
		out = append(out, &sttv1.SpeakerProfile{
			Id: p.ID, DisplayName: p.Name,
			CreatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
			EmbeddingDim: int32(len(p.Embedding)),
		})
	}
	return connect.NewResponse(&sttv1.ListSpeakerProfilesResponse{Profiles: out, Count: int32(len(out))}), nil
}

func (h *connectHandler) EnrollSpeakerProfile(ctx context.Context, req *connect.Request[sttv1.EnrollSpeakerProfileRequest]) (*connect.Response[sttv1.EnrollSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker store not configured"))
	}
	m := req.Msg
	id := m.GetProfileId()
	if id == "" {
		id = uuid.NewString()
	}
	if len(m.GetAudio()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("audio required"))
	}
	// Greenfield-minimum embedding: the raw audio length serves as a
	// stand-in fingerprint until the speaker encoder lands.
	if err := h.deps.Speaker.Upsert(ctx, store.SpeakerProfile{
		ID: id, Name: m.GetDisplayName(),
		Embedding: m.GetAudio()[:minInt(len(m.GetAudio()), 128)],
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	speakerCfgMu.Lock()
	if m.GetHasAddToActive() && m.GetAddToActive() {
		speakerCfg.ProfileIDs = append(speakerCfg.ProfileIDs, id)
	}
	if m.GetHasEnable() && m.GetEnable() {
		speakerCfg.Enabled = true
	}
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.EnrollSpeakerProfileResponse{
		Enrollment: &sttv1.SpeakerEnrollment{
			ProfileId:    id,
			DisplayName:  m.GetDisplayName(),
			EmbeddingDim: int32(minInt(len(m.GetAudio()), 128)),
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		},
		Config: cfg.toProto(),
	}), nil
}

func (h *connectHandler) ClearSpeakerProfileBinding(_ context.Context, _ *connect.Request[sttv1.ClearSpeakerProfileBindingRequest]) (*connect.Response[sttv1.ClearSpeakerProfileBindingResponse], error) {
	speakerCfgMu.Lock()
	speakerCfg.ProfileIDs = nil
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.ClearSpeakerProfileBindingResponse{Config: cfg.toProto()}), nil
}

func (h *connectHandler) RemoveSpeakerProfile(_ context.Context, req *connect.Request[sttv1.RemoveSpeakerProfileRequest]) (*connect.Response[sttv1.RemoveSpeakerProfileResponse], error) {
	id := req.Msg.GetProfileId()
	speakerCfgMu.Lock()
	out := speakerCfg.ProfileIDs[:0]
	for _, p := range speakerCfg.ProfileIDs {
		if p != id {
			out = append(out, p)
		}
	}
	speakerCfg.ProfileIDs = out
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.RemoveSpeakerProfileResponse{Config: cfg.toProto()}), nil
}

func (h *connectHandler) DeleteSpeakerProfile(ctx context.Context, req *connect.Request[sttv1.DeleteSpeakerProfileRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker store not configured"))
	}
	id := req.Msg.GetProfileId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("profile_id required"))
	}
	if _, err := h.deps.Speaker.Delete(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	speakerCfgMu.Lock()
	out := speakerCfg.ProfileIDs[:0]
	for _, p := range speakerCfg.ProfileIDs {
		if p != id {
			out = append(out, p)
		}
	}
	speakerCfg.ProfileIDs = out
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.DeleteSpeakerProfileResponse{Config: cfg.toProto()}), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

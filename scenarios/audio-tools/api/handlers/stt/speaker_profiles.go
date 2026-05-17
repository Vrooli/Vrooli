// Speaker profile management: list / enroll / clear-binding / remove
// / delete. Profile storage lives in h.deps.Speaker; this file only
// owns the wire-layer translation and the in-process speakerCfg
// binding updates.
package stt

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/protomap"
	"audio-tools/internal/store"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

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
			Id:           p.ID,
			DisplayName:  p.Name,
			CreatedAt:    protomap.TimeToProto(p.CreatedAt),
			UpdatedAt:    protomap.TimeToProto(p.CreatedAt),
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
	if m.AddToActive != nil && *m.AddToActive {
		speakerCfg.ProfileIDs = append(speakerCfg.ProfileIDs, id)
	}
	if m.Enable != nil && *m.Enable {
		speakerCfg.Enabled = true
	}
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.EnrollSpeakerProfileResponse{
		Enrollment: &sttv1.SpeakerEnrollment{
			ProfileId:    id,
			DisplayName:  m.GetDisplayName(),
			EmbeddingDim: int32(minInt(len(m.GetAudio()), 128)),
			CreatedAt:    protomap.TimeToProto(h.deps.Clock.Now().UTC()),
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

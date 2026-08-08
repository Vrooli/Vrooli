// Speaker profile management: list / enroll / clear-binding / remove
// / delete. Profile storage lives in h.deps.Speaker; this file only
// owns the wire-layer translation and the in-process speakerCfg
// binding updates.
package stt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/protoint"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	sttpipeline "audio-tools/internal/stt/pipeline"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
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
		out = append(out, speakerProfileToProto(p))
	}
	return connect.NewResponse(&sttv1.ListSpeakerProfilesResponse{Profiles: out, Count: protoint.FromInt(len(out))}), nil
}

// speakerProfileToProto projects a stored profile to the wire shape, surfacing
// the enrollment metadata cached at enroll time (List/Status both render it).
// EmbeddingDim falls back to the locally stored embedding length only when the
// cached dim is zero (older rows enrolled before the metadata column existed).
func speakerProfileToProto(p store.SpeakerProfile) *sttv1.SpeakerProfile {
	embeddingDim := protoint.FromInt64(int64(p.EmbeddingDim))
	if embeddingDim == 0 {
		embeddingDim = protoint.FromInt(len(p.Embedding))
	}
	return &sttv1.SpeakerProfile{
		Id:                 p.ID,
		DisplayName:        p.Name,
		CreatedAt:          protomap.TimeToProto(p.CreatedAt),
		UpdatedAt:          protomap.TimeToProto(p.CreatedAt),
		ModelName:          p.ModelName,
		EmbeddingDim:       embeddingDim,
		SampleRate:         protoint.FromInt64(int64(p.SampleRate)),
		ClipCount:          protoint.FromInt64(int64(p.ClipCount)),
		TotalVoicedSeconds: p.TotalVoicedSeconds,
	}
}

func (h *connectHandler) EnrollSpeakerProfile(ctx context.Context, req *connect.Request[sttv1.EnrollSpeakerProfileRequest]) (*connect.Response[sttv1.EnrollSpeakerProfileResponse], error) {
	if h.deps.Speaker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker store not configured"))
	}
	m := req.Msg
	if len(m.GetAudio()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("audio required"))
	}
	if h.deps.SpeakerResource == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker-verification resource not configured"))
	}
	id := m.GetProfileId()
	if id == "" {
		id = uuid.NewString()
	}
	// Normalize the uploaded audio to canonical-PCM WAV before enrolling so the
	// enrollment embedding is computed from the SAME audio characteristics the
	// streaming verify path uses (pipeline/speaker.go wraps canonical PCM in a
	// WAV header too). Without this, enrollment embeddings would come from the
	// browser's lossy WebM/Opus while verification embeddings come from clean
	// decoded PCM — an apples-to-oranges pairing that degrades matching.
	enrollAudio, enrollFilename := h.normalizeEnrollAudio(ctx, m.GetAudio(), m.GetFormat())
	// Enroll against the speaker-verification resource: it computes and OWNS
	// the real ECAPA embedding (keyed by profile id) used by streaming verify.
	// audio-tools persists the profile metadata + binding locally — the
	// embedding never round-trips back, so the resource stays the single
	// authority for identity comparison.
	enroll, err := h.deps.SpeakerResource.Enroll(ctx, enrollAudio, id, m.GetDisplayName(), m.GetNotes(), m.GetLabel(), enrollFilename)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("speaker-verification enroll: %w", err))
	}
	if enroll.ProfileID != "" {
		id = enroll.ProfileID
	}
	if err := h.deps.Speaker.Upsert(ctx, store.SpeakerProfile{
		ID:                 id,
		Name:               m.GetDisplayName(),
		ClipCount:          enroll.ClipCount,
		TotalVoicedSeconds: enroll.TotalVoicedSeconds,
		SampleRate:         enroll.SampleRate,
		EmbeddingDim:       enroll.EmbeddingDim,
		ModelName:          enroll.ModelName,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	speakerCfgMu.Lock()
	d := speakerCfg
	changed := false
	if m.AddToActive != nil && *m.AddToActive {
		d.ProfileIDs = appendUnique(d.ProfileIDs, id)
		changed = true
	}
	if m.Enable != nil && *m.Enable {
		d.Enabled = true
		// "Enabled" with mode=off is an inert dead state: the gate never runs,
		// so enrollment appears to do nothing. Lift a freshly-enabled config to
		// advisory (verify + annotate, never drops) so the enrolled voice
		// actually takes effect. An explicit filter/advisory choice is preserved
		// — only the inert "off" (or an unset mode) is replaced.
		if d.Mode == "" || d.Mode == "off" {
			d.Mode = "advisory"
		}
		changed = true
	}
	// Persist the binding/enable so it survives a restart (loadPersistedSpeakerCfg
	// rehydrates from the same row). Only write when something actually changed.
	if changed {
		if err := h.persistSpeakerCfgLocked(ctx, d); err != nil {
			speakerCfgMu.Unlock()
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	cfg := d
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.EnrollSpeakerProfileResponse{
		Enrollment: &sttv1.SpeakerEnrollment{
			ProfileId:          id,
			DisplayName:        m.GetDisplayName(),
			EmbeddingDim:       protoint.FromInt64(int64(enroll.EmbeddingDim)),
			SampleRate:         protoint.FromInt64(int64(enroll.SampleRate)),
			ModelName:          enroll.ModelName,
			CreatedAt:          protomap.TimeToProto(h.deps.Clock.Now().UTC()),
			ClipId:             enroll.ClipID,
			Label:              enroll.Label,
			VoicedSeconds:      enroll.VoicedSeconds,
			ClipCount:          protoint.FromInt64(int64(enroll.ClipCount)),
			TotalVoicedSeconds: enroll.TotalVoicedSeconds,
		},
		Config: cfg.toProto(),
	}), nil
}

// normalizeEnrollAudio decodes uploaded enrollment audio (honoring the declared
// format) to canonical PCM and WAV-wraps it, so the enrollment embedding is
// computed from the same audio the verify path produces. It returns the audio
// bytes to enroll plus an honest filename. When the engine is unwired or the
// format is unknown/undecodable, it returns the original bytes (the resource
// decodes by content sniffing) and logs the degraded, lower-fidelity path —
// enrollment still succeeds, it just isn't preprocessing-matched to verify.
func (h *connectHandler) normalizeEnrollAudio(ctx context.Context, audio []byte, format commonv1.AudioFormat) ([]byte, string) {
	if h.deps.Engine == nil {
		return audio, "enrollment.bin"
	}
	codec, ok := audioformat.FromProto(format)
	if !ok {
		if h.deps.Logger != nil {
			h.deps.Logger.Printf("speaker-enroll: unknown audio format %v; enrolling raw bytes (resource will sniff)", format)
		}
		return audio, "enrollment.bin"
	}
	pcm, err := h.deps.Engine.Normalize(ctx, codec, audio)
	if err != nil {
		if h.deps.Logger != nil {
			h.deps.Logger.Printf("speaker-enroll: normalize %s failed (%v); enrolling raw bytes", codec, err)
		}
		return audio, "enrollment.bin"
	}
	return audioformat.WAVFromCanonicalPCM(pcm), "enrollment.wav"
}

func (h *connectHandler) ClearSpeakerProfileBinding(_ context.Context, _ *connect.Request[sttv1.ClearSpeakerProfileBindingRequest]) (*connect.Response[sttv1.ClearSpeakerProfileBindingResponse], error) {
	speakerCfgMu.Lock()
	speakerCfg.ProfileIDs = nil
	cfg := speakerCfg
	speakerCfgMu.Unlock()
	return connect.NewResponse(&sttv1.ClearSpeakerProfileBindingResponse{Config: cfg.toProto()}), nil
}

func (h *connectHandler) UnbindSpeakerProfile(_ context.Context, req *connect.Request[sttv1.UnbindSpeakerProfileRequest]) (*connect.Response[sttv1.UnbindSpeakerProfileResponse], error) {
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
	return connect.NewResponse(&sttv1.UnbindSpeakerProfileResponse{Config: cfg.toProto()}), nil
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
	// Best-effort purge of the resource-side embedding so the verification
	// service does not keep a profile the operator deleted. A resource error
	// must not fail the local delete (the binding is already gone).
	if h.deps.SpeakerResource != nil {
		_ = h.deps.SpeakerResource.DeleteProfile(ctx, id)
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

// speakerClipToProto maps a resource clip-metadata record to the wire shape.
func speakerClipToProto(c sttpipeline.SpeakerProfileClip) *sttv1.SpeakerProfileClip {
	var created time.Time
	if c.CreatedAt != "" {
		created, _ = time.Parse(time.RFC3339, c.CreatedAt)
	}
	return &sttv1.SpeakerProfileClip{
		ClipId:        c.ClipID,
		Label:         c.Label,
		VoicedSeconds: c.VoicedSeconds,
		AudioSeconds:  c.AudioSeconds,
		CreatedAt:     protomap.TimeToProto(created),
		EmbeddingDim:  protoint.FromInt64(int64(c.EmbeddingDim)),
	}
}

func (h *connectHandler) ListSpeakerProfileClips(ctx context.Context, req *connect.Request[sttv1.ListSpeakerProfileClipsRequest]) (*connect.Response[sttv1.ListSpeakerProfileClipsResponse], error) {
	if h.deps.SpeakerResource == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker-verification resource not configured"))
	}
	id := req.Msg.GetProfileId()
	list, err := h.deps.SpeakerResource.ListClips(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("list speaker clips: %w", err))
	}
	clips := make([]*sttv1.SpeakerProfileClip, 0, len(list.Clips))
	for _, c := range list.Clips {
		clips = append(clips, speakerClipToProto(c))
	}
	return connect.NewResponse(&sttv1.ListSpeakerProfileClipsResponse{
		ProfileId: id,
		Clips:     clips,
		Count:     protoint.FromInt(len(clips)),
	}), nil
}

func (h *connectHandler) DeleteSpeakerProfileClip(ctx context.Context, req *connect.Request[sttv1.DeleteSpeakerProfileClipRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileClipResponse], error) {
	if h.deps.SpeakerResource == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("speaker-verification resource not configured"))
	}
	id := req.Msg.GetProfileId()
	clipID := req.Msg.GetClipId()
	res, err := h.deps.SpeakerResource.DeleteClip(ctx, id, clipID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("delete speaker clip: %w", err))
	}

	if res.DeletedProfile {
		// The profile lost its last clip and no longer exists on the resource;
		// purge the local cache row and unbind it so config stays consistent.
		if h.deps.Speaker != nil {
			if _, derr := h.deps.Speaker.Delete(ctx, id); derr != nil {
				return nil, connect.NewError(connect.CodeInternal, derr)
			}
		}
		speakerCfgMu.Lock()
		d := speakerCfg
		out := d.ProfileIDs[:0]
		for _, p := range d.ProfileIDs {
			if p != id {
				out = append(out, p)
			}
		}
		d.ProfileIDs = out
		if err := h.persistSpeakerCfgLocked(ctx, d); err != nil {
			speakerCfgMu.Unlock()
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		speakerCfgMu.Unlock()
	} else if h.deps.Speaker != nil {
		// Refresh the cached clip totals for the surviving profile.
		if p, ok, gerr := h.deps.Speaker.Get(ctx, id); gerr == nil && ok {
			p.ClipCount = res.ClipCount
			p.TotalVoicedSeconds = res.TotalVoicedSeconds
			if uerr := h.deps.Speaker.Upsert(ctx, p); uerr != nil {
				return nil, connect.NewError(connect.CodeInternal, uerr)
			}
		}
	}

	return connect.NewResponse(&sttv1.DeleteSpeakerProfileClipResponse{
		ProfileId:          res.ProfileID,
		ClipId:             res.ClipID,
		DeletedProfile:     res.DeletedProfile,
		ClipCount:          protoint.FromInt64(int64(res.ClipCount)),
		TotalVoicedSeconds: res.TotalVoicedSeconds,
	}), nil
}

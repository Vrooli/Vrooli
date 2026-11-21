package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	sttpipeline "audio-tools/internal/stt/pipeline"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// capturingEnrollResource is a fake speaker-verification resource that records
// the bytes uploaded to /v1/profiles so a test can assert how audio-tools
// preprocessed the enrollment audio before sending it.
type capturingEnrollResource struct {
	mu       sync.Mutex
	gotAudio []byte
	srv      *httptest.Server
}

func newCapturingEnrollResource(t *testing.T, seconds float64) (*capturingEnrollResource, *sttpipeline.SpeakerClient) {
	t.Helper()
	res := &capturingEnrollResource{}
	res.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/clips") {
			_ = json.NewEncoder(w).Encode(map[string]any{"profile_id": "profile-1", "count": 1, "clips": []map[string]any{{"clip_id": "clip-1", "label": "office", "voiced_seconds": 1.25, "audio_seconds": 2.0, "created_at": "2026-07-24T20:00:00Z", "embedding_dim": 192}}})
			return
		}
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/clips/") {
			_ = json.NewEncoder(w).Encode(map[string]any{"profile_id": "profile-1", "clip_id": "clip-1", "clip_count": 1, "total_voiced_seconds": 1.25})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/v1/profiles" {
			require.NoError(t, r.ParseMultipartForm(1<<20))
			f, _, err := r.FormFile("audio")
			require.NoError(t, err)
			defer f.Close()
			buf := make([]byte, 0, 1024)
			tmp := make([]byte, 512)
			for {
				n, readErr := f.Read(tmp)
				buf = append(buf, tmp[:n]...)
				if readErr != nil {
					break
				}
			}
			res.mu.Lock()
			res.gotAudio = buf
			res.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile_id": r.FormValue("profile_id"), "clip_id": "clip-1", "label": r.FormValue("label"),
				"voiced_seconds": seconds, "audio_seconds": seconds,
				"clip_count": 1, "total_voiced_seconds": seconds,
				"embedding_dim": 192, "sample_rate": 16000,
				"model_name": "speechbrain/spkrec-ecapa-voxceleb", "created_at": "2026-05-27T00:00:00Z",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(res.srv.Close)
	return res, &sttpipeline.SpeakerClient{BaseURL: res.srv.URL, Doer: http.DefaultClient}
}

func (c *capturingEnrollResource) audio() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gotAudio
}

// fakeFFmpegRunner returns canned PCM for any one-shot run, standing in for the
// ffmpeg decode so the normalization path is exercised without the binary.
type fakeFFmpegRunner struct {
	mu   sync.Mutex
	pcm  []byte
	name string
}

func (r *fakeFFmpegRunner) Run(_ context.Context, name string, _ []byte, _ ...string) ([]byte, error) {
	r.mu.Lock()
	r.name = name
	r.mu.Unlock()
	return r.pcm, nil
}

// TestEnrollSpeakerProfile_PersistsAndSurfacesMetadata is the regression for the
// "0.0s" bug: enrollment metadata reported by the resource must be persisted and
// then surfaced by list/status (previously discarded — list/status always read
// 0). It also proves the audio is normalized to canonical-PCM WAV before enroll.
func TestEnrollSpeakerProfile_PersistsAndSurfacesMetadata(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)

	spStore := newSpeakerStoreT(t)
	res, client := newCapturingEnrollResource(t, 2.5)

	// 320 bytes = 10ms of canonical PCM (16kHz mono s16le); the fake "decode".
	runner := &fakeFFmpegRunner{pcm: make([]byte, 320)}
	engine := audioformat.New(
		audioformat.WithRunner(runner),
		audioformat.WithFfmpegProbe(func() bool { return true }),
	)

	c := newSTTClient(t, Deps{Speaker: spStore, SpeakerResource: client, Engine: engine})

	enrollResp, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		Audio:       []byte("fake-webm-opus-bytes"),
		Format:      commonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		DisplayName: "Laptop",
	}))
	require.NoError(t, err)
	require.InDelta(t, 2.5, enrollResp.Msg.GetEnrollment().GetVoicedSeconds(), 1e-9)
	require.Equal(t, int32(1), enrollResp.Msg.GetEnrollment().GetClipCount())
	require.InDelta(t, 2.5, enrollResp.Msg.GetEnrollment().GetTotalVoicedSeconds(), 1e-9)

	// The resource received WAV-wrapped canonical PCM, not the raw WebM bytes —
	// matching the verify path's preprocessing.
	require.Equal(t, "ffmpeg", runner.name, "webm enroll must go through the ffmpeg decode")
	got := res.audio()
	require.GreaterOrEqual(t, len(got), 4)
	require.Equal(t, "RIFF", string(got[:4]), "enrollment audio must be WAV-wrapped canonical PCM")

	// THE BUG FIX: list reads the PERSISTED metadata (was always 0 before).
	list, err := c.ListSpeakerProfiles(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfilesRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetProfiles(), 1)
	p := list.Msg.GetProfiles()[0]
	require.Equal(t, int32(1), p.GetClipCount())
	require.InDelta(t, 2.5, p.GetTotalVoicedSeconds(), 1e-9)
	require.Equal(t, int32(16000), p.GetSampleRate())
	require.Equal(t, int32(192), p.GetEmbeddingDim())
	require.Equal(t, "speechbrain/spkrec-ecapa-voxceleb", p.GetModelName())

	// Status surfaces it through the same projection.
	status, err := c.GetSpeakerStatus(context.Background(), connect.NewRequest(&sttv1.GetSpeakerStatusRequest{}))
	require.NoError(t, err)
	require.Len(t, status.Msg.GetStatus().GetProfiles(), 1)
	require.Equal(t, int32(1), status.Msg.GetStatus().GetProfiles()[0].GetClipCount())
	require.InDelta(t, 2.5, status.Msg.GetStatus().GetProfiles()[0].GetTotalVoicedSeconds(), 1e-9)
}

// TestEnrollSpeakerProfile_UnknownFormatEnrollsRaw proves the degraded path:
// when the engine is wired but the format is unknown, enrollment still succeeds
// by sending the raw bytes (the resource decodes by sniffing) rather than
// failing — fidelity is reduced, not the feature.
func TestEnrollSpeakerProfile_UnknownFormatEnrollsRaw(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)

	spStore := newSpeakerStoreT(t)
	res, client := newCapturingEnrollResource(t, 1.0)
	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true }))

	c := newSTTClient(t, Deps{Speaker: spStore, SpeakerResource: client, Engine: engine})
	_, err := c.EnrollSpeakerProfile(context.Background(), connect.NewRequest(&sttv1.EnrollSpeakerProfileRequest{
		Audio:       []byte("raw-unsniffable-bytes"),
		Format:      commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED,
		DisplayName: "Laptop",
	}))
	require.NoError(t, err)
	require.Equal(t, []byte("raw-unsniffable-bytes"), res.audio(), "unknown format -> raw bytes passed through")
}

func TestSpeakerClipToProto_PreservesMetadataAndToleratesInvalidTime(t *testing.T) {
	clip := speakerClipToProto(sttpipeline.SpeakerProfileClip{
		ClipID: "clip-1", Label: "office", VoicedSeconds: 1.25, AudioSeconds: 2,
		CreatedAt: "2026-07-24T20:00:00Z", EmbeddingDim: 192,
	})
	require.Equal(t, "clip-1", clip.GetClipId())
	require.Equal(t, "office", clip.GetLabel())
	require.Equal(t, int32(192), clip.GetEmbeddingDim())
	require.NotNil(t, clip.GetCreatedAt())
	invalid := speakerClipToProto(sttpipeline.SpeakerProfileClip{CreatedAt: "not-a-time"})
	require.Nil(t, invalid.GetCreatedAt())
}

func TestListSpeakerProfileClips_ProjectsResourceMetadata(t *testing.T) {
	_, resource := newCapturingEnrollResource(t, 1)
	c := newSTTClient(t, Deps{SpeakerResource: resource})
	resp, err := c.ListSpeakerProfileClips(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfileClipsRequest{ProfileId: "profile-1"}))
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Msg.GetCount())
	require.Equal(t, "office", resp.Msg.GetClips()[0].GetLabel())
}

func TestSpeakerClipOperations_RequireResource(t *testing.T) {
	c := newSTTClient(t, Deps{})
	_, err := c.ListSpeakerProfileClips(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfileClipsRequest{ProfileId: "missing"}))
	require.Error(t, err)
	_, err = c.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{ProfileId: "missing", ClipId: "clip"}))
	require.Error(t, err)
}

func TestDeleteSpeakerProfileClip_ProjectsResourceResult(t *testing.T) {
	_, resource := newCapturingEnrollResource(t, 1)
	c := newSTTClient(t, Deps{SpeakerResource: resource})
	resp, err := c.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{ProfileId: "profile-1", ClipId: "clip-1"}))
	require.NoError(t, err)
	require.Equal(t, "profile-1", resp.Msg.GetProfileId())
	require.Equal(t, "clip-1", resp.Msg.GetClipId())
	require.Equal(t, int32(1), resp.Msg.GetClipCount())
}

package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/store"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func newDeleteClipResource(t *testing.T, status int, result sttpipeline.SpeakerClipDeleteResult) *sttpipeline.SpeakerClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
	t.Cleanup(srv.Close)
	return &sttpipeline.SpeakerClient{BaseURL: srv.URL, Doer: http.DefaultClient}
}

func TestDeleteSpeakerProfileClip_DeletedProfilePurgesLocalRow(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)

	spStore := newSpeakerStoreT(t)
	require.NoError(t, spStore.Upsert(context.Background(), store.SpeakerProfile{ID: "profile-1", Name: "Laptop"}))
	resource := newDeleteClipResource(t, http.StatusOK, sttpipeline.SpeakerClipDeleteResult{
		ProfileID: "profile-1", ClipID: "clip-1", DeletedProfile: true,
	})
	c := newSTTClient(t, Deps{Speaker: spStore, SpeakerResource: resource})

	resp, err := c.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{
		ProfileId: "profile-1", ClipId: "clip-1",
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDeletedProfile())
	require.Equal(t, "profile-1", resp.Msg.GetProfileId())

	_, found, err := spStore.Get(context.Background(), "profile-1")
	require.NoError(t, err)
	require.False(t, found, "the last clip deleted the profile on the resource, so the local row must go")
}

func TestDeleteSpeakerProfileClip_RefreshesSurvivingProfile(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)

	spStore := newSpeakerStoreT(t)
	require.NoError(t, spStore.Upsert(context.Background(), store.SpeakerProfile{
		ID: "profile-1", Name: "Laptop", ClipCount: 5, TotalVoicedSeconds: 10,
	}))
	resource := newDeleteClipResource(t, http.StatusOK, sttpipeline.SpeakerClipDeleteResult{
		ProfileID: "profile-1", ClipID: "clip-1", ClipCount: 2, TotalVoicedSeconds: 3.5,
	})
	c := newSTTClient(t, Deps{Speaker: spStore, SpeakerResource: resource})

	resp, err := c.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{
		ProfileId: "profile-1", ClipId: "clip-1",
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetDeletedProfile())
	require.EqualValues(t, 2, resp.Msg.GetClipCount())

	row, found, err := spStore.Get(context.Background(), "profile-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 2, row.ClipCount)
	require.Equal(t, 3.5, row.TotalVoicedSeconds)
}

func TestDeleteSpeakerProfileClip_MapsResourceFailureToUnavailable(t *testing.T) {
	resource := newDeleteClipResource(t, http.StatusInternalServerError, sttpipeline.SpeakerClipDeleteResult{})
	c := newSTTClient(t, Deps{SpeakerResource: resource})
	_, err := c.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{
		ProfileId: "profile-1", ClipId: "clip-1",
	}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestSpeakerCodecMapsEveryIngressVocabulary(t *testing.T) {
	cases := map[string]audioformat.Codec{
		" pcm ":       audioformat.CodecPCMS16LE,
		"PCM_S16LE":   audioformat.CodecPCMS16LE,
		"wav":         audioformat.CodecWAV,
		"mp3":         audioformat.CodecMP3,
		"flac":        audioformat.CodecFLAC,
		"ogg":         audioformat.CodecOGG,
		"webm":        audioformat.CodecWebM,
		"opus":        audioformat.CodecOpus,
		"aac":         audioformat.CodecAAC,
		"unsupported": audioformat.CodecUnknown,
	}
	for input, want := range cases {
		require.Equal(t, want, speakerCodec(input), "input %q", input)
	}
}

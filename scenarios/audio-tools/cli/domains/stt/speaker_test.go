package stt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestSpeakerConfigSet(t *testing.T) {
	var gotMask []string
	app := mountSTT(t, &fakeSvc{
		updateSpeakerCfg: func(req *sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error) {
			gotMask = req.GetUpdateMask().GetPaths()
			return &sttv1.SpeakerConfig{
				Enabled:   req.GetConfig().GetEnabled(),
				Mode:      req.GetConfig().GetMode(),
				Threshold: req.GetConfig().GetThreshold(),
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
		{Name: "min-decision-seconds"},
		{Name: "score-smoothing"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"mode": "filter", "threshold": "0.8", "enabled": "true"}, JSON: true,
	})
	require.NoError(t, h.speakerConfig(ctx))
	require.ElementsMatch(t, []string{"mode", "threshold", "enabled"}, gotMask)
	out := buf.String()
	require.Contains(t, out, "mode            = filter")
	require.Contains(t, out, "threshold       = 0.80")
	require.Contains(t, out, "enabled         = true")
}

// --min-decision-seconds and --score-smoothing map to their session-decision
// mask paths and are echoed in the resolved-config output.

func TestSpeakerConfigSetSessionDecision(t *testing.T) {
	var gotMask []string
	app := mountSTT(t, &fakeSvc{
		updateSpeakerCfg: func(req *sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error) {
			gotMask = req.GetUpdateMask().GetPaths()
			return &sttv1.SpeakerConfig{
				MinDecisionSeconds: req.GetConfig().GetMinDecisionSeconds(),
				ScoreSmoothing:     req.GetConfig().GetScoreSmoothing(),
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
		{Name: "min-decision-seconds"},
		{Name: "score-smoothing"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"min-decision-seconds": "2.5", "score-smoothing": "0.3"},
	})
	require.NoError(t, h.speakerConfig(ctx))
	require.ElementsMatch(t, []string{"min_decision_seconds", "score_smoothing"}, gotMask)
	out := buf.String()
	require.Contains(t, out, "min_decision_s  = 2.5")
	require.Contains(t, out, "score_smoothing = 0.30")
}

// speaker-clips lists the enrollment clips of a profile.

func TestSpeakerClips(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		listClips: func(req *sttv1.ListSpeakerProfileClipsRequest) *sttv1.ListSpeakerProfileClipsResponse {
			require.Equal(t, "p1", req.GetProfileId())
			return &sttv1.ListSpeakerProfileClipsResponse{
				ProfileId: "p1", Count: 2,
				Clips: []*sttv1.SpeakerProfileClip{
					{ClipId: "c1", Label: "laptop-normal", VoicedSeconds: 3.2},
					{ClipId: "c2", Label: "phone-whisper", VoicedSeconds: 2.1},
				},
			}
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "profile"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"profile": "p1"},
	})
	require.NoError(t, h.speakerClips(ctx))
	out := buf.String()
	require.Contains(t, out, "Profile p1: 2 clip(s)")
	require.Contains(t, out, "c1")
	require.Contains(t, out, "laptop-normal")
	require.Contains(t, out, "phone-whisper")
}

// speaker-delete-clip reports the recomputed clip count; deleting the last clip
// reports profile removal.

func TestSpeakerDeleteClip(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		deleteClip: func(req *sttv1.DeleteSpeakerProfileClipRequest) *sttv1.DeleteSpeakerProfileClipResponse {
			require.Equal(t, "p1", req.GetProfileId())
			require.Equal(t, "c1", req.GetClipId())
			return &sttv1.DeleteSpeakerProfileClipResponse{
				ProfileId: "p1", ClipId: "c1", DeletedProfile: false,
				ClipCount: 1, TotalVoicedSeconds: 2.1,
			}
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "profile"}, {Name: "clip"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"profile": "p1", "clip": "c1"},
	})
	require.NoError(t, h.speakerDeleteClip(ctx))
	require.Contains(t, buf.String(), "1 clip(s) remain")
}

func TestSpeakerClipPrimitivesPreserveHumanAndJSONContracts(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		listClips: func(req *sttv1.ListSpeakerProfileClipsRequest) *sttv1.ListSpeakerProfileClipsResponse {
			return &sttv1.ListSpeakerProfileClipsResponse{ProfileId: req.GetProfileId(), Count: 1, Clips: []*sttv1.SpeakerProfileClip{{ClipId: "c1", Label: "laptop", VoicedSeconds: 2}}}
		},
		deleteClip: func(req *sttv1.DeleteSpeakerProfileClipRequest) *sttv1.DeleteSpeakerProfileClipResponse {
			return &sttv1.DeleteSpeakerProfileClipResponse{ProfileId: req.GetProfileId(), ClipId: req.GetClipId(), DeletedProfile: true}
		},
	})
	h := newHandlers(app)
	clipSchema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "profile"}}}
	ctx, out := cliapptest.NewCapturedRunContext(app, clipSchema, cliapptest.TestRunContextOptions{Flags: map[string]string{"profile": "p1"}})
	require.NoError(t, h.speakerClipsPrimitive().Run(ctx))
	require.Contains(t, out.String(), "Profile p1: 1 clip(s)")
	require.Contains(t, out.String(), `c1 (label="laptop", 2.0s voiced)`)
	ctx, out = cliapptest.NewCapturedRunContext(app, clipSchema, cliapptest.TestRunContextOptions{Flags: map[string]string{"profile": "p1"}, JSON: true})
	require.NoError(t, h.speakerClipsPrimitive().Run(ctx))
	var clips sttv1.ListSpeakerProfileClipsResponse
	require.NoError(t, json.Unmarshal(out.Bytes(), &clips))
	require.Equal(t, "p1", clips.GetProfileId())

	deleteSchema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "profile"}, {Name: "clip"}}}
	ctx, out = cliapptest.NewCapturedRunContext(app, deleteSchema, cliapptest.TestRunContextOptions{Flags: map[string]string{"profile": "p1", "clip": "c1"}})
	require.NoError(t, h.speakerDeleteClipPrimitive().Run(ctx))
	require.Contains(t, out.String(), "that was the last clip, so profile p1 was removed")
	ctx, out = cliapptest.NewCapturedRunContext(app, deleteSchema, cliapptest.TestRunContextOptions{Flags: map[string]string{"profile": "p1", "clip": "c1"}, JSON: true})
	require.NoError(t, h.speakerDeleteClipPrimitive().Run(ctx))
	var deleted sttv1.DeleteSpeakerProfileClipResponse
	require.NoError(t, json.Unmarshal(out.Bytes(), &deleted))
	require.True(t, deleted.GetDeletedProfile())
}

func TestSpeakerEnrollPrimitivePreservesBatchAndJSON(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "one.wav"), filepath.Join(dir, "two.wav")
	require.NoError(t, os.WriteFile(first, []byte("one"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("two"), 0o600))
	var calls int
	app := mountSTT(t, &fakeSvc{enrollProfile: func(req *sttv1.EnrollSpeakerProfileRequest) *sttv1.EnrollSpeakerProfileResponse {
		calls++
		require.Equal(t, []byte(map[int]string{1: "one", 2: "two"}[calls]), req.GetAudio())
		if calls == 2 {
			require.Equal(t, "p1", req.GetProfileId())
		}
		return &sttv1.EnrollSpeakerProfileResponse{Enrollment: &sttv1.SpeakerEnrollment{ProfileId: "p1", ClipId: fmt.Sprintf("c%d", calls), Label: "laptop", VoicedSeconds: 2, ClipCount: int32(calls), TotalVoicedSeconds: float64(calls * 2), ModelName: "embed", EmbeddingDim: 64}, Config: &sttv1.SpeakerConfig{Enabled: true, ProfileIds: []string{"p1"}}}
	}})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file"}, {Name: "profile"}, {Name: "name"}, {Name: "label"}, {Name: "notes"}, {Name: "format"}, {Name: "activate"}}}
	flags := map[string]string{"file": first + "," + second, "activate": "true", "label": "laptop"}
	ctx, out := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Flags: flags})
	require.NoError(t, h.speakerEnrollPrimitive().Run(ctx))
	require.Contains(t, out.String(), "Enrolling 2 clip(s):")
	require.Contains(t, out.String(), "[1] "+first+" -> clip c1")
	require.Contains(t, out.String(), "Profile p1: 2 clip(s), 4.0s total voiced; model embed (dim 64)")
	humanOutput := out.String()
	calls = 0
	ctx, out = cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{Flags: flags, JSON: true})
	require.NoError(t, h.speakerEnrollPrimitive().Run(ctx))
	require.Equal(t, humanOutput, out.String(), "legacy JSON-mode output is plain text")
}

// --extraction-enabled maps to the extraction_enabled mask path and is echoed
// in the resolved-config output.

func TestSpeakerConfigSetExtraction(t *testing.T) {
	var gotMask []string
	app := mountSTT(t, &fakeSvc{
		updateSpeakerCfg: func(req *sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error) {
			gotMask = req.GetUpdateMask().GetPaths()
			return &sttv1.SpeakerConfig{ExtractionEnabled: req.GetConfig().GetExtractionEnabled()}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
		{Name: "min-decision-seconds"},
		{Name: "score-smoothing"},
	}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"extraction-enabled": "true"},
	})
	require.NoError(t, h.speakerConfig(ctx))
	require.ElementsMatch(t, []string{"extraction_enabled"}, gotMask)
	require.Contains(t, buf.String(), "extraction      = true")
}

// speaker-config with no flags is a usage error (nothing to update).

func TestSpeakerConfigNoFlags(t *testing.T) {
	app := mountSTT(t, &fakeSvc{})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "mode"},
		{Name: "threshold"},
		{Name: "enabled"},
		{Name: "profiles"},
		{Name: "bind-profile"},
		{Name: "reject-behavior"},
		{Name: "fallback"},
		{Name: "extraction-enabled"},
		{Name: "min-decision-seconds"},
		{Name: "score-smoothing"},
	}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{})
	require.Error(t, h.speakerConfig(ctx))
}

// speaker-status surfaces the Whisper-only protection caveat when verification
// is enabled, so a user on the streaming engine isn't silently unprotected.

func TestSpeakerStatusWhisperCaveat(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getSpeakerStatus: func() *sttv1.SpeakerStatus {
			return &sttv1.SpeakerStatus{
				Config: &sttv1.SpeakerConfig{
					Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
					Threshold: 0.7, ProfileIds: []string{"my-voice"},
					ExtractionEnabled: true,
				},
				Capability: "available", CapabilityLabel: "Speaker store",
				ResourceReady: true, ProfileCount: 1,
				Profiles: []*sttv1.SpeakerProfile{{Id: "my-voice", DisplayName: "Me", ClipCount: 2, TotalVoicedSeconds: 12.5}},
			}
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.speakerStatus(ctx))
	out := buf.String()
	require.Contains(t, out, "mode            = filter")
	require.Contains(t, out, "extraction      = true")
	require.Contains(t, out, "my-voice")
	require.Contains(t, out, "[active]")
	require.Contains(t, out, "Kyutai streaming engine emits no per-segment audio")
}

func TestSpeakerStatusJSONUsesProtoShape(t *testing.T) {
	app := mountSTT(t, &fakeSvc{
		getSpeakerStatus: func() *sttv1.SpeakerStatus {
			return &sttv1.SpeakerStatus{
				Config: &sttv1.SpeakerConfig{
					Enabled: true,
					Mode:    sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
				},
				Capability:      "available",
				CapabilityLabel: "Speaker store",
				ResourceReady:   true,
				ProfileCount:    1,
				Profiles:        []*sttv1.SpeakerProfile{{Id: "my-voice", DisplayName: "Me"}},
			}
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.speakerStatus(ctx))
	out := buf.String()
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	require.Equal(t, "available", payload["capability"])
	require.Equal(t, true, payload["resource_ready"])
	require.NotEmpty(t, payload["profiles"])
	require.NotContains(t, out, "Speaker verification:")
}

// formats prints the capability matrix as human-readable lines and surfaces
// the canonical PCM target. When ffmpeg is present, no degrade note appears.

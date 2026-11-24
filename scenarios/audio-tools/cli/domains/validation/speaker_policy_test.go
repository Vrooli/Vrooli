package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type policySTT struct {
	sttconnect.STTServiceClient
	admin          *policyAdmin
	batchErr, leak bool
}

func (f *policySTT) Transcribe(_ context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	if !bytes.Equal(f.admin.audio, req.Msg.GetAudio()) || req.Msg.GetFormat() != commonv1.AudioFormat_AUDIO_FORMAT_WAV {
		return nil, errors.New("unexpected batch request payload")
	}
	if f.batchErr {
		return nil, errors.New("negative batch unavailable")
	}
	if f.admin.negative && f.leak {
		return connect.NewResponse(&sttv1.TranscribeResponse{Text: "leaked"}), nil
	}
	if f.admin.negative {
		return connect.NewResponse(&sttv1.TranscribeResponse{Filtered: true}), nil
	}
	return connect.NewResponse(&sttv1.TranscribeResponse{Text: "private transcript", ProviderId: "whisper-stt", ModelId: "whisper-large-v3-turbo", PolicyDetails: map[string]string{"speaker_policy": "degraded", "speaker_fallback_used": "true"}}), nil
}

func (f *policySTT) ListEngines(context.Context, *connect.Request[sttv1.ListEnginesRequest]) (*connect.Response[sttv1.ListEnginesResponse], error) {
	return connect.NewResponse(&sttv1.ListEnginesResponse{Engines: []*sttv1.EngineInfo{{Id: "sherpa-streaming", Available: true, IsActive: true, Verdict: "selected"}}}), nil
}

type policyAdmin struct {
	sttconnect.STTAdminServiceClient
	config      *sttv1.SpeakerConfig
	audio       []byte
	negative    bool
	failRestore bool
	updates     []*sttv1.SpeakerConfig
}

func (f *policyAdmin) GetSpeakerConfig(context.Context, *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	return connect.NewResponse(&sttv1.GetSpeakerConfigResponse{Config: f.config}), nil
}

func (f *policyAdmin) UpdateSpeakerConfig(_ context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	f.updates = append(f.updates, req.Msg.GetConfig())
	paths := req.Msg.GetUpdateMask().GetPaths()
	profiles := req.Msg.GetConfig().GetProfileIds()
	if f.failRestore && len(paths) > 3 && len(profiles) == 1 && profiles[0] == "original" {
		return nil, errors.New("restore unavailable")
	}
	if len(paths) == 3 {
		f.negative = true
	}
	if len(paths) > 3 {
		f.negative = false
	}
	return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: req.Msg.GetConfig()}), nil
}

type policyExperiment struct {
	experimentconnect.ExperimentServiceClient
	recorded *experimentv1.QualificationEvidence
}

func (f *policyExperiment) RecordQualificationEvidence(_ context.Context, req *connect.Request[experimentv1.RecordQualificationEvidenceRequest]) (*connect.Response[experimentv1.RecordQualificationEvidenceResponse], error) {
	f.recorded = req.Msg.GetEvidence()
	return connect.NewResponse(&experimentv1.RecordQualificationEvidenceResponse{}), nil
}

func runPolicyHarness(t *testing.T, batchErr, leak, failRestore bool) (error, *policyAdmin, *policyExperiment, *bytes.Buffer) {
	t.Helper()
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "speaker-policy-test", Version: "test", Description: "test", DefaultAPIBase: "http://127.0.0.1", AllowAnonymous: true})
	require.NoError(t, err)
	ctx, out := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "device-profile"}}}, cliapptest.TestRunContextOptions{JSON: true})
	audio := []byte("audio bytes must not enter receipt")
	admin := &policyAdmin{config: &sttv1.SpeakerConfig{Mode: sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, ProfileIds: []string{"original"}, Threshold: .8}, audio: audio, failRestore: failRestore}
	stt := &policySTT{admin: admin, batchErr: batchErr, leak: leak}
	exp := &policyExperiment{}
	h := newHandlersWithClock(app, func() time.Time { return time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC) })
	calls := 0
	runner := func(_ context.Context, got []byte, _ func() time.Time, skip bool) (speakerStreamObservation, error) {
		if !bytes.Equal(audio, got) || skip {
			return speakerStreamObservation{}, errors.New("unexpected stream request")
		}
		calls++
		if calls == 1 {
			return speakerStreamObservation{Policies: []string{"degraded"}, TextSegments: 1, ProviderID: "sherpa-stt", ModelID: "zipformer-en-2023-06-26"}, nil
		}
		return speakerStreamObservation{Rejections: 1, ProviderID: "sherpa-stt", ModelID: "zipformer-en-2023-06-26"}, nil
	}
	err = h.runSpeakerPolicyWithClients(ctx, audio, "build-test", stt, runner, admin, exp)
	return err, admin, exp, out
}

func TestRunSpeakerPolicyRestoresAndRecordsPayloadFreeReceipt(t *testing.T) {
	err, admin, exp, out := runPolicyHarness(t, false, false, false)
	require.NoError(t, err)
	require.Len(t, admin.updates, 3)
	probe := admin.updates[0]
	require.True(t, probe.GetEnabled())
	require.Equal(t, sttv1.SpeakerMode_SPEAKER_MODE_FILTER, probe.GetMode())
	require.True(t, probe.GetFallbackWithoutVerification())
	require.Len(t, probe.GetProfileIds(), 1)
	require.Contains(t, probe.GetProfileIds()[0], "-unregistered")
	require.Equal(t, probe.GetProfileIds(), admin.updates[1].GetProfileIds())
	require.False(t, admin.updates[1].GetFallbackWithoutVerification())
	require.Equal(t, "original", admin.updates[2].GetProfileIds()[0])
	require.NotNil(t, exp.recorded)
	require.True(t, exp.recorded.GetPassed())
	require.Equal(t, "sherpa-streaming", exp.recorded.GetEngineId())
	require.Equal(t, "zipformer-en-2023-06-26", exp.recorded.GetModelId())
	require.NotContains(t, exp.recorded.GetMachineJson(), "audio bytes")
	require.NotContains(t, exp.recorded.GetMachineJson(), "private transcript")
	var receipt map[string]any
	require.NoError(t, json.Unmarshal([]byte(exp.recorded.GetMachineJson()), &receipt))
	require.Equal(t, "native", receipt["execution_lane"])
	cell := receipt["provider_cell"].(map[string]any)
	require.Equal(t, "sherpa-streaming", cell["engine_id"])
	require.Equal(t, "sherpa-stt", cell["provider_id"])
	require.Equal(t, "zipformer-en-2023-06-26", cell["model_id"])
	details := receipt["details"].(map[string]any)
	require.Equal(t, "whisper-stt", details["batch_provider_id"])
	require.Equal(t, "whisper-large-v3-turbo", details["batch_model_id"])
	require.Equal(t, "unregistered-profile-degraded-then-fail-closed", details["verification_probe"])
	require.NotContains(t, out.String(), "private transcript")
}

func TestRunSpeakerPolicyRestoresOnEarlyFailureWithoutEvidence(t *testing.T) {
	err, admin, exp, _ := runPolicyHarness(t, true, false, false)
	require.ErrorContains(t, err, "speaker-policy batch baseline")
	require.Len(t, admin.updates, 2)
	require.Equal(t, "original", admin.updates[1].GetProfileIds()[0])
	require.Nil(t, exp.recorded)
}

func TestRunSpeakerPolicyFailsNegativeLeakAndRestores(t *testing.T) {
	err, admin, exp, _ := runPolicyHarness(t, false, true, false)
	require.ErrorContains(t, err, "negative batch control emitted text")
	require.Len(t, admin.updates, 3)
	require.Equal(t, "original", admin.updates[2].GetProfileIds()[0])
	require.Nil(t, exp.recorded)
}

func TestRunSpeakerPolicyDoesNotRecordWhenRestoreFails(t *testing.T) {
	err, _, exp, _ := runPolicyHarness(t, false, false, true)
	require.ErrorContains(t, err, "restore speaker config")
	require.Nil(t, exp.recorded)
}

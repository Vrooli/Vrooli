package audioports

import (
	"context"
	"errors"
	"testing"

	"web-console/integrations/audiotools"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestRemotePortsReportUnavailableWithoutClient(t *testing.T) {
	ctx := context.Background()
	if _, err := (*RemoteSpeechToText)(nil).Transcribe(ctx, nil, STTOptions{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("stt: %v", err)
	}
	if _, err := (*RemoteTextToSpeech)(nil).Synthesize(ctx, TTSRequest{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("tts: %v", err)
	}
	if _, err := (*RemoteTextToSpeech)(nil).ListVoices(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("voices: %v", err)
	}
	if _, hit := (*RemoteTextToSpeech)(nil).GetCached(ctx, CacheLookup{}); hit {
		t.Error("nil cache should miss")
	}
	if err := (*RemotePlaybackEventRecorder)(nil).RecordPlaybackEvent(ctx, PlaybackEvent{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("playback: %v", err)
	}
	if _, err := (*RemoteSummarizer)(nil).Summarize(ctx, SummarizeInput{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("summarizer: %v", err)
	}
	if _, err := (*RemoteStreamConfigAdmin)(nil).GetStreamConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("stream get: %v", err)
	}
	if _, err := (*RemoteStreamConfigAdmin)(nil).UpdateStreamConfig(ctx, FieldMask{}, StreamConfig{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("stream update: %v", err)
	}
	if _, err := (*RemoteWakeWordAdmin)(nil).GetWakeWordConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("wake get: %v", err)
	}
	if _, err := (*RemoteWakeWordAdmin)(nil).UpdateWakeWordTemplate(ctx, WakeWordTemplate{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("wake update: %v", err)
	}
	if _, err := (*RemoteWakeWordAdmin)(nil).DeleteWakeWordTemplate(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("wake delete: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).GetSpeakerConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker config: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).UpdateSpeakerConfig(ctx, FieldMask{}, SpeakerConfig{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker update: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).GetSpeakerStatus(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker status: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).ListSpeakerProfiles(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker profiles: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).EnrollSpeakerProfile(ctx, EnrollSpeakerInput{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker enroll: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).ClearSpeakerProfileBinding(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker clear: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).UnbindSpeakerProfile(ctx, ""); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker unbind: %v", err)
	}
	if _, err := (*RemoteSpeakerAdmin)(nil).DeleteSpeakerProfile(ctx, ""); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("speaker delete: %v", err)
	}
	if _, err := (*RemoteTTSConfigAdmin)(nil).GetTTSConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("tts config: %v", err)
	}
	if _, err := (*RemoteTTSConfigAdmin)(nil).UpdateTTSConfig(ctx, FieldMask{}, TTSConfig{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("tts config update: %v", err)
	}
	if _, err := (*RemoteSummarizeConfigAdmin)(nil).GetSummarizeConfig(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("summary config: %v", err)
	}
	if _, err := (*RemoteSummarizeConfigAdmin)(nil).UpdateSummarizeConfig(ctx, FieldMask{}, SummarizeConfig{}); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("summary config update: %v", err)
	}
	if _, err := (*RemoteSummarizeConfigAdmin)(nil).ListSummarizeModels(ctx); !errors.Is(err, audiotools.ErrUnavailable) {
		t.Errorf("summary models: %v", err)
	}
}

func TestAudioPortPureMappingsAndPassthrough(t *testing.T) {
	if got := responseFormatFromString("wat"); got != 0 {
		t.Fatalf("unknown response format: %v", got)
	}
	for _, level := range []string{"", "light", "moderate", "heavy", "other"} {
		_ = summarizeLevelFromString(level)
	}
	for _, tier := range []int32{0, 1, 2, 3, 99} {
		_ = providerTierToString(commonv1.ProviderTier(tier))
	}
	p := PassthroughSpeechTextProcessor{}
	if p.NormalizeForSpeech("x") != "x" || p.SplitIntoParagraphs("") != nil || len(p.SplitIntoParagraphs("x")) != 1 {
		t.Fatal("passthrough processor")
	}
}

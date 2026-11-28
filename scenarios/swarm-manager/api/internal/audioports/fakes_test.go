package audioports

import (
	"context"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// fakeResolver returns a static URL so audiotools.New() resolves immediately
// and Ensure() is a no-op afterward.
type fakeResolver struct{}

func (fakeResolver) Resolve() (string, error) { return "http://127.0.0.1:1", nil }

// newTestClient builds a real *audiotools.Client whose URL resolves but whose
// generated clients we overwrite with fakes. It must be resolved already so
// Ensure() short-circuits (we never want real network I/O in tests).
func newTestClient(t interface{ Fatalf(string, ...any) }) *audiotools.Client {
	c, err := audiotools.New(fakeResolver{}, audiotools.Policy{})
	if err != nil {
		t.Fatalf("audiotools.New: %v", err)
	}
	if !c.Resolved() {
		t.Fatalf("expected client to be resolved after New")
	}
	return c
}

// unavailableErr is a connect transport-failure error used to assert the
// HandleTransportFailure() / re-resolve path.
func unavailableErr() error {
	return connect.NewError(connect.CodeUnavailable, errUnavailable)
}

var errUnavailable = errSentinel("boom")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

// -----------------------------------------------------------------------------
// fakeSTT implements sttconnect.STTServiceClient. Only Transcribe is wired;
// everything else panics so a miswired call is caught.
// -----------------------------------------------------------------------------

type fakeSTT struct {
	resp *sttv1.TranscribeResponse
	err  error
}

var _ sttconnect.STTServiceClient = (*fakeSTT)(nil)

func (f *fakeSTT) Transcribe(_ context.Context, _ *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.resp == nil {
		return nil, nil
	}
	return connect.NewResponse(f.resp), nil
}

func (f *fakeSTT) TranscribeStream(context.Context) *connect.BidiStreamForClient[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent] {
	panic("unexpected TranscribeStream")
}

func (f *fakeSTT) GetSupportedFormats(context.Context, *connect.Request[sttv1.GetSupportedFormatsRequest]) (*connect.Response[sttv1.GetSupportedFormatsResponse], error) {
	panic("unexpected GetSupportedFormats")
}

func (f *fakeSTT) ListEngines(context.Context, *connect.Request[sttv1.ListEnginesRequest]) (*connect.Response[sttv1.ListEnginesResponse], error) {
	panic("unexpected ListEngines")
}

// -----------------------------------------------------------------------------
// fakeSTTAdmin implements sttconnect.STTAdminServiceClient. Per-test the
// caller installs the closure for the one method under test; the rest panic.
// -----------------------------------------------------------------------------

type fakeSTTAdmin struct {
	getStreamConfig    func() (*sttv1.GetStreamConfigResponse, error)
	updateStreamConfig func() (*sttv1.UpdateStreamConfigResponse, error)

	getWakeWordConfig   func() (*sttv1.GetWakeWordConfigResponse, error)
	updateWakeWord      func() (*sttv1.UpdateWakeWordTemplateResponse, error)
	deleteWakeWord      func() (*sttv1.DeleteWakeWordTemplateResponse, error)
	getSpeakerConfig    func() (*sttv1.GetSpeakerConfigResponse, error)
	updateSpeakerConfig func() (*sttv1.UpdateSpeakerConfigResponse, error)
	getSpeakerStatus    func() (*sttv1.GetSpeakerStatusResponse, error)
	listSpeakerProfiles func() (*sttv1.ListSpeakerProfilesResponse, error)
	enrollSpeaker       func(*sttv1.EnrollSpeakerProfileRequest) (*sttv1.EnrollSpeakerProfileResponse, error)
	clearSpeakerBinding func() (*sttv1.ClearSpeakerProfileBindingResponse, error)
	unbindSpeaker       func() (*sttv1.UnbindSpeakerProfileResponse, error)
	deleteSpeaker       func() (*sttv1.DeleteSpeakerProfileResponse, error)
}

var _ sttconnect.STTAdminServiceClient = (*fakeSTTAdmin)(nil)

func mustResp[T any](v *T, err error) (*connect.Response[T], error) {
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return connect.NewResponse(v), nil
}

func (f *fakeSTTAdmin) GetStreamConfig(context.Context, *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	return mustResp(f.getStreamConfig())
}

func (f *fakeSTTAdmin) UpdateStreamConfig(context.Context, *connect.Request[sttv1.UpdateStreamConfigRequest]) (*connect.Response[sttv1.UpdateStreamConfigResponse], error) {
	return mustResp(f.updateStreamConfig())
}

func (f *fakeSTTAdmin) GetEngineSwitchImpact(context.Context, *connect.Request[sttv1.GetEngineSwitchImpactRequest]) (*connect.Response[sttv1.GetEngineSwitchImpactResponse], error) {
	panic("unexpected GetEngineSwitchImpact")
}

func (f *fakeSTTAdmin) GetWakeWordConfig(context.Context, *connect.Request[sttv1.GetWakeWordConfigRequest]) (*connect.Response[sttv1.GetWakeWordConfigResponse], error) {
	return mustResp(f.getWakeWordConfig())
}

func (f *fakeSTTAdmin) UpdateWakeWordTemplate(context.Context, *connect.Request[sttv1.UpdateWakeWordTemplateRequest]) (*connect.Response[sttv1.UpdateWakeWordTemplateResponse], error) {
	return mustResp(f.updateWakeWord())
}

func (f *fakeSTTAdmin) DeleteWakeWordTemplate(context.Context, *connect.Request[sttv1.DeleteWakeWordTemplateRequest]) (*connect.Response[sttv1.DeleteWakeWordTemplateResponse], error) {
	return mustResp(f.deleteWakeWord())
}

func (f *fakeSTTAdmin) GetSpeakerConfig(context.Context, *connect.Request[sttv1.GetSpeakerConfigRequest]) (*connect.Response[sttv1.GetSpeakerConfigResponse], error) {
	return mustResp(f.getSpeakerConfig())
}

func (f *fakeSTTAdmin) UpdateSpeakerConfig(context.Context, *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	return mustResp(f.updateSpeakerConfig())
}

func (f *fakeSTTAdmin) GetSpeakerStatus(context.Context, *connect.Request[sttv1.GetSpeakerStatusRequest]) (*connect.Response[sttv1.GetSpeakerStatusResponse], error) {
	return mustResp(f.getSpeakerStatus())
}

func (f *fakeSTTAdmin) ListSpeakerProfiles(context.Context, *connect.Request[sttv1.ListSpeakerProfilesRequest]) (*connect.Response[sttv1.ListSpeakerProfilesResponse], error) {
	return mustResp(f.listSpeakerProfiles())
}

func (f *fakeSTTAdmin) EnrollSpeakerProfile(_ context.Context, req *connect.Request[sttv1.EnrollSpeakerProfileRequest]) (*connect.Response[sttv1.EnrollSpeakerProfileResponse], error) {
	return mustResp(f.enrollSpeaker(req.Msg))
}

func (f *fakeSTTAdmin) ClearSpeakerProfileBinding(context.Context, *connect.Request[sttv1.ClearSpeakerProfileBindingRequest]) (*connect.Response[sttv1.ClearSpeakerProfileBindingResponse], error) {
	return mustResp(f.clearSpeakerBinding())
}

func (f *fakeSTTAdmin) UnbindSpeakerProfile(context.Context, *connect.Request[sttv1.UnbindSpeakerProfileRequest]) (*connect.Response[sttv1.UnbindSpeakerProfileResponse], error) {
	return mustResp(f.unbindSpeaker())
}

func (f *fakeSTTAdmin) DeleteSpeakerProfile(context.Context, *connect.Request[sttv1.DeleteSpeakerProfileRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileResponse], error) {
	return mustResp(f.deleteSpeaker())
}

func (f *fakeSTTAdmin) ListSpeakerProfileClips(context.Context, *connect.Request[sttv1.ListSpeakerProfileClipsRequest]) (*connect.Response[sttv1.ListSpeakerProfileClipsResponse], error) {
	panic("unexpected ListSpeakerProfileClips")
}

func (f *fakeSTTAdmin) DeleteSpeakerProfileClip(context.Context, *connect.Request[sttv1.DeleteSpeakerProfileClipRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileClipResponse], error) {
	panic("unexpected DeleteSpeakerProfileClip")
}

// -----------------------------------------------------------------------------
// fakeTTS implements ttsconnect.TTSServiceClient.
// -----------------------------------------------------------------------------

type fakeTTS struct {
	synthesize   func() (*ttsv1.SynthesizeResponse, error)
	listVoices   func() (*ttsv1.ListVoicesResponse, error)
	getCache     func() (*ttsv1.GetCacheResponse, error)
	getConfig    func() (*ttsv1.GetConfigResponse, error)
	updateConfig func() (*ttsv1.UpdateConfigResponse, error)
	recordEvent  func() (*ttsv1.RecordPlaybackEventResponse, error)
	normalize    func() (*ttsv1.NormalizeForSpeechResponse, error)
	split        func() (*ttsv1.SplitParagraphsResponse, error)
}

var _ ttsconnect.TTSServiceClient = (*fakeTTS)(nil)

func (f *fakeTTS) Synthesize(context.Context, *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error) {
	return mustResp(f.synthesize())
}

func (f *fakeTTS) SynthesizeStream(context.Context, *connect.Request[ttsv1.SynthesizeRequest]) (*connect.ServerStreamForClient[ttsv1.AudioFrame], error) {
	panic("unexpected SynthesizeStream")
}

func (f *fakeTTS) ListVoices(context.Context, *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error) {
	return mustResp(f.listVoices())
}

func (f *fakeTTS) GetCache(context.Context, *connect.Request[ttsv1.GetCacheRequest]) (*connect.Response[ttsv1.GetCacheResponse], error) {
	return mustResp(f.getCache())
}

func (f *fakeTTS) GetConfig(context.Context, *connect.Request[ttsv1.GetConfigRequest]) (*connect.Response[ttsv1.GetConfigResponse], error) {
	return mustResp(f.getConfig())
}

func (f *fakeTTS) UpdateConfig(context.Context, *connect.Request[ttsv1.UpdateConfigRequest]) (*connect.Response[ttsv1.UpdateConfigResponse], error) {
	return mustResp(f.updateConfig())
}

func (f *fakeTTS) GetStatus(context.Context, *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error) {
	panic("unexpected GetStatus")
}

func (f *fakeTTS) RecordPlaybackEvent(context.Context, *connect.Request[ttsv1.RecordPlaybackEventRequest]) (*connect.Response[ttsv1.RecordPlaybackEventResponse], error) {
	return mustResp(f.recordEvent())
}

func (f *fakeTTS) GetSupportedFormats(context.Context, *connect.Request[ttsv1.GetSupportedFormatsRequest]) (*connect.Response[ttsv1.GetSupportedFormatsResponse], error) {
	panic("unexpected GetSupportedFormats")
}

func (f *fakeTTS) NormalizeForSpeech(context.Context, *connect.Request[ttsv1.NormalizeForSpeechRequest]) (*connect.Response[ttsv1.NormalizeForSpeechResponse], error) {
	return mustResp(f.normalize())
}

func (f *fakeTTS) SplitParagraphs(context.Context, *connect.Request[ttsv1.SplitParagraphsRequest]) (*connect.Response[ttsv1.SplitParagraphsResponse], error) {
	return mustResp(f.split())
}

// -----------------------------------------------------------------------------
// fakeSummarize implements summconnect.SummarizeServiceClient.
// -----------------------------------------------------------------------------

type fakeSummarize struct {
	summarize    func() (*summv1.SummarizeResponse, error)
	getConfig    func() (*summv1.GetSummarizeConfigResponse, error)
	updateConfig func() (*summv1.UpdateSummarizeConfigResponse, error)
	listModels   func() (*summv1.ListSummarizeModelsResponse, error)
}

var _ summconnect.SummarizeServiceClient = (*fakeSummarize)(nil)

func (f *fakeSummarize) Summarize(context.Context, *connect.Request[summv1.SummarizeRequest]) (*connect.Response[summv1.SummarizeResponse], error) {
	return mustResp(f.summarize())
}

func (f *fakeSummarize) GetSummarizeConfig(context.Context, *connect.Request[summv1.GetSummarizeConfigRequest]) (*connect.Response[summv1.GetSummarizeConfigResponse], error) {
	return mustResp(f.getConfig())
}

func (f *fakeSummarize) UpdateSummarizeConfig(context.Context, *connect.Request[summv1.UpdateSummarizeConfigRequest]) (*connect.Response[summv1.UpdateSummarizeConfigResponse], error) {
	return mustResp(f.updateConfig())
}

func (f *fakeSummarize) ListSummarizeModels(context.Context, *connect.Request[summv1.ListSummarizeModelsRequest]) (*connect.Response[summv1.ListSummarizeModelsResponse], error) {
	return mustResp(f.listModels())
}

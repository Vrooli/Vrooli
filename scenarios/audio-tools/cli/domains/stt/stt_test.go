package stt

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	sttconnect.UnimplementedSTTServiceHandler
	sttconnect.UnimplementedSTTAdminServiceHandler
	transcribe        func(*sttv1.TranscribeRequest) (*sttv1.TranscribeResponse, error)
	transcribeModelID string
	transcribeStream  func(context.Context, *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error
	getCfg            func() (*sttv1.StreamConfig, error)
	listEngines       func() *sttv1.ListEnginesResponse
	engineImpact      func(*sttv1.GetEngineSwitchImpactRequest) (*sttv1.GetEngineSwitchImpactResponse, error)
	getFormats        func() *sttv1.GetSupportedFormatsResponse
	updateSpeakerCfg  func(*sttv1.UpdateSpeakerConfigRequest) (*sttv1.SpeakerConfig, error)
	getSpeakerStatus  func() *sttv1.SpeakerStatus
	listClips         func(*sttv1.ListSpeakerProfileClipsRequest) *sttv1.ListSpeakerProfileClipsResponse
	deleteClip        func(*sttv1.DeleteSpeakerProfileClipRequest) *sttv1.DeleteSpeakerProfileClipResponse
	enrollProfile     func(*sttv1.EnrollSpeakerProfileRequest) *sttv1.EnrollSpeakerProfileResponse
}

func (f *fakeSvc) EnrollSpeakerProfile(_ context.Context, req *connect.Request[sttv1.EnrollSpeakerProfileRequest]) (*connect.Response[sttv1.EnrollSpeakerProfileResponse], error) {
	return connect.NewResponse(f.enrollProfile(req.Msg)), nil
}

func (f *fakeSvc) ListSpeakerProfileClips(_ context.Context, req *connect.Request[sttv1.ListSpeakerProfileClipsRequest]) (*connect.Response[sttv1.ListSpeakerProfileClipsResponse], error) {
	return connect.NewResponse(f.listClips(req.Msg)), nil
}

func (f *fakeSvc) DeleteSpeakerProfileClip(_ context.Context, req *connect.Request[sttv1.DeleteSpeakerProfileClipRequest]) (*connect.Response[sttv1.DeleteSpeakerProfileClipResponse], error) {
	return connect.NewResponse(f.deleteClip(req.Msg)), nil
}

func (f *fakeSvc) UpdateSpeakerConfig(_ context.Context, req *connect.Request[sttv1.UpdateSpeakerConfigRequest]) (*connect.Response[sttv1.UpdateSpeakerConfigResponse], error) {
	if f.updateSpeakerCfg != nil {
		cfg, err := f.updateSpeakerCfg(req.Msg)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: cfg}), nil
	}
	return connect.NewResponse(&sttv1.UpdateSpeakerConfigResponse{Config: req.Msg.GetConfig()}), nil
}

func (f *fakeSvc) GetSpeakerStatus(_ context.Context, _ *connect.Request[sttv1.GetSpeakerStatusRequest]) (*connect.Response[sttv1.GetSpeakerStatusResponse], error) {
	return connect.NewResponse(&sttv1.GetSpeakerStatusResponse{Status: f.getSpeakerStatus()}), nil
}

func (f *fakeSvc) GetSupportedFormats(_ context.Context, _ *connect.Request[sttv1.GetSupportedFormatsRequest]) (*connect.Response[sttv1.GetSupportedFormatsResponse], error) {
	return connect.NewResponse(f.getFormats()), nil
}

func (f *fakeSvc) ListEngines(_ context.Context, _ *connect.Request[sttv1.ListEnginesRequest]) (*connect.Response[sttv1.ListEnginesResponse], error) {
	return connect.NewResponse(f.listEngines()), nil
}

func (f *fakeSvc) Transcribe(_ context.Context, req *connect.Request[sttv1.TranscribeRequest]) (*connect.Response[sttv1.TranscribeResponse], error) {
	f.transcribeModelID = req.Header().Get("X-Audio-Model-ID")
	resp, err := f.transcribe(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeSvc) TranscribeStream(ctx context.Context, stream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error {
	if f.transcribeStream != nil {
		return f.transcribeStream(ctx, stream)
	}
	return connect.NewError(connect.CodeUnimplemented, errors.New("stream fake not configured"))
}

func (f *fakeSvc) GetStreamConfig(_ context.Context, _ *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	cfg, err := f.getCfg()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sttv1.GetStreamConfigResponse{Config: cfg}), nil
}

func (f *fakeSvc) GetEngineSwitchImpact(_ context.Context, req *connect.Request[sttv1.GetEngineSwitchImpactRequest]) (*connect.Response[sttv1.GetEngineSwitchImpactResponse], error) {
	resp, err := f.engineImpact(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mountSTT(t *testing.T, svc *fakeSvc) *cliapp.ScenarioApp {
	t.Helper()
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(svc)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(runtimePath, runtimeHandler)
	mux.Handle(adminPath, adminHandler)
	return testutil.NewTestApp(t, mux)
}

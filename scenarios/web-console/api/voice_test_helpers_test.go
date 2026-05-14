package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"

	voiceH "web-console/handlers/voice"
	"web-console/internal/capabilities"
	"web-console/internal/metrics"
	intvoice "web-console/internal/voice"
)

// newVoiceOnlyServer builds a Server with just enough voice plumbing for
// tests that exercise the voice Connect handlers / Service surface. caps
// may be nil — a no-checker registry is supplied.
func newVoiceOnlyServer(caps *capabilities.Registry) *Server {
	if caps == nil {
		caps = capabilities.NewRegistry(capabilities.Known, nil, 0)
	}
	m := metrics.New()
	srv := &Server{capabilities: caps, metrics: m}
	srv.voice = intvoice.NewService(
		intvoice.DefaultConfig(), "",
		nil, "",
		intvoice.DefaultSpeakerConfig(), "",
		nil,
		caps,
		&m.VoiceSkipVerificationTotal,
		intvoice.ResolveWhisperURL(),
		nil,
	)
	return srv
}

// voiceConnectIface mirrors the methods on the unexported *connectHandler in
// handlers/voice so tests in package main can drive each RPC directly.
type voiceConnectIface interface {
	Transcribe(context.Context, *connect.Request[voicev1.TranscribeRequest]) (*connect.Response[voicev1.TranscribeResponse], error)

	GetStreamConfig(context.Context, *connect.Request[voicev1.GetStreamConfigRequest]) (*connect.Response[voicev1.GetStreamConfigResponse], error)
	UpdateStreamConfig(context.Context, *connect.Request[voicev1.UpdateStreamConfigRequest]) (*connect.Response[voicev1.UpdateStreamConfigResponse], error)

	GetWakeWordConfig(context.Context, *connect.Request[voicev1.GetWakeWordConfigRequest]) (*connect.Response[voicev1.GetWakeWordConfigResponse], error)
	UpdateWakeWordTemplate(context.Context, *connect.Request[voicev1.UpdateWakeWordTemplateRequest]) (*connect.Response[voicev1.UpdateWakeWordTemplateResponse], error)
	DeleteWakeWordTemplate(context.Context, *connect.Request[voicev1.DeleteWakeWordTemplateRequest]) (*connect.Response[voicev1.DeleteWakeWordTemplateResponse], error)

	GetSpeakerConfig(context.Context, *connect.Request[voicev1.GetSpeakerConfigRequest]) (*connect.Response[voicev1.GetSpeakerConfigResponse], error)
	UpdateSpeakerConfig(context.Context, *connect.Request[voicev1.UpdateSpeakerConfigRequest]) (*connect.Response[voicev1.UpdateSpeakerConfigResponse], error)
	GetSpeakerStatus(context.Context, *connect.Request[voicev1.GetSpeakerStatusRequest]) (*connect.Response[voicev1.GetSpeakerStatusResponse], error)
	ListSpeakerProfiles(context.Context, *connect.Request[voicev1.ListSpeakerProfilesRequest]) (*connect.Response[voicev1.ListSpeakerProfilesResponse], error)
	EnrollSpeakerProfile(context.Context, *connect.Request[voicev1.EnrollSpeakerProfileRequest]) (*connect.Response[voicev1.EnrollSpeakerProfileResponse], error)
	ClearSpeakerProfileBinding(context.Context, *connect.Request[voicev1.ClearSpeakerProfileBindingRequest]) (*connect.Response[voicev1.ClearSpeakerProfileBindingResponse], error)
	RemoveSpeakerProfile(context.Context, *connect.Request[voicev1.RemoveSpeakerProfileRequest]) (*connect.Response[voicev1.RemoveSpeakerProfileResponse], error)
	DeleteSpeakerProfile(context.Context, *connect.Request[voicev1.DeleteSpeakerProfileRequest]) (*connect.Response[voicev1.DeleteSpeakerProfileResponse], error)
}

func newVoiceConnectHandlerForServer(srv *Server) voiceConnectIface {
	return voiceH.NewConnectHandler(voiceH.Deps{Service: &voiceH.Adapter{Backend: srv.voice}})
}

func callVoiceTranscribe(t *testing.T, srv *Server, req *voicev1.TranscribeRequest) (string, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).Transcribe(context.Background(), connect.NewRequest(req))
	if err != nil {
		return "", err
	}
	return resp.Msg.GetText(), nil
}

func callGetVoiceStreamConfig(t *testing.T, srv *Server) (*voicev1.StreamConfig, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).GetStreamConfig(context.Background(),
		connect.NewRequest(&voicev1.GetStreamConfigRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callUpdateVoiceStreamConfig(t *testing.T, srv *Server, req *voicev1.UpdateStreamConfigRequest) (*voicev1.StreamConfig, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).UpdateStreamConfig(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callGetSpeakerConfig(t *testing.T, srv *Server) (*voicev1.SpeakerConfig, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).GetSpeakerConfig(context.Background(),
		connect.NewRequest(&voicev1.GetSpeakerConfigRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callUpdateSpeakerConfig(t *testing.T, srv *Server, req *voicev1.UpdateSpeakerConfigRequest) (*voicev1.SpeakerConfig, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).UpdateSpeakerConfig(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callGetSpeakerStatus(t *testing.T, srv *Server) (*voicev1.SpeakerStatus, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).GetSpeakerStatus(context.Background(),
		connect.NewRequest(&voicev1.GetSpeakerStatusRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetStatus(), nil
}

func callListSpeakerProfiles(t *testing.T, srv *Server) (*voicev1.ListSpeakerProfilesResponse, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).ListSpeakerProfiles(context.Background(),
		connect.NewRequest(&voicev1.ListSpeakerProfilesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callEnrollSpeakerProfile(t *testing.T, srv *Server, req *voicev1.EnrollSpeakerProfileRequest) (*voicev1.EnrollSpeakerProfileResponse, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).EnrollSpeakerProfile(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callClearSpeakerProfileBinding(t *testing.T, srv *Server) (*voicev1.SpeakerConfig, error) {
	t.Helper()
	resp, err := newVoiceConnectHandlerForServer(srv).ClearSpeakerProfileBinding(context.Background(),
		connect.NewRequest(&voicev1.ClearSpeakerProfileBindingRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

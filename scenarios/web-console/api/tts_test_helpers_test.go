package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"

	ttsH "web-console/handlers/tts"
)

// ttsConnectIface mirrors the methods on the unexported *connectHandler in
// handlers/tts, so tests in package main can drive each RPC directly.
type ttsConnectIface interface {
	GetConfig(context.Context, *connect.Request[ttsv1.GetConfigRequest]) (*connect.Response[ttsv1.GetConfigResponse], error)
	UpdateConfig(context.Context, *connect.Request[ttsv1.UpdateConfigRequest]) (*connect.Response[ttsv1.UpdateConfigResponse], error)
	GetStatus(context.Context, *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error)
	RecordPlaybackEvent(context.Context, *connect.Request[ttsv1.RecordPlaybackEventRequest]) (*connect.Response[ttsv1.RecordPlaybackEventResponse], error)
	GetSummarizeConfig(context.Context, *connect.Request[ttsv1.GetSummarizeConfigRequest]) (*connect.Response[ttsv1.GetSummarizeConfigResponse], error)
	UpdateSummarizeConfig(context.Context, *connect.Request[ttsv1.UpdateSummarizeConfigRequest]) (*connect.Response[ttsv1.UpdateSummarizeConfigResponse], error)
	Synthesize(context.Context, *connect.Request[ttsv1.SynthesizeRequest]) (*connect.Response[ttsv1.SynthesizeResponse], error)
	GetCache(context.Context, *connect.Request[ttsv1.GetCacheRequest]) (*connect.Response[ttsv1.GetCacheResponse], error)
	ListVoices(context.Context, *connect.Request[ttsv1.ListVoicesRequest]) (*connect.Response[ttsv1.ListVoicesResponse], error)
}

func newTTSConnectHandlerForServer(srv *Server) ttsConnectIface {
	return ttsH.NewConnectHandler(ttsH.Deps{Service: newTTSAdapter(srv)})
}

func callGetTTSConfig(t *testing.T, srv *Server) (*ttsv1.Config, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).GetConfig(context.Background(),
		connect.NewRequest(&ttsv1.GetConfigRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callUpdateTTSConfig(t *testing.T, srv *Server, req *ttsv1.UpdateConfigRequest) (*ttsv1.Config, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).UpdateConfig(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callGetTTSStatus(t *testing.T, srv *Server) (*ttsv1.Status, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).GetStatus(context.Background(),
		connect.NewRequest(&ttsv1.GetStatusRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetStatus(), nil
}

func callRecordTTSEvent(t *testing.T, srv *Server, ev *ttsv1.PlaybackEvent) error {
	t.Helper()
	_, err := newTTSConnectHandlerForServer(srv).RecordPlaybackEvent(context.Background(),
		connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{Event: ev}))
	return err
}

func callGetTTSSummarizeConfig(t *testing.T, srv *Server) (*ttsv1.SummarizeConfig, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).GetSummarizeConfig(context.Background(),
		connect.NewRequest(&ttsv1.GetSummarizeConfigRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callUpdateTTSSummarizeConfig(t *testing.T, srv *Server, req *ttsv1.UpdateSummarizeConfigRequest) (*ttsv1.SummarizeConfig, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).UpdateSummarizeConfig(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetConfig(), nil
}

func callTTSSynthesize(t *testing.T, srv *Server, req *ttsv1.SynthesizeRequest) (*ttsv1.SynthesizeResponse, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).Synthesize(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callTTSGetCache(t *testing.T, srv *Server, req *ttsv1.GetCacheRequest) (*ttsv1.GetCacheResponse, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).GetCache(context.Background(),
		connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callTTSVoices(t *testing.T, srv *Server) ([]*ttsv1.Voice, error) {
	t.Helper()
	resp, err := newTTSConnectHandlerForServer(srv).ListVoices(context.Background(),
		connect.NewRequest(&ttsv1.ListVoicesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetVoices(), nil
}

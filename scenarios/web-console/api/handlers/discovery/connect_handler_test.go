package discovery

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/discovery"
)

type fakeResolver struct {
	url string
	err error
}

func (f fakeResolver) Resolve(context.Context) (string, error) {
	return f.url, f.err
}

func TestGetAudioToolsEndpoint_Available_HTTP(t *testing.T) {
	h := NewConnectHandler(Deps{AudioTools: fakeResolver{url: "http://localhost:15000/"}})
	resp, err := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Available {
		t.Fatalf("expected available=true")
	}
	if resp.Msg.BaseUrl != "http://localhost:15000" {
		t.Fatalf("want base=http://localhost:15000, got %q", resp.Msg.BaseUrl)
	}
	if resp.Msg.WsBaseUrl != "ws://localhost:15000" {
		t.Fatalf("want ws=ws://localhost:15000, got %q", resp.Msg.WsBaseUrl)
	}
	if resp.Msg.UnavailableReason != "" {
		t.Fatalf("want empty reason, got %q", resp.Msg.UnavailableReason)
	}
}

func TestGetAudioToolsEndpoint_Available_HTTPS(t *testing.T) {
	h := NewConnectHandler(Deps{AudioTools: fakeResolver{url: "https://audio.example.com"}})
	resp, _ := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if resp.Msg.WsBaseUrl != "wss://audio.example.com" {
		t.Fatalf("want wss base, got %q", resp.Msg.WsBaseUrl)
	}
}

func TestGetAudioToolsEndpoint_Unavailable_EnvMisconfigured(t *testing.T) {
	h := NewConnectHandler(Deps{AudioTools: fakeResolver{err: errors.New("audiotools: AUDIO_TOOLS_URL not set and no default provided")}})
	resp, _ := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if resp.Msg.Available {
		t.Fatalf("expected available=false")
	}
	if resp.Msg.UnavailableReason != "env_misconfigured" {
		t.Fatalf("want env_misconfigured, got %q", resp.Msg.UnavailableReason)
	}
}

func TestGetAudioToolsEndpoint_Unavailable_ScenarioNotRunning(t *testing.T) {
	h := NewConnectHandler(Deps{AudioTools: fakeResolver{err: errors.New("scenario not running")}})
	resp, _ := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if resp.Msg.UnavailableReason != "scenario_not_running" {
		t.Fatalf("want scenario_not_running, got %q", resp.Msg.UnavailableReason)
	}
}

func TestGetAudioToolsEndpoint_Unavailable_Generic(t *testing.T) {
	h := NewConnectHandler(Deps{AudioTools: fakeResolver{err: errors.New("transport closed")}})
	resp, _ := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if resp.Msg.UnavailableReason != "discovery_failed" {
		t.Fatalf("want discovery_failed, got %q", resp.Msg.UnavailableReason)
	}
}

func TestGetAudioToolsEndpoint_NoResolver(t *testing.T) {
	h := NewConnectHandler(Deps{})
	resp, _ := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
	if resp.Msg.Available {
		t.Fatalf("expected available=false")
	}
	if resp.Msg.UnavailableReason != "resolver_not_configured" {
		t.Fatalf("want resolver_not_configured, got %q", resp.Msg.UnavailableReason)
	}
}

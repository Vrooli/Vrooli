package discovery

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/discovery"
)

type stubResolver struct {
	url string
	err error
}

func (s stubResolver) Resolve(ctx context.Context) (string, error) {
	return s.url, s.err
}

func TestGetAudioToolsEndpoint(t *testing.T) {
	cases := []struct {
		name             string
		resolver         AudioToolsResolver
		wantAvailable    bool
		wantBaseURL      string
		wantWS           string
		wantUnavailReson string
	}{
		{
			name:          "nil resolver",
			resolver:      nil,
			wantAvailable: false, wantUnavailReson: "resolver_not_configured",
		},
		{
			name:          "https url",
			resolver:      stubResolver{url: "https://audio.example.com/"},
			wantAvailable: true, wantBaseURL: "https://audio.example.com", wantWS: "wss://audio.example.com",
		},
		{
			name:          "http url",
			resolver:      stubResolver{url: "http://localhost:15000"},
			wantAvailable: true, wantBaseURL: "http://localhost:15000", wantWS: "ws://localhost:15000",
		},
		{
			name:          "env misconfigured",
			resolver:      stubResolver{err: errors.New("AUDIO_TOOLS_URL not set and no default provided")},
			wantAvailable: false, wantUnavailReson: "env_misconfigured",
		},
		{
			name:          "scenario not running",
			resolver:      stubResolver{err: errors.New("scenario 'audio-tools' is not running")},
			wantAvailable: false, wantUnavailReson: "scenario_not_running",
		},
		{
			name:          "generic failure",
			resolver:      stubResolver{err: errors.New("dial timeout")},
			wantAvailable: false, wantUnavailReson: "discovery_failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(Deps{AudioTools: tc.resolver})
			resp, err := h.GetAudioToolsEndpoint(context.Background(), connect.NewRequest(&discoveryv1.GetAudioToolsEndpointRequest{}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			msg := resp.Msg
			if msg.Available != tc.wantAvailable {
				t.Errorf("available: got %v want %v", msg.Available, tc.wantAvailable)
			}
			if msg.BaseUrl != tc.wantBaseURL {
				t.Errorf("base_url: got %q want %q", msg.BaseUrl, tc.wantBaseURL)
			}
			if msg.WsBaseUrl != tc.wantWS {
				t.Errorf("ws_base_url: got %q want %q", msg.WsBaseUrl, tc.wantWS)
			}
			if msg.UnavailableReason != tc.wantUnavailReson {
				t.Errorf("unavailable_reason: got %q want %q", msg.UnavailableReason, tc.wantUnavailReson)
			}
		})
	}
}

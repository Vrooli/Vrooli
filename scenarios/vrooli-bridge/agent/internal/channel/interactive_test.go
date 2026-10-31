package channel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"vrooli-bridge/agent/internal/config"
)

func TestIceServersCarriesOnlyBoundedTURNRoutes(t *testing.T) {
	routes := []*interactivev1.RouteCandidate{
		{Kind: interactivev1.RouteKind_ROUTE_KIND_DIRECT, Url: "stun:ignored.example"},
		{Kind: interactivev1.RouteKind_ROUTE_KIND_TURN, Url: "turn:relay.example", Username: "user", Credential: "credential"},
	}
	servers := iceServers(routes)
	if len(servers) != 1 || servers[0].GetUrl() != "turn:relay.example" || servers[0].GetCredential() != "credential" {
		t.Fatalf("servers = %+v", servers)
	}
	for i := 0; i < 16; i++ {
		routes = append(routes, &interactivev1.RouteCandidate{Kind: interactivev1.RouteKind_ROUTE_KIND_TURN, Url: "turn:" + strings.Repeat("x", i+1)})
	}
	if len(iceServers(routes)) > 8 {
		t.Fatal("route conversion exceeded the bounded server count")
	}
}

func TestLocalInteractiveAdapterForwardsOnlyTURNRoutes(t *testing.T) {
	var forwarded desktopv1.DesktopSignalRequest
	var resolvedName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.device_control.v1.desktop.DesktopSessionService/Signal" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		if err := proto.Unmarshal(readBody(t, r), &forwarded); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		response, err := proto.Marshal(&desktopv1.DesktopSignalResponse{Accepted: true, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ANSWER, Generation: "generation-1"})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(response)
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatal(err)
	}
	adapter := localDeviceControlInteractiveAdapter{httpClient: server.Client(), resolvePort: func(_ context.Context, name string) (int, error) { resolvedName = name; return port, nil }}
	response, err := adapter.HandleInteractiveSignal(context.Background(), &channelv1.InteractiveSignal{
		SessionId: "session-1", LeaseId: "lease-1", LeaseEpoch: 4, Kind: uint32(desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER), Generation: "generation-1", Payload: []byte("offer"),
		Routes: []*interactivev1.RouteCandidate{
			{Kind: interactivev1.RouteKind_ROUTE_KIND_DIRECT, Url: "stun:ignored.example"},
			{Kind: interactivev1.RouteKind_ROUTE_KIND_TURN, Url: "turn:relay.example", Username: "user", Credential: "credential"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !response.GetAccepted() || response.GetGeneration() != "generation-1" {
		t.Fatalf("response=%v", response)
	}
	if resolvedName != "device-control-companion" {
		t.Fatalf("resolved local service %q, want dedicated companion", resolvedName)
	}
	if len(forwarded.GetIceServers()) != 1 || forwarded.GetIceServers()[0].GetUrl() != "turn:relay.example" || forwarded.GetIceServers()[0].GetCredential() != "credential" {
		t.Fatalf("ice servers=%v", forwarded.GetIceServers())
	}
}

func TestResolveDesktopCompanionPortUsesDedicatedDefaultAndRejectsInvalidOverride(t *testing.T) {
	t.Setenv(desktopCompanionPortEnv, "")
	port, err := resolveDesktopCompanionPort(context.Background(), "ignored")
	if err != nil || port != defaultDesktopCompanionPort {
		t.Fatalf("default port=%d err=%v", port, err)
	}
	t.Setenv(desktopCompanionPortEnv, "17654")
	port, err = resolveDesktopCompanionPort(context.Background(), "ignored")
	if err != nil || port != 17654 {
		t.Fatalf("override port=%d err=%v", port, err)
	}
	for _, value := range []string{"0", "65536", "not-a-port"} {
		t.Setenv(desktopCompanionPortEnv, value)
		if _, err := resolveDesktopCompanionPort(context.Background(), "ignored"); err == nil {
			t.Fatalf("override %q was accepted", value)
		}
	}
}

func TestNewClientWiresInteractiveTrafficToDedicatedCompanionResolver(t *testing.T) {
	t.Setenv(desktopCompanionPortEnv, "")
	client := NewClient(config.Config{ControlPlaneURL: "http://127.0.0.1:1", NodeID: "node-1"})
	adapter, ok := client.interactiveAdapter.(localDeviceControlInteractiveAdapter)
	if !ok {
		t.Fatalf("interactive adapter = %T, want local companion adapter", client.interactiveAdapter)
	}
	port, err := adapter.resolvePort(context.Background(), "device-control-companion")
	if err != nil || port != defaultDesktopCompanionPort {
		t.Fatalf("wired resolver port=%d err=%v", port, err)
	}
}

func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestVerifyInteractiveSignalRequiresMatchingEpochAndExpiry(t *testing.T) {
	now := time.Unix(100, 0)
	grant := &interactivev1.ChannelGrant{ChannelId: "c1", LeaseEpoch: 7, Protocol: interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8, ExpiresAt: timestamppb.New(now.Add(time.Minute))}
	if err := VerifyInteractiveSignal(grant, &interactivev1.SignalRequest{Grant: grant}, now); err != nil {
		t.Fatal(err)
	}
	stale := proto.Clone(grant).(*interactivev1.ChannelGrant)
	stale.LeaseEpoch++
	if err := VerifyInteractiveSignal(grant, &interactivev1.SignalRequest{Grant: stale}, now); err != ErrInteractiveGrant {
		t.Fatalf("stale error = %v", err)
	}
	if err := VerifyInteractiveSignal(grant, &interactivev1.SignalRequest{Grant: grant}, now.Add(2*time.Minute)); err != ErrInteractiveExpired {
		t.Fatalf("expired error = %v", err)
	}
}

func TestValidInteractiveSignalFailsClosed(t *testing.T) {
	base := &channelv1.InteractiveSignal{
		ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 4, Kind: uint32(desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER),
		RequestId: "request-1", Generation: "generation-1", SessionId: "session-1", LeaseId: "lease-1", Payload: []byte(`{"type":"offer"}`),
	}
	if !validInteractiveSignal(base, "node-1") {
		t.Fatal("expected valid signal")
	}
	cases := []struct {
		name   string
		mutate func(*channelv1.InteractiveSignal)
	}{
		{"missing request id", func(s *channelv1.InteractiveSignal) { s.RequestId = "" }},
		{"missing generation", func(s *channelv1.InteractiveSignal) { s.Generation = "" }},
		{"missing session", func(s *channelv1.InteractiveSignal) { s.SessionId = "" }},
		{"missing lease", func(s *channelv1.InteractiveSignal) { s.LeaseId = "" }},
		{"unsupported kind", func(s *channelv1.InteractiveSignal) {
			s.Kind = uint32(desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ANSWER)
		}},
		{"empty payload", func(s *channelv1.InteractiveSignal) { s.Payload = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := proto.Clone(base).(*channelv1.InteractiveSignal)
			tc.mutate(candidate)
			if validInteractiveSignal(candidate, "node-1") {
				t.Fatal("expected signal to be rejected")
			}
		})
	}
}

func TestValidInteractiveRevokeFailsClosedForWrongNodeOrBinding(t *testing.T) {
	base := &channelv1.InteractiveRevoke{ChannelId: "channel-1", NodeId: "node-1", SessionId: "session-1", LeaseId: "lease-1", LeaseEpoch: 4}
	if !validInteractiveRevoke(base, "node-1") {
		t.Fatal("expected valid revoke")
	}
	wrongNode := proto.Clone(base).(*channelv1.InteractiveRevoke)
	wrongNode.NodeId = "node-2"
	if validInteractiveRevoke(wrongNode, "node-1") {
		t.Fatal("wrong-node revoke was accepted")
	}
	missingLease := proto.Clone(base).(*channelv1.InteractiveRevoke)
	missingLease.LeaseId = ""
	if validInteractiveRevoke(missingLease, "node-1") {
		t.Fatal("unbound revoke was accepted")
	}
}

package discovery

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/api-core/targetmodel"
)

type fakeTargetResolver struct {
	target targetmodel.Target
	err    error
}

func TestResolveScenarioUsesCatalogNamespaceGrammar(t *testing.T) {
	tests := []struct {
		name    string
		command string
		scopes  []string
		wantErr string
	}{
		{name: "exact", command: "scenario status", scopes: []string{"vrooli-bridge:read", "vrooli:read"}},
		{name: "namespace wildcard", command: "scenario status", scopes: []string{"vrooli-bridge:read", "vrooli:*"}},
		{name: "effect wildcard", command: "scenario status", scopes: []string{"vrooli-bridge:read", "*:read"}},
		{name: "universal", command: "scenario status", scopes: []string{"*"}},
		{name: "unrelated namespace", command: "scenario status", scopes: []string{"vrooli-bridge:read", "web-console:read"}, wantErr: "vrooli:read"},
		{name: "higher effect", command: "scenario test", scopes: []string{"vrooli-bridge:read", "*:read"}, wantErr: "vrooli:write"},
		{name: "malformed whitespace", command: "scenario status", scopes: []string{"vrooli-bridge:read", " vrooli:read"}, wantErr: "vrooli:read"},
		{name: "transport scope", command: "scenario status", scopes: []string{"vrooli:read"}, wantErr: "vrooli-bridge:read"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := bridgeTarget()
			target.Scopes = test.scopes
			resolver := NewResolver(ResolverConfig{
				TargetResolver: fakeTargetResolver{target: target}, Relay: &fakeRelay{}, CommandScope: projectCommandScope,
			})
			_, err := resolver.ResolveScenario(context.Background(), "minimouse/web-search", "API_PORT", test.command, nil)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("ResolveScenario: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want missing scope %q", err, test.wantErr)
			}
		})
	}
}

func (f fakeTargetResolver) ResolveTarget(context.Context, string) (targetmodel.Target, error) {
	return f.target, f.err
}

type fakeRelay struct {
	request RelayRequest
}

func (f *fakeRelay) Call(_ context.Context, request RelayRequest) (RelayResponse, error) {
	f.request = request
	return RelayResponse{CorrelationID: "corr-1", Outcome: "completed", Data: []byte(`{"status":"running"}`)}, nil
}

func bridgeTarget() targetmodel.Target {
	return targetmodel.Target{
		ID: "node-1", NodeID: "node-1", Label: "minimouse", Platform: "desktop", OS: "darwin",
		Architecture: "amd64", DeviceKind: "agent", Available: true,
		Transport:   targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: "bridge", Available: true},
		Scopes:      []string{"vrooli-bridge:read", "vrooli:read"},
		BridgeTrust: &targetmodel.BridgeTrust{Registered: true, Online: true, DispatchAuthorized: true},
	}
}

func projectCommandScope(command string) (string, bool) {
	if command == "scenario status" {
		return "vrooli:read", true
	}
	if command == "scenario test" {
		return "vrooli:write", true
	}
	return "", false
}

func TestResolveScenarioKeepsBareAddressOnLocalTransport(t *testing.T) {
	var argv []string
	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			argv = append([]string{name}, args...)
			return []byte("19001\n"), nil
		},
		Host: "localhost", Scheme: "http", CacheTTL: -1,
	})
	got, err := resolver.ResolveScenario(context.Background(), "agent-manager", "API_PORT", "scenario status", nil)
	if err != nil {
		t.Fatalf("ResolveScenario: %v", err)
	}
	if got.Transport != targetmodel.TransportLocal || got.URL != "http://localhost:19001" {
		t.Fatalf("local resolution = %#v", got)
	}
	want := []string{"vrooli", "scenario", "port", "agent-manager", "API_PORT"}
	if len(argv) != len(want) {
		t.Fatalf("argv = %v, want %v", argv, want)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv = %v, want %v", argv, want)
		}
	}
}

func TestResolveScenarioRoutesAddressedNodeThroughRelay(t *testing.T) {
	relay := &fakeRelay{}
	resolver := NewResolver(ResolverConfig{
		TargetResolver: fakeTargetResolver{target: bridgeTarget()}, Relay: relay, CommandScope: projectCommandScope,
	})
	got, err := resolver.ResolveScenario(context.Background(), "minimouse/web-search@shadow", "API_PORT", "scenario status", []string{"--json"})
	if err != nil {
		t.Fatalf("ResolveScenario: %v", err)
	}
	if got.Transport != targetmodel.TransportBridge || got.URL != "" || string(got.Response.Data) != `{"status":"running"}` {
		t.Fatalf("remote resolution = %#v", got)
	}
	if relay.request.NodeID != "node-1" || relay.request.Scenario != "web-search@shadow" || relay.request.Command != "scenario status" {
		t.Fatalf("relay request = %#v", relay.request)
	}
	if len(relay.request.Args) != 1 || relay.request.Args[0] != "--json" {
		t.Fatalf("relay args = %v", relay.request.Args)
	}
}

func TestResolveScenarioNamesRemoteReadinessReasons(t *testing.T) {
	tests := []struct {
		name string
		edit func(*targetmodel.Target)
		kind ErrorKind
	}{
		{name: "offline", edit: func(target *targetmodel.Target) {
			target.Available = false
			target.Transport.Available = false
		}, kind: ErrNodeOffline},
		{name: "out of scope", edit: func(target *targetmodel.Target) {
			target.Scopes = []string{"scenario test*"}
		}, kind: ErrNodeOutOfScope},
		{name: "unpaired", edit: func(target *targetmodel.Target) {
			target.BridgeTrust.Registered = false
		}, kind: ErrNodeUnpaired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := bridgeTarget()
			tc.edit(&target)
			resolver := NewResolver(ResolverConfig{TargetResolver: fakeTargetResolver{target: target}, Relay: &fakeRelay{}, CommandScope: projectCommandScope})
			_, err := resolver.ResolveScenario(context.Background(), "minimouse/web-search", "API_PORT", "scenario status", nil)
			var discoveryErr *Error
			if !errors.As(err, &discoveryErr) || discoveryErr.Kind != tc.kind {
				t.Fatalf("error = %v, want kind %q", err, tc.kind)
			}
		})
	}
}

func TestDefaultResolverWiresRemoteAdapters(t *testing.T) {
	resolver := NewResolver(ResolverConfig{})
	if resolver.targetResolver == nil || resolver.relay == nil || resolver.commandScope == nil {
		t.Fatal("default resolver must wire target, relay, and command-scope adapters")
	}
}

func TestBridgeAdaptersUsePublicCLIShapes(t *testing.T) {
	runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "nodes" {
			return []byte(`{"targets":[{"id":"node-1","label":"desk","node_id":"node-1","platform":"desktop","available":true,"transport":{"kind":"bridge","available":true},"scopes":["vrooli:read"]}]}`), nil
		}
		return []byte(`{"correlation_id":"c-1","outcome":"completed","data":"` + base64.StdEncoding.EncodeToString([]byte("ok")) + `"}`), nil
	}
	target, err := (bridgeTargetResolver{runner: runner, path: "vrooli-bridge"}).ResolveTarget(context.Background(), "desk")
	if err != nil || target.NodeID != "node-1" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
	response, err := (bridgeRelay{runner: runner, path: "vrooli-bridge"}).Call(context.Background(), RelayRequest{NodeID: "node-1", Scenario: "web-search", Command: "scenario status"})
	if err != nil || string(response.Data) != "ok" || response.CorrelationID != "c-1" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

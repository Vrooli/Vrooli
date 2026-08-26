package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sharedsession "github.com/vrooli/api-core/operatorsession"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

func TestConfiguredRemoteTargetFailsClosedWithoutAllCredentials(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "http://bridge.test")
	t.Setenv("WEB_CONSOLE_BRIDGE_NODE_ID", "node-1")
	t.Setenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN", "owner")
	t.Setenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN", "")
	target := configuredRemoteTarget()
	if target.Available {
		t.Fatal("target became available without re-authentication proof")
	}
	if target.DispatchReason == "" {
		t.Fatal("unavailable target did not explain its failure rung")
	}
}

func TestConfiguredRemoteTargetAcceptsEnrolledLocalSessionWithoutReauth(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "http://bridge.test")
	t.Setenv("WEB_CONSOLE_BRIDGE_NODE_ID", "node-1")
	t.Setenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN", "LocalSession signed-session")
	t.Setenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN", "")
	target := configuredRemoteTarget()
	if !target.Available {
		t.Fatalf("enrolled local session target unavailable: %+v", target)
	}
	if len(target.ReadinessFacts) == 0 || target.ReadinessFacts[len(target.ReadinessFacts)-1].Detail != "Bridge is configured" {
		t.Fatalf("readiness facts = %v, want capability evidence", target.ReadinessFacts)
	}
}

func TestConfiguredRemoteTargetMintsEnrolledLocalSessionByDefault(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "http://bridge.test")
	t.Setenv("WEB_CONSOLE_BRIDGE_NODE_ID", "")
	t.Setenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN", "")
	t.Setenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN", "")

	store, err := sharedsession.DefaultFileStore()
	if err != nil {
		t.Fatal(err)
	}
	private, err := sharedsession.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(private, sharedsession.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: sharedsession.IdentityProviderAuthenticator,
		Mode:             sharedsession.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Now().UTC(),
		ScopeCeiling:     []string{"vrooli-bridge:read", "vrooli-bridge:session"},
	}); err != nil {
		t.Fatal(err)
	}

	target := configuredRemoteTarget()
	if !target.Available {
		t.Fatalf("enrolled local session was not selected: %+v", target)
	}
	if !hasExplicitAuthScheme(target.OwnerToken, sharedsession.LocalSessionScheme) {
		t.Fatalf("owner credential did not use LocalSession scheme: %q", target.OwnerToken)
	}
	if target.ReauthToken != "" {
		t.Fatal("enrolled local session unexpectedly retained fallback re-authentication")
	}
}

func TestBridgeOwnerTransportPreservesExplicitAuthScheme(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "LocalSession signed-session" {
			t.Errorf("authorization = %q, want explicit LocalSession scheme", got)
		}
		if got := r.Header.Get("X-Bridge-Owner-Reauth"); got != "" {
			t.Errorf("reauth header = %q, want omitted for local session", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := (bridgeOwnerTransport{base: http.DefaultTransport, owner: "LocalSession signed-session"}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestBridgeOwnerTransportRefreshesExpiredLocalSessionFromEnrollment(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	store, err := sharedsession.DefaultFileStore()
	if err != nil {
		t.Fatal(err)
	}
	private, err := sharedsession.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	public, err := sharedsession.PublicKey(private)
	if err != nil {
		t.Fatal(err)
	}
	enrollment := sharedsession.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: sharedsession.IdentityProviderAuthenticator,
		Mode:             sharedsession.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Now().UTC(),
		ScopeCeiling:     []string{"vrooli-bridge:read"},
	}
	if err := store.Save(private, enrollment); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := r.Header.Get("Authorization")
		if !strings.HasPrefix(value, sharedsession.LocalSessionScheme+" ") {
			t.Fatalf("authorization = %q, want a refreshed local session", value)
		}
		if value == sharedsession.LocalSessionScheme+" stale-session" {
			t.Fatal("transport reused the expired local session")
		}
		if _, err := sharedsession.Verify(public, strings.TrimPrefix(value, sharedsession.LocalSessionScheme+" "), time.Now()); err != nil {
			t.Fatalf("refreshed local session failed verification: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, server.URL, nil)
	resp, err := (bridgeOwnerTransport{base: http.DefaultTransport, owner: sharedsession.LocalSessionScheme + " stale-session"}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestTargetFromRegistryNodeUsesDispatchabilityAndReadinessRung(t *testing.T) {
	base := remoteTerminalTarget{BaseURL: "http://bridge.test", OwnerToken: "owner", ReauthToken: "reauth"}
	ready := targetFromRegistryNode(base, &registryv1.Node{
		Id:                    "node-ready",
		Name:                  "Swarminator",
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
	})
	if !ready.Available || ready.ID != "bridge-node:node-ready" || len(ready.ReadinessFacts) != 5 {
		t.Fatalf("ready target was not projected correctly: %+v", ready)
	}

	offline := targetFromRegistryNode(base, &registryv1.Node{
		Id:                    "node-offline",
		Name:                  "Offline host",
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		HeartbeatFresh:        false,
		ChannelHeld:           false,
		ProtocolCompatible:    true,
	})
	if offline.Available || offline.DispatchReason != "heartbeat freshness" {
		t.Fatalf("offline target did not fail closed at first rung: %+v", offline)
	}
}

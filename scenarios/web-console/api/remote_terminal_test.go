package main

import (
	"testing"
	"time"

	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/targetmodel"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

func TestConfiguredRemoteTargetFailsClosedWithoutAllCredentials(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("VROOLI_BRIDGE_NODE_ID", "node-1")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "owner")
	t.Setenv("VROOLI_BRIDGE_REAUTH_TOKEN", "")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "")
	target := configuredRemoteTarget()
	if target.Available {
		t.Fatal("target became available without re-authentication proof")
	}
	if target.Reason == "" {
		t.Fatal("unavailable target did not explain its failure rung")
	}
}

func TestConfiguredRemoteTargetAcceptsExplicitBridgeToken(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("VROOLI_BRIDGE_NODE_ID", "node-1")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "")
	t.Setenv("VROOLI_BRIDGE_REAUTH_TOKEN", "")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "bridge-token")
	target := configuredRemoteTarget()
	if !target.Available {
		t.Fatalf("enrolled local session target unavailable: %+v", target)
	}
	if len(target.Readiness) == 0 || target.Readiness[len(target.Readiness)-1].Detail != "Bridge credentials available" {
		t.Fatalf("readiness facts = %v, want capability evidence", target.Readiness)
	}
}

func TestConfiguredRemoteTargetMintsEnrolledLocalSessionByDefault(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("VROOLI_BRIDGE_NODE_ID", "")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "")
	t.Setenv("VROOLI_BRIDGE_REAUTH_TOKEN", "")

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

func TestTargetFromRegistryNodeUsesDispatchabilityAndReadinessRung(t *testing.T) {
	base := targetConnection{Target: targetmodel.Target{}, BaseURL: "http://bridge.test", OwnerToken: "owner", ReauthToken: "reauth"}
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
	if !ready.Available || ready.ID != "bridge-node:node-ready" {
		t.Fatalf("ready target was not projected correctly: %+v", ready)
	}
	// Assert which facts are present, not how many. A count fails identically
	// for a fact correctly added and a fact wrongly dropped, and says neither.
	wantFacts := []string{
		targetmodel.ReadinessRegistry, targetmodel.ReadinessHeartbeat, targetmodel.ReadinessChannel,
		targetmodel.ReadinessProtocol, targetmodel.ReadinessDispatch, targetmodel.ReadinessBridgeScope,
	}
	got := make(map[string]bool, len(ready.Readiness))
	for _, fact := range ready.Readiness {
		got[fact.Identity] = true
	}
	for _, want := range wantFacts {
		if !got[want] {
			t.Errorf("readiness fact %q is missing from a ready target: %+v", want, ready.Readiness)
		}
		delete(got, want)
	}
	for extra := range got {
		t.Errorf("readiness fact %q was projected but is not part of the shared vocabulary", extra)
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
	if offline.Available || offline.Reason != "heartbeat freshness" {
		t.Fatalf("offline target did not fail closed at first rung: %+v", offline)
	}
}

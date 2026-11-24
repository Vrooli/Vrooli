package operatorstateapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

func TestOperatorStateRegistryReadsOnlyForVerifiedBoundPrincipal(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte(`{"version":"1.0.0","updated_at":"2026-11-20T00:00:00Z","session_mode":"hosted","enrollment_reference":"legacy-copy"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service := operatorstateapiForTest(statePath)
	registry, err := service.PermissionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	entries, err := registry.Inventory(ctx)
	if err != nil || len(entries) != 1 || entries[0].ID != permissionID || entries[0].Owner != "vrooli-onboarding" {
		t.Fatalf("inventory = %+v, err=%v", entries, err)
	}
	result, err := registry.Read(ctx, permissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: "vrooli-onboarding"})
	if err != nil || result.Revision == "" || len(result.Value) == 0 {
		t.Fatalf("read = %+v, err=%v", result, err)
	}
	var decoded operatorstate.Document
	if err := json.Unmarshal(result.Value, &decoded); err != nil {
		t.Fatalf("decode read value: %v", err)
	}
	if _, leaked := decoded.RawFields["session_mode"]; leaked {
		t.Fatal("generic operator-state read leaked enrollment-owned session mode")
	}
	if _, err := registry.Read(ctx, permissionID, permissions.ReadRequest{ActorID: "operator-b", Owner: "vrooli-onboarding"}); err == nil {
		t.Fatal("read for a different actor must be refused")
	}
	if _, err := registry.Inventory(context.Background()); err == nil {
		t.Fatal("anonymous inventory must be refused")
	}
}

func TestOperatorStatePermissionRejectsSpecializedMechanismMasks(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	service := operatorstateapiForTest(statePath)
	registry, err := service.PermissionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:write"}})
	initial, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ field, mask, request string }{
		{`"host_safeguards":{"firewall":{"opted_in":true}}`, "host_safeguards.firewall", "generic-safeguard-write"},
		{`"host_tools":{"git":{"opted_in":true}}`, "host_tools.git", "generic-host-tool-write"},
		{`"core":{"seed":["writer"],"trusted_base":[]}`, "core", "generic-core-write"},
		{`"trust_posture":"shared"`, "trust_posture", "generic-trust-posture-write"},
		{`"update_control":"own"`, "update_control", "generic-update-control-write"},
		{`"session_mode":"interactive"`, "session_mode", "generic-session-mode-write"},
	}
	for _, tc := range cases {
		value, err := json.Marshal(PermissionPatch{State: json.RawMessage("{" + tc.field + "}"), UpdateMask: []string{tc.mask}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = registry.Write(ctx, permissionID, permissions.WriteRequest{ActorID: "operator-a", Owner: "vrooli-onboarding", ExpectedRevision: operatorstate.Revision(initial), IdempotencyKey: tc.request, Value: value})
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Errorf("generic %s write error = %v, want FailedPrecondition", tc.mask, err)
		}
	}
	current, err := service.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.HostSafeguards) != 0 {
		t.Fatalf("generic safeguard write mutated state: %+v", current.HostSafeguards)
	}
	for _, field := range []string{"update_control", "session_mode"} {
		if _, exists := current.RawFields[field]; exists {
			t.Fatalf("generic %s write mutated opaque operator state", field)
		}
	}
}

func TestOperatorStateProviderReturnsDurableReceiptOnExactRetry(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	service := operatorstateapiForTest(statePath)
	registry, err := service.PermissionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:write"}})
	initial, err := registry.Read(ctx, permissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: "vrooli-onboarding"})
	if err != nil {
		t.Fatal(err)
	}
	value, err := json.Marshal(PermissionPatch{State: json.RawMessage(`{"scenarios":{"demo":{"enabled":true}}}`), UpdateMask: []string{"scenarios.demo.enabled"}})
	if err != nil {
		t.Fatal(err)
	}
	request := permissions.WriteRequest{ActorID: "operator-a", Owner: "vrooli-onboarding", ExpectedRevision: initial.Revision, IdempotencyKey: "request-1", Value: value}
	first, err := registry.Write(ctx, permissionID, request)
	if err != nil || first.ResultRevision == initial.Revision || first.RequestDigest == "" {
		t.Fatalf("first receipt=%+v err=%v", first, err)
	}
	second, err := registry.Write(ctx, permissionID, request)
	if err != nil || second != first {
		t.Fatalf("exact replay receipt=%+v err=%v first=%+v", second, err, first)
	}
	restarted := operatorstateapiForTest(statePath)
	restartedRegistry, err := restarted.PermissionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := restartedRegistry.Write(ctx, permissionID, request)
	if err != nil || replayed != first {
		t.Fatalf("restart replay receipt=%+v err=%v first=%+v", replayed, err, first)
	}
	changed := request
	changed.Value = append([]byte(nil), request.Value...)
	changed.Value = []byte(strings.Replace(string(changed.Value), `"enabled":true`, `"enabled":false`, 1))
	if _, err := restartedRegistry.Write(ctx, permissionID, changed); err == nil {
		t.Fatal("request ID reuse with changed payload must refuse")
	}
	current, err := restartedRegistry.Read(ctx, permissionID, permissions.ReadRequest{ActorID: "operator-a", Owner: "vrooli-onboarding"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(current.Value), "permission_receipts") {
		t.Fatal("bulk operator-state read leaked the private receipt ledger")
	}
	stored, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var doc operatorstate.Document
	if err := json.Unmarshal(stored, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Scenarios["demo"].Enabled == nil || !*doc.Scenarios["demo"].Enabled || len(doc.PermissionReceipts) != 1 {
		t.Fatalf("state/receipt ledger after refusal = %+v", doc)
	}
}

func operatorstateapiForTest(statePath string) Service {
	store := operatorstate.New(operatorstate.Config{RepoRoot: filepath.Dir(statePath), StatePath: func(context.Context) (string, error) { return statePath, nil }})
	return New(store, nil)
}

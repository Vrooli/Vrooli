package session

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type extensionTestCaller string

func (c extensionTestCaller) BrowserCallerID(context.Context) (string, error) { return string(c), nil }

func TestManagedExtensionRequestBindsExactSessionOwnerProfileAndCaller(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/session/session-1/extensions/install" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get(driver.SessionExecutionHeader) != "owner-1" || r.Header.Get(driver.SessionLeaseHeader) != "lease-1" {
			t.Errorf("session owner headers = %v", r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), sessionpermission.ExtensionPermissionSourceChromeGetAll) {
			t.Errorf("driver request lost permission source binding: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session_id":"session-1","extension":{"id":"ext-1"}}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	owned := &Session{
		id: "session-1", executionID: "owner-1", leaseID: "lease-1", browserBackend: "managed", mode: ModeExecution, client: client,
		permissionProfile: &sessionpermission.Profile{
			ID: "permission-ref", ProfileID: "profile-1", ProfileMode: "fresh",
			ExtensionInstall: true, ExtensionLoad: true, ExtensionAccess: true,
			ExtensionPermissions: map[string][]string{"ext-1": {"storage", "https://example.test/*"}},
		},
		callerResolver: extensionTestCaller("caller-1"),
	}

	request := []byte(`{"artifact":{"extension_id":"ext-1","artifact_ref":"build-1","artifact_sha256":"` + strings.Repeat("a", 64) + `","manifest_sha256":"` + strings.Repeat("b", 64) + `","version":"1.0.0","permissions":["storage"],"host_permissions":["https://example.test/*"],"permission_probe":"` + sessionpermission.ExtensionPermissionSourceChromeGetAll + `"}}`)
	response, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, "/extensions/install", request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if calls != 1 {
		t.Fatalf("driver calls = %d, want 1", calls)
	}

	for _, mismatch := range []struct {
		name, owner, lease, profile, permission, caller string
	}{
		{name: "owner", owner: "other", lease: "lease-1", profile: "profile-1", permission: "permission-ref", caller: "caller-1"},
		{name: "lease", owner: "owner-1", lease: "other", profile: "profile-1", permission: "permission-ref", caller: "caller-1"},
		{name: "profile", owner: "owner-1", lease: "lease-1", profile: "other", permission: "permission-ref", caller: "caller-1"},
		{name: "permission ref", owner: "owner-1", lease: "lease-1", profile: "profile-1", permission: "other", caller: "caller-1"},
		{name: "caller", owner: "owner-1", lease: "lease-1", profile: "profile-1", permission: "permission-ref", caller: "other"},
	} {
		t.Run(mismatch.name, func(t *testing.T) {
			_, err := owned.ManagedExtensionRequest(context.Background(), mismatch.owner, mismatch.lease, mismatch.profile, mismatch.permission, mismatch.caller, http.MethodPost, "/extensions/install", request)
			if err == nil {
				t.Fatal("mismatched owner binding was accepted")
			}
		})
	}
	if calls != 1 {
		t.Fatalf("mismatched bindings reached driver: calls=%d", calls)
	}
	unauthorized := []byte(`{"artifact":{"extension_id":"ext-1","artifact_ref":"build-1","artifact_sha256":"` + strings.Repeat("a", 64) + `","manifest_sha256":"` + strings.Repeat("b", 64) + `","version":"1.0.0","permissions":["debugger"]}}`)
	if _, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, "/extensions/install", unauthorized); err == nil {
		t.Fatal("extension permission outside the admitted profile was accepted")
	}
	if calls != 1 {
		t.Fatalf("permission mismatch reached driver: calls=%d", calls)
	}
	unsupportedProbe := []byte(strings.Replace(string(request), sessionpermission.ExtensionPermissionSourceChromeGetAll, "manifest-only", 1))
	if _, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, "/extensions/install", unsupportedProbe); err == nil {
		t.Fatal("unsupported runtime permission source was forwarded to driver")
	}
	if calls != 1 {
		t.Fatalf("unsupported permission source reached driver: calls=%d", calls)
	}
}

func TestManagedExtensionInventoryPOSTRequiresExactSessionProfileAndArtifact(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/session/session-1/extensions/inventory" {
			t.Errorf("route = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"profile_id":"profile-1"`) || !strings.Contains(string(body), `"artifact_sha256":"`+strings.Repeat("a", 64)+`"`) {
			t.Errorf("inventory did not carry the owner-bound profile and exact artifact: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session_id":"session-1","extensions":[]}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	owned := &Session{
		id: "session-1", executionID: "owner-1", leaseID: "lease-1", browserBackend: "managed", mode: ModeExecution, client: client,
		permissionProfile: &sessionpermission.Profile{
			ID: "permission-ref", ProfileID: "profile-1", ProfileMode: "fresh", ExtensionAccess: true,
			ExtensionPermissions: map[string][]string{"ext-1": {"storage"}},
		},
		callerResolver: extensionTestCaller("caller-1"),
	}
	body := []byte(`{"session_id":"session-1","profile_id":"profile-1","artifacts":[{"directory_path":"/owner/staged","artifact":{"extension_id":"ext-1","artifact_ref":"build-1","artifact_sha256":"` + strings.Repeat("a", 64) + `","manifest_sha256":"` + strings.Repeat("b", 64) + `","version":"1.0.0","permissions":["storage"],"host_permissions":[],"permission_probe":"chrome.permissions.getAll.v1"}}]}`)
	response, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, "/extensions/inventory", body)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if calls != 1 {
		t.Fatalf("driver calls=%d, want one", calls)
	}
	for _, mismatch := range []string{
		strings.Replace(string(body), `"session_id":"session-1"`, `"session_id":"another-session"`, 1),
		strings.Replace(string(body), `"profile_id":"profile-1"`, `"profile_id":"another-profile"`, 1),
		strings.Replace(string(body), `"permission_probe":"chrome.permissions.getAll.v1"`, `"permission_probe":"unsupported"`, 1),
	} {
		if _, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, "/extensions/inventory", []byte(mismatch)); err == nil {
			t.Fatal("mismatched inventory binding was accepted")
		}
	}
	if calls != 1 {
		t.Fatalf("invalid inventory reached driver: calls=%d", calls)
	}
}

func TestManagedExtensionRequestFailsClosedForUnsupportedProfileAndRoute(t *testing.T) {
	var driverCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		driverCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	base := &Session{
		id: "s", executionID: "o", leaseID: "l", browserBackend: "installed", mode: ModeExecution, client: client,
		permissionProfile: &sessionpermission.Profile{ID: "p", ProfileID: "profile", ProfileMode: "user_attached", ExtensionInstall: true, ExtensionPermissions: map[string][]string{"ext": {"storage"}}},
		callerResolver:    extensionTestCaller("c"),
	}
	call := func(method, suffix string) error {
		_, err := base.ManagedExtensionRequest(context.Background(), "o", "l", "profile", "p", "c", method, suffix, []byte(`{"artifact":{"extension_id":"ext","artifact_ref":"build","artifact_sha256":"`+strings.Repeat("a", 64)+`","manifest_sha256":"`+strings.Repeat("b", 64)+`","version":"1","permissions":["storage"]}}`))
		return err
	}
	if err := call(http.MethodPost, "/extensions/install"); err == nil {
		t.Fatal("user-attached session profile must be denied")
	}
	base.permissionProfile.ProfileMode = "fresh"
	for _, route := range []string{"/extensions/install", "/extensions/load", "/extensions/reload", "/extensions/unload", "/extensions/validate", "/extensions/inventory"} {
		if err := call(http.MethodPost, route); err == nil {
			t.Fatalf("installed backend route %q must be denied before driver request", route)
		}
	}
	base.browserBackend = "managed"
	if err := call(http.MethodPost, "/unlisted"); err == nil {
		t.Fatal("unlisted managed driver route must be denied")
	}
	base.permissionProfile.ExtensionInstall = false
	if err := call(http.MethodPost, "/extensions/install"); err == nil {
		t.Fatal("disabled extension installation must be denied")
	}
	if driverCalls != 0 {
		t.Fatalf("unsupported profile/backend/route reached driver %d times", driverCalls)
	}
}

func TestManagedExtensionRequestPreservesApprovalProfilesWithoutReceiptRoute(t *testing.T) {
	var driverCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		driverCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	owned := &Session{
		id: "session-1", executionID: "owner-1", leaseID: "lease-1", browserBackend: "managed", mode: ModeExecution, client: client,
		permissionProfile: &sessionpermission.Profile{
			ID: "permission-ref", ProfileID: "profile-1", ProfileMode: "fresh",
			Approval: sessionpermission.ApprovalEveryAction, ExtensionInstall: true, ExtensionLoad: true, ExtensionAccess: true,
			ExtensionPermissions: map[string][]string{"ext-1": {"storage"}},
		},
		callerResolver: extensionTestCaller("caller-1"),
	}
	body := []byte(`{"artifact":{"extension_id":"ext-1","artifact_ref":"build-1","artifact_sha256":"` + strings.Repeat("a", 64) + `","manifest_sha256":"` + strings.Repeat("b", 64) + `","version":"1.0.0","permissions":["storage"]}}`)
	request := func(suffix string) error {
		_, err := owned.ManagedExtensionRequest(context.Background(), "owner-1", "lease-1", "profile-1", "permission-ref", "caller-1", http.MethodPost, suffix, body)
		return err
	}
	for _, route := range []string{"/extensions/install", "/extensions/load", "/extensions/reload", "/extensions/unload", "/extensions/validate", "/extensions/inventory"} {
		if err := request(route); err == nil {
			t.Errorf("every-action route %s accepted without a receipt", route)
		}
	}
	if driverCalls != 0 {
		t.Fatalf("every-action profile reached driver %d times", driverCalls)
	}

	owned.permissionProfile.Approval = sessionpermission.ApprovalSafetyPass
	if err := request("/extensions/install"); err == nil {
		t.Fatal("safety-pass install accepted without a receipt")
	}
	if driverCalls != 0 {
		t.Fatalf("safety-pass install reached driver %d times", driverCalls)
	}
	if err := request("/extensions/load"); err != nil {
		t.Fatalf("safety-pass low-impact lifecycle should remain available: %v", err)
	}
	if driverCalls != 1 {
		t.Fatalf("legitimate safety-pass lifecycle driver calls=%d, want 1", driverCalls)
	}
}

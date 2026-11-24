package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type extensionClientStub struct {
	calls            []string
	receipt          BASExtensionReceipt
	validation       BASExtensionValidation
	inventory        BASExtensionInventory
	errAt            string
	cleanupErrAt     bool
	installOffset    time.Duration
	inventoryOffset  time.Duration
	replayReload     bool
	preflightStarted chan struct{}
	preflightRelease chan struct{}
}

func (s *extensionClientStub) Install(_ context.Context, sessionID, artifactRef, digest string) (BASExtensionReceipt, error) {
	s.calls = append(s.calls, "install:"+sessionID+":"+artifactRef+":"+digest)
	if s.errAt == "install" {
		return BASExtensionReceipt{}, context.DeadlineExceeded
	}
	receipt := s.receipt
	receipt.OccurredAt = time.Now().UTC().Add(s.installOffset)
	return receipt, nil
}
func (s *extensionClientStub) Preflight(_ context.Context, sessionID, artifactRef, digest, expectedExtensionID string) error {
	s.calls = append(s.calls, "preflight:"+sessionID+":"+artifactRef+":"+digest+":"+expectedExtensionID)
	if s.preflightStarted != nil {
		close(s.preflightStarted)
		<-s.preflightRelease
	}
	if s.errAt == "preflight" {
		return context.DeadlineExceeded
	}
	if expectedExtensionID != "" && expectedExtensionID != s.receipt.ExtensionID {
		return errors.New("caller extension_id does not match artifact-derived extension id")
	}
	return nil
}
func (s *extensionClientStub) Reload(_ context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	s.calls = append(s.calls, "reload:"+sessionID+":"+extensionID)
	if s.errAt == "reload" {
		return BASExtensionReceipt{}, context.DeadlineExceeded
	}
	receipt := s.receipt
	receipt.State = "reloaded"
	receipt.Operation = "reload"
	if s.replayReload {
		receipt.OccurredAt = s.receipt.OccurredAt
	} else {
		receipt.OccurredAt = time.Now().UTC()
	}
	return receipt, nil
}
func (s *extensionClientStub) Unload(ctx context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	s.calls = append(s.calls, "unload:"+sessionID+":"+extensionID)
	if s.cleanupErrAt {
		return BASExtensionReceipt{}, context.DeadlineExceeded
	}
	receipt := s.receipt
	receipt.State = "unloaded"
	receipt.Operation = "unload"
	receipt.OccurredAt = time.Now().UTC()
	if ctx.Err() != nil {
		return BASExtensionReceipt{}, ctx.Err()
	}
	if _, ok := ctx.Deadline(); !ok {
		return BASExtensionReceipt{}, errors.New("cleanup context is unbounded")
	}
	return receipt, nil
}
func (s *extensionClientStub) Validate(_ context.Context, sessionID, extensionID string) (BASExtensionValidation, error) {
	s.calls = append(s.calls, "validate:"+sessionID+":"+extensionID)
	if s.errAt == "validate" {
		return BASExtensionValidation{}, context.DeadlineExceeded
	}
	validation := s.validation
	validation.Receipt.OccurredAt = time.Now().UTC()
	return validation, nil
}
func (s *extensionClientStub) Inventory(_ context.Context, sessionID string) (BASExtensionInventory, error) {
	s.calls = append(s.calls, "inventory:"+sessionID)
	if s.errAt == "inventory" {
		return BASExtensionInventory{}, context.DeadlineExceeded
	}
	inventory := s.inventory
	inventory.ObservedAt = time.Now().UTC().Add(s.inventoryOffset)
	for i := range inventory.Extensions {
		inventory.Extensions[i].PermissionEvidence.ObservedAt = time.Now().UTC()
	}
	return inventory, nil
}

func extensionRunnerFixture() (*ExtensionBuild, BASExtensionReceipt, *extensionClientStub) {
	build := &ExtensionBuild{BuildID: "build-1", Status: ExtensionBuildReady, ArtifactSHA256: "artifact-digest", ManifestSHA256: "manifest-digest"}
	now := time.Now().UTC()
	receipt := BASExtensionReceipt{SessionID: "session-1", Backend: "managed", ProfileID: "profile-1", PermissionRef: "permission-1", OwnerID: "owner-1", LeaseID: "lease-1", CallerID: "caller-run-1", ArtifactRef: build.BuildID, Operation: "install", ExtensionID: "extension-1", Version: "1.0", Digest: build.ArtifactSHA256, ManifestSHA256: build.ManifestSHA256, State: "installed", OccurredAt: now}
	entry := BASExtensionInventoryEntry{ID: receipt.ExtensionID, Version: receipt.Version, Enabled: true, Permissions: []string{"storage"}, HostPermissions: []string{"https://example.com/*"}, PermissionEvidence: BASPermissionEvidence{Status: "effective", Source: "chrome.permissions.getAll.v1", Scope: "grant_set_only", PerTabSiteAccess: "not_measured", ObservedAt: now.Add(3 * time.Nanosecond)}, DeclaredPermissions: []string{"storage"}, DeclaredHostPermissions: []string{"https://example.com/*"}, ArtifactSHA256: receipt.Digest, ManifestSHA256: receipt.ManifestSHA256, ArtifactVerified: true}
	client := &extensionClientStub{receipt: receipt, validation: BASExtensionValidation{Receipt: BASExtensionReceipt{SessionID: receipt.SessionID, Backend: receipt.Backend, ProfileID: receipt.ProfileID, PermissionRef: receipt.PermissionRef, OwnerID: receipt.OwnerID, LeaseID: receipt.LeaseID, CallerID: receipt.CallerID, ArtifactRef: receipt.ArtifactRef, Operation: "validate", ExtensionID: receipt.ExtensionID, Version: receipt.Version, Digest: receipt.Digest, ManifestSHA256: receipt.ManifestSHA256, State: "valid", OccurredAt: now.Add(2 * time.Nanosecond)}, Valid: true, Checks: map[string]bool{"installed": true, "enabled": true, "id": true, "version": true}}, inventory: BASExtensionInventory{Binding: struct {
		SessionID     string `json:"session_id"`
		Backend       string `json:"backend"`
		ProfileID     string `json:"profile_id"`
		PermissionRef string `json:"permission_ref"`
		OwnerID       string `json:"owner_id"`
		LeaseID       string `json:"lease_id"`
		CallerID      string `json:"caller_id"`
	}{SessionID: receipt.SessionID, Backend: receipt.Backend, ProfileID: receipt.ProfileID, PermissionRef: receipt.PermissionRef, OwnerID: receipt.OwnerID, LeaseID: receipt.LeaseID, CallerID: receipt.CallerID}, ObservedAt: now.Add(4 * time.Nanosecond), Extensions: []BASExtensionInventoryEntry{entry}}}
	return build, receipt, client
}

func TestRunBASSessionExtensionTestInstallsReloadsValidatesAndConfirmsInventory(t *testing.T) {
	build, _, client := extensionRunnerFixture()
	result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	if !result.Success || result.Status != "passed" || result.Summary.Passed != 1 || result.ManifestSHA256 != build.ManifestSHA256 || result.PermissionEvidenceSource != "chrome.permissions.getAll.v1" || result.PermissionEvidenceScope != "grant_set_only" || result.PerTabSiteAccess != "not_measured" || !reflect.DeepEqual(result.EffectivePermissions, []string{"storage"}) {
		t.Fatalf("result = %#v", result)
	}
	if result.LifecycleReceipt == nil || result.LifecycleReceipt.BuildID != build.BuildID || result.LifecycleReceipt.RunBinding.CallerID != "caller-run-1" || result.LifecycleReceipt.RunBinding.PermissionRef != "permission-1" || result.LifecycleReceipt.Install.Operation != "install" || result.LifecycleReceipt.Reload.Operation != "reload" || result.LifecycleReceipt.Validation.Receipt.Operation != "validate" {
		t.Fatalf("typed lifecycle receipt is missing the exact authority/artifact chain: %#v", result.LifecycleReceipt)
	}
	want := []string{"preflight:session-1:build-1:artifact-digest:extension-1", "install:session-1:build-1:artifact-digest", "reload:session-1:extension-1", "validate:session-1:extension-1", "inventory:session-1"}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("calls = %#v, want %#v", client.calls, want)
	}
}

func TestRunBASSessionExtensionTestStopsBeforeInstallWhenLifecyclePreflightFails(t *testing.T) {
	build, _, client := extensionRunnerFixture()
	client.errAt = "preflight"
	result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	if result.Success || !strings.Contains(result.Reason, "lifecycle preflight failed before install") {
		t.Fatalf("unexpected result: %#v", result)
	}
	want := []string{"preflight:session-1:build-1:artifact-digest:extension-1"}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("calls=%#v want %#v", client.calls, want)
	}
}

func TestRunBASSessionExtensionTestRejectsConcurrentDuplicateBeforeBrokerCall(t *testing.T) {
	build, _, client := extensionRunnerFixture()
	client.preflightStarted = make(chan struct{})
	client.preflightRelease = make(chan struct{})
	firstDone := make(chan *ExtensionTestResult, 1)
	go func() {
		firstDone <- runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	}()
	<-client.preflightStarted

	second := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	if second.Success || !strings.Contains(second.Reason, "already active") {
		close(client.preflightRelease)
		t.Fatalf("concurrent duplicate was not rejected: %#v", second)
	}
	if len(client.calls) != 1 {
		close(client.preflightRelease)
		t.Fatalf("duplicate reached Broker before first run completed: %#v", client.calls)
	}
	close(client.preflightRelease)
	first := <-firstDone
	if !first.Success {
		t.Fatalf("first lifecycle did not finish after releasing preflight: %#v", first)
	}
	client.preflightStarted = nil
	client.preflightRelease = nil
	third := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	if !third.Success {
		t.Fatalf("subsequent lifecycle remained blocked after completion: %#v", third)
	}
	if len(client.calls) != 10 {
		t.Fatalf("serialized lifecycle calls = %#v", client.calls)
	}
}

func TestRunBASSessionExtensionTestRejectsExpectedExtensionIDMismatchBeforeInstall(t *testing.T) {
	build, _, client := extensionRunnerFixture()
	result := runBASSessionExtensionTest(context.Background(), build, "session-1", "different-extension", client)
	if result.Success || !strings.Contains(result.Reason, "lifecycle preflight failed before install") || result.CleanupReceipt != nil {
		t.Fatalf("expected id mismatch must fail before installation: %#v", result)
	}
	want := []string{"preflight:session-1:build-1:artifact-digest:different-extension"}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("calls = %#v, want preflight-only rejection %#v", client.calls, want)
	}
}

type cancelDuringReloadClient struct {
	*extensionClientStub
	reloadStarted chan struct{}
	unloadContext chan error
}

func (c *cancelDuringReloadClient) Reload(ctx context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	c.calls = append(c.calls, "reload:"+sessionID+":"+extensionID)
	close(c.reloadStarted)
	<-ctx.Done()
	return BASExtensionReceipt{}, ctx.Err()
}

func (c *cancelDuringReloadClient) Unload(ctx context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	c.unloadContext <- ctx.Err()
	return c.extensionClientStub.Unload(ctx, sessionID, extensionID)
}

func TestRunBASSessionExtensionTestCleansUpAfterCallerCancellation(t *testing.T) {
	build, _, base := extensionRunnerFixture()
	client := &cancelDuringReloadClient{extensionClientStub: base, reloadStarted: make(chan struct{}), unloadContext: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *ExtensionTestResult, 1)
	go func() { resultCh <- runBASSessionExtensionTest(ctx, build, "session-1", "extension-1", client) }()
	<-client.reloadStarted
	cancel()
	result := <-resultCh
	if result.Success || !strings.Contains(result.Reason, "installed extension unloaded after failure") {
		t.Fatalf("caller cancellation must fail the run and report cleanup: %#v", result)
	}
	if err := <-client.unloadContext; err != nil {
		t.Fatalf("cleanup inherited canceled context: %v", err)
	}
	if result.CleanupReceipt == nil || result.CleanupReceipt.Operation != "unload" || result.CleanupReceipt.State != "unloaded" {
		t.Fatalf("successful cleanup receipt missing after cancellation: %#v", result.CleanupReceipt)
	}
	want := []string{"preflight:session-1:build-1:artifact-digest:extension-1", "install:session-1:build-1:artifact-digest", "reload:session-1:extension-1", "unload:session-1:extension-1"}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("calls = %#v, want bounded install/reload/unload %#v", client.calls, want)
	}
}

func TestRunBASSessionExtensionTestRejectsMissingOrForgedPermissionScope(t *testing.T) {
	for _, scope := range []string{"", "per_tab_access"} {
		t.Run("scope="+scope, func(t *testing.T) {
			build, _, client := extensionRunnerFixture()
			client.inventory.Extensions[0].PermissionEvidence.Scope = scope
			result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
			if result.Success {
				t.Fatalf("S2E accepted noncanonical evidence scope %q: %#v", scope, result)
			}
		})
	}
	t.Run("per-tab claim", func(t *testing.T) {
		build, _, client := extensionRunnerFixture()
		client.inventory.Extensions[0].PermissionEvidence.PerTabSiteAccess = "effective"
		result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
		if result.Success {
			t.Fatalf("S2E accepted an unmeasured per-tab access claim: %#v", result)
		}
	})
}

func TestRunBASSessionExtensionTestFailsClosedAtEveryBoundary(t *testing.T) {
	for _, stage := range []string{"install", "reload", "validate", "inventory"} {
		t.Run(stage, func(t *testing.T) {
			build, _, client := extensionRunnerFixture()
			client.errAt = stage
			result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
			if result.Success || result.Status != "failed" {
				t.Fatalf("result = %#v", result)
			}
		})
	}
	for _, tc := range []struct {
		name      string
		mutate    func(*extensionClientStub)
		wantCalls int
	}{
		{"cross-run validation receipt", func(c *extensionClientStub) { c.validation.Receipt.CallerID = "another-run" }, 5},
		{"replayed reload receipt", func(c *extensionClientStub) { c.replayReload = true }, 4},
		{"stale install receipt", func(c *extensionClientStub) { c.installOffset = -3 * time.Minute }, 2},
		{"wrong install operation", func(c *extensionClientStub) { c.receipt.Operation = "reload" }, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, _, base := extensionRunnerFixture()
			c := *base
			tc.mutate(&c)
			result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", &c)
			if result.Success || len(c.calls) != tc.wantCalls {
				t.Fatalf("stale/cross-run receipt must stop the next effect: result=%#v calls=%#v", result, c.calls)
			}
		})
	}
	build, receipt, base := extensionRunnerFixture()
	cases := map[string]func(*extensionClientStub){
		"wrong session":               func(c *extensionClientStub) { c.inventory.Binding.SessionID = "other" },
		"cross-run inventory binding": func(c *extensionClientStub) { c.inventory.Binding.CallerID = "another-run" },
		"wrong permission binding":    func(c *extensionClientStub) { c.inventory.Binding.PermissionRef = "another-permission" },
		"stale inventory":             func(c *extensionClientStub) { c.inventoryOffset = -2 * time.Minute },
		"duplicate id": func(c *extensionClientStub) {
			c.inventory.Extensions = append(c.inventory.Extensions, c.inventory.Extensions[0])
		},
		"unverified artifact":       func(c *extensionClientStub) { c.inventory.Extensions[0].ArtifactVerified = false },
		"wrong artifact":            func(c *extensionClientStub) { c.inventory.Extensions[0].ArtifactSHA256 = "other" },
		"wrong manifest":            func(c *extensionClientStub) { c.inventory.Extensions[0].ManifestSHA256 = "other" },
		"unknown permission source": func(c *extensionClientStub) { c.inventory.Extensions[0].PermissionEvidence.Source = "unsupported" },
		"permission exceeds manifest": func(c *extensionClientStub) {
			c.inventory.Extensions[0].Permissions = append(c.inventory.Extensions[0].Permissions, "tabs")
		},
		"missing validation check": func(c *extensionClientStub) { delete(c.validation.Checks, "version") },
		"wrong install manifest":   func(c *extensionClientStub) { c.receipt.ManifestSHA256 = "other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := *base
			c.calls = nil
			mutate(&c)
			result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", &c)
			if result.Success {
				t.Fatalf("expected fail closed: %#v", result)
			}
		})
	}
	wrongID := *base
	wrongID.calls = nil
	result := runBASSessionExtensionTest(context.Background(), build, "session-1", "different-id", &wrongID)
	wantWrongIDCalls := []string{"preflight:session-1:build-1:artifact-digest:different-id"}
	if result.Success || !strings.Contains(result.Reason, "lifecycle preflight failed before install") || !reflect.DeepEqual(wrongID.calls, wantWrongIDCalls) {
		t.Fatalf("wrong id should fail before install: %#v calls=%#v want=%#v", result, wrongID.calls, wantWrongIDCalls)
	}
	_ = receipt
}

func TestRunBASSessionExtensionTestUnloadsAfterPostInstallFailures(t *testing.T) {
	for _, stage := range []string{"reload", "validate", "inventory"} {
		t.Run(stage, func(t *testing.T) {
			build, _, client := extensionRunnerFixture()
			client.errAt = stage
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			result := runBASSessionExtensionTest(ctx, build, "session-1", "extension-1", client)
			if result.Success || !strings.Contains(result.Reason, "installed extension unloaded after failure") {
				t.Fatalf("failed lifecycle did not report bounded unload: %#v", result)
			}
			wantUnload := "unload:session-1:extension-1"
			if client.calls[len(client.calls)-1] != wantUnload {
				t.Fatalf("cleanup was not same-session and same-extension: %#v", client.calls)
			}
		})
	}
	t.Run("cleanup error remains visible", func(t *testing.T) {
		build, _, client := extensionRunnerFixture()
		client.errAt = "reload"
		client.cleanupErrAt = true
		result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
		if result.Success || !strings.Contains(result.Reason, "extension cleanup failed") {
			t.Fatalf("cleanup failure was hidden: %#v", result)
		}
	})
}

func TestBASSessionExtensionHTTPClientCallsOwnedRoutesAndForwardsOnlyRunCredentials(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Header.Get("X-Agent-Identity-Token") != "signed-agent-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run-1" {
			t.Errorf("run credentials not forwarded: %v", r.Header)
		}
		if r.Header.Get("X-Forwarded-User") != "" {
			t.Errorf("untrusted identity header forwarded: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/extensions/install"):
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["extension_id"] != "" || body["artifact_ref"] != "build-1" || body["expected_digest"] != "sha256:artifact" {
				t.Errorf("install body = %#v", body)
			}
			_, _ = w.Write([]byte(`{"session_id":"session-1","extension_id":"extension-1","digest":"sha256:artifact"}`))
		case strings.HasSuffix(r.URL.Path, "/extensions/preflight"):
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["artifact_ref"] != "build-1" || body["expected_digest"] != "sha256:artifact" || body["extension_id"] != "extension-1" {
				t.Errorf("preflight body=%#v", body)
			}
			_, _ = w.Write([]byte(`{"ready":true}`))
		case strings.HasSuffix(r.URL.Path, "/reload"):
			_, _ = w.Write([]byte(`{"session_id":"session-1","extension_id":"extension-1","digest":"sha256:artifact"}`))
		case strings.HasSuffix(r.URL.Path, "/unload"):
			_, _ = w.Write([]byte(`{"session_id":"session-1","extension_id":"extension-1","digest":"sha256:artifact"}`))
		default:
			_, _ = w.Write([]byte(`{"receipt":{"session_id":"session-1","extension_id":"extension-1","digest":"sha256:artifact"},"valid":true,"checks":{"installed":true}}`))
		}
	}))
	defer server.Close()
	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", "signed-agent-token")
	headers.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run-1")
	headers.Set("X-Forwarded-User", "must-not-forward")
	client, err := newBASSessionExtensionHTTPClient(server.URL, server.Client(), headers)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Install(context.Background(), "session-1", "build-1", "sha256:artifact"); err != nil {
		t.Fatal(err)
	}
	if err := client.Preflight(context.Background(), "session-1", "build-1", "sha256:artifact", "extension-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Reload(context.Background(), "session-1", "extension-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Unload(context.Background(), "session-1", "extension-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Validate(context.Background(), "session-1", "extension-1"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /api/v1/browser/sessions/session-1/extensions/install",
		"POST /api/v1/browser/sessions/session-1/extensions/preflight",
		"POST /api/v1/browser/sessions/session-1/extensions/extension-1/reload",
		"POST /api/v1/browser/sessions/session-1/extensions/extension-1/unload",
		"GET /api/v1/browser/sessions/session-1/extensions/extension-1/validate",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}

func TestBASSessionExtensionHTTPClientRejectsTrailingAndOversizedReceiptBodies(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "trailing JSON", body: `{"session_id":"session-1"} {"state":"installed"}`, want: "one JSON value"},
		{name: "over one MiB", body: `{"receipt":"` + strings.Repeat("x", (1<<20)+1) + `"}`, want: "size limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			headers := make(http.Header)
			headers.Set("X-Agent-Identity-Token", "signed-run")
			headers.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run")
			client, err := newBASSessionExtensionHTTPClient(server.URL, server.Client(), headers)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Install(context.Background(), "session-1", "build-1", "digest"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid response err=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestRunBASSessionExtensionTestStopsAfterAmbiguousInstallReceipt(t *testing.T) {
	build, receipt, _ := extensionRunnerFixture()
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/preflight") {
			_, _ = io.WriteString(w, `{"ready":true}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/install") {
			_, _ = w.Write(append(encoded, []byte(` {"unexpected":"trailing receipt"}`)...))
			return
		}
		t.Errorf("untrusted install response allowed follow-up effect %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected lifecycle operation", http.StatusInternalServerError)
	}))
	defer server.Close()
	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", "signed-run")
	headers.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run")
	client, err := newBASSessionExtensionHTTPClient(server.URL, server.Client(), headers)
	if err != nil {
		t.Fatal(err)
	}
	result := runBASSessionExtensionTest(context.Background(), build, "session-1", "extension-1", client)
	if result.Success || !strings.Contains(result.Reason, "one JSON value") {
		t.Fatalf("ambiguous install receipt accepted: %#v", result)
	}
	want := []string{
		"POST /api/v1/browser/sessions/session-1/extensions/preflight",
		"POST /api/v1/browser/sessions/session-1/extensions/install",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("untrusted install receipt advanced lifecycle: calls=%#v want=%#v", calls, want)
	}
}

func TestBASSessionExtensionHTTPClientRequiresCallerAuthentication(t *testing.T) {
	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", "only-token")
	if _, err := newBASSessionExtensionHTTPClient("http://bas.local", nil, headers); err == nil {
		t.Fatal("expected unauthenticated internal call to be rejected")
	}
}

func TestBASSessionExtensionHTTPClientDoesNotReflectUnauthorizedBody(t *testing.T) {
	const identityToken = "identity-secret-fixture"
	const grantReceipt = "grant-secret-fixture"
	const profileRef = "profile-secret-fixture"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Identity-Token") != identityToken || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != grantReceipt {
			t.Errorf("test request did not use the injected caller credentials: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintf(w, `{"error":"denied","identity":%q,"grant":%q,"profile":%q,"project_id":"project-fixture"}`, identityToken, grantReceipt, profileRef)
	}))
	defer server.Close()

	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", identityToken)
	headers.Set("X-Vrooli-Browser-Grant-Receipt", grantReceipt)
	client, err := newBASSessionExtensionHTTPClient(server.URL, server.Client(), headers)
	if err != nil {
		t.Fatal(err)
	}
	build := &ExtensionBuild{BuildID: "build-fixture", Status: ExtensionBuildReady, ArtifactSHA256: "artifact-digest", ManifestSHA256: "manifest-digest"}
	result := runBASSessionExtensionTest(context.Background(), build, "session-fixture", "", client)
	if result.Success || !strings.Contains(result.Reason, "HTTP 403") {
		t.Fatalf("unauthorized install result=%#v", result)
	}
	for _, secret := range []string{identityToken, grantReceipt, profileRef, "project-fixture"} {
		if strings.Contains(result.Reason, secret) {
			t.Fatalf("caller-facing error reflected untrusted authority material %q: %s", secret, result.Reason)
		}
	}
}

func TestBASSessionExtensionHTTPClientDoesNotForwardRunCredentialsAcrossRedirect(t *testing.T) {
	var redirected bool
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected = true
		if r.Header.Get("X-Agent-Identity-Token") != "signed-run" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run" {
			t.Errorf("expected request not to reach redirect target with run credentials: %v", r.Header)
		}
	}))
	defer attacker.Close()
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL+"/collect", http.StatusFound)
	}))
	defer broker.Close()
	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", "signed-run")
	headers.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run")
	client, err := newBASSessionExtensionHTTPClient(broker.URL, broker.Client(), headers)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Install(context.Background(), "session-1", "build-1", "digest")
	if err == nil || redirected {
		t.Fatalf("redirect should fail without forwarding run credentials: err=%v redirected=%t", err, redirected)
	}
}

func TestBASSessionExtensionHTTPClientAcceptsExactInboundRunToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Identity-Token") != "signed-run-token" {
			t.Errorf("run token not forwarded exactly: %v", r.Header)
		}
		if r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run-2" {
			t.Errorf("grant receipt not forwarded exactly: %v", r.Header)
		}
		if r.Header.Get("X-Caller-ID") != "" {
			t.Errorf("synthesized identity header was forwarded: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session_id":"session-1","extension_id":"extension-1","digest":"sha256:artifact"}`))
	}))
	defer server.Close()
	headers := make(http.Header)
	headers.Set("X-Agent-Identity-Token", "signed-run-token")
	headers.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run-2")
	headers.Set("X-Caller-ID", "untrusted")
	client, err := newBASSessionExtensionHTTPClient(server.URL, server.Client(), headers)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Install(context.Background(), "session-1", "build-1", "sha256:artifact"); err != nil {
		t.Fatal(err)
	}
}

func TestTestExtensionHandlerUsesBuildReferenceForExistingSession(t *testing.T) {
	previous := buildManager
	buildManager = NewBuildManager()
	t.Cleanup(func() { buildManager = previous })
	build := &ExtensionBuild{BuildID: "ready-build", Status: ExtensionBuildReady, ArtifactSHA256: "artifact-digest", ManifestSHA256: "manifest-digest", ArtifactBytes: 12, CreatedAt: time.Now().UTC()}
	if err := buildManager.Add(build); err != nil {
		t.Fatal(err)
	}

	var methods []string
	receiptBase := time.Now().UTC().Add(2 * time.Second)
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Header.Get("X-Agent-Identity-Token") != "signed-run" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "browser:run-1" {
			t.Errorf("Agent Manager run credentials not forwarded: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		binding := BASExtensionReceipt{SessionID: "existing-session", Backend: "managed", ProfileID: "profile-1", PermissionRef: "permission-1", OwnerID: "owner-1", LeaseID: "lease-1", CallerID: "caller-run-1", ArtifactRef: build.BuildID, ExtensionID: "installed-extension", Version: "1.0", Digest: build.ArtifactSHA256, ManifestSHA256: build.ManifestSHA256}
		switch {
		case strings.HasSuffix(r.URL.Path, "/extensions/install"):
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request["artifact_ref"] != build.BuildID || request["expected_digest"] != build.ArtifactSHA256 {
				t.Errorf("install body = %#v", request)
			}
			binding.Operation, binding.State, binding.OccurredAt = "install", "installed", receiptBase
			_ = json.NewEncoder(w).Encode(binding)
		case strings.HasSuffix(r.URL.Path, "/reload"):
			binding.Operation, binding.State, binding.OccurredAt = "reload", "reloaded", receiptBase.Add(time.Nanosecond)
			_ = json.NewEncoder(w).Encode(binding)
		case strings.HasSuffix(r.URL.Path, "/validate"):
			binding.Operation, binding.State, binding.OccurredAt = "validate", "valid", receiptBase.Add(2*time.Nanosecond)
			_ = json.NewEncoder(w).Encode(BASExtensionValidation{Receipt: binding, Valid: true, Checks: map[string]bool{"installed": true, "enabled": true, "id": true, "version": true}})
		default:
			inventory := BASExtensionInventory{ObservedAt: receiptBase.Add(4 * time.Nanosecond)}
			inventory.Binding.SessionID, inventory.Binding.Backend, inventory.Binding.ProfileID = binding.SessionID, binding.Backend, binding.ProfileID
			inventory.Binding.PermissionRef, inventory.Binding.OwnerID, inventory.Binding.LeaseID, inventory.Binding.CallerID = binding.PermissionRef, binding.OwnerID, binding.LeaseID, binding.CallerID
			inventory.Extensions = []BASExtensionInventoryEntry{{ID: binding.ExtensionID, Version: binding.Version, Enabled: true, Permissions: []string{"storage"}, HostPermissions: []string{"https://example.com/*"}, PermissionEvidence: BASPermissionEvidence{Status: "effective", Source: "chrome.permissions.getAll.v1", Scope: "grant_set_only", PerTabSiteAccess: "not_measured", ObservedAt: receiptBase.Add(3 * time.Nanosecond)}, DeclaredPermissions: []string{"storage"}, DeclaredHostPermissions: []string{"https://example.com/*"}, ArtifactSHA256: binding.Digest, ManifestSHA256: binding.ManifestSHA256, ArtifactVerified: true}}
			_ = json.NewEncoder(w).Encode(inventory)
		}
	}))
	defer broker.Close()
	t.Setenv("BAS_AGENT_ACCESS_URL", broker.URL)
	body, _ := json.Marshal(ExtensionTestRequest{BuildID: build.BuildID, SessionID: "existing-session"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/test", bytes.NewReader(body))
	req.Header.Set("X-Agent-Identity-Token", "signed-run")
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run-1")
	recorder := httptest.NewRecorder()
	testExtensionHandler(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var result ExtensionTestResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.BuildID != build.BuildID || result.SessionID != "existing-session" || result.ExtensionID != "installed-extension" {
		t.Fatalf("result = %#v", result)
	}
	if result.LifecycleReceipt == nil || result.LifecycleReceipt.RunBinding.CallerID != "caller-run-1" || result.LifecycleReceipt.RunBinding.PermissionRef != "permission-1" || result.LifecycleReceipt.Install.ArtifactRef != build.BuildID || result.LifecycleReceipt.Install.Digest != build.ArtifactSHA256 || result.LifecycleReceipt.SessionID != "existing-session" {
		t.Fatalf("production HTTP path did not return a typed exact-run receipt: %#v", result.LifecycleReceipt)
	}
	if len(methods) != 4 {
		t.Fatalf("broker calls = %#v", methods)
	}
}

func TestTestExtensionHandlerRejectsCallerOwnedLifecycleInputs(t *testing.T) {
	for _, raw := range []string{
		`{"build_id":"build-1","session_id":"session-1","extension_path":"/tmp/ext"}`,
		`{"build_id":"build-1","session_id":"session-1","profile_id":"profile-1"}`,
		`{"build_id":"build-1","session_id":"session-1","origin":"https://example.com"}`,
		`{"build_id":"build-1","session_id":"session-1","executable_path":"/tmp/chrome"}`,
		`{"build_id":"build-1","session_id":"session-1","operation":"unload"}`,
		`{"build_id":"build-1","session_id":"session-1"} {"build_id":"other"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/extension/test", strings.NewReader(raw))
			testExtensionHandler(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("caller-owned lifecycle input accepted: status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestTestExtensionHandlerStopsOnStaleOrCrossRunReceipts(t *testing.T) {
	for _, tc := range []struct {
		name            string
		badOperation    string
		wantBASRequests int
	}{
		{"stale install", "stale-install", 1},
		{"recent replayed install", "replayed-install", 1},
		{"cross-run reload", "cross-run-reload", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous := buildManager
			buildManager = NewBuildManager()
			t.Cleanup(func() { buildManager = previous })
			build := &ExtensionBuild{BuildID: "build-1", Status: ExtensionBuildReady, ArtifactSHA256: "artifact", ManifestSHA256: "manifest"}
			if err := buildManager.Add(build); err != nil {
				t.Fatal(err)
			}
			baseTime := time.Now().UTC().Add(2 * time.Second)
			var calls int
			broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				receipt := BASExtensionReceipt{SessionID: "session-1", Backend: "managed", ProfileID: "profile-1", PermissionRef: "permission-1", OwnerID: "owner-1", LeaseID: "lease-1", CallerID: "caller-run-1", ArtifactRef: "build-1", ExtensionID: "extension-1", Version: "1.0", Digest: "artifact", ManifestSHA256: "manifest"}
				switch {
				case strings.HasSuffix(r.URL.Path, "/install"):
					receipt.Operation, receipt.State, receipt.OccurredAt = "install", "installed", baseTime
					if tc.badOperation == "stale-install" {
						receipt.OccurredAt = baseTime.Add(-3 * time.Minute)
					} else if tc.badOperation == "replayed-install" {
						receipt.OccurredAt = time.Now().UTC().Add(-time.Second)
					}
					_ = json.NewEncoder(w).Encode(receipt)
				case strings.HasSuffix(r.URL.Path, "/reload"):
					receipt.Operation, receipt.State, receipt.OccurredAt = "reload", "reloaded", baseTime.Add(time.Nanosecond)
					if tc.badOperation == "cross-run-reload" {
						receipt.CallerID = "caller-from-another-run"
					}
					_ = json.NewEncoder(w).Encode(receipt)
				case strings.HasSuffix(r.URL.Path, "/unload") && tc.badOperation == "cross-run-reload":
					receipt.Operation, receipt.State, receipt.OccurredAt = "unload", "unloaded", time.Now().UTC()
					_ = json.NewEncoder(w).Encode(receipt)
				default:
					t.Errorf("unexpected effect after invalid receipt: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected operation", http.StatusInternalServerError)
				}
			}))
			defer broker.Close()
			t.Setenv("BAS_AGENT_ACCESS_URL", broker.URL)
			body, _ := json.Marshal(ExtensionTestRequest{BuildID: build.BuildID, SessionID: "session-1"})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/extension/test", bytes.NewReader(body))
			request.Header.Set("X-Agent-Identity-Token", "signed-run")
			request.Header.Set("X-Vrooli-Browser-Grant-Receipt", "browser:run")
			recorder := httptest.NewRecorder()
			testExtensionHandler(recorder, request)
			var result ExtensionTestResult
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || result.Success || calls != tc.wantBASRequests {
				t.Fatalf("invalid receipt advanced runner: status=%d result=%#v calls=%d want=%d", recorder.Code, result, calls, tc.wantBASRequests)
			}
		})
	}
}

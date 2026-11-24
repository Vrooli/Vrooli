package agentaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const artifactInstallID = "abcdefghijklmnopabcdefghijklmnop"

var artifactInstallDigest = strings.Repeat("a", 64)
var artifactInstallManifestDigest = strings.Repeat("b", 64)

type extensionArtifactPolicy struct {
	mu       sync.Mutex
	profile  sessionpermission.Profile
	revoked  bool
	requests []sessionpermission.ActionRequest
}

func (p *extensionArtifactPolicy) ProfileSnapshot() sessionpermission.Profile { return p.profile }
func (p *extensionArtifactPolicy) Authorize(_ context.Context, request sessionpermission.ActionRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, request)
	if p.revoked {
		return sessionpermission.ErrDenied
	}
	if (request.Kind == "extension_install" || request.Kind == "extension_artifact_read") && !p.profile.ExtensionInstall {
		return sessionpermission.ErrDenied
	}
	if (request.Kind == "extension_load" || request.Kind == "extension_unload") && !p.profile.ExtensionLoad {
		return sessionpermission.ErrDenied
	}
	if !p.profile.AllowsURL(request.CurrentURL) {
		return sessionpermission.ErrDenied
	}
	if strings.HasPrefix(request.Kind, "extension_") && request.Kind != "extension_artifact_read" && request.Kind != "extension_inventory" {
		if _, ok := p.profile.ExtensionPermissions[request.ExtensionID]; !ok {
			return sessionpermission.ErrDenied
		}
	}
	return nil
}
func (p *extensionArtifactPolicy) revoke() { p.mu.Lock(); p.revoked = true; p.mu.Unlock() }

type extensionArtifactVerifierFunc func(context.Context, string, string) (ExtensionArtifact, error)

func (f extensionArtifactVerifierFunc) VerifyBrowserExtensionArtifact(ctx context.Context, ref, digest string) (ExtensionArtifact, error) {
	return f(ctx, ref, digest)
}

type extensionArtifactController struct {
	mu                    sync.Mutex
	installs              int
	installID             string
	installRef            string
	installHash           string
	lastBinding           ExtensionSessionBinding
	lastOperation         string
	lastLifecycleArtifact ExtensionArtifact
	corruptReceiptBinding string
	corruptInventoryBind  bool
	receipt               ExtensionReceipt
	validation            ExtensionValidation
	inventory             []sessionpermission.ExtensionInventoryEntry
	inventoryCalls        int
	installIn             chan struct{}
	installGo             chan struct{}
}

func (c *extensionArtifactController) Install(_ context.Context, request ExtensionInstallRequest) (ExtensionReceipt, error) {
	c.mu.Lock()
	c.installs++
	c.installID, c.installRef, c.installHash = request.Artifact.ExtensionID, request.Artifact.ArtifactRef, request.Artifact.Digest
	c.lastBinding, c.lastOperation, c.lastLifecycleArtifact = request.Binding, "install", request.Artifact
	if c.installIn != nil {
		select {
		case c.installIn <- struct{}{}:
		default:
		}
	}
	installGo := c.installGo
	receipt := c.receipt
	c.mu.Unlock()
	if installGo != nil {
		<-installGo
	}
	receipt = bindExtensionReceipt(receipt, request.Binding, request.Artifact, "install")
	c.corruptReceipt(&receipt)
	return receipt, nil
}
func (c *extensionArtifactController) Load(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.mu.Lock()
	c.lastBinding, c.lastOperation, c.lastLifecycleArtifact = request.Binding, "load", request.Artifact
	c.mu.Unlock()
	receipt := c.receipt
	receipt = bindExtensionReceipt(receipt, request.Binding, request.Artifact, "load")
	c.corruptReceipt(&receipt)
	return receipt, nil
}
func (c *extensionArtifactController) Reload(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.mu.Lock()
	c.lastBinding, c.lastOperation, c.lastLifecycleArtifact = request.Binding, "reload", request.Artifact
	c.mu.Unlock()
	receipt := bindExtensionReceipt(c.receipt, request.Binding, request.Artifact, "reload")
	c.corruptReceipt(&receipt)
	return receipt, nil
}
func (c *extensionArtifactController) Unload(_ context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	c.mu.Lock()
	c.lastBinding, c.lastOperation, c.lastLifecycleArtifact = request.Binding, "unload", request.Artifact
	c.mu.Unlock()
	receipt := bindExtensionReceipt(c.receipt, request.Binding, request.Artifact, "unload")
	c.corruptReceipt(&receipt)
	return receipt, nil
}
func (c *extensionArtifactController) Validate(_ context.Context, request ExtensionLifecycleRequest) (ExtensionValidation, error) {
	c.mu.Lock()
	c.lastBinding, c.lastOperation, c.lastLifecycleArtifact = request.Binding, "validate", request.Artifact
	c.mu.Unlock()
	validation := c.validation
	validation.Receipt = bindExtensionReceipt(validation.Receipt, request.Binding, request.Artifact, "validate")
	c.corruptReceipt(&validation.Receipt)
	return validation, nil
}
func (c *extensionArtifactController) Inventory(_ context.Context, binding ExtensionSessionBinding) (ExtensionInventoryReceipt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inventoryCalls++
	c.lastBinding, c.lastOperation = binding, "inventory"
	receipt := ExtensionInventoryReceipt{Binding: binding, ObservedAt: time.Now().UTC(), Extensions: append([]sessionpermission.ExtensionInventoryEntry(nil), c.inventory...)}
	if c.corruptInventoryBind {
		receipt.Binding.OwnerID = "other-owner"
	}
	return receipt, nil
}

func (c *extensionArtifactController) corruptReceipt(receipt *ExtensionReceipt) {
	switch c.corruptReceiptBinding {
	case "owner":
		receipt.OwnerID = "other-owner"
	case "profile":
		receipt.ProfileID = "other-profile"
	case "lease":
		receipt.LeaseID = "other-lease"
	}
}

func bindExtensionReceipt(receipt ExtensionReceipt, binding ExtensionSessionBinding, artifact ExtensionArtifact, operation string) ExtensionReceipt {
	receipt.SessionID, receipt.Backend = binding.SessionID, binding.Backend
	receipt.ProfileID, receipt.PermissionRef = binding.ProfileID, binding.PermissionRef
	receipt.OwnerID, receipt.LeaseID, receipt.CallerID = binding.OwnerID, binding.LeaseID, binding.CallerID
	receipt.ArtifactRef, receipt.Operation = artifact.ArtifactRef, operation
	receipt.ExtensionID, receipt.Version = artifact.ExtensionID, artifact.Version
	receipt.Digest, receipt.ManifestSHA256 = artifact.Digest, artifact.ManifestSHA256
	switch operation {
	case "load":
		receipt.State = "loaded"
	case "reload":
		receipt.State = "reloaded"
	case "unload":
		receipt.State = "unloaded"
	case "validate":
		receipt.State = "valid"
	default:
		receipt.State = "installed"
	}
	return receipt
}

type extensionArtifactHarness struct {
	broker          *Broker
	entry           *entry
	policy          *extensionArtifactPolicy
	verifier        *atomic.Int64
	controller      *extensionArtifactController
	callerID        string
	url             *atomic.Value
	driverInventory *managedDriverInventoryControl
}

type managedDriverInventoryOverride struct {
	SessionID       string
	ExtensionID     string
	Version         string
	Enabled         *bool
	Path            string
	Permissions     []string
	HostPermissions []string
	Evidence        *sessionpermission.ExtensionPermissionEvidence
	HoldStarted     chan struct{}
	HoldRelease     <-chan struct{}
}

type managedDriverInventoryControl struct {
	override atomic.Pointer[managedDriverInventoryOverride]
	calls    atomic.Int64
}

func newExtensionArtifactHarness(t *testing.T, artifact ExtensionArtifact, onVerify func()) *extensionArtifactHarness {
	return newExtensionArtifactHarnessWithLifecyclePermissions(t, artifact, onVerify, true, true)
}

func newExtensionArtifactHarnessWithLifecyclePermissions(t *testing.T, artifact ExtensionArtifact, onVerify func(), install, load bool) *extensionArtifactHarness {
	t.Helper()
	urlValue := &atomic.Value{}
	urlValue.Store("https://example.test/public/start")
	const sessionID = "extension-artifact-session"
	var expectedExecutionID string
	var installedExtension *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
	}
	var installedPermissionProbe string
	var installedPermissions, installedHostPermissions []string
	var installedArtifactRef string
	var installedArtifactDigest, installedManifestDigest string
	driverInventory := &managedDriverInventoryControl{}
	var extensionMu sync.Mutex
	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session/start":
			var request struct {
				ExecutionID string `json:"execution_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			expectedExecutionID = request.ExecutionID
			_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: sessionID, ActivePageID: "page-1", LeaseID: "driver-lease"})
		case r.URL.Path == "/session/"+sessionID+"/record/navigation-state":
			_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: sessionID, URL: urlValue.Load().(string), Title: "fixture"})
		case strings.HasPrefix(r.URL.Path, "/session/"+sessionID+"/extensions/"):
			if r.Header.Get(driver.SessionExecutionHeader) != expectedExecutionID || r.Header.Get(driver.SessionLeaseHeader) != "driver-lease" {
				http.Error(w, "wrong owner lease", http.StatusForbidden)
				return
			}
			extensionMu.Lock()
			defer extensionMu.Unlock()
			switch strings.TrimPrefix(r.URL.Path, "/session/"+sessionID+"/extensions/") {
			case "install", "load", "reload", "validate":
				var request struct {
					Directory string `json:"directory_path"`
					Artifact  struct {
						ID              string   `json:"extension_id"`
						ArtifactRef     string   `json:"artifact_ref"`
						Version         string   `json:"version"`
						Digest          string   `json:"artifact_sha256"`
						ManifestSHA256  string   `json:"manifest_sha256"`
						PermissionProbe string   `json:"permission_probe"`
						Permissions     []string `json:"permissions"`
						HostPermissions []string `json:"host_permissions"`
					} `json:"artifact"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Directory == "" || request.Artifact.ID == "" {
					http.Error(w, "invalid staged artifact request", http.StatusBadRequest)
					return
				}
				manifestPath := filepath.Join(request.Directory, "manifest.json")
				if _, err := os.Stat(manifestPath); err != nil {
					http.Error(w, "staged artifact is missing", http.StatusBadRequest)
					return
				}
				installedExtension = &struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Version string `json:"version"`
					Enabled bool   `json:"enabled"`
					Path    string `json:"path"`
				}{ID: request.Artifact.ID, Name: "Verified extension", Version: request.Artifact.Version, Enabled: true, Path: request.Directory}
				installedPermissionProbe = request.Artifact.PermissionProbe
				installedArtifactRef = request.Artifact.ArtifactRef
				installedPermissions = append([]string(nil), request.Artifact.Permissions...)
				installedHostPermissions = append([]string(nil), request.Artifact.HostPermissions...)
				installedArtifactDigest, installedManifestDigest = request.Artifact.Digest, request.Artifact.ManifestSHA256
			case "unload":
				installedExtension = nil
				installedPermissionProbe = ""
				installedPermissions, installedHostPermissions = nil, nil
			case "inventory":
				driverInventory.calls.Add(1)
				if r.Method == http.MethodPost {
					var request struct {
						SessionID string `json:"session_id"`
						ProfileID string `json:"profile_id"`
						Artifacts []struct {
							DirectoryPath string `json:"directory_path"`
							Artifact      struct {
								ExtensionID     string   `json:"extension_id"`
								ArtifactRef     string   `json:"artifact_ref"`
								Digest          string   `json:"artifact_sha256"`
								ManifestSHA256  string   `json:"manifest_sha256"`
								Version         string   `json:"version"`
								Permissions     []string `json:"permissions"`
								HostPermissions []string `json:"host_permissions"`
								PermissionProbe string   `json:"permission_probe"`
							} `json:"artifact"`
						} `json:"artifacts"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.SessionID != sessionID || request.ProfileID != "agent-profile-artifact" || len(request.Artifacts) == 0 {
						http.Error(w, "managed inventory request omitted exact owner/profile/artifact binding", http.StatusBadRequest)
						return
					}
					matched := false
					for _, item := range request.Artifacts {
						if installedExtension != nil && item.Artifact.ExtensionID == installedExtension.ID && item.Artifact.ArtifactRef == installedArtifactRef && item.Artifact.Digest == installedArtifactDigest && item.Artifact.ManifestSHA256 == installedManifestDigest && item.Artifact.Version == installedExtension.Version && item.Artifact.PermissionProbe == installedPermissionProbe && equalStrings(sortedUnique(item.Artifact.Permissions), sortedUnique(installedPermissions)) && equalStrings(sortedUnique(item.Artifact.HostPermissions), sortedUnique(installedHostPermissions)) && filepath.Clean(item.DirectoryPath) == filepath.Clean(installedExtension.Path) {
							matched = true
						}
					}
					if !matched {
						http.Error(w, "managed inventory artifact does not match the installed path", http.StatusBadRequest)
						return
					}
				}
				override := driverInventory.override.Load()
				if override != nil && override.HoldStarted != nil && override.HoldRelease != nil {
					select {
					case override.HoldStarted <- struct{}{}:
					default:
					}
					select {
					case <-override.HoldRelease:
					case <-r.Context().Done():
						return
					}
				}
				entries := []any{}
				if installedExtension != nil {
					extensionID, version, enabled, path := installedExtension.ID, installedExtension.Version, installedExtension.Enabled, installedExtension.Path
					permissions := append([]string(nil), installedPermissions...)
					hostPermissions := append([]string(nil), installedHostPermissions...)
					if override != nil {
						if override.ExtensionID != "" {
							extensionID = override.ExtensionID
						}
						if override.Version != "" {
							version = override.Version
						}
						if override.Enabled != nil {
							enabled = *override.Enabled
						}
						if override.Path != "" {
							path = override.Path
						}
						if override.Permissions != nil {
							permissions = append([]string(nil), override.Permissions...)
						}
						if override.HostPermissions != nil {
							hostPermissions = append([]string(nil), override.HostPermissions...)
						}
					}
					entry := map[string]any{
						"id": extensionID, "name": installedExtension.Name, "version": version,
						"enabled": enabled, "path": path,
					}
					if override != nil && override.Evidence != nil {
						entry["permission_evidence"] = override.Evidence
					} else if installedPermissionProbe == extensionPermissionProbeChromeGetAllV1 {
						entry["permission_evidence"] = permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeGetAll, time.Now().UTC())
					} else {
						entry["permission_evidence"] = permissionEvidence(sessionpermission.ExtensionPermissionEvidenceUnavailable, sessionpermission.ExtensionPermissionSourceChromiumCDP, time.Time{})
					}
					if installedPermissionProbe == extensionPermissionProbeChromeGetAllV1 {
						entry["permissions"] = permissions
						entry["host_permissions"] = hostPermissions
					}
					entries = append(entries, entry)
				}
				sessionIDReply := sessionID
				if override != nil && override.SessionID != "" {
					sessionIDReply = override.SessionID
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"session_id": sessionIDReply, "extensions": entries})
				return
			default:
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": sessionID, "extension": installedExtension})
		case r.URL.Path == "/session/"+sessionID+"/close":
			_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(driverServer.Close)
	driverClient, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	profile := sessionpermission.Profile{
		ID: "artifact-profile", ProfileID: "agent-profile-artifact", ProfileMode: "fresh", Effect: sessionpermission.EffectAct,
		Approval: sessionpermission.ApprovalNone, URLAllow: []string{"https://example.test/public/*"},
		View: true, Control: true, ExtensionAccess: true, ExtensionInstall: install, ExtensionLoad: load,
		ExtensionPermissions: map[string][]string{artifact.ExtensionID: {"storage", "tabs", "https://example.test/*"}},
	}
	policy := &extensionArtifactPolicy{profile: profile}
	caller := bridgeContextCaller{}
	ctx := withBridgeContextCaller(context.Background(), selectedTabOwner)
	authorizer := &sessionAuthorizer{resolver: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: caller, permissionRef: profile.ID, profile: profile, policy: policy, expectedCallerID: selectedTabOwner}
	manager := session.NewManagerWithClient(driverClient)
	managed, err := manager.Create(ctx, session.Spec{
		ExecutionID: uuid.New(), WorkflowID: uuid.New(), Mode: session.ModeExecution,
		BaseURL: "https://example.test/public/start", ReuseMode: "fresh", BrowserBackend: string(sessionpermission.BackendManaged), ViewportWidth: 1280, ViewportHeight: 720,
		Authorizer: authorizer, OpenAuthorizer: &bridgeOpenRecorder{}, OpenRequest: &sessionpermission.OpenRequest{Backend: sessionpermission.BackendManaged, ProfileID: profile.ID, InitialURL: "https://example.test/public/start", GrantReceipt: "grant"},
		CallerResolver: caller, PermissionProfile: &profile,
	})
	if err != nil {
		t.Fatalf("create managed session fixture: %v", err)
	}
	t.Cleanup(func() { _ = managed.Close(ctx) })
	verifierCalls := &atomic.Int64{}
	verifier := extensionArtifactVerifierFunc(func(_ context.Context, ref, digest string) (ExtensionArtifact, error) {
		verifierCalls.Add(1)
		if ref != artifact.ArtifactRef || digest != artifact.Digest {
			return ExtensionArtifact{}, errors.New("unexpected artifact reference")
		}
		if onVerify != nil {
			onVerify()
		}
		return artifact, nil
	})
	controller := &extensionArtifactController{
		receipt:    ExtensionReceipt{ExtensionID: artifactInstallID, Version: "1.0.0", Digest: artifactInstallDigest, ManifestSHA256: artifactInstallManifestDigest, State: "installed"},
		validation: ExtensionValidation{Receipt: ExtensionReceipt{ExtensionID: artifactInstallID, Version: "1.0.0", Digest: artifactInstallDigest, ManifestSHA256: artifactInstallManifestDigest, State: "valid"}, Valid: true, Checks: map[string]bool{"loaded": true, "artifact_bound": true}},
		inventory:  []sessionpermission.ExtensionInventoryEntry{{ID: artifactInstallID, Name: "Verified extension", Version: "1.0.0", Enabled: true, Permissions: []string{"storage"}, HostPermissions: []string{"https://example.test/*"}, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{})}},
	}
	ownerID, leaseID := managed.OwnerIdentity()
	e := &entry{session: managed, permissionRef: profile.ID, profile: profile, policy: policy, backend: string(sessionpermission.BackendManaged), sessionID: managed.ID(), leaseID: leaseID, ownerID: ownerID, callerID: selectedTabOwner, grantReceipt: "grant", openedAt: time.Now()}
	broker := &Broker{manager: manager, profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: caller, extensions: controller, extensionVerifier: verifier, entries: map[string]*entry{managed.ID(): e}}
	return &extensionArtifactHarness{broker: broker, entry: e, policy: policy, verifier: verifierCalls, controller: controller, callerID: selectedTabOwner, url: urlValue, driverInventory: driverInventory}
}

func extensionArtifactFixture() ExtensionArtifact {
	return ExtensionArtifact{ArtifactRef: "artifact-build", ExtensionID: artifactInstallID, Digest: artifactInstallDigest, ManifestSHA256: artifactInstallManifestDigest, Version: "1.0.0", Permissions: []string{"storage"}, HostPermissions: []string{"https://example.test/*"}}
}

func TestInstallLoadAndInventoryReceiptsStayBoundToVerifiedArtifact(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	installed, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !extensionReceiptMatchesArtifact(installed, h.entry.sessionID, extensionArtifactFixture()) || h.controller.installID != artifactInstallID || h.controller.installRef != "artifact-build" {
		t.Fatalf("install did not bind the derived artifact identity: receipt=%#v controller=%#v", installed, h.controller)
	}
	expectedBinding, err := h.broker.extensionSessionBinding(h.entry)
	if err != nil || h.controller.lastBinding != expectedBinding || !extensionReceiptMatchesBinding(installed, expectedBinding) {
		t.Fatalf("install did not bind current BAS owner/profile lease: expected=%#v got=%#v receipt=%#v err=%v", expectedBinding, h.controller.lastBinding, installed, err)
	}
	reloaded, err := h.broker.ReloadExtension(ctx, h.entry.sessionID, artifactInstallID)
	if err != nil || !extensionReceiptMatchesArtifact(reloaded, h.entry.sessionID, extensionArtifactFixture()) {
		t.Fatalf("reload receipt=%#v err=%v", reloaded, err)
	}
	unloaded, err := h.broker.UnloadExtension(ctx, h.entry.sessionID, artifactInstallID)
	if err != nil || unloaded.State != "unloaded" {
		t.Fatalf("unload receipt=%#v err=%v", unloaded, err)
	}
	loaded, err := h.broker.LoadExtension(ctx, h.entry.sessionID, artifactInstallID)
	if err != nil || loaded.State != "loaded" {
		t.Fatalf("load receipt=%#v err=%v", loaded, err)
	}
	validated, err := h.broker.ValidateExtension(ctx, h.entry.sessionID, artifactInstallID)
	if err != nil || !validated.Valid || !extensionReceiptMatchesArtifact(validated.Receipt, h.entry.sessionID, extensionArtifactFixture()) {
		t.Fatalf("validation=%#v err=%v", validated, err)
	}
	if h.controller.lastLifecycleArtifact.ArtifactRef != "artifact-build" || h.controller.lastLifecycleArtifact.Digest != artifactInstallDigest || h.controller.lastLifecycleArtifact.ManifestSHA256 != artifactInstallManifestDigest {
		t.Fatalf("load/validate owner did not receive the verified artifact binding: %#v", h.controller.lastLifecycleArtifact)
	}
	inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
	if err != nil || len(inventory.Extensions) != 1 {
		t.Fatalf("inventory=%#v err=%v", inventory, err)
	}
	item := inventory.Extensions[0]
	if !item.ArtifactVerified || item.ArtifactSHA256 != artifactInstallDigest || item.ManifestSHA256 != artifactInstallManifestDigest || len(item.DeclaredPermissions) != 1 || item.DeclaredPermissions[0] != "storage" {
		t.Fatalf("inventory did not expose the installed artifact's verified declarations: %#v", item)
	}
	if inventory.Binding != expectedBinding || inventory.SessionID != h.entry.sessionID {
		t.Fatalf("inventory receipt lost authorized owner/profile binding: got=%#v expected=%#v", inventory, expectedBinding)
	}
	if item.PermissionEvidence.Status != sessionpermission.ExtensionPermissionEvidenceEffective || item.PermissionEvidence.Source != sessionpermission.ExtensionPermissionSourceChromeManagement {
		t.Fatalf("inventory did not label effective permissions with their browser owner source: %#v", item.PermissionEvidence)
	}
}

func TestExtensionControllerReceiptsRejectWrongOwnerOrProfile(t *testing.T) {
	for _, field := range []string{"owner", "profile", "lease"} {
		t.Run(field, func(t *testing.T) {
			h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
			h.controller.corruptReceiptBinding = field
			ctx := withBridgeContextCaller(context.Background(), h.callerID)
			if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err == nil || !strings.Contains(err.Error(), "authorized session, profile, owner") {
				t.Fatalf("mismatched %s receipt accepted: %v", field, err)
			}
			if _, ok := h.entry.installedExtensions[artifactInstallID]; ok {
				t.Fatalf("mismatched %s receipt was retained by Broker", field)
			}
		})
	}
}

func TestManagedInventoryRejectsWrongOwnerReceipt(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	h.controller.corruptInventoryBind = true
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil || !strings.Contains(err.Error(), "mismatched binding") {
		t.Fatalf("mismatched managed inventory owner receipt accepted: %v", err)
	}
}

func TestExtensionLifecycleRechecksRevocationBeforeControllerCall(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err != nil {
		t.Fatal(err)
	}
	h.policy.revoke()
	if _, err := h.broker.UnloadExtension(ctx, h.entry.sessionID, artifactInstallID); err == nil {
		t.Fatal("unload proceeded after permission revocation")
	}
	if h.controller.lastOperation != "install" {
		t.Fatalf("unload reached controller after revocation: last operation %q", h.controller.lastOperation)
	}
}

func TestExtensionLifecycleRequiresIndependentInstallAndLoadOptIns(t *testing.T) {
	ctxFor := func(h *extensionArtifactHarness) context.Context {
		return withBridgeContextCaller(context.Background(), h.callerID)
	}
	t.Run("install permission absent", func(t *testing.T) {
		h := newExtensionArtifactHarnessWithLifecyclePermissions(t, extensionArtifactFixture(), nil, false, true)
		if _, err := h.broker.InstallExtension(ctxFor(h), h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err == nil {
			t.Fatal("extension install succeeded without its explicit opt-in")
		}
		if h.controller.installs != 0 {
			t.Fatalf("denied install reached controller %d times", h.controller.installs)
		}
	})
	t.Run("load permission absent", func(t *testing.T) {
		h := newExtensionArtifactHarnessWithLifecyclePermissions(t, extensionArtifactFixture(), nil, true, false)
		if err := h.broker.PreflightExtensionLifecycle(ctxFor(h), h.entry.sessionID, "artifact-build", artifactInstallDigest, ""); err == nil {
			t.Fatalf("preflight without extension-load opt-in = %v", err)
		}
		if h.controller.lastOperation != "" || h.controller.installs != 0 {
			t.Fatalf("preflight denial reached extension controller: last=%q installs=%d", h.controller.lastOperation, h.controller.installs)
		}
	})
	t.Run("all lifecycle opt-ins preflight", func(t *testing.T) {
		h := newExtensionArtifactHarnessWithLifecyclePermissions(t, extensionArtifactFixture(), nil, true, true)
		if err := h.broker.PreflightExtensionLifecycle(ctxFor(h), h.entry.sessionID, "artifact-build", artifactInstallDigest, ""); err != nil {
			t.Fatalf("preflight failed: %v", err)
		}
		if h.controller.lastOperation != "" || h.controller.installs != 0 {
			t.Fatalf("preflight caused browser effect: last=%q installs=%d", h.controller.lastOperation, h.controller.installs)
		}
	})
}

func TestExtensionLifecyclePreflightRequiresProductionControllerBeforeReady(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	h.broker.extensions = nil
	err := h.broker.PreflightExtensionLifecycle(withBridgeContextCaller(context.Background(), h.callerID), h.entry.sessionID, "artifact-build", artifactInstallDigest, "")
	if err == nil || !strings.Contains(err.Error(), "BAS session extension controller unavailable") {
		t.Fatalf("preflight without lifecycle owner = %v, want owner-unavailable denial", err)
	}
	if h.verifier.Load() != 0 || h.controller.lastOperation != "" || h.controller.installs != 0 {
		t.Fatalf("unavailable lifecycle owner should deny before artifact/controller work: verifier=%d operation=%q installs=%d", h.verifier.Load(), h.controller.lastOperation, h.controller.installs)
	}
}

func TestExtensionLifecyclePreflightRejectsWrongExpectedIDBeforeControllerEffect(t *testing.T) {
	h := newExtensionArtifactHarnessWithLifecyclePermissions(t, extensionArtifactFixture(), nil, true, true)
	err := h.broker.PreflightExtensionLifecycle(withBridgeContextCaller(context.Background(), h.callerID), h.entry.sessionID, "artifact-build", artifactInstallDigest, "different-extension")
	if err == nil || !strings.Contains(err.Error(), "does not match artifact-derived extension id") {
		t.Fatalf("preflight expected-ID mismatch = %v, want artifact identity denial", err)
	}
	if h.controller.lastOperation != "" || h.controller.installs != 0 {
		t.Fatalf("expected-ID mismatch reached controller: last=%q installs=%d", h.controller.lastOperation, h.controller.installs)
	}
}

func TestExtensionInstallRechecksAuthorityAfterLifecyclePreflight(t *testing.T) {
	h := newExtensionArtifactHarnessWithLifecyclePermissions(t, extensionArtifactFixture(), nil, true, true)
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	if err := h.broker.PreflightExtensionLifecycle(ctx, h.entry.sessionID, "artifact-build", artifactInstallDigest, ""); err != nil {
		t.Fatalf("lifecycle preflight failed: %v", err)
	}
	if h.controller.lastOperation != "" || h.controller.installs != 0 {
		t.Fatalf("preflight caused controller effect: last=%q installs=%d", h.controller.lastOperation, h.controller.installs)
	}

	// Preflight is a completeness check, not a reusable grant. A revocation
	// before the actual operation must be rechecked before the controller call.
	h.policy.revoke()
	if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err == nil {
		t.Fatal("install succeeded after authority revocation between preflight and install")
	}
	if h.controller.lastOperation != "" || h.controller.installs != 0 {
		t.Fatalf("install after revocation reached controller: last=%q installs=%d", h.controller.lastOperation, h.controller.installs)
	}
}

func TestExtensionInstallRejectsChangedSessionOwnerLeaseAndManager(t *testing.T) {
	for _, field := range []string{"owner", "lease", "manager"} {
		t.Run(field, func(t *testing.T) {
			h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
			switch field {
			case "owner":
				h.entry.ownerID = "replaced-owner"
			case "lease":
				h.entry.leaseID = "replaced-lease"
			case "manager":
				h.broker.manager = session.NewManagerWithClient(nil)
			}
			ctx := withBridgeContextCaller(context.Background(), h.callerID)
			if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err == nil {
				t.Fatalf("install accepted a changed %s binding", field)
			}
			if h.controller.installs != 0 {
				t.Fatalf("install with changed %s binding reached browser controller", field)
			}
		})
	}
}

func TestExtensionInstallRejectsWrongOwnerScopeIDAndPermissionsBeforeController(t *testing.T) {
	tests := []struct {
		name              string
		artifact          ExtensionArtifact
		caller            string
		outOfScope        bool
		wantVerifierCalls int64
	}{
		{name: "wrong owner", artifact: extensionArtifactFixture(), caller: "run:attacker", wantVerifierCalls: 0},
		{name: "out of scope before resolve", artifact: extensionArtifactFixture(), caller: selectedTabOwner, outOfScope: true, wantVerifierCalls: 0},
		{name: "unapproved manifest permission", artifact: func() ExtensionArtifact {
			a := extensionArtifactFixture()
			a.Permissions = []string{"cookies"}
			return a
		}(), caller: selectedTabOwner, wantVerifierCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newExtensionArtifactHarness(t, tc.artifact, nil)
			if tc.outOfScope {
				h.url.Store("https://outside.example/private")
			}
			_, err := h.broker.InstallExtension(withBridgeContextCaller(context.Background(), tc.caller), h.entry.sessionID, "", "artifact-build", artifactInstallDigest)
			if err == nil {
				t.Fatal("adversarial extension install was accepted")
			}
			if h.controller.installs != 0 || h.verifier.Load() != tc.wantVerifierCalls {
				t.Fatalf("side effect or authority calls: installs=%d verifier=%d err=%v", h.controller.installs, h.verifier.Load(), err)
			}
		})
	}
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	if _, err := h.broker.InstallExtension(withBridgeContextCaller(context.Background(), h.callerID), h.entry.sessionID, strings.Repeat("c", 32), "artifact-build", artifactInstallDigest); err == nil || !strings.Contains(err.Error(), "does not match artifact-derived") {
		t.Fatalf("wrong caller ID assertion = %v", err)
	}
	if h.controller.installs != 0 {
		t.Fatal("caller ID mismatch reached install controller")
	}
}

func TestExtensionInstallRechecksAuthorityAfterArtifactResolution(t *testing.T) {
	var policy *extensionArtifactPolicy
	artifact := extensionArtifactFixture()
	h := newExtensionArtifactHarness(t, artifact, func() { policy.revoke() })
	policy = h.policy
	_, err := h.broker.InstallExtension(withBridgeContextCaller(context.Background(), h.callerID), h.entry.sessionID, "", "artifact-build", artifactInstallDigest)
	if err == nil {
		t.Fatal("install proceeded after permission revocation during artifact lookup")
	}
	if h.verifier.Load() != 1 || h.controller.installs != 0 {
		t.Fatalf("revoked install side effects: verifier=%d installs=%d err=%v", h.verifier.Load(), h.controller.installs, err)
	}
}

func TestCompetingRunCannotShareArtifactInstallOperation(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	h.controller.installIn = make(chan struct{}, 1)
	h.controller.installGo = make(chan struct{})
	ownerDone := make(chan error, 1)
	go func() {
		_, err := h.broker.InstallExtension(withBridgeContextCaller(context.Background(), h.callerID), h.entry.sessionID, "", "artifact-build", artifactInstallDigest)
		ownerDone <- err
	}()
	select {
	case <-h.controller.installIn:
	case <-time.After(2 * time.Second):
		t.Fatal("owner install did not reach controller")
	}
	attackerDone := make(chan error, 1)
	go func() {
		_, err := h.broker.InstallExtension(withBridgeContextCaller(context.Background(), "run:attacker"), h.entry.sessionID, "", "artifact-build", artifactInstallDigest)
		attackerDone <- err
	}()
	close(h.controller.installGo)
	if err := <-ownerDone; err != nil {
		t.Fatalf("owner install: %v", err)
	}
	if err := <-attackerDone; err == nil {
		t.Fatal("competing run installed an extension into the owner's session")
	}
	if h.controller.installs != 1 || h.verifier.Load() != 1 {
		t.Fatalf("competing operation reached extension owner: installs=%d verifier=%d", h.controller.installs, h.verifier.Load())
	}
}

func TestManagedInventoryChecksOwnerScopeAndArtifactPermissionCeiling(t *testing.T) {
	h := newExtensionArtifactHarness(t, extensionArtifactFixture(), nil)
	ctx := withBridgeContextCaller(context.Background(), h.callerID)
	if _, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", "artifact-build", artifactInstallDigest); err != nil {
		t.Fatal(err)
	}
	if _, err := h.broker.SelectedTabExtensions(withBridgeContextCaller(context.Background(), "run:attacker"), h.entry.sessionID); err == nil {
		t.Fatal("wrong run read managed extension inventory")
	}
	if h.controller.inventoryCalls != 0 {
		t.Fatalf("wrong run reached inventory owner %d times", h.controller.inventoryCalls)
	}
	h.url.Store("https://outside.example/private")
	if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil {
		t.Fatal("out-of-scope session read extension inventory")
	}
	if h.controller.inventoryCalls != 0 {
		t.Fatalf("out-of-scope read reached inventory owner %d times", h.controller.inventoryCalls)
	}
	h.url.Store("https://example.test/public/start")
	h.controller.inventory = []sessionpermission.ExtensionInventoryEntry{{ID: artifactInstallID, Name: "Verified extension", Version: "1.0.0", Enabled: true, Permissions: []string{"tabs"}, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{})}}
	if _, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID); err == nil || !strings.Contains(err.Error(), "exceed artifact manifest") {
		t.Fatalf("live inventory exceeded artifact declaration: %v", err)
	}
	if h.controller.inventoryCalls != 1 {
		t.Fatalf("owner inventory calls = %d, want one scoped owner read", h.controller.inventoryCalls)
	}
}

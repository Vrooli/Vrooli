package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var activeBASExtensionRuns = struct {
	sync.Mutex
	keys map[string]struct{}
}{keys: make(map[string]struct{})}

func beginBASExtensionRun(sessionID, buildID string) (func(), bool) {
	key := sessionID + "\x00" + buildID
	activeBASExtensionRuns.Lock()
	defer activeBASExtensionRuns.Unlock()
	if _, busy := activeBASExtensionRuns.keys[key]; busy {
		return nil, false
	}
	activeBASExtensionRuns.keys[key] = struct{}{}
	return func() {
		activeBASExtensionRuns.Lock()
		delete(activeBASExtensionRuns.keys, key)
		activeBASExtensionRuns.Unlock()
	}, true
}

// BASSessionExtensionClient calls the canonical BAS session owner. It never
// starts a browser or accepts a local extension path; artifact references are
// resolved by scenario-to-extension's artifact owner.
type BASSessionExtensionClient interface {
	Preflight(context.Context, string, string, string, string) error
	Install(context.Context, string, string, string) (BASExtensionReceipt, error)
	Reload(context.Context, string, string) (BASExtensionReceipt, error)
	Unload(context.Context, string, string) (BASExtensionReceipt, error)
	Validate(context.Context, string, string) (BASExtensionValidation, error)
	Inventory(context.Context, string) (BASExtensionInventory, error)
}

type BASExtensionReceipt struct {
	SessionID      string    `json:"session_id"`
	Backend        string    `json:"backend"`
	ProfileID      string    `json:"profile_id"`
	PermissionRef  string    `json:"permission_ref"`
	OwnerID        string    `json:"owner_id"`
	LeaseID        string    `json:"lease_id"`
	CallerID       string    `json:"caller_id"`
	ArtifactRef    string    `json:"artifact_ref"`
	Operation      string    `json:"operation"`
	ExtensionID    string    `json:"extension_id"`
	Version        string    `json:"version"`
	Digest         string    `json:"digest"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	State          string    `json:"state"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type BASExtensionValidation struct {
	Receipt BASExtensionReceipt `json:"receipt"`
	Valid   bool                `json:"valid"`
	Checks  map[string]bool     `json:"checks"`
	Message string              `json:"message,omitempty"`
}

type BASExtensionInventory struct {
	Binding struct {
		SessionID     string `json:"session_id"`
		Backend       string `json:"backend"`
		ProfileID     string `json:"profile_id"`
		PermissionRef string `json:"permission_ref"`
		OwnerID       string `json:"owner_id"`
		LeaseID       string `json:"lease_id"`
		CallerID      string `json:"caller_id"`
	} `json:"binding"`
	ObservedAt time.Time                    `json:"observed_at"`
	Extensions []BASExtensionInventoryEntry `json:"extensions"`
}

type BASExtensionInventoryEntry struct {
	ID                      string                `json:"id"`
	Version                 string                `json:"version"`
	Enabled                 bool                  `json:"enabled"`
	Permissions             []string              `json:"effective_permissions"`
	HostPermissions         []string              `json:"effective_host_permissions"`
	PermissionEvidence      BASPermissionEvidence `json:"permission_evidence"`
	DeclaredPermissions     []string              `json:"declared_permissions"`
	DeclaredHostPermissions []string              `json:"declared_host_permissions"`
	ArtifactSHA256          string                `json:"artifact_sha256"`
	ManifestSHA256          string                `json:"manifest_sha256"`
	ArtifactVerified        bool                  `json:"artifact_verified"`
}

type BASPermissionEvidence struct {
	Status           string    `json:"status"`
	Source           string    `json:"source"`
	Scope            string    `json:"scope"`
	PerTabSiteAccess string    `json:"per_tab_site_access"`
	ObservedAt       time.Time `json:"observed_at"`
}

type basSessionExtensionHTTPClient struct {
	baseURL string
	client  *http.Client
	// Only exact inbound credential headers are forwarded. Caller identity is
	// never constructed or accepted from request JSON or other headers.
	callerHeaders http.Header
}

func newBASSessionExtensionHTTPClient(base string, client *http.Client, callerHeaders http.Header) (*basSessionExtensionHTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("BAS_AGENT_ACCESS_URL must be an absolute http(s) URL without credentials, query, or fragment")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, errors.New("BAS_AGENT_ACCESS_URL must be the BAS origin without an API path prefix")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	headers := make(http.Header)
	for _, key := range []string{"X-Agent-Identity-Token", "X-Vrooli-Browser-Grant-Receipt"} {
		for _, value := range callerHeaders.Values(key) {
			headers.Add(key, value)
		}
	}
	if len(headers.Values("X-Agent-Identity-Token")) != 1 || len(headers.Values("X-Vrooli-Browser-Grant-Receipt")) != 1 {
		return nil, errors.New("one Agent Manager identity token and browser grant receipt are required for BAS extension operations")
	}
	return &basSessionExtensionHTTPClient{baseURL: strings.TrimRight(parsed.String(), "/"), client: &clientCopy, callerHeaders: headers}, nil
}

func (c *basSessionExtensionHTTPClient) Install(ctx context.Context, sessionID, artifactRef, digest string) (BASExtensionReceipt, error) {
	var receipt BASExtensionReceipt
	err := c.request(ctx, http.MethodPost, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/install", map[string]string{"artifact_ref": artifactRef, "expected_digest": digest}, &receipt)
	return receipt, err
}

func (c *basSessionExtensionHTTPClient) Reload(ctx context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	var receipt BASExtensionReceipt
	err := c.request(ctx, http.MethodPost, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/"+url.PathEscape(extensionID)+"/reload", nil, &receipt)
	return receipt, err
}

func (c *basSessionExtensionHTTPClient) Unload(ctx context.Context, sessionID, extensionID string) (BASExtensionReceipt, error) {
	var receipt BASExtensionReceipt
	err := c.request(ctx, http.MethodPost, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/"+url.PathEscape(extensionID)+"/unload", nil, &receipt)
	return receipt, err
}

func (c *basSessionExtensionHTTPClient) Validate(ctx context.Context, sessionID, extensionID string) (BASExtensionValidation, error) {
	var result BASExtensionValidation
	err := c.request(ctx, http.MethodGet, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/"+url.PathEscape(extensionID)+"/validate", nil, &result)
	return result, err
}

func (c *basSessionExtensionHTTPClient) Inventory(ctx context.Context, sessionID string) (BASExtensionInventory, error) {
	var result BASExtensionInventory
	err := c.request(ctx, http.MethodGet, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/inventory", nil, &result)
	return result, err
}

func (c *basSessionExtensionHTTPClient) request(ctx context.Context, method, path string, body any, result any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode BAS extension request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, payload)
	if err != nil {
		return fmt.Errorf("create BAS extension request: %w", err)
	}
	req.Header = c.callerHeaders.Clone()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("call BAS extension operation: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A server error body is untrusted and may echo the injected run identity,
		// grant receipt, profile or session data. Keep only the status in the
		// caller-facing result, except for the two exact, stable Broker denial
		// messages that preserve the existing operator-facing opt-in guidance.
		return safeBASHTTPError(resp.StatusCode, resp.Body)
	}
	const maxResponseBytes = 1 << 20
	encoded, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read BAS extension response: %w", err)
	}
	if len(encoded) > maxResponseBytes {
		return errors.New("BAS extension response exceeds the size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decode BAS extension response: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("BAS extension response must contain one JSON value")
	}
	return nil
}

func safeBASHTTPError(status int, body io.Reader) error {
	const maxErrorBytes = 4096
	encoded, _ := io.ReadAll(io.LimitReader(body, maxErrorBytes))
	var response struct {
		Error string `json:"error"`
	}
	safeMessage := ""
	if json.Unmarshal(encoded, &response) == nil {
		switch response.Error {
		case "browser action denied by permission profile: extension installation is disabled":
			safeMessage = "extension installation is disabled"
		case "browser action denied by permission profile: extension loading is disabled":
			safeMessage = "extension loading is disabled"
		}
	}
	if safeMessage == "" {
		return fmt.Errorf("BAS extension operation returned HTTP %d", status)
	}
	return fmt.Errorf("BAS extension operation returned HTTP %d: %s", status, safeMessage)
}

// runBASSessionExtensionTest resolves the artifact through the local build
// owner, then installs, reloads and validates it inside the caller's existing
// BAS session. Each step fails closed and returns the precise stage.
func runBASSessionExtensionTest(ctx context.Context, build *ExtensionBuild, sessionID, expectedExtensionID string, client BASSessionExtensionClient) *ExtensionTestResult {
	if build == nil || build.Status != ExtensionBuildReady || strings.TrimSpace(build.BuildID) == "" || strings.TrimSpace(build.ArtifactSHA256) == "" || strings.TrimSpace(build.ManifestSHA256) == "" {
		return sessionExtensionResult(sessionID, "", "", "", "", nil, nil, "", time.Time{}, false, "extension build is not ready or lacks an artifact/manifest digest", time.Time{})
	}
	if strings.TrimSpace(sessionID) == "" || client == nil {
		return sessionExtensionResult(sessionID, build.BuildID, "", build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, "BAS session id or extension client is unavailable", time.Time{})
	}
	releaseRun, ok := beginBASExtensionRun(sessionID, build.BuildID)
	if !ok {
		return sessionExtensionResult(sessionID, build.BuildID, "", build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, "an extension lifecycle run is already active for this BAS session and build", time.Now())
	}
	defer releaseRun()
	start := time.Now()
	if err := client.Preflight(ctx, sessionID, build.BuildID, build.ArtifactSHA256, expectedExtensionID); err != nil {
		return sessionExtensionResult(sessionID, build.BuildID, "", build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, "lifecycle preflight failed before install: "+err.Error(), start)
	}
	installStartedAt := time.Now().UTC()
	install, err := client.Install(ctx, sessionID, build.BuildID, build.ArtifactSHA256)
	if err != nil {
		return sessionExtensionResult(sessionID, build.BuildID, "", build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, "install failed: "+err.Error(), start)
	}
	if install.SessionID != sessionID || install.ExtensionID == "" || install.Digest != build.ArtifactSHA256 || install.ManifestSHA256 != build.ManifestSHA256 || install.Version == "" || install.State != "installed" || !validBASReceiptBinding(install, sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, "install", time.Now().UTC(), installStartedAt) {
		return sessionExtensionResult(sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, "install returned a stale or mismatched run, session, artifact, operation, or lifecycle receipt", start)
	}
	failAndUnload := func(reason string) *ExtensionTestResult {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		cleanupStartedAt := time.Now().UTC()
		unload, cleanupErr := client.Unload(cleanupCtx, sessionID, install.ExtensionID)
		var acceptedCleanup *BASExtensionReceipt
		if cleanupErr != nil {
			reason += "; extension cleanup failed: " + cleanupErr.Error()
		} else if unload.SessionID != sessionID || unload.ExtensionID != install.ExtensionID || unload.Digest != install.Digest || unload.ManifestSHA256 != install.ManifestSHA256 || unload.Version != install.Version || unload.State != "unloaded" || !sameBASReceiptBinding(install, unload) || !validBASReceiptBinding(unload, sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, "unload", time.Now().UTC(), cleanupStartedAt) {
			reason += "; extension cleanup returned a stale or mismatched lifecycle receipt"
		} else {
			reason += "; installed extension unloaded after failure"
			acceptedCleanup = &unload
		}
		result := sessionExtensionResult(sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, nil, nil, "", time.Time{}, false, reason, start)
		result.CleanupReceipt = acceptedCleanup
		return result
	}
	// At this point the install receipt is fresh and binds the exact current
	// session, caller, lease, build, artifact, and extension. A caller-supplied
	// expected ID mismatch is therefore a trusted post-install failure and the
	// staged extension must be unloaded before returning.
	if expectedExtensionID != "" && install.ExtensionID != expectedExtensionID {
		return failAndUnload("installed extension id did not match the expected extension id")
	}
	reloadStartedAt := time.Now().UTC()
	reload, err := client.Reload(ctx, sessionID, install.ExtensionID)
	if err != nil {
		return failAndUnload("reload failed: " + err.Error())
	}
	if reload.SessionID != sessionID || reload.ExtensionID != install.ExtensionID || reload.Digest != install.Digest || reload.ManifestSHA256 != install.ManifestSHA256 || reload.Version != install.Version || reload.State != "reloaded" || !sameBASReceiptBinding(install, reload) || !validBASReceiptBinding(reload, sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, "reload", time.Now().UTC(), reloadStartedAt) {
		return failAndUnload("reload returned a stale, replayed, or mismatched run, session, artifact, operation, or lifecycle receipt")
	}
	validateStartedAt := time.Now().UTC()
	validation, err := client.Validate(ctx, sessionID, install.ExtensionID)
	if err != nil {
		return failAndUnload("validation failed: " + err.Error())
	}
	if validation.Receipt.SessionID != sessionID || validation.Receipt.ExtensionID != install.ExtensionID || validation.Receipt.Digest != install.Digest || validation.Receipt.ManifestSHA256 != install.ManifestSHA256 || validation.Receipt.Version != install.Version || validation.Receipt.State != "valid" || !sameBASReceiptBinding(install, validation.Receipt) || !validBASReceiptBinding(validation.Receipt, sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, "validate", time.Now().UTC(), validateStartedAt) {
		return failAndUnload("validation returned a stale, replayed, or mismatched run, session, artifact, operation, or lifecycle receipt")
	}
	requiredChecks := []string{"installed", "enabled", "id", "version"}
	loaded := validation.Valid && len(validation.Checks) >= len(requiredChecks)
	for _, check := range requiredChecks {
		loaded = loaded && validation.Checks[check]
	}
	if !loaded {
		reason := validation.Message
		if reason == "" {
			reason = "BAS extension validation did not provide an all-passing check receipt"
		}
		return failAndUnload(reason)
	}
	inventoryStartedAt := time.Now().UTC()
	inventory, err := client.Inventory(ctx, sessionID)
	if err != nil {
		return failAndUnload("permission inventory failed: " + err.Error())
	}
	entry, ok := exactBASPermissionEvidence(inventory, sessionID, install, validation.Receipt, inventoryStartedAt, time.Now().UTC())
	if !ok {
		return failAndUnload("BAS did not return fresh same-session effective permission evidence for the exact staged artifact")
	}
	result := sessionExtensionResult(sessionID, build.BuildID, install.ExtensionID, build.ArtifactSHA256, build.ManifestSHA256, entry.Permissions, entry.HostPermissions, entry.PermissionEvidence.Source, entry.PermissionEvidence.ObservedAt, true, validation.Message, start)
	result.PermissionEvidenceScope = entry.PermissionEvidence.Scope
	result.PerTabSiteAccess = entry.PermissionEvidence.PerTabSiteAccess
	result.LifecycleReceipt = &ExtensionLifecycleRunReceipt{
		BuildID: build.BuildID, SessionID: sessionID, ArtifactSHA256: build.ArtifactSHA256, ManifestSHA256: build.ManifestSHA256,
		RunBinding: BASRunBindingReceipt{Backend: install.Backend, ProfileID: install.ProfileID, PermissionRef: install.PermissionRef, OwnerID: install.OwnerID, LeaseID: install.LeaseID, CallerID: install.CallerID},
		Install:    install, Reload: reload, Validation: validation, InventoryObservedAt: inventory.ObservedAt,
	}
	return result
}

func (c *basSessionExtensionHTTPClient) Preflight(ctx context.Context, sessionID, artifactRef, digest, expectedExtensionID string) error {
	body := map[string]string{"artifact_ref": artifactRef, "expected_digest": digest}
	if strings.TrimSpace(expectedExtensionID) != "" {
		body["extension_id"] = expectedExtensionID
	}
	var response struct {
		Ready bool `json:"ready"`
	}
	if err := c.request(ctx, http.MethodPost, "/browser/sessions/"+url.PathEscape(sessionID)+"/extensions/preflight", body, &response); err != nil {
		return err
	}
	if !response.Ready {
		return errors.New("BAS extension lifecycle preflight was not ready")
	}
	return nil
}

func exactBASPermissionEvidence(inventory BASExtensionInventory, sessionID string, installed, validated BASExtensionReceipt, inventoryStartedAt, now time.Time) (BASExtensionInventoryEntry, bool) {
	if inventory.Binding.SessionID != sessionID || inventory.Binding.Backend != installed.Backend || inventory.Binding.ProfileID != installed.ProfileID || inventory.Binding.PermissionRef != installed.PermissionRef || inventory.Binding.OwnerID != installed.OwnerID || inventory.Binding.LeaseID != installed.LeaseID || inventory.Binding.CallerID != installed.CallerID || inventory.ObservedAt.IsZero() || !freshBASReceiptTime(inventory.ObservedAt, now, inventoryStartedAt) || len(inventory.Extensions) > 1024 {
		return BASExtensionInventoryEntry{}, false
	}
	var matches []BASExtensionInventoryEntry
	for _, entry := range inventory.Extensions {
		if entry.ID == installed.ExtensionID {
			matches = append(matches, entry)
		}
	}
	if len(matches) != 1 {
		return BASExtensionInventoryEntry{}, false
	}
	entry := matches[0]
	evidenceAt := entry.PermissionEvidence.ObservedAt
	if !entry.Enabled || entry.Version != installed.Version || !entry.ArtifactVerified || entry.ArtifactSHA256 != installed.Digest || entry.ManifestSHA256 != installed.ManifestSHA256 ||
		entry.PermissionEvidence.Status != "effective" || entry.PermissionEvidence.Source != "chrome.permissions.getAll.v1" ||
		entry.PermissionEvidence.Scope != "grant_set_only" || entry.PermissionEvidence.PerTabSiteAccess != "not_measured" ||
		!freshBASReceiptTime(evidenceAt, now, validated.OccurredAt) ||
		!stringSetContainsAll(entry.DeclaredPermissions, entry.Permissions) || !stringSetContainsAll(entry.DeclaredHostPermissions, entry.HostPermissions) {
		return BASExtensionInventoryEntry{}, false
	}
	return entry, true
}

const maxBASReceiptAge = 2 * time.Minute

func freshBASReceiptTime(occurredAt, now, notBefore time.Time) bool {
	return !occurredAt.IsZero() && !occurredAt.After(now.Add(5*time.Second)) && now.Sub(occurredAt) <= maxBASReceiptAge && (notBefore.IsZero() || occurredAt.After(notBefore))
}

func validBASReceiptBinding(receipt BASExtensionReceipt, sessionID, artifactRef, extensionID, digest, manifestDigest, operation string, now, after time.Time) bool {
	return receipt.SessionID == sessionID && receipt.Backend == "managed" && receipt.ProfileID != "" && receipt.PermissionRef != "" && receipt.OwnerID != "" && receipt.LeaseID != "" && receipt.CallerID != "" && receipt.ArtifactRef == artifactRef && receipt.ExtensionID == extensionID && receipt.Digest == digest && receipt.ManifestSHA256 == manifestDigest && receipt.Operation == operation && freshBASReceiptTime(receipt.OccurredAt, now, after)
}

func sameBASReceiptBinding(first, next BASExtensionReceipt) bool {
	return next.SessionID == first.SessionID && next.Backend == first.Backend && next.ProfileID == first.ProfileID && next.PermissionRef == first.PermissionRef && next.OwnerID == first.OwnerID && next.LeaseID == first.LeaseID && next.CallerID == first.CallerID && next.ArtifactRef == first.ArtifactRef && next.ExtensionID == first.ExtensionID && next.Digest == first.Digest && next.ManifestSHA256 == first.ManifestSHA256 && next.Version == first.Version
}

func stringSetContainsAll(allowed, values []string) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		set[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range values {
		if _, ok := set[strings.TrimSpace(value)]; !ok || strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func sessionExtensionResult(sessionID, buildID, extensionID, digest, manifestSHA256 string, permissions, hostPermissions []string, permissionEvidenceSource string, permissionObservedAt time.Time, passed bool, reason string, started time.Time) *ExtensionTestResult {
	status := "failed"
	rowErrors := []string(nil)
	if passed {
		status = "passed"
	} else if reason != "" {
		rowErrors = []string{reason}
	}
	testResult := ExtensionSiteResult{Site: "BAS session " + sessionID, Loaded: passed, Errors: rowErrors}
	if started.IsZero() {
		testResult.LoadTime = 0
	} else {
		testResult.LoadTime = int(time.Since(started).Milliseconds())
	}
	summary := ExtensionTestSummary{TotalTests: 1, Failed: 1}
	if passed {
		summary.Passed, summary.Failed, summary.SuccessRate = 1, 0, 1
	}
	return &ExtensionTestResult{
		Success: passed, Status: status, Reason: reason, BuildID: buildID,
		SessionID: sessionID, ExtensionID: extensionID, ArtifactSHA256: digest, ManifestSHA256: manifestSHA256,
		EffectivePermissions: append([]string(nil), permissions...), EffectiveHostPermissions: append([]string(nil), hostPermissions...),
		PermissionEvidenceSource: permissionEvidenceSource, PermissionObservedAt: permissionObservedAt,
		TestResults: []ExtensionSiteResult{testResult}, Summary: summary, ReportTime: time.Now().UTC(),
	}
}

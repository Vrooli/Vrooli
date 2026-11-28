package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

// ManagedExtensionRequest forwards one lifecycle call to this exact BAS
// session's existing Playwright driver lease. The caller must have already
// passed Broker authorization; this method independently binds the request to
// the API-owned session, profile and immutable owner/lease before forwarding.
func (s *Session) ManagedExtensionRequest(ctx context.Context, ownerID, leaseID, profileID, permissionRef, callerID, method, suffix string, body []byte) (*http.Response, error) {
	if s == nil || s.isClosed() {
		return nil, errors.New("managed extension session is closed")
	}
	if strings.TrimSpace(ownerID) == "" || strings.TrimSpace(leaseID) == "" ||
		ownerID != s.executionID || leaseID != s.leaseID {
		return nil, errors.New("managed extension request does not match the session owner lease")
	}
	s.mu.RLock()
	profile := s.permissionProfile
	appTarget := s.appTarget
	s.mu.RUnlock()
	if appTarget {
		return nil, errors.New("managed extension operations are unsupported on externally controlled app targets")
	}
	if profile == nil || strings.TrimSpace(profileID) == "" || profile.ProfileID != profileID || profile.ID != permissionRef {
		return nil, errors.New("managed extension request does not match the admitted permission profile")
	}
	s.mu.RLock()
	callerResolver := s.callerResolver
	s.mu.RUnlock()
	if callerResolver == nil {
		return nil, errors.New("managed extension request has no trusted caller resolver")
	}
	resolvedCaller, err := callerResolver.BrowserCallerID(ctx)
	if err != nil || strings.TrimSpace(callerID) == "" || resolvedCaller != callerID {
		return nil, errors.New("managed extension request does not match the trusted browser caller")
	}
	if profile.ProfileMode != "fresh" && profile.ProfileMode != "isolated" {
		return nil, errors.New("managed extension operations require a managed fresh or isolated profile")
	}
	if s.browserBackend != "managed" {
		return nil, errors.New("extension lifecycle and inventory are supported only by the managed browser backend")
	}

	allowed := map[string]string{
		http.MethodPost + " /extensions/install":   "install",
		http.MethodPost + " /extensions/load":      "load",
		http.MethodPost + " /extensions/reload":    "load",
		http.MethodPost + " /extensions/unload":    "load",
		http.MethodPost + " /extensions/validate":  "load",
		http.MethodGet + " /extensions/inventory":  "inventory",
		http.MethodPost + " /extensions/inventory": "inventory",
	}
	kind, ok := allowed[method+" "+suffix]
	if !ok {
		return nil, errors.New("managed extension driver route is not allowed")
	}
	// Extension routes do not carry an approval receipt to the driver. Mirror
	// the permission owner's approval classification here so a direct Session
	// route cannot bypass approval profiles. Safety-pass permits lower-impact
	// lifecycle operations, but installation and every-action profiles fail
	// closed until this route contract can carry a verified receipt.
	if profile.Approval == sessionpermission.ApprovalEveryAction ||
		(profile.Approval == sessionpermission.ApprovalSafetyPass && suffix == "/extensions/install") {
		return nil, errors.New("managed extension route requires an approval receipt that this route does not support")
	}
	type artifactRequest struct {
		ExtensionID     string   `json:"extension_id"`
		ArtifactRef     string   `json:"artifact_ref"`
		Digest          string   `json:"artifact_sha256"`
		ManifestSHA     string   `json:"manifest_sha256"`
		Version         string   `json:"version"`
		Permissions     []string `json:"permissions"`
		HostPermissions []string `json:"host_permissions"`
		PermissionProbe string   `json:"permission_probe"`
	}
	var request struct {
		ExtensionID string          `json:"extension_id"`
		Artifact    artifactRequest `json:"artifact"`
		SessionID   string          `json:"session_id"`
		ProfileID   string          `json:"profile_id"`
		Artifacts   []struct {
			DirectoryPath string          `json:"directory_path"`
			Artifact      artifactRequest `json:"artifact"`
		} `json:"artifacts"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &request); err != nil {
			return nil, errors.New("managed extension driver request is invalid")
		}
	}
	switch kind {
	case "install":
		if !profile.ExtensionInstall {
			return nil, errors.New("extension installation is disabled in the admitted profile")
		}
	case "load":
		if !profile.ExtensionLoad {
			return nil, errors.New("extension loading is disabled in the admitted profile")
		}
	case "inventory":
		if !profile.ExtensionAccess {
			return nil, errors.New("extension inventory is disabled in the admitted profile")
		}
		if method == http.MethodPost {
			if request.SessionID != s.id || request.ProfileID != profile.ProfileID || len(request.Artifacts) == 0 {
				return nil, errors.New("managed extension inventory request does not match the session profile or verified artifacts")
			}
			for _, candidate := range request.Artifacts {
				a := candidate.Artifact
				if strings.TrimSpace(candidate.DirectoryPath) == "" || a.ExtensionID == "" || a.ArtifactRef == "" || a.Digest == "" || a.ManifestSHA == "" || a.Version == "" {
					return nil, errors.New("managed extension inventory request contains incomplete artifact metadata")
				}
				approved, exists := profile.ExtensionPermissions[a.ExtensionID]
				if !exists || len(approved) == 0 {
					return nil, errors.New("managed extension inventory artifact is not allowlisted by the admitted profile")
				}
				for _, declared := range append(append([]string(nil), a.Permissions...), a.HostPermissions...) {
					if !containsExtensionPermission(approved, declared) {
						return nil, errors.New("managed extension inventory artifact requests a permission outside the admitted profile")
					}
				}
				if a.PermissionProbe != "" && a.PermissionProbe != sessionpermission.ExtensionPermissionSourceChromeGetAll {
					return nil, errors.New("managed extension inventory names an unsupported runtime permission probe")
				}
			}
		}
	}
	if kind != "inventory" {
		extensionID := request.Artifact.ExtensionID
		if extensionID == "" || request.Artifact.ArtifactRef == "" || request.Artifact.Digest == "" || request.Artifact.ManifestSHA == "" || request.Artifact.Version == "" {
			return nil, errors.New("managed extension driver request requires verified artifact identity metadata")
		}
		if request.ExtensionID != "" && request.ExtensionID != extensionID {
			return nil, errors.New("managed extension request id differs from the verified artifact")
		}
		permissions, exists := profile.ExtensionPermissions[extensionID]
		if !exists || len(permissions) == 0 {
			return nil, errors.New("extension id is not allowlisted by the admitted profile")
		}
		for _, declared := range append(append([]string(nil), request.Artifact.Permissions...), request.Artifact.HostPermissions...) {
			if !containsExtensionPermission(permissions, declared) {
				return nil, errors.New("verified extension artifact requests a permission outside the admitted profile")
			}
		}
		if request.Artifact.PermissionProbe != "" && request.Artifact.PermissionProbe != sessionpermission.ExtensionPermissionSourceChromeGetAll {
			return nil, errors.New("managed extension request names an unsupported runtime permission probe")
		}
	}
	return s.client.SessionRouteRequest(ctx, s.id, s.executionID, s.leaseID, method, suffix, body, nil)
}

func containsExtensionPermission(allowed []string, requested string) bool {
	for _, permission := range allowed {
		if permission == requested {
			return true
		}
	}
	return false
}

// Package target contains target-owned desktop validation seams. It is kept
// independent from the HTTP handlers so the security boundary is testable
// before a real Electron process is launched.
package target

import (
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

// ElectronLaunchOptions are the target-owned inputs used to start a
// validation-only Electron process. The returned arguments are already split
// for exec.Cmd; callers must not concatenate them into a shell command.
type ElectronLaunchOptions struct {
	CDPPort          int
	UserDataDir      string
	ContextID        string
	ScenarioName     string
	ArtifactDigest   string
	TargetID         string
	JourneyID        string
	ProfileID        string
	IsolationLeaseID string
	BundledRuntime   bool
	OfflineNetwork   bool
}

// BuildElectronLaunchArguments creates the narrow Chromium/Electron launch
// seam required by BAS. It deliberately uses a dedicated profile and numeric
// loopback binding. Validation identity travels in the environment rather
// than argv so it cannot be mistaken for an application argument or exposed
// in process listings as a bearer token.
func BuildElectronLaunchArguments(opts ElectronLaunchOptions) ([]string, map[string]string, error) {
	if opts.CDPPort < 1 || opts.CDPPort > 65535 {
		return nil, nil, fmt.Errorf("electron CDP port must be between 1 and 65535")
	}
	if strings.TrimSpace(opts.UserDataDir) == "" {
		return nil, nil, fmt.Errorf("electron validation user-data directory is required")
	}
	identity := []struct {
		name  string
		value string
	}{
		{"context_id", opts.ContextID},
		{"scenario_name", opts.ScenarioName},
		{"artifact_digest", opts.ArtifactDigest},
		{"target_id", opts.TargetID},
		{"journey_id", opts.JourneyID},
		{"profile_id", opts.ProfileID},
		{"isolation_lease_id", opts.IsolationLeaseID},
	}
	for _, item := range identity {
		if strings.TrimSpace(item.value) == "" {
			return nil, nil, fmt.Errorf("electron validation %s is required", item.name)
		}
	}

	args := []string{
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=" + strconv.Itoa(opts.CDPPort),
		"--user-data-dir=" + opts.UserDataDir,
	}
	// AppImages launched with APPIMAGE_EXTRACT_AND_RUN cannot retain the
	// packaged chrome-sandbox setuid bit in the temporary extraction tree.
	// The validation target is already constrained to a managed display,
	// numeric loopback CDP, and an isolated profile, so Linux must explicitly
	// select Chromium's supported unsandboxed mode for this launch seam.
	if runtime.GOOS == "linux" {
		args = append(args, "--no-sandbox")
	}
	env := map[string]string{
		"VROOLI_VALIDATION_CONTEXT_ID":      opts.ContextID,
		"VROOLI_VALIDATION_SCENARIO":        opts.ScenarioName,
		"VROOLI_VALIDATION_ARTIFACT_DIGEST": opts.ArtifactDigest,
		"VROOLI_VALIDATION_TARGET_ID":       opts.TargetID,
		"VROOLI_VALIDATION_JOURNEY_ID":      opts.JourneyID,
		"VROOLI_VALIDATION_PROFILE_ID":      opts.ProfileID,
		"VROOLI_VALIDATION_ISOLATION_LEASE": opts.IsolationLeaseID,
	}
	return args, env, nil
}

// ValidateAppTarget rejects a target unless BAS can attach through an
// authenticated loopback-owned CDP transport. Hostname aliases are rejected:
// the target must prove a numeric loopback address rather than relying on DNS
// resolution that could change between validation and attachment.
func ValidateAppTarget(target *domainv1.AppTarget) error {
	if target == nil {
		return fmt.Errorf("electron target is required")
	}
	for name, value := range map[string]string{
		"target_id":       target.GetTargetId(),
		"scenario_name":   target.GetScenarioName(),
		"artifact_digest": target.GetArtifactDigest(),
		"context_id":      target.GetContextId(),
		"process_id":      target.GetProcessId(),
		"cdp_endpoint":    target.GetCdpEndpoint(),
		"cdp_transport":   target.GetCdpTransport(),
		"renderer_id":     target.GetRendererId(),
		"renderer_url":    target.GetRendererUrl(),
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("electron target %s is required", name)
		}
	}
	if !target.GetLoopbackOnly() {
		return fmt.Errorf("electron CDP must be loopback-only")
	}
	if !target.GetAuthenticated() {
		return fmt.Errorf("electron CDP transport must be authenticated")
	}
	if target.GetCdpTransport() != "loopback-authenticated" && target.GetCdpTransport() != "bridge-authenticated" {
		return fmt.Errorf("unsupported authenticated CDP transport %q", target.GetCdpTransport())
	}

	u, err := url.Parse(target.GetCdpEndpoint())
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid CDP endpoint")
	}
	if err := ValidateAppTargetEndpoint(u); err != nil {
		return err
	}
	if apiURL := strings.TrimSpace(target.GetApiUrl()); apiURL != "" {
		if _, err := offlineAPIPort(apiURL); err != nil {
			return fmt.Errorf("invalid app target API URL: %w", err)
		}
	}
	return nil
}

// ValidateAppTargetEndpoint applies the shared endpoint boundary to
// discovery and attachment callers. Numeric loopback plus an explicit port
// avoids DNS rebinding and accidental exposure on a host interface.
func ValidateAppTargetEndpoint(u *url.URL) error {
	if u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid CDP endpoint")
	}
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("unsupported CDP endpoint scheme %q", u.Scheme)
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("CDP endpoint must use a numeric loopback address")
	}
	if u.Port() == "" {
		return fmt.Errorf("CDP endpoint must include a port")
	}
	return nil
}

// ValidateElectronContext binds the target to the exact validation cell and
// rejects expired or mismatched isolation authority before any app request is
// allowed to run.
func ValidateElectronContext(target *domainv1.AppTarget, context *domainv1.ValidationContext, now time.Time) error {
	if err := ValidateAppTarget(target); err != nil {
		return err
	}
	if context == nil {
		return fmt.Errorf("validation context is required")
	}
	if strings.TrimSpace(context.GetContextId()) == "" || strings.TrimSpace(context.GetJourneyId()) == "" || strings.TrimSpace(context.GetProfileId()) == "" {
		return fmt.Errorf("validation context identity is incomplete")
	}
	if strings.TrimSpace(context.GetIsolationLeaseId()) == "" {
		return fmt.Errorf("validation context requires an isolation lease")
	}
	if context.GetScenarioName() != target.GetScenarioName() {
		return fmt.Errorf("validation context scenario does not match target")
	}
	if context.GetArtifactDigest() != target.GetArtifactDigest() {
		return fmt.Errorf("validation context artifact does not match target")
	}
	if context.GetTargetId() != target.GetTargetId() {
		return fmt.Errorf("validation context target does not match target identity")
	}
	if context.GetExpiresAt() == nil || !context.GetExpiresAt().AsTime().After(now) {
		return fmt.Errorf("validation context lease is expired")
	}
	return nil
}

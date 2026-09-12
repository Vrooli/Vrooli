package vps

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/internal/stringutil"
	"scenario-to-cloud/secrets"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// DeployRequest is the request body for VPS deployment.
type DeployRequest struct {
	Manifest domain.CloudManifest `json:"manifest"`
	// PlanDigest is the semantic digest of the reviewed plan (apply only).
	PlanDigest string `json:"plan_digest,omitempty"`
	// DeploymentID and RequestKey identify the durable operation owner; apply
	// refuses requests that omit them.
	DeploymentID string `json:"deployment_id,omitempty"`
	RequestKey   string `json:"request_key,omitempty"`
}

// ValidateUserPromptSecrets checks that all required user_prompt secrets are provided.
// Returns nil if all required secrets are present, or an error with details about missing secrets.
func ValidateUserPromptSecrets(manifest domain.CloudManifest, providedSecrets map[string]string) ([]domain.MissingSecretInfo, error) {
	if manifest.Secrets == nil || len(manifest.Secrets.BundleSecrets) == 0 {
		return nil, nil // No secrets required
	}

	var missing []domain.MissingSecretInfo
	for _, secret := range manifest.Secrets.BundleSecrets {
		if secret.Class != "user_prompt" {
			continue // Not a user-provided secret
		}
		if secret.Descriptor == nil && strings.TrimSpace(secret.DescriptorReason) == "" {
			return nil, fmt.Errorf("user_prompt secret %q has no credential descriptor address or documented reason", secret.ID)
		}
		if !secret.Required {
			continue // Optional secret
		}

		key := secret.Target.Name
		if key == "" {
			key = secret.ID
		}

		// Check if secret was provided
		if _, ok := providedSecrets[key]; ok {
			continue // Secret provided
		}
		if secret.Descriptor != nil {
			address := strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
			if _, ok := providedSecrets[address]; ok {
				continue
			}
		}

		// Secret is missing - collect info for error message
		label := key
		description := secret.Description
		if secret.Prompt != nil {
			if secret.Prompt.Label != "" {
				label = secret.Prompt.Label
			}
			if secret.Prompt.Description != "" {
				description = secret.Prompt.Description
			}
		}

		missing = append(missing, domain.MissingSecretInfo{
			ID:          secret.ID,
			Key:         key,
			Label:       label,
			Description: description,
		})
	}

	if len(missing) == 0 {
		return nil, nil
	}

	// Build actionable error message
	var sb strings.Builder
	sb.WriteString("missing required secrets:\n")
	for _, m := range missing {
		sb.WriteString(fmt.Sprintf("  - %s (%s)", m.Label, m.Key))
		if m.Description != "" {
			sb.WriteString(": " + m.Description)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nProvide secrets via --secret KEY=VALUE flags or environment variables")

	return missing, fmt.Errorf("%s", sb.String())
}

// credentialFieldFor derives the durable field name for a bundle secret.
//
// The normalization itself lives in secrets.CredentialField and is shared with
// the remote provisioning path deliberately: a value written under one
// normalization and read under another is a credential that silently is not
// there. Only the choice of which part of the plan names it belongs here.
func credentialFieldFor(secret domain.BundleSecretPlan) string {
	if secret.Descriptor != nil && strings.TrimSpace(secret.Descriptor.Field) != "" {
		return secrets.CredentialField(secret.Descriptor.Field)
	}
	raw := strings.TrimSpace(secret.ID)
	if raw == "" {
		raw = strings.TrimSpace(secret.Target.Name)
	}
	return secrets.CredentialField(raw)
}

// buildUserSecretMap resolves the operator-supplied secrets a bundle needs.
//
// Values come from the credential authority, never from a file. The two
// plaintext stores this used to read — ~/.vrooli/secrets.json and
// ~/.vrooli/scenarios/<id>/secrets.json — are gone along with the API that
// maintained them, so a cloud deploy no longer depends on, or recreates, a
// credential sitting unencrypted on the operator's disk.
//
// The identity namespace is the same one Tier 1 and Tier 2 use,
// vrooli/<scenario>, so a credential provisioned once during onboarding is the
// credential a cloud deploy ships. That is the whole point of a durable
// backend-neutral name: the deployment tier must not change where a value
// lives.
func buildUserSecretMap(manifest domain.CloudManifest, providedSecrets map[string]string) (map[string]string, error) {
	if manifest.Secrets == nil || len(manifest.Secrets.BundleSecrets) == 0 {
		return nil, nil
	}

	authority, authErr := credentialauthority.Default()
	identity, identityErr := credentialauthority.ParseIdentity("vrooli/" + strings.TrimSpace(manifest.Scenario.ID))
	if authErr != nil {
		return nil, fmt.Errorf("initialize credential authority: %w", authErr)
	}
	if identityErr != nil {
		return nil, fmt.Errorf("parse deployment identity: %w", identityErr)
	}

	out := make(map[string]string)
	descriptorKey := func(address *domain.DescriptorAddress) string {
		if address == nil {
			return ""
		}
		return strings.TrimSpace(address.LogicalID) + ":" + strings.TrimSpace(address.Field)
	}
	for _, secret := range manifest.Secrets.BundleSecrets {
		if secret.Class != "user_prompt" {
			continue
		}
		key := secret.Target.Name
		if key == "" {
			key = secret.ID
		}
		if key == "" {
			continue
		}
		descriptor := descriptorKey(secret.Descriptor)
		if descriptor != "" {
			if v, ok := providedSecrets[descriptor]; ok && strings.TrimSpace(v) != "" {
				out[key] = v
				continue
			}
		}
		if v, ok := providedSecrets[key]; ok && strings.TrimSpace(v) != "" {
			out[key] = v
			continue
		}

		// Merge precedence (lowest -> highest): the stored credential, then a
		// value the caller supplied explicitly for this deploy.
		resolveIdentity := identity
		if secret.Descriptor != nil {
			resolveIdentity, identityErr = credentialauthority.ParseIdentity(strings.TrimSpace(secret.Descriptor.LogicalID))
			if identityErr != nil {
				return nil, fmt.Errorf("parse descriptor identity for %s: %w", key, identityErr)
			}
		}
		if field := credentialFieldFor(secret); field != "" {
			// Unconfigured is handled by the caller's missing-secret check, but
			// provider failures must remain visible and never become empty input.
			value, resolveErr := authority.Require(resolveIdentity, field)
			if resolveErr == nil && strings.TrimSpace(value) != "" {
				out[key] = value
			} else if resolveErr != nil && !errors.Is(resolveErr, credentialauthority.ErrUnconfigured) {
				return nil, fmt.Errorf("resolve deployment credential %s:%s: %w", resolveIdentity, field, resolveErr)
			}
		}
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ServiceJSON represents the structure of .vrooli/service.json
type ServiceJSON struct {
	Service struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	} `json:"service"`
	Ports        map[string]PortConfig `json:"ports"`
	Dependencies struct {
		Resources map[string]ResourceDependency     `json:"resources"`
		Scenarios map[string]ScenarioDependencySpec `json:"scenarios"`
	} `json:"dependencies"`
}

// PortConfig represents a port configuration from service.json
type PortConfig struct {
	EnvVar      string `json:"env_var,omitempty"`
	Description string `json:"description,omitempty"`
	Range       string `json:"range,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// ResourceDependency represents a resource dependency from service.json
type ResourceDependency struct {
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// ScenarioDependencySpec represents a scenario dependency from service.json
type ScenarioDependencySpec struct {
	Enabled     bool   `json:"enabled"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// BundleFinder is an interface for finding the repo root.
type BundleFinder interface {
	FindRepoRootFromCWD() (string, error)
}

// DefaultBundleFinder uses the bundle package to find the repo root.
type DefaultBundleFinder struct{}

// FindRepoRootFromCWD finds the repo root from the current working directory.
func (DefaultBundleFinder) FindRepoRootFromCWD() (string, error) {
	return bundle.FindRepoRootFromCWD()
}

var bundleFinder BundleFinder = DefaultBundleFinder{}

// RequiredResourcesForScenario reads the service.json file for a scenario
// and returns the list of required resource IDs.
func RequiredResourcesForScenario(scenarioID string) ([]string, error) {
	repoRoot, err := bundleFinder.FindRepoRootFromCWD()
	if err != nil {
		return nil, fmt.Errorf("repo root not found for dependency validation: %w", err)
	}
	serviceJSONPath, err := bundle.ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err != nil {
		return nil, fmt.Errorf("resolve service.json for dependency validation: %w", err)
	}
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read service.json for dependency validation: %w", err)
	}
	var svc ServiceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parse service.json for dependency validation: %w", err)
	}
	var required []string
	for name, dep := range svc.Dependencies.Resources {
		if dep.Enabled || dep.Required {
			required = append(required, name)
		}
	}
	return stringutil.SortedUnique(required), nil
}

func validateManifestResourceDependencies(manifest domain.CloudManifest) error {
	required, err := RequiredResourcesForScenario(manifest.Scenario.ID)
	if err != nil {
		return err
	}
	if len(required) == 0 {
		return nil
	}
	declared := stringutil.SortedUnique(manifest.Dependencies.Resources)
	var missing []string
	for _, name := range required {
		if !stringutil.Contains(declared, name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("deployment manifest missing required resources: %s. Re-export the manifest from scenario-dependency-analyzer or ensure .vrooli/service.json resources are captured in dependencies.resources", strings.Join(missing, ", "))
	}
	return nil
}

// BuildDeployExecutablePlan compiles the runtime-scope plan for a manifest.
// Preview and apply both call this.
func BuildDeployExecutablePlan(ctx context.Context, manifest domain.CloudManifest, scope string) (*execplan.Plan, error) {
	if scope == "" {
		scope = execplan.ScopeRuntime
	}
	return CompilePlan(ctx, PlanRequest{Manifest: manifest, Scope: scope})
}

// BuildDeployPlan is the legacy step view of the runtime plan: id = action
// id, command = shell preview. It is derived from the same compiled plan the
// executor runs.
func BuildDeployPlan(manifest domain.CloudManifest) ([]domain.VPSPlanStep, error) {
	plan, err := BuildDeployExecutablePlan(context.Background(), manifest, execplan.ScopeRuntime)
	if err != nil {
		return nil, err
	}
	return RenderSteps(plan, manifest), nil
}

// DeployOptions configures which plan scope runs and how progress is
// weighted. There is no per-step allowlist: the compiled plan is the only
// selection of actions.
type DeployOptions struct {
	// Scope is execplan.ScopeRuntime (default) or execplan.ScopeStart.
	Scope string
	// StepWeights overrides the default action weights. If nil, uses StepWeights.
	StepWeights map[string]float64
}

// RunDeployPlanWithProgress executes an already compiled (and reviewed)
// runtime plan. Results are keyed by action id.
func RunDeployPlanWithProgress(
	ctx context.Context,
	plan *execplan.Plan,
	manifest domain.CloudManifest,
	rt Runtime,
	hub ProgressBroadcaster,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
	opts DeployOptions,
) domain.VPSDeployResult {
	start := time.Now()
	steps := RenderSteps(plan, manifest)
	digest, _ := plan.SemanticDigest()
	trace, execErr := ExecutePlan(ctx, ExecuteRequest{Plan: plan, Manifest: manifest, Runtime: rt, Hub: hub, Repo: repo, DeploymentID: deploymentID, Progress: progress, Weights: opts.StepWeights})
	result := domain.VPSDeployResult{
		OK:         execErr == nil,
		Steps:      steps,
		PlanDigest: digest,
		Actions:    trace.Actions,
		DurationMs: time.Since(start).Milliseconds(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if execErr != nil {
		result.Error = execErr.Error()
		result.ErrorInfo = execErr.Info
		result.FailedStep = execErr.ActionID
	}
	return result
}

func checkPublicHealth(ctx context.Context, url string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build public health request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("public health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("public health check returned %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("public health check returned %s", resp.Status)
	}
	return nil
}

var checkOriginHealthFunc = checkOriginHealth

func checkOriginHealth(ctx context.Context, domain, host string, timeout time.Duration) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("origin health check failed: domain is empty")
	}
	ip, err := resolveHostIP(ctx, host)
	if err != nil {
		return fmt.Errorf("origin health check failed: resolve VPS host %q: %w", host, err)
	}
	targetURL := fmt.Sprintf("https://%s/health", domain)
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip, "443"))
		},
		TLSClientConfig: &tls.Config{
			ServerName: strings.TrimSpace(domain),
		},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return fmt.Errorf("origin health check failed: build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("origin health check failed for %s via %s: %w", domain, ip, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("origin health check returned %s via %s: %s", resp.Status, ip, msg)
		}
		return fmt.Errorf("origin health check returned %s via %s", resp.Status, ip)
	}
	return nil
}

func resolveHostIP(ctx context.Context, host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is empty")
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip.IP != nil {
			return ip.IP.String(), nil
		}
	}
	return "", fmt.Errorf("no IPs found for host")
}

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
		if err := validatePromptDescriptor(secret); err != nil {
			return nil, err
		}
		if !secret.Required {
			continue // Optional secret
		}
		key := promptSecretKey(secret)
		if promptSecretProvided(secret, key, providedSecrets) {
			continue
		}
		missing = append(missing, missingPromptSecretInfo(secret, key))
	}

	if len(missing) == 0 {
		return nil, nil
	}

	return missing, missingPromptSecretsError(missing)
}

func validatePromptDescriptor(secret domain.BundleSecretPlan) error {
	if secret.Descriptor == nil && strings.TrimSpace(secret.DescriptorReason) == "" {
		return fmt.Errorf("user_prompt secret %q has no credential descriptor address or documented reason", secret.ID)
	}
	return nil
}

func promptSecretKey(secret domain.BundleSecretPlan) string {
	if secret.Target.Name != "" {
		return secret.Target.Name
	}
	return secret.ID
}

func promptSecretProvided(secret domain.BundleSecretPlan, key string, provided map[string]string) bool {
	if _, ok := provided[key]; ok {
		return true
	}
	if secret.Descriptor == nil {
		return false
	}
	address := strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
	_, ok := provided[address]
	return ok
}

func missingPromptSecretInfo(secret domain.BundleSecretPlan, key string) domain.MissingSecretInfo {
	label, description := key, secret.Description
	if secret.Prompt != nil {
		if secret.Prompt.Label != "" {
			label = secret.Prompt.Label
		}
		if secret.Prompt.Description != "" {
			description = secret.Prompt.Description
		}
	}
	return domain.MissingSecretInfo{ID: secret.ID, Key: key, Label: label, Description: description}
}

func missingPromptSecretsError(missing []domain.MissingSecretInfo) error {
	var message strings.Builder
	message.WriteString("missing required secrets:\n")
	for _, secret := range missing {
		message.WriteString(fmt.Sprintf("  - %s (%s)", secret.Label, secret.Key))
		if secret.Description != "" {
			message.WriteString(": " + secret.Description)
		}
		message.WriteString("\n")
	}
	message.WriteString("\nProvide secrets via --secret KEY=VALUE flags or environment variables")
	return fmt.Errorf("%s", message.String())
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
	authority, identity, err := deploymentCredentialIdentity(manifest)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, secret := range manifest.Secrets.BundleSecrets {
		key, value, found, err := resolveUserPromptSecret(authority, identity, secret, providedSecrets)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func deploymentCredentialIdentity(manifest domain.CloudManifest) (*credentialauthority.Authority, credentialauthority.Identity, error) {
	authority, authErr := credentialauthority.Default()
	identity, identityErr := credentialauthority.ParseIdentity("vrooli/" + strings.TrimSpace(manifest.Scenario.ID))
	if authErr != nil {
		return nil, "", fmt.Errorf("initialize credential authority: %w", authErr)
	}
	if identityErr != nil {
		return nil, "", fmt.Errorf("parse deployment identity: %w", identityErr)
	}
	return authority, identity, nil
}

func resolveUserPromptSecret(authority *credentialauthority.Authority, identity credentialauthority.Identity, secret domain.BundleSecretPlan, provided map[string]string) (string, string, bool, error) {
	if secret.Class != "user_prompt" {
		return "", "", false, nil
	}
	key := promptSecretKey(secret)
	if key == "" {
		return "", "", false, nil
	}
	if value, found := suppliedPromptSecret(secret, key, provided); found {
		return key, value, true, nil
	}
	resolveIdentity, err := credentialIdentityForSecret(identity, secret, key)
	if err != nil {
		return "", "", false, err
	}
	field := credentialFieldFor(secret)
	if field == "" {
		return "", "", false, nil
	}
	value, err := authority.Require(resolveIdentity, field)
	if err == nil && strings.TrimSpace(value) != "" {
		return key, value, true, nil
	}
	if err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
		return "", "", false, fmt.Errorf("resolve deployment credential %s:%s: %w", resolveIdentity, field, err)
	}
	return "", "", false, nil
}

func suppliedPromptSecret(secret domain.BundleSecretPlan, key string, provided map[string]string) (string, bool) {
	if secret.Descriptor != nil {
		address := strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
		if value, ok := provided[address]; ok && strings.TrimSpace(value) != "" {
			return value, true
		}
	}
	value, ok := provided[key]
	return value, ok && strings.TrimSpace(value) != ""
}

func credentialIdentityForSecret(identity credentialauthority.Identity, secret domain.BundleSecretPlan, key string) (credentialauthority.Identity, error) {
	if secret.Descriptor == nil {
		return identity, nil
	}
	resolved, err := credentialauthority.ParseIdentity(strings.TrimSpace(secret.Descriptor.LogicalID))
	if err != nil {
		return "", fmt.Errorf("parse descriptor identity for %s: %w", key, err)
	}
	return resolved, nil
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

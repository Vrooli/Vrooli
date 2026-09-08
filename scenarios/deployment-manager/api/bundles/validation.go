package bundles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"deployment-manager/shared"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

var (
	desktopSchemaOnce sync.Once
	desktopSchema     *jsonschema.Schema
	desktopSchemaErr  error
)

// ValidateManifestBytes performs a lightweight structural validation.
// Note: We intentionally do NOT use DisallowUnknownFields() here because:
//  1. The JSON Schema validation at the end catches any truly invalid fields
//  2. Strict parsing breaks compatibility with analyzer responses that may include
//     additional metadata fields alongside the core manifest structure
//  3. Being lenient here allows forward compatibility with schema extensions
func ValidateManifestBytes(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	// Removed: dec.DisallowUnknownFields() - JSON schema validation handles this

	var manifest Manifest
	if err := dec.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate schema version using domain decision helper
	if !shared.IsValidSchemaVersion(manifest.SchemaVersion) {
		return fmt.Errorf("schema_version must be %s", shared.BundleSchemaVersionV01)
	}

	// Validate bundle target using domain decision helper
	if !shared.IsValidBundleTarget(manifest.Target) {
		return fmt.Errorf("target must be %s", shared.BundleTargetDesktop)
	}

	if manifest.App.Name == "" || manifest.App.Version == "" {
		return fmt.Errorf("app.name and app.version are required")
	}

	// Validate IPC mode using domain decision helper
	if !shared.IsValidIPCMode(manifest.IPC.Mode) || manifest.IPC.Host == "" || manifest.IPC.Port == 0 || manifest.IPC.AuthTokenPath == "" {
		return fmt.Errorf("ipc must define %s host, port, and auth_token_path", shared.IPCModeLoopbackHTTP)
	}
	if manifest.Telemetry.File == "" {
		return fmt.Errorf("telemetry.file is required")
	}
	if len(manifest.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}
	if err := validateAuthentication(manifest.Authentication, manifest.Services); err != nil {
		return err
	}

	for _, secret := range manifest.Secrets {
		if err := validateSecret(secret); err != nil {
			return fmt.Errorf("secret %s: %w", secret.ID, err)
		}
	}

	for _, svc := range manifest.Services {
		if err := validateService(svc); err != nil {
			return fmt.Errorf("service %s: %w", svc.ID, err)
		}
	}
	if err := validateAgainstDesktopSchema(data); err != nil {
		return err
	}
	return nil
}

func validateAuthentication(profile *AuthenticationProfile, services []ServiceEntry) error {
	if profile == nil {
		return nil
	}
	if err := validateAuthenticationProfile(profile, services); err != nil {
		return err
	}
	for mode := range profile.ModeProfiles {
		alternate, ok := profile.profileForMode(mode)
		if !ok {
			return fmt.Errorf("authentication.mode_profiles[%q] is not readable", mode)
		}
		if err := validateAuthenticationProfile(alternate, services); err != nil {
			return fmt.Errorf("authentication.mode_profiles[%q]: %w", mode, err)
		}
	}
	return nil
}

func (profile *AuthenticationProfile) profileForMode(mode string) (*AuthenticationProfile, bool) {
	if profile == nil {
		return nil, false
	}
	if mode == profile.Mode {
		copy := *profile
		copy.ModeProfiles = nil
		return &copy, true
	}
	config, ok := profile.ModeProfiles[mode]
	if !ok {
		return nil, false
	}
	return &AuthenticationProfile{
		Version:               profile.Version,
		Mode:                  mode,
		Provider:              config.Provider,
		Resource:              config.Resource,
		Audience:              config.Audience,
		ProviderEndpoint:      config.ProviderEndpoint,
		ProviderServiceID:     config.ProviderServiceID,
		HumanSignIn:           config.HumanSignIn,
		Offline:               config.Offline,
		PublicRoutes:          append([]string(nil), config.PublicRoutes...),
		ProtectedRoutes:       append([]string(nil), config.ProtectedRoutes...),
		LeasePath:             config.LeasePath,
		RecoveryURL:           config.RecoveryURL,
		RequiresAuthenticator: config.RequiresAuthenticator,
	}, true
}

func validateAuthenticationProfile(profile *AuthenticationProfile, services []ServiceEntry) error {
	if profile.Version != 1 {
		return fmt.Errorf("authentication.version must be 1")
	}
	switch profile.Mode {
	case "personal_local":
		if profile.Provider != "" || !profile.Offline || profile.RequiresAuthenticator {
			return fmt.Errorf("personal_local authentication must be offline, provider-free, and authenticator-free")
		}
	case "local_multi_user":
		if profile.Provider != "scenario-authenticator" || profile.HumanSignIn != "required" {
			return fmt.Errorf("local_multi_user authentication requires scenario-authenticator and human sign-in")
		}
	case "remote_vrooli":
		if profile.Provider != "scenario-authenticator" || profile.HumanSignIn != "required" || strings.TrimSpace(profile.ProviderEndpoint) == "" {
			return fmt.Errorf("remote_vrooli authentication requires scenario-authenticator, an endpoint, and human sign-in")
		}
		if err := validateProviderEndpoint(profile.ProviderEndpoint); err != nil {
			return fmt.Errorf("remote_vrooli authentication provider endpoint: %w", err)
		}
	case "shared_provider":
		if profile.Provider == "" || profile.HumanSignIn != "required" || strings.TrimSpace(profile.ProviderEndpoint) == "" || strings.TrimSpace(profile.LeasePath) == "" {
			return fmt.Errorf("shared_provider authentication requires a provider, endpoint, lease path, and human sign-in")
		}
		if err := validateProviderEndpoint(profile.ProviderEndpoint); err != nil {
			return fmt.Errorf("shared_provider authentication provider endpoint: %w", err)
		}
	default:
		return fmt.Errorf("authentication.mode %q is unsupported", profile.Mode)
	}
	if profile.Mode != "personal_local" && strings.TrimSpace(profile.Resource) == "" {
		return fmt.Errorf("authentication.resource is required for networked modes")
	}
	if profile.RequiresAuthenticator {
		if strings.TrimSpace(profile.ProviderServiceID) == "" {
			return fmt.Errorf("authentication.provider_service_id is required when authenticator startup is required")
		}
		for _, service := range services {
			if service.ID == profile.ProviderServiceID {
				return nil
			}
		}
		return fmt.Errorf("authentication provider service %q is not bundled", profile.ProviderServiceID)
	}
	return nil
}

func validateProviderEndpoint(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return fmt.Errorf("must be an HTTP(S) URL with a host and no credentials")
	}
	if strings.ContainsAny(raw, "\r\n\"'\\") {
		return fmt.Errorf("must not contain control characters or quoting")
	}
	return nil
}

func validateSecret(secret ManifestSecret) error {
	// Validate secret classification using domain decision helper
	if !shared.IsValidSecretClass(secret.Class) {
		return fmt.Errorf("unsupported class %q (valid: per_install_generated, user_prompt, remote_fetch, infrastructure)", secret.Class)
	}

	// Validate secret target is specified
	if secret.Target.Type == "" || secret.Target.Name == "" {
		return fmt.Errorf("target.type and target.name are required")
	}

	// Validate secret target type using domain decision helper
	if !shared.IsValidSecretTargetType(secret.Target.Type) {
		return fmt.Errorf("unsupported target.type %q (valid: env, file)", secret.Target.Type)
	}

	return nil
}

func validateService(svc ServiceEntry) error {
	if svc.ID == "" {
		return fmt.Errorf("id is required")
	}

	// Validate service type using domain decision helper
	if err := shared.GetServiceTypeError(svc.Type); err != nil {
		return err
	}

	requiresBinary := svc.Type != shared.ServiceTypeEmbeddedStorage
	if requiresBinary {
		if len(svc.Binaries) == 0 {
			return fmt.Errorf("at least one platform binary is required")
		}
		for platform, bin := range svc.Binaries {
			if bin.Path == "" {
				return fmt.Errorf("binary path is required for platform %s", platform)
			}
		}
	}
	if svc.Health.Type == "" {
		return fmt.Errorf("health.type is required")
	}
	if svc.Readiness.Type == "" {
		return fmt.Errorf("readiness.type is required")
	}
	return nil
}

func validateAgainstDesktopSchema(data []byte) error {
	schema, err := loadDesktopBundleSchema()
	if err != nil {
		return fmt.Errorf("failed to load desktop bundle schema: %w", err)
	}
	var payload interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("bundle schema validation failed: %w", err)
	}
	return nil
}

func loadDesktopBundleSchema() (*jsonschema.Schema, error) {
	desktopSchemaOnce.Do(func() {
		repoRoot, err := repocontract.ResolveRepoRoot()
		if err != nil {
			desktopSchemaErr = fmt.Errorf("resolve repository root: %w", err)
			return
		}
		scenarioRoot, err := repocontract.ResolveScenarioPath(repoRoot, "deployment-manager")
		if err != nil {
			desktopSchemaErr = fmt.Errorf("resolve deployment-manager path: %w", err)
			return
		}
		schemaPath := filepath.Join(scenarioRoot, "docs", "schemas", "bundle-schema.desktop.v0.1.json")
		schemaBytes, readErr := os.ReadFile(schemaPath)
		if readErr != nil {
			desktopSchemaErr = fmt.Errorf("failed to read bundle schema: %w", readErr)
			return
		}

		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("bundle-schema.desktop.v0.1.json", bytes.NewReader(schemaBytes)); err != nil {
			desktopSchemaErr = fmt.Errorf("failed to add schema resource: %w", err)
			return
		}
		desktopSchema, desktopSchemaErr = compiler.Compile("bundle-schema.desktop.v0.1.json")
	})
	return desktopSchema, desktopSchemaErr
}

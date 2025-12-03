package bundles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"deployment-manager/shared"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	desktopSchemaOnce sync.Once
	desktopSchema     *jsonschema.Schema
	desktopSchemaErr  error
)

// ValidateManifestBytes performs a lightweight structural validation.
func ValidateManifestBytes(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var manifest Manifest
	if err := dec.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid JSON or unexpected fields: %w", err)
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

	if len(svc.Binaries) == 0 {
		return fmt.Errorf("at least one platform binary is required")
	}
	for platform, bin := range svc.Binaries {
		if bin.Path == "" {
			return fmt.Errorf("binary path is required for platform %s", platform)
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
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			desktopSchemaErr = fmt.Errorf("unable to resolve schema path from caller")
			return
		}
		schemaPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "docs", "deployment", "bundle-schema.desktop.v0.1.json"))
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

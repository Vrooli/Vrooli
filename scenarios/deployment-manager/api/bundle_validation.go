package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	desktopSchemaOnce sync.Once
	desktopSchema     *jsonschema.Schema
	desktopSchemaErr  error
)

// desktopBundleManifest mirrors the v0.1 desktop bundle schema enough for
// lightweight validation without pulling in external schema validators.
type desktopBundleManifest struct {
	SchemaVersion string                 `json:"schema_version"`
	Target        string                 `json:"target"`
	App           manifestApp            `json:"app"`
	IPC           manifestIPC            `json:"ipc"`
	Telemetry     manifestTelemetry      `json:"telemetry"`
	Ports         *manifestPorts         `json:"ports,omitempty"`
	Swaps         []manifestSwap         `json:"swaps,omitempty"`
	Secrets       []manifestSecret       `json:"secrets,omitempty"`
	Services      []manifestServiceEntry `json:"services"`
}

type manifestApp struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type manifestIPC struct {
	Mode          string `json:"mode"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	AuthTokenPath string `json:"auth_token_path"`
}

type manifestTelemetry struct {
	File      string `json:"file"`
	UploadURL string `json:"upload_url"`
}

type manifestPorts struct {
	DefaultRange manifestPortRange `json:"default_range"`
	Reserved     []int             `json:"reserved"`
}

type manifestPortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type manifestSwap struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason"`
	Limitations string `json:"limitations"`
}

type manifestSecret struct {
	ID          string                 `json:"id"`
	Class       string                 `json:"class"`
	Description string                 `json:"description"`
	Format      string                 `json:"format"`
	Required    *bool                  `json:"required"`
	Prompt      *manifestSecretPrompt  `json:"prompt,omitempty"`
	Generator   map[string]interface{} `json:"generator,omitempty"`
	Target      manifestSecretTarget   `json:"target"`
}

type manifestSecretPrompt struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type manifestSecretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type manifestServiceEntry struct {
	ID           string                           `json:"id"`
	Type         string                           `json:"type"`
	Description  string                           `json:"description"`
	Binaries     map[string]manifestServiceBinary `json:"binaries"`
	Env          map[string]string                `json:"env,omitempty"`
	Secrets      []string                         `json:"secrets,omitempty"`
	DataDirs     []string                         `json:"data_dirs,omitempty"`
	LogDir       string                           `json:"log_dir,omitempty"`
	Ports        *manifestServicePorts            `json:"ports,omitempty"`
	Health       manifestHealth                   `json:"health"`
	Readiness    manifestReadiness                `json:"readiness"`
	Dependencies []string                         `json:"dependencies,omitempty"`
	Migrations   []manifestMigration              `json:"migrations,omitempty"`
	Assets       []manifestAsset                  `json:"assets,omitempty"`
	GPU          *manifestGPU                     `json:"gpu,omitempty"`
	Critical     *bool                            `json:"critical,omitempty"`
}

type manifestServiceBinary struct {
	Path string            `json:"path"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	Cwd  string            `json:"cwd,omitempty"`
}

type manifestServicePorts struct {
	Requested []manifestRequestedPort `json:"requested"`
}

type manifestRequestedPort struct {
	Name           string            `json:"name"`
	Range          manifestPortRange `json:"range"`
	RequiresSocket bool              `json:"requires_socket"`
}

type manifestHealth struct {
	Type     string   `json:"type"`
	Path     string   `json:"path"`
	PortName string   `json:"port_name"`
	Command  []string `json:"command,omitempty"`
	Interval int      `json:"interval_ms"`
	Timeout  int      `json:"timeout_ms"`
	Retries  int      `json:"retries"`
}

type manifestReadiness struct {
	Type     string `json:"type"`
	PortName string `json:"port_name"`
	Pattern  string `json:"pattern"`
	Timeout  int    `json:"timeout_ms"`
}

type manifestMigration struct {
	Version string            `json:"version"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	RunOn   string            `json:"run_on"`
}

type manifestAsset struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type manifestGPU struct {
	Requirement string `json:"requirement"`
}

// validateDesktopBundleManifestBytes performs a lightweight structural validation.
func validateDesktopBundleManifestBytes(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var manifest desktopBundleManifest
	if err := dec.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid JSON or unexpected fields: %w", err)
	}

	// Validate schema version using domain decision helper
	if !IsValidSchemaVersion(manifest.SchemaVersion) {
		return fmt.Errorf("schema_version must be %s", BundleSchemaVersionV01)
	}

	// Validate bundle target using domain decision helper
	if !IsValidBundleTarget(manifest.Target) {
		return fmt.Errorf("target must be %s", BundleTargetDesktop)
	}

	if manifest.App.Name == "" || manifest.App.Version == "" {
		return fmt.Errorf("app.name and app.version are required")
	}

	// Validate IPC mode using domain decision helper
	if !IsValidIPCMode(manifest.IPC.Mode) || manifest.IPC.Host == "" || manifest.IPC.Port == 0 || manifest.IPC.AuthTokenPath == "" {
		return fmt.Errorf("ipc must define %s host, port, and auth_token_path", IPCModeLoopbackHTTP)
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

func validateSecret(secret manifestSecret) error {
	// Validate secret classification using domain decision helper
	if !IsValidSecretClass(secret.Class) {
		return fmt.Errorf("unsupported class %q (valid: per_install_generated, user_prompt, remote_fetch, infrastructure)", secret.Class)
	}

	// Validate secret target is specified
	if secret.Target.Type == "" || secret.Target.Name == "" {
		return fmt.Errorf("target.type and target.name are required")
	}

	// Validate secret target type using domain decision helper
	if !IsValidSecretTargetType(secret.Target.Type) {
		return fmt.Errorf("unsupported target.type %q (valid: env, file)", secret.Target.Type)
	}

	return nil
}

func validateService(svc manifestServiceEntry) error {
	if svc.ID == "" {
		return fmt.Errorf("id is required")
	}

	// Validate service type using domain decision helper
	if err := GetServiceTypeError(svc.Type); err != nil {
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
		schemaPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "docs", "deployment", "bundle-schema.desktop.v0.1.json"))
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

package main

import "fmt"

// Secret class constants define the lifecycle and source of secrets.
// These determine how the runtime handles secret provisioning.
const (
	// SecretClassInfrastructure represents secrets that belong to infrastructure
	// and should NEVER be bundled (e.g., database admin passwords, cloud API keys).
	SecretClassInfrastructure = "infrastructure"

	// SecretClassPerInstallGenerated represents secrets that are generated
	// uniquely per installation (e.g., encryption keys, session secrets).
	SecretClassPerInstallGenerated = "per_install_generated"

	// SecretClassUserPrompt represents secrets that users must provide
	// during first-run setup (e.g., license keys, third-party API tokens).
	SecretClassUserPrompt = "user_prompt"

	// SecretClassRemoteFetch represents secrets fetched from a remote source
	// at runtime (e.g., secrets from a vault service).
	SecretClassRemoteFetch = "remote_fetch"
)

// validSecretClasses lists all accepted secret classifications.
// Empty string is allowed for backwards compatibility with unclassified secrets.
var validSecretClasses = map[string]bool{
	"":                             true, // unclassified (legacy)
	SecretClassInfrastructure:      true,
	SecretClassPerInstallGenerated: true,
	SecretClassUserPrompt:          true,
	SecretClassRemoteFetch:         true,
}

// IsValidSecretClass decides whether a secret class value is recognized.
// Returns true for valid classes, false for unknown classifications.
func IsValidSecretClass(class string) bool {
	return validSecretClasses[class]
}

// IsBundleSafeSecretClass decides whether a secret class can be included in bundles.
// Infrastructure secrets must NEVER be bundled as they pose security risks.
func IsBundleSafeSecretClass(class string) bool {
	return class != SecretClassInfrastructure
}

// Service type constants define the role of a service in a desktop bundle.
const (
	// ServiceTypeUIBundle is a frontend UI bundle served by the runtime.
	ServiceTypeUIBundle = "ui-bundle"

	// ServiceTypeAPIBinary is a backend API compiled binary.
	ServiceTypeAPIBinary = "api-binary"

	// ServiceTypeWorker is a background worker process.
	ServiceTypeWorker = "worker"

	// ServiceTypeResource is a bundled resource (database, cache, etc.).
	ServiceTypeResource = "resource"
)

// validServiceTypes lists all accepted service type values.
var validServiceTypes = map[string]bool{
	ServiceTypeUIBundle:  true,
	ServiceTypeAPIBinary: true,
	ServiceTypeWorker:    true,
	ServiceTypeResource:  true,
}

// IsValidServiceType decides whether a service type value is recognized.
func IsValidServiceType(serviceType string) bool {
	return validServiceTypes[serviceType]
}

// GetServiceTypeError returns a descriptive error for an invalid service type.
func GetServiceTypeError(serviceType string) error {
	if IsValidServiceType(serviceType) {
		return nil
	}
	return fmt.Errorf("type %q is not supported (valid: ui-bundle, api-binary, worker, resource)", serviceType)
}

// Secret target type constants define how secrets are injected into services.
const (
	// SecretTargetEnv injects the secret as an environment variable.
	SecretTargetEnv = "env"

	// SecretTargetFile writes the secret to a file at a specified path.
	SecretTargetFile = "file"
)

// validSecretTargetTypes lists accepted secret injection methods.
var validSecretTargetTypes = map[string]bool{
	SecretTargetEnv:  true,
	SecretTargetFile: true,
}

// IsValidSecretTargetType decides whether a secret target type is supported.
func IsValidSecretTargetType(targetType string) bool {
	return validSecretTargetTypes[targetType]
}

// Health check type constants define how service health is verified.
const (
	HealthCheckTypeHTTP    = "http"
	HealthCheckTypeTCP     = "tcp"
	HealthCheckTypeCommand = "command"
)

// Readiness check type constants define how service readiness is determined.
const (
	ReadinessTypeHealthSuccess = "health_success"
	ReadinessTypeLogMatch      = "log_match"
	ReadinessTypePortOpen      = "port_open"
)

// IPC mode constants define how the runtime control API is accessed.
const (
	IPCModeLoopbackHTTP = "loopback-http"
)

// IsValidIPCMode decides whether an IPC mode is supported.
// Currently only loopback-http is supported for security.
func IsValidIPCMode(mode string) bool {
	return mode == IPCModeLoopbackHTTP
}

// Schema version constants for bundle manifests.
const (
	BundleSchemaVersionV01 = "v0.1"
	BundleTargetDesktop    = "desktop"
)

// IsValidSchemaVersion decides whether a manifest schema version is supported.
func IsValidSchemaVersion(version string) bool {
	return version == BundleSchemaVersionV01
}

// IsValidBundleTarget decides whether a bundle target platform is supported.
func IsValidBundleTarget(target string) bool {
	return target == BundleTargetDesktop
}

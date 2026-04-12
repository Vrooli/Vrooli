// Package secrets provides safe access to project-local plaintext secrets files.
//
// The package centralizes:
//   - canonical project secrets path resolution
//   - environment vs file precedence for secret lookup
//   - local file trust validation (symlink, regular-file, permission checks)
//   - strict JSON parsing for secret keys
//   - atomic writes for plaintext secret stores
//   - narrow mutation helpers for project or explicit secrets files
//
// Path resolution prefers the repo contract when available. When callers provide
// an explicit root that is not contract-valid yet, or when discovery begins from
// a non-contract temp fixture, the package falls back to the conventional
// "<root>/.vrooli/secrets.json" shape so scenario and test migrations can land
// incrementally without re-encoding the old path logic at every call site.
//
// Plaintext secret documents may contain underscore-prefixed metadata entries
// such as "_metadata". Those keys are preserved as metadata and excluded from
// secret value resolution. All non-metadata secret entries must be JSON strings.
//
// Callers should make degraded-read policy explicit at their boundary.
// Best-effort runtime resolution can suppress local-file errors when appropriate,
// but administrative and configuration flows should surface invalid, insecure,
// or symlinked secret files as hard failures.
package secrets

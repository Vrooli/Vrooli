// Package secrets provides safe access to user-scoped plaintext secrets files.
//
// The package centralizes:
//   - canonical user secrets path resolution
//   - environment vs file precedence for secret lookup
//   - local file trust validation (symlink, regular-file, permission checks)
//   - strict JSON parsing for secret keys
//   - atomic writes for plaintext secret stores
//   - narrow mutation helpers for user or explicit secrets files
//
// The canonical shared location is "~/.vrooli/secrets.json". Callers may supply
// an explicit home directory for tests or controlled runtimes, but repo-local
// discovery and repo-local fallbacks are intentionally unsupported.
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

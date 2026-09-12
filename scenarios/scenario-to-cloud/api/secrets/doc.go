// Package secrets provides secret management for scenario-to-cloud workflows.
// It handles:
//   - fetching deployment secret manifests from secrets-manager
//   - validating and mutating local plaintext secrets files through api-core/secrets
//   - generating per-install secrets
//   - writing secrets to remote VPS targets
//
// Local project/scenario secret files and remote VPS secret files are separate
// boundaries. Local files should use api-core/secrets for path resolution and
// trust checks. Remote VPS management keeps its own transport-specific logic
// while preserving the same top-level JSON shape and metadata conventions.
package secrets

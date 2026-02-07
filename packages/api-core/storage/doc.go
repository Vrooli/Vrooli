// Package storage provides profile-aware runtime storage paths and safe filesystem helpers.
//
// It standardizes where scenarios should place mutable runtime state across environments:
//
//   - config: small durable configuration
//   - data: primary mutable application data
//   - cache: rebuildable/non-authoritative data
//   - logs: operational and diagnostic logs
//   - state: runtime state/checkpoints/lockfiles
//
// The resolver is intentionally policy-based: callers provide a scenario ID and optional
// profile/override hints, and storage resolves absolute paths without exposing OS-specific
// branch logic at call sites.
//
// Package boundaries:
//
//   - Resolver: path policy and deterministic path construction
//   - Filesystem helpers: directory creation + atomic file writes
//   - Error model: structured errors for caller-facing classification
//
// This package does not perform migration/copy logic and does not manage retention policies.
// Those concerns remain in scenario/application orchestration layers.
package storage

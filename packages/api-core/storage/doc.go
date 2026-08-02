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
//   - Namespace helpers: variant-aware Redis/Qdrant namespace composition
//   - Error model: structured errors for caller-facing classification
//
// This package does not perform migration/copy logic.
//
// # Retention
//
// Retention is owned by api-core/retention, not by scenario orchestration code
// and not by this package. A scenario declares storage ceilings in its manifest
// and the framework enforces them; declaring "pruner": "builtin" needs no Go
// code at all. Budget target paths resolve through the class roots above, so a
// shadow variant prunes its own data and never live's.
//
// The reason this is framework-owned rather than per-scenario: a correctly
// configured 30-day retention policy freed nothing while its database grew to
// 453 GB, because every row was 17 days old. An age bound alone is a promise
// proportional to an ingest rate the scenario usually does not control. See
// docs/reference/storage-retention.md and the api-core/retention package
// documentation.
//
// # Variant-aware storage namespaces (shadow isolation)
//
// A scenario can run as more than one instance — the canonical "live" instance
// plus zero or more shadow variants that Baseline Modes stands up so a scenario
// can be developed and validated while its last-known-good version keeps
// serving. For that to be safe, EVERY store a scenario writes must be scoped to
// its variant. The filesystem classes above isolate automatically because the
// lifecycle hands each variant its own data/config/cache roots; Postgres and
// SQLite isolate because the lifecycle injects a variant-aware POSTGRES_DB /
// SQLITE_PATH (see the database package). Redis and Qdrant have no such built-in
// scoping, so scenarios MUST compose their key prefixes and collection names
// through the helpers in namespace.go:
//
//	collection, _ := storage.Collection("backlog")     // "swarm-manager_backlog" live,
//	                                                    // "swarm-manager_shadow_backlog" shadow
//	key, _ := storage.RedisKey("idea", id, "research")  // "swarm-manager:idea:<id>:research"
//
// These read the lifecycle-injected VROOLI_STORAGE_NAMESPACE root, which is the
// single source of truth (scenarioruntime.InstanceKey.Namespace()). HARDCODING
// the scenario name into a collection or key prefix bypasses shadow isolation —
// a shadow would read and write live's state — and is exactly the pattern
// storage-steer / test-genie flag as a finding. Always go through the helpers.
package storage

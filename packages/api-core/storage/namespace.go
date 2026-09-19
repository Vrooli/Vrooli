package storage

import (
	"os"
	"strconv"
	"strings"
)

// Variant-aware storage namespace helpers (Baseline Modes, plan P5).
//
// A scenario can run as more than one instance — the canonical "live" instance
// plus zero or more shadow variants (e.g. "shadow") that Baseline Modes stands
// up so a scenario can be developed and validated while its last-known-good
// version keeps serving. For that to be safe, each variant must read and write
// its OWN Redis keys and Qdrant collections; a shadow that wrote into live's
// namespace would corrupt the very state the engagement is protecting.
//
// The single source of truth for the namespace root is
// scenarioruntime.InstanceKey.Namespace().StorageNamespace ("<scenario>" for
// live, "<scenario>_<variant>" otherwise). That root crosses the module
// boundary into a scenario process as the VROOLI_STORAGE_NAMESPACE environment
// variable, injected by the lifecycle. This package is a pure CONSUMER of that
// pre-composed root: it appends the per-domain suffix and never re-derives the
// scenario+variant composition itself (doing so would create a second, drifting
// SSOT). A scenario therefore gets shadow isolation for free by calling these
// helpers instead of hardcoding its slug:
//
//	collection, _ := storage.Collection("backlog") // "swarm-manager_backlog" (live)
//	                                                // "swarm-manager_shadow_backlog" (shadow)
//	prefix, _ := storage.RedisPrefix("idea")        // "swarm-manager:idea:"
//	key, _ := storage.RedisKey("idea", id, "research") // "swarm-manager:idea:<id>:research"
//
// Filesystem state (SQLite files, the data/blob dirs) isolates the same way:
// pass the variant-aware identity to the path resolver via ScenarioNamespace,
//
//	scenarioID, _ := storage.ScenarioNamespace("swarm-manager") // "swarm-manager" live,
//	                                                            // "swarm-manager_shadow" shadow
//	path, _ := resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassData, "db.sqlite")
//
// Hardcoding the scenario name in a collection, key prefix, or path scope
// BYPASSES shadow isolation and is exactly what storage-steer / test-genie flag
// as a finding.
const (
	// EnvStorageNamespace carries the variant-aware namespace ROOT injected by
	// the lifecycle from scenarioruntime.InstanceKey.Namespace().StorageNamespace
	// — "<scenario>" for the live instance, "<scenario>_<variant>" otherwise. It
	// is the single source of truth; this package reads it verbatim and never
	// re-composes it.
	EnvStorageNamespace = "VROOLI_STORAGE_NAMESPACE"
	// EnvVariant carries the normalized variant ("live", "shadow", ...). It is
	// advisory here (the root already folds the variant in) and is used only to
	// (a) populate the Variant()/IsLive() accessors and (b) detect an
	// inconsistent environment (non-live variant with no namespace root).
	EnvVariant = "VROOLI_VARIANT"
	// EnvScenario carries the bare scenario slug. It is the live-only fallback
	// root used when EnvStorageNamespace is unset (e.g. a process started outside
	// the variant-aware lifecycle, or a test).
	EnvScenario = "VROOLI_SCENARIO"
)

// defaultVariant mirrors scenarioruntime.DefaultVariant. The canonical primary
// instance is "live"; an empty variant normalizes to it.
const defaultVariant = "live"

// Separators between the namespace root, the domain, and any key segments.
// Qdrant collections join with "_" (matching the "<scenario>_<variant>" root
// composition); Redis keys join with ":" (the conventional Redis hierarchy
// delimiter). One canonical separator per engine, chosen once here.
const (
	collectionSep = "_"
	redisSep      = ":"
)

// Namespace composes variant-aware storage names from a resolved namespace root.
// Construct it with ResolveNamespace (reads the injected env) or, for tests and
// explicit callers, by passing NamespaceConfig.Root directly.
//
// The zero value is not usable; always go through ResolveNamespace.
type Namespace struct {
	root    string
	variant string
}

// NamespaceConfig configures namespace resolution and provides test seams.
type NamespaceConfig struct {
	// EnvGet reads environment variables. Defaults to os.Getenv.
	EnvGet func(key string) string
	// Root forces the namespace root, bypassing the environment entirely. This
	// is the explicit/test seam; production code leaves it empty and lets the
	// lifecycle-injected VROOLI_STORAGE_NAMESPACE drive resolution.
	Root string
	// Variant forces the variant label for the Variant()/IsLive() accessors when
	// Root is set. Ignored when Root is empty (the env supplies the variant).
	Variant string
	// FallbackScenario is the compile-time scenario slug used as the LIVE-only
	// fallback root when neither Root nor the injected VROOLI_STORAGE_NAMESPACE /
	// VROOLI_SCENARIO env is present — e.g. a binary run outside the variant-aware
	// lifecycle (local `go run`, a unit test). It NEVER overrides an injected
	// root, and a non-live VROOLI_VARIANT with no injected root is still a hard
	// error, so the fallback can never alias a shadow onto live. Lifecycle-driven
	// code leaves this empty; a scenario passes its own slug (the template's
	// "{{SCENARIO_ID}}") only as the no-env safety net. See ScenarioNamespace.
	FallbackScenario string
}

// ResolveNamespace resolves the variant-aware namespace from configuration and
// the environment.
//
// Resolution order:
//  1. NamespaceConfig.Root (explicit/test seam) — used verbatim.
//  2. VROOLI_STORAGE_NAMESPACE — the lifecycle-injected, pre-composed root.
//  3. VROOLI_SCENARIO — a live-only fallback root (bare scenario slug).
//
// Fail-loud safety: if VROOLI_VARIANT advertises a non-live variant but no
// namespace root was injected (steps 1 and 2 both empty), resolution returns an
// error rather than silently falling back to the bare scenario slug — writing a
// shadow's data into the live namespace is the one failure this whole mechanism
// exists to prevent.
func ResolveNamespace(cfg NamespaceConfig) (Namespace, error) {
	get := cfg.EnvGet
	if get == nil {
		get = os.Getenv
	}

	root := strings.TrimSpace(cfg.Root)
	variant := normalizeVariant(cfg.Variant)

	if root != "" {
		if !isValidNamespaceRoot(root) {
			return Namespace{}, &Error{Kind: ErrInvalidInput, Message: "invalid storage namespace root", Details: root}
		}
		return Namespace{root: root, variant: variant}, nil
	}

	envVariant := normalizeVariant(get(EnvVariant))

	if root = strings.TrimSpace(get(EnvStorageNamespace)); root != "" {
		if !isValidNamespaceRoot(root) {
			return Namespace{}, &Error{Kind: ErrInvalidInput, Message: "invalid " + EnvStorageNamespace, Details: root}
		}
		return Namespace{root: root, variant: envVariant}, nil
	}

	// No namespace root injected. Fall back to the bare scenario slug (the env
	// var, else the caller-supplied compile-time slug), but only for the live
	// variant — a non-live variant here means the lifecycle did not inject the
	// root, and proceeding would alias shadow state onto live.
	scenario := strings.TrimSpace(get(EnvScenario))
	if scenario == "" {
		scenario = strings.TrimSpace(cfg.FallbackScenario)
	}
	if scenario == "" {
		return Namespace{}, &Error{
			Kind:    ErrInvalidInput,
			Message: "no storage namespace available: set " + EnvStorageNamespace + " (injected by the lifecycle), NamespaceConfig.Root, or NamespaceConfig.FallbackScenario",
		}
	}
	if envVariant != "" && envVariant != defaultVariant {
		return Namespace{}, &Error{
			Kind:    ErrInvalidInput,
			Message: "inconsistent storage environment: " + EnvVariant + " is non-live but " + EnvStorageNamespace + " is unset",
			Details: envVariant,
		}
	}
	if !isValidNamespaceRoot(scenario) {
		return Namespace{}, &Error{Kind: ErrInvalidInput, Message: "invalid " + EnvScenario, Details: scenario}
	}
	return Namespace{root: scenario, variant: defaultVariant}, nil
}

// Root returns the resolved namespace root ("<scenario>" or "<scenario>_<variant>").
func (n Namespace) Root() string { return n.root }

// Variant returns the normalized variant ("live" when unknown/unset).
func (n Namespace) Variant() string {
	if n.variant == "" {
		return defaultVariant
	}
	return n.variant
}

// IsLive reports whether this namespace addresses the canonical live instance.
func (n Namespace) IsLive() bool { return n.Variant() == defaultVariant }

// Collection composes a variant-aware Qdrant collection name for domain:
// "<root>_<domain>". For live swarm-manager and domain "backlog" this is
// "swarm-manager_backlog"; under the shadow variant it is
// "swarm-manager_shadow_backlog", so the two instances never share a collection.
func (n Namespace) Collection(domain string) (string, error) {
	d, err := cleanDomain(domain)
	if err != nil {
		return "", err
	}
	return n.root + collectionSep + d, nil
}

// RedisPrefix composes a variant-aware Redis key prefix for domain, terminated
// with the separator so callers can append their own tokens:
// "<root>:<domain>:". Prefer RedisKey when the full key is known.
func (n Namespace) RedisPrefix(domain string) (string, error) {
	d, err := cleanDomain(domain)
	if err != nil {
		return "", err
	}
	return n.root + redisSep + d + redisSep, nil
}

// RedisKey composes a full variant-aware Redis key:
// "<root>:<domain>:seg1:seg2:...". Segments are joined verbatim with ":", so a
// caller reproduces a key like "swarm-manager:idea:<id>:research" with
// RedisKey("idea", id, "research"). This is why a flat prefix is insufficient:
// real keys interleave dynamic tokens mid-string.
//
// Each segment must be non-empty after trimming; empty segments are rejected so
// a missing dynamic token can never silently collapse the key hierarchy.
func (n Namespace) RedisKey(domain string, segments ...string) (string, error) {
	d, err := cleanDomain(domain)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(segments)+2)
	parts = append(parts, n.root, d)
	for i, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return "", &Error{Kind: ErrInvalidInput, Message: "redis key segment must not be empty", Details: "segment " + strconv.Itoa(i)}
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, redisSep), nil
}

// Collection resolves the process namespace from the environment and composes a
// variant-aware Qdrant collection name. It is the zero-config adoption seam: a
// scenario launched by the lifecycle gets shadow isolation simply by calling
// storage.Collection(domain) instead of hardcoding "<scenario>_<domain>".
func Collection(domain string) (string, error) {
	ns, err := ResolveNamespace(NamespaceConfig{})
	if err != nil {
		return "", err
	}
	return ns.Collection(domain)
}

// RedisPrefix resolves the process namespace from the environment and composes a
// variant-aware Redis key prefix. See Namespace.RedisPrefix.
func RedisPrefix(domain string) (string, error) {
	ns, err := ResolveNamespace(NamespaceConfig{})
	if err != nil {
		return "", err
	}
	return ns.RedisPrefix(domain)
}

// RedisKey resolves the process namespace from the environment and composes a
// full variant-aware Redis key. See Namespace.RedisKey.
func RedisKey(domain string, segments ...string) (string, error) {
	ns, err := ResolveNamespace(NamespaceConfig{})
	if err != nil {
		return "", err
	}
	return ns.RedisKey(domain, segments...)
}

// ScenarioNamespace resolves the variant-aware scenario identity for FILESYSTEM
// path scoping — SQLite database files and the data/blob directories — the path
// analogue of Collection/RedisKey for Redis/Qdrant. It returns the namespace
// ROOT ("<scenario>" for live, "<scenario>_<variant>" for a shadow), which a
// scenario passes as storage.Options.ScenarioID so the resolver scopes its
// on-disk state to "<class-root>/<app>/<root>/" and live and shadow never share
// a SQLite file or data dir.
//
// The lifecycle injects VROOLI_STORAGE_NAMESPACE for every instance, so a
// lifecycle-launched scenario isolates automatically. fallback is the
// compile-time scenario slug, used only when the process runs OUTSIDE the
// variant-aware lifecycle (no namespace env injected) so local `go run` and
// tests keep working; live resolves to fallback verbatim, leaving existing
// on-disk paths unchanged. A non-live VROOLI_VARIANT with no injected root is
// still a hard error — the fail-loud guard that prevents a shadow from writing
// into live's filesystem state.
func ScenarioNamespace(fallback string) (string, error) {
	ns, err := ResolveNamespace(NamespaceConfig{FallbackScenario: fallback})
	if err != nil {
		return "", err
	}
	return ns.Root(), nil
}

func normalizeVariant(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// isValidNamespaceRoot accepts the characters that appear in a composed root:
// scenario slugs (lower-case kebab-case) plus the "_" the variant join uses.
// It reuses the scenario-id grammar (alnum plus '-', '_', '.').
func isValidNamespaceRoot(s string) bool { return isValidScenarioID(s) }

// cleanDomain validates and normalizes a storage domain. Domains are simple
// identifiers (e.g. "backlog", "idea", "records"); they must not contain
// whitespace, the Redis delimiter ":", or path separators that would let a
// domain escape its namespace.
func cleanDomain(domain string) (string, error) {
	d := strings.TrimSpace(domain)
	if d == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "storage domain must not be empty"}
	}
	for _, r := range d {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return "", &Error{Kind: ErrInvalidInput, Message: "invalid storage domain", Details: domain}
		}
	}
	return d, nil
}

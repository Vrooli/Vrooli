package scenarioruntime

import (
	"fmt"
	"strings"
)

// DefaultVariant is the canonical variant of the always-present primary
// instance of a scenario. An empty variant normalizes to it, so every existing
// call site that passes only a scenario name (variant == "") is a
// backward-compatible no-op: it continues to address the live instance.
const DefaultVariant = "live"

// Environment variables injected into a scenario process to advertise the
// variant it is running as. Scenarios MUST derive their storage namespaces from
// these (via the api-core storage helpers) instead of hardcoding their own slug
// — hardcoding bypasses shadow isolation. See the Baseline Modes plan, P5.
const (
	// EnvVariant carries the normalized variant ("live", "shadow", ...).
	EnvVariant = "VROOLI_VARIANT"
	// EnvStorageNamespace carries the Redis/Qdrant namespace root
	// ("<scenario>" for live, "<scenario>_<variant>" otherwise) that the
	// variant-aware storage helpers compose domain names from.
	EnvStorageNamespace = "VROOLI_STORAGE_NAMESPACE"
)

// InstanceKey identifies one running instance of a scenario by (scenario,
// variant). It is the single source of truth for instance addressing AND for
// every storage namespace derived from that addressing — record/log directory,
// advisory lock file, Postgres database name, data directory, deterministic
// port seed, and the Redis/Qdrant namespace root. No other layer concatenates
// these strings itself; they all flow from InstanceKey.Namespace(). See §1a of
// the Baseline Modes plan ("One naming SSOT — addressing AND storage").
//
// The zero value ({Scenario: "x"}) addresses the live instance, which keeps the
// pre-variant world unchanged: variant "" ⇒ "live", and the live forms are
// byte-identical to the legacy slug-keyed forms.
type InstanceKey struct {
	Scenario string
	Variant  string
}

// Normalize trims whitespace, lower-cases the variant for case-insensitive
// matching, and collapses an empty variant to DefaultVariant. It is idempotent.
// The scenario name is trimmed but otherwise preserved (scenario slugs are
// already canonical kebab-case identifiers).
func (k InstanceKey) Normalize() InstanceKey {
	scenario := strings.TrimSpace(k.Scenario)
	variant := strings.ToLower(strings.TrimSpace(k.Variant))
	if variant == "" {
		variant = DefaultVariant
	}
	return InstanceKey{Scenario: scenario, Variant: variant}
}

// IsLive reports whether this key addresses the canonical (live) instance.
func (k InstanceKey) IsLive() bool {
	return k.Normalize().Variant == DefaultVariant
}

// Slug returns the canonical, round-trip-symmetric identifier:
//   - "scenario"          for the live instance (unchanged from the legacy slug)
//   - "scenario@variant"  for any non-live instance
//
// Slug is BOTH the canonical input form parsed by ParseInstanceKey and the
// output form rendered to humans, so ParseInstanceKey(k.Slug(), "") == k.
// "@" is filename-safe on ext4 and Windows, so Slug is used directly for record
// and log directory names and advisory lock files.
func (k InstanceKey) Slug() string {
	n := k.Normalize()
	if n.Variant == DefaultVariant {
		return n.Scenario
	}
	return n.Scenario + "@" + n.Variant
}

// String implements fmt.Stringer and returns Slug.
func (k InstanceKey) String() string {
	return k.Slug()
}

// Namespace holds every addressing and storage name derived from an InstanceKey.
// It is the single source of truth — callers read these fields, they never
// re-derive the strings. Engines whose names forbid "@" (Postgres, Redis,
// Qdrant) receive the "@→_" / "-→_" mapped forms here; the filesystem-safe
// forms (record dir, data dir) keep "@".
type Namespace struct {
	// Variant is the normalized variant ("live", "shadow", ...).
	Variant string
	// RecordSlug is the on-disk record/log directory name and advisory-lock
	// base: "scenario" for live, "scenario@variant" otherwise.
	RecordSlug string
	// PostgresDB is the Postgres database name: "vrooli_<scenario>" for live,
	// "vrooli_<scenario>_<variant>" otherwise ("-" and "@" mapped to "_").
	PostgresDB string
	// DataDirName is the per-variant filesystem data subdirectory name. "@" is
	// filesystem-safe, so this equals RecordSlug.
	DataDirName string
	// PortSeed is the prefix fed to the CRC32 port-allocation seed. It equals
	// RecordSlug, so live keeps its existing deterministic ports and non-live
	// variants hash to a different first-choice port.
	PortSeed string
	// StorageNamespace is the Redis/Qdrant namespace root: "<scenario>" for
	// live, "<scenario>_<variant>" otherwise. The variant-aware storage helpers
	// (P5) compose per-domain names from it (e.g. "<root>_<domain>" for a Qdrant
	// collection, "<root>:<domain>:..." for a Redis key).
	StorageNamespace string
	// EnvVars are injected into the scenario process so it can resolve its own
	// variant-aware namespaces from the environment instead of hardcoding a slug.
	EnvVars map[string]string
}

// Namespace derives the full set of addressing and storage names for this key.
// This is the SSOT referenced throughout the Baseline Modes plan: the lifecycle
// reads RecordSlug/PostgresDB/DataDirName/PortSeed and injects EnvVars; the
// api-core storage helpers (P5) read StorageNamespace from the injected env.
func (k InstanceKey) Namespace() Namespace {
	n := k.Normalize()
	live := n.Variant == DefaultVariant

	recordSlug := n.Slug()

	pgScenario := strings.ReplaceAll(n.Scenario, "-", "_")
	postgresDB := "vrooli_" + pgScenario
	storageRoot := n.Scenario
	if !live {
		pgVariant := strings.ReplaceAll(n.Variant, "-", "_")
		postgresDB += "_" + pgVariant
		storageRoot += "_" + n.Variant
	}

	return Namespace{
		Variant:          n.Variant,
		RecordSlug:       recordSlug,
		PostgresDB:       postgresDB,
		DataDirName:      recordSlug,
		PortSeed:         recordSlug,
		StorageNamespace: storageRoot,
		EnvVars: map[string]string{
			EnvVariant:          n.Variant,
			EnvStorageNamespace: storageRoot,
		},
	}
}

// ParseInstanceKey resolves a scenario argument and an optional --instance flag
// value into a single InstanceKey. It is the one shared name-resolver that sits
// beneath every command surface (the cli-core global flag, bare-name resolution,
// `vrooli scenario port`, api-core discovery, test-genie targetruntime), so no
// layer re-implements the "@" split.
//
// Rules (§1a "name@variant is the canonical identifier; --instance is sugar"):
//   - nameArg is split on its LAST "@"; the part after it is the suffix variant.
//   - If both the suffix and flagVariant are supplied and DISAGREE, it is a hard
//     error (never silent precedence).
//   - If they agree, or only one is supplied, that variant is used.
//   - If neither is supplied, the variant is "" ⇒ Normalize ⇒ live.
//
// So `scenario port swarm-manager --instance shadow` and
// `scenario port swarm-manager@shadow` resolve identically.
func ParseInstanceKey(nameArg, flagVariant string) (InstanceKey, error) {
	scenario := strings.TrimSpace(nameArg)
	suffixVariant := ""
	if idx := strings.LastIndex(scenario, "@"); idx >= 0 {
		suffixVariant = strings.TrimSpace(scenario[idx+1:])
		scenario = strings.TrimSpace(scenario[:idx])
	}
	flagVariant = strings.TrimSpace(flagVariant)

	if scenario == "" {
		return InstanceKey{}, fmt.Errorf("instance key: missing scenario name in %q", nameArg)
	}

	variant := ""
	switch {
	case suffixVariant != "" && flagVariant != "":
		if !strings.EqualFold(suffixVariant, flagVariant) {
			return InstanceKey{}, fmt.Errorf(
				"instance key: --instance %q disagrees with %q in %q (drop one)",
				flagVariant, "@"+suffixVariant, nameArg)
		}
		variant = suffixVariant
	case suffixVariant != "":
		variant = suffixVariant
	default:
		variant = flagVariant
	}

	return InstanceKey{Scenario: scenario, Variant: variant}.Normalize(), nil
}

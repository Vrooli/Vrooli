// Package config is the cartographer-global control surface: the small,
// bounded, named set of tunable levers that govern derivation and the
// advisory heuristics. There is NO per-scenario configuration — cartographer
// derives a scenario's domain map and scores its boundaries with zero
// per-scenario setup; these levers tune cartographer itself.
//
// Every lever has a sane default that keeps the day-one behavior (and the
// test suite) green. Out-of-range or unparseable values are clamped/ignored
// with a diagnostic rather than failing startup. The canonical documentation
// lives in docs/reference/configuration.md; keep the two in lockstep.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Envelope of env-var keys. All levers share the CARTOGRAPHER_ prefix,
// matching the existing project-dir overrides in main.go.
const (
	EnvGodDomainFanOut     = "CARTOGRAPHER_GOD_DOMAIN_FANOUT"
	EnvInstabilityWarnBand = "CARTOGRAPHER_INSTABILITY_WARN_BAND"
	EnvAutoPlaceMin        = "CARTOGRAPHER_AUTO_PLACE_MIN"
	EnvSuggestMin          = "CARTOGRAPHER_SUGGEST_MIN"
	EnvTieDelta            = "CARTOGRAPHER_TIE_DELTA"
	EnvQuorumHigh          = "CARTOGRAPHER_QUORUM_HIGH"
	EnvQuorumLow           = "CARTOGRAPHER_QUORUM_LOW"
	EnvArchetypeExemptions = "CARTOGRAPHER_ARCHETYPE_EXEMPTIONS"
	EnvNonDomainFolders    = "CARTOGRAPHER_NON_DOMAIN_FOLDERS"
	EnvLadderOrder         = "CARTOGRAPHER_LADDER_ORDER"
	EnvBannedVocab         = "CARTOGRAPHER_BANNED_VOCAB"
	EnvLayeringStrict      = "CARTOGRAPHER_LAYERING_STRICT"
)

// Config is the resolved control surface.
type Config struct {
	// GodDomainFanOut is the efferent fan-out fraction at/above which a
	// non-exempt domain earns a god_domain smell. Range (0,1].
	GodDomainFanOut float64
	// InstabilityWarnBand is the instability at/above which a depended-upon
	// domain earns an unstable_dependency smell. Range [0,1].
	InstabilityWarnBand float64
	// AutoPlaceMin / SuggestMin are the aggregator tier thresholds. Range
	// [0,1]; AutoPlaceMin must be >= SuggestMin.
	AutoPlaceMin float64
	SuggestMin   float64
	// QuorumHigh / QuorumLow are the participation thresholds required before
	// a directional score may become auto_place or suggest. Range [0,1];
	// QuorumHigh must be >= QuorumLow.
	QuorumHigh float64
	QuorumLow  float64
	// TieDelta is the minimum gap between top and runner-up before a verdict
	// is a tie (→ conflict). Range [0,1].
	TieDelta float64
	// ArchetypeExemptions are archetypes exempt from the god_domain smell.
	ArchetypeExemptions []string
	// ExtraNonDomainFolders extends the built-in api/internal infrastructure
	// exemption set used by the folder extractor.
	ExtraNonDomainFolders []string
	// LadderOrder is the domain-extraction trust order (highest first), by
	// source name. UI features are always advisory and appended regardless.
	LadderOrder []string
	// BannedVocabulary names generic package/domain terms the naming detector
	// treats as intent-hiding architecture drift.
	BannedVocabulary []string
	// LayeringStrict promotes blocker-eligible wrong-direction dependencies to
	// blocker severity. When false, they remain error severity.
	LayeringStrict bool
}

// Diagnostic is a non-fatal config-resolution finding.
type Diagnostic struct {
	Key     string
	Message string
}

// Default returns the canonical defaults. These MUST match the hardcoded
// day-one values (boundaries.DefaultConfig + signals aggregator constants)
// so omitting all env vars reproduces the original behavior.
func Default() Config {
	return Config{
		GodDomainFanOut:       0.6,
		InstabilityWarnBand:   0.7,
		AutoPlaceMin:          0.85,
		SuggestMin:            0.55,
		QuorumHigh:            0.45,
		QuorumLow:             0.30,
		TieDelta:              0.10,
		ArchetypeExemptions:   []string{"composition-root", "infrastructure"},
		ExtraNonDomainFolders: nil,
		LadderOrder:           []string{"domains_doc", "api_folders", "cli_groups"},
		BannedVocabulary:      []string{"common", "helpers", "manager", "misc", "stuff", "utils"},
		LayeringStrict:        true,
	}
}

// validLadderSources is the set of non-advisory ladder source names a user
// may order. UI features are advisory and not orderable here.
var validLadderSources = map[string]struct{}{
	"api_manifest": {}, "domains_doc": {}, "api_folders": {}, "cli_groups": {},
}

// Load resolves the config from getenv (pass os.Getenv in production),
// clamping out-of-range values and ignoring unparseable ones, each with a
// diagnostic. It never returns an error — a misconfigured lever degrades to
// its default, it does not break startup.
func Load(getenv func(string) string) (Config, []Diagnostic) {
	cfg := Default()
	var diags []Diagnostic

	cfg.GodDomainFanOut = floatLever(getenv, EnvGodDomainFanOut, cfg.GodDomainFanOut, 0, 1, false, &diags)
	cfg.InstabilityWarnBand = floatLever(getenv, EnvInstabilityWarnBand, cfg.InstabilityWarnBand, 0, 1, true, &diags)
	cfg.AutoPlaceMin = floatLever(getenv, EnvAutoPlaceMin, cfg.AutoPlaceMin, 0, 1, true, &diags)
	cfg.SuggestMin = floatLever(getenv, EnvSuggestMin, cfg.SuggestMin, 0, 1, true, &diags)
	cfg.QuorumHigh = floatLever(getenv, EnvQuorumHigh, cfg.QuorumHigh, 0, 1, true, &diags)
	cfg.QuorumLow = floatLever(getenv, EnvQuorumLow, cfg.QuorumLow, 0, 1, true, &diags)
	cfg.TieDelta = floatLever(getenv, EnvTieDelta, cfg.TieDelta, 0, 1, true, &diags)

	if cfg.AutoPlaceMin < cfg.SuggestMin {
		diags = append(diags, Diagnostic{Key: EnvAutoPlaceMin, Message: fmt.Sprintf("auto_place_min (%.2f) < suggest_min (%.2f); raising auto_place_min to suggest_min", cfg.AutoPlaceMin, cfg.SuggestMin)})
		cfg.AutoPlaceMin = cfg.SuggestMin
	}
	if cfg.QuorumHigh < cfg.QuorumLow {
		diags = append(diags, Diagnostic{Key: EnvQuorumHigh, Message: fmt.Sprintf("quorum_high (%.2f) < quorum_low (%.2f); raising quorum_high to quorum_low", cfg.QuorumHigh, cfg.QuorumLow)})
		cfg.QuorumHigh = cfg.QuorumLow
	}

	if v := strings.TrimSpace(getenv(EnvArchetypeExemptions)); v != "" {
		cfg.ArchetypeExemptions = splitCSV(v)
	}
	if v := strings.TrimSpace(getenv(EnvNonDomainFolders)); v != "" {
		cfg.ExtraNonDomainFolders = splitCSV(v)
	}
	if v := strings.TrimSpace(getenv(EnvLadderOrder)); v != "" {
		order := splitCSV(v)
		valid := order[:0]
		for _, s := range order {
			if _, ok := validLadderSources[s]; ok {
				valid = append(valid, s)
			} else {
				diags = append(diags, Diagnostic{Key: EnvLadderOrder, Message: fmt.Sprintf("unknown ladder source %q ignored (valid: api_manifest, domains_doc, api_folders, cli_groups)", s)})
			}
		}
		if len(valid) > 0 {
			cfg.LadderOrder = valid
		} else {
			diags = append(diags, Diagnostic{Key: EnvLadderOrder, Message: "no valid ladder sources; keeping default order"})
		}
	}
	if v := strings.TrimSpace(getenv(EnvBannedVocab)); v != "" {
		cfg.BannedVocabulary = splitCSV(v)
		if len(cfg.BannedVocabulary) == 0 {
			diags = append(diags, Diagnostic{Key: EnvBannedVocab, Message: "no valid banned vocabulary terms; keeping default list"})
			cfg.BannedVocabulary = Default().BannedVocabulary
		}
	}
	cfg.LayeringStrict = boolLever(getenv, EnvLayeringStrict, cfg.LayeringStrict, &diags)

	return cfg, diags
}

func floatLever(getenv func(string) string, key string, def, lo, hi float64, allowLo bool, diags *[]Diagnostic) float64 {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%q is not a number; using default %.2f", raw, def)})
		return def
	}
	min := lo
	if !allowLo {
		// exclusive lower bound: nudge just above lo for clamping intent
		if v <= lo {
			*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%.4f must be > %.2f; using default %.2f", v, lo, def)})
			return def
		}
	}
	if v < min || v > hi {
		clamped := v
		if v < min {
			clamped = min
		}
		if v > hi {
			clamped = hi
		}
		*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%.4f out of range [%.2f,%.2f]; clamped to %.2f", v, lo, hi, clamped)})
		return clamped
	}
	return v
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func boolLever(getenv func(string) string, key string, def bool, diags *[]Diagnostic) bool {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%q is not a boolean; using default %t", raw, def)})
		return def
	}
}

func intLever(getenv func(string) string, key string, def, lo, hi int, diags *[]Diagnostic) int {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%q is not an integer; using default %d", raw, def)})
		return def
	}
	if v < lo || v > hi {
		clamped := v
		if v < lo {
			clamped = lo
		}
		if v > hi {
			clamped = hi
		}
		*diags = append(*diags, Diagnostic{Key: key, Message: fmt.Sprintf("%d out of range [%d,%d]; clamped to %d", v, lo, hi, clamped)})
		return clamped
	}
	return v
}

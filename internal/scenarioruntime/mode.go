package scenarioruntime

import (
	"fmt"
	"os"
	"strings"
)

const (
	ModeEnv = "VROOLI_RUNTIME_REGISTRY"
	// AllowlistEnv scopes non-off registry migration modes to selected
	// scenarios during soak. Empty means the selected mode applies globally.
	AllowlistEnv = "VROOLI_RUNTIME_REGISTRY_ALLOWLIST"

	ModeOff    = "off"
	ModeDual   = "dual"
	ModePrefer = "prefer"
	ModeStrict = "strict"
)

func ModeFromEnv() (string, error) {
	return ModeFromString(os.Getenv(ModeEnv))
}

func ModeFromString(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return ModeOff, nil
	}
	switch mode {
	case ModeOff, ModeDual, ModePrefer, ModeStrict:
		return mode, nil
	default:
		return "", fmt.Errorf("%s must be one of off, dual, prefer, strict; got %q", ModeEnv, raw)
	}
}

func WriteEnabled(mode string) bool {
	return mode == ModeDual || mode == ModePrefer || mode == ModeStrict
}

func ReadEnabled(mode string) bool {
	return mode == ModePrefer || mode == ModeStrict
}

func StrictReads(mode string) bool {
	return mode == ModeStrict
}

func AllowlistFromString(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	}) {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out
}

func HasAllowlist(raw string) bool {
	return len(AllowlistFromString(raw)) > 0
}

func HasAllowlistByEnv() bool {
	return HasAllowlist(os.Getenv(AllowlistEnv))
}

func ScenarioAllowed(rawAllowlist, scenarioName string) bool {
	allowlist := AllowlistFromString(rawAllowlist)
	if len(allowlist) == 0 {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(scenarioName))
	if name == "" {
		return false
	}
	_, ok := allowlist[name]
	return ok
}

func ScenarioAllowedByEnv(scenarioName string) bool {
	return ScenarioAllowed(os.Getenv(AllowlistEnv), scenarioName)
}

func WriteEnabledForScenario(mode, scenarioName string) bool {
	return WriteEnabled(mode) && ScenarioAllowedByEnv(scenarioName)
}

func ReadEnabledForScenario(mode, scenarioName string) bool {
	return ReadEnabled(mode) && ScenarioAllowedByEnv(scenarioName)
}

func StrictReadsForScenario(mode, scenarioName string) bool {
	return StrictReads(mode) && ScenarioAllowedByEnv(scenarioName)
}

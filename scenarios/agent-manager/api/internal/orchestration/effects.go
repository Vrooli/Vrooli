package orchestration

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// validateEffectGrant is the Agent Manager owner-side backstop for grants
// received over the workflow transport. Swarm owns approval; Agent Manager
// owns refusing malformed or unknown effects before a child run is persisted.
func validateEffectGrant(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		identity, params, err := parseEffectGrant(raw)
		if err != nil {
			return err
		}
		if _, exists := seen[raw]; exists {
			return errors.New("effect grant contains a duplicate entry")
		}
		seen[raw] = struct{}{}
		switch identity {
		case "filesystem.write":
			if err := validateRelativeEffectPath(params["paths"]); err != nil {
				return fmt.Errorf("filesystem.write: %w", err)
			}
		case "process.test", "network.internal":
			if !canonicalSlug(params["scope"]) && !canonicalScenarioPath(params["scope"]) {
				return fmt.Errorf("%s scope must be a canonical scenario slug or scenarios/<slug>", identity)
			}
		case "network.external":
			if strings.TrimSpace(params["hosts"]) == "" || strings.ContainsAny(params["hosts"], "\\/\n\r") {
				return errors.New("network.external hosts must be an explicit host allowlist")
			}
		case "credentials.use":
			if !canonicalSlug(params["account"]) || !canonicalSlug(params["name"]) {
				return errors.New("credentials.use requires canonical account and name")
			}
		case "billing.reserve":
			if !canonicalSlug(params["account"]) || (params["mode"] != "simulation" && params["mode"] != "live") {
				return errors.New("billing.reserve requires a canonical account and simulation/live mode")
			}
		case "publication.write":
			if !canonicalSlug(params["target"]) {
				return errors.New("publication.write target must be canonical")
			}
		default:
			return fmt.Errorf("effect identity %q has no Agent Manager enforcement owner", identity)
		}
		if err := validateEffectParameters(identity, params); err != nil {
			return err
		}
	}
	return nil
}

func validateEffectParameters(identity string, params map[string]string) error {
	required := map[string]map[string]struct{}{
		"filesystem.write":  {"paths": {}},
		"process.test":      {"scope": {}},
		"network.internal":  {"scope": {}},
		"network.external":  {"hosts": {}},
		"credentials.use":   {"account": {}, "name": {}},
		"billing.reserve":   {"account": {}, "mode": {}},
		"publication.write": {"target": {}},
	}
	keys, ok := required[identity]
	if !ok {
		return fmt.Errorf("effect identity %q has no Agent Manager enforcement owner", identity)
	}
	for key := range params {
		if _, ok := keys[key]; !ok {
			return fmt.Errorf("%s parameter %q is unsupported", identity, key)
		}
	}
	for key := range keys {
		if strings.TrimSpace(params[key]) == "" {
			return fmt.Errorf("%s requires parameter %q", identity, key)
		}
	}
	return nil
}

func parseEffectGrant(raw string) (string, map[string]string, error) {
	raw = strings.TrimSpace(raw)
	open := strings.IndexByte(raw, '[')
	if open <= 0 || !strings.HasSuffix(raw, "]") {
		return "", nil, errors.New("effect grant must use identity[key=value] syntax")
	}
	identity := raw[:open]
	if !canonicalEffectIdentity(identity) {
		return "", nil, fmt.Errorf("effect identity %q is invalid", identity)
	}
	params := map[string]string{}
	for _, pair := range strings.Split(raw[open+1:len(raw)-1], ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 || !canonicalParam(parts[0]) || strings.TrimSpace(parts[1]) == "" {
			return "", nil, errors.New("effect grant parameter is malformed")
		}
		key := strings.TrimSpace(parts[0])
		if _, exists := params[key]; exists {
			return "", nil, errors.New("effect grant parameter is repeated")
		}
		params[key] = strings.TrimSpace(parts[1])
	}
	return identity, params, nil
}

func effectParameter(raw, name string) string {
	_, params, err := parseEffectGrant(raw)
	if err != nil {
		return ""
	}
	return params[name]
}

func validateRelativeEffectPath(value string) error {
	if !fs.ValidPath(value) || strings.HasPrefix(value, "/") || strings.Contains(value, "..") || strings.Contains(value, "\\") {
		return errors.New("paths must be repository-relative and cannot escape their root")
	}
	return nil
}

func canonicalEffectIdentity(value string) bool {
	for i, part := range strings.Split(value, ".") {
		if part == "" || (i == 0 && (part[0] < 'a' || part[0] > 'z')) {
			return false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
				return false
			}
		}
	}
	return strings.Contains(value, ".")
}

func canonicalParam(value string) bool {
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, r := range value[1:] {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return true
}

func canonicalSlug(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for i, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && !(r == '-' && i > 0) {
			return false
		}
	}
	return true
}

func canonicalScenarioPath(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "scenarios/") && canonicalSlug(strings.TrimPrefix(value, "scenarios/"))
}

package bundleruntime

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// renderEnvMap builds the environment variable map for a service.
// It starts with inherited environment, adds standard bundle hints,
// then applies service and binary-specific overrides.
func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	env := make(map[string]string)

	// Inherit current environment.
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Add standard bundle hints.
	env["APP_DATA_DIR"] = s.appData
	env["BUNDLE_ROOT"] = s.opts.BundlePath

	// Apply service environment (with template expansion).
	for k, v := range svc.Env {
		env[k] = s.renderValue(v)
	}

	// Apply binary-specific environment (overrides service env).
	for k, v := range bin.Env {
		env[k] = s.renderValue(v)
	}

	return env, nil
}

// renderArgs expands template variables in command arguments.
func (s *Supervisor) renderArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, s.renderValue(a))
	}
	return out
}

// renderValue expands template variables in a string.
// Supported templates:
//   - ${data}  -> app data directory
//   - ${bundle} -> bundle root path
//   - ${service.port} -> allocated port for service.port
func (s *Supervisor) renderValue(input string) string {
	// Static replacements.
	replacements := map[string]string{
		"data":   s.appData,
		"bundle": s.opts.BundlePath,
	}

	// Port lookup function for ${service.port} patterns.
	portLookup := func(token string) (string, bool) {
		parts := strings.Split(token, ".")
		if len(parts) != 2 {
			return "", false
		}
		svcID, portName := parts[0], parts[1]
		if svcPorts, ok := s.portMap[svcID]; ok {
			if port, ok := svcPorts[portName]; ok {
				return strconv.Itoa(port), true
			}
		}
		return "", false
	}

	// Match ${...} patterns.
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")

		// Try static replacement first.
		if v, ok := replacements[key]; ok {
			return v
		}

		// Try port lookup.
		if port, ok := portLookup(key); ok {
			return port
		}

		// Keep original if no match.
		return match
	})
}

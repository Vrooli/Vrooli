// Package env provides environment variable management and template expansion.
//
// The runtime supports variable interpolation using ${...} syntax:
//   - ${data}         -> app data directory
//   - ${bundle}       -> bundle root path
//   - ${service.port} -> allocated port for service.port
package env

import (
	"regexp"
	"strconv"
	"strings"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/ports"
)

// Renderer handles template expansion for environment variables and arguments.
type Renderer struct {
	AppData    string
	BundlePath string
	Ports      ports.Allocator
	EnvReader  infra.EnvReader
}

// NewRenderer creates a new environment renderer.
func NewRenderer(appData, bundlePath string, ports ports.Allocator, envReader infra.EnvReader) *Renderer {
	return &Renderer{
		AppData:    appData,
		BundlePath: bundlePath,
		Ports:      ports,
		EnvReader:  envReader,
	}
}

// RenderEnvMap builds the environment variable map for a service.
// It starts with inherited environment, adds standard bundle hints,
// then applies service and binary-specific overrides.
func (r *Renderer) RenderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	env := make(map[string]string)

	// Inherit current environment.
	for _, kv := range r.EnvReader.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Add standard bundle hints.
	env["APP_DATA_DIR"] = r.AppData
	env["BUNDLE_ROOT"] = r.BundlePath

	// Apply service environment (with template expansion).
	for k, v := range svc.Env {
		env[k] = r.RenderValue(v)
	}

	// Apply binary-specific environment (overrides service env).
	for k, v := range bin.Env {
		env[k] = r.RenderValue(v)
	}

	return env, nil
}

// RenderArgs expands template variables in command arguments.
func (r *Renderer) RenderArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, r.RenderValue(a))
	}
	return out
}

// RenderValue expands template variables in a string.
// Supported templates:
//   - ${data}  -> app data directory
//   - ${bundle} -> bundle root path
//   - ${service.port} -> allocated port for service.port
func (r *Renderer) RenderValue(input string) string {
	// Static replacements.
	replacements := map[string]string{
		"data":   r.AppData,
		"bundle": r.BundlePath,
	}

	// Port lookup function for ${service.port} patterns.
	portLookup := func(token string) (string, bool) {
		parts := strings.Split(token, ".")
		if len(parts) != 2 {
			return "", false
		}
		svcID, portName := parts[0], parts[1]
		if port, err := r.Ports.Resolve(svcID, portName); err == nil {
			return strconv.Itoa(port), true
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

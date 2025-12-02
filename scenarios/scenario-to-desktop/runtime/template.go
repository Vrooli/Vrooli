// template.go provides template expansion for environment variables and arguments.
//
// The runtime supports variable interpolation using ${...} syntax:
//   - ${data}         -> app data directory
//   - ${bundle}       -> bundle root path
//   - ${service.port} -> allocated port for service.port
package bundleruntime

import (
	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/manifest"
)

// envRenderer returns an environment renderer for the current supervisor state.
func (s *Supervisor) envRenderer() *env.Renderer {
	return env.NewRenderer(s.appData, s.opts.BundlePath, s.portAllocator, s.envReader)
}

// renderEnvMap builds the environment variable map for a service.
// Delegates to env.Renderer.
func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	return s.envRenderer().RenderEnvMap(svc, bin)
}

// renderArgs expands template variables in command arguments.
// Delegates to env.Renderer.
func (s *Supervisor) renderArgs(args []string) []string {
	return s.envRenderer().RenderArgs(args)
}

// renderValue expands template variables in a string.
// Delegates to env.Renderer.
func (s *Supervisor) renderValue(input string) string {
	return s.envRenderer().RenderValue(input)
}

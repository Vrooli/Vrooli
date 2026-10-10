package nativeisolation

import (
	"path/filepath"
	"strings"
)

// PrepareEnvironment narrows the actual codec environment for an installed
// finite work view. It performs no filesystem/credential/manager operation.
// Duplicate values refuse before normalization; the host HOME/PATH are never
// used inside the image. Other values require the immutable owner's allowlist.
func (rt *Runtime) PrepareEnvironment(repository string, env []string) ([]string, error) {
	if !rt.Enabled() || rt.manifest.Validate() != nil {
		return nil, ErrRefused
	}
	bound := false
	for _, b := range rt.manifest.Bindings {
		if b.Repository == repository {
			bound = true
		}
	}
	if !bound {
		return nil, ErrRefused
	}
	allowed := map[string]bool{}
	for _, key := range rt.manifest.EnvKeys {
		allowed[key] = true
	}
	seen := map[string]bool{}
	out := []string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !envKey.MatchString(key) || seen[key] || strings.ContainsRune(value, 0) {
			return nil, ErrRefused
		}
		seen[key] = true
		if strings.HasPrefix(key, "LD_") || strings.HasPrefix(key, "DYLD_") || key == "DBUS_SESSION_BUS_ADDRESS" || key == "DBUS_SYSTEM_BUS_ADDRESS" || (strings.Contains(key, "PROXY") && value != "") {
			return nil, ErrRefused
		}
		if key != "HOME" && key != "PATH" && allowed[key] {
			out = append(out, entry)
		}
	}
	return append(out, "HOME="+filepath.Join(repository, ".native-home"), "PATH=/usr/bin:/bin"), nil
}

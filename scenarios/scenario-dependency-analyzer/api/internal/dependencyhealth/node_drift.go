package dependencyhealth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

// nodeDependencyDrift compares declared ranges with lockfile root specifiers.
// It compares ranges, not resolved versions, so repair cannot silently upgrade.
func nodeDependencyDrift(surface *healthv1.DependencyHealthSurface) ([]string, error) {
	root := packageRoot(surface)
	manifest, err := readNodeManifest(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, err
	}
	manager := packageManagerForSurface(surface)
	lock, err := readLockSpecifiers(manager, filepath.Join(root, lockfileName(manager)))
	if err != nil {
		return nil, err
	}
	if len(lock) == 0 {
		return nil, nil
	}
	var drift []string
	for name, want := range manifest {
		got, ok := lock[name]
		if !ok || strings.TrimSpace(got) != strings.TrimSpace(want) {
			drift = append(drift, name)
		}
	}
	for name := range lock {
		if _, ok := manifest[name]; !ok {
			drift = append(drift, name)
		}
	}
	sort.Strings(drift)
	return uniqueStrings(drift), nil
}

func readNodeManifest(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package manifest %s: %w", path, err)
	}
	var raw struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode package manifest %s: %w", path, err)
	}
	out := map[string]string{}
	for _, group := range []map[string]string{raw.Dependencies, raw.DevDependencies, raw.OptionalDependencies, raw.PeerDependencies} {
		for name, spec := range group {
			out[name] = spec
		}
	}
	return out, nil
}

func readLockSpecifiers(manager, path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s lockfile: %w", manager, err)
	}
	switch manager {
	case "pnpm":
		if !strings.Contains(string(data), "specifiers:") {
			return map[string]string{}, nil
		}
		var raw struct {
			Specifiers map[string]string `yaml:"specifiers"`
		}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode pnpm lockfile %s: %w", path, err)
		}
		// Older pnpm lockfiles and intentionally minimal health fixtures may
		// omit root specifiers. They are not evidence of divergence; report
		// drift only when the lockfile actually records a comparable contract.
		if len(raw.Specifiers) == 0 {
			return map[string]string{}, nil
		}
		return raw.Specifiers, nil
	case "npm":
		var raw struct {
			Packages map[string]struct {
				Dependencies         map[string]string `json:"dependencies"`
				DevDependencies      map[string]string `json:"devDependencies"`
				OptionalDependencies map[string]string `json:"optionalDependencies"`
			} `json:"packages"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode npm lockfile %s: %w", path, err)
		}
		root := raw.Packages[""]
		out := map[string]string{}
		for _, group := range []map[string]string{root.Dependencies, root.DevDependencies, root.OptionalDependencies} {
			for name, spec := range group {
				out[name] = spec
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("lockfile specifier drift is not supported for %s", manager)
	}
}

func lockfileName(manager string) string {
	switch manager {
	case "npm":
		return "package-lock.json"
	case "yarn":
		return "yarn.lock"
	default:
		return "pnpm-lock.yaml"
	}
}

func uniqueStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

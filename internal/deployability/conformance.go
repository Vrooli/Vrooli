package deployability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ConformanceTarget binds an authored platform claim to the Go module that
// implements it. The manifest path is retained in every result so a compiler
// failure points at the claim an operator must repair.
type ConformanceTarget struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	CodeRoot     string `json:"code_root"`
}

type ConformanceFinding struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	CodeRoot     string `json:"code_root"`
	Message      string `json:"message"`
}

type ConformanceReport struct {
	Targets  []ConformanceTarget  `json:"targets"`
	Findings []ConformanceFinding `json:"findings"`
}

// DiscoverConformanceTargets finds every platform claim whose implementation
// is Go-backed. It intentionally discovers modules rather than maintaining a
// hand-written list of packages.
func DiscoverConformanceTargets(root string) ([]ConformanceTarget, error) {
	var targets []ConformanceTarget
	for _, pattern := range []string{
		filepath.Join(root, "internal", "tools", "*", "tool.json"),
		filepath.Join(root, "internal", "safeguards", "*", "safeguard.json"),
	} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			var manifest struct {
				Platforms []string `json:"platforms"`
			}
			if err := decodeJSON(path, &manifest); err != nil {
				return nil, err
			}
			module, err := nearestGoModule(root, filepath.Dir(path))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			for _, osName := range manifest.Platforms {
				hostOS, ok := conformanceHostOS(osName)
				if !ok {
					return nil, fmt.Errorf("%s declares unknown platform %q", path, osName)
				}
				targets = append(targets, ConformanceTarget{ManifestPath: relativePath(root, path), OS: hostOS, CodeRoot: relativePath(root, module)})
			}
		}
	}
	// Scenario platform_capabilities are claims about their API module.
	scenarioPaths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range scenarioPaths {
		var service struct {
			Service struct {
				PlatformCapabilities map[string]map[string]json.RawMessage `json:"platform_capabilities"`
			} `json:"service"`
		}
		if err := decodeJSON(path, &service); err != nil {
			return nil, err
		}
		apiRoot := filepath.Join(filepath.Dir(filepath.Dir(path)), "api")
		module, err := nearestGoModule(root, apiRoot)
		if err != nil {
			continue
		}
		for _, capability := range service.Service.PlatformCapabilities {
			for osName := range capability {
				hostOS, ok := conformanceHostOS(osName)
				if !ok {
					return nil, fmt.Errorf("%s declares unknown platform %q", path, osName)
				}
				targets = append(targets, ConformanceTarget{ManifestPath: relativePath(root, path), OS: hostOS, CodeRoot: relativePath(root, module)})
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targetKey(targets[i]) < targetKey(targets[j]) })
	return dedupeTargets(targets), nil
}

// CheckRepository cross-compiles each discovered module/OS pair once and
// fans failures back out to every manifest making that claim.
func CheckRepository(ctx context.Context, root string) (ConformanceReport, error) {
	targets, err := DiscoverConformanceTargets(root)
	if err != nil {
		return ConformanceReport{}, err
	}
	report := ConformanceReport{Targets: targets}
	seen := make(map[string]error)
	for _, target := range targets {
		key := target.CodeRoot + "\x00" + string(target.OS)
		buildErr, ok := seen[key]
		if !ok {
			buildErr = crossCompile(ctx, filepath.Join(root, target.CodeRoot), target.OS)
			seen[key] = buildErr
		}
		if buildErr != nil {
			report.Findings = append(report.Findings, ConformanceFinding{ManifestPath: target.ManifestPath, OS: target.OS, CodeRoot: target.CodeRoot, Message: buildErr.Error()})
		}
	}
	return report, nil
}

func crossCompile(ctx context.Context, module string, hostOS HostOS) error {
	arch := "amd64"
	goos := string(hostOS)
	if hostOS == HostOSMacOS {
		goos = "darwin"
		arch = "arm64"
	}
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = module
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOOS="+goos, "GOARCH="+arch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GOOS=%s GOARCH=%s: %w: %s", goos, arch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func decodeJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
func targetKey(t ConformanceTarget) string {
	return t.ManifestPath + "\x00" + string(t.OS) + "\x00" + t.CodeRoot
}
func dedupeTargets(in []ConformanceTarget) []ConformanceTarget {
	out := in[:0]
	seen := map[string]bool{}
	for _, target := range in {
		key := targetKey(target)
		if !seen[key] {
			seen[key] = true
			out = append(out, target)
		}
	}
	return out
}

func nearestGoModule(root, start string) (string, error) {
	path := start
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path, nil
		}
		if path == root || filepath.Dir(path) == path {
			break
		}
		path = filepath.Dir(path)
	}
	return "", fmt.Errorf("no go.mod found from %s", start)
}

func conformanceHostOS(value string) (HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(HostOSLinux):
		return HostOSLinux, true
	case string(HostOSMacOS), "darwin":
		return HostOSMacOS, true
	case string(HostOSWindows):
		return HostOSWindows, true
	default:
		return "", false
	}
}

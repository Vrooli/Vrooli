package installgateway

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// optionalDependencySizer is the registry capability needed to bound a host
// tool whose executable lives in an optional platform package. The production
// registry HTTP client implements it; test fakes that do not are still handled
// by counting only the main package (and therefore refusing an obviously
// oversized install later).
type optionalDependencySizer interface {
	OptionalDependencies(ctx context.Context, name, version string) (map[string]string, error)
}

// probeResourceAgentGrowth bounds a `resource-agent` npm install without
// staging it. The package is a host tool: its main tarball plus the optional
// platform packages npm may install are the whole artifact. Summing the
// platform-relevant optional dependencies is deliberately conservative; when
// no name matches the host platform, counting all of them is safer than
// guessing.
func probeResourceAgentGrowth(ctx context.Context, r Resolution) (int64, error) {
	if strings.TrimSpace(r.PackageName) == "" || strings.TrimSpace(r.HostToolTargetVersion) == "" {
		return 0, fmt.Errorf("resource-agent install has no package/version to size")
	}
	sizer, err := newRegistrySizer()
	if err != nil {
		return 0, err
	}
	var total int64
	add := func(name, version string) error {
		registryName, registryVersion, err := resolveRegistryPackage(name, version)
		if err != nil {
			return err
		}
		size, err := sizer.Registry(ctx, registryName, registryVersion, "")
		if err != nil {
			return err
		}
		total += size.Unpacked + size.Archive
		return nil
	}
	if err := add(r.PackageName, r.HostToolTargetVersion); err != nil {
		return 0, err
	}
	if withOptional, ok := sizer.(optionalDependencySizer); ok {
		dependencies, err := withOptional.OptionalDependencies(ctx, r.PackageName, r.HostToolTargetVersion)
		if err != nil {
			return 0, err
		}
		selected := platformOptionalDependencies(dependencies, runtime.GOOS, runtime.GOARCH)
		if len(selected) == 0 {
			selected = dependencies
		}
		for name, version := range selected {
			if err := add(name, version); err != nil {
				return 0, err
			}
		}
	}
	if total <= 0 {
		return 0, fmt.Errorf("resource-agent package size probe returned no positive bound")
	}
	return total, nil
}

// resolveRegistryPackage converts an npm optional-dependency alias into the
// real package name and exact version that the public registry serves. The
// alias name is the local install path; registry metadata is keyed by the
// target package instead.
func resolveRegistryPackage(installName, spec string) (string, string, error) {
	if !strings.HasPrefix(spec, "npm:") {
		if !exactNpmVersion.MatchString(spec) {
			return "", "", fmt.Errorf("optional npm dependency %s has no exact registry version", installName)
		}
		return installName, spec, nil
	}
	target := strings.TrimPrefix(spec, "npm:")
	separator := strings.LastIndexByte(target, '@')
	if separator <= 0 || separator == len(target)-1 {
		return "", "", fmt.Errorf("optional npm alias %s has no exact package target", installName)
	}
	name, version := target[:separator], target[separator+1:]
	if !validRegistryPackageName(name) || !exactNpmVersion.MatchString(version) {
		return "", "", fmt.Errorf("optional npm alias %s must target a valid package and exact version", installName)
	}
	return name, version, nil
}

func validRegistryPackageName(name string) bool {
	if strings.HasPrefix(name, "@") {
		rest := name[1:]
		scope, packageName, ok := strings.Cut(rest, "/")
		return ok && scope != "" && packageName != "" && !strings.Contains(packageName, "/") && !strings.ContainsAny(rest, "@\\:#? \t\r\n")
	}
	return name != "" && !strings.ContainsAny(name, "@/\\:#? \t\r\n")
}

func platformOptionalDependencies(dependencies map[string]string, goos, goarch string) map[string]string {
	arch := map[string]string{"amd64": "x64", "arm64": "arm64", "386": "ia32"}[goarch]
	if arch == "" {
		return nil
	}
	token := "-" + goos + "-" + arch
	out := map[string]string{}
	for name, version := range dependencies {
		if strings.Contains(name, token) {
			out[name] = version
		}
	}
	return out
}

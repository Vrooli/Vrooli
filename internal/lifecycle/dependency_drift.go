package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
)

const dependencySpecifierMatchParts = 3

// checkNodeLockfileDrift is deliberately read-only. It runs immediately
// before a JavaScript component install/build and makes a stale lockfile a
// typed lifecycle failure instead of an opaque package-manager error.
func checkNodeLockfileDrift(root string) ([]string, error) {
	manifest, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Optional        map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(manifest, &raw); err != nil {
		return nil, fmt.Errorf("decode package.json: %w", err)
	}
	lock, err := os.ReadFile(filepath.Join(root, "pnpm-lock.yaml"))
	if err != nil {
		return nil, nil
	}
	text := string(lock)
	if !strings.Contains(text, "specifiers:") {
		return nil, nil
	}
	declared := map[string]string{}
	for _, group := range []map[string]string{raw.Dependencies, raw.DevDependencies, raw.Optional} {
		for name, spec := range group {
			declared[name] = spec
		}
	}
	found := map[string]string{}
	linePattern := regexp.MustCompile(`^\s{2}([^:#]+):\s*(\S+)\s*$`)
	inSpecifiers := false
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "specifiers:" {
			inSpecifiers = true
			continue
		}
		if inSpecifiers && len(line) > 0 && line[0] != ' ' {
			break
		}
		if inSpecifiers {
			match := linePattern.FindStringSubmatch(line)
			if len(match) == dependencySpecifierMatchParts {
				found[strings.TrimSpace(match[1])] = strings.TrimSpace(match[2])
			}
		}
	}
	var drift []string
	for name, spec := range declared {
		if found[name] != spec {
			drift = append(drift, name)
		}
	}
	for name := range found {
		if _, ok := declared[name]; !ok {
			drift = append(drift, name)
		}
	}
	sort.Strings(drift)
	return drift, nil
}

// diagnoseGoModuleDrift runs only after a Go component build fails. It asks the
// toolchain, read-only, whether the module's go.mod/go.sum lag its import
// closure — the class where one new import in a shared in-repo package breaks
// every consumer module that lacks the matching requirement or checksum — and
// returns the toolchain's diagnostics for that class. Any other failure, or an
// unavailable toolchain, yields nil so the original build error stands alone.
func diagnoseGoModuleDrift(parent context.Context, dir string) []string {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, tuning.LifecycleOperationTimeout())
	defer cancel()
	cmd := shell.NewCommandContext(ctx, goBin, "list", "-mod=readonly", "-deps", "./...")
	cmd.Dir = dir
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off"})
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		return nil
	}
	return goModuleDriftDiagnostics(stderr.String())
}

// goModuleDriftSignatures are the toolchain diagnostics for module metadata
// that lags the import closure under -mod=readonly.
var goModuleDriftSignatures = []string{
	"missing go.sum entry",
	"updates to go.mod needed",
	"no required module provides package",
	"implicitly required module",
}

// goModuleDriftDiagnostics extracts the distinct drift diagnostics from
// toolchain output, at most three. The toolchain's own advice (`go get`,
// `go mod tidy`) is dropped: dependency changes go through the governed
// reconcile named in the lifecycle error.
func goModuleDriftDiagnostics(output string) []string {
	const limit = 3
	var out []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		matched := false
		for _, signature := range goModuleDriftSignatures {
			if strings.Contains(line, signature) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if cut := strings.Index(line, "; to add"); cut >= 0 {
			line = line[:cut]
		}
		if cut := strings.Index(line, "; to update it"); cut >= 0 {
			line = line[:cut]
		}
		line = strings.TrimSuffix(strings.TrimSpace(line), ":")
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
		if len(out) == limit {
			break
		}
	}
	return out
}

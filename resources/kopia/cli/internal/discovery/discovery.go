// Package discovery locates the kopia host binary and probes its version. It is
// the read side of binary provisioning; the install command set (manifest
// cli.source_build) is the write side.
package discovery

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"resource-kopia/cli/internal/version"
)

// Locator finds the kopia binary on the host.
type Locator struct {
	// LookPath resolves a command to an absolute path. Overridable in tests.
	LookPath func(string) (string, error)
	// Binary is the binary name to look up (default "kopia").
	Binary string
}

// NewLocator returns a Locator wired to the host PATH.
func NewLocator() Locator {
	return Locator{LookPath: exec.LookPath, Binary: "kopia"}
}

func (l Locator) binary() string {
	if strings.TrimSpace(l.Binary) == "" {
		return "kopia"
	}
	return l.Binary
}

// Resolve returns the absolute path to the kopia binary or an error if it is
// not installed.
func (l Locator) Resolve() (string, error) {
	lookPath := l.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(l.binary())
	if err != nil {
		return "", fmt.Errorf("kopia binary not found on PATH; run `vrooli setup --resources kopia`: %w", err)
	}
	return path, nil
}

// ProbeVersion runs `kopia --version` and returns the parsed installed version
// plus whether it satisfies the pinned version.
func (l Locator) ProbeVersion(ctx context.Context, run func(ctx context.Context, name string, args ...string) (string, error)) (version.Semver, bool, error) {
	binary, err := l.Resolve()
	if err != nil {
		return version.Semver{}, false, err
	}
	if run == nil {
		run = runCommand
	}
	out, err := run(ctx, binary, "--version")
	if err != nil {
		return version.Semver{}, false, fmt.Errorf("probe kopia version: %w", err)
	}
	return version.Check(out)
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

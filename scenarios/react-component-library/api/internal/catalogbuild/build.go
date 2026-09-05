// Package catalogbuild owns the typed Go seam for the catalog generator.
// The JavaScript projections remain the implementation because they already
// own package export and TypeScript graph semantics; callers do not need to
// know that there are multiple internal projection steps.
package catalogbuild

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Options struct{ Check bool }

type Report struct {
	Generator string `json:"generator"`
	Check     bool   `json:"check"`
	Status    string `json:"status"`
	Output    string `json:"output,omitempty"`
}

// Build runs the single public catalog generator. It is deliberately rooted
// at an explicit repository path and never falls back to the process cwd.
func Build(root string, options Options) (Report, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return Report{}, errors.New("catalog build root is required")
	}
	script := filepath.Join(root, "packages", "react-component-library", "tooling", "catalog-build.mjs")
	if _, err := os.Stat(script); err != nil {
		return Report{}, fmt.Errorf("catalog generator: %w", err)
	}
	args := []string{script}
	if options.Check {
		args = append(args, "--check")
	}
	cmd := exec.Command("node", args...)
	cmd.Dir = root
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	runErr := cmd.Run()
	report := Report{Check: options.Check, Output: output.String()}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		if err := json.Unmarshal([]byte(line), &report); err == nil {
			break
		}
	}
	if runErr != nil {
		return report, fmt.Errorf("catalog generator failed: %w\n%s", runErr, strings.TrimSpace(output.String()))
	}
	if report.Status == "" {
		return report, errors.New("catalog generator returned no status")
	}
	return report, nil
}

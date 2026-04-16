package cliutil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
)

// FreshnessSpec defines the exact source contract used to compute a CLI
// freshness fingerprint.
type FreshnessSpec struct {
	SourceRoot  string
	ContextRoot string
	Inputs      []string
	SkipFiles   []string
}

// ComputeFreshnessFingerprint computes a deterministic fingerprint from one
// canonical freshness contract. When Inputs are declared, they are resolved
// from ContextRoot. Otherwise the entire SourceRoot is fingerprinted.
func ComputeFreshnessFingerprint(spec FreshnessSpec) (string, error) {
	sourceRoot := filepath.Clean(strings.TrimSpace(spec.SourceRoot))
	if sourceRoot == "" {
		return "", fmt.Errorf("freshness source root must not be empty")
	}

	skipFiles := append([]string(nil), spec.SkipFiles...)
	inputs := trimNonEmpty(spec.Inputs)
	if len(inputs) == 0 {
		return buildinfo.ComputeFingerprint(sourceRoot, skipFiles...)
	}

	contextRoot := filepath.Clean(strings.TrimSpace(spec.ContextRoot))
	if contextRoot == "" {
		contextRoot = sourceRoot
	}
	return computeFingerprintFromDeclaredInputs(contextRoot, inputs, skipFiles...)
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

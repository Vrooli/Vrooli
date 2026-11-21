package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestCoverageFloors keeps the historical per-package coverage contract in the
// normal Go test suite. Keeping this policy in Go, rather than an unmanaged
// shell script, makes it visible to Test Genie and portable across platforms.
func TestCoverageFloors(t *testing.T) {
	floors := map[string]int{
		"audio-tools/internal/byok/envelope":      90,
		"audio-tools/internal/protomap":           80,
		"audio-tools/internal/database":           80,
		"audio-tools/internal/middleware":         80,
		"audio-tools/internal/modules":            80,
		"audio-tools/internal/ai/chains":          80,
		"audio-tools/internal/server":             63,
		"audio-tools/internal/session":            80,
		"audio-tools/internal/stt/strategy":       80,
		"audio-tools/internal/capabilities":       70,
		"audio-tools/handlers/health_status":      80,
		"audio-tools/handlers/provider_lifecycle": 65,
		"audio-tools/internal/store":              70,
		"audio-tools/internal/byokstore":          70,
		"audio-tools/internal/byok":               80,
		"audio-tools/internal/summarize":          80,
		"audio-tools/internal/stt/segmenter":      75,
		// Chain floors restored after the catalog/model work: the duplicated
		// provider plumbing was consolidated and covered with behaviour tests
		// for fallback ordering, caching, error mapping, and credential
		// redaction.
		"audio-tools/internal/ai/sttchain":       80,
		"audio-tools/internal/ai/summarizechain": 85,
		"audio-tools/internal/ai/ttschain":       80,
		"audio-tools/internal/diagnostics":       90,
		"audio-tools/handlers/stt":               80,
		"audio-tools/internal/audio":             75,
		"audio-tools/internal/stt/pipeline":      55,
		"audio-tools/internal/tts":               70,
		"audio-tools/internal/usagereport":       65,
		"audio-tools/internal/testutil/vendorws": 80,
		// Composition and lab/qualification packages were previously
		// unenforced. These floors are measured-baseline, not aspirations:
		// the gate exists to stop silent regression, not to claim coverage.
		// Floors keep at least five points of headroom so an ordinary
		// refactor does not trip the gate.
		"audio-tools/internal/bootstrap": 50,
		// Lab packages are product-mounted but not serving paths. The
		// boundary that keeps them out of request handling is documented and
		// enforced in api/internal/lab.
		"audio-tools/internal/experiment":          64,
		"audio-tools/internal/experiment/evaldeps": 70,
		"audio-tools/internal/experiment/recipe":   68,
		"audio-tools/internal/experiment/report":   73,
		"audio-tools/internal/eval":                77,
		"audio-tools/internal/soak":                62,
		"audio-tools/handlers/experiment":          38,
		"audio-tools/handlers/eval":                87,
		"audio-tools/handlers/soak":                32,
		"audio-tools/handlers/validation":          56,
	}

	packages := make([]string, 0, len(floors))
	for pkg := range floors {
		packages = append(packages, "./"+strings.TrimPrefix(pkg, "audio-tools/"))
	}
	cmd := exec.Command("go", append([]string{"test", "-race", "-cover", "-timeout", "300s"}, packages...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run package coverage: %v\n%s", err, output)
	}

	coverageLine := regexp.MustCompile(`(?m)^ok\s+(audio-tools/\S+).*coverage:\s+([0-9]+(?:\.[0-9]+)?)%`)
	seen := make(map[string]float64, len(floors))
	for _, match := range coverageLine.FindAllStringSubmatch(string(output), -1) {
		coverage, parseErr := strconv.ParseFloat(match[2], 64)
		if parseErr == nil {
			seen[match[1]] = coverage
		}
	}
	for pkg, floor := range floors {
		coverage, found := seen[pkg]
		if !found {
			t.Errorf("%s: coverage result missing", pkg)
			continue
		}
		if coverage < float64(floor) {
			t.Errorf("%s: %.1f%% coverage is below the %d%% floor", pkg, coverage, floor)
		}
	}
}

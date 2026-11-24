package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func assertCoverageFloors(t *testing.T, floors map[string]int, extraArgs []string) {
	t.Helper()
	packages := make([]string, 0, len(floors))
	for pkg := range floors {
		packages = append(packages, "./"+strings.TrimPrefix(pkg, "audio-tools/"))
	}
	args := []string{"test"}
	args = append(args, extraArgs...)
	args = append(args, "-race", "-cover", "-timeout", "300s")
	args = append(args, packages...)
	cmd := exec.Command("go", args...)
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

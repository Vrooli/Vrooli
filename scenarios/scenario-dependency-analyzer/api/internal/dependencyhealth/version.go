package dependencyhealth

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func (h *connectHandler) checkVersion(ctx context.Context, dir string, runner commandRunner, command string, constraintRaw string) []*healthv1.DependencyHealthFinding {
	out, err := runner.Run(ctx, dir, command, versionArgs(command)...)
	if err != nil {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".unavailable",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version unavailable",
			Description:  fmt.Sprintf("SDA could not read %s version.", command),
			Remediation:  fmt.Sprintf("Install or activate %s matching %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     err.Error(),
			Expected:     command + " " + constraintRaw,
		}}
	}
	version, err := parseVersionOutput(out)
	if err != nil {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".unparseable",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version unparseable",
			Description:  fmt.Sprintf("SDA could not parse %s version output.", command),
			Remediation:  fmt.Sprintf("Ensure %s prints a parseable version matching %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     strings.TrimSpace(out),
			Expected:     command + " " + constraintRaw,
		}}
	}
	min, err := parseVersion(strings.TrimPrefix(constraintRaw, ">="))
	if err != nil {
		return nil
	}
	if !version.atLeast(min) {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".too-old",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version too old",
			Description:  fmt.Sprintf("%s version %s is below required %s.", command, version, constraintRaw),
			Remediation:  fmt.Sprintf("Install or activate %s %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     version.String(),
			Expected:     constraintRaw,
		}}
	}
	return nil
}

type version struct {
	major int
	minor int
	patch int
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) atLeast(min version) bool {
	if v.major != min.major {
		return v.major > min.major
	}
	if v.minor != min.minor {
		return v.minor > min.minor
	}
	return v.patch >= min.patch
}

var versionNumberRE = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

func parseVersionOutput(output string) (version, error) {
	match := versionNumberRE.FindString(strings.TrimSpace(output))
	if match == "" {
		return version{}, fmt.Errorf("could not parse version from %q", strings.TrimSpace(output))
	}
	return parseVersion(match)
}

func parseVersion(raw string) (version, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	values := []int{0, 0, 0}
	if len(parts) == 0 || len(parts) > 3 {
		return version{}, fmt.Errorf("invalid version %q", raw)
	}
	for i, part := range parts {
		var value int
		if _, err := fmt.Sscanf(part, "%d", &value); err != nil {
			return version{}, fmt.Errorf("invalid version %q: %w", raw, err)
		}
		values[i] = value
	}
	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func versionArgs(command string) []string {
	if command == "go" {
		return []string{"version"}
	}
	return []string{"--version"}
}

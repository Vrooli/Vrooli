package dependencies

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// CommandRunner runs read-only host commands for dependency probes.
type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

type VersionConstraint struct {
	Min Version
}

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersionConstraint(raw string) (VersionConstraint, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return VersionConstraint{}, fmt.Errorf("version constraint is empty")
	}
	if !strings.HasPrefix(value, ">=") {
		return VersionConstraint{}, fmt.Errorf("only >= constraints are supported")
	}
	version, err := ParseVersion(strings.TrimSpace(strings.TrimPrefix(value, ">=")))
	if err != nil {
		return VersionConstraint{}, err
	}
	return VersionConstraint{Min: version}, nil
}

var versionNumberRE = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

func ParseVersionOutput(command, output string) (Version, error) {
	text := strings.TrimSpace(output)
	if text == "" {
		return Version{}, fmt.Errorf("%s version output is empty", command)
	}
	match := versionNumberRE.FindString(text)
	if match == "" {
		return Version{}, fmt.Errorf("could not parse %s version from %q", command, text)
	}
	return ParseVersion(match)
}

func ParseVersion(raw string) (Version, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	values := []int{0, 0, 0}
	for i, part := range parts {
		if part == "" {
			return Version{}, fmt.Errorf("invalid version %q", raw)
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %w", raw, err)
		}
		values[i] = value
	}
	return Version{Major: values[0], Minor: values[1], Patch: values[2]}, nil
}

func (v Version) AtLeast(min Version) bool {
	if v.Major != min.Major {
		return v.Major > min.Major
	}
	if v.Minor != min.Minor {
		return v.Minor > min.Minor
	}
	return v.Patch >= min.Patch
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func versionArgs(command string) []string {
	switch command {
	case "go":
		return []string{"version"}
	case "node", "python3", "pnpm", "npm", "yarn":
		return []string{"--version"}
	default:
		return []string{"--version"}
	}
}

func (r *Runner) checkVersion(ctx context.Context, command string, constraintRaw string, observations *[]Observation, summary ValidationSummary) *RunResult {
	if strings.TrimSpace(constraintRaw) == "" {
		return nil
	}
	constraint, err := ParseVersionConstraint(constraintRaw)
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassMisconfiguration,
			Remediation:  "Fix dependencies.runtime_versions in .vrooli/testing.json.",
			Observations: *observations,
			Summary:      summary,
		}
	}
	out, err := r.commandRunner.Run(ctx, r.config.ScenarioDir, command, versionArgs(command)...)
	if err != nil {
		if r.settings.Strict {
			return &RunResult{
				Success:      false,
				Error:        fmt.Errorf("read %s version: %w", command, err),
				FailureClass: FailureClassMissingDependency,
				Remediation:  fmt.Sprintf("Install or activate %s matching %s.", command, constraintRaw),
				Observations: *observations,
				Summary:      summary,
			}
		}
		*observations = append(*observations, NewWarningObservation(fmt.Sprintf("could not read %s version: %v", command, err)))
		return nil
	}
	version, err := ParseVersionOutput(command, out)
	if err != nil {
		if r.settings.Strict {
			return &RunResult{
				Success:      false,
				Error:        err,
				FailureClass: FailureClassMissingDependency,
				Remediation:  fmt.Sprintf("Ensure %s prints a parseable version matching %s.", command, constraintRaw),
				Observations: *observations,
				Summary:      summary,
			}
		}
		*observations = append(*observations, NewWarningObservation(err.Error()))
		return nil
	}
	if !version.AtLeast(constraint.Min) {
		return &RunResult{
			Success:      false,
			Error:        fmt.Errorf("%s version %s is below required %s", command, version, constraintRaw),
			FailureClass: FailureClassMissingDependency,
			Remediation:  fmt.Sprintf("Install or activate %s %s before rerunning the dependencies phase.", command, constraintRaw),
			Observations: append(*observations, NewErrorObservation(fmt.Sprintf("runtime version too old: %s %s < %s", command, version, constraintRaw))),
			Summary:      summary,
		}
	}
	*observations = append(*observations, NewSuccessObservation(fmt.Sprintf("runtime version OK: %s %s", command, version)))
	return nil
}

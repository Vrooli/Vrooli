package support

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func ScenarioPath(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		input = "."
	}
	if info, err := os.Stat(input); err == nil && info.IsDir() {
		return filepath.Abs(input)
	}
	resolved := cliutil.ResolveScenarioPath(input)
	if strings.TrimSpace(resolved) == "" {
		return "", fmt.Errorf("invalid scenario path or name: %s", input)
	}
	return filepath.Abs(resolved)
}

func ScenarioName(input string) string {
	input = strings.TrimSpace(input)
	if input == "" || input == "." {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Base(cwd)
		}
		return ""
	}
	if info, err := os.Stat(input); err == nil && info.IsDir() {
		return filepath.Base(input)
	}
	return input
}

func Decode(body []byte, dest interface{}) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func StatusLine(ok bool, label, detail string) string {
	prefix := "FAIL"
	if ok {
		prefix = "OK"
	}
	if strings.TrimSpace(detail) == "" {
		return fmt.Sprintf("%s: %s", prefix, label)
	}
	return fmt.Sprintf("%s: %s (%s)", prefix, label, detail)
}

func NormalizePathList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func SeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 0
	case "high", "error":
		return 1
	case "medium", "warning", "warn":
		return 2
	case "low":
		return 3
	case "info":
		return 4
	default:
		return 5
	}
}

func ResolveCampaignID(campaigns []Campaign, scenario, preferredStatus string) (int, error) {
	scenario = strings.TrimSpace(scenario)
	preferredStatus = strings.TrimSpace(preferredStatus)
	for _, campaign := range campaigns {
		if campaign.Scenario == scenario && (preferredStatus == "" || campaign.Status == preferredStatus) {
			return campaign.ID, nil
		}
	}
	for _, campaign := range campaigns {
		if campaign.Scenario == scenario {
			return campaign.ID, nil
		}
	}
	return 0, fmt.Errorf("no matching campaign found for scenario %s", scenario)
}

func ParseIntArg(value, flagName string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", flagName)
	}
	return parsed, nil
}

func RunVisitedTracker(args ...string) error {
	cmd := exec.Command("visited-tracker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("visited-tracker exited with status %d", exitErr.ExitCode())
		}
		return fmt.Errorf("run visited-tracker: %w", err)
	}
	return nil
}

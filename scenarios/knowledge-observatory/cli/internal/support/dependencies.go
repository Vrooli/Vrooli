package support

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type Dependencies struct {
	Core func() *cliapp.ScenarioApp
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func SplitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func SplitLeadingPositional(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}
	first := strings.TrimSpace(args[0])
	if first == "" || strings.HasPrefix(first, "-") {
		return "", args
	}
	return first, args[1:]
}

func StripBoolFlag(args []string, flagName string) ([]string, bool) {
	if strings.TrimSpace(flagName) == "" || len(args) == 0 {
		return args, false
	}
	filtered := make([]string, 0, len(args))
	found := false
	for _, arg := range args {
		if strings.TrimSpace(arg) == flagName {
			found = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, found
}

func ParseMetadata(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid metadata JSON: %w", err)
	}
	if out == nil {
		return nil, fmt.Errorf("metadata must be a JSON object")
	}
	return out, nil
}

func ReadContent(explicit string, args []string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if StdinHasData() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

func StdinHasData() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func ParseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("interval is required")
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	if seconds, err := time.ParseDuration(raw + "s"); err == nil {
		return seconds, nil
	}
	return 0, fmt.Errorf("invalid interval %q", raw)
}

func SortStringsDedup(values []string) []string {
	filtered := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}
	sort.Strings(filtered)
	return filtered
}

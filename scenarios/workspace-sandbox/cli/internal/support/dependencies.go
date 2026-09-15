package support

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const CLIName = "workspace-sandbox"

type Dependencies struct {
	Core func() *cliapp.ScenarioApp
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func ResolveSandboxID(core *cliapp.ScenarioApp, shortID string) (string, error) {
	shortID = strings.TrimSpace(shortID)
	if shortID == "" {
		return "", fmt.Errorf("sandbox ID required")
	}
	if len(shortID) == 36 && strings.Count(shortID, "-") == 4 {
		return shortID, nil
	}

	body, err := core.Get("/sandboxes", nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sandboxes for ID resolution: %w", err)
	}

	var resp ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse sandbox list: %w", err)
	}
	if strings.EqualFold(shortID, "latest") {
		for _, sandbox := range resp.Sandboxes {
			if sandbox.Status == "active" || sandbox.Status == "creating" {
				return sandbox.ID, nil
			}
		}
		return "", fmt.Errorf("no active sandbox found for %q", shortID)
	}

	shortIDLower := strings.ToLower(shortID)
	matches := make([]SandboxResponse, 0, len(resp.Sandboxes))
	for _, sandbox := range resp.Sandboxes {
		if strings.HasPrefix(strings.ToLower(sandbox.ID), shortIDLower) {
			matches = append(matches, sandbox)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no sandbox found matching prefix %q", shortID)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, TruncateID(match.ID))
		}
		return "", fmt.Errorf("ambiguous prefix %q matches %d sandboxes: %s", shortID, len(matches), strings.Join(ids, ", "))
	}
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func TruncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func TruncateID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}

func TailTruncate(value string, max int) string {
	if max <= 3 || len(value) <= max {
		return value
	}
	return "..." + value[len(value)-(max-3):]
}

func DisplayOwner(owner string) string {
	if strings.TrimSpace(owner) == "" {
		return "-"
	}
	return owner
}

func ChangeTypeSymbol(changeType string) string {
	switch changeType {
	case "added":
		return "+"
	case "deleted":
		return "-"
	case "modified":
		return "~"
	default:
		return " "
	}
}

func RenderValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "-"
	case string:
		if strings.TrimSpace(v) == "" {
			return "-"
		}
		return v
	case time.Time:
		if v.IsZero() {
			return "-"
		}
		return v.Format(time.RFC3339)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(raw)
	}
}

func SortedMapLines(values map[string]any) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", key, RenderValue(values[key])))
	}
	return lines
}

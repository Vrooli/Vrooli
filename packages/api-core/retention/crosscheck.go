package retention

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// RetentionConflict is one disagreement between a manifest's retention.budgets
// block and its durable_data block about the same path.
type RetentionConflict struct {
	Path   string
	Budget string
	Entry  string
	Reason string
}

func (c RetentionConflict) String() string {
	return fmt.Sprintf("retention budget %q and durable_data entry %q both declare %q: %s", c.Budget, c.Entry, c.Path, c.Reason)
}

type crosscheckManifest struct {
	Retention struct {
		Budgets map[string]struct {
			Target struct {
				Kind     string `json:"kind"`
				Database string `json:"database"`
				Path     string `json:"path"`
			} `json:"target"`
		} `json:"budgets"`
	} `json:"retention"`
	DurableData struct {
		Entries map[string]struct {
			Path   string `json:"path"`
			Kind   string `json:"kind"`
			Format string `json:"format"`
		} `json:"entries"`
	} `json:"durable_data"`
}

// ValidateRetentionAgainstDurableData reports every path declared in both a
// manifest's retention.budgets and durable_data with a disagreeing kind or
// format. Paths are compared by segment suffix because the two blocks use
// different storage roots.
func ValidateRetentionAgainstDurableData(manifest []byte) ([]RetentionConflict, error) {
	var m crosscheckManifest
	if err := json.Unmarshal(manifest, &m); err != nil {
		return nil, fmt.Errorf("parse manifest for retention cross-check: %w", err)
	}

	var conflicts []RetentionConflict
	for budgetName, budget := range m.Retention.Budgets {
		targetPath, wantKind, wantFormat := retentionTargetShape(budget.Target.Kind, budget.Target.Database, budget.Target.Path)
		if targetPath == "" {
			continue
		}
		for entryName, entry := range m.DurableData.Entries {
			if !pathsOverlap(targetPath, entry.Path) {
				continue
			}
			if entry.Kind != "" && entry.Kind != wantKind {
				conflicts = append(conflicts, RetentionConflict{Path: targetPath, Budget: budgetName, Entry: entryName, Reason: fmt.Sprintf("durable_data kind is %q but a %q retention target is a %q", entry.Kind, budget.Target.Kind, wantKind)})
			}
			if wantFormat == "" && entry.Format != "" {
				conflicts = append(conflicts, RetentionConflict{Path: targetPath, Budget: budgetName, Entry: entryName, Reason: fmt.Sprintf("durable_data format is %q but a %q retention target has no on-disk file format", entry.Format, budget.Target.Kind)})
			}
			if wantFormat != "" && entry.Format != "" && entry.Format != wantFormat {
				conflicts = append(conflicts, RetentionConflict{Path: targetPath, Budget: budgetName, Entry: entryName, Reason: fmt.Sprintf("durable_data format is %q but a %q retention target is %q", entry.Format, budget.Target.Kind, wantFormat)})
			}
		}
	}

	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Budget != conflicts[j].Budget {
			return conflicts[i].Budget < conflicts[j].Budget
		}
		if conflicts[i].Entry != conflicts[j].Entry {
			return conflicts[i].Entry < conflicts[j].Entry
		}
		return conflicts[i].Reason < conflicts[j].Reason
	})
	return conflicts, nil
}

func retentionTargetShape(kind, database, dirPath string) (string, string, string) {
	switch kind {
	case "sqlite_table":
		return normalizeManifestPath(database), "file", "sqlite"
	case "directory":
		return normalizeManifestPath(dirPath), "dir", ""
	default:
		return "", "", ""
	}
}

func normalizeManifestPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.Trim(path.Clean(value), "/")
}

func pathsOverlap(a, b string) bool {
	a, b = normalizeManifestPath(a), normalizeManifestPath(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	longer, shorter := a, b
	if len(shorter) > len(longer) {
		longer, shorter = shorter, longer
	}
	return strings.HasSuffix(longer, "/"+shorter)
}

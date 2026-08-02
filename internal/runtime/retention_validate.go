package runtime

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// A manifest may declare the same on-disk location twice, once under
// durable_data ("back this up") and once under retention.budgets ("never let
// this exceed this much"). The two blocks are deliberately separate because
// they answer near-opposite questions, but where they name the same path they
// must agree about what that path IS. A durable_data entry that calls
// autoheal.sqlite a directory while a retention budget treats it as a SQLite
// database means one of the two declarations is wrong, and the wrong one is not
// discoverable at runtime: the backup would silently capture nothing, or the
// pruner would refuse to open the target. This validator makes that
// disagreement a manifest error instead.

// RetentionConflict is one disagreement between a manifest's retention.budgets
// block and its durable_data block about the same path.
type RetentionConflict struct {
	// Path is the retention target path that collided.
	Path string
	// Budget is the retention.budgets key that declared it.
	Budget string
	// Entry is the durable_data entry key that declared it.
	Entry string
	// Reason states which field disagreed and how.
	Reason string
}

func (c RetentionConflict) String() string {
	return fmt.Sprintf("retention budget %q and durable_data entry %q both declare %q: %s", c.Budget, c.Entry, c.Path, c.Reason)
}

// retentionManifest is the minimal projection of a manifest needed for the
// cross-block check. It deliberately ignores every other field so the check
// applies uniformly to service, resource, tool, and safeguard manifests.
type retentionManifest struct {
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
// manifest's retention.budgets and its durable_data with a disagreeing kind or
// format. It returns conflicts sorted by budget key so output is stable.
//
// The two blocks are anchored at different bases — durable_data paths are
// relative to a host base such as $HOME, retention target paths are relative to
// a storage class root — so paths are matched when one is a path-segment suffix
// of the other rather than by exact equality.
func ValidateRetentionAgainstDurableData(manifest []byte) ([]RetentionConflict, error) {
	var m retentionManifest
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
				conflicts = append(conflicts, RetentionConflict{
					Path:   targetPath,
					Budget: budgetName,
					Entry:  entryName,
					Reason: fmt.Sprintf("durable_data kind is %q but a %q retention target is a %q", entry.Kind, budget.Target.Kind, wantKind),
				})
			}
			if wantFormat == "" && entry.Format != "" {
				conflicts = append(conflicts, RetentionConflict{
					Path:   targetPath,
					Budget: budgetName,
					Entry:  entryName,
					Reason: fmt.Sprintf("durable_data format is %q but a %q retention target has no on-disk file format", entry.Format, budget.Target.Kind),
				})
			}
			if wantFormat != "" && entry.Format != "" && entry.Format != wantFormat {
				conflicts = append(conflicts, RetentionConflict{
					Path:   targetPath,
					Budget: budgetName,
					Entry:  entryName,
					Reason: fmt.Sprintf("durable_data format is %q but a %q retention target is %q", entry.Format, budget.Target.Kind, wantFormat),
				})
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

// retentionTargetShape maps a retention target kind onto the durable_data
// vocabulary: the path it names, the durable_data kind it implies, and the
// durable_data format it implies (empty when the target has no file format).
func retentionTargetShape(kind, database, dirPath string) (targetPath, wantKind, wantFormat string) {
	switch kind {
	case "sqlite_table":
		return normalizeManifestPath(database), "file", "sqlite"
	case "directory":
		return normalizeManifestPath(dirPath), "dir", ""
	default:
		return "", "", ""
	}
}

// normalizeManifestPath slash-cleans a manifest path so comparison is not
// defeated by "./" prefixes or duplicate separators. Manifest paths are always
// slash-normalized by schema, so no backslash handling is needed.
func normalizeManifestPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return strings.Trim(path.Clean(p), "/")
}

// pathsOverlap reports whether two manifest paths anchored at different bases
// name the same location, which is true when one is a path-segment suffix of
// the other.
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

package monetization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MeterInventory is the generated vocabulary consumed by release-ladder
// planning. It is derived from scenario declarations, never hand-maintained.
type MeterInventory struct {
	Source string         `json:"source"`
	Meters []MeterSummary `json:"meters"`
}

type MeterSummary struct {
	LimitKey   string   `json:"limit_key"`
	Class      string   `json:"class"`
	DeclaredBy []string `json:"declared_by"`
	BundleKeys []string `json:"bundle_keys"`
	Byok       bool     `json:"byok,omitempty"`
}

type monetizationDeclaration struct {
	BundleKey string `json:"bundle_key"`
	Meters    []struct {
		LimitKey string `json:"limit_key"`
		Class    string `json:"class"`
		Byok     bool   `json:"byok"`
	} `json:"meters"`
}

// BuildMeterInventory walks scenarios/*/.vrooli/monetization.json beneath
// repoRoot and aggregates each declared limit_key in stable order.
func BuildMeterInventory(repoRoot string) (MeterInventory, error) {
	paths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "monetization.json"))
	if err != nil {
		return MeterInventory{}, fmt.Errorf("find monetization declarations: %w", err)
	}
	byKey := make(map[string]*MeterSummary)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return MeterInventory{}, fmt.Errorf("read %s: %w", path, err)
		}
		var declaration monetizationDeclaration
		if err := json.Unmarshal(data, &declaration); err != nil {
			return MeterInventory{}, fmt.Errorf("parse %s: %w", path, err)
		}
		scenario := filepath.Base(filepath.Dir(filepath.Dir(path)))
		for _, meter := range declaration.Meters {
			if meter.LimitKey == "" || meter.Class == "" {
				return MeterInventory{}, fmt.Errorf("%s contains an incomplete meter", path)
			}
			summary := byKey[meter.LimitKey]
			if summary == nil {
				summary = &MeterSummary{LimitKey: meter.LimitKey, Class: meter.Class}
				byKey[meter.LimitKey] = summary
			} else if summary.Class != meter.Class {
				return MeterInventory{}, fmt.Errorf("meter %q has conflicting classes %q and %q", meter.LimitKey, summary.Class, meter.Class)
			}
			summary.DeclaredBy = appendUnique(summary.DeclaredBy, scenario)
			summary.BundleKeys = appendUnique(summary.BundleKeys, declaration.BundleKey)
			summary.Byok = summary.Byok || meter.Byok
		}
	}
	out := MeterInventory{Source: "scenarios/*/.vrooli/monetization.json", Meters: make([]MeterSummary, 0, len(byKey))}
	for _, summary := range byKey {
		sort.Strings(summary.DeclaredBy)
		sort.Strings(summary.BundleKeys)
		out.Meters = append(out.Meters, *summary)
	}
	sort.Slice(out.Meters, func(i, j int) bool { return out.Meters[i].LimitKey < out.Meters[j].LimitKey })
	return out, nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

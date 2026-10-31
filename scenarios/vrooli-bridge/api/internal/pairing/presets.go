package pairing

import (
	"sort"
	"strings"

	"github.com/vrooli/api-core/scopecatalog"
)

type Preset string

const (
	PresetReadOnly Preset = "read-only"
	PresetOperate  Preset = "operate"
	PresetFull     Preset = "full-control"
)

type PermissionPreset struct {
	Name        Preset
	Description string
	Scopes      []string
	Withholds   []string
}

// PermissionPresets derives concrete grants from the CLI catalog. The default
// is explicit read access only; no preset silently grants the wildcard write
// namespace.
func PermissionPresets(catalog scopecatalog.Catalog) []PermissionPreset {
	read := scopesForEffects(catalog, scopecatalog.EffectRead)
	operate := scopesForEffects(catalog, scopecatalog.EffectRead, scopecatalog.EffectWrite)
	full := scopesForEffects(catalog, scopecatalog.EffectRead, scopecatalog.EffectWrite, scopecatalog.EffectDestructive)
	return []PermissionPreset{
		{Name: PresetReadOnly, Description: "Read status and telemetry", Scopes: read, Withholds: []string{"write", "destructive"}},
		{Name: PresetOperate, Description: "Read and operate governed commands", Scopes: operate, Withholds: []string{"destructive"}},
		{Name: PresetFull, Description: "Read, operate, and destructive commands", Scopes: full},
	}
}

func ScopesForPreset(catalog scopecatalog.Catalog, preset Preset) ([]string, bool) {
	for _, candidate := range PermissionPresets(catalog) {
		if candidate.Name == preset {
			return append([]string(nil), candidate.Scopes...), true
		}
	}
	return nil, false
}

func scopesForEffects(catalog scopecatalog.Catalog, effects ...scopecatalog.Effect) []string {
	allowed := make(map[scopecatalog.Effect]struct{}, len(effects))
	for _, effect := range effects {
		allowed[effect] = struct{}{}
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, scope := range catalog.Scopes {
		if _, ok := allowed[scope.Effect]; !ok {
			continue
		}
		value := strings.TrimSpace(scope.Value)
		if value == "" || strings.Contains(value, "*") {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

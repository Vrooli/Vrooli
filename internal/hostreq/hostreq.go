package hostreq

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type Kind = hostreqspec.Kind

const (
	KindTool      = hostreqspec.KindTool
	KindSafeguard = hostreqspec.KindSafeguard
)

type Declaration = hostreqspec.Declaration

type Provenance = hostreqspec.Provenance

type ResolvedRequirement = hostreqspec.ResolvedRequirement

func normalizeSelector(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

var (
	ValidateDeclarations = hostreqspec.ValidateDeclarations
	CurrentPlatform      = hostreqspec.CurrentPlatform
	NormalizeEnvironment = hostreqspec.NormalizeEnvironment
)

func normalizeCSV(value string) []string {
	parts := strings.Split(value, ",")
	names := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		names = append(names, part)
	}
	sort.Strings(names)
	return names
}

func manifestSourcePath(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

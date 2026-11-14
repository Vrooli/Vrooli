package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

// crossPlatform checks the declaration itself rather than the host filesystem.
// Its findings are advisory: authors can adopt the contract incrementally while
// still getting a precise portability defect and a concrete replacement.
type crossPlatform struct{}

func init() { register(&crossPlatform{}) }

func (crossPlatform) Name() string { return "storage.cross-platform" }

func (crossPlatform) Applies(ac AnalyzerContext) bool {
	return ac.Owner != nil && len(ac.Owner.StorageEntries) > 0
}

func (crossPlatform) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	if ac.Owner == nil {
		return nil, nil
	}
	findings := make([]Finding, 0)
	for _, entry := range ac.Owner.StorageEntries {
		platforms := corestorage.EffectivePlatforms(*ac.Owner, entry)
		if entry.Path.ByOS == nil {
			findings = append(findings, checkBarePath(ac, entry, platforms)...)
		} else {
			findings = append(findings, checkBranches(ac, entry, platforms)...)
		}
	}
	return compactFindings(findings), nil
}

func checkBarePath(ac AnalyzerContext, entry corestorage.StorageEntry, platforms []corestorage.Platform) []Finding {
	path := strings.TrimSpace(entry.Path.Value)
	layout := classifyPathLayout(path)
	if layout == layoutNone {
		return nil
	}
	hasLinuxOrMac, hasWindows := false, false
	for _, platform := range platforms {
		if platform == corestorage.PlatformWindows {
			hasWindows = true
		} else {
			hasLinuxOrMac = true
		}
	}
	findings := make([]Finding, 0, 2)
	if (layout == layoutUnix && hasWindows) || (layout == layoutWindows && hasLinuxOrMac) {
		findings = append(findings, crossFinding(ac, entry, "STORAGE_PATH_NOT_PORTABLE", fmt.Sprintf("a %s-layout absolute path is used across platforms", layout), "Use a portable token, or provide a byOS path map for genuinely divergent locations."))
	}
	if (layout == layoutUnix && !hasLinuxOrMac) || (layout == layoutWindows && !hasWindows) {
		findings = append(findings, crossFinding(ac, entry, "STORAGE_PATH_PLATFORM_MISMATCH", fmt.Sprintf("a %s-layout path is incompatible with the declared platform set %v", layout, platforms), "Narrow the owner or entry platforms, or declare a path whose syntax matches the target platform."))
	}
	return findings
}

func checkBranches(ac AnalyzerContext, entry corestorage.StorageEntry, platforms []corestorage.Platform) []Finding {
	findings := make([]Finding, 0)
	for _, platform := range platforms {
		if _, ok := entry.Path.ByOS[platform]; !ok {
			findings = append(findings, crossFinding(ac, entry, "STORAGE_PATH_BRANCH_MISSING", fmt.Sprintf("byOS path has no branch for declared platform %s", platform), fmt.Sprintf("Add a %q branch, or add an explicit null branch when the entry is intentionally absent there.", platform)))
		}
	}
	if replacement, ok := supersedableToken(entry, platforms); ok {
		findings = append(findings, crossFinding(ac, entry, "STORAGE_TOKEN_SUPERSEDABLE", "every platform branch follows the same portable directory convention", "Replace the repeated byOS map with "+replacement+" and keep byOS only for genuinely divergent layouts."))
	}
	return findings
}

type storagePathLayout string

const (
	layoutNone    storagePathLayout = ""
	layoutUnix    storagePathLayout = "Unix"
	layoutWindows storagePathLayout = "Windows"
)

func classifyPathLayout(value string) storagePathLayout {
	if strings.HasPrefix(value, "/") {
		return layoutUnix
	}
	if strings.HasPrefix(value, `\\`) || (len(value) >= 3 && isASCIILetter(value[0]) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')) || strings.Contains(value, `\`) {
		return layoutWindows
	}
	return layoutNone
}

func isASCIILetter(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func supersedableToken(entry corestorage.StorageEntry, platforms []corestorage.Platform) (string, bool) {
	if len(platforms) == 0 || entry.Path.ByOS == nil {
		return "", false
	}
	// Prefer durable per-user data, then the other portable conventions. The
	// data/config directories intentionally coincide on macOS and Windows, so
	// this order keeps durable resource data on $USER_DATA_DIR.
	tokens := []struct {
		name string
		base func(corestorage.Platform, corestorage.UserIdentity) string
	}{
		{"$USER_DATA_DIR", func(platform corestorage.Platform, identity corestorage.UserIdentity) string {
			return seamValue(platform, identity, "data")
		}},
		{"$USER_CONFIG_DIR", func(platform corestorage.Platform, identity corestorage.UserIdentity) string {
			return seamValue(platform, identity, "config")
		}},
		{"$USER_CACHE_DIR", func(platform corestorage.Platform, identity corestorage.UserIdentity) string {
			return seamValue(platform, identity, "cache")
		}},
		{"$USER_STATE_DIR", func(platform corestorage.Platform, identity corestorage.UserIdentity) string {
			return seamValue(platform, identity, "state")
		}},
	}
	for _, token := range tokens {
		var suffix string
		matched := true
		for _, platform := range platforms {
			branch, ok := entry.Path.ByOS[platform]
			if !ok || branch == nil {
				matched = false
				break
			}
			identity := corestorage.SyntheticIdentity(platform)
			resolved, err := corestorage.ResolvePortablePath(entry.Name, corestorage.PortablePath{Value: *branch}, platform, corestorage.DefaultSeams(platform, identity))
			if err != nil {
				matched = false
				break
			}
			current, ok := relativeToBase(platform, resolved, token.base(platform, identity))
			if !ok {
				matched = false
				break
			}
			if suffix == "" {
				suffix = current
			} else if normalizePathForComparison(platform, suffix) != normalizePathForComparison(platform, current) {
				matched = false
				break
			}
		}
		if matched {
			return token.name, true
		}
	}
	return "", false
}

func seamValue(platform corestorage.Platform, identity corestorage.UserIdentity, kind string) string {
	seams := corestorage.DefaultSeams(platform, identity)
	var value string
	switch kind {
	case "config":
		value, _ = seams.UserConfigDir()
	case "cache":
		value, _ = seams.UserCacheDir()
	case "state":
		value, _ = seams.UserStateDir()
	case "data":
		if platform == corestorage.PlatformWindows {
			value = strings.TrimRight(identity.HomeDir, `/\`) + `\AppData\Local`
		} else if platform == corestorage.PlatformMacOS {
			value = filepath.Join(identity.HomeDir, "Library", "Application Support")
		} else {
			value = filepath.Join(identity.HomeDir, ".local", "share")
		}
	}
	return value
}

func relativeToBase(platform corestorage.Platform, resolved, base string) (string, bool) {
	left := normalizePathForComparison(platform, resolved)
	right := strings.TrimRight(normalizePathForComparison(platform, base), "/")
	if left == right {
		return "", true
	}
	if !strings.HasPrefix(left, right+"/") {
		return "", false
	}
	return strings.TrimPrefix(left, right+"/"), true
}

func normalizePathForComparison(platform corestorage.Platform, value string) string {
	value = strings.ReplaceAll(value, `\`, "/")
	if platform == corestorage.PlatformWindows {
		value = strings.ToLower(value)
	}
	return filepath.ToSlash(value)
}

func crossFinding(ac AnalyzerContext, entry corestorage.StorageEntry, code, detail, remediation string) Finding {
	location := ""
	if ac.Owner != nil {
		location = filepath.ToSlash(ac.Owner.ManifestPath)
		if ac.RepoRoot != "" {
			if relative, err := filepath.Rel(ac.RepoRoot, ac.Owner.ManifestPath); err == nil {
				location = filepath.ToSlash(relative)
			}
		}
	}
	severity := SeverityWarning
	if code == "STORAGE_PATH_NOT_PORTABLE" || code == "STORAGE_PATH_PLATFORM_MISMATCH" || code == "STORAGE_PATH_BRANCH_MISSING" {
		severity = SeverityError
	}
	return Finding{
		Code:        code,
		Severity:    severity,
		Title:       "Cross-platform storage declaration",
		Message:     fmt.Sprintf("storage entry %q at %q: %s", entry.Name, displayPortablePath(entry.Path), detail),
		Location:    location,
		Remediation: remediation,
		Analyzer:    "storage.cross-platform",
	}
}

func displayPortablePath(path corestorage.PortablePath) string {
	if path.ByOS == nil {
		return path.Value
	}
	platforms := make([]string, 0, len(path.ByOS))
	for platform := range path.ByOS {
		platforms = append(platforms, string(platform))
	}
	sort.Strings(platforms)
	branches := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		value := path.ByOS[corestorage.Platform(platform)]
		if value == nil {
			branches = append(branches, platform+":null")
		} else {
			branches = append(branches, platform+":"+*value)
		}
	}
	return "byOS{" + strings.Join(branches, ", ") + "}"
}

func compactFindings(findings []Finding) []Finding {
	result := findings[:0]
	for _, finding := range findings {
		if finding.Code != "" {
			result = append(result, finding)
		}
	}
	return result
}

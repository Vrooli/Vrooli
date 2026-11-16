package versions

import (
	"sort"
	"strconv"
	"strings"
)

// CompareVersionCoreLabels preserves the legacy ordering used by adoption
// status: compare numeric major/minor/patch parts and ignore suffixes.
func CompareVersionCoreLabels(a, b string) (int, bool) {
	left, ok := parseVersionCore(a)
	if !ok {
		return 0, false
	}
	right, ok := parseVersionCore(b)
	if !ok {
		return 0, false
	}
	for i := range left {
		if left[i] < right[i] {
			return -1, true
		}
		if left[i] > right[i] {
			return 1, true
		}
	}
	return 0, true
}

func parseVersionCore(raw string) ([3]int, bool) {
	var values [3]int
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return values, false
	}
	if i := strings.IndexAny(raw, "-+"); i >= 0 {
		raw = raw[:i]
	}
	for i, part := range strings.SplitN(raw, ".", 3) {
		part = strings.TrimSpace(part)
		if part == "" || part == "x" || part == "X" || part == "*" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return values, false
		}
		values[i] = value
	}
	return values, true
}

type semanticVersion struct {
	major, minor, patch int
	pre                 []string
}

func parseSemanticVersion(raw string) (semanticVersion, bool) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if raw == "" {
		return semanticVersion{}, false
	}
	parts := strings.SplitN(raw, "+", 2)
	coreAndPre := strings.SplitN(parts[0], "-", 2)
	core := strings.Split(coreAndPre[0], ".")
	if len(core) != 3 {
		return semanticVersion{}, false
	}
	values := [3]int{}
	for i := range core {
		value, err := strconv.Atoi(core[i])
		if err != nil || value < 0 {
			return semanticVersion{}, false
		}
		values[i] = value
	}
	out := semanticVersion{major: values[0], minor: values[1], patch: values[2]}
	if len(coreAndPre) == 2 {
		if coreAndPre[1] == "" {
			return semanticVersion{}, false
		}
		out.pre = strings.Split(coreAndPre[1], ".")
		for _, identifier := range out.pre {
			if identifier == "" {
				return semanticVersion{}, false
			}
		}
	}
	return out, true
}

// compareVersionLabels implements semver precedence. A released version is
// greater than its pre-release variants, so a stable version remains above
// draft.1 even when both share the same numeric core.
func compareVersionLabels(a, b string) int {
	left, leftOK := parseSemanticVersion(a)
	right, rightOK := parseSemanticVersion(b)
	if !leftOK || !rightOK {
		return strings.Compare(a, b)
	}
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(left.pre) == 0 && len(right.pre) != 0 {
		return 1
	}
	if len(left.pre) != 0 && len(right.pre) == 0 {
		return -1
	}
	for i := 0; i < len(left.pre) && i < len(right.pre); i++ {
		l, r := left.pre[i], right.pre[i]
		if l == r {
			continue
		}
		ln, lNumericErr := strconv.Atoi(l)
		rn, rNumericErr := strconv.Atoi(r)
		switch {
		case lNumericErr == nil && rNumericErr == nil:
			if ln < rn {
				return -1
			}
			return 1
		case lNumericErr == nil:
			return -1
		case rNumericErr == nil:
			return 1
		default:
			return strings.Compare(l, r)
		}
	}
	if len(left.pre) < len(right.pre) {
		return -1
	}
	if len(left.pre) > len(right.pre) {
		return 1
	}
	return 0
}

func sortVersions(rows []Version) {
	sort.SliceStable(rows, func(i, j int) bool {
		if cmp := compareVersionLabels(rows[i].Version, rows[j].Version); cmp != 0 {
			return cmp > 0
		}
		return rows[i].ID < rows[j].ID
	})
}

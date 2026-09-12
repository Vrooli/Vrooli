package dependencygovernance

import (
	"strconv"
	"strings"
)

type versionPolicyDecision struct {
	Allowed      bool
	FindingClass string
	Title        string
	Description  string
	Remediation  string
	Expected     string
}

func evaluateVersionPolicy(ecosystem, observed, approvedRange, rangePolicy string) versionPolicyDecision {
	approvedRange = strings.TrimSpace(approvedRange)
	observed = strings.TrimSpace(observed)
	policy := normalize(rangePolicy)
	if approvedRange == "" || approvedRange == "*" || observed == "" || observed == approvedRange {
		return versionPolicyDecision{Allowed: true}
	}
	if policy == "" {
		policy = defaultRangePolicy(approvedRange)
	}
	if !supportedRangePolicy(policy) {
		return unparseableVersionDecision(observed, approvedRange, "unsupported range policy "+policy)
	}
	switch normalize(ecosystem) {
	case "npm", "go":
	default:
		return outOfRangeDecision(approvedRange)
	}

	var allowed bool
	var ok bool
	switch policy {
	case "exact":
		allowed, ok = exactVersionAllowed(approvedRange, observed)
	case "major_line", "dev_tooling":
		allowed, ok = majorLineAllowed(approvedRange, observed)
	case "minimum":
		allowed, ok = minimumVersionAllowed(approvedRange, observed)
	case "security_denied":
		allowed, ok = rangeAllowsVersionChecked(approvedRange, observed)
	default:
		return unparseableVersionDecision(observed, approvedRange, "unsupported range policy "+policy)
	}
	if !ok {
		return unparseableVersionDecision(observed, approvedRange, "could not parse version/range")
	}
	if allowed {
		return versionPolicyDecision{Allowed: true}
	}
	return outOfRangeDecision(approvedRange)
}

func defaultRangePolicy(approvedRange string) string {
	if strings.ContainsAny(approvedRange, "<>^~*| ") {
		return "minimum"
	}
	return "exact"
}

func supportedRangePolicy(policy string) bool {
	switch policy {
	case "", "exact", "major_line", "minimum", "dev_tooling", "security_denied":
		return true
	default:
		return false
	}
}

func exactVersionAllowed(approvedRange, observed string) (bool, bool) {
	if strings.ContainsAny(approvedRange, "<>^~*| ") {
		return rangeAllowsVersionChecked(approvedRange, observed)
	}
	observedVersion, ok := parseVersion(firstVersionToken(observed))
	if !ok {
		return false, false
	}
	want, ok := parseVersion(approvedRange)
	return ok && compareVersion(observedVersion, want) == 0, ok
}

func majorLineAllowed(approvedRange, observed string) (bool, bool) {
	observedVersion, ok := parseVersion(firstVersionToken(observed))
	if !ok {
		return false, false
	}
	base, ok := parseVersion(firstVersionToken(approvedRange))
	if !ok {
		return false, false
	}
	return compareVersion(observedVersion, base) >= 0 && observedVersion.Major == base.Major, true
}

func minimumVersionAllowed(approvedRange, observed string) (bool, bool) {
	if strings.Contains(approvedRange, "<") || strings.Contains(approvedRange, ">") || strings.Contains(approvedRange, "||") {
		return rangeAllowsVersionChecked(approvedRange, observed)
	}
	return majorLineAllowed(approvedRange, observed)
}

func rangeAllowsVersionChecked(constraint, observed string) (bool, bool) {
	observedVersion, ok := parseVersion(firstVersionToken(observed))
	if !ok {
		return false, false
	}
	parsedAny := false
	for _, clause := range strings.Split(constraint, "||") {
		allowed, parsed := clauseAllowsVersionChecked(strings.TrimSpace(clause), observedVersion)
		parsedAny = parsedAny || parsed
		if parsed && allowed {
			return true, true
		}
	}
	return false, parsedAny
}

func clauseAllowsVersionChecked(clause string, observed semanticVersion) (bool, bool) {
	if clause == "" || clause == "*" {
		return true, true
	}
	tokens := splitConstraintTokens(clause)
	if len(tokens) == 0 {
		return false, false
	}
	for _, token := range tokens {
		allowed, parsed := constraintTokenAllowsVersionChecked(token, observed)
		if !parsed || !allowed {
			return false, parsed
		}
	}
	return true, true
}

func constraintTokenAllowsVersionChecked(token string, observed semanticVersion) (bool, bool) {
	token = strings.TrimSpace(token)
	if token == "" || token == "*" || token == "x" || token == "X" {
		return true, true
	}
	if strings.HasPrefix(token, "^") {
		base, ok := parseVersion(strings.TrimPrefix(token, "^"))
		return ok && compareVersion(observed, base) >= 0 && observed.Major == base.Major, ok
	}
	if strings.HasPrefix(token, "~") {
		base, ok := parseVersion(strings.TrimPrefix(token, "~"))
		return ok && compareVersion(observed, base) >= 0 && observed.Major == base.Major && observed.Minor == base.Minor, ok
	}
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(token, op) {
			want, ok := parseVersion(strings.TrimPrefix(token, op))
			if !ok {
				return false, false
			}
			cmp := compareVersion(observed, want)
			switch op {
			case ">=":
				return cmp >= 0, true
			case "<=":
				return cmp <= 0, true
			case ">":
				return cmp > 0, true
			case "<":
				return cmp < 0, true
			case "=":
				return cmp == 0, true
			}
		}
	}
	want, ok := parseVersion(token)
	return ok && compareVersion(observed, want) == 0, ok
}

func outOfRangeDecision(approvedRange string) versionPolicyDecision {
	return versionPolicyDecision{
		FindingClass: "VERSION_OUT_OF_RANGE",
		Title:        "Dependency version is outside recorded approval",
		Description:  "The dependency is recorded, but the observed version/range does not match the approved range policy.",
		Remediation:  "Review whether the observed version should be approved, constrained, or changed.",
		Expected:     firstNonEmpty(approvedRange, "recorded approved range"),
	}
}

func unparseableVersionDecision(observed, approvedRange, reason string) versionPolicyDecision {
	return versionPolicyDecision{
		FindingClass: "VERSION_RANGE_UNPARSEABLE",
		Title:        "Dependency version range could not be evaluated",
		Description:  "The dependency is recorded, but SDA could not safely evaluate the observed version against the approved range: " + reason + ".",
		Remediation:  "Normalize the approved version range or observed dependency declaration to a supported semantic version/range.",
		Expected:     firstNonEmpty(approvedRange, "parseable approved range") + " for observed " + firstNonEmpty(observed, "version"),
	}
}

type semanticVersion struct {
	Major int
	Minor int
	Patch int
}

func firstVersionToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == ',' || r == '|'
	})
	if len(fields) == 0 {
		return value
	}
	return fields[0]
}

func parseVersion(value string) (semanticVersion, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "^~<>= ")
	value = strings.TrimPrefix(value, "v")
	value = strings.SplitN(value, "-", 2)[0]
	value = strings.SplitN(value, "+", 2)[0]
	if value == "" || value == "*" {
		return semanticVersion{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return semanticVersion{}, false
	}
	nums := []int{0, 0, 0}
	for i, part := range parts {
		if part == "x" || part == "X" || part == "*" {
			nums[i] = 0
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return semanticVersion{}, false
		}
		nums[i] = n
	}
	return semanticVersion{Major: nums[0], Minor: nums[1], Patch: nums[2]}, true
}

func compareVersion(left, right semanticVersion) int {
	if left.Major != right.Major {
		return left.Major - right.Major
	}
	if left.Minor != right.Minor {
		return left.Minor - right.Minor
	}
	return left.Patch - right.Patch
}

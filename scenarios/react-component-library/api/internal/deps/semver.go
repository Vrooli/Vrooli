package deps

import (
	"fmt"
	"strconv"
	"strings"
)

// semver is the parsed major.minor.patch we operate on. Pre-release
// tags are deliberately dropped — the library declares ranges over
// public APIs, not pre-release identities.
type semver struct {
	major, minor, patch int
}

// parseVersion accepts a leading-cleaned version string (no range
// operator) and returns its triple. Missing minor / patch default to 0
// so "18" parses as 18.0.0, matching what package.json fields often
// carry.
func parseVersion(raw string) (semver, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return semver{}, fmt.Errorf("empty version")
	}
	if i := strings.IndexAny(raw, "-+"); i >= 0 {
		raw = raw[:i]
	}
	parts := strings.SplitN(raw, ".", 3)
	out := semver{}
	dst := []*int{&out.major, &out.minor, &out.patch}
	for i, p := range parts {
		if p == "" || p == "x" || p == "X" || p == "*" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, fmt.Errorf("invalid semver part %q: %w", p, err)
		}
		*dst[i] = v
	}
	return out, nil
}

// parseTarget extracts a single semver from a package.json version
// string. The string may carry a range operator (^, ~, >=) which we
// strip — packages typically pin a single advertised version even when
// the operator allows newer.
func parseTarget(raw string) (semver, error) {
	raw = strings.TrimSpace(raw)
	for _, prefix := range []string{">=", ">", "<=", "<", "=", "~", "^"} {
		if strings.HasPrefix(raw, prefix) {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, prefix))
			break
		}
	}
	// "1.x" / "1.*" → 1.0.0
	if raw == "" || raw == "*" {
		return semver{}, nil // matches anything; caller handles
	}
	return parseVersion(raw)
}

// classify is the semver intersection at the heart of ValidateAdoption.
// Returns (IssueKind, detail) describing how target satisfies (or
// fails) the declared range. Issue kind "" with empty detail means
// "fully satisfied".
//
// Supported range operators:
//
//	^X.Y.Z   — same major; minor/patch >= Y.Z
//	~X.Y.Z   — same major.minor; patch >= Z
//	>=X.Y.Z  — target >= X.Y.Z componentwise
//	X.Y.Z    — exact match
//	*        — anything
//
// Anything else returns IssueUnparseableRange so the operator sees a
// clear "I don't understand this range" issue rather than silent block.
func classify(declaredRange, targetRaw string) (IssueKind, string) {
	declaredRange = strings.TrimSpace(declaredRange)
	if declaredRange == "" || declaredRange == "*" {
		return "", ""
	}
	target, terr := parseTarget(targetRaw)
	if terr != nil {
		return IssueUnparseableTarget, fmt.Sprintf("target %q is not a valid semver", targetRaw)
	}
	switch {
	case strings.HasPrefix(declaredRange, "^"):
		want, err := parseVersion(strings.TrimPrefix(declaredRange, "^"))
		if err != nil {
			return IssueUnparseableRange, err.Error()
		}
		if target.major != want.major {
			return IssueIncompatibleMajor, fmt.Sprintf("declared ^%d.x.x but target is %d.x.x", want.major, target.major)
		}
		if target.minor < want.minor || (target.minor == want.minor && target.patch < want.patch) {
			return IssueRangeDoesNotMatch, fmt.Sprintf("declared ^%d.%d.%d, target is %d.%d.%d (below minimum)", want.major, want.minor, want.patch, target.major, target.minor, target.patch)
		}
		return "", ""
	case strings.HasPrefix(declaredRange, "~"):
		want, err := parseVersion(strings.TrimPrefix(declaredRange, "~"))
		if err != nil {
			return IssueUnparseableRange, err.Error()
		}
		if target.major != want.major {
			return IssueIncompatibleMajor, fmt.Sprintf("declared ~%d.%d.%d but target major is %d", want.major, want.minor, want.patch, target.major)
		}
		if target.minor != want.minor {
			return IssueRangeDoesNotMatch, fmt.Sprintf("declared ~%d.%d.%d, target minor is %d", want.major, want.minor, want.patch, target.minor)
		}
		if target.patch < want.patch {
			return IssueRangeDoesNotMatch, fmt.Sprintf("declared ~%d.%d.%d, target patch is %d (below minimum)", want.major, want.minor, want.patch, target.patch)
		}
		return "", ""
	case strings.HasPrefix(declaredRange, ">="):
		want, err := parseVersion(strings.TrimPrefix(declaredRange, ">="))
		if err != nil {
			return IssueUnparseableRange, err.Error()
		}
		if compareSemver(target, want) < 0 {
			if target.major < want.major {
				return IssueIncompatibleMajor, fmt.Sprintf("declared >=%d.%d.%d, target is %d.%d.%d", want.major, want.minor, want.patch, target.major, target.minor, target.patch)
			}
			return IssueRangeDoesNotMatch, fmt.Sprintf("declared >=%d.%d.%d, target is %d.%d.%d", want.major, want.minor, want.patch, target.major, target.minor, target.patch)
		}
		return "", ""
	default:
		// Treat as exact pin.
		want, err := parseVersion(declaredRange)
		if err != nil {
			return IssueUnparseableRange, fmt.Sprintf("range %q not understood", declaredRange)
		}
		if compareSemver(target, want) != 0 {
			if target.major != want.major {
				return IssueIncompatibleMajor, fmt.Sprintf("declared exact %s, target is %d.%d.%d", declaredRange, target.major, target.minor, target.patch)
			}
			return IssueRangeDoesNotMatch, fmt.Sprintf("declared exact %s, target is %d.%d.%d", declaredRange, target.major, target.minor, target.patch)
		}
		return "", ""
	}
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}

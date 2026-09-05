// Package version owns the single pinned kopia version and the parsing /
// compatibility checks for the version string reported by the kopia binary.
//
// The pinned version is the single source of truth for the resource. The
// manifest's install.platforms block must reference the same version; a test
// (TestManifestReferencesPinnedVersion) asserts resource.json contains Pinned
// so the literal cannot drift between the Go code and the manifest.
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Pinned is the kopia version this resource provisions and validates against.
// Stored without a leading "v" because `kopia --version` reports e.g.
// "kopia version 0.23.0 build: ...".
const Pinned = "0.23.0"

// Tag is the GitHub release tag form of the pinned version.
const Tag = "v" + Pinned

// semverRe extracts the first dotted numeric version from arbitrary output.
var semverRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Semver is a parsed major.minor.patch triple.
type Semver struct {
	Major int
	Minor int
	Patch int
}

// String renders the triple as "major.minor.patch".
func (s Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}

// AtLeast reports whether s is greater than or equal to other, compared by
// major then minor then patch.
func (s Semver) AtLeast(other Semver) bool {
	switch {
	case s.Major != other.Major:
		return s.Major > other.Major
	case s.Minor != other.Minor:
		return s.Minor > other.Minor
	default:
		return s.Patch >= other.Patch
	}
}

// Parse extracts a Semver from a version string such as the output of
// `kopia --version` ("kopia version 0.23.0 build: ...") or a bare "0.23.0".
func Parse(output string) (Semver, error) {
	match := semverRe.FindStringSubmatch(output)
	if match == nil {
		return Semver{}, fmt.Errorf("no version found in %q", strings.TrimSpace(output))
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return Semver{Major: major, Minor: minor, Patch: patch}, nil
}

// PinnedSemver returns the pinned version parsed as a Semver.
func PinnedSemver() Semver {
	s, _ := Parse(Pinned)
	return s
}

// Check parses the version output and reports whether the installed kopia
// satisfies the pinned version (installed major.minor.patch >= pinned). The
// parsed installed version is returned for messaging even on failure.
func Check(output string) (Semver, bool, error) {
	installed, err := Parse(output)
	if err != nil {
		return Semver{}, false, err
	}
	return installed, installed.AtLeast(PinnedSemver()), nil
}

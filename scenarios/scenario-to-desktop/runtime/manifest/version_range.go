package manifest

import "github.com/vrooli/platform-go/versionrange"

// VersionSatisfiesRange evaluates the numeric range constraints used by the
// scenario dependency contract.
func VersionSatisfiesRange(version, constraint string) bool {
	return versionrange.Satisfies(version, constraint)
}

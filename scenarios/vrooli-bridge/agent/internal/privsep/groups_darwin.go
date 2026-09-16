//go:build darwin

package privsep

// maxSupplementaryGroups is macOS's NGROUPS_MAX: an exec whose credential
// carries more groups is refused with EINVAL.
const maxSupplementaryGroups = 16

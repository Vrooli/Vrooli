//go:build !darwin && !windows

package privsep

// maxSupplementaryGroups is unlimited outside macOS; Linux allows far more
// supplementary groups than any account here carries.
const maxSupplementaryGroups = 0

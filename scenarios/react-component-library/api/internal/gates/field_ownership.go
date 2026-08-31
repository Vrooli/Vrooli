package gates

import (
	"regexp"
)

var ownedHeaderRE = regexp.MustCompile(`(?m)^\s*\*?\s*@description\b`)

// ValidateFieldOwnership keeps catalog-owned metadata in the catalog. A
// manifest is still a useful generated pointer, but carrying a second copy
// of the same identity or presentation fact makes the generator's direction
// ambiguous and permits drift.

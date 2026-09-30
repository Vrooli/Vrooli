package signals_test

import (
	"architecture-cartographer/internal/domains"
)

// emptyDomainMap returns an empty derived domain map used by aggregator
// tests that don't care about domain membership.
func emptyDomainMap() domains.DerivedDomainMap {
	return domains.DerivedDomainMap{Scenario: "demo"}
}

package facts

import "regexp"

// invariantScenarioPattern validates the one scenario slug accepted by the
// bounded fact search handlers before any source path is resolved.
var invariantScenarioPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

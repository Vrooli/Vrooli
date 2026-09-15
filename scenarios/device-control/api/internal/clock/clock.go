// Package clock exposes the scenario's time seam at the package boundary while
// retaining api-core's canonical timer contract.
package clock

import "github.com/vrooli/api-core/schedule"

// Clock is the shared injectable time contract used by device-control's
// cross-cutting services and tests.
type Clock interface{ schedule.Clock }

// System returns the production clock.
func System() Clock { return schedule.System() }

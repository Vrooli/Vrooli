package facts

import "fmt"

// FamilyCost records the measured wall-clock cost of a family on a named
// target. Keeping the measurements explicit makes regressions reviewable and
// lets tests assert that an intentionally lowered budget fails loudly.
type FamilyCost struct {
	Family string
	Target string
	ColdMS int64
	WarmMS int64
}

func AssertFamilyCost(cost FamilyCost, coldBudgetMS, warmBudgetMS int64) error {
	if cost.ColdMS > coldBudgetMS {
		return fmt.Errorf("%s cold cost %dms exceeds %dms budget", cost.Family, cost.ColdMS, coldBudgetMS)
	}
	if cost.WarmMS > warmBudgetMS {
		return fmt.Errorf("%s warm cost %dms exceeds %dms budget", cost.Family, cost.WarmMS, warmBudgetMS)
	}
	return nil
}

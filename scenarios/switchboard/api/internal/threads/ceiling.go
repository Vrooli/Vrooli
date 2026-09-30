package threads

import "switchboard/internal/trust"

func RoomCeiling(roster []trust.Tier) trust.Tier {
	if len(roster) == 0 {
		return trust.Stranger
	}
	result := roster[0]
	for _, tier := range roster[1:] {
		if tier < result {
			result = tier
		}
	}
	return result
}

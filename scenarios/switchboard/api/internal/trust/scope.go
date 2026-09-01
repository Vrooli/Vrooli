package trust

import "sort"

type Tier int

const (
	Stranger Tier = iota
	Known
	Trusted
	Owner
)

func (t Tier) String() string { return []string{"stranger", "known", "trusted", "owner"}[t] }

type (
	Grant      struct{ Scopes []string }
	Resolution struct {
		Scopes  []string
		Refused bool
		Reason  string
	}
)

// Resolve intersects the sender/room authority with the agent grant. A scope
// is never widened and owner-only scopes cannot cross a lower tier boundary.
func Resolve(sender, ceiling Tier, grant Grant) Resolution {
	if sender < 0 || ceiling < 0 {
		return Resolution{Refused: true, Reason: "invalid trust tier"}
	}
	limit := sender
	if ceiling < limit {
		limit = ceiling
	}
	if limit < Owner {
		return Resolution{Scopes: filterByTier(grant.Scopes, limit), Reason: "scope attenuated to " + limit.String()}
	}
	return Resolution{Scopes: unique(grant.Scopes), Reason: "scope granted"}
}

func filterByTier(scopes []string, tier Tier) []string {
	out := []string{}
	for _, s := range scopes {
		if s != "owner" && tier >= Known {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return Resolution{}.Scopes
	}
	return unique(out)
}

func unique(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	j := 0
	for i := range out {
		if i == 0 || out[i] != out[i-1] {
			out[j] = out[i]
			j++
		}
	}
	return out[:j]
}

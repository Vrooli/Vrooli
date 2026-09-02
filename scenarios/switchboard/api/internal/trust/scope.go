package trust

import (
	"fmt"
	"sort"
	"strings"
)

type Tier int

const (
	Stranger Tier = iota
	Known
	Trusted
	Owner
)

func (t Tier) String() string {
	if t < Stranger || t > Owner {
		return "stranger"
	}
	return []string{"stranger", "known", "trusted", "owner"}[t]
}

// Tiers lists every tier in ascending rank order.
var Tiers = []Tier{Stranger, Known, Trusted, Owner}

// ParseTier converts a stored tier name back to its rank. Unknown names are an
// error rather than a default, so a typo can never widen a contact.
func ParseTier(name string) (Tier, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "stranger":
		return Stranger, nil
	case "known":
		return Known, nil
	case "trusted":
		return Trusted, nil
	case "owner":
		return Owner, nil
	}
	return Stranger, fmt.Errorf("unknown trust tier %q", name)
}

// OwnerOnly reports whether a scope can only ever be exercised by the owner
// tier. It is the single definition shared by scope resolution and the
// console, so the UI never renders such a scope as a movable control.
func OwnerOnly(scope string) bool {
	s := strings.ToLower(strings.TrimSpace(scope))
	return s == "owner" || strings.HasPrefix(s, "owner:") || strings.HasSuffix(s, ":owner-only")
}

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
		if !OwnerOnly(s) && tier >= Known {
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

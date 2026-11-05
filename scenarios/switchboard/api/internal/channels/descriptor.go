package channels

import (
	"fmt"
	"regexp"
	"strings"
)

type Descriptor struct {
	Kind          string        `json:"kind"`
	SchemaVersion int           `json:"schemaVersion"`
	ID            string        `json:"id"`
	DisplayName   string        `json:"displayName"`
	Transport     string        `json:"transport"`
	Supports      Supports      `json:"supports"`
	Limits        Limits        `json:"limits"`
	Setup         Setup         `json:"setup"`
	Cost          string        `json:"cost"`
	Requires      []Requirement `json:"requires,omitempty"`
	// Trust answers the policy question "what tier does a sender on this
	// channel hold before anyone assigns one?" from data, so no core code
	// has to know which channel is the owner's own authenticated surface.
	Trust *TrustPolicy `json:"trust,omitempty"`
	// Accent is the channel's brand colour for rendering. It travels with
	// every channel reference so consumers never map colours by id.
	Accent string `json:"accent,omitempty"`
}

// TrustPolicy is the descriptor's trust block. DefaultTier names the tier a
// previously unseen sender holds; it is one of stranger, known, trusted, owner.
type TrustPolicy struct {
	DefaultTier string `json:"defaultTier"`
}

// DefaultTier returns the tier an unknown sender on this channel starts at.
// An absent block fails closed to stranger.
func (d Descriptor) DefaultTier() string {
	if d.Trust == nil || strings.TrimSpace(d.Trust.DefaultTier) == "" {
		return "stranger"
	}
	return strings.TrimSpace(d.Trust.DefaultTier)
}

var validTiers = map[string]struct{}{"stranger": {}, "known": {}, "trusted": {}, "owner": {}}

var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type Supports struct {
	Text    bool `json:"text"`
	Images  bool `json:"images"`
	Files   bool `json:"files"`
	Groups  bool `json:"groups"`
	Threads bool `json:"threads"`
}
type Limits struct {
	MaxTextBytes  int64 `json:"maxTextBytes"`
	MaxMediaBytes int64 `json:"maxMediaBytes"`
}
type Setup struct {
	Friction int `json:"friction"`
}
type Requirement struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func (d Descriptor) Validate() error {
	checks := []struct{ field, value string }{
		{"kind", d.Kind},
		{"schemaVersion", fmt.Sprint(d.SchemaVersion)},
		{"id", d.ID},
		{"displayName", d.DisplayName},
		{"transport", d.Transport},
		{"cost", d.Cost},
	}
	for _, c := range checks {
		if c.value == "" || (c.field == "schemaVersion" && c.value == "0") {
			return fmt.Errorf("field %s is required", c.field)
		}
	}
	if d.Limits.MaxTextBytes < 0 || d.Limits.MaxMediaBytes < 0 {
		return fmt.Errorf("field limits must be non-negative")
	}
	if d.Setup.Friction < 0 {
		return fmt.Errorf("field setup.friction must be non-negative")
	}
	if _, ok := validTiers[d.DefaultTier()]; !ok {
		return fmt.Errorf("field trust.defaultTier must be one of stranger, known, trusted, owner")
	}
	if d.Accent != "" && !accentPattern.MatchString(d.Accent) {
		return fmt.Errorf("field accent must be a #rrggbb hex colour")
	}
	return nil
}

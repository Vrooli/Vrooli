package teamconfig

import (
	"fmt"
	"strings"
)

// FiniteLeader binds one coordinator to an accepted effort. References are
// context, not executable files or authority. ProfileKey lives on the heartbeat.
type FiniteLeader struct {
	EffortRef            string   `json:"effortRef"`
	AcceptedRevision     string   `json:"acceptedRevision"`
	CoordinatorPromptRef string   `json:"coordinatorPromptRef"`
	SourceRefs           []string `json:"sourceRefs"`
	Retired              bool     `json:"retired,omitempty"`
}

func (b *FiniteLeader) Validate(profileKey string, supervision *Supervision) error {
	if b == nil {
		return nil
	}
	if supervision != nil {
		return fmt.Errorf("finiteLeader and standing supervision are mutually exclusive")
	}
	for _, value := range append([]string{b.EffortRef, b.AcceptedRevision, b.CoordinatorPromptRef, profileKey}, b.SourceRefs...) {
		if value == "" || strings.TrimSpace(value) != value || len(value) > 1024 || strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("finiteLeader requires exact bounded effort, acceptance, coordinator prompt, source references and profileKey")
		}
	}
	if len(b.SourceRefs) < 1 || len(b.SourceRefs) > 8 {
		return fmt.Errorf("finiteLeader.sourceRefs requires 1..8 references")
	}
	return nil
}

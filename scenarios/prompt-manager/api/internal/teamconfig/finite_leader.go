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
	// KeepAlive declares liveness semantics for a long-lived delivery
	// orchestrator (large-effort-orchestration): the heartbeat skips while the
	// leader run is live or parked and relaunches it, with backoff and a cap,
	// when it is terminal while the effort is open. Without it the leader keeps
	// one dispatch per effort. It is policy, not binding identity.
	KeepAlive bool `json:"keepAlive,omitempty"`
}

func (b *FiniteLeader) Validate(profileKey string) error {
	if b == nil {
		return nil
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

package heartbeat

import (
	"fmt"

	"prompt-manager/internal/store"
)

// holdFiniteLeaderForCaller makes a caller refusal on a retained dispatch
// visible in the leader's own state. A scheduled tick carries no human or
// effort proof, so under AUTH-01 it cannot replay or relaunch the leader; the
// projection must say so rather than keep a stale owner error. The reservation
// is unchanged, and the refusal is still returned to the caller.
func holdFiniteLeaderForCaller(state *store.FiniteLeaderState, save func() error, refusal error) error {
	held := fmt.Sprintf("finite leader held: %v; resume with an operator heartbeat-trigger that carries a verified human caller", refusal)
	if state.Error != held {
		state.Error = held
		if err := save(); err != nil {
			return err
		}
	}
	return refusal
}

package heartbeat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

// WakeSignal is the bounded identity of the inputs a heartbeat declared as
// wake-worthy. It is an identity, not a prompt or an authority grant.
type WakeSignal struct {
	Key     string            `json:"key"`
	Sources map[string]string `json:"sources"`
}

// WakeSignalProvider reads only machine-owned change signals. Implementations
// must not build prompts or invoke an agent.
type WakeSignalProvider interface {
	Snapshot(context.Context, string, string, []string) (WakeSignal, error)
}

type wakeAdmissionStateStore interface {
	GetHeartbeatAdmissionState(context.Context, string, string) (*store.HeartbeatAdmissionState, error)
	SetHeartbeatAdmissionState(context.Context, string, string, *store.HeartbeatAdmissionState) error
}

// FileWakeSignalProvider uses existing Prompt Manager-owned revisions and
// bounded corpus identities. It is deliberately conservative: a source that
// cannot be read returns an error, causing the caller to admit work rather
// than incorrectly suppress it.
type FileWakeSignalProvider struct{ Store *store.FileTeamStore }

func (p FileWakeSignalProvider) Snapshot(ctx context.Context, teamID, agentID string, sources []string) (WakeSignal, error) {
	if p.Store == nil {
		return WakeSignal{}, fmt.Errorf("wake signal store unavailable")
	}
	values := make(map[string]string, len(sources))
	team, err := p.Store.Get(ctx, teamID)
	if err != nil {
		return WakeSignal{}, err
	}
	for _, source := range sources {
		switch source {
		case "team":
			values[source] = fmt.Sprintf("%d:%s:%t:%t", team.Revision, team.UpdatedAt, team.Enabled, team.Archived)
		case "member":
			members, readErr := p.Store.GetMembers(ctx, teamID)
			if readErr != nil {
				return WakeSignal{}, readErr
			}
			for _, member := range members {
				if member.AgentID == agentID {
					payload, _ := json.Marshal(struct {
						Kind   string   `json:"kind"`
						Status string   `json:"status"`
						Roles  []string `json:"roles"`
					}{member.Kind, member.Status, member.Roles})
					values[source] = digest(payload)
					break
				}
			}
		case "inbox":
			inbox, readErr := p.Store.GetInbox(ctx, teamID, agentID)
			if readErr != nil {
				return WakeSignal{}, readErr
			}
			payload, _ := json.Marshal(inbox)
			values[source] = digest(payload)
		case "corpus":
			entries, readErr := p.Store.ListTeamCorpus(ctx, teamID, "", "", 256)
			if readErr != nil {
				return WakeSignal{}, readErr
			}
			payload, _ := json.Marshal(entries)
			values[source] = digest(payload)
		default:
			return WakeSignal{}, fmt.Errorf("unsupported wake signal source %q", source)
		}
	}
	payload, _ := json.Marshal(values)
	return WakeSignal{Key: digest(payload), Sources: values}, nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

type wakeDecision struct {
	Decision string
	Reason   string
}

const (
	wakeDecisionAdmit = "admit"
	wakeDecisionQuiet = "quiet"
)

// evaluateWakeAdmission is intentionally deterministic and side-effect free.
// State persistence is performed by the scheduler only after this decision.
func evaluateWakeAdmission(policy *teamconfig.WakeAdmission, previous *store.HeartbeatAdmissionState, signal WakeSignal) wakeDecision {
	if policy == nil || policy.Mode == "" || policy.Mode == teamconfig.WakeAdmissionAlways {
		return wakeDecision{Decision: wakeDecisionAdmit, Reason: "policy-always"}
	}
	if policy.Mode != teamconfig.WakeAdmissionOnChange {
		return wakeDecision{Decision: wakeDecisionAdmit, Reason: "policy-invalid-fail-open"}
	}
	if previous == nil || previous.LastSignal == "" {
		return wakeDecision{Decision: wakeDecisionAdmit, Reason: "first-observation"}
	}
	if previous.LastSignal == signal.Key {
		return wakeDecision{Decision: wakeDecisionQuiet, Reason: "signal-unchanged"}
	}
	return wakeDecision{Decision: wakeDecisionAdmit, Reason: "signal-changed"}
}

func recordWakeAdmission(state *store.HeartbeatAdmissionState, signal WakeSignal, decision wakeDecision, now time.Time) {
	if state.Version == 0 {
		state.Version = 1
	}
	state.LastSignal = signal.Key
	state.LastDecision = decision.Decision
	state.LastReason = decision.Reason
	state.LastCheckedAt = now.UTC().Format(time.RFC3339Nano)
	if decision.Decision == wakeDecisionAdmit {
		state.LastAdmittedAt = state.LastCheckedAt
		state.LastAdmissionKey = signal.Key
	}
}

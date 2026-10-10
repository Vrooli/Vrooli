package teamconfig

import (
	"fmt"
	"strings"
	"time"
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
	// Checkin selects how a keepAlive orchestrator is woken. It is policy, not
	// binding identity. Absent means parked.
	Checkin *CheckinPolicy `json:"checkin,omitempty"`
}

// Check-in modes. Parked keeps one long-lived leader that parks between
// wakes. Event dispatches a fresh, short check-in run for each new effort
// signal or an expired backstop; the check-in ends instead of parking.
const (
	CheckinModeParked = "parked"
	CheckinModeEvent  = "event"

	DefaultCheckinPollInterval = 2 * time.Minute
	DefaultCheckinBackstop     = 6 * time.Hour
)

// CheckinPolicy configures event-driven check-ins. Durations use Go syntax
// ("2m", "6h"); empty fields take the defaults.
type CheckinPolicy struct {
	Mode         string `json:"mode,omitempty"`
	PollInterval string `json:"pollInterval,omitempty"`
	Backstop     string `json:"backstop,omitempty"`
}

// CheckinMode reports the effective mode; anything unset is parked.
func (b *FiniteLeader) CheckinMode() string {
	if b == nil || b.Checkin == nil || b.Checkin.Mode == "" {
		return CheckinModeParked
	}
	return b.Checkin.Mode
}

// EventCheckins reports whether the leader is dispatched as event check-ins.
func (b *FiniteLeader) EventCheckins() bool {
	return b != nil && b.KeepAlive && b.CheckinMode() == CheckinModeEvent
}

// CheckinPollInterval is how often the heartbeat reads effort signals.
func (b *FiniteLeader) CheckinPollInterval() time.Duration {
	return checkinDuration(b, func(p *CheckinPolicy) string { return p.PollInterval }, DefaultCheckinPollInterval)
}

// CheckinBackstop is how long after a check-in ends the next one starts
// without any signal.
func (b *FiniteLeader) CheckinBackstop() time.Duration {
	return checkinDuration(b, func(p *CheckinPolicy) string { return p.Backstop }, DefaultCheckinBackstop)
}

func checkinDuration(b *FiniteLeader, field func(*CheckinPolicy) string, fallback time.Duration) time.Duration {
	if b == nil || b.Checkin == nil || field(b.Checkin) == "" {
		return fallback
	}
	d, err := time.ParseDuration(field(b.Checkin))
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func (p *CheckinPolicy) validate(keepAlive bool) error {
	if p == nil {
		return nil
	}
	switch p.Mode {
	case "", CheckinModeParked:
	case CheckinModeEvent:
		if !keepAlive {
			return fmt.Errorf("finiteLeader.checkin.mode=event requires keepAlive")
		}
	default:
		return fmt.Errorf("finiteLeader.checkin.mode must be %q or %q", CheckinModeParked, CheckinModeEvent)
	}
	durations := map[string]string{"pollInterval": p.PollInterval, "backstop": p.Backstop}
	for name, value := range durations {
		if value == "" {
			continue
		}
		d, err := time.ParseDuration(value)
		if err != nil || d < time.Minute {
			return fmt.Errorf("finiteLeader.checkin.%s must be a duration of at least 1m", name)
		}
	}
	return nil
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
	return b.Checkin.validate(b.KeepAlive)
}

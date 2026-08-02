package runsignal

import (
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// TimeAttributionState is the generic state inferred from an event stream.
// Its values deliberately describe evidence, rather than a scenario or tool.
type TimeAttributionState string

const (
	TimeAttributionModelGenerating TimeAttributionState = "model_generating"
	TimeAttributionToolExecuting   TimeAttributionState = "tool_executing"
	TimeAttributionIdleWaiting     TimeAttributionState = "idle_waiting"
	TimeAttributionAwaitingHuman   TimeAttributionState = "awaiting_human"
	TimeAttributionUnattributable  TimeAttributionState = "unattributable"
)

// TimeAccounting is a conservative, duration-conserving summary. The state
// totals plus UnattributableMS always equal the supplied run interval.
type TimeAccounting struct {
	ModelGeneratingMS    int64 `json:"modelGeneratingMs" db:"model_generating_ms"`
	ToolExecutingMS      int64 `json:"toolExecutingMs" db:"tool_executing_ms"`
	IdleWaitingMS        int64 `json:"idleWaitingMs" db:"idle_waiting_ms"`
	AwaitingHumanMS      int64 `json:"awaitingHumanMs" db:"awaiting_human_ms"`
	UnattributableMS     int64 `json:"unattributableMs" db:"unattributable_ms"`
	ModelTokens          int64 `json:"modelTokens" db:"model_tokens"`
	ToolTokens           int64 `json:"toolTokens" db:"tool_tokens"`
	IdleTokens           int64 `json:"idleTokens" db:"idle_tokens"`
	HumanTokens          int64 `json:"humanTokens" db:"human_tokens"`
	UnattributableTokens int64 `json:"unattributableTokens" db:"unattributable_tokens"`
}

func (a TimeAccounting) DurationMS() int64 {
	return a.ModelGeneratingMS + a.ToolExecutingMS + a.IdleWaitingMS + a.AwaitingHumanMS + a.UnattributableMS
}

func (a TimeAccounting) Tokens() int64 {
	return a.ModelTokens + a.ToolTokens + a.IdleTokens + a.HumanTokens + a.UnattributableTokens
}

// DeriveTimeAccounting attributes every interval in [startedAt, endedAt]. It
// treats absent boundaries and contradictory clocks as unknowable instead of
// manufacturing a state. Cost events are credited to the state active just
// before the usage landed.
func DeriveTimeAccounting(events []*domain.RunEvent, startedAt, endedAt *time.Time) TimeAccounting {
	var out TimeAccounting
	if startedAt == nil || endedAt == nil || !endedAt.After(*startedAt) {
		return out
	}
	ordered := make([]*domain.RunEvent, 0, len(events))
	for _, event := range events {
		if event != nil && !event.Timestamp.IsZero() {
			ordered = append(ordered, event)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Timestamp.Before(ordered[j].Timestamp) })
	previousAt := *startedAt
	state := TimeAttributionUnattributable
	for _, event := range ordered {
		if event.Timestamp.Before(*startedAt) || event.Timestamp.After(*endedAt) {
			continue
		}
		addDuration(&out, state, event.Timestamp.Sub(previousAt))
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			addTokens(&out, state, int64(usage.InputTokens+usage.OutputTokens+usage.CacheCreationTokens+usage.CacheReadTokens))
		} else {
			state = timeStateAfter(event)
		}
		previousAt = event.Timestamp
	}
	addDuration(&out, state, endedAt.Sub(previousAt))
	return out
}

func timeStateAfter(event *domain.RunEvent) TimeAttributionState {
	if _, ok := event.Data.(*domain.ToolCallEventData); ok {
		return TimeAttributionToolExecuting
	}
	if message, ok := event.Data.(*domain.MessageEventData); ok {
		switch strings.ToLower(message.Role) {
		case "assistant":
			return TimeAttributionAwaitingHuman
		case "user":
			return TimeAttributionModelGenerating
		}
	}
	if _, ok := event.Data.(*domain.ToolResultEventData); ok {
		return TimeAttributionModelGenerating
	}
	return TimeAttributionIdleWaiting
}

func addDuration(accounting *TimeAccounting, state TimeAttributionState, duration time.Duration) {
	ms := duration.Milliseconds()
	if ms <= 0 {
		return
	}
	switch state {
	case TimeAttributionModelGenerating:
		accounting.ModelGeneratingMS += ms
	case TimeAttributionToolExecuting:
		accounting.ToolExecutingMS += ms
	case TimeAttributionIdleWaiting:
		accounting.IdleWaitingMS += ms
	case TimeAttributionAwaitingHuman:
		accounting.AwaitingHumanMS += ms
	default:
		accounting.UnattributableMS += ms
	}
}

func addTokens(accounting *TimeAccounting, state TimeAttributionState, tokens int64) {
	if tokens <= 0 {
		return
	}
	switch state {
	case TimeAttributionModelGenerating:
		accounting.ModelTokens += tokens
	case TimeAttributionToolExecuting:
		accounting.ToolTokens += tokens
	case TimeAttributionIdleWaiting:
		accounting.IdleTokens += tokens
	case TimeAttributionAwaitingHuman:
		accounting.HumanTokens += tokens
	default:
		accounting.UnattributableTokens += tokens
	}
}

package runsignal

import (
	"sort"
	"time"

	"agent-manager/internal/domain"
)

// BeginsRunInvocation identifies the durable owner status boundary emitted
// before a fresh execution or continuation starts consuming provider work.
// Consecutive start events before usage describe the same empty interval.
func BeginsRunInvocation(event *domain.RunEvent) bool {
	if event == nil {
		return false
	}
	status, ok := event.Data.(*domain.StatusEventData)
	return ok && status.NewStatus == string(domain.RunStatusRunning)
}

// UsageAccount is the one derivation of a run's provider token usage. Every
// token total for a run (read model, report, time accounting, episodes, token
// cap, workflow metering) reads it, so they cannot disagree.
type UsageAccount struct {
	// Contributions[i] is the part of the run total that events[i] carries.
	// The sum over any subset of events never counts a provider report twice,
	// and the sum over all events equals Total.
	Contributions []domain.UsageCounters
	// PerTurn[i] is the turn-local usage events[i] reports, for attributing
	// usage to the work before it. It equals Contributions except where a
	// closing receipt (a total) replaced an invocation's per-message usage:
	// there the messages keep their amounts and the receipt has none.
	PerTurn []domain.UsageCounters
	// Invocations[i] is the owner invocation (status=running boundary) of
	// events[i]; zero precedes the first boundary.
	Invocations []int
	Total       domain.UsageCounters
	// AllInvocationsAuthoritative is true when every invocation that reported
	// usage ended on a provider closing report; LatestInvocationAuthoritative
	// says the same for the latest invocation only.
	AllInvocationsAuthoritative   bool
	LatestInvocationAuthoritative bool
}

// Tokens is the contribution of events[index] as one number.
func (a UsageAccount) Tokens(index int) int64 {
	if index < 0 || index >= len(a.Contributions) {
		return 0
	}
	return a.Contributions[index].Total()
}

// Between sums the contributions of events stamped within [from, to]. events
// must be the slice the account was derived from.
func (a UsageAccount) Between(events []*domain.RunEvent, from, to time.Time) int64 {
	var tokens int64
	for index, event := range events {
		if event != nil && !event.Timestamp.Before(from) && !event.Timestamp.After(to) {
			tokens += a.Tokens(index)
		}
	}
	return tokens
}

// TotalTokens is the run's token total.
func (a UsageAccount) TotalTokens() int64 { return a.Total.Total() }

// AccountUsage derives run usage from retained events.
//
//   - Cumulative usage (Codex) is grouped by provider session. A session's
//     usage is its last cumulative counter minus the counter before the run's
//     first observed report, where each report's "before" value is its
//     cumulative counter minus the provider's explicit amount for it. Each
//     distinct counter value is credited once, at its first occurrence, with
//     the difference from the next lower value. The result is invariant under
//     re-tails, restarts and replays, and a parser that starts mid-transcript
//     can only lower the before value it reports when it knows the amount.
//   - Per-message usage (Claude message.id) counts each message once, at the
//     largest amount reported for it.
//   - Other usage (a print-mode closing receipt without a message id, legacy
//     rows) keeps the receipt rule: a closing receipt replaces the interim
//     usage of its invocation, and an identical replayed row counts once.
func AccountUsage(events []*domain.RunEvent) UsageAccount {
	account := UsageAccount{Contributions: make([]domain.UsageCounters, len(events)), PerTurn: make([]domain.UsageCounters, len(events)), Invocations: make([]int, len(events))}
	receiptInvocation := map[int]bool{}
	latestAuthority := map[int]bool{}
	soleSession, sessions := "", map[string]bool{}
	invocation := 0
	for index, event := range events {
		if BeginsRunInvocation(event) {
			invocation++
			account.LatestInvocationAuthoritative = false
		}
		account.Invocations[index] = invocation
		usage, ok := usageData(event)
		if !ok {
			continue
		}
		latestAuthority[invocation] = usage.ReconciliationAuthority
		account.LatestInvocationAuthoritative = usage.ReconciliationAuthority
		if usage.Cumulative == nil && usage.MessageID == "" && usage.ReconciliationAuthority {
			receiptInvocation[invocation] = true
		}
		if usage.Cumulative != nil && usage.SessionID != "" {
			sessions[usage.SessionID] = true
			soleSession = usage.SessionID
		}
	}
	account.AllInvocationsAuthoritative = len(latestAuthority) > 0
	for _, authoritative := range latestAuthority {
		account.AllInvocationsAuthoritative = account.AllInvocationsAuthoritative && authoritative
	}
	if len(sessions) != 1 {
		soleSession = ""
	}

	type cumulativePoint struct {
		session string
		value   domain.UsageCounters
	}
	firstAt := map[cumulativePoint]int{}
	sessionPoints := map[string][]domain.UsageCounters{}
	sessionBefore := map[string]domain.UsageCounters{}
	messageAt := map[string]int{}
	type receiptIdentity struct {
		invocation int
		counters   domain.UsageCounters
		turns      int
		turnIndex  int
		authority  bool
		model      string
		runner     string
		tier       string
		webSearch  int
		serverTool int
	}
	receipts := map[receiptIdentity]bool{}
	for index, event := range events {
		usage, ok := usageData(event)
		if !ok {
			continue
		}
		switch {
		case usage.Cumulative != nil:
			// A parser that resumed mid-transcript may not have seen the session
			// id; it can only belong to the run's one known session.
			session := usage.SessionID
			if session == "" {
				session = soleSession
			}
			point := cumulativePoint{session: session, value: *usage.Cumulative}
			before := subtractCounters(point.value, usage.Counters())
			if current, seen := sessionBefore[session]; !seen || before.Total() < current.Total() {
				sessionBefore[session] = before
			}
			if _, seen := firstAt[point]; !seen {
				firstAt[point] = index
				sessionPoints[session] = append(sessionPoints[session], point.value)
			}
		case usage.MessageID != "":
			first, seen := messageAt[usage.MessageID]
			if !seen {
				messageAt[usage.MessageID] = index
				account.PerTurn[index] = usage.Counters()
			} else if usage.Counters().Total() > account.PerTurn[first].Total() {
				account.PerTurn[first] = usage.Counters()
			}
		default:
			identity := receiptIdentity{
				invocation: account.Invocations[index], counters: usage.Counters(), turns: usage.Turns, turnIndex: usage.TurnIndex,
				authority: usage.ReconciliationAuthority, model: usage.Model, runner: usage.RunnerType, tier: usage.ServiceTier,
				webSearch: usage.WebSearchRequests, serverTool: usage.ServerToolUseRequests,
			}
			if receipts[identity] {
				continue
			}
			receipts[identity] = true
			if usage.ReconciliationAuthority {
				account.Contributions[index] = usage.Counters()
			} else {
				account.PerTurn[index] = usage.Counters()
			}
		}
	}
	// Per-turn usage is the total unless a closing receipt replaced its
	// invocation; a receipt is a total, never a turn.
	for index, perTurn := range account.PerTurn {
		if !receiptInvocation[account.Invocations[index]] {
			account.Contributions[index] = perTurn
		}
	}
	for session, points := range sessionPoints {
		sortCounters(points)
		previous := sessionBefore[session]
		for _, value := range points {
			index := firstAt[cumulativePoint{session: session, value: value}]
			account.Contributions[index] = subtractCounters(value, previous)
			account.PerTurn[index] = account.Contributions[index]
			previous = value
		}
	}
	for _, contribution := range account.Contributions {
		account.Total = addCounters(account.Total, contribution)
	}
	return account
}

func usageData(event *domain.RunEvent) (*domain.UsageEventData, bool) {
	if event == nil {
		return nil, false
	}
	usage, ok := event.Data.(*domain.UsageEventData)
	return usage, ok
}

// subtractCounters clamps each component at zero: provider counters are
// monotone within a session, so a negative component is a malformed amount and
// must not reduce the total.
func subtractCounters(a, b domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{
		InputTokens:         max(0, a.InputTokens-b.InputTokens),
		OutputTokens:        max(0, a.OutputTokens-b.OutputTokens),
		CacheReadTokens:     max(0, a.CacheReadTokens-b.CacheReadTokens),
		CacheCreationTokens: max(0, a.CacheCreationTokens-b.CacheCreationTokens),
	}
}

func addCounters(a, b domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{
		InputTokens:         a.InputTokens + b.InputTokens,
		OutputTokens:        a.OutputTokens + b.OutputTokens,
		CacheReadTokens:     a.CacheReadTokens + b.CacheReadTokens,
		CacheCreationTokens: a.CacheCreationTokens + b.CacheCreationTokens,
	}
}

func sortCounters(values []domain.UsageCounters) {
	sort.SliceStable(values, func(i, j int) bool { return values[i].Total() < values[j].Total() })
}

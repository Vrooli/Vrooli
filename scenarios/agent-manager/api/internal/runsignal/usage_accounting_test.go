package runsignal

import (
	"testing"

	"agent-manager/internal/domain"
)

func running() *domain.RunEvent {
	return &domain.RunEvent{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}}
}

// report is a Codex usage report: the session counter after it and the
// provider's explicit amount for it (zero when unknown).
func report(session string, cumulative, amount domain.UsageCounters, closing bool) *domain.RunEvent {
	c := cumulative
	return &domain.RunEvent{Data: &domain.UsageEventData{
		SessionID: session, Cumulative: &c, ReconciliationAuthority: closing,
		InputTokens: amount.InputTokens, OutputTokens: amount.OutputTokens, CacheReadTokens: amount.CacheReadTokens,
	}}
}

func counters(input, output, cached int) domain.UsageCounters {
	return domain.UsageCounters{InputTokens: input, OutputTokens: output, CacheReadTokens: cached}
}

func TestAccountUsageDerivesCodexUsageFromCountersUnderReplay(t *testing.T) {
	c1, c2, c3 := counters(100, 10, 50), counters(150, 20, 90), counters(210, 25, 140)
	turn1 := []*domain.RunEvent{report("s", c1, c1, false), report("s", c2, counters(50, 10, 40), false), report("s", c2, counters(150, 20, 90), true)}
	turn2 := []*domain.RunEvent{report("s", c3, counters(60, 5, 50), false), report("s", c3, counters(60, 5, 50), true)}
	want := c3.Total()
	cases := map[string][]*domain.RunEvent{
		"live":                    append(append([]*domain.RunEvent{running()}, turn1...), turn2...),
		"re-tail repeats a turn":  append(append(append([]*domain.RunEvent{running()}, turn1...), turn2...), turn2...),
		"replay of the whole run": append(append(append(append([]*domain.RunEvent{running()}, turn1...), turn2...), turn1...), turn2...),
		// A parser that resumed mid-session knows no amount for its first
		// counter; an earlier report still fixes the run's starting counter.
		"mid-file parser without amount": append(append([]*domain.RunEvent{running()}, turn1...), report("s", c3, domain.UsageCounters{}, false)),
		"recovered receipt alone":        {running(), report("s", c2, counters(150, 20, 90), true)},
	}
	for name, events := range cases {
		account := AccountUsage(events)
		expected := want
		if name == "recovered receipt alone" {
			expected = c2.Total()
		}
		if account.TotalTokens() != expected {
			t.Errorf("%s: total = %d, want %d", name, account.TotalTokens(), expected)
		}
		var sum, perTurn int64
		for index := range events {
			sum += account.Tokens(index)
			perTurn += account.PerTurn[index].Total()
		}
		if sum != account.TotalTokens() || perTurn != account.TotalTokens() {
			t.Errorf("%s: contributions %d / per-turn %d do not conserve total %d", name, sum, perTurn, account.TotalTokens())
		}
	}
}

func TestAccountUsageNeverCountsAMidSessionCounterAsUsage(t *testing.T) {
	// A run that resumes a session someone else started: the first counter
	// covers earlier work and has no known amount, so only growth counts.
	events := []*domain.RunEvent{running(), report("s", counters(1000, 100, 500), domain.UsageCounters{}, true), running(), report("s", counters(1200, 130, 700), domain.UsageCounters{}, true)}
	if got := AccountUsage(events).TotalTokens(); got != 430 {
		t.Fatalf("total = %d, want only the observed growth 430", got)
	}
}

func TestAccountUsageKeepsSessionsIndependentAndFoldsUnnamedReports(t *testing.T) {
	events := []*domain.RunEvent{
		running(),
		report("a", counters(10, 1, 0), counters(10, 1, 0), false),
		report("", counters(30, 3, 0), counters(20, 2, 0), false), // parser had not seen the id yet
		report("b", counters(5, 1, 0), counters(5, 1, 0), false),
	}
	// Two named sessions: the unnamed report cannot be assigned and forms its own series.
	if got := AccountUsage(events).TotalTokens(); got != 11+22+6 {
		t.Fatalf("two sessions total = %d", got)
	}
	single := events[:3]
	if got := AccountUsage(single).TotalTokens(); got != 33 {
		t.Fatalf("one session with an unnamed report = %d, want 33", got)
	}
}

func message(id string, input, output int, closing bool) *domain.RunEvent {
	return &domain.RunEvent{Data: &domain.UsageEventData{MessageID: id, InputTokens: input, OutputTokens: output, ReconciliationAuthority: closing, RunnerType: string(domain.RunnerTypeClaudeCode)}}
}

func TestAccountUsageCountsEachClaudeMessageOnceAtItsLargestReport(t *testing.T) {
	events := []*domain.RunEvent{
		running(),
		message("m1", 100, 1, false), message("m1", 100, 40, false), message("m1", 100, 40, false), // streamed blocks
		message("m2", 200, 9, true), // on-disk end_turn
		message("m2", 200, 9, true), // re-tail of the same turn
	}
	account := AccountUsage(events)
	if account.TotalTokens() != 140+209 {
		t.Fatalf("total = %d, want 349", account.TotalTokens())
	}
	if !account.AllInvocationsAuthoritative || !account.LatestInvocationAuthoritative {
		t.Fatalf("end_turn closing report not recognized: %+v", account)
	}
}

func TestAccountUsagePrintModeReceiptReplacesItsMessages(t *testing.T) {
	receipt := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 500, OutputTokens: 50, ReconciliationAuthority: true}}
	events := []*domain.RunEvent{running(), message("m1", 100, 10, false), message("m2", 300, 30, false), receipt, receipt, running(), message("m3", 7, 1, false)}
	account := AccountUsage(events)
	if account.TotalTokens() != 550+8 {
		t.Fatalf("total = %d, want the receipt once plus the open invocation", account.TotalTokens())
	}
	// Attribution keeps the receipt invocation's turn-local message usage.
	if account.PerTurn[1].Total() != 110 || account.PerTurn[2].Total() != 330 || account.PerTurn[3].Total() != 0 {
		t.Fatalf("per-turn attribution = %+v", account.PerTurn[:4])
	}
	if account.AllInvocationsAuthoritative || account.LatestInvocationAuthoritative {
		t.Fatal("an open invocation was treated as closed")
	}
}

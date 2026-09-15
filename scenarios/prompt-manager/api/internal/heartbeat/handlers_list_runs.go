package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// This is a read-time join of PM's existing declarations with the AM owner.
// It persists no run state and never treats a dispatch log as an owner outcome.
type teamRunAccounting struct {
	TeamID      string    `json:"teamId"`
	AgentID     string    `json:"agentId,omitempty"`
	WindowStart time.Time `json:"windowStart"`
	WindowEnd   time.Time `json:"windowEnd"`
	ObservedAt  time.Time `json:"observedAt"`
	// KnownRuns counts distinct AM IDs, including IDs whose owner read fails.
	// State/model aggregates count observed executions after session deduplication.
	KnownRuns              int                  `json:"knownRuns"`
	ObservedRuns           int                  `json:"observedRuns"`
	UnavailableRuns        int                  `json:"unavailableRuns"`
	UnqueriedRuns          int                  `json:"unqueriedRuns"`
	ObservedExecutions     int                  `json:"observedExecutions"`
	DuplicateExecutions    int                  `json:"duplicateExecutions"`
	ActualModels           map[string]int       `json:"actualModels"`
	UnknownModelExecutions int                  `json:"unknownModelExecutions"`
	RuntimeStates          map[string]int       `json:"runtimeStates"`
	TerminalReasons        map[string]int       `json:"terminalReasons"`
	Usage                  teamRunUsage         `json:"usage"`
	Coverage               teamRunCoverage      `json:"coverage"`
	Runs                   []teamRunObservation `json:"runs"`
}

type teamRunUsage struct {
	Tokens            *int64   `json:"tokens"`
	CostUSD           *float64 `json:"costUSD"`
	QualifiedRuns     int      `json:"qualifiedRuns"`
	ReportedTokenRuns int      `json:"reportedTokenRuns"`
	ReportedCostRuns  int      `json:"reportedCostRuns"`
	Partial           bool     `json:"partial"`
}

type teamRunCoverage struct {
	Partial           bool     `json:"partial"`
	DeclarationsRead  int      `json:"declarationsRead"`
	DeclarationLimit  int      `json:"declarationLimit"`
	OwnerReadLimit    int      `json:"ownerReadLimit"`
	InvalidTimestamps int      `json:"invalidTimestamps"`
	Limitations       []string `json:"limitations"`
}

type teamRunObservation struct {
	RunID             string    `json:"runId"`
	AgentIDs          []string  `json:"agentIds"`
	DeclaredAt        time.Time `json:"declaredAt"`
	Availability      string    `json:"availability"`
	ExecutionIdentity string    `json:"executionIdentity,omitempty"`
	RuntimeState      string    `json:"runtimeState,omitempty"`
	ActualModel       string    `json:"actualModel,omitempty"`
	TerminalClass     string    `json:"terminalClass,omitempty"`
	StopReason        string    `json:"stopReason,omitempty"`
	run               *Run
}

func (h *Handlers) listTeamRunAccounting(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	q := r.URL.Query()
	teamID, agentID := strings.TrimSpace(q.Get("team_id")), strings.TrimSpace(q.Get("agent_id"))
	if teamID == "" {
		http.Error(w, "team_id is required for accounting", http.StatusBadRequest)
		return
	}
	if h.teamStore == nil {
		http.Error(w, "team declaration owner unavailable", http.StatusServiceUnavailable)
		return
	}
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}
	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)
	for _, item := range []struct {
		name  string
		value *time.Time
	}{{"start", &start}, {"end", &end}} {
		if raw := q.Get(item.name); raw != "" {
			value, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				http.Error(w, "invalid accounting window", http.StatusBadRequest)
				return
			}
			*item.value = value
		}
	}
	if !start.Before(end) || end.Sub(start) > 31*24*time.Hour {
		http.Error(w, "accounting window must be positive and at most 31 days", http.StatusBadRequest)
		return
	}
	limit := 50
	if raw := q.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			http.Error(w, "accounting limit must be 1-100", http.StatusBadRequest)
			return
		}
		limit = value
	}
	const declarationLimit = 500
	out := teamRunAccounting{TeamID: teamID, AgentID: agentID, WindowStart: start, WindowEnd: end, ObservedAt: time.Now().UTC(),
		ActualModels: map[string]int{}, RuntimeStates: map[string]int{}, TerminalReasons: map[string]int{},
		Usage: teamRunUsage{Partial: true}, Runs: []teamRunObservation{},
		Coverage: teamRunCoverage{Partial: true, DeclarationLimit: declarationLimit, OwnerReadLimit: limit, Limitations: []string{
			"Known PM-declared runs only; retained heartbeat history is a bounded sample and does not prove complete team history.",
			"Window selects PM declarations by start time; owner state is observed now. Reads are not an atomic snapshot.",
			"Execution identity uses runner-qualified sessions when present, otherwise run IDs; missing session identity can hide duplicates.",
			"AM summaries do not qualify per-attempt token usage or actual charges. Terminal state does not establish outcome acceptance.",
		}},
	}
	declared := map[string]*teamRunObservation{}
	add := func(team, member, id, at string) {
		if id == "" || team != teamID || (agentID != "" && member != agentID) {
			return
		}
		stamp, err := time.Parse(time.RFC3339, at)
		if err != nil {
			out.Coverage.InvalidTimestamps++
			return
		}
		if stamp.Before(start) || !stamp.Before(end) {
			return
		}
		row := declared[id]
		if row == nil {
			row = &teamRunObservation{RunID: id, AgentIDs: []string{}, DeclaredAt: stamp, Availability: "not_queried"}
			declared[id] = row
		}
		if stamp.After(row.DeclaredAt) {
			row.DeclaredAt = stamp
		}
		for _, existing := range row.AgentIDs {
			if existing == member {
				return
			}
		}
		row.AgentIDs = append(row.AgentIDs, member)
	}
	attempts, total, err := h.teamStore.ListHeartbeatAttempts(ctx, teamID, agentID, "", "", declarationLimit, 0)
	if err != nil {
		out.Coverage.Limitations = append(out.Coverage.Limitations, "Heartbeat declaration history unavailable.")
	} else {
		out.Coverage.DeclarationsRead = len(attempts)
		if total > len(attempts) {
			out.Coverage.Limitations = append(out.Coverage.Limitations, "Declaration page limit reached; additional known history was not read.")
		}
		for _, a := range attempts {
			add(a.TeamID, a.AgentID, a.RunID, a.StartedAt)
		}
	}
	configs, err := h.teamStore.ListHeartbeatConfigs(ctx, teamID)
	if err != nil {
		out.Coverage.Limitations = append(out.Coverage.Limitations, "Latest heartbeat declarations unavailable.")
	} else {
		for _, cfg := range configs {
			if cfg.LastExecution != nil {
				add(teamID, cfg.AgentID, cfg.LastExecution.RunID, cfg.LastExecution.StartedAt)
			}
		}
	}
	if h.runRegistry != nil {
		for _, a := range h.runRegistry.ListActive() {
			add(a.TeamID, a.AgentID, a.RunID, a.StartedAt.Format(time.RFC3339Nano))
		}
	} else {
		out.Coverage.Limitations = append(out.Coverage.Limitations, "Active run registry unavailable.")
	}
	for _, row := range declared {
		sort.Strings(row.AgentIDs)
		out.Runs = append(out.Runs, *row)
	}
	sort.Slice(out.Runs, func(i, j int) bool {
		if out.Runs[i].DeclaredAt.Equal(out.Runs[j].DeclaredAt) {
			return out.Runs[i].RunID < out.Runs[j].RunID
		}
		return out.Runs[i].DeclaredAt.After(out.Runs[j].DeclaredAt)
	})
	out.KnownRuns = len(out.Runs)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4)
	for i := 0; i < len(out.Runs) && i < limit; i++ {
		wg.Add(1)
		go func(row *teamRunObservation) {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			if ctx.Err() != nil {
				return
			}
			row.Availability = "unavailable"
			if h.agentClient == nil {
				return
			}
			readCtx, done := context.WithTimeout(ctx, 2*time.Second)
			defer done()
			run, err := h.agentClient.GetRun(readCtx, row.RunID)
			if err != nil || run == nil || run.ID != row.RunID {
				return
			}
			row.run, row.Availability = run, "available"
			row.ExecutionIdentity = accountingExecutionIdentity(run)
			row.RuntimeState = strings.ToLower(strings.TrimPrefix(run.Status, "RUN_STATUS_"))
			if row.RuntimeState == "" || row.RuntimeState == "unspecified" {
				row.RuntimeState = "unknown"
			}
			if !IsTerminalStatus(run.Status) && run.EndedAt != "" {
				row.RuntimeState = "unknown"
			}
			row.ActualModel, row.TerminalClass, row.StopReason = run.ActualModel, run.TerminalClass, run.StopReason
		}(&out.Runs[i])
	}
	wg.Wait()
	executions := map[string]*teamRunObservation{}
	for i := range out.Runs {
		row := &out.Runs[i]
		switch row.Availability {
		case "unavailable":
			out.UnavailableRuns++
			continue
		case "not_queried":
			out.UnqueriedRuns++
			continue
		}
		out.ObservedRuns++
		var summary struct {
			Tokens *int64   `json:"tokens_used"`
			Cost   *float64 `json:"cost_estimate"`
		}
		if json.Unmarshal(row.run.Summary, &summary) == nil {
			if summary.Tokens != nil && *summary.Tokens >= 0 {
				out.Usage.ReportedTokenRuns++
			}
			if summary.Cost != nil && *summary.Cost >= 0 {
				out.Usage.ReportedCostRuns++
			}
		}
		old := executions[row.ExecutionIdentity]
		if old == nil || accountingPreferRun(row.run, old.run) {
			executions[row.ExecutionIdentity] = row
		}
	}
	out.ObservedExecutions = len(executions)
	out.DuplicateExecutions = out.ObservedRuns - out.ObservedExecutions
	for _, row := range executions {
		out.RuntimeStates[row.RuntimeState]++
		if strings.TrimSpace(row.ActualModel) == "" {
			out.UnknownModelExecutions++
		} else {
			out.ActualModels[row.ActualModel]++
		}
		if IsTerminalStatus(row.run.Status) && row.StopReason != "" {
			out.TerminalReasons[row.StopReason]++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func accountingExecutionIdentity(run *Run) string {
	harness, session := run.HarnessKind, run.SessionID
	if run.ImportSourceHarness != "" && run.ImportSourceSessionID != "" {
		harness, session = run.ImportSourceHarness, run.ImportSourceSessionID
	}
	if harness != "" && session != "" {
		return "session:" + harness + ":" + session
	}
	return "run:" + run.ID
}

func accountingPreferRun(candidate, previous *Run) bool {
	c, ce := time.Parse(time.RFC3339, candidate.UpdatedAt)
	p, pe := time.Parse(time.RFC3339, previous.UpdatedAt)
	if ce == nil && pe == nil && !c.Equal(p) {
		return c.After(p)
	}
	if (candidate.ImportSourceSessionID == "") != (previous.ImportSourceSessionID == "") {
		return candidate.ImportSourceSessionID == ""
	}
	return candidate.ID < previous.ID
}

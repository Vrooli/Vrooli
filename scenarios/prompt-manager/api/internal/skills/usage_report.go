// Per-skill usage aggregation: how often discovery offered a skill, how often
// an agent actually read it, and the ratio between them.
//
// The ratio is the point. Discovery returning a skill 129 times while agents
// read it 3 times is not a popular skill — it is a search-precision defect that
// spends discovery budget on every call it loses. A raw read count cannot
// distinguish that from a skill nobody needs.
//
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md § Three reachability classes
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"prompt-manager/internal/store"
)

// SkillUsageRow is one skill's demand picture over the window.
type SkillUsageRow struct {
	SkillID string `json:"skillId"`
	// Returned counts discover calls whose result set included this skill.
	Returned int `json:"returned"`
	// Reads counts every recorded read, whatever the caller kind.
	Reads int `json:"reads"`
	// DemandReads counts reads by agent-member callers only — the lane work
	// that a usage-weighted selection ladder should rank on. Audit and operator
	// reads are excluded so that visiting a skill does not raise its rank.
	DemandReads int `json:"demandReads"`
	// ViaDiscovery counts reads a recent discover call by the same caller had
	// surfaced.
	ViaDiscovery int `json:"viaDiscovery"`
	// ReadsByCallerKind breaks the total down; the keys are attribution kinds.
	ReadsByCallerKind map[string]int `json:"readsByCallerKind,omitempty"`
	// ConversionRate is Reads/Returned, present only when Returned > 0. A low
	// rate against a high Returned is the precision defect described above.
	ConversionRate *float64 `json:"conversionRate,omitempty"`
	// LastReadAt is the newest read timestamp in the window, RFC3339.
	LastReadAt string `json:"lastReadAt,omitempty"`
	// Projected identifies skills resident in a generated native harness scope.
	// It prevents an operator from interpreting a projected skill's absent CLI
	// reads as absence of demand.
	Projected       bool     `json:"projected"`
	ReadsWithRun    int      `json:"readsWithRun,omitempty"`
	SucceededRuns   int      `json:"succeededRuns,omitempty"`
	FailedRuns      int      `json:"failedRuns,omitempty"`
	OutcomeCoverage *float64 `json:"outcomeCoverage,omitempty"`
}

// SkillUsageReport is the windowed aggregation across all skills seen.
type SkillUsageReport struct {
	Since string `json:"since"`
	// Unread names skills discovery returned in the window that were never
	// read. These are the precision suspects, ordered by how often they were
	// offered.
	Unread []string        `json:"unread,omitempty"`
	Rows   []SkillUsageRow `json:"rows"`
}

// UsageReporter builds SkillUsageReport from the two telemetry logs.
type UsageReporter struct {
	reads           *store.SkillReadStore
	calls           *store.DiscoveryCallStore
	projectedDir    string
	outcomeResolver RunStatusResolver
}

// RunStatusResolver is the narrow composition seam for the opt-in outcomes
// view. The skills package does not depend on heartbeat or agent-manager.
type RunStatusResolver interface {
	RunStatus(context.Context, string) (string, error)
}

// NewUsageReporter builds a reporter over the read and discovery logs.
func NewUsageReporter(reads *store.SkillReadStore, calls *store.DiscoveryCallStore) *UsageReporter {
	return &UsageReporter{reads: reads, calls: calls, projectedDir: strings.TrimSpace(os.Getenv("VROOLI_SKILL_PROJECTION_DIR"))}
}

func (ur *UsageReporter) SetOutcomeResolver(resolver RunStatusResolver) {
	ur.outcomeResolver = resolver
}

// Report aggregates the window. A nil store contributes zero rather than
// failing, so the report degrades to whichever half is wired.
func (ur *UsageReporter) Report(window time.Duration, outcomes ...bool) (SkillUsageReport, error) {
	report := SkillUsageReport{Since: window.String(), Rows: []SkillUsageRow{}}
	if ur == nil {
		return report, nil
	}
	rows := map[string]*SkillUsageRow{}
	runIDs := map[string]map[string]struct{}{}
	row := func(id string) *SkillUsageRow {
		if existing, ok := rows[id]; ok {
			return existing
		}
		created := &SkillUsageRow{SkillID: id, ReadsByCallerKind: map[string]int{}}
		rows[id] = created
		return created
	}

	if ur.calls != nil {
		calls, err := ur.calls.ReadSince(window)
		if err != nil {
			return report, err
		}
		for _, call := range calls {
			// One call counts once per skill even if the pipeline listed it
			// twice (topic and search sources both contribute results).
			seen := map[string]bool{}
			for _, result := range call.Results {
				if result.ID == "" || seen[result.ID] {
					continue
				}
				seen[result.ID] = true
				row(result.ID).Returned++
			}
		}
	}

	if ur.reads != nil {
		entries, err := ur.reads.ReadSince(window)
		if err != nil {
			return report, err
		}
		for _, entry := range entries {
			if entry.SkillID == "" {
				continue
			}
			r := row(entry.SkillID)
			r.Reads++
			kind := entry.CallerKind
			if kind == "" {
				kind = "unattributed"
			}
			r.ReadsByCallerKind[kind]++
			if kind == "agent-member" {
				r.DemandReads++
			}
			if entry.ViaDiscovery {
				r.ViaDiscovery++
			}
			if entry.At > r.LastReadAt {
				r.LastReadAt = entry.At
			}
			if len(outcomes) > 0 && outcomes[0] && ur.outcomeResolver != nil && entry.AgentRunID != "" {
				if runIDs[entry.SkillID] == nil {
					runIDs[entry.SkillID] = map[string]struct{}{}
				}
				runIDs[entry.SkillID][entry.AgentRunID] = struct{}{}
			}
		}
	}
	if len(outcomes) > 0 && outcomes[0] && ur.outcomeResolver != nil {
		for skillID, ids := range runIDs {
			r := rows[skillID]
			for runID := range ids {
				status, err := ur.outcomeResolver.RunStatus(context.Background(), runID)
				if err != nil {
					continue
				}
				r.ReadsWithRun++
				if strings.Contains(strings.ToLower(status), "fail") || strings.Contains(strings.ToLower(status), "cancel") {
					r.FailedRuns++
				} else {
					r.SucceededRuns++
				}
			}
			coverage := float64(r.ReadsWithRun) / float64(maxInt(r.DemandReads, r.Reads))
			if r.Reads > 0 {
				coverage = float64(r.ReadsWithRun) / float64(r.Reads)
			}
			r.OutcomeCoverage = &coverage
		}
	}

	projectedDirs := []string{ur.projectedDir}
	if configured := strings.TrimSpace(os.Getenv("VROOLI_SKILL_PROJECTION_DIRS")); configured != "" {
		projectedDirs = append(projectedDirs, strings.Split(configured, string(os.PathListSeparator))...)
	}
	for _, r := range rows {
		for _, dir := range projectedDirs {
			if strings.TrimSpace(dir) == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, r.SkillID, "SKILL.md")); err == nil {
				r.Projected = true
				break
			}
		}
		if r.Returned > 0 {
			rate := float64(r.Reads) / float64(r.Returned)
			r.ConversionRate = &rate
		}
		if len(r.ReadsByCallerKind) == 0 {
			r.ReadsByCallerKind = nil
		}
		report.Rows = append(report.Rows, *r)
		if r.Reads == 0 && r.Returned > 0 {
			report.Unread = append(report.Unread, r.SkillID)
		}
	}

	// Most-offered first: the top of this list is where budget is spent, so it
	// is where a precision defect costs the most.
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].Returned != report.Rows[j].Returned {
			return report.Rows[i].Returned > report.Rows[j].Returned
		}
		if report.Rows[i].Reads != report.Rows[j].Reads {
			return report.Rows[i].Reads > report.Rows[j].Reads
		}
		return report.Rows[i].SkillID < report.Rows[j].SkillID
	})
	sort.Slice(report.Unread, func(i, j int) bool {
		return rows[report.Unread[i]].Returned > rows[report.Unread[j]].Returned
	})
	return report, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// parseSinceWindow accepts the same window vocabulary the discovery telemetry
// commands use ("7d", "24h", "7d12h"), so `--since` means one thing across the
// three surfaces an operator reads together.
func parseSinceWindow(raw string) (time.Duration, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return 7 * 24 * time.Hour, nil
	}
	var days int
	if idx := strings.IndexByte(value, 'd'); idx >= 0 {
		parsed, err := strconv.Atoi(strings.TrimSpace(value[:idx]))
		if err != nil {
			return 0, fmt.Errorf("invalid since window %q", raw)
		}
		days = parsed
		value = strings.TrimSpace(value[idx+1:])
	}
	total := time.Duration(days) * 24 * time.Hour
	if value != "" {
		rest, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("invalid since window %q", raw)
		}
		total += rest
	}
	if total <= 0 {
		return 0, fmt.Errorf("since window must be positive: %q", raw)
	}
	return total, nil
}

// SkillUsage handles GET /skill-usage.
func (h *Handlers) SkillUsage(w http.ResponseWriter, r *http.Request) {
	if h.usageReporter == nil {
		http.Error(w, "skill-usage telemetry is not configured", http.StatusServiceUnavailable)
		return
	}
	window := 7 * 24 * time.Hour
	outcomes := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("outcomes")), "true")
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		parsed, err := parseSinceWindow(raw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		window = parsed
	}
	report, err := h.usageReporter.Report(window, outcomes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

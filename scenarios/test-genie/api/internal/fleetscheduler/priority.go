package fleetscheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// cliPrioritySource is the production PrioritySource: it shells
// `scenario-completeness-scoring score list --json` and parses the ScoreRow
// page. This is the keystone fleet query the backbone is built on — importance,
// precomputed priority, and scenario-level test recency (last_run_at/last_status)
// in one call — mirroring how the execute report already reads the scoring CLI
// (cli/execute/report/completeness.go). The scheduler depends on the SCS CLI
// contract, not on a generated client, keeping the build edge minimal.
type cliPrioritySource struct {
	// run is the seam over the subprocess; tests substitute it. It returns the
	// raw protojson stdout of the score-list call.
	run func(ctx context.Context) ([]byte, error)
	// pageSize bounds one read (scheduler ranks the page locally and selects a
	// small per-cycle budget from it).
	pageSize int
}

// NewCLIPrioritySource builds the production source. pageSize<=0 defaults to a
// large page so the whole fleet is ranked in one read.
func NewCLIPrioritySource(pageSize int) PrioritySource {
	if pageSize <= 0 {
		pageSize = 500
	}
	src := &cliPrioritySource{pageSize: pageSize}
	src.run = src.runCLI
	return src
}

// scoreListBudget bounds the subprocess; the score list is a cached read.
const scoreListBudget = 10 * time.Second

func (s *cliPrioritySource) runCLI(ctx context.Context) ([]byte, error) {
	path, err := exec.LookPath("scenario-completeness-scoring")
	if err != nil {
		return nil, fmt.Errorf("scenario-completeness-scoring CLI not found: %w", err)
	}
	cctx, cancel := context.WithTimeout(ctx, scoreListBudget)
	defer cancel()
	// Priority-sorted server-side so even a truncated page is the right tail;
	// the scheduler re-ranks locally with staleness weighting.
	return exec.CommandContext(cctx, path, "score", "list",
		"--json", "--sort", "priority", "--order", "desc",
		"--limit", fmt.Sprintf("%d", s.pageSize)).Output()
}

// scoreListPayload is the minimal slice of ListScoresResponse protojson the
// scheduler reads. Unknown fields are ignored, so the contract can grow.
type scoreListPayload struct {
	Scores []struct {
		Scenario   string  `json:"scenario"`
		Importance float64 `json:"importance"`
		Priority   float64 `json:"priority"`
		LastRunAt  string  `json:"last_run_at"`
		LastStatus string  `json:"last_status"`
	} `json:"scores"`
}

func (s *cliPrioritySource) Candidates(ctx context.Context) ([]Candidate, error) {
	out, err := s.run(ctx)
	if err != nil {
		return nil, err
	}
	var payload scoreListPayload
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("parse score list: %w", err)
	}
	candidates := make([]Candidate, 0, len(payload.Scores))
	for _, row := range payload.Scores {
		name := strings.TrimSpace(row.Scenario)
		if name == "" {
			continue
		}
		c := Candidate{
			Scenario:   name,
			Importance: row.Importance,
			Priority:   row.Priority,
			LastStatus: strings.TrimSpace(row.LastStatus),
		}
		if ts := strings.TrimSpace(row.LastRunAt); ts != "" {
			if t, perr := time.Parse(time.RFC3339Nano, ts); perr == nil {
				c.LastRunAt = t
			}
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

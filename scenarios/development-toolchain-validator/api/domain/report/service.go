package report

import (
	"context"
	"fmt"
	"time"

	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/domain/skill"
)

// Service generates aggregated reports from validation data.
type Service struct {
	connections  SkillConnectionLister
	expectations ExpectationLister
	results      ValidationResultReader
}

// NewService creates a report service with the required data sources.
func NewService(
	connections SkillConnectionLister,
	expectations ExpectationLister,
	results ValidationResultReader,
) *Service {
	return &Service{
		connections:  connections,
		expectations: expectations,
		results:      results,
	}
}

// Conflicts detects cross-skill contradictions on references.
//
// Two types of conflict are detected:
//   - Structural: different skills on the same reference have overlapping
//     file/folder expectations with incompatible required flags or content.
//   - CLI: different skills on the same reference assert different expected
//     values for the same command+jsonpath combination.
func (s *Service) Conflicts(ctx context.Context, opts ListOptions) (*ConflictsReport, error) {
	conns, err := s.listConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	// Group connections by reference for cross-skill comparison.
	byRef := groupByReference(conns)

	var conflicts []*Conflict
	for refID, refConns := range byRef {
		if len(refConns) < 2 {
			continue
		}
		structural, err := s.detectStructuralConflicts(ctx, refID, refConns)
		if err != nil {
			return nil, fmt.Errorf("structural conflicts for %s: %w", refID, err)
		}
		conflicts = append(conflicts, structural...)

		cli, err := s.detectCLIConflicts(ctx, refID, refConns)
		if err != nil {
			return nil, fmt.Errorf("CLI conflicts for %s: %w", refID, err)
		}
		conflicts = append(conflicts, cli...)
	}

	return &ConflictsReport{
		Conflicts:   conflicts,
		TotalCount:  len(conflicts),
		GeneratedAt: time.Now(),
	}, nil
}

// Drift aggregates drift status across all connections.
//
// Unlike per-connection drift checking in the skill domain, this report
// identifies ALL drifted connections at once. It requires current hashes
// to be passed in via the DriftInput map (skill_id -> current hash).
// Connections whose stored hash differs from the provided current hash
// are reported as drifted.
func (s *Service) Drift(ctx context.Context, opts ListOptions, currentHashes map[string]string) (*DriftReport, error) {
	conns, err := s.listConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	var drifted []*DriftEntry
	for _, conn := range conns {
		currentHash, ok := currentHashes[conn.SkillID]
		if !ok {
			continue
		}
		if conn.SkillContentHash == currentHash {
			continue
		}
		drifted = append(drifted, &DriftEntry{
			ConnectionID:   conn.ID,
			ReferenceID:    conn.ReferenceID,
			SkillID:        conn.SkillID,
			StoredHash:     conn.SkillContentHash,
			CurrentHash:    currentHash,
			StoredVersion:  conn.SkillVersion,
			CurrentVersion: currentHashes[conn.SkillID+"_version"],
			VersionChanged: conn.SkillVersion != currentHashes[conn.SkillID+"_version"],
			ContentChanged: true,
		})
	}

	return &DriftReport{
		DriftedConnections: drifted,
		TotalConnections:   len(conns),
		DriftedCount:       len(drifted),
		GeneratedAt:        time.Now(),
	}, nil
}

// Maturity scores skill expectation coverage across connections.
//
// Scoring:
//   - 0 expectations = low maturity (score 0)
//   - Only structural OR only CLI = medium maturity (score 0.5)
//   - Both structural AND CLI = high maturity (score 1.0)
//
// The score is weighted: 40% structural presence, 40% CLI presence,
// 20% total expectation count (capped at 10 for full credit).
func (s *Service) Maturity(ctx context.Context, opts ListOptions) (*MaturityReport, error) {
	conns, err := s.listConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	var skills []*SkillMaturity
	dist := map[MaturityLevel]int{
		MaturityLow:    0,
		MaturityMedium: 0,
		MaturityHigh:   0,
	}
	var totalScore float64

	for _, conn := range conns {
		structural, err := s.expectations.ListStructural(ctx, expectation.ListOptions{ConnectionID: conn.ID})
		if err != nil {
			return nil, fmt.Errorf("listing structural for %s: %w", conn.ID, err)
		}
		cli, err := s.expectations.ListCLI(ctx, expectation.ListOptions{ConnectionID: conn.ID})
		if err != nil {
			return nil, fmt.Errorf("listing CLI for %s: %w", conn.ID, err)
		}

		hasStructural := len(structural) > 0
		hasCLI := len(cli) > 0
		total := len(structural) + len(cli)

		score := computeMaturityScore(hasStructural, hasCLI, total)
		level := classifyMaturity(hasStructural, hasCLI, total)

		sm := &SkillMaturity{
			ConnectionID:      conn.ID,
			ReferenceID:       conn.ReferenceID,
			SkillID:           conn.SkillID,
			StructuralCount:   len(structural),
			CLICount:          len(cli),
			TotalExpectations: total,
			HasStructural:     hasStructural,
			HasCLI:            hasCLI,
			Level:             level,
			Score:             score,
		}
		skills = append(skills, sm)
		dist[level]++
		totalScore += score
	}

	var avg float64
	if len(skills) > 0 {
		avg = totalScore / float64(len(skills))
	}

	return &MaturityReport{
		Skills:       skills,
		Distribution: dist,
		AverageScore: avg,
		GeneratedAt:  time.Now(),
	}, nil
}

// ToolBaselines checks tool accuracy against stored validation results.
//
// For each reference, it examines the latest CLI assertion results and
// groups them by the tool invoked (extracted from the command prefix).
// A tool "passes" if all its assertions pass; it "fails" if any fail.
func (s *Service) ToolBaselines(ctx context.Context, opts ListOptions) (*ToolBaselinesReport, error) {
	conns, err := s.listConnections(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	// Collect unique reference IDs.
	refIDs := uniqueReferenceIDs(conns)

	var baselines []*ToolBaseline
	passingTools := 0
	failingTools := 0

	for _, refID := range refIDs {
		rows, err := s.results.CLIResultsByReference(ctx, refID)
		if err != nil {
			return nil, fmt.Errorf("CLI results for %s: %w", refID, err)
		}
		if len(rows) == 0 {
			continue
		}

		// Group results by tool name (first token of command).
		byTool := groupCLIResultsByTool(rows)
		for tool, results := range byTool {
			var passCount, failCount, errorCount int
			for _, r := range results {
				switch r.Status {
				case "pass":
					passCount++
				case "fail":
					failCount++
				default:
					errorCount++
				}
			}

			status := BaselinePass
			msg := "all assertions pass"
			if failCount > 0 {
				status = BaselineFail
				msg = fmt.Sprintf("%d/%d assertions failed", failCount, len(results))
			} else if errorCount > 0 {
				status = BaselineError
				msg = fmt.Sprintf("%d/%d assertions errored", errorCount, len(results))
			}

			bl := &ToolBaseline{
				ReferenceID:   refID,
				ToolName:      tool,
				Status:        status,
				PassCount:     passCount,
				FailCount:     failCount,
				ErrorCount:    errorCount,
				TotalAsserted: len(results),
				Message:       msg,
			}
			baselines = append(baselines, bl)

			if status == BaselinePass {
				passingTools++
			} else {
				failingTools++
			}
		}
	}

	return &ToolBaselinesReport{
		Baselines:    baselines,
		TotalTools:   len(baselines),
		PassingTools: passingTools,
		FailingTools: failingTools,
		GeneratedAt:  time.Now(),
	}, nil
}

// --- helpers ---

func (s *Service) listConnections(ctx context.Context, opts ListOptions) ([]*skill.Connection, error) {
	return s.connections.List(ctx, skill.ListOptions{
		ReferenceID: opts.ReferenceID,
		SkillID:     opts.SkillID,
		Limit:       0, // no limit for reports
	})
}

func groupByReference(conns []*skill.Connection) map[string][]*skill.Connection {
	m := make(map[string][]*skill.Connection)
	for _, c := range conns {
		m[c.ReferenceID] = append(m[c.ReferenceID], c)
	}
	return m
}

func (s *Service) detectStructuralConflicts(ctx context.Context, refID string, conns []*skill.Connection) ([]*Conflict, error) {
	type expKey struct {
		typ     string
		pattern string
	}
	type entry struct {
		conn *skill.Connection
		exp  *expectation.StructuralExpectation
	}

	seen := make(map[expKey][]entry)

	for _, conn := range conns {
		exps, err := s.expectations.ListStructural(ctx, expectation.ListOptions{ConnectionID: conn.ID})
		if err != nil {
			return nil, err
		}
		for _, exp := range exps {
			k := expKey{typ: string(exp.Type), pattern: exp.Pattern}
			seen[k] = append(seen[k], entry{conn: conn, exp: exp})
		}
	}

	var conflicts []*Conflict
	for k, entries := range seen {
		if len(entries) < 2 {
			continue
		}
		// Check pairs for incompatibility: different Required or different ExpectedContent.
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				a, b := entries[i], entries[j]
				if a.exp.Required != b.exp.Required || a.exp.ExpectedContent != b.exp.ExpectedContent {
					conflicts = append(conflicts, &Conflict{
						ReferenceID:  refID,
						Type:         ConflictStructural,
						Description:  fmt.Sprintf("conflicting structural expectation on %s pattern %q", k.typ, k.pattern),
						SkillA:       a.conn.SkillID,
						ConnectionA:  a.conn.ID,
						SkillB:       b.conn.SkillID,
						ConnectionB:  b.conn.ID,
						ExpectationA: a.exp.ID,
						ExpectationB: b.exp.ID,
					})
				}
			}
		}
	}
	return conflicts, nil
}

func (s *Service) detectCLIConflicts(ctx context.Context, refID string, conns []*skill.Connection) ([]*Conflict, error) {
	type cliKey struct {
		command  string
		jsonPath string
	}
	type entry struct {
		conn      *skill.Connection
		assertion *expectation.CLIAssertion
	}

	seen := make(map[cliKey][]entry)

	for _, conn := range conns {
		assertions, err := s.expectations.ListCLI(ctx, expectation.ListOptions{ConnectionID: conn.ID})
		if err != nil {
			return nil, err
		}
		for _, a := range assertions {
			k := cliKey{command: a.Command, jsonPath: a.JSONPath}
			seen[k] = append(seen[k], entry{conn: conn, assertion: a})
		}
	}

	var conflicts []*Conflict
	for _, entries := range seen {
		if len(entries) < 2 {
			continue
		}
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				a, b := entries[i], entries[j]
				if fmt.Sprintf("%v", a.assertion.ExpectedValue) != fmt.Sprintf("%v", b.assertion.ExpectedValue) {
					conflicts = append(conflicts, &Conflict{
						ReferenceID:  refID,
						Type:         ConflictCLI,
						Description:  fmt.Sprintf("conflicting CLI assertion: %s %s expects different values", a.assertion.Command, a.assertion.JSONPath),
						SkillA:       a.conn.SkillID,
						ConnectionA:  a.conn.ID,
						SkillB:       b.conn.SkillID,
						ConnectionB:  b.conn.ID,
						ExpectationA: a.assertion.ID,
						ExpectationB: b.assertion.ID,
					})
				}
			}
		}
	}
	return conflicts, nil
}

func computeMaturityScore(hasStructural, hasCLI bool, totalExpectations int) float64 {
	var score float64
	if hasStructural {
		score += 0.4
	}
	if hasCLI {
		score += 0.4
	}
	// Depth bonus: up to 0.2 for 10+ total expectations.
	cap := totalExpectations
	if cap > 10 {
		cap = 10
	}
	score += 0.2 * float64(cap) / 10.0
	return score
}

func classifyMaturity(hasStructural, hasCLI bool, total int) MaturityLevel {
	if total == 0 {
		return MaturityLow
	}
	if hasStructural && hasCLI {
		return MaturityHigh
	}
	return MaturityMedium
}

func uniqueReferenceIDs(conns []*skill.Connection) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, c := range conns {
		if !seen[c.ReferenceID] {
			seen[c.ReferenceID] = true
			ids = append(ids, c.ReferenceID)
		}
	}
	return ids
}

func groupCLIResultsByTool(rows []*CLIResultRow) map[string][]*CLIResultRow {
	m := make(map[string][]*CLIResultRow)
	for _, r := range rows {
		tool := extractToolName(r.Command)
		m[tool] = append(m[tool], r)
	}
	return m
}

// extractToolName returns the first whitespace-delimited token of a command.
func extractToolName(command string) string {
	for i, c := range command {
		if c == ' ' || c == '\t' {
			return command[:i]
		}
	}
	return command
}

// Package snapshot generates requirement snapshots for operational targets.
package snapshot

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"time"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// Writer abstracts file writing operations.
type Writer interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// osWriter implements Writer using the os package.
type osWriter struct{}

func (w *osWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (w *osWriter) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Builder builds requirement snapshots.
type Builder interface {
	// Build creates a snapshot from the current state.
	Build(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary) (*Snapshot, error)
}

// Snapshot represents a point-in-time view of requirements.
type Snapshot struct {
	// Metadata
	GeneratedAt  time.Time `json:"generated_at"`
	ScenarioName string    `json:"scenario_name,omitempty"`
	Version      string    `json:"version,omitempty"`

	// Summary statistics
	Summary SnapshotSummary `json:"summary"`

	// Operational targets
	OperationalTargets []OperationalTarget `json:"operational_targets"`

	// Module breakdown
	Modules []ModuleSnapshot `json:"modules"`
}

// SnapshotSummary contains high-level statistics.
type SnapshotSummary struct {
	TotalRequirements int     `json:"total_requirements"`
	TotalValidations  int     `json:"total_validations"`
	CompletionRate    float64 `json:"completion_rate"`
	PassRate          float64 `json:"pass_rate"`
	CriticalGap       int     `json:"critical_gap"`
}

// OperationalTarget represents a high-level business objective.
type OperationalTarget struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Priority       string   `json:"priority"`
	Status         string   `json:"status"`
	RequirementIDs []string `json:"requirement_ids"`
	CompletionRate float64  `json:"completion_rate"`
}

// ModuleSnapshot contains module-level snapshot data.
type ModuleSnapshot struct {
	Name           string  `json:"name"`
	FilePath       string  `json:"file_path"`
	Total          int     `json:"total"`
	Complete       int     `json:"complete"`
	InProgress     int     `json:"in_progress"`
	Pending        int     `json:"pending"`
	CompletionRate float64 `json:"completion_rate"`
}

// builder implements Builder.
type builder struct{}

// New creates a Builder.
func New() Builder {
	return &builder{}
}

// Build creates a snapshot from the current state.
func (b *builder) Build(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary) (*Snapshot, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	snapshot := &Snapshot{
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
	}

	// Build summary
	complete := summary.ByDeclaredStatus[types.StatusComplete]
	snapshot.Summary = SnapshotSummary{
		TotalRequirements: summary.Total,
		TotalValidations:  summary.ValidationStats.Total,
		CriticalGap:       summary.CriticalityGap,
	}

	if summary.Total > 0 {
		snapshot.Summary.CompletionRate = float64(complete) / float64(summary.Total) * 100
	}

	passed := summary.ByLiveStatus[types.LivePassed]
	failed := summary.ByLiveStatus[types.LiveFailed]
	tested := passed + failed
	if tested > 0 {
		snapshot.Summary.PassRate = float64(passed) / float64(tested) * 100
	}

	// Build operational targets from P0/P1 requirements
	snapshot.OperationalTargets = b.buildOperationalTargets(index)

	// Build module snapshots
	snapshot.Modules = b.buildModuleSnapshots(index)

	return snapshot, nil
}

// buildOperationalTargets extracts operational targets from critical requirements.
func (b *builder) buildOperationalTargets(index *parsing.ModuleIndex) []OperationalTarget {
	if index == nil {
		return nil
	}

	// Group requirements by PRD ref
	byPRDRef := make(map[string][]string) // prdRef -> []reqID
	prdTitles := make(map[string]string)
	prdPriorities := make(map[string]string)
	reqStatuses := make(map[string]types.DeclaredStatus)

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			if req.PRDRef == "" {
				continue
			}

			byPRDRef[req.PRDRef] = append(byPRDRef[req.PRDRef], req.ID)
			reqStatuses[req.ID] = req.Status

			// Use first title and criticality as representative
			if prdTitles[req.PRDRef] == "" {
				prdTitles[req.PRDRef] = req.Title
			}
			if prdPriorities[req.PRDRef] == "" && req.Criticality != "" {
				prdPriorities[req.PRDRef] = string(req.Criticality)
			}
		}
	}

	var targets []OperationalTarget
	for prdRef, reqIDs := range byPRDRef {
		// Calculate completion for this OT
		var complete, total int
		for _, reqID := range reqIDs {
			total++
			if reqStatuses[reqID] == types.StatusComplete {
				complete++
			}
		}

		var completionRate float64
		if total > 0 {
			completionRate = float64(complete) / float64(total) * 100
		}

		// Determine overall status
		status := "pending"
		if complete == total {
			status = "complete"
		} else if complete > 0 {
			status = "in_progress"
		}

		targets = append(targets, OperationalTarget{
			ID:             prdRef,
			Title:          prdTitles[prdRef],
			Priority:       prdPriorities[prdRef],
			Status:         status,
			RequirementIDs: reqIDs,
			CompletionRate: completionRate,
		})
	}

	return targets
}

// buildModuleSnapshots creates snapshot data for each module.
func (b *builder) buildModuleSnapshots(index *parsing.ModuleIndex) []ModuleSnapshot {
	if index == nil {
		return nil
	}

	var modules []ModuleSnapshot

	for _, module := range index.Modules {
		var complete, inProgress, pending int

		for _, req := range module.Requirements {
			switch req.Status {
			case types.StatusComplete:
				complete++
			case types.StatusInProgress:
				inProgress++
			case types.StatusPending, types.StatusPlanned:
				pending++
			}
		}

		total := len(module.Requirements)
		var completionRate float64
		if total > 0 {
			completionRate = float64(complete) / float64(total) * 100
		}

		modules = append(modules, ModuleSnapshot{
			Name:           module.EffectiveName(),
			FilePath:       module.FilePath,
			Total:          total,
			Complete:       complete,
			InProgress:     inProgress,
			Pending:        pending,
			CompletionRate: completionRate,
		})
	}

	return modules
}

// WriteSnapshot writes a snapshot to a file.
func WriteSnapshot(writer Writer, path string, snapshot *Snapshot) error {
	fileWriter := &jsonFileWriter{writer: writer}
	return fileWriter.WriteJSON(path, snapshot)
}

// jsonFileWriter wraps Writer for JSON writing.
type jsonFileWriter struct {
	writer Writer
}

// WriteJSON writes any value as JSON to a file.
func (w *jsonFileWriter) WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return w.writer.WriteFile(path, data, 0644)
}

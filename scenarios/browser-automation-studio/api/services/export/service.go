// Package export provides unified execution export capabilities.
//
// The ExportService consolidates multiple export pathways into a single
// interface with format strategies:
//   - MovieSpec: Video render specification for replay movies
//   - Timeline: Raw timeline data for programmatic consumption
//   - Folder: File system export with markdown reports and screenshots
//
// Each format is implemented as an ExportFormat strategy, enabling extension
// with new formats without modifying the core service.
package export

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

// ExportFormatType identifies the export format.
type ExportFormatType string

const (
	// FormatMovieSpec produces a ReplayMovieSpec for video rendering.
	FormatMovieSpec ExportFormatType = "movie-spec"
	// FormatTimeline produces raw ExecutionTimeline data.
	FormatTimeline ExportFormatType = "timeline"
	// FormatFolder exports to a file system folder with markdown reports.
	FormatFolder ExportFormatType = "folder"
	// FormatMarkdownSummary produces markdown execution summary.
	FormatMarkdownSummary ExportFormatType = "markdown-summary"
)

// ExportRequest encapsulates all parameters for an export operation.
type ExportRequest struct {
	ExecutionID uuid.UUID
	Format      ExportFormatType
	// OutputDir is required for FormatFolder, ignored for other formats.
	OutputDir string
	// StorageClient is optional for FormatFolder to fetch screenshots.
	StorageClient storage.StorageInterface
}

// ExportResult holds the output of an export operation.
// Only one of the result fields will be populated based on format.
type ExportResult struct {
	Format    ExportFormatType
	MovieSpec *ReplayMovieSpec
	Timeline  *ExecutionTimeline
	// Markdown contains generated markdown for markdown formats.
	Markdown string
	// FolderPath contains the output path for folder exports.
	FolderPath string
}

// ExportFormat defines the strategy interface for export formats.
type ExportFormat interface {
	// FormatType returns the format identifier.
	FormatType() ExportFormatType
	// Export performs the export and returns the result.
	Export(ctx context.Context, req ExportRequest, data *ExportData) (*ExportResult, error)
}

// ExportData contains all the data needed for export operations.
// This is fetched once and passed to format strategies.
type ExportData struct {
	Execution *database.ExecutionIndex
	Workflow  *database.WorkflowIndex
	Timeline  *ExecutionTimeline
}

// Service provides unified export capabilities.
type Service struct {
	repo    ExecutionRepository
	formats map[ExportFormatType]ExportFormat
	mu      sync.RWMutex
}

// ExecutionRepository defines the data access interface for exports.
type ExecutionRepository interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.WorkflowIndex, error)
}

// TimelineBuilder constructs timeline data from execution records.
type TimelineBuilder interface {
	BuildTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error)
}

// NewService creates a new export service.
func NewService(repo ExecutionRepository) *Service {
	s := &Service{
		repo:    repo,
		formats: make(map[ExportFormatType]ExportFormat),
	}
	// Register built-in formats
	s.RegisterFormat(&movieSpecFormat{})
	s.RegisterFormat(&timelineFormat{})
	s.RegisterFormat(&markdownSummaryFormat{})
	return s
}

// RegisterFormat adds a format strategy to the service.
func (s *Service) RegisterFormat(format ExportFormat) {
	if format == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.formats[format.FormatType()] = format
}

// Export executes an export operation with the specified format.
func (s *Service) Export(ctx context.Context, req ExportRequest) (*ExportResult, error) {
	s.mu.RLock()
	format, ok := s.formats[req.Format]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported export format: %s (supported: %v)", req.Format, s.SupportedFormats())
	}

	// Fetch export data
	data, err := s.fetchExportData(ctx, req.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("fetch export data: %w", err)
	}

	return format.Export(ctx, req, data)
}

// SupportedFormats returns a list of registered format types.
func (s *Service) SupportedFormats() []ExportFormatType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	formats := make([]ExportFormatType, 0, len(s.formats))
	for f := range s.formats {
		formats = append(formats, f)
	}
	return formats
}

func (s *Service) fetchExportData(ctx context.Context, executionID uuid.UUID) (*ExportData, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}

	var workflow *database.WorkflowIndex
	if wf, wfErr := s.repo.GetWorkflow(ctx, execution.WorkflowID); wfErr == nil {
		workflow = wf
	}

	// Timeline will be built by the format that needs it
	// This avoids unnecessary work for formats that don't need timeline
	return &ExportData{
		Execution: execution,
		Workflow:  workflow,
		Timeline:  nil, // Lazy loaded
	}, nil
}

// =============================================================================
// BUILT-IN FORMAT IMPLEMENTATIONS
// =============================================================================

// movieSpecFormat produces ReplayMovieSpec for video rendering.
type movieSpecFormat struct{}

func (f *movieSpecFormat) FormatType() ExportFormatType {
	return FormatMovieSpec
}

func (f *movieSpecFormat) Export(ctx context.Context, req ExportRequest, data *ExportData) (*ExportResult, error) {
	if data.Timeline == nil {
		return nil, fmt.Errorf("timeline data required for movie-spec export")
	}

	spec, err := BuildReplayMovieSpec(data.Execution, data.Workflow, data.Timeline)
	if err != nil {
		return nil, fmt.Errorf("build movie spec: %w", err)
	}

	return &ExportResult{
		Format:    FormatMovieSpec,
		MovieSpec: spec,
	}, nil
}

// timelineFormat produces raw ExecutionTimeline data.
type timelineFormat struct{}

func (f *timelineFormat) FormatType() ExportFormatType {
	return FormatTimeline
}

func (f *timelineFormat) Export(ctx context.Context, req ExportRequest, data *ExportData) (*ExportResult, error) {
	if data.Timeline == nil {
		return nil, fmt.Errorf("timeline data required for timeline export")
	}

	return &ExportResult{
		Format:   FormatTimeline,
		Timeline: data.Timeline,
	}, nil
}

// markdownSummaryFormat produces markdown execution summary.
type markdownSummaryFormat struct{}

func (f *markdownSummaryFormat) FormatType() ExportFormatType {
	return FormatMarkdownSummary
}

func (f *markdownSummaryFormat) Export(ctx context.Context, req ExportRequest, data *ExportData) (*ExportResult, error) {
	if data.Timeline == nil {
		return nil, fmt.Errorf("timeline data required for markdown export")
	}

	workflowName := ""
	if data.Workflow != nil {
		workflowName = data.Workflow.Name
	}
	if workflowName == "" {
		workflowName = "Unnamed Workflow"
	}

	markdown := GenerateTimelineMarkdown(data.Timeline, workflowName)

	return &ExportResult{
		Format:   FormatMarkdownSummary,
		Markdown: markdown,
	}, nil
}

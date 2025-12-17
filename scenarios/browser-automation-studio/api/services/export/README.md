# Export Service

Package `export` provides unified execution export capabilities with format strategies.

## Overview

The export service consolidates multiple export pathways into a single interface:

- **MovieSpec**: Video render specification for replay movies
- **Timeline**: Raw timeline data for programmatic consumption
- **Folder**: File system export with markdown reports and screenshots
- **MarkdownSummary**: Human-readable markdown execution report

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Export Service                               │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Service.Export()                       │   │
│  │                 Unified entry point                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                            │                                     │
│         ┌──────────────────┼──────────────────┐                 │
│         ▼                  ▼                  ▼                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │
│  │ MovieSpec   │   │  Timeline   │   │  Markdown   │           │
│  │  Format     │   │   Format    │   │   Summary   │           │
│  └─────────────┘   └─────────────┘   └─────────────┘           │
│         │                  │                  │                 │
│         ▼                  ▼                  ▼                 │
│  ReplayMovieSpec   ExecutionTimeline    Markdown text           │
└─────────────────────────────────────────────────────────────────┘
```

## Key Types

### ExportFormatType

Identifies the export format:

```go
const (
    FormatMovieSpec       ExportFormatType = "movie-spec"
    FormatTimeline        ExportFormatType = "timeline"
    FormatFolder          ExportFormatType = "folder"
    FormatMarkdownSummary ExportFormatType = "markdown-summary"
)
```

### ExportRequest

Encapsulates export parameters:

```go
type ExportRequest struct {
    ExecutionID   uuid.UUID
    Format        ExportFormatType
    OutputDir     string                    // For folder exports
    StorageClient storage.StorageInterface  // Optional, for screenshots
}
```

### ExportResult

Contains the export output:

```go
type ExportResult struct {
    Format     ExportFormatType
    MovieSpec  *ReplayMovieSpec
    Timeline   *ExecutionTimeline
    Markdown   string
    FolderPath string
}
```

## Usage

```go
// Create service with repository
svc := export.NewService(repo)

// Export as movie spec
result, err := svc.Export(ctx, export.ExportRequest{
    ExecutionID: executionID,
    Format:      export.FormatMovieSpec,
})
if err != nil {
    return err
}
movieSpec := result.MovieSpec

// Export as markdown
result, err = svc.Export(ctx, export.ExportRequest{
    ExecutionID: executionID,
    Format:      export.FormatMarkdownSummary,
})
markdown := result.Markdown
```

## Adding New Export Formats

Implement the `ExportFormat` interface:

```go
type myCustomFormat struct{}

func (f *myCustomFormat) FormatType() ExportFormatType {
    return "custom"
}

func (f *myCustomFormat) Export(ctx context.Context, req ExportRequest, data *ExportData) (*ExportResult, error) {
    // Build your export output
    return &ExportResult{
        Format: "custom",
        // ... populate result fields
    }, nil
}

// Register at init or runtime
svc.RegisterFormat(&myCustomFormat{})
```

## File Structure

- `service.go` - Unified export service and format registry
- `exporter.go` - Movie spec builder (`BuildReplayMovieSpec`)
- `types_timeline.go` - Timeline data types
- `types_overrides.go` - Override/preset types for customization
- `markdown.go` - Markdown report generators
- `markdown_helpers.go` - Markdown formatting utilities
- `presets.go` - Export preset configurations
- `spec_overrides.go` - Spec customization logic
- `spec_harmonizer.go` - Spec validation/harmonization

## Design Principles

This package follows:
- **Strategy pattern**: Each format is a pluggable strategy
- **Single entry point**: `Service.Export()` handles all formats
- **Lazy data loading**: Timeline data loaded only when needed
- **Extensibility**: New formats added via `RegisterFormat()`

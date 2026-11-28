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
- `exporter.go` - Movie spec builder (`BuildReplayMovieSpec`) and core types
- `types_timeline.go` - Timeline data types
- `markdown.go` - Markdown report generators
- `markdown_helpers.go` - Markdown formatting utilities
- `presets.go` - Export preset configurations (theme/cursor presets)

Generated spec construction, overrides and watermark enforcement live in this
package. The handler adapts saved preferences once for preview and binary export;
`handlers/export` only decodes HTTP requests and aliases service input types.
`JobRunner` owns admitted binary jobs, and `PublishFile` / `CommitExport` own output
publication and persisted completion. `Service.Export` remains the report/timeline
format strategy entrypoint; it is not the binary job scheduler.

## Design Principles

This package follows:
- **Strategy pattern**: Each format is a pluggable strategy
- **One owner per boundary**: report strategies, binary rendering, job lifetime and final publication each have one maintained owner
- **Lazy data loading**: Timeline data loaded only when needed
- **Extensibility**: New formats added via `RegisterFormat()`

## Continuous recordings and presentation (2026-10-16)

A recording remains continuous source media. Styled MP4/GIF exports now pass that
source through the same `ReplayPlayer` / `ReplayStyleFrame` presentation owner as
preview and screenshot replay. `ReplaySpec.recorded_video` supplies the measured
source; the server sets its execution-owned recording URL. The source's measured
raster and duration replace editorial screenshot timing. The original is retained.
Non-WebM containers use an owned temporary WebM derivative for browser decoding;
source audio is restored during final MP4 mastering. Temporary derivatives and
browser/encoder resources are cleaned up on completion or failure.

`ReplayDecor` now carries device-frame theme, explicit desktop/browser/device
visibility and browser scale. Optional booleans preserve false across protobuf
round-trips. Saved preferences and spec adapters retain these fields. Device bezels
surround the product window, inside the canvas; they do not wrap the entire export.

The REST request accepts `stylization: raw|stylized`. Raw preserves source pixels
and bypasses presentation even when a preview spec is supplied. Omitted intent
retains spec/override clients, applying their requested styling. Styled recordings
support MP4/GIF; raw WebM remains available. Enabled intro/outro cards are explicitly
refused for continuous video rather than silently ignored; use a composition edit
or screenshot replay. Required watermark policy still applies to styled exports.

Cursor pixels already in a recording cannot be replaced. Continuous exports keep
those pixels and disable unaligned replay cursor overlays. The receipt records
`cursor_application: source-pixels; replay-overlay-disabled`. Choose a cursor
before capture, or select screenshot replay for editable pointer trails. No fake
second pointer is added. Timed edit-map/cursor parity remains a separate gap.

Native capture uses DPR 1 because the requested canvas is already in output
pixels. Incidental navigation/evaluate screenshots are disabled. Explicit capture
frames cover the full timeline; the old 120-frame prefix clamp is removed. Requests
above the configured frame budget fail with lower-FPS/shorter-section guidance;
they never silently truncate. Canvas limits are 4096 per axis and 4096×2160 pixels;
retained screenshots are bounded at 128 MiB. Raster mismatches fail explicitly.
The fixed canvas hides root overflow rather than painting a scrollbar into the
movie. FFmpeg encoding/filter workers are bounded to two threads.

Validation uses focused Go/proto/UI regressions plus a separate admitted native
integration test. Build UI into an isolated output directory, then set
`BAS_REPLAY_TEST_DIST`, `PLAYWRIGHT_DRIVER_URL` and an optional new
`BAS_REPLAY_TEST_EVIDENCE` prefix. Run:

```bash
go test -tags integration ./services/export/render -run '^TestPlaywright(Capture|PhoneFrame)Integration$' -count=1 -v
```

The test uses synthetic source media and an isolated static fixture server. It
checks full timeline pixels and an encoded MP4 with audio, plus phone framing.
It does not certify real-product creative quality, every codec, remote drivers,
all cursor modes or a live installation. Retain exact build/export evidence.

## Export job reliability contract (2026-10-16)

Binary REST exports have one preparation, scenario-owned runner and finalization
path. Supported formats are admitted explicitly; JSON/HTML remain synchronous and
folder export remains its existing operation. Saved settings and explicit overrides
are resolved before policy enforcement. Replay and styled continuous media share
watermark policy; raw recording pixels retain their existing policy and cursor limits.

The runner admits at most two active exports, gives each a fifteen-minute deadline,
and owns cancellation/drain independently of HTTP disconnect. Explicit cancellation
and process interruption use the existing `failed` status with an explanatory error;
no new wire status is introduced. Startup marks previous-process processing exports
interrupted before accepting new work. Shutdown cancels/drains jobs before closing
the driver or database. Interrupted encoding is not automatically resumed.

Publication writes a unique sibling staging file and atomically links a complete
file into its requested directory without replacing existing files. A filename is
one path component; traversal and symlink destinations are refused. Completion is
broadcast only after its conditional database transition succeeds. A completed
file whose database finalization fails is retained with a sibling recovery receipt;
it is never falsely announced as completed. Consumers recover from persisted state
on reconnect and fence stale replies by export identity.

Cancellation is exposed through `ExportsService.CancelExport` and
`browser-automation-studio exports cancel <id>`. `cancellation_requested` acknowledges
a request only; `exports status <id>` supplies the persisted terminal result. Library
deletion also requests owned cancellation. Status persistence failures remain explicit
server diagnostics rather than false terminal events. Successful completion atomically
stores the final SHA-256/settings together with path and size.

Recorded source ingestion verifies actual size/hash, bounds regular source files and
remote downloads at 512 MiB, inline sources at 64 MiB, and honors the job context.
Typed timeline aliases are deduplicated by stable reference and verified byte identity;
distinct contents are refused rather than selected by timeline order. Inline sources
are measured individually before alias comparison. Older local recordings filter
supported regular media files, include measured hashes and refuse ambiguous multiples.
Local source acquisition and freezing reject nonregular files and symlink leaves before
open; Unix uses nonblocking/no-follow open and verifies the opened inode to close the
FIFO replacement race. Non-Unix replacement-race behavior remains unqualified.

Raw copy shortcuts require a container matching its filename, a measured video raster
and finite duration within 600 seconds, and a strict full decode of the selected video
and optional audio streams with at least one decoded video frame. Probe and decode
have 15-second and 90-second bounds; decode uses one input/output worker thread.
Corrupt bytes never reach successful raw publication just because their suffix is
MP4 or WebM. Immutable 0.12-second 16×16 H.264/VP9 test fixtures are embedded under
`internal/testutil/videofixture`; tests do not generate media or depend on live capture.
Terminal updates require a confirmed single affected database row; driver errors and
unconfirmed counts are failures, so they cannot authorize completion events.
Per-job bounds do not certify a total process memory ceiling or full-resolution speed.

Driver unit coverage excludes the five-minute managed motion qualification. Its
explicit owner command is `pnpm run test:motion` in `playwright-driver`, with
`BAS_REHAB_LIVE_API_BASE`, `BAS_REHAB_LIVE_UI_BASE` and
`BAS_MOTION_OBSERVATION_PATH`; missing inputs remain failures there. Journey and
motion qualification outcomes must be retained separately from unit coverage.

# Smoke Test Pipeline Stage

The smoke test is **stage 5 of 6** in the scenario-to-desktop desktop deployment pipeline. It serves as a quality gate that validates built application artifacts can start correctly and report telemetry before deploy.

## Pipeline Position

```
┌─────────┐   ┌───────────┐   ┌──────────┐   ┌───────┐   ┌───────────┐   ┌──────────────┐
│ Bundle  │ → │ Preflight │ → │ Generate │ → │ Build │ → │ SmokeTest │ → │ Deploy       │
└─────────┘   └───────────┘   └──────────┘   └───────┘   └───────────┘   └──────────────┘
                                                              ▲
                                                         YOU ARE HERE
```

The smoke test stage:
- **Depends on**: Build stage (requires compiled artifacts)
- **Blocks**: Deploy stage (won't proceed unless smoke test passes)
- **Skippable**: Yes, via `SkipSmokeTest` configuration flag

---

## Execution Flow

```
                        ┌──────────────────────────────────────────────┐
                        │           SMOKE TEST STAGE                   │
                        └──────────────────────────────────────────────┘
                                           │
                    ┌──────────────────────┴──────────────────────┐
                    ▼                                              │
           ┌────────────────┐                                     │
           │ 1. INITIALIZE  │                                     │
           │ - Check skip   │                                     │
           │ - Validate svc │                                     │
           └───────┬────────┘                                     │
                   │                                              │
                   ▼                                              │
           ┌────────────────┐                                     │
           │ 2. SELECT      │◄── Prefers current platform         │
           │ ARTIFACT       │    Falls back to any available      │
           │ (from build)   │                                     │
           └───────┬────────┘                                     │
                   │                                              │
                   ▼                                              │
           ┌────────────────┐     ┌─────────────────────────┐    │
           │ 3. PREPARE     │────►│ ENV VARS SET:           │    │
           │ - Generate ID  │     │ SMOKE_TEST=1            │    │
           │ - Init status  │     │ SMOKE_TEST_TIMEOUT_MS   │    │
           │ - Setup env    │     │ SMOKE_TEST_UPLOAD_URL   │    │
           └───────┬────────┘     └─────────────────────────┘    │
                   │                                              │
                   ▼                                              │
     ┌─────────────┴─────────────────────────────────┐           │
     │         4. PLATFORM-SPECIFIC EXECUTION        │           │
     └───────────────────────────────────────────────┘           │
             │              │              │                      │
             ▼              ▼              ▼                      │
     ┌───────────┐   ┌───────────┐   ┌───────────┐               │
     │  LINUX    │   │  WINDOWS  │   │   macOS   │               │
     │ AppImage  │   │   .exe    │   │   .app    │               │
     ├───────────┤   ├───────────┤   ├───────────┤               │
     │chmod +x   │   │ Direct    │   │ Resolve   │               │
     │xvfb-run   │   │ execution │   │ Contents/ │               │
     │(headless) │   │           │   │ MacOS/bin │               │
     └─────┬─────┘   └─────┬─────┘   └─────┬─────┘               │
           └───────────────┼───────────────┘                      │
                           ▼                                      │
              ┌─────────────────────────┐                         │
              │   ./app --smoke-test    │◄─── All platforms       │
              │   (30 second timeout)   │     use this flag       │
              └────────────┬────────────┘                         │
                           │                                      │
                           ▼                                      │
              ┌────────────────────────────┐                      │
              │    5. ASYNC MONITORING     │                      │
              │ ┌────────────────────────┐ │                      │
              │ │ Poll every 500ms       │ │                      │
              │ │ Max wait: 2 minutes    │ │                      │
              │ │ Capture stdout/stderr  │ │                      │
              │ └────────────────────────┘ │                      │
              └────────────┬───────────────┘                      │
                           │                                      │
                           ▼                                      │
              ┌────────────────────────────┐                      │
              │   6. CHECK OUTPUT FOR:     │                      │
              │  "SMOKE_TEST_RESULT=passed"│                      │
              │  "SMOKE_TEST_UPLOAD=ok"    │                      │
              └────────────┬───────────────┘                      │
                           │                                      │
           ┌───────────────┼───────────────┐                      │
           ▼               ▼               ▼                      │
     ┌──────────┐   ┌──────────┐   ┌───────────────┐             │
     │ PASSED   │   │ FAILED   │   │ UPLOAD FAILED │             │
     │ (exit 0  │   │ (exit≠0  │   │ (fallback to  │             │
     │  +marker)│   │  or no   │   │  disk read)   │             │
     └────┬─────┘   │  marker) │   └───────┬───────┘             │
          │         └────┬─────┘           │                      │
          │              │                 ▼                      │
          │              │     ┌────────────────────────┐        │
          │              │     │ 7. TELEMETRY FALLBACK  │        │
          │              │     │ Read from:             │        │
          │              │     │ Win: %APPDATA%/...     │        │
          │              │     │ Mac: ~/Library/...     │        │
          │              │     │ Linux: ~/.config/...   │        │
          │              │     └──────────┬─────────────┘        │
          │              │                │                       │
          └──────────────┴────────────────┘                       │
                           │                                      │
                           ▼                                      │
              ┌────────────────────────────┐                      │
              │   8. UPDATE STATUS         │                      │
              │  - Record completion time  │                      │
              │  - Store execution logs    │                      │
              │  - Persist to disk         │                      │
              └────────────────────────────┘                      │
                           │                                      │
                           └──────────────────────────────────────┘
```

---

## Data Flow

```
┌─────────────────┐          ┌─────────────────┐          ┌────────────────┐
│   BUILD STAGE   │          │  SMOKE TEST     │          │  DEPLOY STAGE  │
│                 │          │     STAGE       │          │     STAGE      │
│ BuildStatus {   │          │                 │          │                │
│  PlatformResults│─────────►│ SmokeTestResult │─────────►│ Only proceeds  │
│  - linux.amd64  │  reads   │  - passed/failed│ if pass  │ if smoke test  │
│  - darwin.arm64 │ artifact │  - logs         │          │ succeeded      │
│  - windows.amd64│          │  - telemetry    │          │                │
│ }               │          │                 │          │                │
└─────────────────┘          └─────────────────┘          └────────────────┘
```

---

## Key Implementation Files

| File | Role |
|------|------|
| [CODE: api/pipeline/stage_smoketest.go] | Stage orchestration (polling, waiting, result handling) |
| [CODE: api/smoketest/service.go] | Actual test execution logic (command resolution, process mgmt) |
| [CODE: api/smoketest/store.go] | Persists test status to disk |
| [CODE: api/smoketest/interfaces.go] | Service contracts (Service, Store, CancelManager) |
| [CODE: api/smoketest/types.go] | Data structures (Status, StartRequest, CancelResponse) |
| [CODE: api/pipeline/config_defaults.go] | Timeout and polling constants |

---

## Configuration

### Pipeline Config Options

```go
type Config struct {
    // Skip the smoke test entirely
    SkipSmokeTest bool `json:"skip_smoke_test,omitempty"`

    // Stop after smoke test completes (don't run deploy)
    StopAfterStage string `json:"stop_after_stage,omitempty"` // "smoketest"

    // Resume from smoke test (if previously stopped)
    ResumeFromStage string `json:"resume_from_stage,omitempty"` // "smoketest"
}
```

### Default Timeouts

| Constant | Value | Purpose |
|----------|-------|---------|
| `DefaultSmokeTestTimeout` | 2 minutes | Max time to wait for test completion |
| `DefaultSmokePollInterval` | 500ms | How often to check test status |
| `SMOKE_TEST_TIMEOUT_MS` | 30000 (30s) | App startup timeout passed via env var |

### Environment Variables Set During Test

| Variable | Value | Purpose |
|----------|-------|---------|
| `SMOKE_TEST` | `1` | Signals app is running in smoke test mode |
| `SMOKE_TEST_TIMEOUT_MS` | `30000` | Timeout for app startup |
| `SMOKE_TEST_UPLOAD_URL` | `http://127.0.0.1:{port}/api/v1/deployment/telemetry` | Where to upload telemetry |

---

## Platform-Specific Execution

### Linux (AppImage)

```bash
# Make executable
chmod +x ./app.AppImage

# Run with xvfb if DISPLAY not set (headless)
xvfb-run ./app.AppImage --smoke-test

# Or direct execution if DISPLAY is set
./app.AppImage --smoke-test
```

### Windows (.exe)

```powershell
# Direct execution
.\app.exe --smoke-test
```

### macOS (.app bundle)

```bash
# Resolve executable inside bundle
EXECUTABLE=./MyApp.app/Contents/MacOS/MyApp

# Execute
$EXECUTABLE --smoke-test
```

---

## Success Criteria

For a smoke test to **pass**, the application output must contain:

```
SMOKE_TEST_RESULT=passed
```

For telemetry to be considered **successfully uploaded**:

```
SMOKE_TEST_UPLOAD=ok
```

If telemetry upload fails during the test, the fallback mechanism reads from the app's data directory:

| Platform | Telemetry Path |
|----------|----------------|
| Windows | `%APPDATA%\{AppName}\deployment-telemetry.jsonl` |
| macOS | `~/Library/Application Support/{AppName}/deployment-telemetry.jsonl` |
| Linux | `~/.config/{AppName}/deployment-telemetry.jsonl` |

---

## Storage & Persistence

### Smoke Test Store

The smoke test service persists test status to:
```
<data-root>/vrooli/scenario-to-desktop/smoke_tests_v2.json
```

### Status Data Structure

```go
type Status struct {
    SmokeTestID          string     `json:"smoke_test_id"`
    ScenarioName         string     `json:"scenario_name"`
    Platform             string     `json:"platform"`
    Status               string     `json:"status"` // running, passed, failed
    ArtifactPath         string     `json:"artifact_path"`
    StartedAt            time.Time  `json:"started_at"`
    CompletedAt          *time.Time `json:"completed_at,omitempty"`
    Logs                 []string   `json:"logs,omitempty"`
    Error                string     `json:"error,omitempty"`
    TelemetryUploaded    bool       `json:"telemetry_uploaded"`
    TelemetryUploadError string     `json:"telemetry_upload_error,omitempty"`
}
```

---

## Async Execution Pattern

The smoke test uses an **asynchronous execution with polling** pattern:

```go
// Step 1: Start async smoke test (non-blocking)
go s.service.PerformSmokeTest(ctx, smokeTestID, scenarioName, artifactPath, platform)

// Step 2: Poll for completion (blocking with timeout)
smokeStatus, err := s.waitForSmokeTest(ctx, smokeTestID)
```

### Polling Implementation

```go
func (s *SmokeTestStage) waitForSmokeTest(ctx context.Context, smokeTestID string) (*smoketest.Status, error) {
    timeout := time.After(DefaultSmokeTestTimeout)  // 2 minutes
    ticker := time.NewTicker(DefaultSmokePollInterval)  // 500ms

    for {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("smoke test cancelled")
        case <-timeout:
            return nil, fmt.Errorf("smoke test timed out after %v", DefaultSmokeTestTimeout)
        case <-ticker.C:
            status, ok := s.store.Get(smokeTestID)
            if !ok { continue }  // Not yet registered

            switch status.Status {
            case "passed", "failed":
                return status, nil  // Complete
            }
        }
    }
}
```

---

## Error Handling

### Common Failure Modes

| Failure | Cause | Resolution |
|---------|-------|------------|
| Artifact not found | Build stage didn't produce expected files | Re-run build stage |
| DISPLAY not set | Can't run GUI on headless Linux | Install xvfb-run |
| Timeout | App didn't start within 2 minutes | Check app startup time, increase timeout |
| Missing marker | App didn't report `SMOKE_TEST_RESULT=passed` | Add smoke test support to app |
| Process crash | App crashed during startup | Check app logs for errors |

### Logging

All test execution is logged and stored in the smoke test status:
- Command executed
- Exit code
- Output (truncated to first 500 bytes if longer)
- Telemetry upload status
- Completion status

---

## Seam Architecture

The smoke test stage follows the scenario's seam architecture for testability:

| Seam | Interface | Purpose |
|------|-----------|---------|
| `Service` | `smoketest.Service` | Abstracts test execution |
| `Store` | `smoketest.Store` | Abstracts status persistence |
| `CancelManager` | `smoketest.CancelManager` | Manages cancellation of running tests |

See [SEAMS.md](../internal/SEAMS.md) for the full seam architecture.

---

## Screen Recording

When `ScreenRecordingConfig.Enabled` is set on a smoke test status, the service records the virtual display during execution. This enables visual validation — a human reviewer can watch the recording to confirm the app launched correctly.

### How It Works

1. **Display creation**: `DisplayManager.CreateDisplay()` starts an Xvfb instance (Linux only)
2. **Capture start**: `Recorder.StartCapture()` calls `resource-ffmpeg screen-capture start` to begin x11grab recording
3. **Execution**: The smoke test runs on the virtual display (xvfb-run wrapper is skipped since the display is managed directly)
4. **Capture stop**: After execution completes (pass or fail), `Recorder.StopCapture()` finalizes the video
5. **Result storage**: The `ScreenRecordingResult` is stored in the smoke test status

### Video Serving

Recorded videos are served via `GET /api/v1/smoketest/{id}/video` with Range header support for browser playback. The deployment-manager downloads videos through this endpoint for its review workflow.

### Configuration

```json
{
  "recording_config": {
    "enabled": true,
    "display_width": 1920,
    "display_height": 1080,
    "fps": 15
  }
}
```

Defaults: 1920x1080 at 15fps. Lower FPS keeps file sizes reasonable for review purposes.

---

## Related Documentation

- [API Architecture](./api-architecture.md) - Overall pipeline system overview
- [Telemetry](telemetry.md) - Telemetry collection and upload
- [Build & Packaging](../guides/build-and-packaging.md) - Prerequisites and build commands
- [SEAMS.md](../internal/SEAMS.md) - Integration boundaries and testability patterns

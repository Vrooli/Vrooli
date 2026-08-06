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

## Journey evidence contract

When visual evidence is enabled, the normal run launches a second, non-smoke
demo process on the recording display and executes the registered capability
plan. Hello Desktop is registered as `hello-desktop`; the generic runner has no
scenario-name branch. Each chapter records its purpose, action, bounded
readiness policy, settle policy, expected/observed result, assertion status,
capture references, and monotonic/wall-clock timing. Readiness and settle
events are in the same timeline.

The persisted journey sidecar uses `journey-evidence.v2`. The status API exposes
an `EvidenceReview` projection, while the raw journey and recording remain
producer-owned captures. The UI must display the backend verdict verbatim:
`pass`, `failed`, `degraded`, `unavailable`, `unsupported`, or `not_run`.

Visual pass requires all of the following:

- the capability and plan are registered;
- the target window and semantic assertions pass;
- every required chapter has before/after evidence;
- the MP4 decodes and contains useful application frames;
- timeline order, chapter coverage, checksums, persistence, and redaction pass.

An unavailable host capability or missing recording offset is visible and
never promoted to pass. Cross-platform compilation/package checks do not imply
native Windows or macOS visual execution.

### Launch-performance trace alignment

The protocol and demo are separate producer runs even though one smoke-test
status owns both. The protocol trace records validation and cleanup; the demo
trace is aligned with the recording and owns the user-visible launch timeline.
Trace events use monotonic timestamps for durations and wall-clock timestamps
for review alignment. The demo event sequence distinguishes process creation,
Electron readiness, splash first paint, runtime/server readiness, main-window
load/show, and app readiness. A missing event is unavailable evidence, not a
zero-duration phase.

Trace artifacts are redacted and checksum-addressed in the producer manifest.
They are reference-only when deployment-manager receives the evidence report;
raw trace/profile bytes remain producer-owned. See the [live desktop API
reference](live-desktop-api.md#launch-performance-evidence) for the phase
projection, role attribution, optional profiling modes, and comparability rules.

## Pacing profiles

The runner uses explicit bounded policies rather than unlabelled sleeps. The
The `normal-review` profile leaves a named visual settle window for human review.
`fast-ci` shortens bounded readiness and settle windows for deterministic CI;
`diagnostic-slow` lengthens them within explicit upper bounds. Select them with
`S2D_JOURNEY_PROFILE`; unknown values fail closed. `S2D_JOURNEY_CAPABILITY`
selects a registered behavior fixture when a journey is not the baseline
scenario identity. The app demo hold remains owned by this smoke orchestration
(`SMOKE_TEST_DEMO_HOLD_MS`); a journey step may not silently extend it.

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
| `DEPLOYMENT_MANAGER_URL` | Optional Connect base URL | Enables reference-only `ReportTargetVerdict` after the journey |
| `DEPLOYMENT_MANAGER_PROFILE_ID` | Optional profile ID | Identifies the release profile for the evidence report |
| `S2D_JOURNEY_CAPABILITY` | Optional registered capability | Selects a behavior fixture such as bundled-private or shared-resource |
| `S2D_JOURNEY_PROFILE` | `normal-review`, `fast-ci`, or `diagnostic-slow` | Selects bounded journey pacing; unknown profiles fail closed |
| `VROOLI_GIT_COMMIT` | Optional exact commit hash | Binds the evidence report to the reviewed source |

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

## Decision-grade desktop journey

When recording is enabled, capture starts before the demo process starts. After
the smoke-test process passes, the service keeps the normal app open and runs a
bounded journey: wait for a usable application window (not the generated
400x300 splash), activate it, maximize it, run the fixture's semantic action,
click, send Return, resize, move, and close it. Every interaction has a
screenshot before and after; window actions also persist geometry. The ordered
step list is stored as a `journey` capture beside the recording and screenshots.

Hello Desktop is the canonical deterministic fixture. Its journey types a
run-specific name, activates `Say Hello`, and verifies the test bridge observed
the exact `Hello, <name>!` state. The semantic step records an assertion ID,
expected state, and observed state. Generic pointer and keyboard actions remain
structural diagnostics and cannot replace this application-level assertion.

The journey is `pass` only when Linux, xdotool, a started titlebar-capable
window manager, a usable application window, successful interactions, and a
maximize geometry of at least 90% of the display are all observed. A desktop
root/helper window (including a tiny 1x1 window) is not an application window
and cannot satisfy this contract. Missing prerequisites are explicit degraded
outcomes and never pass:

- `platform_not_linux`
- `window_manager_not_started`
- `window_manager_titlebar_unavailable`
- `xdotool_unavailable`
- `no_visible_window`

An action, screenshot, geometry read, or maximize assertion failure is a
failed journey. Each action is bounded by the smoke-test context so a hung
desktop command cannot hold the pipeline indefinitely.

When recording is enabled, a smoke test that requested desktop evidence also
requires a passing journey. A successful `--smoke-test` protocol alone is not
enough to approve a video: an absent application window, failed journey, or
missing journey is a smoke-test failure. The recording and journey captures
are retained so the failed evidence can be diagnosed.

### Capture integrity and manifest

After the recorder stops, the producer runs `ffprobe` and decodes sampled
frames with FFmpeg. A recording must be a non-empty MP4 with a video stream,
positive dimensions and duration, and at least one bright application frame
across the recording; the uniform dark Xvfb desktop and cursor do not satisfy
that gate. The producer persists a versioned manifest beside the MP4 at
`<recording>.manifest.json`. It contains the artifact digest, target and
runner identity, capture checksums and absolute paths, media dimensions and
duration, journey assertion, and each required gate disposition.

The default local profile is `visual`. `release_visual` additionally requires
successful reference-only reporting to deployment-manager; unavailable
governance is never converted into a release pass. Windows and macOS remain
compile/package results until a native or remote runner executes this same
visual contract.

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
5. **Result storage**: The recording status stores only the canonical capture identity and checksum in the smoke test status

### Video Serving

Recorded videos remain producer-owned and are served via the canonical captures
route with Range header support for browser playback. deployment-manager stores
only the capture identity, checksum, kind, and size through
`EvidenceService.ReportTargetVerdict`; it never downloads or stores the video.

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

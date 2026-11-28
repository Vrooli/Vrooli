# Playwright Driver v2.0

**Production-ready TypeScript browser automation driver for Vrooli Ascension**

## Overview

Complete TypeScript rewrite of the Playwright driver with:
- ✅ **All 28 instruction types implemented** (100% feature coverage)
- ✅ **Type-safe** with strict TypeScript and Zod validation
- ✅ **Modular architecture** - Clean separation of concerns
- ✅ **Production-ready** - Logging, metrics, error handling
- ✅ **Contract-compliant** - Implements all Go automation contracts
- ✅ **Multi-tab support** - `AllowsParallelTabs: true`
- ✅ **Iframe navigation** - frame-switch implemented

## Quick Start

### Development
```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Or use compiled version
npm run build
npm start
```

### Production (Docker)
```bash
# Build and run with Docker Compose
docker-compose up -d

# Check health
curl http://localhost:39400/health

# View logs
docker-compose logs -f playwright-driver
```

### Configuration

**Environment Variables**:
```bash
PLAYWRIGHT_DRIVER_PORT=39400          # HTTP port
PLAYWRIGHT_DRIVER_HOST=127.0.0.1      # Bind address
HEADLESS=true                         # Run headless
MAX_SESSIONS=10                       # Max concurrent sessions
LOG_LEVEL=info                        # debug | info | warn | error
METRICS_ENABLED=true                  # Enable Prometheus metrics
# Legacy default-browser WAV source for deterministic getUserMedia sessions.
# Prefer per-execution workflow settings.fake_media.microphone_wav, which
# pools a dedicated fake-media browser per WAV without env configuration.
BAS_FAKE_MICROPHONE_FILE=/absolute/path/to/reference.wav
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#configuration) for the full configuration reference.

## API Endpoints

- `GET /health` - Health check
- `POST /session/start` - Create session
- `POST /session/:id/run` - Execute instruction
- `POST /session/:id/reset` - Reset session state
- `POST /session/:id/close` - Close session
- `GET /metrics` - Prometheus metrics (if enabled)

See [docs/API.md](docs/API.md) for complete API specification.

## Supported Instructions (28 total)

### Navigation & Frames
- `navigate` - Navigate to URL
- `frame-switch` - Enter/exit iframes

### Interaction
- `click`, `hover`, `type` - Basic interactions
- `focus`, `blur` - Focus management
- `drag-drop`, `swipe`, `pinch` - Gestures

### Input
- `uploadfile` - File upload
- `select` - Dropdown selection
- `keyboard` - Keyboard events

### Wait & Assert
- `wait` - Wait for element or timeout
- `assert` - Element assertions

### Data Extraction
- `extract` - Extract element data
- `evaluate` - Execute JavaScript

### Capture
- `screenshot` - Capture screenshot
- `download` - Handle downloads

### Navigation
- `scroll` - Scroll to element/coordinates

### Storage
- `cookie-storage` - Cookie/localStorage operations

### Multi-Context
- `tab-switch` - Multi-tab management

### Network
- `network-mock` - Request interception/mocking

### Device
- `rotate` - Device orientation

## Architecture

**Purpose-Driven Modular Design** - Each directory screams its responsibility:

```
src/
├── server.ts              # Application entrypoint and bootstrap
├── config.ts              # Configuration loading (environment → typed config)
├── constants.ts           # Shared constants and defaults
│
├── types/                 # Type Definitions & Contracts
│   ├── contracts.ts       # Go API wire format (StepOutcome, DriverOutcome)
│   ├── session.ts         # Session domain types + HTTP request/response
│   └── instruction.ts     # Zod schemas for instruction parameter validation
│
├── session/               # Session Lifecycle Management
│   ├── manager.ts         # Session CRUD, browser pooling, resource limits
│   ├── cleanup.ts         # Idle session reaper
│   └── context-builder.ts # Playwright BrowserContext factory
│
├── handlers/              # Instruction Execution (28 types)
│   ├── base.ts            # Handler interface and base utilities
│   ├── registry.ts        # Handler lookup by instruction type
│   ├── navigation.ts      # navigate
│   ├── interaction.ts     # click, hover, type, focus, blur
│   ├── wait.ts            # wait
│   ├── assertion.ts       # assert
│   └── ...                # extraction, screenshot, scroll, frame, etc.
│
├── recording/             # Record Mode Feature
│   ├── controller.ts      # Recording lifecycle orchestration
│   ├── buffer.ts          # In-memory action buffer
│   ├── selectors.ts       # Multi-strategy selector generation
│   ├── injector.ts        # Browser event listener injection
│   └── types.ts           # RecordedAction, SelectorSet, etc.
│
├── telemetry/             # Observability Data Collection
│   ├── screenshot.ts      # Screenshot capture with caching
│   ├── dom.ts             # DOM snapshot extraction
│   └── collector.ts       # Console logs, network events
│
├── outcome/               # Wire-Format Transformation
│   └── outcome-builder.ts # HandlerResult → StepOutcome → DriverOutcome
│
├── routes/                # HTTP Transport Layer
│   ├── router.ts          # Lightweight path-matching router
│   ├── health.ts          # GET /health
│   ├── session-start.ts   # POST /session/start
│   ├── session-run.ts     # POST /session/:id/run
│   └── record-mode/       # Recording API endpoints
│       ├── recording-lifecycle.ts   # start/stop/status/actions
│       ├── recording-validation.ts  # selector validation, replay
│       ├── recording-navigation.ts  # URL navigation, back/forward
│       ├── recording-frames.ts      # frame capture, screenshots
│       ├── recording-input.ts       # pointer, keyboard, viewport
│       ├── recording-pages.ts       # multi-tab page management
│       └── recording-diagnostics-routes.ts # debug, testing
│
├── middleware/            # HTTP Request/Response Processing
│   ├── body-parser.ts     # JSON body parsing with size limits
│   └── error-handler.ts   # Error normalization and HTTP responses
│
└── utils/                 # Cross-Cutting Infrastructure
    ├── logger.ts          # Winston logger with scoped contexts
    ├── metrics.ts         # Prometheus metrics definitions
    ├── metrics-server.ts  # Standalone metrics HTTP server
    └── errors.ts          # Typed error classes (SessionNotFound, etc.)
```

**Key Flows:**

1. **Instruction Execution**: `routes/session-run.ts` → `handlers/` → `outcome/outcome-builder.ts` → HTTP response
2. **Recording**: `routes/record-mode/` → `recording/controller.ts` → `recording/buffer.ts`
3. **Session Lifecycle**: `routes/session-*.ts` → `session/manager.ts` → Playwright

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Testing

```bash
# Run mocked unit contracts (no native browser qualification)
pnpm test

# Watch mode
npm run test:watch

# Mocked unit coverage (unchanged floors and full source denominator)
pnpm run test:coverage

# Type checking
npm run typecheck

# Linting
npm run lint
```

**Current Coverage**: ~70% (target: >90%)

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [API Specification](docs/API.md) - HTTP API reference
- [Control Surface](CONTROL-SURFACE.md) - Deployment inputs and endpoints

## Performance

**Typical Metrics**:
- Session startup: < 2s
- Instruction overhead: < 100ms
- Screenshot capture: < 500ms
- DOM snapshot: < 50ms

**Resource Usage**:
- Memory per session: ~50-150 MB
- CPU: Variable (screenshots are intensive)
- Max concurrent sessions: 10 (configurable)

## Integration with Go API

```bash
# Set driver URL
export PLAYWRIGHT_DRIVER_URL=http://localhost:39400

# Verify in Go
cd ../../engine
go test -v
```

The Go `PlaywrightEngine` (api/automation/engine/playwright_engine.go) communicates with this driver over HTTP.

## Migration from v1.0

v2.0 is a complete TypeScript rewrite with:
- 28 instruction types (vs 13 in v1.0)
- Full type safety
- Modular architecture
- Enhanced error handling
- Production-ready observability

**Breaking Changes**: None - API is backward compatible

## Contributing

1. Follow TypeScript best practices
2. Add tests for new handlers
3. Update documentation
4. Run `npm run lint` before committing

## License

MIT

---

**Version**: 2.0.0
**Status**: ✅ Production Ready
**Feature Completeness**: 28/28 (100%)
**Build Status**: ✅ Passing

### Managed motion qualification

`pnpm run test:motion` explicitly runs the five-minute managed motion/slow-reader
qualification. Set `BAS_REHAB_LIVE_API_BASE`, `BAS_REHAB_LIVE_UI_BASE` and
`BAS_MOTION_OBSERVATION_PATH` through the admitted qualification owner. It fails
when inputs are missing. Ordinary `test:coverage` does not implicitly run this
live workload; its unit floors remain unchanged. Preservation journeys still use
`test:journeys` and their maintained shadow ownership.


### Driver test boundaries

Default `test`, `test:unit`, `test:watch` and `test:coverage` select mocked contracts
under `tests/unit` and `tests/recording`. Browser launch in a unit contract must use
the existing `playwrightProvider` spy seam and shared Page/Context/Browser fixtures.
The unit setup refuses unmocked native launch/attach before browser work; mocked
fixtures can explicitly override that guard. Local HTTP contract fixtures remain
subject to their declared workspace network policy; this is not full hermeticity.
A directory name alone does not prove that its collaborators are mocked.

Explicit admitted commands from this directory:

- `pnpm run test:selection` lists each maintained Jest selection, verifies every
  ordinary test file is assigned exactly once, and prints file identities. No test
  body or browser runs. Motion and journeys are checked as separate preserved sets.
- `pnpm run test:integration` runs local Chromium/Electron/media contracts through
  `jest.integration.config.js`, one Jest worker at a time. Selecting the Electron
  file explicitly requires `VROOLI_ELECTRON_INTEGRATION_APP` and `DISPLAY`; missing
  prerequisites fail rather than skip. Other selected native files do not require
  those Electron inputs.
- `pnpm run test:e2e:pipeline` runs only the local native pipeline file, including
  its 10,000-action cohorts. `pnpm run test:recording` explicitly runs the unit
  recording subset followed by its native recording subset and needs native admission.
- `pnpm run test:managed` runs saved-workflow fresh-context/live journal cases through
  `jest.managed.config.js`; `BAS_REHAB_LIVE_API_BASE` must be an explicit HTTP(S) URL. This entrypoint
  also owns the whole input-feedback file, including its local diagnostics and
  live cohort; that file additionally requires `BAS_REHAB_LIVE_UI_BASE`. The local
  diagnostics remain accessible through this managed command, without duplicating
  shared probes or implicitly enabling a live cohort in local-native qualification.
  Missing input fails instead of silently skipping. The operator owns an admitted
  existing BAS runtime and routed SQL/file test leases; this command does not start
  or restart a scenario.
- `pnpm run test:motion` and `pnpm run test:journeys` retain their separate managed
  qualification contracts described above and their maintained fixture owners.

Unit, local-native and managed saved-workflow use the same production source
coverage denominator and numeric floors (branches10/functions15/lines30/statements30),
with separate `coverage`, `coverage/native` and `coverage/managed` outputs. Moving
native cases out of unit can expose a coverage failure; keep it visible. Each
command retains only its own evidence. No combined coverage or full qualification
is claimed without a supported merger that rejects missing shards. Existing motion
and journey qualification deliberately do not collect coverage.

Root must admit native/browser/media work separately from production capture. For
bounded qualification, select an exact file, for example
`pnpm run test:integration --runTestsByPath tests/integration/session-admission-capacity.test.ts`.
Native and managed reporters identify the active suite/case and print a bounded
15-second activity heartbeat, leaving the declared per-case/hook deadlines intact.
The local pipeline has 120-second cases; the managed journal has a 600-second case.
Choose an admitted command budget including setup/cleanup and coverage, rather than
raising the ordinary unit silence budget. These package scripts and retained owner
receipts are the qualification handoff; no Test Genie phase binding is invented.


### Explicit host-audio qualification

`pnpm run test:audio-device` selects only
`tests/integration/audio-device-evidence.test.ts` through
`jest.audio-device.config.js`. It requires `VROOLI_DEVICE_EVIDENCE=1` and explicit
host-audio admission because it temporarily changes PipeWire/PulseAudio topology.
Missing opt-in fails before host device work; ordinary unit and local-native configs
do not select this case, even when the variable is inherited. The original unit
file retains only the mocked navigation/retry contract. The tautological opt-in
notice was retired. Source case allocation is recorded in the selection receipt;
this additional native file increases total file count by one.

The host case preserves its three actual device assertions, optional retained WAV
playback and device cleanup. Its output belongs to `coverage/audio-device`, using
the unchanged source denominator/floors; it is not combined coverage. No real device
qualification is implied by an ordinary unit pass or by file discovery.

# Packaging Strategies

**Status:** target design, not yet implemented (2026-10-06)
**Owners:** `packages/delivery-ramp-go` (strategy contract, eligibility, selection),
each delivery ramp (`scenario-to-desktop` first, then `scenario-to-android` and
`scenario-to-ios`) for its strategies, `performance-health` (packaged-launch
measurement, experiments, evidence), `code-facts` (stack facts)
**First target:** desktop on Linux x64. Mobile reuses every contract below.
**Delivery:** `path:.vrooli/work/delivery/desktop-packaging-delivery/GOAL.md`
**Last reviewed:** 2026-10-06

This is the canonical cross-ramp design for **how** a scenario is packaged for a
deployment target. The design is shared; each ramp's PRD and requirements
describe what that ramp must satisfy.

## 1. Why

Users increasingly expect apps to feel native: they start fast, stay small, and
idle quietly. Electron-style wrappers have become a public target ("rebuild
Discord natively"). Vrooli ships desktop apps as Electron plus a bundled Go
runtime, and mobile apps as Capacitor wrappers. We have to decide from
measurements, not opinion, which packaging makes each scenario fast, and we
must not maintain one hand-written app per target.

Evidence at design time:

- **The shell is not our main measured cost.** hello-desktop on Linux Xvfb
  (5 cold + 5 warm runs, `path:scenarios/scenario-to-desktop/docs/internal/performance-baseline.json`)
  shows process-to-splash at about 0.63 s p50. Runtime spawn-to-ready is about
  6.2 s p50, which is the bundled Go runtime waiting for scenario services to
  report ready. Any shell that keeps the sidecar topology keeps that cost.
- **System-webview shells mostly save disk, not memory, on real apps.** On one
  app measured across shells, idle RAM ranged from 98 to 208 MB. Electron sat
  at 128 MB and Tauri at 109 MB, while bundle size ranged from 57 MB (Tauri)
  to 323 MB (Electron). Better Stack, 2026; see §10.
- **Testability differs by engine.** CDP works for Electron, WebView2 and
  Android WebView. WebKitGTK (Linux Wails/Tauri) offers only WebDriver, and
  WKWebView offers only the Safari inspector protocol. BAS currently attaches
  over CDP only.
- **124 of 127 scenarios have a Go API** (`build.kind: go_module`), so a shell
  that embeds Go in-process can remove the sidecar, the port, and readiness
  polling for most of the fleet. Scenarios are free to use other stacks, so
  packaging must be selected from facts, not assumed.

## 2. Concepts

**Packaging strategy.** A named, versioned way to turn a scenario into a
runnable artifact for one ramp. A strategy fixes four axes:

| Axis | Examples |
|---|---|
| Shell | `electron`, `wails`, `tauri`, `capacitor` |
| Runtime topology | `sidecar` (backend in separate processes under the bundled runtime), `in-process` (backend linked into the shell process), `remote-api` (backend elsewhere), `none` (static UI) |
| UI transport | `loopback-http`, `custom-scheme` (shell-registered protocol, no TCP), `file` |
| Optimization profile | a named set of measured changes, e.g. V8 code cache, lazy runtime start, parallel service start |

A strategy ID is `<ramp>/<shell>-<topology>[-<profile>]@<version>`. Example:
`desktop/electron-sidecar@1`, `desktop/wails-inprocess@1`.

**Eligibility.** Each strategy declares predicates over stack facts. A strategy
is eligible for a scenario only when every predicate holds. Examples:

- `wails-inprocess`: `components.api.build.kind == go_module`, and the API
  exposes the embeddable server contract (§5.3).
- `*-sidecar`: every component has a bundleable `build.kind`, and every
  declared resource is bundleable or has a swap.
- `remote-api`: the scenario declares a reachable API deployment (mobile).

**Capabilities.** Each strategy declares which platforms it supports, which
automation adapter can drive its UI (`cdp`, `webdriver`, `webkit-inspector`,
`none`), which profilers it supports, and whether it can auto-update.

**Evidence.** Packaged-launch samples and journey-parity results from
`performance-health` (§6), keyed by scenario, strategy, platform, host
fingerprint and artifact digest.

**Selection.** The ramp chooses the strategy for a build, recording the policy
it applied and why (§4).

## Optional components and component catalog

A packaged scenario's **required dependencies** are part of its runnable
application closure. An **optional component** is installed only when its
consumer chooses it; it is not copied into every download. The component
catalog and its compatibility range are owned by the component/dependency
contract, while this document owns how delivery strategies package and select
compatible artifacts.

A desktop application may use an in-range standalone full application when
available, or its bundled component according to the peer lookup contract. A
slice is not a default full-application provider. Optional component install,
update, rollback, and removal mechanics belong to the scenario-to-desktop
[lifecycle guide](../../scenarios/scenario-to-desktop/docs/guides/AUTO_UPDATES.md);
this page does not duplicate that state machine.

## 3. Ownership

| Concern | Owner | Notes |
|---|---|---|
| Strategy type, registry, eligibility evaluation, selection policy evaluation | `packages/delivery-ramp-go` | One implementation shared by every ramp. Nothing in it names a desktop- or mobile-only concept. |
| Strategy implementation: generator template, builder, `packaged-launch.v1` emission | The ramp (`scenario-to-desktop` now) | One directory per strategy. Shell-agnostic parts (bundle manifest, runtime supervisor, the `packaged-launch.v1` producer) stay shared. `launch-trace.v1` is kept only to read old evidence. |
| Stack facts | `code-facts` | A `stack` fact family: per-component `role`, `build.kind`, `dir`, language, detected UI framework and bundler, declared resources. Today code-facts reads `service.json` but drops `build.kind` (`api/internal/facts/discovery.go:82-169`). |
| Packaged-launch measurement, experiments, comparisons, budgets | `performance-health` | A new packaged axis next to build, startup and UI (§6). |
| Driving the packaged UI for journeys | BAS through workflow-health for CDP engines. For non-CDP engines, an adapter in scenario-to-desktop's target layer (`api/target/`) | CDP exists. The WebDriver adapter is built first in scenario-to-desktop (E4). Changes to BAS or workflow-health are requested through the BAS delivery goal's FEEDBACK.md, never made unilaterally. |
| Pinning a strategy for a deployment | `deployment-manager` profiles, or an explicit build request | An override always wins over auto-selection and is recorded as such. |
| Host toolchains (Rust, WebKitGTK headers, WebKitWebDriver) | `internal/tools/<name>/tool.json` plus `vrooli setup` | Privileged installs happen only through setup. |

## 4. Selection

```
eligible  = strategies(ramp) where predicates(stack facts) hold
                              and platform ∈ capabilities.platforms
                              and adapter satisfies the requested evidence level
candidates = eligible with journey parity = pass and launch reliability ≥ floor
choice     = override                                    if one is set
           | argmin score(evidence for this scenario)    if evidence exists
           | argmin score(evidence for the stack class)  if class evidence exists
           | ramp default                                otherwise
```

- **Score** is a declared, versioned policy (`selection-policy.v1`). Its inputs
  are cold time-to-interactive p50 and p95, steady PSS, idle CPU, artifact size
  and update size, each normalized to the ramp default, with weights in the
  policy file. Speed alone never wins: the parity and reliability floors are
  hard gates.
- **Stack class** is the canonical tuple of stack facts that eligibility
  reads, such as `api=go_module, ui=pnpm_vite, resources=[sqlite]`. Evidence
  measured on one scenario can then inform selection for others with the same
  class.
- **Explainability.** Every build record stores the selection as data: the
  policy version, the facts read, the eligible set with reasons for each
  exclusion, the evidence IDs consulted, and whether an override applied.
  `scenario-to-desktop` exposes this through its CLI (`--json`).
- **Default.** Until evidence promotes another strategy, the default is
  `desktop/electron-sidecar`, which is today's behavior. A strategy becomes the
  default for a stack class only through an experiment report (§6.4) and a
  recorded decision (§9).

## 5. Desktop strategy catalog

All strategies share the bundle manifest, the runtime supervisor contract (for
sidecar topologies), the `packaged-launch.v1` marks, and the journey suite.

### 5.1 `electron-sidecar` (baseline)

This is today's generator (`templates/vanilla/`): Electron, then a Go runtime
spawn, then token, IPC port, `/readyz` polling and port discovery, then
`loadURL(http://127.0.0.1:<ui>)`. It is kept as the reference and the fallback.

### 5.2 `electron-optimized`

Same topology, changed only by measured optimizations:

- Runtime readiness: start services in parallel, replace 350 ms and 500 ms
  polling with push readiness over the control channel, and show the window
  before non-critical services are ready.
- UI over a registered custom scheme (`protocol.handle`, standard and secure)
  instead of a loopback HTTP server. API calls stay on the runtime's
  authenticated transport.
- Electron V8 snapshot and code cache. Build-time code cache is available for
  every Node-hosting process in Electron 42.
- Electron fuses that disable unused capabilities.

Each optimization is a toggle in the profile, so an experiment can attribute
gains to each change. Runtime improvements that help every sidecar strategy go
into the shared runtime, not into this strategy.

### 5.3 `wails-inprocess` (Go APIs only)

Wails v3 (beta at design time) runs the system webview and the scenario's Go
API in one process. The asset server takes an `http.Handler`, so the
scenario's Connect handlers serve UI and API requests over the webview's
custom scheme, with no TCP listener and no readiness polling.

Requirement: the scenario API must be embeddable. Its `main` must be a thin
wrapper over an importable constructor that returns an `http.Handler` and a
lifecycle (start and stop hooks for storage, background workers and
resources). This becomes a template contract in
`templates/scenarios/react-vite` once proven. Scenarios that do not satisfy it
are ineligible, not broken.

Risks: WebKitGTK on Linux (rendering and DMABUF problems with NVIDIA, which is
the reference host's GPU), WKWebView behavior on macOS, and automation that
needs a WebDriver adapter on Linux.

### 5.4 `tauri-sidecar`

Tauri 2 shell (Rust) with the bundled Go runtime as a sidecar (`externalBin`).
It isolates the effect of the shell: same topology as 5.1 and 5.2, with a
different engine and size. Tauri's official WebDriver (`tauri-driver`) covers
Linux and Windows.

### 5.5 Out of scope for now

- Native UI per target (SwiftUI, WinUI, GTK) and cross-platform native UI
  (React Native, Flutter): every scenario UI and the component library are DOM
  based. Revisit when an experiment shows the webview, not the backend or app
  code, is the bottleneck for a stack class we care about.
- CEF-based shells (Tauri CEF runtime, Deno Desktop shared CEF): too immature
  at design time. Each would be added later as a strategy, without any change
  to this contract.

## 6. Measurement: the packaged axis in performance-health

### 6.1 Metric contract (`packaged-launch.v1`)

This supersedes `launch-trace.v1` for new evidence, so that every shell on every
platform emits the same marks. Every mark carries an availability state and a
reason, as `launch-trace.v1` does, because not every platform can observe every
mark (Android `am start -W` has no `shell_ready`, for example). Platform-specific
data goes in `platform_detail`, a `oneof` of per-platform messages.

| Mark or metric | Meaning |
|---|---|
| `process_spawn` | Launcher exec (monotonic zero) |
| `shell_ready` | Shell main process initialized |
| `first_window` | First window shown (splash or main) |
| `backend_ready` | API reachable (sidecar ready or in-process handler mounted); absent for `none` topology |
| `first_contentful_paint` | FCP of the main UI, from the renderer's `PerformanceObserver` |
| `app_interactive` | The app's declared readiness mark: first meaningful data rendered and input accepted. A scenario declares it with `performance.mark('app-interactive')`. Until the template carries it (E6 proposes that through template-manager), the fallback is FCP plus the first completed API response, then 500 ms without a long task, labelled `app_interactive_source=fallback`. |
| `tree_pss_mb` / `tree_uss_mb` | Sum over the whole process tree, attributed by role. RSS double-counts shared pages, so it is reported but never compared. |
| `idle_cpu_pct` | Mean CPU over 60 s after `app_interactive`, with no input |
| `steady_pss_mb` | PSS after the scenario's reference journey plus 60 s |
| `artifact_bytes`, `installed_bytes`, `update_bytes` | Size of the download, of the installed footprint, and of a one-line-change update |

Process roles are a string taxonomy. The generic roles `shell_main`,
`webview_renderer`, `webview_gpu`, `webview_other`, `backend` and `unknown` are
required. Engine-specific roles (for example `electron_utility`) are allowed
alongside them, and each maps to one generic role for comparison across
strategies.

### 6.2 Method

- **Cold:** fresh install and fresh user-data directory, and the artifact's
  files evicted from the page cache with `posix_fadvise(DONTNEED)` or
  `vmtouch -e`. Needs no root, so `drop_caches` is not used.
- **Warm:** an immediate relaunch with the same user-data directory.
- At least 10 runs per cell. Report nearest-rank p50 and p95 plus min, max and
  spread.
- Only samples with the same host fingerprint, artifact digest, display,
  profiler mode and power state are comparable; anything else is marked
  non-comparable.
- Linux reference: Xvfb is used for automation. Because Xvfb hides GPU
  effects, a real-display run is also taken when the host has a session, with
  the display recorded.

### 6.3 Storage

`perf_samples` (or a sibling table) gains `target` (ramp), `strategy`,
`platform`, `experiment_id`, `host_fingerprint`, `artifact_digest`,
`condition`, `display`, `profiler_mode` and `power_state`. Packaged budgets are
declared in the measured scenario's own `.vrooli/testing.json` under
`performance.budgets.packaged.<ramp>`. Packaged captures run outside the gated
test run, like the existing sweep and workload producers, because they build
artifacts.

### 6.4 Experiments

An **experiment** is a declared matrix: scenarios × strategies (with profile
toggles) × platforms × {cold, warm} × N. It is defined as
`packaged-experiment.v1` JSON and kept with the effort that asks the question.

1. A governed program (`performance-health.packaged-experiment`) builds each
   cell through the ramp, with the strategy pinned as an override.
2. It captures the samples and replays the scenario's journey for parity.
3. It writes a comparison report as JSON plus markdown: the median table, the
   attribution of each optimization toggle, and the parity and reliability
   results.

Reports are evidence for selection (§4) and inputs to the decision record (§9).
`analysis.CompareTraces` stays the primitive for comparing two traces.

## 7. Mobile mapping (next effort, same contracts)

| Ramp | Strategies to add | Notes |
|---|---|---|
| Android | `capacitor-remote-api`, `capacitor-inprocess-go` (gomobile `bind` or `c-shared` with 16 KB page alignment; Connect handler served through a Capacitor plugin or custom scheme) | Launch marks from `am start -W` plus the renderer marks; PSS from `dumpsys meminfo`. The CDP adapter exists through device-control. |
| iOS | `capacitor-remote-api`, `capacitor-inprocess-go` (`c-archive` in-process; iOS has no subprocesses) | Needs a real project generator and Mac build routing. Automation uses the WebKit inspector or XCUITest. |

Today no real scenario's API is reachable from a mobile app (the UI resolves
`localhost`). Fixing that is a precondition of every mobile strategy, and the
mobile effort owns it.

## 8. Testability matrix

| Engine | Platforms | Adapter | State |
|---|---|---|---|
| Chromium (Electron) | Linux, Windows, macOS | CDP | Working |
| WebView2 | Windows | CDP (`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`) | Untested |
| WebKitGTK | Linux | W3C WebDriver (`WebKitWebDriver`) | Missing |
| WKWebView | macOS, iOS | Safari inspector protocol or XCUITest | Missing |
| Android WebView | Android | CDP over adb socket forward | Working through device-control |

A strategy without an adapter for a platform can still ship there, but its
selection evidence lacks journey parity, so it is never auto-selected on that
platform.

## 9. Decisions

| Date | Decision | Why | Switch cost |
|---|---|---|---|
| 2026-10-06 | Keep one web UI source and vary packaging, not the UI, per target | One codebase per scenario at fleet scale; the measured cost is in the runtime, not the renderer | Native per-target UI would mean rewriting each scenario per target |
| 2026-10-06 | Selection by eligibility from stack facts, then evidence, then a declared default | Scenarios may adopt other stacks; Go-only optimizations must not break them | Changing the policy is a versioned policy file; past builds keep their recorded policy |
| 2026-10-06 | Desktop first, mobile on the same contracts | Desktop and cloud are the only ramps used in practice; mobile ramps need basic repairs first | Mobile adds strategies and adapters, not new contracts |
| 2026-10-06 | Every working strategy stays selectable; experiments choose defaults | The operator wants options alongside, not one winner | Removing a strategy is a separate decision recorded here |

## 10. References

- Shell benchmarks: https://betterstack.com/community/guides/scaling-nodejs/tauri-vs-electron-vs-deno-vs-electrobun/
- Electron 42 code cache and snapshots: https://electronjs.org/blog/electron-42-0 · custom protocols: https://electronjs.org/docs/latest/api/protocol
- Wails v3: https://v3alpha.wails.io/ · Tauri sidecars and WebDriver: https://v2.tauri.app/develop/tests/webdriver/
- WebKitGTK and NVIDIA: https://v2.tauri.app/develop/debug/linux-graphics/
- Go on mobile: https://go.dev/wiki/Mobile
- Desktop ramp: `path:scenarios/scenario-to-desktop/docs/concepts/ARCHITECTURE.md`,
  `path:scenarios/scenario-to-desktop/docs/reference/live-desktop-api.md`
- Measurement owner: `path:scenarios/performance-health/PRD.md`

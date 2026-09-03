# Known Issues and Problems

## Open Issues

### P0 Blockers
- Host-level recovery grant application still requires an operator-authenticated `sudo vrooli setup`; the existing autoheal sudoers entry is stale, mode `0644`, and does not match the managed wildcard-free grant.

### P1 Issues
- The full 24-hour data-retention soak remains outstanding.
- Test Genie storage/workflow evidence is unavailable while `storage-manager` fails its own API build at `internal/validation/hygiene_common.go:19`.

### P2 Minor Issues
_None at initialization_

## Deferred Ideas

### Not In Scope (Intentionally Excluded)
- **Full metrics collection**: system-monitor handles detailed metrics; autoheal focuses on health/recovery
- **Outbound alert delivery (email, webhook, chat)**: findings become autoheal incidents and the typed readiness projection (`system-boot-recovery-readiness`, `system-emergency-watchdog-report`); delivery stays with notification-hub, whose absence `coverage-delivery-reach` reports honestly
- **Non-Vrooli service management**: Only manages Vrooli resources, scenarios, and OS watchdog

### Future Consideration
- **AI-powered root cause analysis** (OT-P2-005): Use Ollama to analyze failure patterns
- **Custom check plugins** (OT-P2-004): Allow external health check definitions
- **Certificate monitoring** (OT-P2-001): Check SSL cert expiration
- **Display manager health** (OT-P2-002): GDM/lightdm/sddm monitoring

## Postmortems

Each entry names the trigger, the gap that let it go undetected, the fix, and
the test that now guards it.

### 2026-08-01: The loop blocked forever on a pipe its child inherited

- **Trigger**: on boot the loop ran `vrooli scenario start vrooli-autoheal` through `exec.Cmd.CombinedOutput`. The start spawned the long-lived runtime supervisor, which inherited the output pipe as its stderr and held it open. `CombinedOutput` reads until every write end closes, so `Wait` never returned. The loop never reached its first tick and started nothing. Observed directly: the loop parked in `futex_do_wait` holding the read end of `pipe:[623670]` while the supervisor held the write end as fd 2.
- **Detection gap**: the loop logged "starting scenario" and then nothing; no heartbeat existed, so "silent" and "healthy" looked the same from outside.
- **Fix**: every `vrooli` subprocess now runs through `repo-contract-go/cliinvoke.Run`, which sets a deadline and `WaitDelay` so an inherited pipe delays a result by seconds instead of ending the supervisor's life. The loop writes a heartbeat to its status file on every tick.
- **Guard**: `TestRunReturnsWhenDescendantHoldsThePipe` in `packages/repo-contract-go/cliinvoke`.

### 2026-09-01: Dependency drift took the loop's recovery path down with it

- **Trigger**: a shared package change reached the loop's module graph through api-core, so the binary that exists to recover from build drift could not itself be rebuilt.
- **Detection gap**: nothing checked what the loop depended on; the failure surfaced only when a rebuild was needed.
- **Fix**: the loop's module depends on repo-contract-go, envkit-go and langrecover only; the invoker it shares with the control plane lives inside repo-contract-go so no proto or api-core edge can reach it.
- **Guard**: `go mod graph` in `cli/loop` shows no `packages/proto` or `api-core` entry (asserted by the loop's module tests) and the recovery floor's tests in `langrecover`.

### 2026-09-02: The runtime supervisor exited 1 silently after every restart

- **Trigger**: every CLI reinstall or unit stop left a `running` supervisor session row with a dead PID and an unexpired 45-second lease. The successor skipped it (PID dead), the claim refused it (any running row), the process exited 1, and systemd retried every 5 seconds until the lease lapsed: 1,070 restarts in 12 hours, none explained in the journal because the unit sends stdout and stderr to `runtime-supervisor.log`.
- **Detection gap**: `NRestarts` was read by nothing, and the lease-honouring reaper was the only cleanup path.
- **Fix**: the successor retires any running row whose PID is dead before it claims; under the native unit it takes over a live peer instead of exiting; the start failure is logged in the supervisor's own log format. The `runtime_supervisor` safeguard records `NRestarts` and `Result` as evidence on every setup and readiness inspection.
- **Guard**: `TestServiceEnsureStartedRetiresDeadPredecessorWithLiveLease` (`internal/runtimesupervisor`), `TestInspectReportsCrashLoopingUnit` (`internal/safeguards/runtime-supervisor`).

### 2026-09-02: A retired CLI flag turned every supervisor into a no-op at boot

- **Trigger**: the `--no-stale-check` global was removed from the CLI at 03:06 and the CLI reinstalled at 03:07. The loop binary (built 23:58), the on-disk runtime-supervisor unit (rendered 2026-08-18), the autoheal API's command runner, the bridge bootstrap and CI still passed the flag. After the 04:50 freeze and 07:35 reboot every `vrooli` call exited 1 with `Unknown command: --no-stale-check`; the loop retried identically every three minutes and the supervisor crash-looped 495 times. The emergency watchdog saw the dead unit 20 seconds after boot and printed it to a journal nobody read.
- **Detection gap**: the CLI's argv surface was an untested contract; no test parsed any supervisor's argv against the real parser, no check evaluated boot readiness while the host was healthy, and the loop could not tell "healing" from "cannot heal".
- **Fix**: retired globals are tolerated with a warning (`rootcli.retiredGlobals`); every argv producer is registered in `internal/cli/rootcli/invokers` and parsed through `rootcli.ResolveArgv`; supervisors pass no global flags; native units render from one `platformgo.ServiceDefinition`; the loop classifies usage errors as non-healable and exits 3 so the scheduler escalates; `system-boot-recovery-readiness` evaluates every boot precondition hourly (not yet built: plan phase 10); `make install` refuses to install a CLI while a supervisory module does not build.
- **Guard**: `TestParseArgsRetiredGlobalIsToleratedWithWarning` and `TestResolveArgvAcceptsEveryRegisteredCommand` (`internal/cli/rootcli`), `TestEveryInvokerResolvesThroughTheRootParser` and `TestEveryDirectSpawnIsRegistered` (`internal/cli/rootcli/invokers`), `TestRunClassifiesUsageError` (`cliinvoke`), `TestUsageErrorIsNonHealableAndExits3` (`cli/loop`).

### 2026-09-02: Three agent sessions built the repository at once and took the host to load 1,499

- **Trigger**: at about 10:14 three coding-agent sessions ran wide builds in one checkout: a root-module `go build ./...`, a four-target cross-compile and a module test sweep. Go starts one compile or link per core, so 32 cores and three sessions became roughly a hundred linkers; memory filled, 70 GB of swap absorbed the overflow, linkers were OOM-killed, and the host was unresponsive for about twenty minutes. The autoheal loop kept its 60-second ticks and restarted its own API at 10:34.
- **Detection gap**: builds had no width cap (the lifecycle budgets admission by memory only; three Scenario Dependency Analyzer sites erased `GOFLAGS`); agent sessions ran in `user-1000.slice` with `MemoryMax=infinity` and `ManagedOOMPreference=omit`, so systemd-oomd protected the desktop and never touched the storm; the emergency watchdog's fork-rate finding went to stdout only, its sustain window reset itself on every not-yet-sustained tick (so it could never fire), three components read the setpoint separately, incidents could not name a parent, and no CLI could say which sessions were in the tree.
- **Fix**: `envkit.Toolchain` and the `BuildWidth` lever put a floor under every toolchain spawn, `mk/toolchain.mk` under every Makefile, and the `no-raw-toolchain-spawn` rule keeps it true; `agent_session_containment` converges `vrooli-agents.slice` and every session is born in a scope under it; `internal/setpoint` is the one bar reader with the authored 10-minute sustain; the watchdog writes `last-report.json` with attribution and `system-emergency-watchdog-report` opens one incident per finding titled by the parent; `contain-storm` freezes an attributed agent scope through the recovery gate and `vrooli agent thaw` reverses it; editor leases make every session visible in `vrooli agent list`.
- **Guard**: `TestToolchainAppendsWidthToInheritedGoflags` (`packages/envkit-go`), `TestLifecycleStepEnvCarriesToolchainFloor` (`internal/lifecycle`), `TestScenarioMakefilesIncludeToolchainMk` (`internal/cli/rootcli/invokers`), `TestContainedCommandLandsInScopeOnLinux` and `TestFreezeAndThawScope` (`packages/platform-go`), `TestSetpointRejectsDuplicateCellRef` (`internal/setpoint`), `TestWatchdogUsesAuthoredSustain` (`cmd/vrooli-watchdog`), `TestHostPressureCheckUsesAuthoredSustain` and `TestIncidentFingerprintSeparatesPressureFindings` (autoheal api), `TestContainStormRefusesPidOutsideAgentSlice` (autoheal api), `TestEditorLeaseExpiresOnlyOnProofOfDeath` (`internal/scenarioruntime`).

## Architecture Decisions

### ADR-001: CLI-based resource/scenario management
**Decision**: Use `vrooli resource/scenario` CLI commands instead of direct API calls
**Rationale**: CLI already handles authentication, port discovery, and error handling; reduces code duplication
**Trade-offs**: Slightly higher latency than direct calls; dependency on vrooli CLI being installed

### ADR-002: In-memory state with SQLite persistence
**Decision**: Keep current health state in memory and persist structured history to SQLite.
**Rationale**: Fast local status queries, durable history across restarts, no separate database resource required.
**Trade-offs**: SQLite write volume and retention must be bounded for small hosts; snapshots are fingerprint-deduplicated.

### ADR-003: OS watchdog keeps autoheal running
**Decision**: Install OS-level service (systemd/launchd/Windows) that monitors autoheal itself
**Rationale**: Ensures autoheal survives even its own crashes; brings system back after reboots
**Trade-offs**: Requires elevated privileges for installation on some platforms

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| WSL platform detection edge cases | Medium | Low | Test explicitly on WSL1 and WSL2 |
| Watchdog installation fails without sudo | Medium | Medium | Graceful degradation with warning |
| Circular dependency (autoheal monitors scenarios that monitor autoheal) | Low | Medium | Exclude autoheal from monitored scenario list |
| Too-aggressive restarts cause thrashing | Low | High | Add restart cooldown and max retry limits |

## UX Issues

### 2026-05-15: App-shell overlap and mobile overflow regression fix

#### Fixed in this pass
- Replaced the dense inline header/tab implementation with an explicit app-shell layout contract so header, navigation, and main content have stable ownership and selector-backed landmarks.
- Added mobile layout regression coverage for shell/header/content ordering, summary-grid responsiveness, selector-backed tabs, and browser-level bounding-box validation at 390px, 360px, and desktop widths.
- Removed the visible horizontal overflow seen in the mobile dashboard by tightening root overflow containment, dashboard summary-card sizing, health-group containers, and variable-length check-card callouts.
- Migrated high-traffic tick notices and check-card importance callouts to the shared `Notice` composite, reducing one-off callout styling in the dashboard path.
- Moved additional shared status visuals (`StatusIcon`, `StatusSparkline`, `ActionHistory`, action result icons) onto semantic theme tokens instead of raw palette classes.

#### Remaining UX debt
- The tab rail is touch-scrollable on very narrow screens; a future pass could add a compact overflow affordance or segmented menu if users miss offscreen tabs.
- `SystemProtection.tsx` and `CheckDetailModal.tsx` remain high-density components and should still be split by responsibility in a follow-up architecture pass.
- The comprehensive scenario runner still depends on healthy external orchestration (`test-genie` + runtime URL discovery); this pass records its environment failure separately from the UI gates.

### 2026-02-18: Header clarity + mobile health-check list pass

#### Fixed in this pass
- Removed the redundant global status badge from the app header so status emphasis stays in dashboard/trends where detailed context already exists.
- Removed the ambiguous header `Auto/Manual` toggle and moved refresh-mode control into `Settings > General` with explicit explanatory text.
- Added a dedicated `Dashboard Refresh` settings section with a labeled switch and mode state (`Enabled` / `Manual mode`) for clearer behavior.
- Reworked health-check group rendering for narrow viewports into a bordered, divided list pattern on mobile while preserving card presentation on `sm+` breakpoints.
- Updated app-level refresh UX test coverage to assert the header toggle removal and retained settings access.

#### Remaining UX debt
- Dashboard group headers still use fairly dense metadata rows; a follow-up could tighten icon/text hierarchy further for very small screens.
- The full comprehensive scenario test suite is long-running; this pass validated targeted UI tests and TypeScript type-checking only.

### 2026-02-18: Shared primitive/mobile containment cleanup pass

#### Fixed in this pass
- Updated shared modal primitive layout to improve small-screen behavior: bottom-aligned presentation on mobile, safe-area aware overlay padding, and tighter adaptive modal corner/height constraints.
- Refactored `CodePreview` to token-first styling and improved narrow-width interaction ergonomics (copy action spacing, label truncation, and border/surface consistency).
- Consolidated duplicate relative-time formatting in check details by reusing `ui/src/lib/utils.ts` for coherence and reduced drift risk.
- Improved docs reader containment for constrained viewports: tighter mobile padding, adaptive sidebar/document max heights using dynamic viewport units, and safe-area bottom spacing.
- Added explicit `aria-label` values to compact app-shell icon controls for clearer accessibility affordance on mobile.

#### Remaining UX debt
- Uptime chart tests still emit SVG-tag warnings in jsdom mocks (non-blocking but noisy for CI signal).
- Docs markdown still uses a custom regex-based parser; future hardening should move to a stricter parser/sanitizer path for long-term rendering reliability.

### 2026-02-18: Dashboard and modal responsiveness/coherence pass

#### Fixed in this pass
- Consolidated repeated relative-time formatting into shared utility (`ui/src/lib/utils.ts`) and reused in dashboard/trends surfaces.
- Refined app shell header for mobile: wrapped controls, reduced title density, scroll-safe tab row, and selector-backed tab test IDs.
- Reworked dashboard health-group headers and cards for clearer scan hierarchy and mobile-safe metadata wrapping.
- Updated events timeline and action buttons to use theme token styling instead of legacy slate/white color classes.
- Improved settings and check-detail modal layouts for constrained viewports: scroll-safe tabs, wrapped action rows, compact spacing, and responsive stats grids.
- Made docs and trends pages adaptive on narrow viewports (stacked layouts, overflow-safe controls, edge-safe padding).

### 2026-02-18: UX polish and coherence follow-up (mobile + dashboard cleanup)

#### Fixed in this pass
- Consolidated repeated dashboard health-group rendering (`critical`, `warning`, `ok`) into a single mapped section component for lower duplication and easier maintenance.
- Improved app-shell usability on small screens: action row now wraps cleanly, auto-refresh mode label remains visible on mobile, and tab test IDs are fully selector-registry backed.
- Improved docs navigation on mobile with progressive disclosure (`Browse documents` toggle), searchable scroll-constrained nav, and automatic nav collapse after document selection.
- Fixed content containment for variable-length text across dashboard and trends lists (check IDs, messages, sub-check details) to prevent clipping/truncation overflow.
- Updated system-protection install/uninstall action groups to stack safely on narrow viewports and expose a labeled uninstall action.
- Tuned docs markdown viewport behavior to keep article scrolling usable on mobile (`max-h` adaptive by breakpoint).

#### Remaining UX debt
- Uptime chart tests still emit SVG-tag warnings in jsdom mocks (non-blocking but noisy for CI signal).
- Some components outside this pass still retain legacy color classes and should be migrated to token-first styling for full coherence.

## 2026-02-17: UI lint cycle detection dependency gap

### Problem
React stability pass completed with runtime status normalization and defensive rendering guards. Remaining gap: eslint import cycle rule (import/no-cycle) is not enabled because eslint-plugin-import and eslint-import-resolver-typescript are not present in scenario UI dependencies, and dependency installation was not performed in this pass.

---
## Work ladder

- Rung: W3
- Evidence: W0 passes because goal `supervision-authority-vrooli-autoheal` is represented by P0 targets `OT-P0-011` through `OT-P0-015`; W1 `business-health validate scenario vrooli-autoheal` and W2 `vrooli scenario requirements validate vrooli-autoheal` pass. W3 governed Test Genie run `20260829-012848-8b736c9c` completed with 22 phases passed, 0 failed, and the optional templates phase skipped.
- Blocker: none at the work-ladder gates; implementation-plan work may proceed and must re-measure after behavior changes.
- Measured: 2026-08-29

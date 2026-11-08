# Known Issues and Problems

## Open Issues

### P0 Blockers
- Resolved in the 2026-11-05 scoped W3 repair: runtime service recovery uses the
  setup-installed privilege broker through `vrooli host service`, replacing the
  private sudoers-file check and direct sudo execution. The operator upgraded the
  root-owned broker through project setup; its new host capabilities and a live
  rsyslog recovery were verified. Focused elevation, infrastructure-check and
  healing tests pass. Previously installed legacy grant files are not consumed
  and are retained for explicit host-policy cleanup.

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

### 2026-09-10: Core supervision had no durable coding-agent escalation window

- **Trigger**: a core scenario could remain unhealthy after mechanical restart attempts without a durable, restart-safe handoff to an investigator; the configured grace period also was not enforced before escalation.
- **Detection gap**: recovery admission was process-local and used the lifecycle recovery broker, so a restart could lose the one-shot state and the recovery action was not an explicit coding-agent fallback chain.
- **Fix**: autoheal now records continuous unhealthy time, waits the configured 10-minute grace period after the restart-attempt threshold, and submits to a durable control-plane coding-agent broker. The broker enforces at most one launch per scenario per rolling hour and invokes the governed `vrooli agent launch` surface, which prefers Agent Manager and falls back to native runners. All computed supervision-closure scenario members are protected from permanent suspension, and the recovery-plane checks run before optional checks in each tick. The duplicate asynchronous startup tick was removed so the external loop remains the sole tick owner.
- **Guard**: `TestScenarioAgentRecoveryWaitsForContinuousGracePeriod`, `TestComputedSupervisionMembersNeverBecomePermanentlySuspended`, and `TestRunAllPrioritizesSupervisedChecks` (autoheal API), the durable broker tests in `internal/recovery/coding_agent_test.go`, and `TestHealthResponseReportsLifecycleBuildIdentity` (system-monitor API). Live validation confirmed a 10-minute grace, 5-minute mechanical cooldown, 43 computed core scenarios, healthy `storage-manager` and `system-monitor`, and zero suspended scenario checks after the core-set update.

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

## Work ladder — remote recovery outage, 2026-11-05

- Rung: W3, observed recovery implementation failure. No full contract certification is claimed.
- Evidence: the user journal records systemd-oomd killing Web Console API at
  08:45:33 EST and Agent Manager API at 09:37:02 EST. Autoheal API's current
  instance last had a listener on November 3 at 20:00:24 UTC; the watchdog
  repeatedly failed recovery with `lifecycle provider ... is not authoritative`.
  A surviving UI kept the instance active while the API claim eventually expired.
- Repair: control-plane lifecycle now distinguishes proven API absence from an
  unreachable live provider, using current-boot component process references and
  retained claim ports. Dead API plus surviving UI can recover; unknown evidence
  and live non-UI processes retain fencing. Expired claims are never rediscovered.
- Validation: focused provider absence/fence/emergency restart tests pass,
  including expired claims and negative liveness/listener/inventory cases.
  Full lifecycle package suite passed in 178.554 seconds; the final active-claim
  filtering refinement also passed focused regressions. Shared-tree execution
  is observational evidence, not an isolated immutable-source certification.
  Final `make install` passed all supervisory build/vet gates. Autoheal recovered
  without override in operation `startop-f62c25bd4488877e4588f0d5b7de7130`
  at 18:56:46 UTC, exercising the retained expired claim in production.
  Web Console recovered in `startop-02a58ec309219b9554dcf76bf023502d`
  at 18:56:12 UTC; direct API health checks returned healthy for both Web
  Console and Agent Manager. Automatic unattended fault recovery is still unproven.
  Agent Manager recovered through normal restart operation
  `startop-cf80bce80b246c6b1aefa582257fcefd` at 18:53:52 UTC. Health alone does
  not verify recovery of each detached executor.
- Remaining scope: verify live Web Console/autoheal recovery; pressure containment
  and critical-service protection; stale running process reporting; automatic
  restart/escalation/cooldown behavior; manual spawn caller identity; remote
  tunnel route and authenticated API reachability; safe cloudflared and safeguard
  updates; adversarial owner-by-owner passes and sustained recovery qualification.
- Separate validation limitation: protobuf verifier reports 23 differences
  involving retired template-validation-react-vite output and descriptor bytes;
  governed `vrooli package generate proto` reports no changed owners and does
  not remove those files. No generated output was hand-edited. This remains
  distinct from the passing control-plane compile/test results.

- Recovery follow-up observation at 18:57 UTC: autoheal status answers, but its
  check timestamps remain November 3 and `tickRunning=false`. The live watchdog
  retains a 15-minute healing backoff. Verify fresh owner ticks and automatic
  recovery before claiming supervision is restored; status endpoint health is
  insufficient. Historical boot-readiness results identify stale loop/watchdog
  binaries, which need current owner inspection before remediation.

## Work ladder — watchdog observation and result freshness, 2026-11-05

- Rung: W3. Restored API served November 3 check results while the live watchdog
  slept through a 15-minute lifecycle backoff. Its process hash differed from
  the current built loop. API freshness fields existed but the UI ignored them.
- Repair: heal backoff now performs identity-verified read-only detection between
  one-minute waits, bounded by a ten-second observation context; external recovery
  resumes ticks without spending a lifecycle retry. UI shows stale/unknown result
  freshness and keeps historical results labelled while a cycle runs. Browser
  title also reflects missing freshness rather than a green historical verdict.
- Evidence: complete loop race suite passes, including externally restored API,
  stranger refusal, cancellation, and unchanged cumulative lifecycle backoff.
  App regression suite and TypeScript check pass. The available CUA inventory
  contains no browsers; no rendered visual verification is claimed.
- Serving adoption: normal `vrooli setup --resources none --scenarios none
  --sudo-mode skip --onboarding none` rebuilt and restarted the loop at
  18:59:58 UTC and updated the emergency watchdog. Live journal confirms tick
  completion at 19:01:29 UTC and transition to healthy. API/UI replacement
  operation `startop-3b2a804b14006b36bd6ef82d14c7395b` succeeded. Final
  observation-budget refinement was adopted through setup; successor PID 811568
  completed its first tick at 19:02:55 UTC.
- Test Genie unit run `20261105-185956-c4b09039` could not execute because
  Scenario Dependency Analyzer API was down. It recovered via lifecycle operation
  `startop-3cd2b0a15b2e641aea9fc8227037ee8e`; retry
  `20261105-190111-a0d28832` passed the unit phase. Final loop race checks
  also passed after the ten-second observation-budget refinement.
- Limits: setup reports unrelated missing orcarouter credential/configuration;
  no full setup-completion claim. Public Web Console and Agent Manager GETs
  returned HTTP 302, which does not verify authenticated remote functionality.
  Controlled recovery faults, memory-pressure containment, manual caller identity,
  escalation, and sustained qualification remain open.

### Pressure protection follow-up — 2026-11-05

SDA was restored at 19:01:03 UTC, then its service scope was killed again by
systemd-oomd at 19:02:33 UTC (user-manager journal). It was restored again via
`startop-2d46f2a639a1375e4e93e483c4ec40c5` for validation. The current
agent-session-containment documentation describes services as the workloads being
saved because their slice lacks `ManagedOOMMemoryPressure=kill`; that does not
establish protection from a kill decision on the ancestor user slice. Observed
service scopes have `ManagedOOMPreference=none`. Inspect and repair the canonical
control-plane containment policy, including critical recovery services and the
source of host pressure; do not disable oomd or blanket-protect runaway workloads.

### Recovery-plane OOM selection policy — 2026-11-05

W3 repair at the control-plane owner: platform-go carries a typed
`ManagedOOMAvoid` property through native unit rendering and transient scope
adoption. Lifecycle selects it only for live instances of the canonical four
`coreset.RequiredOperationalScenarios`; shadow, agent, resource and ordinary
scenario workloads retain their previous selection policy. Native autoheal,
runtime supervisor and emergency watchdog definitions use the same property.
The cgroup-only fallback cannot express this manager-owned policy and refuses
to claim it, leaving an explicit best-effort placement warning.

Evidence: platform-go full package suite passed; focused lifecycle selection
and scope tests passed; affected safeguard handler tests passed. Control-plane
`make install` passed all supervisory build/vet gates. Setup rendered and adopted
all three native supervisor preferences. Live `systemctl --user show` reports
`ManagedOOMPreference=avoid` for autoheal, runtime supervisor and emergency
watchdog services, and for both Agent Manager scopes after normal replacement
`startop-48d924bef65613f3f1b198820d57f9d1`. All 42 original native agent
scopes remained running. Autoheal/Web Console normal replacements completed healthy through
`startop-2e955b5faea40280ba9a25ce8cdf9331` and
`startop-94416ca5a6649467332af85d45983513`. Their API and UI scopes
report avoid; the Web Console presentation scopes and an ordinary dependency
scope report none, confirming the variant boundary. Prompt Manager scope adoption
and direct pressure-event qualification are not established by this evidence.

This is deliberately a last-resort preference, not immunity: under the observed
user-manager ancestor, cgroup owners match and systemd respects the attribute.
A root-owned monitored ancestor can ignore a user-owned preference. Resource
budgets, causes of pressure, monitoring of scope drift, reliable restoration and
escalation after last-resort kills still require qualification. No pressure
storm was induced on the live host.

## Work ladder — recovery coordination versus drain proof, 2026-11-05

- Rung: W3. Live autoheal action history contained `scenario-agent-manager`
  skips with owner admission HTTP503. Its full maintenance inventory exceeds
  the physical-reference protocol bound, although ordinary restart preserves
  detached work and does not require a self-drain proof.
- Repair: Agent Manager serves a distinct read-only `view=fence` projection of
  durable closed/revision state. It declares inventory unobserved and omits all
  drain-proof fields. Autoheal's existing coordinator reads that projection.
  Closed/unknown state still suppresses automatic restart; full maintenance
  inventory errors remain errors for operations that require that proof.
- Adversarial finding: coding-agent escalation was evaluated before trigger
  eligibility and before the runtime pressure/startup/maintenance gate. It now
  honors both, preventing escalation from bypassing an intentional recovery
  hold or acting on an explicitly ineligible observation/soft warning.
- Evidence: complete Agent Manager maintenance race suite passed (11.713s).
  Focused autoheal coordinator/agent escalation race tests passed; tests cover
  open/closed fences, unavailable durable state, absent drain fields, overflowing
  inventory, eligible recovery, gated recovery and ineligible observations.
  Scoped Test Genie `20261105-192454-ecaf8a00` is attached once through wait
  session 59077. Serving adoption is being observed by normal lifecycle waits.
- New unresolved validation evidence: a full checks-package race run also
  exposed fixture clock access after a timed-out action and concurrent writes
  in the mock tracker store. Production tracker persistence uses unordered
  asynchronous snapshots, so audit durable ordering rather than merely adding
  locks to the fixtures. Full race cleanliness is not claimed.
- Remaining: dead-provider coordination, full maintenance capacity, broker
  cross-process deduplication/observable outcomes, actual unattended repair,
  fault qualification and authenticated remote access remain unproven.

Serving evidence: Agent Manager normal replacement
`startop-7dd50d83f0eb4a3c0de8350a872eefb2` completed; all 42 native agent
scopes recorded before replacement remained running. Live fence GET returned
HTTP200 with closed=false/revision=116/inventoryObserved=false while the full
inventory GET retained HTTP503. Autoheal replacement
`startop-4e8cf5d41d06666245c49498c91eb650` completed healthy at 19:26:43 UTC.
The first lifecycle wait returned the previous operation while pre-stop staging
was still active; the original command handle completed later and the final
registry operation established adoption. Lifecycle wait ownership during staging
requires a control-plane repair, not another replacement on observation timeout.

Unit `20261105-192454-ecaf8a00` failed on the newly added escalation fixture:
its run had compiled before the test registered its healable check. Canonical
findings `artifact_f44484efdaa82536e9d5ff6fda2c0376` name only those fixture
cases. The corrected fixture and current focused race cases pass; a scoped unit
retry `20261105-192820-29ee8b9a` was started on the corrected source and
attached through one blocking wait. This is not a product-failure waiver.

## Work ladder — ordered tracker persistence, 2026-11-05

- Rung: W3. Scenario-health observations and action outcomes launched independent
  snapshot-save goroutines; reset launched an independent delete. Old writes
  could arrive after new recovery decisions or resurrect a removed tracker.
- Repair: registry tracker writers now serialize state mutation through the
  bounded persistence attempt. Resume, reset, restore, store replacement and
  disposition reconciliation share the ordering boundary. The ordinary registry
  lock is released during saves/deletes so status stays readable. Failures are
  diagnosed, not claimed durable. SQL schema and historical rows are unchanged.
- Evidence: full checks race package passed in 3.016s, including the previously
  racing timeout/recovery cases. Test clock updates now share a lock with reads
  by late action completions. New regressions block a save, prove mutation does
  not return early while status remains readable, verify final concurrent state,
  and reset/reload without resurrection. Focused persistence-store race tests
  passed in 1.244s. Source-level store tests do not prove crash-atomic side effects.
- Serving: normal autoheal restart `startop-0c26a6184864fd059e2f50e57226b3b6`
  completed healthy at 19:32:23 UTC. The previous corrected coordination unit
  receipt `20261105-192820-29ee8b9a` passed unit in 100s (51 warnings, zero errors).
  Tracker unit receipt `20261105-193213-7b50d022` passed unit in 100s
  (51 advisory warnings, zero errors). No duplicate run was started on observation delay.
- Limits: a failed durable write still leaves in-memory state; error propagation
  and any fail-closed expensive-action policy need separate qualification.
  Dead-provider recovery coordination, broker process deduplication, root pressure,
  maintenance capacity, authenticated remote operation and controlled fault
  qualification remain open under the original goal.

Controlled watchdog fault drill began after the tracker unit pass. Baseline:
Agent Manager healthy/readiness=true, Web Console healthy, autoheal loop healthy
with successful tick at 19:34:00 UTC, native watchdog PID 1001174 active.
Autoheal API PID before fault was 1152147. The authorized reversible drill stops
only the autoheal scenario through the control plane and observes the independent
watchdog through one journal-follow attachment for `state heal -> verify`.
Do not manually restart while that observer and native loop are live; completion
requires replacement API identity and a fresh completed tick, not merely a live
unit or an old start-operation verdict.

Watchdog drill intermediate evidence: control-plane stop completed at 19:34:52
UTC. Native watchdog PID 1001174 recorded failed ticks at 19:35 and 19:36,
then after the third failure started lifecycle operation
`startop-7f80a59afa184699d53ba53aa29794fe` at 19:37:02 UTC. That operation
completed at 19:37:11 with API PID 1184739 replacing 1152147. Native journal
recorded `state heal -> verify`. No manual start was issued. Fresh tick began
at 19:37:11.784 UTC; completion remains under one native journal attachment.

Watchdog drill passed: journal recorded `state verify -> healthy` at 19:38:13
UTC; API status returned statusFresh=true and
lastCompletedTickAt=2026-11-05T19:38:13.331193809Z. Agent Manager remained
healthy/readiness=true and Web Console remained healthy. Autoheal API restoration
completed 139 seconds after the stop; fresh full-cycle evidence completed 201
seconds after the stop. This qualifies recovery from a stopped autoheal scenario
by its independent native loop. It does not qualify a simultaneous host crash,
a hung API, an absent tunnel, failed builds, coding-agent repair, or boot recovery.
Two critical findings remained in the fresh summary; full-goal completion is not
claimed.

### 2026-11-05: Cross-process escalation admission and uncertain agent outcomes

Given overlapping autoheal API processes, each broker previously checked a
cached ledger, admitted its own investigation and wrote a shared fixed temp
file. The update path also hid persistence errors. The CLI listed only the
mechanical recovery ledger, leaving investigation outcomes inaccessible.

Admission and outcome writes now reload the ledger under the platform file
lock with a bounded wait. Atomic publication uses the existing storage helper.
An outcome cannot advance in memory when its write fails, and a failed
pre-launch receipt write stops execution. `vrooli agent recover list --json`
reads both ledgers and reports read errors. Concurrent broker and separate
process regressions cover duplicate admission and preservation of updates.

The worker now falls back only on a typed pre-launch unavailability result.
An ambiguous error or timeout leaves `unconfirmed`; pending work blocks new
admission beyond the hourly window. The launch CLI preserves the caller's
context and adds `--timeout`. Cancellable launches retain their waiting wrapper
instead of replacing the process and losing the deadline. Autoheal gives the
inner timeout ten seconds to cancel and detach before the outer deadline.

Installed runner help confirmed that Codex requires `exec` and Grok requires
`-p` for headless prompts. Their old positional prompt invoked an interactive
interface. The typed wrapper now selects those headless forms.

Remaining boundary: native session identity is not yet linked to a recovery
receipt. An autoheal crash can leave a pending investigation whose process
state cannot be automatically reconciled from this ledger. It stays blocked
instead of assuming death and launching duplicates. A successful runner exit
is not proof of repaired scenario health. No real coding-agent repair or
crash-during-investigation drill has qualified this path yet.

Validation and adoption: full root recovery and CLI race suites passed, and the
full cli-core launcher race suite passed. Test Genie unit run
`20261105-194826-6b612a4b` failed on elevation fixture compile errors during a
concurrent host-broker refactor. The downloaded native findings artifact
confirmed the missing helper references. After those shared fixtures were
updated, `20261105-195459-fc8b17ba` passed in 55 seconds. `make install` passed
the supervisory build/vet preflight. Normal autoheal restart
`startop-c4ef85b9c5ee0b0f52e17090761c7cee` completed healthy at 19:59:22 UTC.
Fresh status subsequently reported lastCompletedTickAt 20:00:43 UTC.

The live JSON ledger contained 89 investigations: 53 failed, 23 succeeded and
13 still running, with the oldest pending receipt from September 10. Pending
legacy receipts have no process identity. They require reconciliation; neither
their age nor restarting autoheal proves that their descendants stopped.

A further native cancellation regression exposed an adapter gap below the CLI:
the session containment implementation accepted a context but ignored it while
waiting. Cancelling during scope acquisition also left a descendant when only
the known root could be stopped. Work at the shared containment boundary now
uses bounded scope discovery during cleanup, requires the exact spawned root
to belong to the resolved scope, and verifies empty membership after stopping.
This cancellation repair is a separate adoption step from the 19:59 API restart.

### 2026-11-05: Runtime supervisor readiness accepted a superseded live image

Given an atomic CLI replacement and an unchanged native unit, the runtime
supervisor safeguard reported `already_present` while PID 1002036 still ran
`vrooli (deleted)`. File-identity comparison confirmed the live image differed
from the installed CLI. A correct unit path and active service did not prove
that recovery-code fixes had reached the daemon.

Linux safeguard inspection now reads MainPID and compares the live executable
inode through procfs with the installed binary. A mismatch, an unreadable
image or missing active PID leaves readiness pending. Apply requests a restart
for image adoption even when unit content is current, then re-inspects. The
focused race suite passed, including failed-adoption and uncertain-identity
regressions. `make install` passed the full supervisory build/vet gate.

The supported `vrooli host safeguard runtime_supervisor --sudo-mode=skip`
refresh adopted PID 1540515 at 20:17:42 UTC. Native manager state was active,
NRestarts=0, and a direct live/installed file-identity check returned true.
The emergency watchdog was also refreshed after platform changes; subsequent
inspection reported its binary and timer current. These are live adoption
proofs, not a boot or host-crash recovery qualification.

### 2026-11-05: Managed-service restart stopped the working service before acquisition

Cloudflared update inspection found that private and user-hosted restart paths
stopped the current process before the start path acquired or verified the
declared replacement. Failed acquisition could therefore turn a healthy
service into an outage. Native helper regressions reproduced that outcome for
an installed-file checksum mismatch and an acquired candidate checksum mismatch.

Both restart paths now prepare the verified artifact before stopping the
service. The private-provider regression also verifies preservation of the
serving companion's PID. Focused lifecycle race tests passed in 2.735 seconds.
The full resource race suite failed in four blueprint-archive success fixtures
that leave their resource enabled, and in the unrelated pii-detector platform
claim check. Those unchanged paths were reported to scenario-qa as
`knw-1793910705143724911` and `knw-1793910713600388740`; full-suite qualification
is not claimed.

This repairs acquisition-failure ordering. It does not yet implement rollback
after a candidate starts incorrectly, update the cloudflared artifact, or
qualify remote continuity during a client upgrade.

### 2026-11-05: Verified cloudflared staging and maintenance drain transport

The cloudflared host requirement and SDA gateway now use the resource-owned
checksum-pinned release, rather than declaring any PATH executable current.
The declared release is staged through SDA while status reports declared,
staged and running artifacts separately. The actual staged Linux AMD64 bytes
matched the official asset digest and the original managed connector retained
its PID and ownership identity throughout acquisition. This is verified staging;
rolling promotion, retained launch rollback and interrupted update recovery
remain in the active Plan Manager workstream `remote-access-safe-managed-updates`.

A subsequent SDA refresh exposed an independent maintenance transport failure:
its ordinary HTTP write deadline severed the Drain receipt after admission had
closed, leaving application requests at 503. The same retained provider later
proved drained, and an ordinary restart recovered it without a force override.
A real-socket regression reproduced EOF at the generic deadline. The shared
lifecycle transport now clears that deadline only for the admitted loopback
Drain RPC and preserves request cancellation. SDA also keeps its graph sweep in
the tracked worker group until exit; it previously detached that loop before
reporting worker shutdown. Focused race tests cover both contracts. Live adoption
and a clean dependency-phase receipt are tracked separately from code checks.

The scoped SDA unit receipt retained failures in trusted-base closure
(`web-console` requires `ai-gateway` outside its declared base), CLI module
freshness fixtures and UI coverage policy. Those are recorded in scenario-qa;
the recovery dependency closure remains an outstanding broader outage concern.
The first dependencies phase overlapped the failed refresh and returned EOF;
it does not establish a dependency implementation failure.

### 2026-11-06: Native coding-agent execution ownership (rollout in progress)

- Rung: W3. Durable admission was followed by an API-owned goroutine. An API
  crash could strand the receipt, and the launch CLI placed the runner in a
  sibling scope outside a proposed wrapper's native deadline.
- Repair in the control-plane owner: retained governed worker image, once-only
  dispatch and native claim, actual boot/image/invocation proof, a lease bound
  before child execution, bounded operation limits and native reconciliation.
  Admission receipts retain their historical format; versioned sidecars carry
  native evidence. Autoheal calls this owner and runs its bounded observer.
  The new record/sidecar envelopes and recovery codecs are included in the
  root's reviewed persistence inventory; future changes cannot evade that gate.
- The actual frozen-root adapter probe exposed an independent storage failure:
  api-core's batched class-root resolution ignored compiled runtime-home pins
  and required a checkout. The repo-contract owner now provides one pinned or
  source-authoritative batch; api-core uses it. Source-free storage and actual
  native adapter regressions pass.
- Worker SIGKILL exposed another shared defect: a populated `ExitType=cgroup`
  unit could retain children after its main died, while stop tried to signal
  only that missing main. The platform owner now excludes the exact tree after
  main loss, including loss during the graceful signal. Red/green regressions
  and an actual native worker-loss drill pass. The first failed fixture's
  remaining unit was verified quiescent and retired; no fixture units remain.
- Evidence: active Plan Manager artifacts under
  `remote-access-safe-managed-updates/native-coding-agent-execution-milestone.json`.
  Native caller-loss, actual root adapter, worker-loss/child exclusion and
  accelerated private lease-expiry proofs passed. No real model or scenario API
  was called by those fixtures. Scoped Test Genie run
  `20261106-145818-65ce7769` passed unit (50s) and dependencies (4s); dependency
  metadata repair used SDA's governed install/reconcile owners.
- Remaining: production adoption of the full shared fix, a real bounded repair,
  natural recovery-loop observation, interrupted admission/dispatch and further
  adversarial review. The serving-root updater refuses the full candidate's
  changed conservative persistence-source digest. Its existing receipt and
  installed image were proved unchanged. Qualify that evolution through its
  owner; do not replace the digest with a claimed equal value or silently bypass
  the gate. No production readiness or whole-goal completion is claimed here.

The next adversarial pass reproduced a shim-only false availability result:
`exec.LookPath` found a managed alias with no real OpenCode executable, and the
old adapter stopped as `unconfirmed` before trying Codex. The fixed worker uses
the shared shim-excluding resolver before attribution, lease creation or child
execution; only that pre-launch failure permits fallback. It also reuses the
ordinary CLI's managed-tool PATH recovery after native claim. A private native
fixture covers shim-only OpenCode followed by a real fixture Codex executable.
Admission now rejects scenario traversal and oversized/invalid UTF-8 requester
values before they can poison the strict ledger.

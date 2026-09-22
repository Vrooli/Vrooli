# Browser Automation Studio: refactor assessment

Initial assessment: 2026-09-21. Follow-ups: 2026-09-22 UTC (2026-09-21 local). Status: seven investigation passes complete for the scope below; proposed design, not an implementation grant.

## Implementation preparation — 2026-09-22 UTC

The operator subsequently asked to complete the remaining launch preparation.
The canonical implementation target is now [REFRACTOR_CONTRACT.json](REFRACTOR_CONTRACT.json),
with [TESTING.md](TESTING.md) defining independent oracles, producer obligations,
comparability, replacement completion and adversarial review. The operator
subsequently rejected the prepared plan; it was archived before execution.
[REFRACTOR_GOAL.md](REFRACTOR_GOAL.md) now defines a continuous file-based goal.
Do not use Plan Manager for this work. REFRACTOR_PROGRESS.md owns checkpoints
and PROBLEMS.md owns defects. This supersedes the historical recommendations
below. No implementation agent, harness goal or queued Swarm work was launched.

The operator's subsequent clarification makes all unavailable validation
non-blocking, including later-discovered platform, environment or tool gaps.
The contract and protocol supersede historical completion gates below. Retain
unverified dispositions and continue useful authorized repairs. Green checks,
an empty known issue list and clean reviews trigger fresh investigation rather
than completion. Continue until operator stop/redirection or runtime interruption.

[Preparation evidence](REFRACTOR_PREPARATION_2026-09-22.json) retains the fresh
owner observations. Ten isolated producer groups exercised 121 expected behaviors:
33 met their expectation, 88 did not, and none was unavailable. The fail-closed
runner is `python3 scenarios/browser-automation-studio/docs/internal/refactor_regressions.py`.
Its synthetic seams do not establish live browser or native-platform outcomes.

The tidiness run `20260922-022553-efaa6ee7` returned 1,104 findings, including 381
high-complexity, 12 high-coupling, 82 long-file, 338 duplicate-code and 290
boilerplate findings. Duplication line debt is 35,603 against recorded baseline
35,303; the existing ratchet fails. These are owner findings, not 1,104 unique
product defects. Programs run `20260922-023953-811244a0` passed. Product gaps
remain open; the rehabilitation board truthfully exposes 17 unmet required rows.

Prior-work reconciliation: the standing `browser-automation-studio-improve` goal
covers operational diagnostics; its pass-rate milestone is not this product's
acceptance. Professionalization still has active product-surface and pending
final-closure phases. Capture truthfulness has active/pending hardening phases.
The completed architecture and desktop-evidence plans supply behavior to preserve,
not current certification. None was silently completed, superseded or archived.

The initial viewer remains bounded CDP/JPEG within the current deployment; a
measured rendering experiment is an implementation choice, not a pre-launch
rewrite decision. The release matrix requires Linux/Windows x64 and macOS x64/arm64,
plus preserved Android/external Electron attachment. Live owner discovery found
local Linux ready, an online Intel Mac without authorized test-dispatch scope,
and no Windows/ARM Mac runner in that inventory. An Android device is registered
and reachable but its capability readiness is not qualification evidence. These
are exact access/evidence gaps, not proof that other machines do not exist.

## Decision brief

BAS needs a substantial rehabilitation. The strongest evidence concerns recording correctness and durability, session/resource ownership, misleading or incomplete measurements, and preservation of platform capabilities. Reorganizing large files alone will not resolve these problems.

Keep the typed workflow compiler/executor, driver adapter, session leases, evidence policies, routed storage, and desktop/mobile attach contracts. Refactor their responsibilities incrementally around a browser-first product contract. Establish a small, independent behavior corpus before replacing any critical path. Treat native desktop rendering as a measured design choice, not a prerequisite for replacing the whole application.

The investigation found functioning capture and substantial prior engineering. It does not support a claim that every BAS operation is broken, that Chromium is the primary memory consumer, or that any particular replacement transport will solve the observed problems.

Start with [the issue register](../PROBLEMS.md#refactor-investigation-register--2026-09-21), [the machine-readable baseline](REFRACTOR_BASELINE_2026-09-21.json), and the [proposed architecture](../concepts/ARCHITECTURE.md#proposed-browser-first-target--2026-09-21). Those three artifacts have different roles: mutable issue state, dated observations, and proposed design.

The [first follow-up](#follow-up-isolated-reproductions--2026-09-22-utc) adds a reproducible execution-queue leak, incorrect network evidence, lost/mismerged recorded input, and live API resource pressure much larger than RSS alone suggested. The [profile/replay pass](#profile-and-replay-investigation--2026-09-22-utc) adds interrupted/concurrent profile-save failures, misleading recovery, lost browser shortcuts and semantically incorrect generated workflows. The [session/frame/retention pass](#session-frame-and-retention-investigation--2026-09-22-utc) adds unsafe retry/reset behavior, stale frame work, a broken polling fallback, stalled retention and successful-looking failed cleanup. The [execution pass](#execution-retry-and-cancellation-investigation--2026-09-22-utc) adds instruction admission/ownership gaps, retry and loop identity collisions, and cancellation/failure evidence loss. The [reuse pass](#reuse-profile-and-evidence-investigation--2026-09-22-utc) adds profile/configuration mismatch, stale evidence ownership and bypassed external-target validation. The [stream/live-fixture pass](#stream-lifecycle-and-live-fixture-investigation--2026-09-22-utc) adds producer cleanup races, ineffective controls, a broken validation flag and lossy outcome projection. The register now contains 49 open issues. These findings make lifecycle and data correctness the first repair boundary; they do not establish a need to replace the rendering framework.

## Authority, scope, and evidence limits

The operator requested investigation, durable current/target state, preservation of core behavior, and a proposal to review before launching implementation. This work did not create a goal, plan, development grant, or agent. It did not change product code, dependencies, runtime configuration, workflows belonging to other scenarios, or existing driver edits. One governed capture of BAS's own homepage and one fresh adhoc about:blank counter fixture created normal execution evidence. The shared CLI defect found in pass seven was filed through the report-bug skill.

Initial HEAD: 129da6b96062fad08f53b34228ae16801befdf79. Five driver files were already modified: routes/session-audio.ts, routes/session-start.ts, server.ts, session/manager.ts, and types/session.ts. The tree has many unrelated changes. Counts describe observed working files, not an isolated commit. The follow-up identifies an older dirty runtime revision; exact equivalence with those source files remains unproved.

Evidence labels in this report:

| Label | Meaning |
| --- | --- |
| Observed | A command or artifact read in this investigation returned the stated result. |
| Isolated reproduction | Actual source modules executed with synthetic I/O, with the stated behavior observed. This is not a live browser or full-workflow result. |
| Code finding | The stated behavior is visible in the inspected source. Its live frequency is not established. |
| Historical | A dated owner result or prior report. It is not current certification. |
| Hypothesis | A causal explanation that needs the named experiment. |
| Proposed | A target or design awaiting acceptance; it is not shipped behavior. |
| Unknown | Evidence was unavailable or the behavior was not exercised. |

Recall returned earlier refactor reviews, the May UI performance audit, the June capture characterization, and BAS usage/improvement programs. Some recall providers reported degradation. Swarm goal discovery failed because swarm-manager was stopped. No conclusion about the existence of applicable prior goals follows from that outage. Read those goals before approving an implementation mandate.

### Work ladder disposition: W0 gap

The supplied operator instruction is the governing investigation brief. It asks for the ability to “use it like a browser,” with passive capture, persistent sign-ins, responsive streaming, cross-OS desktop delivery, and preservation of mobile/desktop testing.

The active PRD opening, dated 2026-07-26, describes “permanent visual browser-workflow capability” and P0 targets for typed execution, trustworthy evidence, maintainable architecture, and agent reuse. It does not explicitly protect browser-first interaction, passive recording completeness, sign-in durability, latency/resource limits, or a desktop/platform acceptance matrix. Its commercial target is P2. The operator did not assign replacement release priorities in this conversation.

Disposition: record the missing outcomes and propose the contract extension; do not silently promote commercialization to P0 or claim W1–W3 readiness. No new downstream readiness gates or full suites were run. Source inspection, historical evidence reads, operational measurements, and one capture characterize the implementation independently of a readiness verdict. PRD and requirement amendments belong to the later approved work.

## Starting measurements

### Source size, measured on the working tree

[refactor_inventory.py](refactor_inventory.py) regenerates this inventory using only Python's standard library and Git. These are physical lines, including comments/blanks. They are size indicators, not cyclomatic complexity.

| Surface | Runtime files | Runtime lines | Files >500 lines | Files >1,000 lines |
| --- | ---: | ---: | ---: | ---: |
| Go API | 422 | 94,165 | 44 | 10 |
| Go CLI | 43 | 3,179 | 0 | 0 |
| Node driver | 210 | 55,567 | 25 | 4 |
| React UI | 685 | 131,532 | 56 | 11 |
| Total | 1,360 | 284,443 | 125 | 25 |

The broader tracked source inventory includes tests and tooling: API 157,506 lines; CLI 5,382; driver 90,527; UI 150,708. Generated proto, distributions, vendor files, and dependency trees are excluded. The ignored/untracked generated Electron application is outside these totals. Driver recording diagnostics under recording/testing are excluded from the main-path runtime metric, even though some are runtime-accessible.

Important hotspots:

| File, relative to scenario | Lines | Review reason |
| --- | ---: | --- |
| api/automation/executor/simple_executor.go | 1,899 | Execution orchestration, retries, evidence, and target context meet here. |
| playwright-driver/src/recording/capture/browser-scripts/recording-script.js | 1,626 | Page injection, queueing, selectors, events, and input semantics. |
| ui/src/domains/recording/RecordingSession.tsx | 1,560 | Browser lifecycle, interaction modes, AI, timeline, persistence, layout. |
| api/automation/execution-writer/file_writer.go | 1,510 | Artifact lifetime and I/O cost. |
| api/automation/compiler/compiler.go | 1,337 | Typed compilation and validation boundary. |
| playwright-driver/src/recording/orchestration/pipeline-manager.ts | 1,328 | Recording lifecycle and recovery. |
| api/handlers/record_mode.go | 1,274 | Session/profile/recording API coordination. |
| playwright-driver/src/session/manager.ts | 1,256 | Admission, reuse, context initialization, external attach, cleanup. |
| api/main.go | 1,242 | Composition and domain registration. |

A function or file is not defective merely because it is long. Prioritize hotspots that cross ownership boundaries and have demonstrated behavioral risk. Extractions should remove duplicate policy and reduce dependencies, not just move lines into helpers.

### Historical complexity and test evidence

Test Genie run 20260909-201227-1e8ed2e9, 2026-09-09, reports 15 passed phases, 12 failed, and 2 skipped. Its mutable-tree provenance limits exact-version attribution.

| Signal | Historical observation | Interpretation |
| --- | --- | --- |
| Tidiness | 1,104 findings: 381 high complexity, 12 high coupling, 82 long files, 338 duplicated-code and 290 duplicated-boilerplate findings, plus one budget failure | A substantial review queue; findings are not all distinct defects. |
| Duplication debt | 35,603 lines vs recorded 35,303; delta +300 | The configured ratchet had regressed. Not today's freshly measured duplication total. |
| Unit coverage | API 33.7%, CLI 25.6%, UI 28.4% vs configured 75/75/85% | Historical reported coverage, with API test misconfiguration and UI command failure. Not reliable current complete-suite coverage. |
| Workflow phase | 901 seconds; timed out waiting on provider run 05a85460-ce90-4a4a-b4a6-bbea1d8aa2ec | Observer deadline is not proof that every workflow case failed. |
| Performance phase | Passed; five capability assessments clean | Does not prove live browser input latency, recording fidelity, or eight-hour resource stability. |
| Security | 779 scanner findings, including repeated static/dependency findings | Triage reachability, installed/shipped versions, and duplicates before counting vulnerabilities. |
| Latest listed run | 20260910-065713-c12e0a04 failed with 27 provider_unavailable phases, two skipped | Provider outage evidence; not 27 newly failing product behaviors. |

Several September 9 runs have overall passed with an advisory programs phase failed. Read phase verdicts and advisory flags. Do not interpret overall status as universal success.

The unit policy declares API, CLI, and UI roles, but does not list the Node driver as a required role (.vrooli/testing.json). The driver has its own Jest suites. Confirm actual runner coverage and admit the driver explicitly before using a suite as protection for its behavior.

Historical claims in PROBLEMS.md that structural refactoring was “DEFINITIVELY AND CONCLUSIVELY COMPLETE,” that no file exceeded 1,000 lines, and that deployment was ready cannot describe the current tree. Preserve them as history; do not use them as current constraints or proof. May's performance work did achieve a documented viewport feedback-loop reduction: RecordModePage commits fell from 8,702 to 145 in that specific flow. Preserve the equality guards and narrow rendering subscriptions; do not re-report the pre-fix churn as current.

### Live reads, 2026-09-21

BAS lifecycle status reported healthy/running, started 03:14:06 UTC. The API supervised the Node sidecar. A driver read returned zero sessions and capacity 10. Other callers continued operating during this investigation; subsequent observations are not a controlled idle interval.

| Read | Observation | Limit |
| --- | --- | --- |
| measures pass-rate --window last_7d | 0.8758 | Mixed execution population; no sample count in response; not a golden-corpus pass rate. |
| measures p95-duration --window last_7d | 32,226 ms | Mixed workloads, including failures; exceeds existing 5,000 ms board band but does not identify the cause. |
| selector-failure-rate / step-failure-rate | Both printed 0.0000; JSON responses were {} | Zero-valued proto fields are omitted; the implementation also maps empty denominators to zero. Cannot infer a populated, failure-free sample. |
| Governed setpoint read | prog_477aa08b-5a1b-490d-99fa-2572b2acaa5b exhausted its 60-second budget; no board returned | Retain as unavailable. The individual hand reads above are explicitly separate observations. |
| Process RSS | API 2,552.04 MiB; summed API/UI/driver/Chromium descendants 3,121.11 MiB | RSS double-counts shared pages. Not PSS, peak memory, or proof of a leak. Operator browser client excluded. |
| Host | Linux x86_64, 32 logical CPUs, load average about 12.64/10.86/9.21 | Shared busy workstation; no isolated performance comparison. |

The inventory snapshot preserves exact process values and metric samples. At the metrics read, lifetime driver counters included 234 screenshot-telemetry failures and 65 DOM-telemetry failures. Successful screenshot instructions numbered 8,373; failed screenshot instructions 261. These counters have a different population from the SQL execution measure. They demonstrate observed error accounting, not user task success rates. Streaming histograms had no samples and active recordings were zero; they cannot establish interactive latency.

The exported session-duration histogram is unsuitable for lifetime analysis: session-teardown.ts starts its timer at teardown entry and observes the duration when teardown finishes. CDP frame capture latency measures base64 decode time; its “e2e” measure ends at the synchronous send call, before client decoding and paint.

### Fresh capture

The governed browser-automation-studio.capture-surface program captured BAS's homepage once:

- Program: prog_eef245c0-a676-4b1e-9fb3-4ca20d921737.
- Execution: 1454e614-3f08-439f-ba76-0dc2e26ef00a.
- Request: 1280×720, scale factor 1, domcontentloaded, 500 ms explicit settle.
- Result: ok; owner duration 2,830 ms; navigation 402 ms; readiness wait 881 ms.
- Screenshot and DOM-tree references were returned. Four execution step screenshots were listed; the final PNG was fetched and visually inspected. It rendered the BAS dashboard.
- All four screenshots were 2560×1440. Their listed bytes totaled 1,462,120. The driver context defaults to scale factor 2; the capture handler path does not reference DeviceScaleFactor. This is a reproduced request/result discrepancy with a plausible source explanation.
- One capture proves this path worked at that time. It does not prove input responsiveness, passive recording, video fidelity, or general website compatibility.

The homepage shows workflow/dashboard entry points and a “Start recording” action. It is not yet evidence of opening directly into a restored everyday browsing workspace. “Unknown Workflow” appeared for an active run while the capture itself was active; attribution was not established, so this is a UI observation rather than a confirmed independent defect.

## Current architecture and preservation boundary

~~~mermaid
flowchart LR
    UI[React workspace and canvas viewer] --> API[Go API and Connect services]
    CLI[CLI and governed programs] --> API
    TG[Test Genie and workflow owners] --> API
    API --> COMP[Typed compiler and executor]
    COMP --> DRIVER[Node driver and session leases]
    DRIVER --> BROWSER[Managed Chromium contexts]
    DRIVER --> TARGET[Electron or Android WebView CDP target]
    BROWSER --> REC[Injected recorder and event delivery]
    REC --> LIVE[Live capture and recording services]
    LIVE --> DB[Routed SQLite and timeline]
    DRIVER --> FRAMES[CDP screencast or polling]
    FRAMES --> HUB[Go WebSocket hub]
    HUB --> UI
    FRAMES --> UI
    COMP --> EVIDENCE[Execution writer and artifact storage]
    EVIDENCE --> EXPORT[Replay package and export consumers]
~~~

There are multiple frame routes: direct driver WebSocket, Go relay, and polling fallback. The direct route is still described in code as a latency research spike. It remains conditional on the relay WebSocket being ready in the CDP send path. This couples a purportedly direct consumer to the availability of another transport.

Recording currently spans injected page JavaScript, driver orchestration and buffers, callbacks/Go handlers, live-capture conversion, a separate persistent recording service, profile persistence, and several UI hooks/stores. Its full field-value semantics and durability are not consistently enforced across those boundaries.

### Capability matrix

“Present” below means source exists; only the fresh capture path was exercised here.

| Capability to preserve | Present owner/path | Acceptance still needed |
| --- | --- | --- |
| Browser workspace, navigation, tabs, history | RecordingSession; useTabs/usePages/useBrowserNavigation; browser chrome | Restore usable workspace; popups, back/forward, shortcuts, focus, clipboard, IME, zoom, downloads/uploads. |
| Passive actions and selected-range workflow creation | recording-script.js; pipeline-manager; services/recording; live-capture/workflow_generator.go | Event completeness, empty input, replacement/deletion, same selector across tabs/frames, route transitions, crash recovery. |
| Persisted sign-ins and profile configuration | services/session-profile; record_mode_persistence.go; context-builder.ts | Cookie/localStorage/IndexedDB fixture matrix, expiry, concurrent profile ownership, restart recovery. |
| Typed workflows, loops, assertions, retry/resume | api/automation/{compiler,executor,engine,state}; workflow service | Exact revisions, pre/postconditions, cancellation, repeat-effect safety, side-effect-aware retry policy. |
| Agent navigation and reusable workflows | driver ai/vision-agent; ai-gateway seam; .vrooli/program-runtime | Independent outcome assertions, budget/cancellation, same task/profile cohort, provenance, no accidental duplicate execution. |
| AI interpretation of observed human activity | Deterministic merge/smart-wait generator is present | AI semantic distillation from passive history was not found in this path. Treat as proposed extension, distinct from AI action execution or text-to-workflow generation. |
| Screenshots, logs, network, trace, accessibility/performance | telemetry collectors; capture producer registry; execution writer | Correlated artifact completeness and failure evidence; optional vs required artifact outcomes. |
| Video and replay/demo export | Session video finalization; services/export; replay packages | Context video, live preview, device screen recording, and rendered demo are separate products. Single-page Capture VIDEO producer remains unavailable. |
| Mobile responsive testing | Context viewport/touch/isMobile settings | Actual mobile media/input behavior at explicit presets; viewport width alone must not define device identity. |
| Real Android WebView | AndroidWebViewEngine; electron-target.ts supports android-webview | Device/forward/recording lifecycle remains scenario-to-android/device-control; join step offsets to device video. |
| Desktop app testing | Validated app target, renderer identity/origin, isolation lease | Preserve attach/detach ownership; never kill external target on session close. |
| Bundled desktop product | bundle/bundle.json; API sidecar supervisor; scenario-to-desktop generated app | Clean-machine install and launch, packaged Node/Chromium/assets, writable paths, update/rollback, OS-specific cleanup. |
| Monetization | Entitlement/credits integration and documented launch constraints | Commercial release is not established by this review; preserve existing enforcement and distinguish local desktop from hosted tenancy. |

App-target capability validation explicitly rejects requested HAR, video, tracing, performance trace, and accessibility for external targets. Android's engine advertises corresponding limitations and delegates screen recording. Preserve explicit capability refusal and external evidence joins. Do not silently replace device or desktop tests with web viewport tests. No iOS implementation or certification was established.

## High-impact findings and experiments

The mutable statuses and closure conditions live only in PROBLEMS.md under BAS-RF identifiers.

| ID | Evidence and implication | Next falsifiable check |
| --- | --- | --- |
| BAS-RF-001 | W0 product gap described above; old completion claims conflict with current facts. | Compare accepted browser-first target against amended PRD and requirements before readiness gating. |
| BAS-RF-002 | api/services/recording/service.go:232–242 bounds cache and logs database save failure, then broadcasts and returns success. | Inject a failing repository; a durable acknowledgement must fail or identify pending durability. Restart must retain every acknowledged event. |
| BAS-RF-003 | record_mode_persistence.go:80–105 logs SaveStorageState/SaveOpenTabs errors and returns nil; handler can answer persisted. | Fail each save independently; response must expose failure/partial state. Test prior profile remains readable. |
| BAS-RF-004 | recording-script.js:1017 captures target.value; :1026 drops empty values. workflow_generator.go:109–133 concatenates snapshots. selectorsMatch ignores page/frame identity. Existing unit fixture assumes text fragments. | Type “a”, pause, then “ab”; clear field; repeat selector in another tab. Generated/replayed fields must equal final observed values in each context. |
| BAS-RF-005 | session/manager.ts:318 checks sessions.size; awaits browser/context work at :379/:394; insertion at :552. Guard is per execution ID. | Delay creation with maxConcurrent=1 and send two different execution IDs. Only one may consume capacity; failed/cancelled admission must release reservation. |
| BAS-RF-006 | browser-pool.ts stores each launch key until closeAll; browser-manager.ts:304 keys audio strategy and WAV path. | Open/close sequential sessions with distinct audio fixtures. Pool/process count must plateau at policy bound without disrupting live sessions. |
| BAS-RF-007 | cdp-screencast.ts:267 and websocket/server.ts:202 send without a buffered-byte limit; CDP direct broadcast sits behind relay readiness. | Throttle one subscriber and disconnect relay. Queue bytes stay bounded; current authorized frame reaches healthy direct subscriber; input ordering holds. |
| BAS-RF-008 | websocket/server.ts:119 accepts a missing session ID; :196 filters only when both IDs exist. No authentication/origin validation in that class. It binds loopback by default. | Use isolated clients, not real browsing: unauthenticated, missing-session, and wrong-session subscriptions must be rejected. Do not claim an Internet exploit from this local code finding. |
| BAS-RF-009 | measures.go:51–75 aggregates mixed executions, uses floor-based p95 rank, defaults missing duration to zero; :93–97 maps zero denominator to zero. Node duration/frame metrics measure different intervals than labels imply. | Use empty/small known datasets and monotonic timing fixtures; return counts/cohort/window/unknown, and verify percentile definition and input-to-paint timestamps. |
| BAS-RF-010 | Live process RSS high; no heap attribution established. | Profile API heap/native allocations and PSS before/after a fixed cohort and idle interval. Distinguish Go retained pages, SQLite/native memory, caches, and true growth. |
| BAS-RF-011 | manager.ts:946 calls storageState() without IndexedDB inclusion. UI flush relies on unload/unmount; save errors can be concealed. | Restart fixture sessions using cookies, localStorage, and IndexedDB; test renderer/API/driver interruption and concurrent profile use. |
| BAS-RF-012 | Bundled manifest names darwin-x64/linux-x64/win-x64. It does not establish ARM support or clean-machine evidence. Supervisor Stop uses SIGTERM with kill fallback; Node defaults to PATH. | Inspect final artifact inventory and run supported OS/arch receipts, including child process cleanup, missing developer tools, update and rollback. |
| BAS-RF-013 | Single-page producer registry keeps VIDEO and DOM unavailable; performance, accessibility, DOM_TREE are implemented, contrary to older problem text. External targets reject several capabilities. | Request each artifact kind on each runtime; verify exact capability/result matrix and fail required evidence explicitly. |
| BAS-RF-014 | Historical unit failures/low coverage; required driver role absent; setpoint read timed out. | Repair/confirm runner admission and sensors, then retain focused receipts and adverse-path corpus results. Preserve unavailable states. |
| BAS-RF-015 | Size and historical complexity debt concentrated at ownership boundaries; UI and Go independently merge actions. | Extract one responsibility; demonstrate one policy owner, stable contracts, reduced dependency/debt readings, and unchanged behavior corpus. |
| BAS-RF-016 | Fresh scale-factor-1 capture produced 2× dimensions and four step PNGs; driver defaults DPR to 2. | Repeat on fixed fixture at DPR 1/2; output dimensions honor request, artifacts follow selected capture policy, and bytes/CPU are attributed. |
| BAS-RF-017 | Raw recorder reads all INPUT/TEXTAREA values without a password-field distinction in handleInput. Separate credential-use policy does not prove passive recorder redaction. | Use only synthetic sentinel credentials; inspect page event, transport, timeline, exports, screenshots, and AI attachments for disclosure. Redact/reference secrets before persistence and inference. |
| BAS-RF-018 | Provider comments promise bot-detection success; recorder modifies page content and uses main-world globals/intercepted fetch. No current site-compatibility corpus was read. | Compare approved site/fixture cohorts by provider/browser version and authenticated profile; record challenge incidence and human recovery. Preserve real browser behavior without claiming universal invisibility. |

Additional hypotheses, not established root causes:

1. UI/per-frame state or decoding may dominate interaction delay. Measure UI event → driver dispatch → affected rendered frame; compare with recording and evidence disabled/enabled. The previous viewport-loop fix is evidence against assuming the old root cause persists.
2. Full evidence capture can dominate short tasks. The fresh one-page operation emitted four screenshots; compare a declared final-only capture policy with diagnostic recording under identical fixtures. Preserve failure evidence.
3. At the first pass, API memory could not be attributed from RSS. The follow-up confirms large allocated Go heap and identifies current-source retention mechanisms; their production byte attribution remains unproved. Do not optimize Chromium first merely because it is a browser product.
4. Text snapshots, DOM mutation, service-worker interception, navigation, and event delivery may explain recording failures independently of streaming. Use a fidelity corpus and instrument each boundary before changing transport.

## Follow-up: isolated reproductions — 2026-09-22 UTC

The operator requested continued investigation. This pass kept product files and
dependencies unchanged and started no service or browser. It added
[Node probes](refactor_probes.cjs), [Go probes](refactor_probes.go), and a
[dated evidence artifact](REFRACTOR_PROBES_2026-09-22.json). The original baseline
remains unchanged. The tracked product-source digest still matches the first
pass: `2df361d5baaaf501715527e57993b6024ca63e798713f5d9e655a9682f584f43`.

There are 21 observations: 19 mismatches against desired behavior and two
successful controls. These are deliberately selected adverse cases, **not** a
failure rate, 19 independent bugs, or a Test Genie verdict. Several observations
belong to one root issue. Commands exited successfully because the probes
completed; `expected_behavior_met` carries each behavioral result.

### API memory: allocated heap, swap, and source identity

Read-only `/health` and `/proc/1647308/{status,smaps_rollup}` observations:

| Time UTC | Go HeapAlloc, MiB | Goroutines | API PSS, KiB | API swap, KiB | Health |
| --- | ---: | ---: | ---: | ---: | --- |
| 00:05:22 | 10,727.81 | 10,301 | 4,380,054 | 9,874,848 | healthy |
| 00:11:47 | 12,128.59 | 10,309 | 3,181,003 | 10,227,812 | healthy |

This narrows BAS-RF-010: substantial allocated Go heap is observed, so the
earlier RSS cannot be explained solely by Chromium or native allocation. RSS
fell between these samples while allocated heap and swap rose. A smaller RSS
alone would therefore be a misleading success criterion. HeapAlloc includes
objects not yet collected; these are not post-GC retained-heap measurements.
Workload and collection timing were uncontrolled, so no growth-rate/soak claim
is justified. Health reports dependency availability; it did not flag this
resource state. Promote memory attribution to proposed priority A.

`go version -m /proc/1647308/exe` reports revision
`92ad79a9dbcbb556afbfabdf7cb86fd3c82553c6`, `vcs.modified=true`, Linux/amd64,
whereas the investigation HEAD is `129da6b96062fad08f53b34228ae16801befdf79`.
The runtime build identity is preserved in the evidence JSON. An older dirty
revision does not prove exactly which current files differ. Do not equate
current-source probes with production stack attribution. No production heap
dump, debugger attachment, GC trigger, or restart was performed.

### New findings and repair boundaries

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-019 | **Isolated reproduction.** `NetworkCollector.getRequestId` in `playwright-driver/src/telemetry/collector.ts:245` returns `String(request)`. Two objects using the installed rebrowser 1.52.0 Request prototype both stringify as `[object Object]`. Overlapping A/B requests with response statuses 201/202 yield one event: URL B with status 201. The non-overlapping control correctly yields two records. | Correlate by actual request identity, independent of URL/method/stringification. Test overlaps, redirects, failures and evictions through the public collector API. Evidence capture must not silently misattribute a response. |
| BAS-RF-020 | **Code + isolated handler reproduction.** `api/websocket/hub.go:659` spawns a goroutine per input; the forwarder sends independent HTTP calls. `playwright-driver/src/routes/record-mode/recording-input.ts` has no per-session serialization. With the first `mouse.move` delayed, a later up request applies before down: `[up, down]`. The fixture uses actual handler code with synthetic parsing/page I/O. | A session input owner orders accepted commands and acknowledges an applied sequence. Coalesce replaceable motion only; preserve buttons, keys and modifier transitions. Test cancellation, stale leases and errors without leaving keys/buttons held. Real network incidence remains unmeasured. |
| BAS-RF-021 | **Code + isolated composition reproduction.** `handlers/handler.go:233` wraps `WSHubSink` in the UX collector; `services/workflow/executions.go:759` closes only a concrete `*WSHubSink`. Completing 24 wrapped sinks leaves 24 queue goroutines and zero hub-close calls. Explicitly closing the underlying sinks releases all 24. Separately, `automation/events/ws_sink.go:248` clears pending events on close: a blocked first event plus an accepted terminal event delivers only the first. | Propagate lifecycle through the sink interface/decorators and all exit paths. Define finish/drain versus abort explicitly. A simple type-assertion repair must not introduce terminal-event loss. Controlled workflow soak and production stack/heap attribution are still required. |
| BAS-RF-022 | **Isolated full-script reproduction.** `recording-script.js:281` treats any resolved fetch, including HTTP 500, as success and removes its pending event. `:140` removes by millisecond timestamp: acknowledging the first of two same-timestamp events removes both. | Stable unique event IDs and session/page epoch, explicit durable acknowledgement, idempotent replay, and gap reporting. Test server rejection, ambiguous delivery, navigation, same-tick multi-event capture and reconnect. The fake fetch establishes client behavior, not how often the live route returns 500. |
| BAS-RF-023 | **Code finding.** The shared FileWriter retains `results` and `timelines` maps (`file_writer.go:51`, `timeline.go:21`). Completed executions defer `ForgetExecution`, but `artifact_config.go:45` removes only per-execution settings. No result/timeline removal or replacement was found in this owner. | Give accumulated execution data a bounded lifetime and release it after durable finalization, including failure/cancellation. Preserve durable readback and prevent late writers from resurrecting retired entries. This is a retained-data mechanism in current source, not a measured allocation breakdown of the running 10–12 GiB heap. |
| BAS-RF-024 | **Isolated service reproduction.** After 1,001 successful synthetic durable writes, the cache has evicted 500. `recording/service.go:317` answers from that tail, reports total 501, performs zero repository reads, and returns the same first sequence (902) for offsets 0 and 100. | Durable history remains fully addressable while the session is active. A bounded tail cache cannot define total history or ignore pagination. Test first/next/oldest pages, filters and recording-to-workflow selection across eviction and restart. |

The network problem is independent of whether response bodies/HAR are enabled.
It corrupts basic URL/status attribution on the affected collector path. The
pagination problem hides durable older actions rather than proving the database
deleted them. The queue leak and writer retention are two separate mechanisms;
neither alone is a complete explanation of the production memory samples.

### Earlier findings strengthened by execution

- **BAS-RF-002:** one failing repository write still returns nil and broadcasts
  one recorded action. Restart survival remains untested.
- **BAS-RF-004:** the unmodified generated recording script emits `a`, `ab` and
  no event for a final empty value. The real Go merge returns `aab`. Same-selector
  inputs in different pages or frames collapse into one action. Stopping while
  an edit is buffered emits no final edit in the script-level fixture. IME,
  contenteditable, shadow DOM, and end-to-end stop ordering remain open.
- **BAS-RF-006:** after a failed launch, two waiting callers create two replacement
  browsers for one key; the pool tracks/closes only one. A launch finishing after
  `closeAll` repopulates the pool. These use synthetic Browser objects, so they
  prove ownership failures, not measured Chromium process growth.
- **BAS-RF-007/008:** the actual direct-server connection/broadcast methods send
  ten synthetic frames to a subscriber without a session. A wrong-session
  subscriber receives none, which is a useful control. A matching socket with
  64 MiB already buffered receives ten additional sends. No listening port or
  real session was used; network exploitability and total process memory were
  not measured.
- **BAS-RF-009:** a two-frame ring containing two 1,000-byte frames reports average
  size zero after three older skipped frames. `performance/collector.go:131`
  subtracts lifetime skips from retained-frame count. Fix the window semantics
  before treating debug bandwidth/frame-size statistics as optimization evidence.
- **BAS-RF-017:** a synthetic password reaches the script's fetch body literally,
  with `elementMeta.attributes.type=password`. This confirms a raw transport
  disclosure at the capture boundary. No real credentials were used, and this
  does not yet establish downstream storage/export/AI exposure.

### Probe method and reproduction

From the repository root:

~~~bash
LOG_LEVEL=error node scenarios/browser-automation-studio/docs/internal/refactor_probes.cjs
~~~

From `scenarios/browser-automation-studio/api`:

~~~bash
GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_probes.go
~~~

The Node probes use installed TypeScript/ts-node and actual modules. Logging is
stubbed to avoid unrelated generated-module loading. Network objects use the
installed Request prototype; the recorder runs its complete generated script in
a VM with a synthetic DOM, timers, storage and fetch. The input-order probe also
stubs the HTTP parser and frame-viewport dependency. Direct frame probes call
the real methods with synthetic sockets. Go probes import the real modules and
use fake repositories/hubs. Source and probe hashes are in the evidence JSON.
No browser rendering, OS event fidelity, HTTP transport, native app packaging,
or platform certification is implied by these probes.

The first local Node attempt encountered an unrelated generated-ESM loading
error through the logging barrel; the probe now substitutes that facade. This
is harness setup history, not a new BAS product defect. Both final probe
commands completed with no probe errors. Per `docs/TESTING.md`, this diagnostic
scope did not require a comprehensive scenario suite; fixes must add normal
owner regressions and obtain scoped Test Genie receipts.

## Profile and replay investigation — 2026-09-22 UTC

The operator requested another investigation pass. This pass added
[profile/replay probes](refactor_profile_probes.go), [input-hook probes](refactor_input_probes.cjs),
and [dated evidence](REFRACTOR_PROFILE_REPLAY_2026-09-22.json). It produced 25
observations: 20 desired-behavior mismatches and five successful controls. They
are selected fault/semantic cases, not a production failure rate or 20 separate
issues. Product source digest remains identical to the initial baseline.

The Go probes use the actual encrypted file repository over an in-memory fake
filesystem, actual services and HTTP handlers with synthetic driver state, and
the actual workflow generator plus strict V2 write ingress. The Node probe runs
the real input hook and coordinate mapper with synthetic React hook primitives
and a WebSocket sink. No production profiles, real credentials, browser
processes, or service lifecycle were changed. The synthetic encryption key is
process-local and is not a deployment default. This is not physical disk
power-loss testing, real browser replay, or an OS shortcut/IME certification.

### Profile durability and recovery

| ID | Observed in the isolated owner probes | Consequence / closure boundary |
| --- | --- | --- |
| BAS-RF-025 | `persistence/file_repository.go:159` saves protected state before metadata commit; `protected_state.go:69` writes it directly. Injecting metadata rename failure returns an error but a fresh reader gets old metadata with new state. An interrupted protected write makes the previously acknowledged profile unreadable. | Atomic metadata rename is not atomic profile persistence. Commit a coherent encrypted generation and preserve the previous committed generation on every failed write. Qualify real filesystem crash durability separately. |
| BAS-RF-026 | `session-profile/service.go:272/292` independently read, modify and save the whole profile. A gated tab save reads the old snapshot, a storage save succeeds, then the tab save succeeds and restores the old storage snapshot. No concurrent filesystem writes are needed to reproduce it. | Serialize or version aggregate updates. Profile leases/forks and conflict receipts must cover browser sessions, navigation-history writers and workflow save-back. Two success receipts must not conceal lost updates. |
| BAS-RF-027 | Removing the protected file returns a profile with empty storage and no error (`protected_state.go:79`). With the wrong valid-length key, direct Get correctly errors, but List skips the unreadable profile and returns an empty list without error (`file_repository.go:95`). GetOrCreateProfile then creates a new default profile without reporting the recovery failure. | Preserve identity and report incomplete/locked/corrupt state. An unreadable profile is not an absent profile, and missing protected bytes are not a successful empty snapshot. Recovery must not silently replace a signed-in workspace. |

**BAS-RF-003 is now an HTTP-handler reproduction:** with a repository that rejects
saves, PersistRecordingSession returns HTTP 200 and `status=persisted`.
CloseRecordingSession closes the synthetic browser, returns HTTP 200/closed,
and removes its profile association despite failed final persistence. A retry
through that association can no longer recover the session. This is stronger
than the earlier source-only finding; real browser crash frequency remains
unmeasured.

There is useful code to preserve: a normal encrypted save/load succeeds through
a fresh repository instance, fixture state does not appear as plaintext in
either persisted file, and missing encryption keys fail closed. Service-level
EndSession and PersistSessionState aggregate methods already exist, but a source
search found no production callers; the inspected live-session handlers still
use separate SaveStorageState and SaveOpenTabs calls. Consolidating those entry
points should reuse one aggregate owner, while correcting the repository commit
semantics underneath it.

**BAS-RF-011/012 qualification:** the current driver still snapshots storageState
without IndexedDB inclusion; the supported authentication-storage matrix remains
unexecuted. `docs/ENVIRONMENT.md` documents BAS_SESSION_STORE_KEY. The checked-in
desktop bundle does not name it, and a presence-only check found it absent from
the running API's startup environment. No key values were printed. The deployed
binary is an older dirty revision, so this does not prove a live profile-creation
failure. It does identify an unresolved packaging obligation: provision and
retain the profile key, detect wrong/missing keys explicitly, and qualify
backup/restore/update and cross-OS transfer without silently losing sign-ins.
Do not remove encryption to make setup appear functional.

### Human input and recorded-workflow semantics

| ID | Reproduction | Required behavior |
| --- | --- | --- |
| BAS-RF-028 | Actual generation plus typed ingress drops click/keyboard modifiers, double-click count and horizontal scroll. A recorded `blur` and the browser script's `drag-drop` both become valid clicks. The registry spelling `dragDrop` produces an unspecified action and fails typed ingress. | Preserve action meaning across capture/normalization/generation. An unsupported observation must remain visible and fail explicitly, not become a different executable action. Test a versioned action corpus with an independent semantic oracle; JSON/proto validity alone is insufficient. |
| BAS-RF-029 | `useInputForwarding.ts:262` forwards Ctrl+A, Command+C and Alt+F as plain text a/c/f. Shift-pointer-down omits modifier state. An `isComposing=true` Process keydown becomes a standalone keyboard command. Plain x and Shift-Tab are successful controls. | A browser input protocol distinguishes text/composition, key transitions/chords, pointer modifiers and clipboard operations. Preserve platform-specific shortcuts and test browser/OS effects; the hook probe proves the emitted payload only. |
| BAS-RF-030 | Two unmerged clicks from distinct pages and frames produce click/wait/click with no target-context binding or switch. V2 ingress accepts the generated flow. `GenerateWorkflow` consumes action records; the inspected conversion emits neither page/frame identity nor page lifecycle operations. | Derive replay from logical page/frame context and the relevant lifecycle observations. Resolve captured context into a new execution; do not reuse expired runtime UUIDs as replay locators. Test tab creation/switching/closure, popup openers and nested frames independently of selector merging (RF-004). |

The generation path discards the second result of each registry builder
(`workflow_generator.go:253`, `data, _ := cfg.BuildNode(action)`). Several builders
place execution details in that discarded config. The subsequent V2 conversion
copies a narrower field set, while unknown action types default to click in
`action_registry.go:112`. These are concrete contract mismatches, not a reason
to keep a permanent legacy/V2 dual conversion. Consolidate one typed semantic
conversion and remove the competing mappings after preservation evidence.

The UI already attaches a page_id to its input message, but the inspected Go
WebSocket forwarder passes only the input object, and the driver chooses
session.page. This strengthens RF-020's need for target/lease binding under tab
switch races; this pass did not reproduce a real cross-tab race.

### Deployed leak mechanism: read-only binary evidence

An offline read of `/proc/1647308/exe` using `go tool nm` and `go tool objdump`
confirms both mechanisms in the deployed artifact: the sink factory returns
the UX Collector wrapper, and workflow cleanup compares its interface type to
WSHubSink before calling CloseExecution. The executable SHA-256 is
`031ee0664d8a1d312971dcf37f66be6cbdb6cbfd436844d833f4d29cdf13572d`.
Relevant symbol and disassembly excerpts are retained in the evidence JSON.
No debugger was attached and no process memory was dumped.

This closes the narrow uncertainty about whether RF-021's compiled mechanism
exists in that binary. It does not identify the active stack of each goroutine,
prove which factory branch every execution took, or allocate the live heap by
owner. At 00:34:03 UTC the API still reported healthy, 10,328 goroutines, and
11,762.77 MiB allocated Go heap. No trend or leak rate is inferred.

Capability discovery and its fallback returned unrelated matches for live Go
heap/stack inspection. The inspected BAS source exposes health and frame
performance diagnostics, with no registered heap/stack profiling endpoint found.
This pass obtained no production heap/stack profile; it does not establish that
no such capability exists elsewhere in Vrooli. A controlled matching-build
profiling cohort remains the next attribution step.

### Reproduce and interpret this pass

From the repository root:

~~~bash
node scenarios/browser-automation-studio/docs/internal/refactor_input_probes.cjs
~~~

From `scenarios/browser-automation-studio/api`:

~~~bash
GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_profile_probes.go
~~~

Both commands completed. Source/probe hashes and every expected/actual result
are in the evidence JSON; false means a desired-behavior mismatch. The controls
cover encrypted roundtrip, absent-key refusal, basic generated click, plain text,
and Shift-Tab. No scenario certification suite was run for these diagnostic
artifacts. Existing snapshots and probe sources from earlier passes are retained.

## Session, frame and retention investigation — 2026-09-22 UTC

The fourth pass adds six issues (BAS-RF-031–036) and executable evidence for
RF-005/007. The [dated results](REFRACTOR_SESSION_FRAME_2026-09-22.json) retain
21 observations: 13 desired-behavior mismatches and eight successful controls.
These are selected adversarial cases, not a production failure rate or 13
independent bugs. No product code, dependency, configuration, real browser,
profile or production evidence was changed. No implementation goal was launched.

The shared repository HEAD advanced to
`629defb5414eac2a4904996ee6d235eea9d263ed` during this pass. BAS's comparable
source inventory digest remained
`2df361d5baaaf501715527e57993b6024ca63e798713f5d9e655a9682f584f43`, the same
as the initial baseline. The prior driver edits remain present. This pass adds
no live memory sample or attribution to the older deployed executable.

### Session ownership and cleanup

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-005, strengthened | **Actual manager/guard, synthetic browser dependencies.** Two distinct execution IDs concurrently call `startSession` with maxConcurrent=1. Both succeed, creating two contexts and retaining two sessions. The same-ID control creates one context and returns one session. `session/manager.ts:318` checks only inserted sessions before async creation. | Reserve admission capacity before awaiting dependencies and release reservations on every failure/cancel path. Test distinct IDs, same-ID retries, reuse and shutdown with a controlled real browser cohort. |
| BAS-RF-031 | **Actual manager/decisions/state machine.** Seed a session in executing phase and repeat its start request. `session-decisions.ts:203` equates the same execution ID with a dead prior attempt; `manager.ts:234` changes the phase to ready. The lease stays the same and `canAcceptInstructions` changes false → true. The route's executing guard is in `routes/session-run.ts:192`. No actual browser instruction runs in this fixture. | Idempotent start returns existing ownership without changing active work. Recovery requires evidence of expiration/cancellation and a fenced operation generation. Test a delayed live action plus retried start; prove the next action cannot overlap it. |
| BAS-RF-032 | **Actual reset owner, fake page/context I/O.** With pages [first, second] and second active, successful reset closes second, leaves first open, retains the closed second as the sole page, keeps both page-map entries, and reports phase ready. Separately, injected navigation failure leaves phase resetting; idle cleanup still refuses it one hour later. `session-reset.ts:14–34` and `session-decisions.ts:269` establish the paths. | Define a complete clean-session operation with explicit page selection, coherent page maps and recoverable failed state. Separate fresh E2E isolation from persistent human browsing. Qualify all supported origin stores and tabs; do not infer complete isolation from clearing cookies alone. |
| BAS-RF-036 | **Actual close route, manager and teardown with injected browser errors.** Page close, context close and tracing.stop each throw. The route returns HTTP 200 with success=true and the never-written trace path; manager removes the session while the fake page remains open. `session-teardown.ts:20–43`, `manager.ts:1163` and `routes/session-close.ts:43` discard failure information across layers. | Teardown returns structured evidence and resource outcomes. Preserve ownership of unresolved resources for bounded recovery; do not return an unverified artifact path as an ordinary successful close. Required evidence failures affect execution outcome. Test flush, close, detach and concurrent-close faults through complete workflow finalization. |

The reset's later page-selection probe deliberately makes navigation and storage
clearing succeed. It does not prove that storage access after about:blank succeeds
in Chromium. That real-browser behavior, all-origin/IndexedDB/service-worker
isolation and multi-tab replay remain unqualified. The injected failure case
independently proves the stuck-phase handling problem. External-target reset
refusal, a ready-session retry and ordinary successful close all passed controls.

RF-036 is a close-boundary result, not a claim that every workflow consumer
accepts missing trace bytes. It joins RF-013's capability/completeness work while
tracking resource ownership separately. An error logger is not a recovery owner
or an accurate result for the caller.

### Frame admission, identity and fallback

These probes transpile the unchanged `useFrameStream.ts` hook and execute it
with an explicit synthetic hook scheduler, controlled bitmap promises, clock,
WebSocket and canvas. They exercise effect dependency changes and cleanup, but
do not measure React renderer, browser decoder, network, GPU or paint timing.

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-007, strengthened | The hook receives 100 frames before any decode resolves and starts 100 `createImageBitmap` calls. The existing stale-frame check happens after decoding (`:412–415`); it does not bound admission. | Bound active decoding and retained compressed bytes, coalesce pending frames, and record discard reasons. The count is outstanding promises, not measured browser threads or memory. Qualify slow-decoder and slow-reader cohorts independently. |
| BAS-RF-033 | Two frames arriving in the same mocked millisecond use the same Date.now ID (`:509`). Resolving the newer bitmap first and the older second paints [newer, older]. A decode from session A can paint after switching to B. A decode resolved after unmount schedules another paint without closing its bitmap, and a pending config response can create a socket after cleanup (`:526–636`). | Monotonic sequence plus session/page generation, checked after every async boundary. Dispose stale decoded frames, ignore stale connection completions and cancel scheduled work. Timestamp is not identity. Qualify rapid tab/session changes, unmount/remount and delayed polling/config/blob/decode completion with a real renderer. |
| BAS-RF-034 | Socket open with zero delivered binary frames stops fallback: the probe observes one initial polling request and zero scheduled polls after connection. The outer effect checks both socket activity and received-frame state, but its tick returns on socket activity alone (`:771–788`). | Base fallback on delivered/renderable frame freshness. An open socket is not proof that the stream supplies current frames. Test connected/no-frame, quiet page, malformed/failed decode, stream stall and recovery without duplicate polling. |

The older-frame control with distinct timestamps does discard and close the old
bitmap. Thus the previous stale-frame guard is real; it needs stronger identity
and lifecycle semantics. The unmount probe asserts scheduled work and bitmap
ownership, not that a removed DOM canvas visibly paints. Same-millisecond
delivery is intentionally forced; live frequency is unmeasured. Comments claiming
67% latency improvement in this hook are historical context, not new benchmarks.

### Retention must make progress without weakening protection

**BAS-RF-035 — isolated service reproduction.** `retention.Sweep` computes
keep_latest protection from its candidate subset (`api/services/retention/retention.go:189`).
The optimized repository query first restricts that set to the oldest MaxItems
rows per status (`:395–401`; `api/database/repository.go:592`), and applying
preview IDs restricts it again (`:351`). A one-workflow fixture with three
terminal executions shows:

- An unbounded sweep with keep_latest=1 deletes the two older entries and keeps
  the newest, as intended.
- Three consecutive max_items=1 / keep_latest=1 sweeps delete zero entries each.
  The oldest entry is repeatedly protected as the newest in its one-row subset.
- A preview correctly marks an older entry eligible; applying that entry alone
  with keep_latest=1 now protects it and deletes nothing.
- Bounded deletion without keep_latest and protection of a running execution
  pass controls.

The fake index models the production oldest-first SQL LIMIT behavior; no SQL or
real deletion is run. The owner cleanup path supplies bounded batches
(`api/owner_cleanup.go:189–211`), so this is a relevant composition, not just
an unreachable argument combination. Incidence under deployed keep_count and
workflow distributions is unknown. BAS also has a separate directory-budget
enforcer; this finding does not prove all evidence retention is disabled.
The automatic policy defaults keep_count to zero (`owner_cleanup.go:394`);
the demonstrated interaction requires a positive keep count. This pass did not
inspect or change the deployed retention environment.

Calculate protection against authoritative per-workflow history independently
of the work batch, preserve it when applying a preview, and recheck active-run
eligibility at apply. Test multiple workflows/statuses and repeated sweeps;
limiting each tick must still make progress. Do not remove keep_latest or active
protections as a shortcut.

### Reproduce and interpret this pass

From the repository root:

~~~bash
node scenarios/browser-automation-studio/docs/internal/refactor_session_frame_probes.cjs
~~~

From `scenarios/browser-automation-studio/api`:

~~~bash
GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_retention_probes.go
~~~

Both commands completed with exit 0, meaning the diagnostics ran. False
expected_behavior_met values remain open defects. The dated JSON records all
results, probe/source hashes, control outcomes and fixture limits. No scenario
certification suite was run for diagnostic/documentation-only changes. Existing
evidence snapshots and source probes remain unchanged.

## Execution retry and cancellation investigation — 2026-09-22 UTC

The fifth pass adds BAS-RF-037–041 and strengthens RF-013/036. Its
[dated evidence](REFRACTOR_EXECUTION_2026-09-22.json) contains 22 observations:
16 desired-behavior mismatches and six successful controls. These are selected
adverse cases, not 16 independent bugs or an estimate of production incidence.
HEAD remains `629defb5414eac2a4904996ee6d235eea9d263ed`; the comparable BAS
source digest remains the initial
`2df361d5baaaf501715527e57993b6024ca63e798713f5d9e655a9682f584f43`.
No product/runtime/dependency changes or implementation launch occurred.

### Execution identity is a cross-language contract

The Go probe runs the public SimpleExecutor through the actual Playwright
engine, session manager, Session and driver client, using an in-memory HTTP
transport. It captures the requests generated by a two-attempt instruction and
a two-iteration repeat loop. The Node probe then feeds those exact bodies into
the actual run route, instruction pipeline and caches with synthetic browser
handlers. This is a composed source-level reproduction; there is no socket,
browser or real website effect. The Node proto codec and output serialization
are explicit seams, so generated-code interoperability is not certified here.

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-037 | **Actual route and manager phase methods.** An instruction in initializing, resetting or closing phase still invokes its handler and returns HTTP 200. The manager rejects the transition to executing, but the route ignores that false result (`routes/session-run.ts:192–209`, `session/manager.ts:1017`). Resetting/initializing end up ready after the unauthorized run. The executing-phase control correctly returns 409 without calling the handler. | One atomic instruction-admission operation checks the full state machine and reserves an operation before awaiting I/O. Rejected transitions cannot proceed. Test delayed reset/close/initialization and normal recording/execution transitions with real target ownership. This is distinct from RF-031's start retry incorrectly changing a valid busy phase. |
| BAS-RF-038 | **Go wire and actual driver route.** The Go Session stores execution and lease IDs but Run sends neither (`api/automation/session/session.go:46`, `driver/client.go:803`). A synthetic request explicitly supplying an expired lease and old owner is executed against the current owner. Close/release carry lease tokens; run does not validate them. | Bind every mutating operation to execution, immutable lease and operation generation. Validate ownership before cache lookup or mutation. Qualify delayed old-owner requests across release/reacquisition, not just ownership-protected close. This proves a protocol gap, not cross-user exposure or an internet exploit. |
| BAS-RF-039 | **Go-to-driver request composition and actual caches.** Two retry requests are identical; the driver executes once and returns the first retryable failure twice. Two loop requests are identical; it executes once but returns two successes. The key is only nodeId:index (`instruction-executor.ts:168`; `session-run.ts:237`). Changing URL under that identity returns the old success without a conflict. The header cache also ignores lease/payload changes. Conversely, an injected handler exception after a simulated effect leaves no cached instruction; retransmitting the same idempotency key performs the simulated effect twice. | Distinguish workflow node, dynamic invocation/iteration, explicit attempt and transport retry. Bind request identity to payload digest and ownership; detect conflicts. Track uncertain effects separately from safe retryable failures. Test counters/postconditions independently of BAS's own step-success records. Adding an attempt suffix alone would risk repeating uncertain effects. |

The Go retry control deliberately supplies first-failure/second-success responses
and passes. The second program shows that the actual driver cache prevents that
second handler attempt. The repeat-loop Go fixture records two body outcomes and
one loop outcome; it does not prove two browser effects. Identical transport
repetition correctly executes once, and different step indices correctly execute
twice in controls. Preserve those protections when fixing dynamic invocation
identity. A safe transport retry must not become a new business action.

The header-cache handoff case models owner/lease replacement and clearing the
session instruction map. The inspected label-reuse path does clear that map
(`manager.ts:289–291`) without clearing the HTTP cache. The current Go run client
does not supply x-idempotency-key; this additional branch concerns callers that
use that header. The counter-based uncertain-effect case makes no claim that a
live site action was duplicated or that all thrown errors follow an effect.

### Cancellation and failure evidence

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-040 | **Actual public executor, synthetic cancelling engine and context-sensitive memory writer.** Cancellation during a linear step saves one terminal outcome and closes the session. The equivalent graph step attempts persistence with a cancelled context, saves zero outcomes and returns `record step outcome: context canceled`. Compare `simple_executor.go:437` with `flow_executor.go:189`. | All graph/linear/loop/subflow termination paths persist terminal step evidence using a bounded context independent of action cancellation. Preserve cancellation as the primary outcome and report persistence failures separately. Qualify real storage, cancellation during capture/flush and resume reconstruction. This does not prove that the top-level execution status is lost. |
| BAS-RF-041 | **Actual instruction pipeline, telemetry orchestrator and console collector.** An injected thrown handler exception after a synthetic console event returns an error with zero logs and no screenshot/DOM capture. The exception branch disposes collectors before collecting failure evidence (`instruction-executor.ts:259–267`). A returned success=false result captures one screenshot, DOM and console entry and releases its listeners correctly. | Unexpected throws retain already-observed evidence and attempt bounded failure capture when the target permits. Always release collectors; distinguish capture failure from action failure and uncertain effect state. Qualify crash/disconnect and ordinary handler failure contracts separately. |

RF-041 applies to unexpected exceptions escaping a handler. The inspected
interaction/assertion handlers normally catch errors and return typed failure
results, which the control preserves. Do not describe this as loss of every
timeout's evidence. No production SQL mutation or browser renderer was used.

Two earlier issues gain evidence across a second ownership boundary:

- **RF-013:** the public Go executor converts an explicitly failed screenshot
  step to success=true, removes its Failure and returns nil for the workflow
  (`simple_executor.go:1502`, `:1560`). The fixture opts into no optional-capture
  policy. The error message/code remain in non_fatal_failure notes, so the
  information is not completely erased. The target needs explicit required vs
  optional capture semantics that affect the terminal verdict, not only a note.
- **RF-036:** even when the engine reports a close error, the Go finalizer only
  logs it and Execute returns nil (`simple_executor.go:1170–1180`, `:1244`).
  Fixing the driver's close receipt alone will not make the workflow result
  reflect failed cleanup. The fixture uses the plain Close interface; artifact
  persistence/download failure paths still need complete qualification.

Recall found July work fixing ignored exists-assertion timeouts; current
`handlers/assertion.ts:201` waits for attachment with the timeout, so that old
cause is not reported again. Current top-level workflow persistence and the
linear step path detach from cancellation, and cancellation registration occurs
before starting the async runner (`services/workflow/executions.go:455`). Those
are useful protections, not absent features. Recall's source-ledger providers
timed out and swarm-manager records remained degraded; prior-art coverage is
therefore incomplete.

**Unresolved cancellation boundary:** StopExecution invokes a stored cancel
function and the RPC returns status stopped, while browser cleanup happens in
the runner. This pass does not measure the interval until browser actions stop,
prove a live action continues after that receipt, or certify timeout cleanup.
The next fixture needs correlated cancellation acceptance, last applied browser
operation, evidence flush and terminal resource ownership. HTTP request
cancellation alone is not an adequate oracle for browser quiescence.

### Reproduce and interpret this pass

From `scenarios/browser-automation-studio/api`:

~~~bash
GOPROXY=off GOTOOLCHAIN=local go run ../docs/internal/refactor_execution_probes.go > /tmp/bas-execution-api-probes.json
node ../docs/internal/refactor_execution_probes.cjs /tmp/bas-execution-api-probes.json
~~~

Both completed with exit 0, meaning the diagnostics ran. Inspect each
expected_behavior_met value. The evidence JSON stores both result sets, exact
Go request packets, their source-output hash, probe/source hashes and limits.
The six controls cover Go retry scheduling, linear cancellation persistence,
busy-phase rejection, transport deduplication, distinct steps and returned
failure telemetry. No scenario certification run was needed for these
diagnostic/documentation-only changes.

Prioritize a real-browser counter fixture for loops, retries and cancellation
before treating a green timeline as proof of effects. This finding reinforces
the proposed independent oracle and versioned operation contract. Broad file
splitting and additional generic pass-rate measurements will not resolve it.

## Reuse, profile and evidence investigation — 2026-09-22 UTC

The sixth pass tests whether the retained browser resource agrees with the
identity, target and evidence requested by its next owner. The
[dated snapshot](REFRACTOR_REUSE_2026-09-22.json) contains **17 observations:
10 desired-behavior mismatches and seven successful controls**, mapping to three
new issues. This is a selected diagnostic corpus, not a live failure rate.

The [probe](refactor_reuse_probes.cjs) executes the actual start route, session
manager, reuse decisions, reset, artifact-path resolver, teardown and app-target
validators. Contexts, storage markers, HTTP objects, capture handles, context
construction and filesystem operations are synthetic. No real signed-in state,
browser, network request, desktop renderer, Android device or artifact file was
accessed. The source digest remains unchanged from the initial assessment;
repository HEAD remains 629defb5414eac2a4904996ee6d235eea9d263ed.

### Reachability and controls

These failures require a ready session whose current owner explicitly released
its exact lease, a new execution requesting reuse/clean and matching labels.
The release route exists, and the probe uses the actual release method before
starting the next owner. Unreleased sessions and distinct labels are correctly
excluded. Fresh mode creates a different context. Compatible released reuse
rotates the lease and retains the intended context.

The inspected ordinary Go executor finalizes with CloseWithArtifacts or Close
(`api/automation/executor/simple_executor.go:1168`, `:1244`). A scoped search of
the BAS API found the Release method and driver release client but no production
Go caller of `.Release()`. Do not claim these failures happen after every normal
workflow, that unrelated users shared a live profile, or that active-owner
sessions can be stolen through this reuse path. Release-aware callers can reach
it; actual usage and frequency remain unmeasured.

The executor supplies only fake_microphone as its pool label
(`simple_executor.go:208`); the Go session manager passes those labels through
(`api/automation/session/manager.go:198`). In the driver, label matching checks
only the requested key/value pairs; it does not compare profile, target or
context configuration (`session/session-decisions.ts:78`). Empty label maps also
match any labelled candidate. This broad matching is a source finding, not a
separate issue or evidence of live cross-user exposure.

### Findings and closure criteria

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-042 | **Actual start/manager/reset, synthetic context identity.** Owner A releases its session; owner B requests a different storage state, locale, proxy and 390×844 viewport with the same labels. Reuse returns HTTP 200, stores B's request in spec, builds zero contexts, and retains A's context identity, locale/proxy and 1280×720 page. Clean reuse clears the synthetic cookie marker but never imports B's state; immutable settings remain A's. | Compare a stable effective context descriptor before transfer. Reject or recreate incompatible profile/configuration requests; preserve compatible signed-in continuation. Fixture assertions must read independently authenticated identity and effective mobile/locale/proxy behavior, not only echoed spec. Maps to preservation journeys 1, 15 and 19 and session-readiness/correctness targets. |
| BAS-RF-043 | **Actual artifact path resolver/reuse/close/teardown; simulated capture and filesystem.** B requests its own video/HAR/trace paths; effective paths remain under A. Closing B returns A's HAR/trace references and simulates moving a video to `/synthetic/a/videos/execution-b-page-1.webm`. Starting with capture disabled, then requiring all three on reuse, still returns HTTP 200 with video=false, tracing=false and no HAR path. | Establish a capture boundary and immutable execution ownership before transfer; effective required capabilities must be satisfied before ready. Recreate a context when creation-time collection cannot be reconfigured. Verify file contents, manifest execution IDs and checksums with two sentinel executions. Maps to journeys 8 and 20 and evidence-completeness targets. |
| BAS-RF-044 | **Actual route/manager and app-target validators.** A fresh Electron or Android request without validation context is rejected before renderer access. The same request with matching released reuse returns HTTP 200 and keeps a managed browser, while spec now names the external target. A seeded external session also satisfies a managed request while remaining external. | Every admission path validates target identity, context, isolation lease and required capabilities before considering reuse. Exact target identity and kind constrain compatibility; report incompatibility instead of silently substituting a renderer. Qualify managed↔Electron↔Android transitions through their owning fixtures. Maps to journeys 9 and 21 and platform correctness targets. |

The transfer occurs at `session/manager.ts:285–311`: it updates execution/lease,
clears instruction history and replaces spec, but preserves context, page and
capture fields. New-context construction, where storage/profile/viewport and
HAR/video/trace options are applied, occurs later at `:395` and in
`session/context-builder.ts:170–275`, `:463`. Clean's existing reset is not a
profile import. Its localStorage evaluation was deliberately stubbed successful;
the actual about:blank storage behavior and all-origin reset matrix remain
unqualified, as recorded under RF-032.

External-target branching is later still in the admission decision, after the
reuse return (`manager.ts:330`). The existing external path has substantial
validation (`:725–756`) and rejects unsupported creation-time capture requests;
RF-044 is a bypass of those checks, not their absence. Fresh target rejection and
the route's required-artifact-root rejection both pass in the controls.

For RF-043, trace handles and video streams are retained across transfer, and
teardown uses old trace/HAR fields but the new spec execution ID for the video
filename (`session-teardown.ts:24`, `:44`; `manager.ts:1171`). This proves mixed
destination/receipt selection in the synthetic composition. Whether a real file
spans both executions, includes sensitive content, or is indexed under the wrong
execution still needs a real-browser sentinel fixture. No artifact corruption or
disclosure is claimed from path selection alone.

The required_capabilities object is a requirement declaration. False does not
necessarily prohibit collection, so continued capture when a new caller does
not require it is not counted as another defect. The target should separately
define required capability, collection policy and retention permission.

### Architectural consequence and reproduction

Do not repair this by copying the requested spec into more state fields. Keep
requested configuration distinct from an immutable effective context descriptor,
and admit only compatible transfers. Browser-process pooling and reuse of a
signed-in context are different decisions: BAS can retain a browser process
while allocating a fresh compatible context. A person's continued browsing can
retain the same profile context when explicitly intended. Pool labels alone
cannot decide those semantics.

Capture needs its own execution boundary even when the browsing context persists.
Some collection can rotate; creation-time HAR/video configuration may require
a different context. The next architecture experiment should measure that cost
against session-readiness targets while preserving required evidence. No new
performance number is inferred from these control-flow probes.

From the repository root:

~~~bash
node scenarios/browser-automation-studio/docs/internal/refactor_reuse_probes.cjs
~~~

Exit 0 means the diagnostics completed; inspect expected_behavior_met. The dated
snapshot includes all results, controls, source/probe hashes and reachability
limits. No scenario certification run was needed for documentation and isolated
diagnostics. Recall found earlier minimal-capture-policy work, so existing policy
support was preserved in this analysis. Source-ledger timeouts and degraded
Swarm records still limit prior-art coverage.

## Stream lifecycle and live fixture investigation — 2026-09-22 UTC

The seventh pass follows capture ownership through replacement, asynchronous
startup, resize, stop and settings updates. The
[snapshot](REFRACTOR_STREAM_2026-09-22.json) preserves **15 isolated observations:
10 desired-behavior mismatches and five successful controls**, a separate live
browser control and read-only CLI reproductions. RF-045–049 are new; RF-009/034
gain evidence. These populations must not be combined into a failure rate.

The [diagnostic](refactor_stream_probes.cjs) executes the actual frame manager
and CDP strategy with deferred browser/CDP operations, synthetic sockets,
collectors and a controlled clock/scheduler. No product code changed. BAS source
digest remains unchanged across seven passes. Repository HEAD and API PID remain
the same as the preceding pass; matching-build certification remains absent.

### Producer ownership and settings

| ID | Evidence and result | Required behavior / next proof |
| --- | --- | --- |
| BAS-RF-045 | **Actual manager with deferred start/stop.** Replacement starts while the old handle is stopping. Once old cleanup completes, getFrameStreamSettings returns null while the replacement capture and socket remain active. A later stop cannot reach them. A start resolved after stop also leaves an active untracked handle. Failed startup deletes its entry without closing its socket. Settled start/stop closes both correctly. | One generation owns registry updates, pending work, capture and transport. Dispose late acquisitions and close failed-start sockets; old cleanup cannot delete a new owner. Qualify concurrent restart/stop, failed startup/fallback and resource counts with a real driver. Maps to journey 22 and resource stability targets. |
| BAS-RF-046 | **Actual CDP strategy, controlled awaits.** Stop finishes while resize awaits a new CDP session; resolving that acquisition then issues Page.startScreencast on an undetached session while isActive is false. Separately, a buffered frame from page A is sent before page B's current frame after a switch. Normal resize and settled stop controls pass. | Fence every asynchronous capture transition and buffer with generation/page identity. Stop disposes late sessions and prevents restart. Real tab/resize/close stress must show no post-stop capture or old-page publication. Maps to journeys 5 and 22. |
| BAS-RF-047 | **Actual manager composed with actual CDP strategy.** Updating quality from 65 to 20 reports success but leaves the current screencast at 65; a later resize applies 20. Updating FPS to 1 reports fps=1 and current_fps=1, while 60 synthetic compositor events in one simulated second all send. Enabling perfMode reports true while outgoing frames keep their original timestamp format. | Requested/effective/pending/unsupported states must be explicit. Apply or reject control changes; measure current FPS independently of target. Verify bytes and capture options through the public settings endpoint and both strategies. Maps to journey 23 and motion/congestion/measurement targets. |

RF-045 follows `frame-streaming/manager.ts:82` (fire-and-forget old stop),
`:145` (unconditional registry deletion), `:373` (late handle assignment) and
`:437` (failed-start entry removal without socket disposal). The synthetic test
cleans up its fake objects explicitly after observing loss of product ownership.
This does not quantify a real Node leak or explain the API's Go heap pressure.

RF-046 follows `strategies/cdp-screencast.ts:151–174`: restart acquires CDP and
starts capture without rechecking the active generation. Pending frame state is
not invalidated when the active page changes. This producer-side failure is
distinct from RF-033's viewer decode/config lifecycle.

For RF-047, manager settings are mutable metadata but the strategy receives a
configuration snapshot. CDP has no updateTargetFps method; quality only affects
the next restart (`cdp-screencast.ts:396`). The strategy interface explicitly
describes FPS updates as polling-only, yet the public manager reports the new
target/current value without unsupported status. The diagnostics route returns
those values as HTTP 200 (`recording-diagnostics-routes.ts:152–173`), and the UI
uses this route for its quality/FPS controls. The test's simulated second is not
a real frame-rate benchmark. RF-009 also gains a concrete measurement defect:
currentFps is assigned targetFps (`manager.ts:224`).

**RF-034, strengthened:** after receiving and acknowledging the initial frame
while transport is unavailable, the CDP strategy retains it. Changing readiness
to true and ticking the page monitor sends zero frames. Only a subsequent
compositor event flushes the pending frame (`cdp-screencast.ts:310–329`). A stable
page therefore lacks a readiness-triggered delivery path in this isolated owner.
Its live interaction with the already-reproduced UI polling fallback still needs
qualification. A later paint successfully sends current content in the control.

### One real browser control and evidence limits

Execution **312e3508-94b0-4aea-88d3-5046984d1245** used the public
workflows execute-adhoc CLI with --wait and an explicitly fresh context. The
validated five-node fixture navigates to about:blank, installs a synthetic
button/counter, clicks through two distinct nodes, and asserts independently
that #count has data-count="2". The saved timeline contains success=true for
that assertion. No real website, account, profile, browser history or external
device was accessed. No reusable workflow revision was saved.

| Observation | Result | Interpretation |
| --- | --- | --- |
| Execution status and independent assertion | Completed; two click effects verified by DOM state | Confirms this ordinary typed execution path works on the deployed browser. |
| Server execution interval | 3,396.457 ms, one sample | Includes startup, configured action behavior and evidence work; not input-to-paint or a percentile. |
| Step durations | Navigate 338 ms; setup 693 ms; clicks 663/533 ms; assert 532 ms | Timeline durations only; this pass does not attribute overhead among browser, policy, capture and persistence. |
| Screenshots | Five PNGs, 82,030 bytes total; 2560×1440 for 1280×720 CSS viewport | Declares the actual artifact cohort; it is not a DPR1 performance comparison. |
| Replay evidence | Replay package f33fff50-fd50-4a62-88e1-d1777fa6ece6; evidence manifest d6225f48-adc7-4fb8-9273-06d624fe2521 | Owner-managed receipts subject to retention; selected timeline fields and receipt hashes are saved in the snapshot. |

This is one shared-host sample. It does not measure live viewer input, passive
recording, replay loops, storage isolation, macOS/Windows, mobile attachment or
resource plateau. Discovery found the adhoc leaf and browser programs, but the
inspected CLI help exposes recording import rather than a live-input fixture
controller. That is an instrumentation/reuse finding, not proof that the wider
project has no such capability. A correlated browser/viewer fixture is still the
next measurement boundary.

### Public validation and outcome fidelity

| ID | Observed behavior and source explanation | Closure evidence |
| --- | --- | --- |
| BAS-RF-048 | The identical valid fixture validates without require-assertion. Adding the documented bare flag fails before RPC with strconv.ParseBool parsing an empty string; =true is rejected because the flag accepts no value. The shared parser records Boolean presence only, but generated proto binding reads ctx.Flag as a string (`packages/cli-core/cliapp/parser.go:100`, `protobindings.go:522`, `:765`). | Repair the shared owner; exercise public CLI Boolean presence/absence with assertion-bearing and assertion-free fixtures. Preserve server enforcement. The custom adhoc handler's BoolFlag path works for --wait and is a useful control. Maps to journey 24 and truthful validation gates. |
| BAS-RF-049 | All five live timeline step_outcome artifact payloads contain string_value with a Go struct/debug representation, including pointer-shaped substrings. Current FileWriter stores the typed StepOutcome in a map (`file_writer.go:363`), then its local anyToJsonValue converts unrecognized structs with fmt.Sprintf (`:975`). Typed success, assertion and screenshot projections remain available. | Preserve versioned structured outcomes through timeline/replay conversion. Validate primitive/map/struct/pointer and failure/attempt fields through public round-trips. Determine canonical result-file fidelity and historical recovery needs before defining migration. Maps to journeys 8 and 24. |

RF-048 was published to scenario-qa as **knw-1790042809480136792** through the
[report-bug skill](../../../prompt-manager/store/skills/packs/core/report-bug/SKILL.md).
The canonical BAS row tracks its impact and closure dependency. Other scenarios'
generated Boolean bindings may share the cause, but their live behavior was not
tested. The source explanation does not assert that installed binaries match the
working tree byte-for-byte.

RF-049 is a lossy timeline projection, not proof that every outcome field or the
canonical result file is lost. The public assertion succeeded independently of
that raw artifact. A separate existing AnyToJsonValue helper handles structs with
a JSON round-trip (`api/internal/typeconv/primitives.go:347`); conversion policy
is already duplicated. Consolidate at an appropriate lower-level owner without
introducing an import cycle. Pointer addresses are normalized in the saved
diagnostic excerpt; hashes identify the original owner responses.

### Reproduce this pass

Run the isolated producer probes from the repository root:

~~~bash
node scenarios/browser-automation-studio/docs/internal/refactor_stream_probes.cjs
~~~

For the live fixture, extract live_control.flow_definition from the evidence JSON
to a temporary file, validate it, then run workflows execute-adhoc with --wait.
The snapshot contains the exact command, assertion, receipts and CLI failures.
Repeat only in a fresh isolated context; a passing control is not authorization
to run arbitrary saved workflows. Read existing evidence using:

~~~bash
browser-automation-studio executions timeline 312e3508-94b0-4aea-88d3-5046984d1245 --json
browser-automation-studio executions replay-package 312e3508-94b0-4aea-88d3-5046984d1245 --json
~~~

No scenario suite or service restart was needed for this diagnostic scope. The
producer probes report mismatches honestly despite process exit 0. Prior-art
recall again had source-ledger timeouts and degraded Swarm records; the July
timing finding reinforces using explicit DOM assertions instead of incidental
screenshot delay as a synchronization mechanism.

## Proposed architecture and cleanup targets

The design in ARCHITECTURE.md defines the proposed target. Its main rule is one owner for each semantic decision, with different delivery policies for interactive frames and durable events.

Build within the current Go/Node/React deployment first. New boundaries do not require new services, another workflow dialect, or a repository-wide language rewrite.

| Boundary | Retain/extract from | Result to seek |
| --- | --- | --- |
| Session admission and ownership | manager.ts, BrowserManager/BrowserPool, session leases | Atomic reservations, per-profile ownership, compatibility-checked released reuse, bounded pool, cancellation and cleanup deadlines. |
| Browser runtime adapter | Playwright provider/context builder; app-target validator | Capability negotiation, explicit browser/profile/DPR options, correct external-target detach, no hidden target substitution. |
| Interactive viewer/input | frame strategies, Go hub, useFrameStream | One negotiated transport; bounded latest-frame queue; ordered acknowledged input; generation IDs invalidate stale frames after resize/tab changes. |
| Recording journal | injection, pipeline manager, recording service | Typed ordered observations, commit acknowledgements, deduplication, reconnect replay, explicit gap detection. |
| Profile persistence | session-profile and recording persistence | Atomic versioned snapshots/checkpoints, no false persisted response, tests for supported storage forms and upgrades. |
| Workflow derivation | Go generator and UI ActionMergeService | One canonical normalization contract for replacement vs delta input, page/frame identity, navigation and scroll; UI presents derived result. |
| Evidence pipeline | telemetry, execution writer, capture producers, retention | Policy-driven capture, explicit execution boundaries, immutable artifact ownership, bounded work, checksummed manifest, required/optional completeness, synchronized clocks and identifiers. |
| AI task/trace interpretation | existing ai-gateway and navigation adapters | AI proposes a typed candidate from selected observations; existing compiler/assertions validate it; original trace stays immutable. |
| UI workspace | RecordingSession and associated hooks | Separate session controller, browser view, recording timeline, workflow editor, and agent controls; frame cadence does not rerender the workspace. |
| Packaging/lifecycle | control plane, scenario-to-desktop, BAS sidecar config | Explicit inventory and runtime path injection; OS/arch qualification at owner. Host remediation stays in internal/, not BAS-private repair code. |

Proposed maintainability targets: no new dependency cycles; no duplicate recording normalization policy; touched public contracts generated from proto; no new compatibility layer without an owner/removal condition. Ratchet measured complexity/duplication downward after a comparable baseline. Review functions over cyclomatic complexity 15 and modules over 500 physical lines for responsibility splits; exceptions require a cohesion rationale, not cosmetic line shuffling. Do not enforce a blanket file-size score as the definition of done.

### Rendering decision to investigate before a major UI rewrite

| Option | Benefit | Cost/risk | Recommendation |
| --- | --- | --- | --- |
| Keep bounded JPEG/CDP stream | Reuses current web/remote and driver integration | Encode/decode bandwidth and canvas/browser UX constraints | Establish a reliable measured baseline first. |
| WebRTC/video stream | Potential frame efficiency and congestion control | Encoder/transport lifecycle, text clarity, synchronization, packaging | Spike only if profiling identifies transport cost; compare equal workload and fidelity. |
| Native desktop WebContentsView | Native local rendering/input may improve everyday browser feel | Electron compatibility, recording/instrumentation bridge, security boundary, remote parity | Bounded desktop-only comparison; do not commit the core architecture to it yet. |
| Replace everything with another browser-agent framework | Could reuse some task features | Does not inherently preserve passive history, evidence, profiles, mobile/desktop ownership | No evidence supports a wholesale replacement now. |

Electron documents WebContentsView as a view displaying WebContents. That establishes an available primitive, not BAS fidelity or a performance win: [Electron API](https://www.electronjs.org/docs/latest/api/web-contents-view). Playwright documents lower-fidelity CDP attachment than its own protocol: [BrowserType.connectOverCDP](https://playwright.dev/docs/api/class-browsertype#browser-type-connect-over-cdp). This is why external-target evidence must be capability-specific.

### Persistent browsing and site compatibility

Separate three session purposes: interactive user browsing, isolated validation, and authorized agent tasks. Share action/recording contracts, not unqualified mutable profiles. A validation run must not race with a person's active signed-in browser.

Preserve the current provider adapter and compatibility settings during refactoring. Pin and govern tested engine versions. Test supported sites and record challenges, renderer errors, profile continuity, and human intervention. Human operation through an instrumented browser does not guarantee a website will classify it as a normal uninstrumented browser. Avoid an “undetectable” acceptance claim.

The current profile export is not a full browser profile. Playwright documents IndexedDB snapshot inclusion as opt-in and sessionStorage as requiring separate persistence logic: [storageState](https://playwright.dev/docs/api/class-browsercontext#browser-context-storage-state), [authentication state](https://playwright.dev/docs/auth#session-storage). Verify against the installed rebrowser version. Decide deliberately between a supported versioned storage snapshot and a dedicated persistent browser-profile directory; never share a writable profile between active owners or assume a raw profile directory is portable across OS/engine versions.

## Proposed acceptance and performance board

These values are engineering starting targets, not measured achievements or silently approved SLOs. Freeze accepted values and fixtures before execution. Record any adjustment with evidence; do not lower floors to close a task.

Reference cohort: local loopback, supported clean desktop, four modern CPU cores minimum and 16 GiB RAM, 1280×720/DPR1 fixture, pinned browser/runtime, one interactive session. Record real hardware, OS, engine, build/input digest, concurrency, CPU load, cache state, recording/evidence policy, and network conditions with every receipt. Measure cold start separately from warm. Remote tests additionally use a declared 50 ms RTT/10 Mbps link. Keep third-party page latency separate from BAS overhead.

| Outcome | Proposed acceptance | Measurement and sample |
| --- | --- | --- |
| Interactive feedback | Input-to-affected-paint p50 ≤50 ms, p95 ≤100 ms, p99 ≤200 ms local; p95 ≤200 ms in declared remote cohort | At least 1,000 correlated input/frame events per cohort; monotonic clocks/clock offset handling, not timestamp at ws.send. |
| Motion and congestion | ≥30 rendered FPS for a 30 FPS changing fixture; p95 visible frame age ≤100 ms local; no stale queue growth | Five-minute scroll/animation tests; record dropped/coalesced frames and bandwidth. Idle page need not emit 30 FPS. |
| Warm readiness | Existing browser to usable tab ≤1 second p95; application cold usable browser ≤5 seconds p95 | 100 warm trials and 30 cold trials per supported platform, with profile size declared. External site load excluded. |
| Capture | Screenshot + computed snapshot ready ≤2 seconds p95 warm on fixed fixture; exact requested viewport/DPR | 100 captures; report navigation, readiness, capture, serialization and persistence separately. No broad claim from the single 2.83-second observation. |
| Passive fidelity | Every declared supported action represented with correct final semantics; zero acknowledged-event loss/duplication | At least 10,000 fixture actions, multiple tabs/frames/navigation, disconnect/crash injection. Report unsupported events explicitly. |
| Profile durability | Supported authenticated fixtures survive normal close/reopen and API/driver restarts; no cross-profile state | Cookies/localStorage/IndexedDB matrix; profile write fault and concurrent-owner cases. Record unacknowledged checkpoint recovery window, initially ≤5 seconds. |
| Known-flow reliability | ≥99% observed first-attempt success on fixed corpus; ≤1% flake rate with same-version run groups | At least 1,000 representative repeated runs, report counts and confidence intervals; every preservation journey must pass. Keep negative tests and exploratory agent tasks separate. |
| Cancellation/recovery | Accepted cancellation stops new input within 1 second; terminal cleanup/detach ≤5 seconds; usable session recovery ≤10 seconds | Faults during navigation, recording, evidence flush, and shutdown. Never auto-repeat an uncertain side effect. |
| Memory/CPU | BAS-owned API+driver idle ≤300 MiB PSS; one fixture browser+shell ≤1 GiB PSS; idle CPU <2% of one core | Controlled desktop cohort, warm steady state; Windows uses comparable private-memory measure, separately reported. These are provisional targets, not today's RSS comparison. |
| Soak stability | After warm-up, retained memory growth ≤50 MiB/hour over eight hours; no orphan process/session after 1,000 open/close cycles | Track PSS/private memory, Go heap, Node heap, contexts, descriptors, queue bytes; concurrency 1/5/10 and slow readers. |
| Evidence | 100% required artifact completeness or explicit failed/degraded terminal result; failure screenshot/log/network context when target permits | Inject step failure, renderer crash, disk-full and export failure; verify hashes and missing reasons. Preserve retention active-run protections. |
| Desktop portability | Every accepted OS/arch row installs, launches, restores profiles, captures, records/replays, updates/rolls back and cleans up on clean host | Native owner receipts. Initial manifest x64 matrix; propose macOS arm64 explicitly. A cross-build or portability phase alone is insufficient. |
| Agent usefulness | Exact known-flow route does not require rediscovery; reported success proves caller postconditions | Compare known replay, fresh-session orientation and unfamiliar navigation separately; observe round trips, elapsed time, cost, repair/qualification success. No invented percentage improvement. |

Use relative regression protection as well: on comparable fixtures, an intervention must not worsen unaffected latency/resource results by more than 5% without reviewed explanation, and must preserve functionality. Absolute targets require calibration across machines; a noisy shared-host run is not sufficient to accept or reject a small delta.

### Preservation corpus and adverse cases

Each row needs a Given/When/Then assertion, a declared owner, an execution receipt, and artifacts. A fixture's passing assertion is stronger than a generic green phase.

1. Given a saved signed-in profile, when the operator closes/reopens BAS, then supported sign-in state and tabs return without crossing into another profile.
2. Given passive recording, when the operator types with pauses, replaces/selects/deletes text, pastes, uses IME, and clears a field, then the selected workflow reproduces the final values.
3. Given equal selectors in two tabs/frames, when actions alternate between them and replay in a fresh browser context, then each action retains its logical target identity and order.
4. Given a redirect, SPA route, popup, shadow DOM, or service worker, when navigation completes, then capture resumes and the timeline reports any unsupported gap.
5. Given a slow/disconnected frame viewer, when inputs and capture continue, then compressed queues and active decoding stay bounded, fallback supplies current frames, and old tab/session work cannot paint after a switch or revive a disposed connection.
6. Given a persistence or disk failure, when recording/profile writes occur, then success is not reported as durable and existing data remains recoverable.
7. Given workflow timeout, cancellation, driver death, API restart or a retried start during a live instruction, then terminal state, last evidence, leases and cleanup have one consistent result, with no concurrent owner created by retry.
8. Given required screenshots/logs/network/video/trace, when supported collection or teardown completes or fails, then manifest completeness and reasons match the returned outcome and unresolved browser resources retain a recovery owner.
9. Given a desktop/Android target issued by its owner, when BAS attaches, acts and detaches, then renderer/isolation/recording identity is preserved and target lifecycle remains owner-controlled.
10. Given a bundled clean-machine install, when Node/Chromium are absent from developer PATH/cache, then the package runs from its declared inventory or reports a precise missing capability.
11. Given selected human history, when AI proposes a workflow, then original observations are retained, secrets are referenced safely, a typed candidate is validated, and repeat-effect authorization precedes qualification.
12. Given ordinary authorized human browsing, when a site presents an authentication or challenge step, then human control and profile continuity remain available and the state is reported honestly.
13. Given browser shortcuts, modified clicks, double-clicks, blur, drag/drop and horizontal scroll, when a person performs and derives those actions, then live input and supported replay preserve their meaning; unsupported actions produce explicit errors without replacement clicks.
14. Given a saved encrypted profile, when tab/history/storage saves overlap, a write fails, or its key/protected bytes become unavailable, then the prior committed identity and state remain recoverable and incomplete recovery is visible.
15. Given a declared session capacity and fresh/clean reuse policy, when starts overlap, reset fails, or the active page is not the first tab, then reservations stay bounded, retained pages are open, page maps are coherent and failed reset has an explicit recovery path. Test isolation must cover the declared origin/storage matrix.
16. Given active executions and a per-workflow keep_latest policy, when bounded retention repeats or applies a preview subset, then actual protected evidence survives and eligible older evidence is eventually removed.
17. Given loops, declared safe retries, repeated transport delivery and changed-payload conflicts, when workflow execution proceeds, then an independent fixture counter matches intended dynamic invocations; uncertain effects are reconciled before repetition and old leases cannot mutate a new owner.
18. Given a cancelled graph/linear execution or a thrown handler error, when termination completes, then available failure context and terminal step evidence remain queryable, required capture failures affect the verdict, and cleanup failures retain explicit ownership.
19. Given a released session, when the next owner requests a different profile, proxy, locale, viewport or clean state, then reuse rejects or satisfies the effective configuration; independent signed-in identity cannot disagree with the admission receipt. Compatible continuation, fresh mode and unreleased-owner controls remain correct.
20. Given two executions sharing a compatible browsing context, when evidence requirements or output destinations change, then each capture has a verifiable boundary and its own manifest; required evidence starts before ready, and no returned artifact mixes execution ownership.
21. Given released managed, desktop or Android sessions, when a caller requests another target or supplies missing/mismatched validation context, then every admission path validates the exact target and isolation lease and cannot silently substitute the retained renderer.
22. Given pending capture startup, replacement or resize, when stop, failure or a page switch overlaps it, then each generation disposes late resources, old cleanup cannot remove the current owner, and stale page frames cannot publish. Every retained handle/socket remains stoppable.
23. Given active CDP or polling capture, when quality/FPS/header settings change or the transport reconnects on a stable page, then effective settings and measured rates match receipts, unsupported controls are explicit, and the latest valid frame is delivered without another paint.
24. Given typed step outcomes and public assertion-validation flags, when a workflow is validated, executed and read through timeline/replay, then enforcement flags reach the server and structured outcome fields survive projection. Client parsing errors and debug strings cannot count as valid evidence.

The recorder's oracle must be independent of its own captured events. Use fixture-app state, a separate test driver, and sequence sentinels. Use direct engine/UI tests for failures that BAS cannot reliably test through itself. Test Genie still owns scenario suite runs.

## Execution shape, checkpoints, and remaining investigation

Recommended next step: approve the product target and measurement definitions, then use an adaptive mandate with bounded architecture decisions and behavior-preserving cuts. BAS already has an improve skill and setpoint program, but the present board does not cover the requested browser/product outcomes and its read failed here. It is not a sufficient finish line until repaired and extended through the owning authoring workflow.

Suggested sequencing expresses dependencies, not a newly created plan:

1. Resolve W0, reconcile relevant Swarm goals, preserve feature/platform matrix, and make metrics/runner coverage truthful.
2. Establish the independent fidelity corpus and controlled latency/memory baseline. Diagnose API memory and evaluate desktop rendering choices.
3. Repair recording/profile durability and semantics; introduce bounded admission/pooling and ordered input/frame policies.
4. Extract ownership boundaries one vertical slice at a time while running affected regressions. Keep prior workflow/profile formats readable through explicit versioned migrations.
5. Qualify evidence, agent workflows, desktop/mobile adapters, packaging, long sessions and UI polish against the accepted matrix.
6. Run release certification and two fresh adversarial reviews of behavior, maintainability, and product usability. Green implementation checks alone do not close known gaps.

Each checkpoint records: current issue IDs; source/build identity and dirty inputs; unchanged acceptance contract; before/after measurements with population; receipt references; migration/rollback state; remaining unknowns. Keep old snapshots. Update the issue register with newly discovered issues instead of creating parallel problem logs. Closure requires the row's evidence, not a percentage of files moved.

During ordinary implementation use focused tests and relevant Test Genie phases from docs/TESTING.md. For example, recording changes require driver/Go/UI contract regressions and scoped unit/workflow evidence; packaging changes require portability plus owner hardware/platform receipts. Admit one suite per scenario, then attach once using the returned test-genie runs wait command. Required platform evidence remains required even if local simulation is green.

No full implementation goal is ready to launch from this report alone: accepted target/bands, runtime-rendering choice, platform support rows, and current sensor qualification need explicit resolution. These are bounded next investigations, not reasons to repeat a broad code inventory.

Follow-up disposition: passive-input, event delivery, network correlation,
pagination, lifecycle, profile commits/recovery, replay semantics, admission,
reset/retry, frame lifecycle, close receipts, retention batching, invocation
identity, instruction admission, cancellation/failure evidence, profile/target
reuse compatibility, capture ownership and producer streaming lifecycle/settings
now have isolated reproductions. One real fresh-browser two-click counter control
passes; correlated viewer latency and passive record/replay are still unmeasured.
The deployed binary contains RF-021's leak mechanism.
A bounded
correctness/resource-lifecycle repair can be scoped independently of the future
rendering choice once implementation is authorized. For further investigation,
prioritize a matching-build heap/stack profile and controlled execution cohort,
then one real-browser record/replay fixture and input-to-paint instrument. The
real-browser authentication-storage/interruption matrix, physical filesystem
crash durability, native desktop key provisioning and rendering comparison remain
unexecuted. Reuse the saved probes instead of repeating broad source discovery.

### Proposed next investigation handoff (not launched)

~~~text
/goal A supplemental report in scenarios/browser-automation-studio/docs/internal/REFRACTOR_ASSESSMENT.md resolves the remaining matching-build API heap/stack attribution, real-browser input-to-paint and record/replay behavior, profile interruption/storage durability, and desktop rendering comparison, using the existing isolated reproductions instead of rediscovering them.

Read the assessment, its baseline JSON, ARCHITECTURE.md proposed target, and PROBLEMS.md BAS-RF register first. Keep product code, dependencies, and accepted targets unchanged. Use isolated fixtures and current lifecycle/owner tools. Label facts, hypotheses, recommendations and unknowns. Do not access real credentials or rerun effectful user workflows.

Proof: dated metrics with cohort/sample definitions, artifact IDs, and an explicit proposed architecture decision. Preserve mobile/desktop attach and evidence capability distinctions. Update this report and the existing issue register only; do not launch implementation or alter requirement status. Checkpoint when the bounded investigation ends with resolved, unresolved, and the smallest next decision.
~~~

### Reproduction and evidence locations

From the repository root:

~~~bash
python3 scenarios/browser-automation-studio/docs/internal/refactor_inventory.py > /tmp/bas-inventory-current.json
vrooli scenario status browser-automation-studio
browser-automation-studio observability sessions
browser-automation-studio observability metrics
program-runtime library run browser-automation-studio.setpoint-read --input 'window=100,evidence_sample=5'
browser-automation-studio measures pass-rate --window last_7d
browser-automation-studio measures p95-duration --window last_7d
test-genie runs show 20260909-201227-1e8ed2e9 --scenario browser-automation-studio --json
test-genie runs artifacts 20260909-201227-1e8ed2e9 --scenario browser-automation-studio --json
~~~

The setpoint program's window is an execution sample count (10–100), not a day count; the owner measure uses last_7d. This investigation first supplied invalid window values, corrected them from the contract, then observed the budget failure on a valid request. Those operator mistakes are not attributed to BAS.

The baseline JSON preserves inventory, live reads, capture references, process observations, and historical artifact IDs/hashes. Historical run logs remain owner-managed; retrieve them through the run's artifact catalog. Capture artifacts follow BAS retention, so the IDs are locators rather than an indefinite retention guarantee. No screenshots, cookies, site contents, or authentication state were copied into the committed baseline.

Reviewed external API documentation on 2026-09-21 is linked beside the relevant architectural claims. It is context for proposed choices, not proof that the installed runtime implements every current upstream capability.

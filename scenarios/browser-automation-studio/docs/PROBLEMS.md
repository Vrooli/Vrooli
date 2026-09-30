# BAS defect register

This is the sole BAS-RF register. It lists open items only.
To update: edit an entry in place. Add a new BAS-RF id only for a distinct defect; the next free id is BAS-RF-151. When an item closes with evidence, delete its row and note the closure in the epoch slice log.
The full pre-2026-09-29 register, including resolved entries and evidence, is archived in `~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/bas-rehabilitation/retired-docs-2026-09-29.tar.gz` (member `docs/PROBLEMS.md`). Retrieve it with `tar -xzOf <archive> docs/PROBLEMS.md`.
Qualification status comes from the 24 preservation journeys in `requirements/08-rehabilitation` and the daily qualification program, not from this file.

## Session, profiles and lifecycle

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-003 | Profile persistence/lifecycle | Profile commit fault handling is repaired. Behavior under abrupt interruption and with shared-profile ownership is unqualified. | An abrupt-interruption case and a shared-profile case preserve the prior saved state. |
| BAS-RF-005 | Session coordinator | Admission race and retry spacing are repaired. The effect of the 500 ms retry policy has not been isolated. Historical 3.8–29.8 s outliers and browser cancellation timing are unattributed. | Retry timestamps from a comparable release trial quantify the policy, and sustained-load recovery is measured. |
| BAS-RF-006 | Browser pool | Retry and shutdown races are repaired. Pool retention and capacity for distinct keys under sustained load are unmeasured. | A distinct-key soak plateaus, and all children close under concurrent failure, retry and shutdown. |
| BAS-RF-011 | Profile runtime/cancellation | Linux checkpoint and graceful-restart durability pass. Unproven: abrupt process death, repeatable checkpoint timing, multi-binding degradation, shared-profile conflicts, non-Linux, native AI-navigation abort. | Each case has a managed/native result. |
| BAS-RF-012 | BAS + scenario-to-desktop | Bundle/platform inventory is unqualified. Profile encryption-key provisioning, native credential recovery and off-host escrow are also unqualified. | Native install/update/rollback/cleanup keeps sign-in and key recovery. |
| BAS-RF-025 | Profile repository | Atomic snapshot publication is repaired. Durability after physical crash or power loss on native filesystems is unproven. | Power-loss durability is shown on supported targets. |
| BAS-RF-026 | Profile aggregate/concurrency | Field updates are serialized, and ambiguous bindings fail closed. There is still no policy for snapshot ownership, forks or conflicts on a shared writable profile, and no native locks. | That policy and supported native locks are implemented and tested. |
| BAS-RF-027 | Profile recovery/API | Recovery fails visibly and preserves identity. The recovery UX and protected-state behavior on native browser targets are unqualified. | Recovery UX works on supported native/browser targets. |
| BAS-RF-031, BAS-RF-032 | Session lease/reset | Start-retry exclusivity, reset retention and input identity are repaired in source. Managed/native and cross-origin reset are unverified; failed-reset recovery lacks fenced cancellation/expiry proof. | Managed reset across origins and failed-reset recovery pass on a native browser. |
| BAS-RF-042 | Execution profile → session reuse | Profile-versioned pooling is repaired. Reuse across multiple browsers and platforms is unqualified. | Reuse passes on supported browsers/platforms. |
| BAS-RF-043, BAS-RF-044 | Evidence/session finalization; target admission | Capture/target reuse boundaries are repaired in source (isolated probes only); managed/platform verification is pending. | Managed build confirms fresh contexts and correct desktop/Android admission. |
| BAS-RF-052 | Profile persistence | Path-traversal validation is repaired. Native OS behavior, hostile symlink-root replacement and filesystem permission boundaries are unqualified. | Those boundaries are tested on supported OSes. |
| BAS-RF-055, BAS-RF-125, BAS-RF-126 | Execution recovery/cancellation | Single-process startup recovery, StopExecution acknowledgement and the startup gate are repaired. Ownership with concurrent or duplicate API processes, and lease release after abrupt death, are unqualified. | An adverse concurrent-start or abrupt-death case keeps a single owner for active rows. |
| BAS-RF-124 | Driver session lifecycle | Reset/close now join in-flight actions. External-target cancellation and managed lease release are unqualified. | An external-target close and a managed release complete with an uncertain outcome retained. |
| BAS-RF-138 | J07 restart / lifecycle setup | On the same build, one J07 restart passed (150 ms preparation) and one failed (1,676 ms preparation, input stop over the 1,000 ms band). Recovery is 8.7–9.5 s against a 10 s band. The transient pre-stop cost is unexplained. | Repeated managed restarts stay inside both bands without widening them. |
| BAS-RF-148 | Navigation/shutdown; api-core server | Shutdown cleanup after the drain deadline and scheduler cancellation are repaired in source. No managed restart has yet shown, together: an outstanding 300 s status wait is cancelled, owner cleanup runs, and browser/driver teardown completes. | A managed restart with an outstanding wait shows all three. |
| BAS-RF-149 | AI navigation lifecycle/conversation | `observation_unavailable` state, stop admission and recovery tests pass in source. Managed proof is missing that, after observation loss, recovered status, new-command admission and conversation history agree with no duplicate effect or orphaned message. The same applies to recording close after driver loss. | A managed observation-loss/recovery case shows that agreement. |

## Recording and input

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-002 | Recording service/journal | Journal ordering, crash/retry and semantics pass on managed Chromium. Supported-event coverage on native OS and other browsers is unqualified. | The event corpus passes on native targets. |
| BAS-RF-004 | Recorder/workflow derivation (J02) | Final-input merge and replay pass in managed Chromium, but paste and composition are fixture-synthesized. Native OS IME and broad tab/frame alternation are untested. | Native IME and tab/frame alternation cases pass. |
| BAS-RF-017 | Recording/evidence privacy | New passive values, managed AnalyzeElements screenshots and timeline/replay responses are redacted. Open: legacy secrets at rest, recovered status/history, exports/derived attachments, credential-use flow; the 2026-09-29 generic-detail redaction is source-only. | Synthetic secrets absent from every storage/export/screenshot/AI/recovered path; legacy data handled safely. |
| BAS-RF-020 | Session input coordinator | Input ordering, reset/close join and modifier recovery are repaired. Unqualified: full UI reload, server-side cancellation during long browser waits, native input/button semantics, and retries after receipt eviction. | Each of these cases passes. |
| BAS-RF-022 | Recording delivery | Callback receipts and pending retention are repaired. Production incidence has not been checked through a maintained full-script owner. Overflow recovery and replay after receipt eviction are unqualified. | A full-script owner covers these cases. |
| BAS-RF-028 | Recording semantic conversion | Typed derivation preserves modifiers, click count, scroll axes and drag. Complete browser record→replay over a versioned action corpus is unqualified. | The versioned corpus replays with its meaning intact, and unsupported actions fail explicitly. |
| BAS-RF-029 | Browser input/UI | Chords, pointer modifiers and IME composition pass in Chromium. Platform Command/clipboard behavior and native IME are open. | A native OS shortcut/clipboard/IME corpus passes. |
| BAS-RF-030 | Timeline → workflow derivation | Main→popup→main replay passes. Nested frame/tab alternation and popup close/reopen timing are unverified. | Frame and popup close/reopen edges replay in a fresh context. |
| BAS-RF-038 | Operation ownership/protocol | Open: same-page navigation epochs, callback retry/generation ordering, external callback authority, page/journal attribution after concurrent handoff, remaining raw interactive mutations, activation-generation protocol, managed human handoff. | Every mutation validates lease/generation, and a managed handoff/interruption case passes. |
| BAS-RF-039 | Invocation/retry/idempotency | Duplicate/late steps and receipts are fenced. Reconciliation of restarts and effects is unqualified. | Interruption and reconciliation preserve exactly one effect. |
| BAS-RF-056 | Recording-to-proto conversion | Full-value replacement replay is repaired. Typed `submit` behavior is unverified. | A saved-workflow case covers `submit`. |
| BAS-RF-101 | UI tab bar | Keyboard focus and activation are repaired. Full ARIA panel linkage, assistive-technology use and mobile/OS behavior are unqualified. | An accessibility audit and AT/mobile checks pass. |

## Execution, resume and cancellation

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-021 | Execution/event lifecycle | Wrapper cleanup and the accepted-event drain are repaired. Contextless downstream blocking, heap attribution and live workflow soak are unqualified. | A soak with stack/heap attribution shows bounded queues and resources. |
| BAS-RF-023 | Execution writer | Terminal cleanup is repaired. Active-run accumulation bounds and late-writer fencing are unqualified. The retained-heap check has not been repeated on a current build. | These bounds are shown on a current build. |
| BAS-RF-036 | Session finalization/evidence | Close/flush ownership is repaired in Go and in the driver. Still open: best-effort policy for performance/accessibility capture, durability of pending callbacks, hung operations, and recovery after process restart. | These fault cases are covered. |
| BAS-RF-041, BAS-RF-065 | Instruction failure evidence; console capture | Diagnostics now survive a handler throw, and native console capture works. Worker, OOPIF, anti-detection and capture-hang coverage are unqualified. | These cases retain console evidence. |
| BAS-RF-054 | Execution query | Filtered paging is repaired. Separate offset requests share no snapshot, so paging while history changes can skip or repeat rows. | Snapshot isolation across pages is implemented or explicitly declined. |
| BAS-RF-107, BAS-RF-109, BAS-RF-110 | Checkpoint/resume | Linear resume, durable store checkpoints and typed evaluate `store_result` are repaired. Resume refuses branched/cyclic/loop/subflow workflows because full cursor recovery is not implemented. External-effect/crash atomicity is unqualified. | Rich control-flow resume and crash atomicity are designed and tested. |
| BAS-RF-117 | Workflow status authority | Status-write gating of effects and notifications is repaired. Eventual reconciliation after a DB outage and crash recovery remain open. | Both cases are covered. |
| BAS-RF-123 | Checkpoint/resume | Resume refuses an uncertain effect. Reconciliation of external effects and live-browser cancellation/recovery timing are open. | A reconciliation path exists and managed timing is measured. |

## Capture, evidence and artifacts

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-013 | Capture API/target owners | DOM, DOM_TREE and browser VIDEO artifacts are repaired in source only (not on a managed build). Device video is unavailable. CAPTURE_TYPE_PERFORMANCE was reported unavailable (2026-05-18) and no later fix is recorded. | Managed targets produce each type, or report it explicitly unavailable, in a published capability matrix. |
| BAS-RF-016 | Capture performance | Duplicate PNGs and DPR are fixed. Capture wall p95 rose after cycle 095 (about 610 → 681–757 ms), and that relative regression was never attributed. Later cohorts are about 575–660 ms, under the 2 s band. | A paired comparison attributes the increase or shows no regression. |
| BAS-RF-049 | Evidence/proto projection | Structured outcomes round-trip. There is no policy for historical debug-string outcomes during replay/consumption. | Historical projection has explicit handling. |
| BAS-RF-119, BAS-RF-121 | Execution writer/storage | Decode admission and GOMEMLIMIT reduce peak memory, but neither is a hard cap. Unqualified: large JPEG, Windows/macOS memory, storage backpressure, MinIO runtime (including the `.jpg` naming), real caller cancellation, and long soak. | These scopes are measured. |

## Frames, streaming and viewer

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-007, BAS-RF-034 | Frame transport/viewer | The 30 FPS motion and slow-reader bounds pass. Cycle 080 saw a device stream stall under concurrent workload (20 capture timeouts) whose cause is unproven. Remote latency was measured only with CDP emulation. The driver/transport performance matrix is unqualified. | The load stall is explained or fixed, and a real remote cohort is measured. |
| BAS-RF-008 | Frame transport/auth | The direct listener is retired. The API frame-subscription authentication/origin matrix is unqualified. | Unauthenticated, wrong-session and wrong-origin clients are rejected, and scoped clients are admitted. |
| BAS-RF-046 | CDP capture lifecycle | Capture generations are fenced. Cleanup of a worst-case unresponsive protocol session is unqualified. | A hung-protocol case releases its resources. |
| BAS-RF-070 | Rebrowser frame realm | Frame realm discovery is repaired. Cross-realm binding/global cleanup and broad stealth behavior are unqualified. | Cleanup is verified across realms. |

## Platform, runtime and security

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-010 | API/runtime performance | Linux idle and workload memory are within budget. Windows private memory, controlled long soak, fixture-browser memory and historical multi-GiB heap attribution are unmeasured. | These are measured with PSS, heap and swap separated. |
| BAS-RF-018 | Runtime/provider | Provider comments overpromise detection compatibility, and injection changes page behavior. Compatibility is unknown. | Versioned site/fixture results, challenge recovery and bounded claims. |
| BAS-RF-068 | AI suggestions | The response contract is enforced. The wider AI quality/grounding/replay corpus is unqualified. | A quality corpus exists. |
| BAS-RF-071 | Security Health | Errors are at 0, but 357 warnings are untriaged, and production security posture is unqualified. | Warnings are triaged to fixed or accepted with reasons. |

## Tooling, docs and structural debt

| ID | Area/owner | Open problem | Close when |
| --- | --- | --- | --- |
| BAS-RF-001 | BAS product/docs | The browser-first target (OT-P0-005, 24 journeys) is explicit. Formal acceptance of the platform/preservation matrix and scoping of stale "refactor complete" claims in other docs are not recorded. | The PRD/requirement mapping is accepted, and historical claims are scoped everywhere. |
| BAS-RF-014 | UI unit coverage | Merged UI coverage is about 31% against the unchanged 85% floor (79 LOW_COVERAGE files). | Coverage meets the floor through behavior tests. |
| BAS-RF-015 | Module structure | Local hotspots are reduced (for example Capture 69→6), but domain-wide complexity, duplication, long files and coupling remain high; the tidiness ratchet was re-baselined on 2026-09-29 (DECISIONS.md) and only ratchets down from there. The E01 navigation/cleanup owners still carry command/runtime cohesion hotspots. | Domain-wide findings fall measurably below the 2026-09-29 baseline with behavior preserved. |
| BAS-RF-048 | cli-core proto binding + BAS CLI | The documented require-assertion flag fails before the RPC, and explicit `true` is rejected. Reported as knw-1790042809480136792. | The shared binding encodes boolean presence, and positive/negative CLI tests pass. |
| BAS-RF-050 | BAS docs | Docs validation still reports 4 errors (placement, plans metadata, PRD external link) plus stale command/code/doc references. | Actual stale references are fixed without suppressing findings. |
| BAS-RF-057 | Driver coverage instrumentation | Babel coverage injects Node counters into `page.evaluate` callbacks (ReferenceError in the renderer). | A compatible coverage producer passes browser fixtures with floors retained. |
| BAS-RF-059 | Driver coverage performance | V8 coverage takes about 350 s against a 300 s owner deadline, so native verdicts are lost. | Coverage cost fits a measured valid deadline with floors retained. |
| BAS-RF-064 | Tidiness Manager analyzers | JS duplication ignores the supplied inventory, scans only `ui/src`, and treats invalid JSON as empty. There is no TS/JS AST complexity metric. | Strict parsing, full inventory, and unavailable analyzers never counted as clean. |
| BAS-RF-127 | Workflow catalog/file store | Delete path resolution is repaired. Storage-fault ordering across source, version and catalog deletion is untested. | The fault-ordering regression passes. |
| WH 2026-07-27 | Workflow Health artifacts | Failed Workflow Health runs keep only `latest.json` and `timeline.json`, with no screenshot, console log or network trace (see run 2d7cec0e…). | Failed-run artifacts include browser evidence. |
| AGENT-REUSE 2026-09-04 | Requirements/provider evidence | The AGENT-REUSE requirement stays in_progress. Provider discrepancies are in knw-1788561851141947801 and knw-1788561878575592107. | Provider evidence is earned without lowering gates. |
| Storage audit 2026-09-05 | Persistence domains | Storage validation still reports direct-writer, permission-proof, cross-domain FK and uncovered-database findings (audit archived with this register). | The findings are fixed or classified. |

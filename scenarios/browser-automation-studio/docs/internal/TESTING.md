# Browser rehabilitation qualification

Status: launch contract, 2026-09-22 UTC. This describes required evidence, not
completed implementation. The operator authorized preparation; assigning the
linked goal authorizes its implementation. The canonical bands and journey IDs
are in [REFRACTOR_CONTRACT.json](REFRACTOR_CONTRACT.json). The earlier assessment
remains dated evidence. Its proposed values are adopted as this engineering
target; they are not claims about current performance or public release promises.

## Work model and stakes

This is a **file-based continuous improvement goal**, explicitly authorized by
the operator. Do not create, resume or use a Plan Manager plan, phased checklist,
child plan or Swarm plan-backed item for this engagement. Its target is durable;
its next intervention comes from current evidence. This instruction overrides
the plan-backed templates in goal and campaign skills for this engagement only.

Read `prompt-manager skill read browser-automation-studio-improve` for the
scenario's rehabilitation posture and `prompt-manager skill read
improvement-do-and-dont` for anti-gaming. Use `scientific-debugging` when a cause
is uncertain. Goal/campaign guidance remains useful for repair and evidence
discipline; its plan requirement and finite convergence stop rule do not apply
to this operator-selected continuous goal.

BAS is infrastructure for Vrooli's browser agents, UI development, end-to-end
tests, screenshots, recordings and debugging evidence. The operator reports
that its unreliability is a major bottleneck to progress, user adoption and the
project's intended first monetized scenario. The operator also reports that
their relationship is at risk if the project does not become production ready
and monetizable soon. These are user-reported stakes, not independently measured
claims. Give them the urgency of durable engineering and honest evidence.

The objective is a substantially more reliable, fast, professional and
maintainable product with **less technical debt and complexity**. Fixing a bug
by leaving another workaround, ownership ambiguity or parallel implementation
is unfinished work. Measure substantive simplification along with correctness.

### Authority and boundaries

Assignment of [REFRACTOR_GOAL.md](REFRACTOR_GOAL.md) grants autonomous engineering
within the contract's `work_model.acceptance_allow`. Necessary BAS-facing repairs
at shared owners are included: record the reason, owner, affected paths and checks
in the progress file before extending the boundary. Host repair belongs in the
control plane. Follow normal lifecycle, dependency and Test Genie owners.

No per-issue approval or human review is needed. Make ordinary product-preserving
engineering choices from the documented target and evidence. Preserve other
agents' changes and saved user data. Commercial publishing, billing changes,
new paid access, credentials or real-account effects still need their actual
authority; defer only an unauthorized effect and continue useful work. Do not
ask the operator to unblock the loop or solve its validation prerequisites.

### Durable tracking and resume protocol

All paths below are within this scenario. There is no external plan/log dependency.

| File | Single responsibility |
| --- | --- |
| `docs/internal/REFRACTOR_CONTRACT.json` | Outcomes, unchanged preservation obligations, bands, scope and continuation policy |
| `docs/concepts/ARCHITECTURE.md` | Current and intended owners, interfaces and product design |
| `docs/PROBLEMS.md` | Sole BAS-RF issue register, including new adversarial findings and dated resolutions |
| `docs/internal/REFRACTOR_PROGRESS.md` | Current checkpoint plus append-only experiment/review history and measured deltas |
| `docs/internal/OPERATOR_FEEDBACK.md` | Verbatim feedback captured before acting, with discrete statuses and resolution receipts |
| `docs/internal/DECISIONS.md` | Durable architecture decisions and tradeoffs |
| `docs/internal/REFRACTOR_*_2026-09-*.json` | Dated baseline/assessment observations; append an amendment rather than rewrite history |
| `docs/internal/evidence/rehabilitation/` | Small durable preparation/experiment evidence; reference owner artifacts for large captures |

On every start or resume, read the goal, contract, this protocol, feedback and
latest progress; then inspect the relevant issue and source. Recover pending
owner operation IDs before admitting another run. An unchecked item is unknown,
not completed merely because the previous agent said it was.

Choose a real high-impact issue or a falsifiable investigation. Prioritize data
loss, ownership/security defects and broken core journeys, then recurring
friction, performance and structural simplification using measured impact.
Document the hypothesis and discriminating check; repair the whole ownership
boundary, validate, critique the result and select the next concern. This is a
repeatable loop, not an ordered implementation plan or fixed list of phases.

After each coherent change or experiment and before compaction/interruption,
update the current checkpoint and append a record with: issue/outcome IDs;
source/build identity and changed paths; hypothesis and result; commands and
receipt IDs; before/after performance, complexity, duplication and coupling;
converted callers/deleted paths; rejected ideas; unverified limitations and
attempted remedies; pending operations; next useful action. Give limitations
explicit recheck triggers so unchanged failures are not retried indefinitely.
Keep full proof with its producer; link it instead of duplicating issue tables.

Append every new operator instruction verbatim to the feedback ledger before
acting, including requests that alter the target or scope. Resolve each with
specific evidence. Keep implementing while any actionable feedback or defect
remains. At interruption, report changed, verified, remaining and unverified;
the checkpoint supports resumption and is not a completion claim. No external
restart service is configured by this text or by preparing the goal.

## Non-blocking validation policy

The operator explicitly clarified that **no unavailable validation or release
qualification may block this engagement**, including a target that initially
looked runnable and later becomes unavailable. This instruction governs goal
continuation and supersedes stricter stop clauses in the initial investigation
and skill templates. It does not turn an unknown result into a pass.

Attempt reasonable diagnosis and authorized remediation. If the check still
cannot run properly, record its outcome ID, exact command/operation, failure or
missing capability, attempted remedy with evidence, remaining limitation and next
validation action in REFRACTOR_PROGRESS.md, linked to the relevant BAS-RF issue
when one exists. Keep the sensor unknown.
Continue every useful independent implementation, simplification or verification
task. Do not stop with `blocked`, wait indefinitely, repeatedly retry unchanged
failures, or ask the operator to solve an unavailable validation prerequisite.

A real failing product assertion is actionable repair work, not an unavailable
runner. Fix it when the active authority permits. If a specific action needs
credentials, external access or effects outside the grant, defer that action
without bypassing the boundary, and continue. When the known repair list is
empty, start a fresh adversarial investigation of behavior, architecture,
performance and UX. Release targets that could not be
validated stay explicitly unverified, including native OS/device targets and
long-running checks whose environment cannot support a valid measurement.

Two fresh clean reviews are a dated confidence observation, never a stop signal.
Documented unavailable validation does not prevent an improvement from being
delivered. Full release certification may be claimed only for outcomes actually
supported by evidence; missing evidence remains explicitly unverified.

## Outcome evidence inventory

Every passing contract row must have an owner-produced receipt. Unavailable
validation follows the operator-approved non-blocking policy above. The initial
rehabilitation board deliberately reports `pending_telemetry`: existing mixed
execution statistics do not measure these outcomes. Building the qualification
fixtures, receipt producers, governed reads and joins is authorized implementation
work. A successful board invocation does not satisfy an unavailable row.

Run from the repository root:

```bash
python3 scenarios/browser-automation-studio/docs/internal/refactor_contract.py
program-runtime library run browser-automation-studio.setpoint-read --input profile=rehabilitation
python3 scenarios/browser-automation-studio/docs/internal/refactor_inventory.py --include-untracked
```

The retained module reproductions also have a fail-closed runner:

```bash
python3 scenarios/browser-automation-studio/docs/internal/refactor_regressions.py
python3 scenarios/browser-automation-studio/docs/internal/refactor_regressions.py --case input
```

Exit 0 requires every selected expected behavior. Exit 1 means observed behavioral
failures; exit 2 means unusable producer evidence. Their synthetic I/O scope does
not qualify full-browser or native-platform journeys. Convert these probes into
maintained owner tests during implementation; preserve the expected semantics.

The maintained capture workload is `api/cmd/capture-cohort`, backed by
`api/internal/capturequalification`. From `api/`, run it against managed services:

```bash
go run ./cmd/capture-cohort --api-url "$BAS_API_URL" --driver-url "$BAS_DRIVER_URL" --output "$BAS_COHORT_OUTPUT"
```

The output directory must not exist. The workload retains 100 first attempts and
one declared warmup, including failures and unattempted positions after
cancellation. It checks decoded PNG pixels, a unique paint marker per attempt,
viewport/DPR, computed snapshot, stored artifacts and an independent fixture
observation. Raw CLI output, errors, artifact hashes, producer/fixture/contract
identities and before/after managed build identities remain in the receipt.
It neither restarts services nor cleans retained evidence. Its controlled oracle
checks run with `go test ./internal/capturequalification ./cmd/capture-cohort`.
Standalone observations do not certify a rehabilitation row. Performance Health
owns invocation, applicability, retention and the governed reading through
`sweep workload-run browser-automation-studio capture --json` and `sweep workload-get`;
Test Genie retains the performance gate result. The dated090 evidence records
the first qualified local capture reading; fresh candidates need fresh receipts.

The first command validates preparation and returns no product verdict. The
second reads required outcomes. The third measures source size, not complexity.
Use only the file responsibilities above. Test Genie and platform owners retain
their receipts; the progress file links those receipts and owns the checkpoint.
Do not create a second issue register or a replacement tracking application.

### Producer and oracle contract

Create a BAS-owned local fixture with an independent action log, monotonically
numbered effects, input values, storage identity, tab/frame identity and explicit
paint sentinels. Use temporary routed test storage and synthetic authentication.
The fixture must not derive expected results from recorder output. Run direct
driver/API/UI tests when the defect would prevent BAS from testing itself.
Promote the existing isolated probes into maintained regressions. A historical
probe or a stubbed I/O reproduction alone does not qualify the full journey.

| Journey | Independent oracle and required adverse cases | Producer |
| --- | --- | --- |
| J01 | Fixture identity endpoint plus cookies/localStorage/IndexedDB and tab list after close/reopen; second profile remains different | BAS profile/driver and workflow suites |
| J02 | Fixture field values and independent input log after pauses, replacement, deletion, paste, IME and clear; replay into a fresh context | Driver, Go derivation, UI and workflow suites |
| J03 | Separate tab/frame counters with identical selectors; recorded/replayed operations affect only the intended counter | Driver and workflow suites |
| J04 | Fixture redirects, SPA, popup, shadow DOM and service worker; sequence sentinels reveal any capture gap | Driver and workflow suites |
| J05 | Correlated input/paint IDs and queue/decode gauges under slow or disconnected viewers, resize and tab switch; disposed generations stay disposed | Driver/UI plus performance workload |
| J06 | Saved-byte hash and fixture identity after disk/write failure; previous commit survives and success is not falsely acknowledged | Profile and recording fault tests |
| J07 | External effect counter, terminal receipt and live resource count during timeout/cancel/death/restart/retried start | Executor/session/driver fault tests |
| J08 | Artifact hashes, ownership and manifest completeness compared with deliberately injected capture and teardown failures | Evidence pipeline and workflow suites |
| J09 | Target-owner identity and before/after process/device state; exact external renderer receives the action and survives detach | Desktop/Android/device owner qualification |
| J10 | Clean host with developer PATH/cache removed; bundle inventory supplies all declared runtime dependencies | Native desktop owner qualification |
| J11 | Original trace digest, typed candidate validation, fixture postconditions and explicit repeat-effect authority; secret values remain protected | AI replay corpus and bounded authorized live smoke |
| J12 | Fixture login/challenge handoff and profile continuity; external supported-site checks are separately attributed | Profile/UI/workflow suites |
| J13 | Fixture input log for shortcuts, modifiers, double-click, blur, drag/drop and horizontal scroll; unsupported operations fail explicitly | Driver/UI/derivation suites |
| J14 | Committed encrypted bytes and identity across overlapping writes, interrupted commit and unavailable key | Profile transaction/fault suites |
| J15 | Independent active-context count, page identities and storage matrix under concurrent starts and reset failure | Session/driver fault suites |
| J16 | Protected active artifact hashes plus eventual deletion of eligible data across repeated bounded sweeps and preview subsets | Retention tests with routed storage |
| J17 | Independent dynamic invocation counter through loops, safe retries, duplicate transport and changed payloads; stale lease cannot mutate | Executor/driver contract and workflow suites |
| J18 | Queryable last step/failure evidence and terminal state after graph/linear cancellation and thrown handler errors | Executor/evidence fault suites |
| J19 | Identity endpoint and effective proxy/locale/viewport/storage under compatible, incompatible, fresh and unreleased reuse | Session/profile/driver suites |
| J20 | Execution-specific artifact sentinels across reused contexts with changed evidence policy/output; required capture starts before ready | Capture/evidence suites |
| J21 | Owner-issued target/isolation receipts on every fresh/reused admission; wrong or missing validation cannot substitute renderer | BAS attach tests plus platform owners |
| J22 | Handle/socket counts and generation sentinels during overlapping capture start/stop/replace/resize/page switch | Stream lifecycle fault tests |
| J23 | Measured FPS, quality, headers and current frame after reconnect on a stable page; unsupported controls explicit | Stream/UI performance suites |
| J24 | Public CLI assertion enforcement plus primitive/map/struct/pointer typed outcome round-trips including failure/attempt identity | CLI-core/API/driver public contract tests |

Each journey's full Given/When/Then statement, owning roots and phase mapping is
in the contract. All 24 requirements start `planned`, with empty validation lists
where a complete behavioral test does not exist. Add exact real test references
and `[REQ:BAS-RH-Jxx]` tags as qualification is implemented. Do not cite this
protocol as a passing test or claim traceability maturity from its existence.

## Measurement validity

Retain every receipt field named by the contract. Hash the contract, fixture,
relevant source/configuration and build inputs. Shared-tree observations remain
useful; record relevant concurrent changes and do not claim exact execution of
a snapshot merely because a digest exists. Release evidence needs applicability
to the final candidate. Never overwrite the original baseline.

Use monotonic input-to-affected-paint correlation across clock domains. Network
send time, decode time, server readiness and website loading are separate metrics.
Separate warm/cold, local/remote, one/five/ten sessions, capture policies and OS
cohorts. Publish denominators, first attempts, retries, quantiles and confidence
intervals. No discarded failures, arbitrary idle sleeps, smaller workloads or
changed screenshot fidelity to improve a result. Apply the contract's 5% relative
regression rule only when repeated comparable trials distinguish it from noise.

Long soaks and native matrices are scheduled once after their affected paths
stabilize. Focused regressions and relevant Test Genie phases govern iterations:

```bash
vrooli scenario test browser-automation-studio --phases unit,workflow
vrooli scenario test browser-automation-studio --phases tidiness
vrooli scenario test browser-automation-studio --phases programs
```

Select actual affected phases from `docs/TESTING.md`. Attach once to the returned
run ID with `test-genie runs wait --json browser-automation-studio <run-id>`;
do not poll or re-admit a pending run.
Full release certification additionally owes the declared performance, portability,
UI, security, business and integration evidence (the registered phases are `ui-health`, `security`, `business` and `workflow`). A green generic performance
phase does not substitute for correlated browser measurements.

### Driver coverage

The Node driver is a required product surface. Its current Jest suites and
`playwright-driver/jest.config.js` are real; the current unit policy lacks a
required driver role. The instrument work must add the role through the unit
provider's supported adapter and verify that a deliberately failing driver
sentinel fails the Test Genie unit verdict. Preserve all existing coverage
floors. Until that routing is qualified, run the driver directly as a focused
package check and retain its result separately; generic unit success cannot
claim driver coverage. Dependency installation still uses Scenario Dependency
Analyzer.

## Simplicity and replacement completion

Eight policy owners are enumerated in the contract. The architecture assigns
their responsibilities; these are modules within the current Go/Node/React
deployment. Do not add a service or generic framework merely to express them.

Each completed change records, in REFRACTOR_PROGRESS.md: affected behavior, authoritative
owner before/after, caller paths converted, obsolete code deleted, temporary
adapters removed, and comparable domain-wide complexity/duplication/coupling
readings. Include newly created and untracked source; report tests and generated
code separately. Moving code across a directory or into a shared package cannot
make it disappear from the comparison.

Keep the existing `.vrooli/testing.json` tidiness budgets. They are an initial
floor, not the definition of architectural success. Remove duplicate semantic
policies and ambiguous resource ownership. Reject new dependency cycles and
review touched functions above cyclomatic complexity 15 and modules above 500
lines for cohesion. An irreducible algorithm can carry a specific evidence-backed
cohesion rationale; a wrapper extraction is not a complexity reduction. Fix
measurement defects with discriminating regression tests, never suppressions.

The existing ratchet is a minimum floor. Barely passing it does not demonstrate
the requested rehabilitation. Show material net improvement in the worst
ownership hotspots and comparable domain-wide complexity, duplication and
coupling. Explain which policies, branches, dependencies and resource owners
were simplified or eliminated. Track the cumulative delta from the retained
baseline as well as each intervention. A lower file count or line count alone
does not prove lower maintenance cost. Include affected shared packages so debt
cannot be exported outside the measurement boundary.

Production quality includes coherent public contracts, actionable errors,
bounded resource ownership, reliable teardown/recovery, protected identity and
evidence, polished and accessible browser interactions, meaningful regression
coverage, accurate docs and deployable runtime dependencies. Investigate these
properties in working journeys; do not award readiness from static checklists.
Over-engineering, generic frameworks without demonstrated need, unexplained
indirection, dead code, compatibility shims and permanent migration paths are
findings to remove. Preserve essential compatibility behavior and data through
the supported contract and verified conversion, without retaining obsolete
implementations. Make the final maintenance story simpler for the next engineer.

A slice is incomplete while its old path still has callers, temporary flags or
an unspecified removal condition. Essential correctness code and meaningful tests
may increase line count; record why and the complexity they replace. Preserve
saved workflows/profiles. If format conversion is necessary, use a bounded,
verified out-of-tree conversion with rollback data; remove transition machinery
from the supported runtime. Do not erase user data to achieve a cleaner tree.

## Platform and rendering decisions

Keep the existing bounded CDP/JPEG web viewer as the initial rendering route.
Its replacement is not a prerequisite for correctness repairs. A rendering
experiment may compare native Electron or video transport only after correlated
profiling identifies the constraint, at equal fidelity and workload. Record the
decision and preserve remote-web capability. A native implementation change
inside the accepted contract is an engineering choice; it must not silently
change the supported platform matrix.

Required release rows are Linux x64, Windows x64, macOS x64 and macOS arm64.
Android WebView and external Electron remain attach capabilities owned by their
respective scenarios. Native receipts prove install, launch, profile recovery,
capture, recording/replay, update/rollback and cleanup. Compilation, Wine,
simulations and local adapter tests do not satisfy missing native receipts.
An owner outage does not prove a device or runner is absent. Discover and repair
authorized access paths; retain the exact missing access if external authority
is actually needed. Continue independent development while it is unresolved.

Preserve instrumented browser compatibility and human challenge handoff. Do not
promise that arbitrary websites cannot identify instrumentation. Retain governed
AI/provider boundaries and credential protections. Use local fixtures and
already-authorized providers; paid spend or real-account effects require their
actual authority. Commercial launch, billing changes and publication are outside
this rehabilitation assignment.

## Continuous adversarial review

Material findings include a violated journey/band, concealed or falsely reported evidence,
ambiguous ownership, unnecessary abstraction, remaining transition code, data
loss, security/privacy failure, or a usability defect that frustrates the declared
browser tasks. Cosmetic preferences outside the target are not new scope.

A green suite or empty known issue list triggers fresh adversarial investigation.
You are the reviewer responsible for detecting the remaining defects; do not
reserve cleanup, UX polish or verification for a future human or delivery stage.
Try to disprove correctness, resilience, evidence fidelity, simplicity and
production readiness. Exercise new cases, not repeated summaries of old checks.

Rotate among data durability and interruption; concurrent sessions and ownership;
fault injection and evidence truthfulness; latency and resource pressure;
keyboard/accessibility and ordinary browser tasks; fresh installs and platform
boundaries; call graphs, duplicate policy and residual compatibility paths.
Record the question, candidate identity, cases, proof, findings and next review
angle in REFRACTOR_PROGRESS.md. Add genuine new issues to BAS-RF, fix them and
repeat. Record clean reviews honestly; do not invent defects to keep busy.

Two consecutive materially different clean passes qualify only a dated review
observation on that candidate. Continue with another under-examined surface,
broader valid workload or maintainability challenge. Do not declare the overall
goal complete because all rows pass, existing issues close or reviews are clean.
Do not churn already-simple code, invent product scope, repeat unchanged tests
or generate abstractions to manufacture activity. When no justified code change
exists, deepen useful investigation and measurement without needless edits.

Continue without a human approval loop until the operator stops/redirects the
work or the runtime enforces an interruption. Preserve a resumable checkpoint
at an interruption; never label it goal achievement or a blocked handoff.

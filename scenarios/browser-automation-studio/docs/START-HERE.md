# Start Here

Browser Automation Studio owns visual workflow authoring, recorded browser interaction, deterministic replay evidence, and automated end-to-end validation for Vrooli scenarios.

For the browser-first rehabilitation proposal, read the [2026-09-21 refactor
assessment](internal/REFRACTOR_ASSESSMENT.md), its [measured
baseline](internal/REFRACTOR_BASELINE_2026-09-21.json), and the [BAS-RF issue
register](PROBLEMS.md#refactor-investigation-register--2026-09-21). The assessment
distinguishes observed behavior, source findings, historical evidence, and
proposed targets; implementation has not been authorized by that document.
The [2026-09-22 UTC follow-up](internal/REFRACTOR_ASSESSMENT.md#follow-up-isolated-reproductions--2026-09-22-utc)
adds executable reproductions, expanded memory evidence, and six further issues.
The [profile/replay pass](internal/REFRACTOR_ASSESSMENT.md#profile-and-replay-investigation--2026-09-22-utc)
adds profile commit/recovery and browser-input/workflow fidelity findings.
The [session/frame/retention pass](internal/REFRACTOR_ASSESSMENT.md#session-frame-and-retention-investigation--2026-09-22-utc)
adds retry/reset, preview lifecycle/fallback, retention progress and failed-cleanup
receipt findings.
The [execution/retry pass](internal/REFRACTOR_ASSESSMENT.md#execution-retry-and-cancellation-investigation--2026-09-22-utc)
adds instruction admission/ownership, loop/retry identity collisions and
cancellation/failure evidence findings.
The [reuse/profile/evidence pass](internal/REFRACTOR_ASSESSMENT.md#reuse-profile-and-evidence-investigation--2026-09-22-utc)
adds incompatible profile reuse, stale evidence ownership and bypassed
desktop/Android admission checks.
The [stream/live-fixture pass](internal/REFRACTOR_ASSESSMENT.md#stream-lifecycle-and-live-fixture-investigation--2026-09-22-utc)
adds producer cleanup races, ineffective settings, a broken validation flag and
lossy outcome projection. The investigation recorded 49 open issues; preparation
validation subsequently added documentation finding BAS-RF-050. A small real
browser counter fixture passes, with explicit limits on what it qualifies.

## Rehabilitation implementation handoff

Start with the [continuous goal message](internal/REFRACTOR_GOAL.md),
[work and qualification protocol](internal/TESTING.md),
[canonical outcome contract](internal/REFRACTOR_CONTRACT.json),
[current checkpoint](internal/REFRACTOR_PROGRESS.md) and
[operator feedback](internal/OPERATOR_FEEDBACK.md).

This engagement uses files only: **do not create or use a Plan Manager plan or
child plans**. PROBLEMS.md retains the sole defect register; the progress file
owns the ongoing work record and resume checkpoint. The previously prepared
plan was archived before execution. No implementation agent or external resumer
has been started by preparation.

The target is substantial improvement in reliability, speed, professional UX,
production readiness and maintainability with materially less debt and
complexity. Green tests and two clean reviews start the next fresh adversarial
investigation; they do not end the goal. Continue autonomously until operator
stop/redirection or runtime interruption, preserving a resumable checkpoint.

Use `program-runtime library run browser-automation-studio.setpoint-read --input
profile=rehabilitation`. Its required unknown rows are instrument-building work,
not passing measurements. Unavailable validation, including later-discovered
release gaps, never blocks useful work. Attempt reasonable authorized remedies,
record unverified outcomes and continue without asking the operator to unblock
prerequisites. [Preparation evidence](internal/REFRACTOR_PREPARATION_2026-09-22.json)
is historical setup evidence and does not certify the product.

## Initialization Protocol

Start the scenario with `make start` or `vrooli scenario start browser-automation-studio`; never launch API or driver binaries directly. Use `vrooli scenario test browser-automation-studio` for the server-owned suite and wait on its returned run once.

## Architecture Rules

The API owns workflow validation, persistence, and orchestration. The Playwright driver owns browser sessions and typed instruction execution. Proto definitions are the cross-language contract. UI code may not recreate retired legacy instruction shapes.

## Replacing The Example Domain

New browser capabilities begin as typed proto actions, then gain API compilation, driver execution, UI authoring, and deterministic coverage. See [Architecture](concepts/ARCHITECTURE.md) and [Domains](concepts/DOMAINS.md).

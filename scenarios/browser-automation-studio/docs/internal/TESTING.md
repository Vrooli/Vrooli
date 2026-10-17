# Browser Automation Studio verification

How BAS changes are verified. The goal, queue and epochs live in the
[goal home](goal/GOAL.md); the epoch standard (brief, slice log, step-back,
acceptance) lives only in the `large-effort-orchestration` skill. The 24
preservation journeys are the requirements in
[`requirements/08-rehabilitation`](../../requirements/08-rehabilitation/module.json);
quality bands are in [TARGETS.md](goal/TARGETS.md).

## Journeys and the fixture site

Epoch gates are the preservation journeys an epoch touches, run against a local
fixture site so no external account or network is needed. Each journey asserts
given/when/then behavior through the product surfaces, tagged `[REQ:BAS-RH-Jnn]`.
Journeys that need native machines or external accounts (J09, J10, J12, J14) stay
unverified until a target exists (see [TARGETS.md](goal/TARGETS.md)).

Run the suite from `scenarios/browser-automation-studio/playwright-driver`
against the `bas-goal` shadow:

```bash
pnpm test:journeys --json --outputFile=<path>
```

It fails fast with "shadow unavailable" when the shadow is down, serves the
local fixture site for the duration of the run, and reports one result per
`tests/journeys/J*.test.ts`. There is no Test Genie journeys phase; this command
is the suite. J11 (AI navigation with a stubbed model) runs as a driver
integration test for now; its journey coverage is pending (QUEUE.md).

## Focused checks

- API: `cd api && go test ./<changed packages>/...`
- UI: `cd ui && pnpm type-check && npx vitest run <changed domain>`
- Driver: `cd playwright-driver && npx jest <changed area>`
- Net lines: `python3 docs/internal/refactor_inventory.py --no-git` before and
  after a work unit (never git).
- Scoped Test Genie phases, one at a time, blocking once on the run:
  `vrooli scenario test browser-automation-studio --phases <phase>` with
  `tidiness` or another phase the change touches.

Run changed BAS code on the goal's shadow, never by restarting the live BAS
mid-run. The shadow, daily qualification and unavailable targets are described
once, in [TARGETS.md](goal/TARGETS.md). Never weaken an assertion or count a
local diagnostic as proof.


## UI project and coverage contract

Recording/session ownership regressions live beside `sessionStore`,
`useRecordingSession` (including its real-store integration) and `useRecordMode`.
Run those files in the `record-mode` project with adjacent `RecordingApiService`
and navigation tests; `RetryService.test.ts` belongs to `shared`. Deferred fake
responses verify A/B/A replacement, unmount, stale finalizers, current retry and
same-lifetime creation deduplication. The validated redirect after creation must
preserve its profile/viewport; a different session must not inherit them.
The explicit created-session recording handover must preserve state with start
completion both before and after the props update. Returning to validated A
must immediately retire pending B validation and its loading indicator.
`export-domain` owns `useExportProgress.test.tsx`: resetting or revisiting a
terminal export rehydrates progress without repeating terminal callbacks or
accepting later processing regression. These deterministic checks do not prove
transport cancellation, remote orphan cleanup or native recording/session
behavior; those require separately admitted owner qualification.

`ui/vitest.projects.json` is the single project/include/exclude inventory and
scenario-wide coverage policy. `pnpm test` runs the five named smoke projects;
`pnpm test:full` and `pnpm test:coverage` select all fifteen projects, including
execution-viewer. The file-selection regression refuses unassigned or multiply
owned ordinary UI tests. Real-browser, managed and host-device driver suites
remain separate; UI selection does not replace their qualification.

For runner/config/coverage changes, run the deterministic owner regressions with
`cd ui && pnpm test:coverage-contract`, then the affected named Vitest tests and
`pnpm exec tsc --noEmit`. The script regressions use synthetic coverage maps and
one isolated Vitest provider fixture; they do not launch browsers. A separate
`tsconfig.node.json` check currently exposes pre-existing missing Node type/lib
configuration and is not a substitute for the maintained source type-check.

Coverage processes remain serial to bound heap usage. One nominated real project
(`stores`) asks Vitest/V8 to instrument every eligible source, including unimported
executable modules; other shards contribute executed maps. Type-only modules use
the provider's executable-statement model. Include/exclude policy and all four
85% thresholds remain unchanged. The location-aware merger reconciles maps by
source locations; per-project numeric counter IDs are never global identities.

Each invocation retains outputs under `ui/coverage/runs/<id>/`, with expected
projects, local UI source/test/config fingerprint, completed-shard report hashes
and final disposition in `manifest.json`. This binds local inputs; it does not
seal external dependency/host state. Missing/unexpected/stale shards or absent
source-closure evidence produce an incomplete merge error (exit 2), not a valid
coverage reading. A coherent below-floor report remains a failed gate (exit 1),
with summary and behavior requirements preserved. Test failures retain partial
behavior evidence and their nonzero exit; they do not publish combined coverage.
Focused no-coverage commands preserve the previous coverage report.

Compatibility outputs remain `coverage/coverage-final.json`,
`coverage/coverage-summary.json`, `coverage/vitest-requirements.json` and
`coverage/vitest-run.json`. Read the final run manifest/disposition together with
its retained run directory; a preserved old summary after an incomplete run is
not new coverage evidence. Keep Test Genie run artifacts/receipts for qualification.
The previous 36.81% report was produced by an invalid numeric-ID merger and omitted
source closure. It is historical failing-gate evidence, not an accepted baseline
or a promise that the truthful replacement will meet 85%. Broad behavior coverage
must be earned through actual obligations, without floor/exclusion changes.


Per-project coverage output is JSON only; the previous fourteen-shard raw tree
contained 1,974 files/69,118,873 bytes, of which 16,979,222 bytes were the actual
coverage maps (read-only 2026-10-16 workspace measurement). Removing redundant
HTML/text/summary generation preserves raw merge diagnostics; final machine
summary, requirements and native-quality handoff remain. Actual new-run bytes
must be measured after root qualification; no throughput or disk-quota claim is
made from this historical observation.

Test Genie's `api/internal/shared/runs/retention.go` is the authority for its
cataloged terminal run artifacts/logs/history; owner-scoped expiring leases
protect retained runs (`docs/internal/INVARIANTS.md` in Test Genie). This does not
prove it collects BAS's direct local `ui/coverage/runs` history. Local historical
run retention remains unverified and cleanup is manual/owner-coordinated. The UI
runner adds no private garbage collector, background retry loop or deletion of
server-owned/pinned evidence. Each invocation has a fixed selected-project bound
(fifteen full shards); retaining history is separate from that bound.


### Qualified complete-source baseline,2026-10-16

Root independently reviewed the reporting repair and ran one actual full UI
coverage command. Run`0e787692-9d8f-493e-a322-4b903f4481ea` completed all fifteen
projects and correctly exited1 with disposition`below-floor`. Its qualified
baseline is lines/statements23,653/74,446=31.77%, functions830/2065=40.19% and
branches4081/6032=67.65%. Per-project native observations and behavior requirements
are retained; assertion-profile support remains a separate claim. Evidence is
`video-toolchain-reliability-2026-10-16/qa/bas-ui-continuation-full-coverage.log`
and the run manifest/raw maps under the reported UI run directory. Do not compare
this to historical36.81% as a coverage increase/decrease: the old merge and source
denominator were invalid. This is a valid failing quality gate. Grow meaningful
export/recording/video-path behavior coverage in bounded owner work; do not chase
85% through imports, assertion padding, source exclusions or lowered floors.

### Native replay adoption without application deployment

`api/services/export/render/playwright_capture_integration_test.go` uses an
isolated current UI dist served by an owned test HTTP server and an explicitly
admitted `PLAYWRIGHT_DRIVER_URL`. It invokes the maintained executor/capture
instructions; it does not require deploying the current application API/UI or
restarting a protected shadow scenario. Missing inputs skip the tests, so a
native qualification receipt must show that the selected cases actually ran.

The optional public metadata `requestToken` acknowledges exact spec adoption;
`requestError: { requestToken, message }` reports an attempted token's published
parse error without claiming adoption. Native capture ignores prior-token
errors, promptly fails a matching current error, and retains its15s deadline.
Legacy visible errors alone cannot identify a failed revision. Mounted and
generated-script fake tests cover this protocol separately from native paint.

After owner admission of a driver and media/browser slot, run from `api`:

```bash
BAS_REPLAY_TEST_DIST=<absolute isolated-current-UI-dist> PLAYWRIGHT_DRIVER_URL=<owned-admitted-endpoint> go test -tags=integration ./services/export/render -run '^TestPlaywrightSamePageReplayAdoptionIntegration$' -count=1 -v
```

This fixture has a60s execution deadline plus the maintained executor's bounded
session-close budget. It uses one normal synthetic executor session/page and
closes it through executor ownership. It first publishes a real invalid-spec
error through the public message contract, then uses the actual generated
scripts for two same-ID requests with red/blue backgrounds. Four screenshots
must contain the respective style pixels. No video encoder or real session-store
allocation is involved. `BAS_REPLAY_TEST_EVIDENCE=<new absolute path>` optionally
retains PNGs at `<path>-same-page`; existing directories refuse overwrite.
Compilation or fake tests do not establish native success. Broader continuous
recording/phone-frame cases remain separate admitted integration selections.

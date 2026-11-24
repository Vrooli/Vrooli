# Local experience telemetry

Local-only interaction telemetry for scenario UIs: what the operator actually
did, where time went, and where the interface got in the way. It exists to feed
experience audits with observed behavior instead of guesses, and to diff
observed flows against the declared journeys in `experience/`.

It is **not** a keylogger. It records semantic actions and timing, never input
values, printable keystrokes, clipboard contents, IP addresses, or user agents.

## Why it lives in `api-base`

Scenarios are isolated pnpm projects, so a new shared package would need 100+
adoptions. `@vrooli/api-base` is already a dependency of 112 UIs and its server
is the process every scenario actually runs: `vrooli develop` and
`vrooli scenario start` both launch `node server.js` over a prebuilt `dist`.
Vite dev servers are refused by deployability conformance.

That makes serve-time injection the only version-proof lever:

- `createScenarioServer` mounts three local routes and injects one module
  script into HTML responses, gated per request.
- The script is a self-contained ES module with no runtime imports, so the
  host UI's React, bundler, Node, and TypeScript versions are irrelevant.
- `file:` dependency contents are build inputs, so bumping api-base rebuilds
  consumers automatically at the next setup. No `ui/package.json` edits and no
  source changes in any scenario.

Coverage today is every UI served by `startScenarioServer` /
`createScenarioServer` (105 of 120 `ui/server.js` files). Custom servers can
mount the router themselves; see [Custom servers](#custom-servers).

## Turning it on and off

| Control | Effect |
| --- | --- |
| Default (no configuration) | On for loopback requests in a local run, off under `NODE_ENV=test`, `VITEST`, and `CI` |
| `VROOLI_EXPERIENCE_TELEMETRY=0` | Off everywhere |
| `VROOLI_EXPERIENCE_TELEMETRY=1` | On, including in tests/CI |
| `VROOLI_EXPERIENCE_DIR=<path>` | Override the session directory |
| `.vrooli/experience.json` | Per-scenario policy: `enabled`, `sample`, `captureLabels`, `performance`, `allowRemoteHosts`, `maxAgeDays`, `maxSessions` |
| `VROOLI_EXPERIENCE_POLICY=<path>` | Explicit policy file instead of the default lookup |
| `experienceTelemetry: false` in `createScenarioServer` | Off for that server |
| `experienceTelemetry: { ... }` | Tune retention, sampling, labels, performance, paths |
| Request header `x-vrooli-experience: off` | Skip this response (automation and e2e) |
| `localStorage['vrooli.experience.disabled'] = '1'` | Browser-tab opt-out before the next load |
| `window.__vrooliExperience.disable()` | Disable, end the session, and flush immediately |

Non-loopback hosts are never instrumented by default, so a tunnel visitor or a
phone on the LAN does not enter the operator's history. Force it for your own
device with `VROOLI_EXPERIENCE_ALLOW_REMOTE=1`.

A scenario can carry its own policy at `.vrooli/experience.json`:

```json
{ "enabled": true, "sample": 0.5, "captureLabels": false }
```

The server reads it from the UI working directory or its parent. Precedence is
explicit option, then `VROOLI_EXPERIENCE_TELEMETRY`, then the policy file, then
the local default. A malformed policy file is ignored rather than blocking
startup.

## What is captured

| Event | Fields | Why |
| --- | --- | --- |
| `session_start` | route, viewport, DPR, reduced motion, theme | Baseline for the tab |
| `route` | route, previous route, `push`/`replace`/`pop`/`load`, scroll depth | SPA-aware dwell and navigation |
| `action` | `click` / `keyboard` / `submit` / `change`, element ref, cursor, hover time, pointer travel/reversals | The semantic action stream |
| `key` | `Tab`, `Enter`, `Escape`, arrows, `Shift+Tab`, `Mod+Z`, ... | Navigation and recovery, never characters |
| `field` | focus/blur, field type, safe label, duration, edited flag | Typing effort and abandonment, never values |
| `friction` | dead click, rage click, repeated action, undo, dismiss, backtrack, truncation | Cheap, high-signal problems |
| `error` | message, source, line | Error paths dominate interaction tuning |
| `performance` | TTFB, DCL, load, LCP, CLS, long tasks | Whether a stall is layout or latency |
| `visibility` | visible/hidden | Active-time accounting, and leave/return cycles |

Element references carry the nearest `data-testid`, role, accessible control
name, a structural fallback path, a stable fingerprint, and the nearest
`data-experience-region` / `data-experience-surface`. They never include text
from inputs.

Pointer movement is reduced to three numbers over a one-second window: travel
distance, direction reversals, and speed. Raw trails are not stored.

Sensitive fields (`password`, `email`, `tel`, payment/OTP/SSN hints, and
`autocomplete` payment/password tokens) report `name: "[redacted]"`; their
values are never read.

## Storage

```
~/.vrooli/experience/sessions/<scenario>/<session-id>.jsonl
```

One append-only line per batch. A crash loses at most the batch in flight; there
is no database to corrupt. Defaults: 30 days, 500 session files, 5,000 events
per session, 250 events per request, 256 KiB per request. Retention runs at
server start and at most once per hour. Each batch records the tab session id,
the server-owned scenario name, the receive time, and the served build identity
(`VROOLI_BUILD_IDENTITY`, falling back to the server version).

Nothing is written into the repository. Derived reports go to
`~/.vrooli/experience/reports/<scenario>/` unless `--out` says otherwise.

## Analysis

```bash
# Report for one scenario, comparing against its declared journeys
node packages/api-base/dist/experience/cli.js report \
  --scenario git-control-tower \
  --spec scenarios/git-control-tower/experience \
  --out /tmp/git-tower-report

# Or through pnpm after a build
pnpm --filter @vrooli/api-base experience:report -- --scenario web-console

# Mark evidence stale unless the discussed build was observed
node packages/api-base/dist/experience/cli.js report --scenario web-console \
  --build "sha256:<lifecycle build identity>"

# Inspect, prune, and locate
vrooli-experience status --scenario web-console
vrooli-experience prune --scenario web-console --days 14 --max 200
vrooli-experience path --scenario web-console
```

Outputs:

- `friction-report.json` — schema `experience-friction-report/v1`, with `builds`
  and a `sha256:` `reportDigest`
- `friction-report.md` — reviewable summary
- `observed-flows.json` — candidate flows, deliberately **not** a valid
  `experience-journey` document, so it cannot masquerade as declared intent
- `observed-behavior.json` — `experience-observed-behavior/v1` with an explicit
  `fresh` / `stale` / `absent` state and the report digest
- `advisory-findings.json` — ranked findings, every one marked `advisory: true`
- `correction-candidates.json` — review candidates for the EX1 corpus, never
  corrections
- `packet-observed.json` — a compact block a redesign packet can show beside a
  captured before/after

The friction report carries per-session IDs, build identities and time bounds,
and the observed-behavior and packet blocks repeat those session references.
Consumers must bind them to the report digest and reject missing session/build
identity as `not-evaluated`; aggregate `builds` alone cannot establish which
session ran which build.

The report ranks by total time cost (frequency × median duration), not by
statistical significance. With one operator that is the only useful ordering.

JSON Schema for the batch and every artifact lives in
`packages/api-base/schemas/` and is exported under
`@vrooli/api-base/schemas/*`.

## Declared versus observed

`--spec` reads the scenario's `experience/` directory. For every declared
journey it reports how many sessions followed the page order, median duration,
and friction inside that window. Flows whose route pairs appear in no declared
journey are listed separately.

That diff is usually the most actionable part:

| Pattern | Likely meaning |
| --- | --- |
| Frequent undeclared flow | Missing affordance or wrong information architecture |
| Declared journey never observed | Undiscoverable first step, or dead weight |
| Declared journey observed with high friction | Right destination, bad route |
| Heavy command/keyboard use | UI discoverability failure |

## Leaving and returning

`visibility` events are not only used for active-time accounting. The report
counts every hidden → visible cycle and attributes the time away to whichever
route was current, so a dialog or job that makes the operator leave the page is
visible as a route metric:

```text
| Route    | Returns | Median away | Total away |
| /commit  | 4       | 38s         | 2.4m       |
```

Two rules keep this honest:

- A return needs two cycles on a route before it is raised, so one interruption
  is not treated as a pattern.
- A hidden tab locates a wait; it does not explain it. The recommendation names
  the wait and asks whether progress is visible and whether other work can
  continue, rather than declaring the wait a defect.

A full page leave (another origin, a closed tab, a hard reload) ends the
session. Returning then starts a new session id, and the two are not currently
correlated. Tab switching and minimize are covered; cross-session correlation is
not.

## Declared capabilities

Interaction data can show that a capability is used. It cannot show that an
available capability is *never* used unless the scenario says the capability is
expected. Declare those in the same per-scenario policy:

```json
{
  "capabilities": [
    { "id": "stage-queue", "label": "Queue files", "testId": "stage-during-suggestion-queue" }
  ]
}
```

The analyzer matches observed clicks by `data-testid` (preferred) or accessible
name and reports `exercised`, `unobserved`, or `insufficient-evidence`:

```text
| Capability         | Observed uses | State                |
| Queue files        | 0             | unobserved           |
```

`insufficient-evidence` is deliberate: below 10 observed actions a report never
claims a capability is unused. An `unobserved` result becomes a medium advisory
recommendation about discovery — whether the path is offered at the moment it
would help, whether its label says what it does, and whether using it costs the
operator anything. It is not proof the capability is wrong.

## From report to redesign

The report is the evidence layer. The redesign layer already exists:

1. Run the report and read `What to fix first`.
2. Capture the real page with `react-component-library page capture --scenario <name>`.
3. Edit the frozen page into a proposal and render it.
4. Present the before/after with the observed numbers attached.

That is the EX11 redesign round trip in the experience-engineering topic. Keep
friction budgets as journey-level claims checked by workflow-health rather than
building a second validator.

## Consumer artifacts

Every artifact is advisory and ties back to the same `reportDigest`:

- **Evaluation (EX2):** `observed-behavior.json` is the only envelope a
  consumer should read. `state: absent` means no sessions; `state: stale` means
  the expected build was not observed. Neither may become PASS/FAIL or fresh
  evidence. Findings are advisory, not a phase.
- **Corrections (EX1):** `correction-candidates.json` is a review queue. A
  candidate is a question ("was this friction a defect?"), not a gold label.
- **Redesign packets (EX11/EX12):** `packet-observed.json` carries a headline,
  top flow, time cost, build, window, digest and a caveat. It changes no
  fidelity label and creates no region correction.
- **Front door:** the dedicated `vrooli-experience` binary (or the CLI path
  above). Do not add a top-level `vrooli` subcommand for this; the control plane
  already carries too many commands.

## Verify

```bash
pnpm --filter @vrooli/api-base build
pnpm --filter @vrooli/api-base test
pnpm --filter @vrooli/api-base experience:e2e
pnpm --filter @vrooli/api-base experience:overhead
```

`experience:e2e` serves a throwaway UI through the real shared server, drives it
with Chromium, and asserts injection, capture, ingest, storage, report
generation, the digest, the advisory findings, the correction candidates and the
packet block. `experience:overhead` serves the same page with telemetry off and
on and reports load, 100-click and long-task deltas. The last measured run was
about +1-3 ms load, +6-8 ms for 100 clicks (worst case), and no long tasks.

## Custom servers

Servers that do not use `startScenarioServer` can call one helper immediately
after creating the Express app, before `express.static` and routes:

```ts
import { installExperienceTelemetry } from '@vrooli/api-base/experience'

const app = express()
installExperienceTelemetry(app, { serviceName: 'my-scenario', distDir })
```

It mounts ingest/status, serves the agent, serves the entry document through
`res.send`, and injects into HTML produced by `res.send` or an HTML
`res.sendFile`. It is loopback-gated and inert when disabled. A CommonJS server
can use `await import('@vrooli/api-base/experience')` inside an async startup
(Node 22.12+ can `require()` this ESM entry). Guard the import if the scenario
may still have an older installed `api-base`.

`ai-chatbot-manager` is the reference adoption. The remaining legacy servers
(vanilla Express/http shells) still need the dependency and the call; they are
not covered by `startScenarioServer`.

Host scenarios that proxy child UIs inject into the child document. Each child
instruments itself, so no cross-origin DOM access is needed. Same-origin frames
— including portal/app-monitor proxy paths and Web Console's `/_local` proxy —
share `sessionStorage` and therefore the session id. Direct cross-origin frames
get their own session; a future bridge message can correlate them.

## Privacy and security

- Same-origin only: the routes sit behind the scenario origin policy, and the
  ingest path is excluded from SPA fallback.
- The scenario name, receive time, and retention are server-owned; a page
  cannot write into another scenario's history.
- Session ids are opaque and validated against `^[A-Za-z0-9_-]{8,64}$`.
- Bodies are size-capped and event counts are bounded.
- No cookies are set, and no identifier survives the tab: the session id lives
  in `sessionStorage`.
- The agent is inert when disabled and every handler is wrapped so telemetry
  can never break the host application.

Residual risk is deliberate: a same-origin script in the page can post bogus
events, and the data is only as private as the operator's home directory. This
is local optimization tooling, not an analytics platform.

## Limitations

- Hesitation is inferred from gaps, hover time, and pointer reversals. It finds
  *where*; it does not explain *why*.
- `n=1` is expected. Trends matter less than cost per week.
- Dead-click detection waits 700 ms for a route, DOM, or network change. A
  delayed response can read as a dead click, which is still useful signal.
- Route detection patches `pushState`/`replaceState`, `popstate`, and
  `hashchange`. Apps that render views without changing the URL will look like
  one long route.
- Leaving is measured by tab visibility. Closing a modal and reopening it later
  is *not* a visibility event: it appears as clicks on the close and reopen
  controls unless the surface emits its own lifecycle events.
- A full page leave ends the session; the return is a new session and is not
  correlated with the first.
- Sampling is per tab. Use `sample < 1` only when volume is a problem; it
  reduces the denominator for every ratio.

## Next steps

The producer and consumer artifacts are implemented. What remains is adoption and
research-side wiring:

1. **Fleet refresh (operational).** `file:` dependency contents are copied at
   install time. Refresh `api-base` across consumers in a controlled window
   (`vrooli package refresh api-base all` or per-scenario setup) and verify a
   sample of standard UIs plus one custom server. Expect
   `running_setup_deferred` for running consumers; do not claim fleet adoption
   from a sample. `vrooli package dependents api-base` currently reports a
   pre-existing `device-studio` consumer-class violation from its evidence
   snapshots; resolve or exclude that before a broad refresh.
2. **Legacy custom servers.** The standalone helper is implemented and adopted by
   `ai-chatbot-manager`. The remaining vanilla Express/http shells still need
   the `api-base` dependency and the one-line call. Do not duplicate the analyzer.
3. **Consumer wiring (research).** The producer now emits `observed-behavior`,
   `advisory-findings`, `correction-candidates` and `packet-observed`. EX2, EX1,
   EX11 and EX12 still need to read them under their own ownership. The handoff
   is `~/.vrooli/research/topics/experience-engineering/work/OBS1/BRIEF.md`.
4. **Front door.** Use the dedicated `vrooli-experience` binary. Do not add a
   top-level `vrooli` subcommand; the control plane already carries too many.
5. **Calibrate.** Overhead is measured (about +1-3 ms load, +6-8 ms/100 clicks, no long
   tasks on the fixture). Record several real sessions in one or two flagships
   and calibrate hesitation thresholds against a small labelled subset.
6. **Cross-origin frames.** Same-origin frames already share the session id via
   `sessionStorage`; direct cross-origin frames need an explicit bridge message
   if correlation becomes necessary.
7. **Dialog and surface lifecycles.** Extend the agent to emit open/close events
   for `[role="dialog"]` and `data-experience-surface`, so "closed the dialog,
   worked elsewhere, reopened it" is measured directly instead of inferred from
   two unrelated clicks. This is the next precision step for GCT's commit
   suggestions dialog.
8. **Adjacent debt.** The 95 duplicated `ui/src/lib/profiler.ts` copies are
   unrelated but show the cost of per-scenario copy adoption; consolidate them
   separately.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| No session file | `Host` must be loopback; `VROOLI_EXPERIENCE_TELEMETRY` must not be `0`; `NODE_ENV`/`CI` must not force it off |
| Agent 404 | Run `pnpm --filter @vrooli/api-base build`; the asset is served from `dist/experience/agent.js` |
| No page in the report | Interaction happened in a different browser tab or a different scenario directory |
| Report has no journeys | Pass `--spec <scenario>/experience`; the directory must contain `index.json` |
| Empty `observed-flows.json` | Raise repetitions with `--min-count 2` or record more sessions |

# Runbook — Personal Planner

## Purpose Of This Document

This document is the operator's guide to running, diagnosing, recovering,
and maintaining Personal Planner. The October 2, 2026 inspection established
secured live access and a separate writable shadow route, with repeatable
Browser Automation Studio flows, screenshots, recordings and storage readbacks.
Start with the verified procedures below; runtime findings and incomplete
journeys are recorded in [TESTING](../internal/TESTING.md) and
[PROBLEMS](../internal/PROBLEMS.md). Access and evidence capture are established
separately from product acceptance; Planner defects remain open.

Current access and evidence navigation:

- **Live:** https://personal-planner.itsagitime.com through existing Cloudflare
  Access. Cloud sign-in and live Today observation succeeded at 04:41 UTC.
  Broader page visits must not be assumed read-only: Settings initialized a
  default profile in shadow. Use the verified shadow for writable inspection.
- **Writable shadow:** `http://localhost:24816` routes to API `17778`; process
  namespaces and actual SQLite handles independently proved separation from
  live. Recheck routing and fixture identity before further writes. A test
  header or shadow flag alone is not isolation proof.
- **Procedure:** [secured access and isolation](#2026-10-02--verified-secured-access-and-isolated-ux-inspection),
  then [repeatable testing, capture and tool friction](#2026-10-02--second-sweep-procedure-and-corrected-access-status).
  Reuse healthy instances; do not restart services for documentation work.
- **Recordings:** local BAS video bytes decode successfully. Drive player
  playback is reported failing and remains unverified; local decoding does
  not prove Drive or cloud-browser playback. Consumers must materialize files
  through the supported Library route and verify local bytes and playback.
  Before further capture or delivery, follow the
  [BAS screenshot/video capture-quality procedure](../../../browser-automation-studio/docs/nodes/screenshot.md#capture-quality-during-native-video).
  Its neutral full-page versus viewport-only comparison is verified; product-level
  workaround validation and destination playback remain pending.

Earlier design-stage material is retained below as an operating contract.
Incident, backup and maintenance descriptions for unverified jobs or domains
are intended behavior, not proof of a current implemented capability. The older
health/notes-only description and first-pass pending sign-in observation are
historical and are superseded by the dated October 2 evidence.

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What do I check when a dependency, provider, job, or port misbehaves?
- How do I back up state and restore it safely into a clean workspace?
- What routine maintenance keeps jobs and storage healthy?
- Where do operational defects get filed?

## Start / Stop / Status

Run everything through the lifecycle from the scenario directory. **Never
run the API or UI binaries directly** — the lifecycle owns process
naming, port allocation, health checks, and logs, and running a binary by
hand bypasses all of it.

```bash
make setup     # wraps `vrooli scenario setup`  — build + install deps/CLI
make start     # wraps `vrooli scenario start`  — start API + UI + resources
make status    # wraps `vrooli scenario status` — running surfaces and ports
make logs      # wraps `vrooli scenario logs`   — tail API + UI logs
make stop      # wraps `vrooli scenario stop`   — clean shutdown
make restart   # stop then start (preferred over restarting one surface)
make test      # wraps `vrooli scenario test`   — validation lifecycle
```

Equivalent control-plane forms (`vrooli scenario start|stop|status|test
personal-planner`) are fine. Resolve a live port with
`vrooli scenario port personal-planner API_PORT` (or `UI_PORT`). Both
surfaces expose `/health`.

## Common Incidents

| Symptom | Checks | Response |
|---|---|---|
| **Scenario will not start** | `make status`, `make logs`; is scenario-authenticator running? | scenario-authenticator is **required** (`must_start`). Without an identity provider a workspace cannot be scoped to a subject, so Personal Planner refuses to start by design. Restore the authenticator, then `make restart`. This is expected fail-closed behavior, not a Personal Planner bug. |
| **notification-hub down/degraded** | `make status` for the dependency; outbox age (see Maintenance) | Notices remain **in-app only**; no external channel delivery. The product stays fully usable. Delivery is retried idempotently on recovery and must never produce duplicate human-facing messages. No action needed beyond restoring the hub; do not disable the notifications feature. |
| **External provider outage / rate limit** | integration health panel: last successful refresh, current problem, next retry, affected scope; provider freshness metric | Bounded exponential backoff with jitter runs automatically. Last-known busy intervals are **retained with a freshness warning**; feasibility is marked uncertain. **Never treat a provider failure as newly free time.** No manual action unless the outage persists past the backoff horizon. |
| **Provider reauth required** | `PROVIDER_REAUTH_REQUIRED` surfaced in the health panel; credential owner state | Token expired/revoked. Reconnect through the provider connection flow; stale busy facts are retained meanwhile. Do not paste tokens into logs or configs — credentials live only in the credential owner. |
| **Provider cursor invalidated** | sync failure logs; cursor state for the connection/calendar/query scope | A **scoped full resync** rebuilds a replacement snapshot and is switched in only after success; native tasks/schedules/history are untouched. A moved recurring instance keeps its stable original-instance key. |
| **Stale forecast** | forecast age metric; whether a material input revision occurred | Forecast refresh coalesces on material input revisions and writes an immutable snapshot (no accepted-schedule mutation). If a forecast is older than expected after a relevant edit, confirm the forecast-refresh job is leased and advancing; a stale forecast is a signal, never a reason to auto-change an accepted schedule. |
| **Stuck job lease** | job status: attempts, next retry, last error class, terminal unresolved state; pending outbox age; cursor advancement | Jobs use leases with observable statuses; a crashed worker must **not** strand a permanent active lock. If a lease is held with no progress, the recovery is lease expiry/reclaim, not a manual DB edit. Watch for tight loops on invalid credentials or unsupported source commands — those must back off, not spin. |
| **Port conflict** | `make status`; a previous instance still holding `API_PORT`/`UI_PORT` | `make restart`. The lifecycle reallocates ports; do not pin a port by hand. |
| **API unhealthy** | `/health` on `API_PORT`; SQLite path writable; API logs via `make logs` | `make setup` to rebuild, verify the data directory is writable. On a SQLite write failure the command returns a **failed** state and preserves user input with retry/export — it never claims saved. |
| **UI blank or stale** | `/health` on `UI_PORT`; browser console; `ui/dist` freshness | `make setup` then `make restart` to rebuild the bundle. |
| **CLI talks to an old API** | `personal-planner status`; resolved API base | Reinstall via `make setup` so the CLI picks up the current resolved port and token. |

## Backup / Restore

- **Backup — native versioned JSON export.** Export produces a
  self-describing package: format version, export time, timezone/policy
  metadata, stable IDs, provenance, native records, allocations,
  meaningful revisions, and actuals — enough to interpret and restore. It
  **excludes** credentials, access/session tokens, reusable share
  secrets, and unnecessary private infrastructure IDs. CSV summaries are
  for actuals/estimates; native backup exports carry the selected history
  needed for restoration and label that choice. Downloadable export
  artifacts are authenticated and regenerable (24-hour default retention).
- **Restore — validated native import into a new workspace.** Import
  validates the whole package first (counts, conflicts, warnings, and a
  preview) before writing, defaults to a **new private workspace**, and is
  **idempotent** — reimporting the same package does not duplicate
  objects.
- **Restored state is inactive / needs-review, by design:**
  - Restored **provider connections** require reconnection (no tokens are
    in the export).
  - Restored **share grants** are inactive until re-granted.
  - A restored **running focus session** becomes "needs review," never
    assumed-productive elapsed work.
- **ICS import/export** is a separate manual fallback, not a live
  connection: import previews with duplicate detection, provenance, and
  bounded recurrence expansion, and never fetches URLs embedded in
  descriptions; export never routes private notes through a
  commitment-share path.
- Always **verify a restore before any cutover** (see DEPLOYMENT
  Rollback): a snapshot you have not test-restored is not a backup you can
  rely on.

## Maintenance Tasks

| Task | Cadence | Procedure / What to check |
|---|---|---|
| Cursor health | ongoing | Confirm each provider connection/calendar/query cursor is advancing only after a batch is durably processed; a stalled cursor points at a sync failure or invalidation. |
| Outbox / receipt age | ongoing | Watch **pending outbox age**; records are retained until safely delivered. Growing age means delivery is stuck (often notification-hub or a job lease). Replays return the prior outcome or safely resume — never a duplicate message. |
| Job status review | ongoing | For each job (provider sync, occurrence expansion, forecast refresh, reminder dispatch, source command retry, learning analysis, share maintenance, cleanup): check attempts, next retry, last error class, and terminal unresolved state; ensure none spin on invalid credentials or unsupported source commands. |
| Cleanup / retention | scheduled | Cleanup preserves user-authored history and bounds transient job/log data per the retention defaults in [`../concepts/DATA.md`](../concepts/DATA.md). Source-receipt cleanup must **never** let an old replay resurrect deleted work — durable origin uniqueness/tombstones are kept. |
| Validate tests | before handoff / release | `make test`; the R1 acceptance suite (T01–T36) is the authoritative evidence. |
| Inspect logs | as needed | `make logs`. |

## Escalation

File operational defects as bug observations to **scenario-qa** using the
report-bug skill (`prompt-manager skill read report-bug`) — it is a skill,
not a shell command, and it writes a bug-inbox entry for the
bug-investigator to drain. Do not carry a private host-repair
implementation in this scenario: detection and remediation of host state
belong to the control plane; Personal Planner may observe and report host
state but must not repair it. Append meaningful completed operational work
to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — tiers, release checklist, rollback
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals, logs, metrics, alerts
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contract and failure modes
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage, import/export, retention, deletion
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common first-boot fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration

## 2026-10-02 — verified secured access and isolated UX inspection

Public route: **https://personal-planner.itsagitime.com**. Tunnel Manager route
`9b39d282-235b-487a-9f0a-616aa776b50d` remains enabled on UI port 20003;
unauthenticated HTTPS returned 302 to the existing Cloudflare Access team login.
Sign in through the existing authorized email-code flow. Do not save codes,
disable authentication, change public exposure, or infer successful sign-in from
an email arriving. Historical first-pass observation: parent cloud sign-in was
pending when this section was written. The second-sweep confirmation below
supersedes it: sign-in and live Today access succeeded at 04:41 UTC.

1. Verify `hostname`, repository ownership, `git rev-parse HEAD`, and existing
   changes before acting. This inspection used swarminator, checkout
   `/home/matthalloran8/Vrooli`, owner matthalloran8, HEAD
   `6173505a87cb9a32e888bb049450fa86231221ac`; the shared checkout was dirty.
   `.agents` exists but `.agents/skills` does not. Read root AGENTS and canonical
   scenario skills; projected/local memories are historical leads only.
2. Discover capabilities using `search-hub query '<intent>' --type
   library,record,skill,doc,command`; use the prescribed Prompt Manager fallback
   when unavailable. Default sandbox loopback calls returned `socket: operation
   not permitted`; the same approved read in `require_escalated` worked. This is
   an execution-mode restriction, not evidence that services are down. Never
   bypass a denied action. Source-ledger discovery providers timed out separately.
3. Inspect `vrooli scenario status personal-planner`, `tunnel-manager routes
   list --json` (filter to this scenario), and `tunnel-manager tunnel status
   --json`. Tunnel was healthy/active/ready at 04:48:48 UTC; its displayed metrics
   sample was historical (August 18), not a current four-connection measurement.
4. For read-only observation, live access needs no isolation setup. For writes,
   prefer an already verified fixture. Otherwise use managed `vrooli scenario
   start personal-planner --instance shadow --json`, never a directly launched
   binary. Verify actual process environment and open SQLite handles, empty
   counts, and shadow UI `/health` upstream before synthetic writes. A shadow
   CLI flag alone proves neither data isolation nor the browser's API target.
5. This run proved live namespace `personal-planner` opens
   `scenarios/personal-planner/data/personal-planner.db` (2 work/4 focus rows);
   shadow namespace `personal-planner_shadow` opens
   `/home/matthalloran8/.vrooli/data/vrooli/personal-planner_shadow/personal-planner.db`
   (initially 0 work/0 focus). Live API/UI = 17604/20003; shadow = 17778/24816.
   Shadow UI health explicitly resolved upstream `http://127.0.0.1:17778/health`.
   Serving API/UI build identity after refresh was
   `sha256:2953bcaf487fec9d7eb4a6e86a0a235e00f4490703c06ce657dd9fb9a1e38620`.
6. BAS live was unhealthy; existing BAS shadow API 15372 worked. Use
   `browser-automation-studio --instance shadow capture --url
   http://localhost:24816/ --capture screenshot,accessibility,video --dimensions
   mobile --wait-for 1000 --out /tmp/<scoped-directory> --json`. Here BAS's
   instance selects the **automation service**, not the target Planner instance.
   Always supply the verified shadow URL explicitly. Chrome extension computer
   use is a verified fallback and reached the same local shadow.
7. For repeatable interactive tests, use `workflows execute-adhoc --flow-file
   <verified-flow.json> --requires-video --requires-trace --wait --json` on BAS
   shadow. Preserve execution ID, terminal result, full timeline and files.
   Shared buttons retain hidden pending labels; semantic/class selectors are
   safer than exact raw `textContent`. Scope assertions to the state-bearing
   surface, wait for asynchronous completion, and independently read API/DB
   state. `body` text can include embedded CSS and give false-positive matches.
   BAS can continue after failed steps: inspect **every** timeline outcome.
8. Do not delete/reset live rows. Retain bounded synthetic shadow records and
   label their provenance. Final count-only live checks remained 2 work/4 focus;
   this is not a whole-database non-mutation hash proof. No live write test ran.

**Unsafe alternative: test header alone.** `api/main.go:112` installs
TestModeMiddleware; `packages/api-core/database/routed.go:205–228` explicitly
returns primary storage when no test pool exists or its lease expires, even
with `X-Vrooli-Test-Mode: 1`. This is source-backed routing evidence; no live
mutating demonstration was performed. The existing
`bas/cases/routed-database/proves-test-pool-routing.json` still declares itself
a scaffold and there is no implemented seed. Never mutate live using that
header without independently proving an active leased pool, actual target DB,
fixture identity, routing statistics, and isolation throughout the run.

**Lifecycle friction and permitted repair.** The first shadow start refused a
stale shared build. Managed dependency traversal rebuilt scenario-authenticator,
attempted optional notification-hub, and restarted stale tunnel-manager; a
runtime-registry deadline left Tunnel Manager start failed. `vrooli scenario
start tunnel-manager --json` restored it. Managed live Planner refresh then
allowed shadow start. Notification Hub's first UI build was killed; the next
managed attempt started it with optional SMTP capability unavailable. No
credentials were read/provisioned; no auth or exposure policy was changed.
Do not repeat this setup unnecessarily when the verified instances are healthy.

Evidence and limits: see [TESTING.md](../internal/TESTING.md), current concrete
findings in [PROBLEMS.md](../internal/PROBLEMS.md), and the scoped evidence bundle
`/tmp/planner-ux-review/delivery`. Library identities are included in its upload
receipt. Consumers on other executors must use current Library materialization
and verify local files; these swarminator paths are not cloud-local paths.

Library export confirmed: `libfile_7d407558468881919b1b9db9aa969b61`
(`personal-planner-ux-20261002.zip`, synthetic captures, flows, timelines, receipt
and snapshot excerpts). Recording: `libfile_4fdaa71ed728819190f5663dd714b263`.
The prepared-upload helper failed before preparation because prepare_uploads
was unavailable in its host runtime; supported Codex host-file uploads succeeded
and returned Library version attributes were applied to original local files.

## 2026-10-02 — second-sweep procedure and corrected access status

Parent confirmed Cloudflare sign-in succeeded04:41UTC and captured read-only
live Today; this supersedes the earlier local report's pending cloud status.
No credentials retained. Broader live page visits may cause implicit writes;
Settings initialized a default planning profile in the independently isolated
shadow. Parent's separate cloud recording experiment remains unverified. Local
BAS video bytes decode, but Drive player playback is reported failing and is
unverified; local video is not evidence that Drive or cloud recording works.

Second preflight05:06:36UTC pinned unchanged HEAD/build,actual separate process
namespaces and SQLite handles,shadow24816→17778 health,live aggregate counts and
responsive shadow work list. No restart was required. Use an aggregate-only live
baseline; do not export populated live rows. Inspect implicit read-side effects:
visiting Settings initialized default planning_profiles in shadow. BAS timezone
America/New_York differs from profileUTC; never infer DST coverage from this run.

Useful repeated test recipe: save a V2 linear flow with explicit verified shadow
URL, viewport and labelled actions; execute through BAS shadow execute-adhoc
--flow-file <path> --requires-trace [--requires-video] --wait --json. Preserve
admitted ID,terminal result and `executions timeline <execution-id> --json`.
Do not repeat a write to recover a missing receipt. Inspect every step; BAS may
continue after failures. Save independent synthetic record reads before/after.

For native video, apply the [owning BAS capture-quality procedure](../../../browser-automation-studio/docs/nodes/screenshot.md#capture-quality-during-native-video)
before selecting screenshot extent. The procedure links the native diagnosis and
separate fix proposal held for owner approval. Viewport-only capture passed a
neutral control; this is not product-level workaround acceptance. Preserve
navigation/loading frames and label presentation edits rather than hiding gaps.

Use label:has(input[name=theme-choice][value=day]) for custom appearance radios;
clicking their raw input testid is intercepted by the decorative indicator.
Scope close buttons inside role=dialog; a bare Close capture locator matches
both backdrop and header button. Escape is verified cancellation/focus-return.
Use KeyboardParams key=k,modifiers=[KEYBOARD_MODIFIER_CTRL] for Control+K;
ShortcutParams Ctrl+k produced Unknown key Ctrl in this runtime. These are
tool/selector frictions,not proof the corresponding product control is broken.

For visible observations use bounded state-bearing innerText and input checked,
selected/validity properties,not body/CSS or hidden pending labels. Native
browser zoom shortcuts had no effect in headless BAS.720×450 effective reflow
is a labelled approximation of1440×900 at200%,not actual200%browser zoom.
Fixed-viewport full_page captures do not expand nested route scrollers; operate
mobile section navigation or explicit scoped scrolling to expose lower controls.

Readback: repeated capture created two rows; goal and target persisted; milestone
persisted but its populated list hangs; profile capacity480→450→reopen→480 was
restored atrevision3. Browser-local Day/Night/art-free/reducedscenery changes use
only the shadow origin and are restored within each test context. Synthetic
tasks/goal/milestone/session/note remain for replay,identified in the manifest.
No integration/share/notification/import/restore/deletion or live write test.
No product fixes or commits/pushes. All original receipts remain intact.

Current broader scope and P01–P42 dispositions live in TESTING.md; concrete new
findings PP-UX-03–06 live in PROBLEMS.md. Local second-sweep export root is
/tmp/planner-second/delivery. Library receipt and file hashes must accompany
cross-executor delivery; receiving executors materialize through Library and
verify local existence/bytes and video playback. Local ffmpeg decoded the32s
keyboard clip without errors; destination playback remains a handoff check.

October2 feedback follow-up: original double Pause used expectedRevision1 twice; first200 paused revision2, second409 aborted. One pause persisted and reopen retained it. Generic failure feedback is PP-UX-07, separate from conservation. End/note-save Countdown selection versus hardcoded Open timer start guidance is PP-UX-08, not a backend mode change. Exact original request/response receipt and trace hash: /tmp/planner-second/delivery/first-double-pause-exact-responses.json. Access/isolation/capture/replay prerequisite is established separately from blocked core-loop product acceptance. Native200% zoom and consuming-executor playback remain unverified. See PROBLEMS and second-sweep report.

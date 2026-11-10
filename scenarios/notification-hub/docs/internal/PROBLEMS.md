# Problems — Notification Hub

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-17 — Template scaffold still documented as if it were the product

**Status: RESOLVED 2026-08-18.** The reference set was rewritten against the
five-domain implementation and each registered document is now tracked at
`active` maturity.

**Symptom:** `SEAMS.md`, `TESTING.md`, `ARCHITECTURE.md`, and the
`reference/` documents still contain generic scaffold prose. `DOMAINS.md`,
`DATA.md`, `FLOWS.md`, `SECURITY.md`, and `INTEGRATIONS.md` were rewritten
against the real domain map on 2026-08-18 and no longer describe the removed
example domain.
A reader who trusts the old generic documents would have concluded this
scenario was still the generated example app.

**Root cause:** the generated documentation set was not fully rewritten when
the real notification domains first landed.

**Resolution:** Rewrote the remaining reference and internal documents
against the actual domain map and raised their manifest maturity only after
the documentation-health checks passed.

**Owner:** codex.

**Refs:** `docs/manifest.json`, `api/internal/hub/`, `cli/domains/`.

### 2026-08-17 — OT-P0-001 has a missing prerequisite: no push provider resource exists

**Status: RESOLVED 2026-08-18 — the prerequisite was removed, not met.**

OT-P0-001 now delivers through Web Push from this scenario's own installed
progressive web app. No push resource exists and none is needed. The
rejected relay design is preserved at
`.vrooli/resources/blueprints/ntfy.json` at `status: candidate`. See the
2026-08-18 decision rows in `DECISIONS.md`.

### 2026-08-17 — Event ingress blocked on an upstream gap in vrooli-events

**Status: RESOLVED 2026-08-18.** `vrooli-events` now enqueues matching
webhook subscriptions and drains them with retry/signature/health updates;
notification-hub also reconciles its optional subscription at startup and
accepts the actual nested webhook payload shape.

The former upstream gap is retained only as historical context. The active
contract is documented in `concepts/INTEGRATIONS.md` and covered by
`internal/integrations/events_test.go`.

### 2026-08-18 — Three confirmed defects in vrooli-events block event ingress

**Status: RESOLVED 2026-08-18.** The upstream queue, matcher, retry,
signature, and health-writer paths are now present and covered by that
scenario's fan-out tests.

The former fan-out, retry/signature, and health-writer defects were fixed in
`vrooli-events`. This entry remains as an audit trail; notification-hub's
current receiver and startup reconciliation are covered by
`internal/integrations/events_test.go`.

### 2026-08-18 — vrooli-bridge needs a per-scenario change to carry a new verb

**Status: RESOLVED 2026-08-18.** Bridge dispatch vocabulary is now derived
from the shared scope catalog. Notification-hub relays through its own
cataloged `notification-hub notifications relay` command and resolves a
machine's current node lineage through MachineService.

The former per-scenario bridge allowlist was replaced by catalog-derived,
run-eligible vocabulary and paired effect/verb grants. Notification-hub now
dispatches the typed `notification-hub notifications relay` command and
resolves the target machine's current node lineage before dispatch.

**Refs:** `scenarios/vrooli-bridge/api/internal/dispatch/allowlist.go`,
`packages/api-core/scopecatalog/catalog.go`,
`api/internal/hub/bridge.go`.

### 2026-08-18 — Escalation can cross a sensitivity tier

**Status: RESOLVED 2026-08-18.** Each escalation step re-evaluates channel
approval and emits a content-free pointer when the next channel is not
approved for the notification label.

**Symptom:** An unanswered critical ask escalates to the next channel in
the recipient's chain. That channel may hold a lower sensitivity approval
than the channel the first attempt used.

**Root cause:** Design gap found while writing `SECURITY.md`. The
sensitivity policy is evaluated once at routing time, and escalation
reuses the resulting decision rather than re-evaluating per step.

The escalation worker now re-evaluates approval for each step and falls back
to a content-free pointer when the next channel is not approved.

**Refs:** `docs/internal/SECURITY.md`, `api/internal/hub/escalation.go`,
OT-P0-010, OT-P1-011.

### 2026-08-18 — External push acceptance environment is not provisioned

**Status: OPEN — operator/platform action required.**

**Symptom:** The implementation and local suites pass, but the stable
`notification-hub.itsagitime.com` origin does not resolve and no operator
iPhone or paired Mac evidence is available for the required real-device
acceptance.

**Root cause:** tunnel-manager has no Cloudflare account ID, tunnel ID, or API
token configured in this host environment; the remaining acceptance steps
also require direct operator actions on iOS and macOS.

**Workaround:** Provision the three tunnel-manager credentials, reconcile the
managed core route, then install the PWA from the stable origin and record the
push body, timestamp, endpoint prefix, and paired-Mac delivery evidence.

**Real fix:** Complete the operator-owned acceptance for OT-P0-001 and
OT-P0-015 without changing the stable-origin design.

**Owner:** operator/platform.

**Refs:** `OT-P0-001`, `OT-P0-015`, `docs/concepts/INTEGRATIONS.md`,
`tunnel-manager config credentials-status --json`.

### 2026-08-25 — This host has no active user desktop session

**Status: OPEN — operator/session action required.**

**Symptom:** The Linux desktop transport is implemented and reports its
`notify-send`/D-Bus prerequisites, but this host currently has no active
desktop session for the operator account. A direct `notify-send` probe cannot
complete, so a real desktop delivery cannot be claimed from this execution.

**Root cause:** The graphical session is at the display-manager boundary; no
operator-owned DISPLAY/WAYLAND session bus is available to the lifecycle
process.

**Workaround:** Start or attach the operator's desktop session, then configure
a `linux_notification` recipient and send a validation notification. A
headless run remains an explicit unavailable channel, not a successful send.

**Real fix:** Complete the operator-owned Linux desktop acceptance and record
the notification receipt and channel status.

**Owner:** operator/platform.

**Refs:** `api/internal/hub/transports.go`,
`docs/concepts/INTEGRATIONS.md`, Definition of Done item 20.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues


## Work ladder

- Rung: W3
- Evidence: current managed NH13.5 light/dark phone captures use the identity-verified selected originals, 390×844 CSS viewport, zero insets, asserted header top, and byte-matched served build. The header is present; current header weight/color and dark disclosure rail remain visible mismatches. Current audit is in `.vrooli/work/delivery/notification-hub-delivery/evidence/E43-source-baseline-20261025/selected-original-identity-audit.json`.
- Result: focused UI suite passed 76/76; type-check, string/selector freshness and managed freshness passed. All 17 registered current E22 order/quiet cases and the five focused E21 readback journeys have passing durable runs; current combined owner journey `bc619f5d-aaa4-448f-b963-ca1d32dd198a` also passed (150 lease-pool requests, zero primary/test-root writes); the empty-draft case now waits for editor readiness before saved-readback assertions. Its critical-only disclosure and vertical content remain an honest visual mismatch. E43 remains rough with 47 captures, 0 matches, 47 mismatches, 61 unavailable rows; do not infer surface acceptance.
- Measured: 2026-10-25.

## 2026-11-09 — Native consent visibility (implementation ready, activation separate)

Existing observation: `knw-1794241497372849888`,
`bug-inbox/code-defect/key-e822855480c8dcf316f6cfa3`. Agent Manager emits
interactive authorization as FYI; the previous Needs You query only returned Hub
asks. The original pending permission could disappear below Updates' 20-item
window, with no original source link rendered. Push acceptance did not establish
OS display.

The repair adds a read-only source-qualified native consent projection, original
source links in Updates, and a loopback status read in Agent Manager. It does not
create an Ask, answer one, approve, resend, or launch work. See
[Native human consent visibility](../concepts/INTEGRATIONS.md#native-human-consent-visibility).
Source currentness is checked on read; unavailable/in-flight/unknown state remains
visible. Closed-only pages still expose candidate pagination. The browser cannot
use Hub answer controls to grant native permission.

Source/API/UI tests and production build are validated separately from live
activation. A temporary unrelated core-update fence was respected and later
cleared by its owner. The normal Agent Manager lifecycle preserves detached
executors and stages a replacement before stop. That is distinct from restoring
the previous deployed version after the replacement starts.

Recovery qualification at 2026-11-09 18:01 UTC retained and hash-verified the
serving API executable (`8c202b749f4487552cc25e271c697d58fc463f1fc35e3607e8fbe023d61ca06d`)
in `/tmp/froggy-needs-you-recovery-20261109-r5q0e7zc/`, with a matching receipt.
This temporary copy preserves bytes; it is not a qualified or durable rollback.
No Agent Manager baseline engagement exists. The documented
[restore-point route](../../../../docs/operations/manual-scenario-recovery.md#4-restore-code-from-a-copy)
excludes compiled artifacts and captures current source, which already contains
the candidate. It therefore cannot be substituted for an exact prior-build
restore. The owning gap is in `internal/lifecycle`: qualify retained-image
selection and restoration through the existing lifecycle owner, including the
original runtime identities, compatible durable state, and failed-activation
recovery, before claiming that protection. Do not restore shared source by git,
launch the copied executable directly, or spoof build freshness.

The lifecycle owner now has a source implementation of opt-in `capture-runtime`,
`verify-runtime` and explicit retained selection on ordinary restart. It retains
content-addressed runtime bytes and pinned interpreters, hashes rather than copies
service configuration, keeps durable state at its canonical owner, and records
release identity separately from observed source provenance. It rejects unsafe
paths, mutable-directory declarations, arbitrary hooks, script wrappers and
external Node loader overrides. See the
[qualification contract](../../../../internal/cli/scenariocli/runtime-release.md).
No real scenario was opted in or restarted. A declared Node dependency graph is
not proof of arbitrary imports, system-library compatibility or modules an older
process already loaded.

Real isolated lifecycle checks exercise capture, poisoned-source retained restart,
fresh-controller restart and stop/start, explicit source return, canonical-state
preservation and pre-stop refusal of stale selections or tampered bytes. The full
lifecycle/scenario/CLI package suites passed (lifecycle 277.969s); final retained
checks passed after independent review corrections (20.288s). Store/binding race
and clean-environment checks passed. These are fixture qualifications, not an
Agent Manager restore or permission to restart it. The expanded real fixture also
passes recovery after a successfully built replacement stops the old process and
exits with status 27; the retained release returns healthy with unchanged data.
Final verbose clean-environment retained checks passed (22.532s). The official
shared-package baseline completed with 51 passing modules and two unrelated
current failures: Treasury's CLI flag type fails api-core scope-catalog validation,
and `packages/proto/.proto-compatibility-rootowned/` generated comments fail
repo-contract-go's manifest-path audit. Full logs: `/tmp/tmp.Tn5av1k4CT/`.
No pre-change baseline was captured, and these failures are not waived.

Focused hermetic checks passed for dependency-before-stop preservation, real
frozen-source isolation, code-copy round trip and runtime-data exclusion:
`go test ./internal/lifecycle ./internal/baselinefloor -run
'Test(AgentManagerRestart|EffectiveSourceDirRealFloorIsolatesContent|CaptureRestore_RoundTrip|RestoreDoesNotOverlayStaleRuntimeData|MaterializeServing)'
-count=1`. These are source-contract tests, not a live rollback drill.

At 2026-11-09 18:34 UTC, Agent Manager was healthy on the unchanged build
`sha256:3cce151d6386a059683c74edaf32c85384b103c160089ae64312070326ef2cb8`.
The source status route still returned HTTP 404; the deployed repair remains
unverified. Fence-only observation was open at revision 116 and explicitly did
not observe inventory. Full inventory still returned HTTP 503 with
`physical executor inventory is incomplete`: attached run
`e4e83e6a-e035-4890-a3bd-b3340bd95f89`, PID/group `1748555`, lacked run identity or
reuse proof. The earlier 46-session observation is not a fresh complete count.
Narrow read-only follow-up identified PID 1748555 as `opencode serve`, child of
Web Console API PID/PGID 1747770. Its environment identifies runtime instance
`inst-41c0b39f232a52b6eb24c001eb9af15d`, generation 1161; the canonical runtime
registry agrees that this is the running Web Console instance, with API start
2026-11-09T15:09:48.552Z. This is evidence of a shared backend owner, not proof
that the attached run is absent or finished. The existing executor-scope owner
must reconcile shared-service ancestry and the exact attached-session lease;
do not manufacture a run identity or stop that server. Agent Manager's mixed
runtime/state contract also needs its own restore qualification before the
requested activation boundary is satisfied.
Identity qualification at 2026-11-09 19:05 UTC found the precise owner mismatch:
the launcher keys the native lease by immutable harness session
`opencode-aec620dc4c27662039c9eac3ca00e64e`, while the inventory joined only the
Agent Manager run UUID. Agent Manager now hydrates and forwards the optional
attached harness identity; the control plane shares its canonical-first selector
with the existing run-binding proof. Legacy run-ID fallback occurs only when the
canonical key is genuinely absent. Bounded exact-key reads include stopped and
expired rows, avoiding false absence from an active-only list. Native membership,
exact PID/harness, stable boot/process incarnation and final lease re-resolution
remain required. A canonical registration during legacy proof cannot be bypassed. The updated
consumer requires the `attached-key-v1` identity capability when it sends harness
anchors; an older reader that silently ignores unknown fields is refused.

The uninstalled candidate's read-only observation of 36,302 fixed source references
proved the target present at PID 1748555, not completed or absent. It returned 25
present, 1,338 absent and 34,939 unknown references. The reference set was read at
19:00:45 UTC and the candidate report at 19:05:56 UTC; it is not the deployed API's
current complete inventory. Unknowns comprised 10,378 missing exact lease keys,
24,536 absent leaders without established containment provenance, 20 process
births outside their run intervals, and five PID/harness conflicts. The launcher
historically recorded a planned scope before best-effort containment; names and
terminal statuses cannot retroactively prove that descendants were contained.
All sampled selected leases lacked containment-method evidence. Newly anchored
historical rows therefore remain unknown. The old unanchored gone-scope policy
is a separate pre-existing residual, not certified by this change. No history was
deleted, reclassified, reattached or supplied with invented identity evidence.

Source follow-up at 2026-11-09 19:40 UTC adds durable per-launch binding
provenance in the launcher/runtime owner. Native evidence is recorded before exec
or spawn wait; late placement, failed setup, missing incarnation and vanished
scopes cannot become successful containment proof. Separate intended/established/
failed/unavailable and active/superseded/terminal states preserve attempt history.
Binding-fenced callbacks, legacy-only heartbeat/stop guards, and BindingID-aware
proof snapshots prevent stale attempts from changing or masking replacements.
The ledger is not yet consumed as descendant-absence proof; historical gaps still
need independent evidence, not replay, current-only filtering or tombstones.

Schema 16 preserves old rows without invented bindings, but pre-16 readers reject
it and already-open old binaries retain unfenced SQL. This requires a coordinated
runtime-owner rollout, not automatic migration during a read. See
[containment provenance](../../../../docs/reference/agent-sessions.md#containment-provenance).
Only disposable stores were migrated. Full cli-core/runtime-store and downstream
maintenance/agentsession/CLI suites, native evidence and race tests, and populated
schema-16/current versus version-15-overlay compatibility checks passed. Logs:
`/tmp/launcher-binding-*.log`, `/tmp/binding-final-downstream.log`, and
`/tmp/editor-provenance-schema-qualification/`. The existing ambient model-policy
argument test passes with temporary VROOLI_HOME/CODEX_HOME; no user configuration
was changed. The positive identity repair remains source-tested, not permission
to activate with incomplete inventory.

Read-only mixed-runtime qualification at 2026-11-09 19:07 UTC confirmed the
original API/UI processes healthy on the unchanged build, and API executable
bytes still match the retained temporary file. The lifecycle's scenario identity
and `SCENARIO_DATA_DIR` preserve the actual scenario-local SQLite owner under
relocated source; the home-root fallback DB is not the serving database. This
proves routing only, not backward-compatible state or recovery behavior.

Actual retained closure remains unqualified for three source-backed reasons:
`internal/lifecycle/retained_node_runtime.go` treats explicitly optional peer
dependencies as required; declaring all UI `node_modules` includes generated
proto code directories named `credentials`, which the capture owner's secret-path
filter rejects; and Agent Manager configuration/receipt roots fall back to
`../../..`, which changes under retained cwd. `PROJECT_ROOT` is absent from the
serving environment, and `wiring/receipt_runtime.go` uses the relative root even
if that variable were supplied. UI disk mtimes precede startup but cannot prove
the exact JavaScript modules already loaded by the running process. Next use
isolated lifecycle-owner tests for production UI packaging, optional-peer
semantics, canonical configuration roots, storage compatibility and actual mixed
recovery; do not loosen secret filtering or label a copy as a proven prior build.
Fresh deployed fence read remained open, revision 116, with inventory explicitly
unobserved. Full admission still returned 503 and the native consent status route
404. No real runtime descriptor, capture, selection or restart was performed.

No activation, drain, permission change or test push was performed. Coordinate
the remaining runtime recovery work without enabling a dormant team or bypassing
lifecycle fences. The source status and actual Needs You UI still require
post-activation readback; phone OS display remains a separate operator check.

Endpoint metadata generation also exposes prior unbound AppDelivery and channel
address RPCs in `cli/manifest.json`. The new browser-only native consent read is
declared there. This existing CLI coverage gap is not silently waived.

Validation receipts for this bounded repair: formal dependencies/structure/quality
passed in `20261109-164627-1a46f059`; final formal unit passed in
`20261109-165854-fda11dfe` after correcting the changed queue-copy assertions and
regenerating CLI primitive evidence through its own test. Full UI: 297/297;
coverage-enabled UI, type checks, scoped lint, protobuf verifier, focused race
checks, and source/API builds passed. Remaining maturity warnings are not waived.
The optional Agent Manager read dependency is reconciled in the PRD and service
manifest through SDA's `deps declare-runtime` owner route. The existing detector
misses the injected resolver call and its inferred optional-edge writer defaults
to `try_start`, so a selected inference scan made no changes. The explicit local
owner route instead fixes `required: false`, `runtime_only: true` and
`startup_policy: ignore`, requires rationale and degraded behavior, and applies
only the selected edge after a digest-checked preview. It neither calls nor
starts Agent Manager and does not grant permission. The applied manifest digest
is `b9ad19e0b0126b9f0f877ecb4f7500fb2961b4fdf47918c2451126d72fb5b230`.

Validation: all SDA CLI packages passed `go test ./... -count=1`, including
isolated declaration tests for preview/apply, no-API registration, byte-preserving
unrelated metadata, repeat idempotence, conflict and stale-digest refusal,
symlinks, missing fields and duplicate keys. Notification Hub dependencies and
structure passed in `20261109-181031-eb50d2f2`; SDA structure and quality passed
in `20261109-181046-a2486339`. These focused phases retain their reported
maturity gaps and do not establish live activation or device delivery. Existing
AppDelivery/channel-address CLI gaps remain unchanged and unwaived.

Activation, 2026-11-10T03:59Z: the operator asked for the repair to be switched on. Agent
Manager was already serving the status route. `vrooli scenario restart notification-hub`
completed healthy in 41 s, and `ConversationsService/ListNativeConsents` on the restarted API
returned one pending consent (`99cfa727-4dad-4844-9f33-dd51550d6dc5`, state `pending`, with its
Agent Manager review URL). Not verified: the Needs You list as rendered on the operator's
phone, a request filed by a new Web Console Codex session, and any rollback path (the
displaced previous binary is not a qualified rollback).


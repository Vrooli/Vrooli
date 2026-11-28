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

Use this section for deferred findings from `architecture-lenses`.
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

- Rung: W3 (scoped overlapping-decision safeguard, 2026-11-16). Existing subject-key handling updates the open decision or replaces it when its menu changes, but packet checks did not flag a similar open decision under a different key. Added advisory matching scoped to the canonical ask-list audience; similarity never merges choices or transfers authority. Focused subject and related-decision regressions pass; scoped unit phase is recorded separately.

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

## Work ladder — decision links from the phone (2026-11-12)

- Rung: scoped W3 in decision reading and response affordances; expected behavior is
  established by operator feedback F2. No W0–W2 readiness claim.
- Evidence: AskPage rendered question, recommendation, and context prose as plain text;
  URLs embedded there were not activatable. Updates already matched notifications to exact
  returned asks and kept the feed available on lookup failure. AskPage now renders safe
  HTTP(S) Markdown/bare links as 44px targets and Updates action links use the same target
  height. Focused UI validation passed (2 files, 39 tests). Physical-device acceptance and
  full scenario unit validation remain open; see the delegation return for current evidence.

### 2026-11-12 — Decision links can open local files in Web Console

**Status:** IMPLEMENTED; phone acceptance and scenario unit suite remain unverified.

Notification Hub decision text now recognizes local file references and sends them
to Web Console's existing file viewer through a same-origin, allowlisted external URL
lookup. Web Console consumes `?file=` only after workspace hydration and an active
session, then opens through its existing preview controller. The links remain inert
when URL resolution fails and expose a retry action. URL links remain restricted to
HTTP(S); no user information is included in the lookup response.

Focused Notification Hub tests and type checking pass. The formal `unit` Test Genie
phase still fails at `pnpm run test:coverage`; details and prior unrelated UI test
failures are in the delegation return. A real phone read-and-open check is still
required before calling the operator-facing acceptance complete.

**Owner:** notification-hub / web-console.

**Refs:** `api/handlers/externalurl/`, `ui/src/features/asks/DecisionText.tsx`,
`scenarios/web-console/ui/src/hooks/useFilePreviewDeepLink.ts`.

### 2026-11-13 — Decision context and mobile capture recheck

The phone decision page now opens the plain-language “Why this matters” section by
default while keeping technical details optional. Notification Hub AskPage tests pass
30/30; Web Console file-link tests pass 5/5; both UI type checks and the external URL
handler test pass. Test Genie unit runs `20261113-041041-3144fd31` and
`20261113-042415-93f85cfe` both failed at UI coverage (`pnpm run test:coverage`);
the first reported 296 passed and six failures, including React Router's
`RequestInit` AbortSignal mismatch. Bug report: `knw-1794543157607928864`. The focused
AskPage test includes a labeled answer group and passed 30/30. Web Console unit run
`20261113-041641-3947f907` also failed in unrelated API/UI suites; focused file-link
tests passed 5/5.

Workflow Health run `ef988e46-5956-4370-a7b9-d9b8e49aded9` executed one phone-sized
read of the real pending ask `91df4be1-58a3-4192-9ae8-b69fb5ed4e97`; capture is retained
at the delegation's `mobile-real-decision.png` (390×844 CSS px). The local capture is
signed out, so it shows the question and More detail link but no answer controls; the
steward's authenticated phone check must verify choices and completion. The one-off
capture case was removed after use. The Workflow Health static assessment reported
`workflow.registry_stale` while that temporary case existed; its selected execution
passed 1/1.



### 2026-11-14 — Shared `file:` packages (file-viewer) cannot be adopted through the governed path

**Status: open for the governed path; Notification Hub itself is adopted and working.**

**Today's state:** Notification Hub's UI depends on `@vrooli/file-viewer` (`file:../../../packages/file-viewer`), with the governed installs of `xlsx`, `mermaid` and `jsdom` 27 done, so the viewer, the spreadsheet and Mermaid engines (lazy chunks), and the jsdom-based host tests all run. Nothing in the host needs a workaround any more.

**What is still open:**
- `scenario-dependency-analyzer deps install npm/@vrooli/file-viewer@file:... --apply` was denied by the closure probe (bounded pnpm closure and registry-metadata budget), and an earlier governed `--apply` into Web Console emptied its `node_modules` because the analyzer's pnpm store (`store/v10`) differs from the lifecycle store (`store/v3`). The first adoption therefore used one operator-approved lifecycle pnpm install. QA bugs: `knw-1794618858815944773`, `knw-1794619329121707178`. This agent did not verify that the analyzer fix has landed.
- `vrooli package refresh file-viewer --no-restart` used to abort without a TTY (`ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`) because the package's `node_modules` was linked against a different pnpm store than the lifecycle selected. Fixed in the control plane (`internal/packagegov/pnpm_store.go`, `internal/lifecycle/phases.go`: lifecycle installs follow the store `node_modules` is already linked to, and fail clearly instead of purging). The installed `vrooli` binary is stale until the operator rebuilds the CLI; until then the refresh runs through `go run ./cmd/vrooli package refresh ...`. Consumer setup stays deferred until their next coordinated restart.

**Real fix:** the analyzer adopts and refreshes `file:` packages through the same store and install profile as the lifecycle, with closure sizing from a staged lock resolution, and the rebuilt CLI is installed; then this entry is deleted.

**Owner:** scenario-dependency-analyzer owner (adoption); the operator installs the rebuilt CLI.

**Refs:** `packages/file-viewer/README.md` (Consumer setup), `scenarios/scenario-dependency-analyzer/api/internal/installgateway/pnpm_closure.go`.

### 2026-11-14 — Senders are not authenticated as services: the event path trusts a self-asserted source, the direct path has no service-principal route

**Symptom:** Any process that can reach vrooli-events can raise a notice or decision ask in the operator's inbox as any source scenario, and can withdraw that source's pending asks. A Treasury (or any other) service credential presented on `NotificationsService.Send`, `ConversationsService.Ask` or `Relay` is refused with 401 rather than admitted as that service.

**Root cause:** Three ingress paths, three trust models.
1. Bearer endpoints (`/api/v1/internal/provider-decisions/{lookup,...}` publish and lookup, and ask reads) are correct: audience, subject, scope and `Verified` are all checked, publish pins the source to the principal (`ProviderDecisionSubject`, never the body), and ask reads bind the audience to the source named by the principal (`askreader.SourceOf`).
2. The signed event webhook (`/api/v1/integrations/events`, where every scenario's `*.notification.v1`, `*.decision_requested.v1` and `*.decision_withdrawn.v1` arrives) checks one shared HMAC secret. That proves "vrooli-events delivered this", not who published. `source_scenario` is whatever the publisher wrote into the envelope: vrooli-events `handleIngest` has no caller authentication (only the agent-run provenance middleware, used for receipts). The only source the Hub refuses to take from the bus is the provider-decision one (`TestEventWebhookRejectsProviderDecisionEvenWithSharedHMACAndClaimedGatewaySource`). Withdrawal is scoped to the claimed source (`TestWebhookGenericWithdrawalAcceptsAnySourceByOwnSubjectOrRef`), which is only as strong as the claim.
3. Direct Connect writes (`Send`, `Relay`, `Ask`) resolve the producer with `ProducerForDirect`: owner login, verified agent run, or no credential at all (shared `source:anonymous` bucket). The owner verifier expects audience `scenario-authenticator:default` and kind human, so a service token (audience `scenario-authenticator:<resource>`) fails verification, and because a credential was presented it is refused instead of downgraded to anonymous. Treasury's funding notice (`walletops.NotificationHub.Received`) sends exactly such a token, so it cannot succeed even once its credential is issued.

**Status 2026-11-14 (later):** every consumer now confirms an operator's answer with the Hub before applying it (`operatordecision.Webhook` reads the ask back as its own ask-reader), so a forged `notification_hub.ask.resolved.v1` cannot approve anything for a consumer whose credential is issued; until the operator runs `service-principals sync --apply`, consumers apply the body and count it as unverified. For the request direction, vrooli-events now classifies and counts the publisher proof of every accepted event (`GET /api/v1/publisher-authentication`, observation only, nothing refused), `eventbus.Client` presents each source's `<source>-event-publisher` identity once a scenario registers `credential-authority-go/eventpublishing` and its credential is issued, and 12 publishers declare theirs. Enforcement per high-consequence family, then everywhere, is not done and is the operator's decision from the observed counts.

**Fixed 2026-11-14 (service tokens reached the handlers):** in the production configuration (Cloudflare Access the only owner provider) `OwnerAuthenticator` handed every request, `Authorization` header included, to api-core `authn.Config.Authenticate`, which refuses any request carrying an `Authorization` header when no provider other than Access exists. So every service token (ask-reader reads, sender sends) and every owner bearer was refused as a failed owner login before the handler could verify it; Research Desk's `ListAsks` failed `unauthenticated` from the moment its ask-reader credential was issued (15:14Z), and the handler tests missed it because their fixtures mount no owner middleware. `OwnerAuthenticator` now shows its providers the request without the bearer (`withoutBearer` in `internal/identity/access.go`); the handlers verify the bearer as before. Needs a Hub restart to take effect.

**Status 2026-11-14 (packets):** a service sender now also stores files (`StatAttachmentBlobs`, `PutAttachmentBlob`) and closes its own ask (`Withdraw`, by id or subject key; before this a sender's ask could never be withdrawn, because the withdrawal compared the ask's source with the recipient). Research Desk sends its decisions this way, with explainer pages; its credential awaits the operator's `service-principals sync --apply`.

**Status 2026-11-14:** path 3 is fixed. A `<source>-notification-hub` principal (api-core `hubsender`) is admitted on `Send`, `Relay` and `Ask` as its own source, and a verified service token that is not a sender is refused with the mismatch named, never downgraded to anonymous. Treasury still has to declare its principal. Paths 2 and the idempotency note below remain open.

**Fixed 2026-11-14 (context links):** direct `Send` and `Relay` refuse an unsafe `context_url`, and the event path drops one (stored without the link, counted per producer, logged) rather than reject the notice. A sample of the live store (4,120 notifications, 183 asks, 22 revisions; 640 distinct links) found every event-sourced link valid; the only 6 that fail are direct asks from the operator's own agents (`file://` paths and a relative path without a leading slash), which the direct `Ask` path deliberately still accepts, with an advice warning; `Ask` refuses only `javascript:`, `data:` and `vbscript:`. The drop count is not yet on any HTTP surface; it is in the log and `integrations.DroppedContextLinks()`.

**Workaround:** None needed for the provider-decision and ask-read paths. For notices, senders that must reach the Hub today do so over the event bus (unauthenticated source) or without a credential (anonymous bucket).

**Also open:** the notification-level idempotency key `(recipient, key)` is shared by every direct producer, so one sender's key can collide with another's (`already bound to different immutable terms`); senders prefix their keys with their own name until the Hub scopes it by source.

**Real fix:** (a) Bind publisher to `source.scenario` where the principal is still known, in vrooli-events ingest (or have it attach a verified publisher identity that the Hub compares with the claimed source); the Hub cannot do this alone because the bus replaces the sender with its own HMAC. (b) Add a service-principal producer route on the Hub's direct writes: a verified service token with the Hub audience and a send scope is admitted with the source taken from the principal name (the ask-reader convention), never from the request; a mismatch is refused naming both. (c) Mark the Hub's high-consequence sources (agent-manager authorizations, storage deletion grants, Treasury approvals) as requiring (a) or (b), as provider decisions already do.

**Owner:** unassigned; (b) is Notification Hub, (a) and (c) need vrooli-events.

**Refs:** `api/internal/integrations/events.go` (`EventWebhookFromSecretSourceWithDecisionDefaults`, `validSignature`), `api/internal/identity/identity.go` (`ProducerForDirect`), `api/internal/identity/reader.go` (`verifiedServiceReader`), `api/internal/identity/provider_decision.go`, `scenarios/vrooli-events/api/handlers.go` (`handleIngest`), `packages/api-core/owneridentity/owneridentity.go` (`ExpectedAudience`), `scenarios/scenario-authenticator/docs/internal/PROBLEMS.md` (Treasury principals).

### 2026-11-14 — Identityless host notification and decision requests were refused

**Symptom:** The steward's systemd timer reported the exact error
`unauthenticated: a verified agent run or verified owner identity is required`
when sending a notification; decision requests from a plain host shell were
also expected to work without a permission step.

**Root cause:** Direct sends and asks must not depend on an agent run, owner
session, or sender principal. Identityless producers resolve to the configured
recipient and the shared `anonymous` push-budget source. A caller that supplies
an invalid credential is still refused; identityless access grants no read,
answer, dismiss, preference, or device-management rights.

**Resolution:** Notification Hub's existing greenfield core owns the change
(`execute/notification-hub-greenfield-core`). The direct Send/Relay and Ask
handlers accept identityless requests; the per-source device-push bucket stores
over-limit requests and suppresses only device pushes. Regression coverage now
exercises the host caller on both notification and ask paths. `NEEDS-YOU.md`
and `INBOX.md` remain the steward's working fallback until a real approval is
linked and its result is actionable end to end.

**Owner:** Notification Hub (`execute/notification-hub-greenfield-core`).

**Refs:** `api/handlers/notifications/module.go`,
`api/handlers/conversations/module.go`,
`api/handlers/notifications/module_producer_test.go`,
`api/handlers/conversations/module_producer_test.go`,
`docs/reference/configuration.md`.

## Work ladder — repeated storage and producer logic (2026-11-14)

- Rung: W3; contract and obligations unchanged.
- Evidence: focused Go tests passed for identity, hub, preferences, recipient,
  conversation, notification, delivery, and dbvalue packages. Test Genie run
  `20261114-081726-47cf595f` confirms no duplicated-code finding, while its
  unrelated UI projection checks and existing long-file budget still fail.
- Result: direct producer resolution, escalation-chain row loading, and
  notification get/state/receipt loading now each have one implementation.
  The separate ensure-recipient and recipient-scoped read rules remain explicit.

### Follow-up — event ingress and unused transport scaffold (2026-11-14)

- Rung: W3; contract and requirements unchanged.
- Result: neutral event facts are decoded once and the unused `httpc.Doer` /
  `FakeDoer` scaffold was removed; no production code used it. Locale labels,
  directions, and bidi formatting now share the side-effect-free locale config.
- Evidence: the full API `go test ./... -count=1` suite and full UI `pnpm test`
  suite passed after `TestRecipientReadViewsDoNotCrossRecipients` stopped the
  startup recovery worker before inserting its read-only fixtures. UI passed
  483 tests; `pnpm run type-check` also passed. Test Genie run
  `20261114-101655-41ea8b9f` did not reach a phase before its one wait timed out
  in `preparing:provider_readiness`; the phase-less run was then explicitly
  aborted. Reports are `knw-1794651212521488486` and `knw-1794651641400210129`.

## Work ladder — warm Android app links (2026-11-15)

- Rung: scoped W3; operator feedback F130 establishes expected routing. No W0–W2 readiness claim.
- Evidence: Android Moto G7 was available; a deep link to the live Hub URL passed on device but opened in Chrome. The Hub PWA was not installed on this target, so its warm-PWA path could not be physically reproduced. Before the fix, the focused regression failed because no launch-queue consumer was registered.
- Result: the PWA now requests reuse of an existing client and routes validated Android launch-queue URLs through the existing same-site navigation guard. Focused UI validation passed (48 tests across 2 files), type-check passed, and the manifest parses as JSON.
- Measured: 2026-11-15.

## Work ladder — in-Hub linked artifact review (2026-11-15)

- Rung: scoped W3; existing decision behavior and ownership contract retained.
- Result: decision text paths under configured local roots open in the existing
  Hub attachment viewer. Reads require the ask owner, reject symlink escapes and
  non-regular files, and cap previews at 64 MiB. Missing/unavailable sources
  report a recoverable viewer error. The Web Console URL handoff was removed.
- Evidence: focused UI tests (103 tests), UI type-check, focused Go resolver and
  owner-route tests, endpoint generation, and Test Genie unit run
  `20261115-111450-cec587f6` passed. A real live ask was unavailable in the
  owner context, so mobile capture of a real decision remains for the steward.

## Work ladder — operator steering Hub host (2026-11-16)

- Rung: scoped W3; the supplied steering contract and obligations were not
  changed.
- Result: producers can send a status or decision with a stable kit name and
  JSON data. The API validates against the kit schema, embeds escaped data in
  the single `hub-data` slot, and stores the result as the ask's sealed explainer.
  The host exposes kind and subject-history recurrence to the page bridge, stores
  page selections with either answer kind, and supports notes and image answers.
  Status updates have no deadline and can be marked read without an answer.
- Evidence: Go renderer tests accept all four published samples and reject bad
  schema data; `TestStatusUpdateStoresSteeringImagesAndSubjectRecurrence` covers
  template delivery, recurrence, steering, image upload/readback and Mark read.
  UI type-check and 111 focused packet tests passed. Production UI build passed.
  Test Genie unit run `20261116-174553-d70e90b2` passed; its findings retain two
  warning classes, including `MISSING_INJECTABLE_SEAM`, and 40 existing low
  coverage file findings. Protobuf freshness verifier passed.
- Visual review: `notification-hub pages shoot` produced phone-width light and
  dark previews for all four kit pages (plus 320 px usage-mode previews). These
  render the kit's clearly marked illustrative sample JSON outside the live Hub
  bridge; they are visual evidence only, not producer data or delivery evidence.
- Remaining: H recorded the kit `READY` marker in shared STATUS, and the Hub
  host resolves those published files. N has no live producer-to-Hub operator
  delivery or answer/readback identifier. A/S/G own those producer integrations.
  A live sample page still needs a host-context visual pass after the shared
  integration run.

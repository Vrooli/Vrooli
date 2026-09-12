# Problems — Switchboard

## What belongs here

Known defects, unowned gaps, and load-bearing assumptions that have not been
verified. An entry here is a claim someone can falsify. Specifically:

- A defect in this scenario, or in something it depends on, that affects it.
- A capability this scenario needs that has no owner anywhere.
- An assumption that would change the design if it turned out to be wrong.
- Architecture drift: the code and the documents disagreeing.

## What does NOT belong here

- Unbuilt features. Absence of implementation is not a problem; it is
  `PRD.md` and `requirements/`. This scenario has no domain code at all, and
  that is a state, not a defect.
- Ideas and wishes — those are operational targets.
- Anything already fixed — move it to the resolved section with its evidence.
- Bug reports against *other* scenarios that do not affect this one. File those
  to `scenario-qa` through the `report-bug` skill.

## Entry template

```
### <ID> — <short title>
- **Status**: open | mitigated | resolved | accepted-risk
- **Severity**: blocker | high | medium | low
- **Affects**: <domain, surface, or dependency>
- **Symptom**: what is observably wrong
- **Impact**: what it costs, and to whom
- **Evidence**: file, line, command, or measurement
- **Blocked on**: what would resolve it
```

## North Star

Switchboard keeps one explicit channel contract across every registered adapter. A provider finding must identify a concrete contract or evidence gap, and every implemented messaging slice must have runnable evidence at the boundary it claims to serve.

## The rungs and their gates

- **L0 — Unavailable**: the shared channel contract cannot run.
- **L1 — Runnable**: the provider can enumerate descriptors and execute the shared cases.
- **L2 — Conformant**: every registered channel passes all shared cases with no required findings.

## What each finding means

Errors block the next conformance rung and require a repair. Warnings identify advisory drift or an ungated transition. An accepted risk remains visible here with its owner and evidence instead of being treated as a passing implementation claim.

## The canonical fix

Repair the descriptor, adapter, fixture, or provider declaration at its source. Do not add a channel-specific branch to shared dispatch or weaken the common contract to accommodate one adapter.

## How to verify

Run `test-genie execute switchboard --phases channel-conformance --json`, then inspect the server-owned result and its presentation. Supplement it with focused adapter tests and the live lifecycle checks recorded in the Work ladder below.

## Entries

### SWBD-PROB-001 — Runtime injection defence has no owner
- **Status**: open
- **Severity**: blocker (for any non-owner exposure)
- **Affects**: `trust`, `turns`, and the whole product promise of giving out a handle
- **Symptom**: The trust ceiling holds — an injected instruction cannot exceed the tier's scope. Nothing detects or resists manipulation *within* a permitted scope.
- **Impact**: Exposing a handle to anyone but the owner is an explicit accepted-risk decision rather than a defended default. This is the gate on the scenario's most monetisable capability.
- **Evidence**: `scenarios/prompt-injection-arena/` is a flat pre-`api-core` `api/*.go` layout with a 12.9 MB binary committed 2025-10-28 and a `PROBLEMS.md` self-declaring "Production-Ready" since that date. Its README describes an injection library, agent robustness scoring, and competitive leaderboards — an offline tournament, not a runtime guard.
- **Blocked on**: A redesign of `prompt-injection-arena` from its own operational targets and technical requirements, or a new owner for runtime defence. Budget it as new work; it is not a dependency that already exists.

### SWBD-PROB-002 — `audio-tools` never sends `engine_id`, so speech runs on CPU
- **Status**: open (in a dependency)
- **Severity**: high (blocks call mode)
- **Affects**: `channels` (call mode), OT-P2-002, OT-P2-003
- **Symptom**: Browser dictation sessions never transmit an engine selection, so every session falls back to CPU transcription while every health signal reads green.
- **Impact**: Call mode cannot be described as a voice product on top of this. A latency budget written against GPU transcription would be wrong by a wide margin.
- **Evidence**: Recorded operator measurement, 2026-08-31.
- **Blocked on**: The `audio-tools` engine-selection fix. This is a prerequisite for the P2 voice targets, not a detail to discover later.

### SWBD-PROB-003 — Address identity is an assertion, not a proof
- **Status**: accepted-risk
- **Severity**: medium
- **Affects**: `trust`
- **Symptom**: Every channel reports a sender address that the transport asserts. SMS sender identity is spoofable at the network level; other channels vary in strength and none is a cryptographic proof.
- **Impact**: A tier assignment is only as strong as the channel's identity guarantee, and the system cannot tell the operator how strong that is.
- **Evidence**: Property of the transports, not of this code.
- **Blocked on**: Nothing — mitigated by policy. No tier above `known` should be assigned to an SMS-only contact without out-of-band confirmation. A per-channel `identity_strength` field in the descriptor would let the console state this rather than leaving it to operator memory.

### SWBD-PROB-004 — iMessage ingress requires broad access to the owner's message store
- **Status**: open
- **Severity**: high
- **Affects**: `channels` (iMessage adapter), OT-P1-010
- **Symptom**: Apple provides no supported inbound interface, so ingress means reading the local message store, which contains every conversation on that Mac — including many with no relationship to any agent.
- **Impact**: A privacy surface far wider than the feature needs.
- **Evidence**: Absence of any Apple-supported inbound API.
- **Blocked on**: A design obligation on the adapter slice: scope the read to bound threads and never materialise anything outside them. This is a requirement of the slice, not an optimisation to consider afterwards.

### SWBD-PROB-005 — Local spend ledger can drift from the LPBS wallet
- **Status**: open
- **Severity**: medium
- **Affects**: `turns`, OT-P1-014
- **Symptom**: Per-thread cap enforcement reads a local mirror of metered spend; LPBS is the authority. Between reconciliations the mirror can under-count.
- **Impact**: A thread can exceed its cap within the drift window.
- **Evidence**: Design consequence of local-first cap enforcement.
- **Blocked on**: A reconciliation interval and a stated maximum drift, decided when metering is built. The alternative — a network round trip per turn — breaks the product on a bad connection and is worse.

### SWBD-PROB-006 — No admission limit below the thread spend cap
- **Status**: open
- **Severity**: medium
- **Affects**: `turns`, `trust`
- **Symptom**: An unknown sender cannot exceed conversation scope but can consume metered inference until the thread cap trips.
- **Impact**: A cheap denial-of-wallet against a published handle.
- **Evidence**: No per-address admission control in the current design.
- **Blocked on**: A per-address rate limit sitting below the thread cap, keyed on the contact rather than the thread.

### SWBD-PROB-007 — Reference documentation still describes the template example domain
- **Status**: resolved
- **Severity**: low
- **Affects**: `docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`, `docs/internal/TESTING.md`, `docs/internal/SEAMS.md`
- **Symptom**: These documents originally described the generated `notes` example domain because no real domain existed yet.
- **Impact**: A reader could have mistaken the example surface for the product surface.
- **Evidence**: `template-manager detemplate switchboard --json` completed on 2026-09-01 and reported no remaining `notes` example-domain residue; `template-manager orient switchboard` reports all 9/9 gates complete.
- **Blocked on**: Nothing. Planned-surface sections remain intentionally generic until product domains land.

### SWBD-PROB-008 — Requirements traceability reads complete against stubs
- **Status**: open (expected)
- **Severity**: low
- **Affects**: `requirements/`
- **Symptom**: `vrooli scenario requirements validate switchboard` passes and reports complete registry, intent linkage, and evidence traceability, while every requirement's validation entry is a `manual` stub reading "replace with a test-typed validation once the behavior exists".
- **Impact**: A green validation here must not be read as evidence of working behavior. It is evidence of a complete, well-linked *claim set*.
- **Evidence**: `requirements/01-must-ship/module.json` — every entry has `"type": "manual"`, `"status": "planned"`.
- **Blocked on**: Real tests tagged `[REQ:SWBD-*]`, arriving with each domain slice.

### SWBD-PROB-009 — The `react-vite` template's docs manifest requires headings its own files do not have
- **Status**: open (defect in `template-manager`, not in this scenario)
- **Severity**: low
- **Affects**: `docs/manifest.json` validation for `docs/reference/api-endpoints.md` and `docs/reference/cli-commands.md`
- **Symptom**: The manifest declares `requiredHeadings` of `Notes (CRUD reference)` and ``Scenario commands — `notes` (CRUD reference)``. The generated files instead carry ``## Domain endpoints — `<domain>` `` and ``### Example domain — `notes` (removed by `template-manager detemplate`)``. No generated scenario can satisfy these two rules as shipped.
- **Impact**: Two docs-contract rules are unsatisfiable from generation onward. Worse, both required headings name the example `notes` domain, which `template-manager detemplate` is supposed to remove — so satisfying them would conflict with completing the `example-domain-removed` orientation gate.
- **Evidence**: `templates/scenarios/react-vite/docs/reference/api-endpoints.md:55` is `## Domain endpoints — \`<domain>\``, while `templates/scenarios/react-vite/docs/manifest.json` requires `Notes (CRUD reference)` for that same path. Verified against the template source, so it is present in every scenario generated from it, not introduced here.
- **Blocked on**: A fix in the template: either relax both rules to the domain-generic headings the files actually use, or drop them, since a required heading that names the deletable example domain cannot survive `detemplate`. Not filed as a bug report yet — reported to the operator pending their decision.

### SWBD-PROB-010 — A declared Class B meter has no limit key to enforce it
- **Status**: open
- **Severity**: low
- **Affects**: `docs/business/MONETIZATION.md`, plan differentiation
- **Symptom**: The packaging table declares agent count and channel count as a gated Class B (local-capacity) meter, but `packages/monetization-go/meter-inventory.json` generates only `ai_credits`, `voice_minutes`, and `workflow_executions`. None counts agents or attached channels, so the row states an intent with no mechanism behind it.
- **Impact**: If it is discovered during wiring rather than now, either a limit key gets invented locally — which the paid-features contract forbids, since the generated registry is the shared source — or the row is quietly dropped after being used to reason about plan tiers.
- **Evidence**: `packages/monetization-go/meter-inventory.json` enumerates three limit keys; `grep` for an agent- or channel-count key returns nothing.
- **Blocked on**: A decision before implementation — add a limit key to the generated inventory, or drop the row and let plan differentiation rest on the gated capabilities alone. Dropping it is legitimate: Class B enforcement is a nudge by design and revenue integrity comes from Class A regardless.

### SWBD-PROB-011 — Capability gate REST identity is not authenticated
- **Status**: resolved
- **Severity**: blocker (was; production owner approval now uses authenticated identity)
- **Affects**: `gates`, owner-only approval
- **Symptom**: Resolved. Gate mutations derive the owner from the shared `owneridentity.Validator`; body identifiers are consistency checks only.
- **Impact**: The REST boundary no longer trusts a caller-supplied owner identifier for authorization.
- **Evidence**: `api/internal/owneridentity` integration in the gate handler and focused authenticated/unauthenticated handler tests; production wiring uses scenario-authenticator, verified 2026-09-01.
- **Blocked on**: Nothing for the authenticated REST boundary. In-thread gate API integration remains a separate open scope item.

### SWBD-PROB-012 — LPBS metering has no production call path
- **Status**: open
- **Severity**: high
- **Affects**: hosted inference billing, OT-P1-014
- **Symptom**: Switchboard has a tested reserve/execute/finalize/release seam, but no concrete LPBS client is wired into agent-manager execution. LPBS's reservation endpoints require an authenticated user token and the Switchboard dispatch path does not yet carry one.
- **Impact**: Hosted inference cannot be claimed as metered. A local seam alone cannot charge the LPBS wallet or prove entitlement enforcement.
- **Evidence**: `api/internal/metering/meter.go` is only called by package tests; `api/main.go` creates no LPBS gateway; LPBS routes are authenticated at `/api/v1/usage/reservations`.
- **Blocked on**: A defined authenticated identity/token handoff at the Switchboard → agent-manager → LPBS boundary. Do not invent a shared service credential or silently charge a guessed user.

## Architecture Drift

Divergences between what the documents assert and what the code does. Checked
against the orientation gates.

| Drift | Documents say | Code says | Resolution |
|---|---|---|---|
| Domain implementation | Five domains with a fixed build order | Only the template's `notes` example exists | Expected. Documentation-first was deliberate; drift closes as slices land |
| Experience contract | Ten page specs, four journeys, twelve component specs | No matching routes or components exist | Expected. Every product spec carries `status: draft`, which the contract defines as intent authored ahead of the build. `experience-manager spec validate` is clean apart from the generated `notes` page, which `detemplate` removes |
| Reference docs | Describe a generic `<domain>` shape plus a marked `notes` example, and a planned surface for the five real domains | Accurate to the code today | The example blocks resolve at `detemplate`; the planned surface resolves as slices land |
| Dependencies | Nine scenario dependencies in `INTEGRATIONS.md` | `.vrooli/service.json` declares the same nine | **No drift** — verified 2026-09-01 |

## Cross-references

- `docs/internal/SECURITY.md` — the threat model these gaps sit in
- `docs/internal/PROGRESS.md` — what has actually been done
- `docs/internal/DECISIONS.md` — decisions that deliberately deferred some of these
- `docs/concepts/FLOWS.md` — deferred and unmodelled flows
- `PRD.md`, `requirements/` — unbuilt features, which do not belong here

## Work ladder

- **W0 Contract** — **Fails on re-measurement (2026-09-01)**. The exact `swarm-manager goals list --json` name/description search returns active goal `phone-agent`; `swarm-manager goals get --name phone-agent` describes a “Phone- and browser-accessible agent assistant” with a Twilio phone channel. The Switchboard PRD places voice calls and phone-number access at `OT-P2-002`/`OT-P2-003`, and the approved Switchboard plan explicitly lists voice and SMS as deferred P2 scope. This is a bidirectional contract conflict: the active goal makes phone access load-bearing while the scenario contract makes it optional/deferred. Per the ladder, lower-rung gates are not authoritative until this is resolved.
- **W1 Business health** — **Passes at L3 (2026-09-01)**. `vrooli scenario requirements validate switchboard --json` reports a clean PRD, registry, intent linkage, and fresh evidence traceability with zero required/uncheckable findings.
- **W2 Requirements traceability** — **Passes at L3 (2026-09-01)**. The same server-owned validation reports all requirement refs resolved and current evidence; no manual status edits were used.
- **W3 Implementation** — Focused Go, CLI, and UI checks pass for the implemented messaging slice. Descriptor-backed channels, durable threads and budgets, durable thread-to-agent-run continuation, bounded asynchronous agent-manager event replies on the originating channel/thread, trust/dispatch seams, binding create/list, shared notification delivery, binding-gated ingress, a WebSocket in-app adapter, a Telegram Bot API adapter, typed capability-grant parsing, owner-only expiring gates, authenticated owner gate endpoints, confirm-before-write authoring seams, Slack/iMessage adapter contracts, and typed translation-backed console surfaces are implemented. Live unbound ingress rejects with HTTP 403; live bound ingress reaches agent-manager but the host currently rejects it at the external concurrent-run capacity gate (`20/10`). The server-owned Test Genie run `20260901-185528-1f067df1` is terminally failed and remains the last full-suite evidence; it must not be treated as a passing gate. External prompt-manager gate/authoring wiring, LPBS production metering, remote iMessage delivery, full requirement traceability, and the contract decision above remain incomplete. The `channel-conformance` phase is registered and its targeted run passes using deterministic transport fixtures; native provider/live binding evidence remains separate.
- **W3 update (2026-09-01)** — Budget suppression now exposes an owner-notice callback and dispatch tests prove it fires once; in-app WebSocket writes are serialized. This closes the local notification seam but not owner-address resolution or production owner delivery, which remain part of the capability-gate/notification integration gap.
- **W3 update (2026-09-01)** — Added typed capability-grant parsing, an injected-clock owner-only gate service, confirm-before-write authoring, and a UI gate that hides grant controls from non-owners. These are focused seams with passing tests; external Prompt Manager write/read wiring, authenticated gate identity, and in-thread API integration remain open.
- **W3 update (2026-09-01)** — Corrected the metering seam to model the real LPBS boundary: reserve, execute the supplied work callback, finalize on success, and release on execution failure. No production LPBS call path exists yet; see SWBD-PROB-012.
- **W3 update (2026-09-01)** — Gate REST mutations now derive the owner from the shared `owneridentity.Validator`; request-body `owner_id` and `actor_id` are only consistency checks. Production wiring uses scenario-authenticator, while isolated tests use the explicit identity-subject seam.
- **W3 update (2026-09-01)** — Authenticated request bearers now cross the Switchboard dispatch boundary transiently and are forwarded to agent-manager for its existing owner-identity path; the credential is neither persisted nor synthesized. Channel-originated messages without a bearer remain attenuated by the channel trust grant.
- **W3 update (2026-09-01)** — Production adapter construction now uses a package-level factory registry. Built-ins register from one builtins file, while a scratch `echo.json` descriptor plus one `api/internal/channels/adapters/echo.go` file was added, restarted, and passed the unchanged `channel-conformance` phase (`20260901-222301-92142e9c`); the two files were then removed. This proves the fifth-channel change boundary without adding a main or conformance edit.
- **W3 update (2026-09-01)** — Registered and executed the server-owned `channel-conformance` phase. It enumerates every registered descriptor and runs all seven shared contract cases; the current result is clean for the deterministic fixture transport. iMessage now has an injectable macOS Messages AppleScript polling contract with bounded parsing tests, but the existing `vrooli-bridge` API still has no typed way to host that poller on a remote Mac node, so remote fleet delivery remains open.
- **W3 update (2026-09-01)** — The Conversations page now opens the registered in-app WebSocket adapter, renders human/agent transcript roles and connection failures, and sends text plus file metadata on the same durable thread. The declared conversation and new-agent routes now use real surfaces rather than generic route placeholders; targeted UI tests, type-check, and production build pass.
- **W3 update (2026-09-01)** — The Channels page now submits the existing typed `ChannelService/CreateBinding` operation from an attach form, reports success/failure, and supplies the binding needed by in-app ingress. Focused channel, conversation, and gate UI tests pass.
- **W3 update (2026-09-01)** — `ChannelService/CreateBinding` now requires an authenticated owner identity (or the explicit isolated-test subject seam); missing credentials are rejected before storage. The attach-path regression test covers both unauthenticated rejection and successful authenticated creation.
- **W3 update (2026-09-01)** — Production startup now owns the external adapter lifecycle: available non-HTTP adapters are probed and connected, unavailable adapters are logged and skipped, and server cleanup cancels and joins receive loops. Focused registry/handler tests pass; the current Linux runtime correctly reports Telegram, Slack, and iMessage unavailable from host facts.
- **W3 update (2026-09-01)** — Added `.vrooli/lint/no-channel-branch.yml` and wired it into the scenario Go lint target. The rule reports a deliberate `channel == "telegram"` branch on stdin and reports no matches across `api/internal` and `api/handlers` outside adapter packages.
- **W3 update (2026-09-01)** — UI coverage is now above the configured floor (116 tests; 96.62% statements, 85.31% branches, 87.87% functions), but the governed Unit Health phase still fails on `UNIT_REQUIRED_ROLE_MISSING`: Code Facts index mutation is unavailable (`permission_denied: index mutation authorization is not configured`) while Unit Health retains a stale UI-role inventory. The direct Code Facts no-cache describe now observes the TypeScript UI parse unit; this remains an infrastructure evidence blocker, not a Switchboard test failure.
- **W3 update (2026-09-01)** — Replaced remaining inline console copy with generated typed translation keys across the real pages, shell, routes, and capability gate. `pnpm run lint` now exits cleanly with only two existing Fast Refresh warnings; string generation, type-check, coverage, and production build pass. Tests run in i18n `cimode`, so the authoring and conversation regressions assert registered translation keys rather than English fallback text.
- **W3 update (2026-09-01)** — The live Settings surface now exposes all declared descriptor, metering/BYOK, quiet-hours, theme, blast-radius, and save bindings. After fixing the desktop-sidebar landscape overflow and the BYOK touch target, `experience-manager spec validate switchboard --json` captured all required viewports and reached L3 with zero findings. `template-manager detemplate switchboard --json` confirms the generated notes domain is already fully removed and `bas/registry.json` names switchboard.
- **W3 update (2026-09-01)** — Navigation labels now use the typed locale registry for every canonical route. UI coverage remains green at 116 tests; the active objective evidence query reports `T2 Personal agency` with `hasEvidence: true` from Switchboard API/CLI channel and binding surfaces, although the separate objective record still reports `served: false`.
- **W3 update (2026-09-01)** — The managed UI now declares the in-app WebSocket upgrade path, so the browser conversation surface reaches the registered adapter through the UI port. Targeted `ui-health` run `20260901-224738-c1702cd8` passes all routes and viewports; the current advisory findings are template/adoption debt only.
- **W3 update (2026-09-01)** — BottomNav adoption was linked through the RCL CLI and generated `selectors.library.ts` plus the library locale bridge. UI type-check, template-library checks, 116 UI tests, and production build pass. The RCL obligations read still reports `component_import: false` despite the pinned import, so that dimension remains explicitly unclaimed.
- **W3 update (2026-09-01)** — Provider conformance documentation now has the required North Star, rung, finding, canonical-fix, and verification sections. Targeted provider-conformance run `20260901-225225-3eb58e7d` passes; one advisory `PROVIDER_RUNG_UNGATED` warning remains in the phase descriptor because the maturity ladder has no destination-rung finding mapping.
- **W3 update (2026-09-01)** — The plan baseline was re-anchored as generation 3 after the original immutable capture was invalidated by source drift. Git Control Tower collection `switchboard-implement-and-validate-the-multi-channel-agent-baseline` completed with notification-hub, react-component-library, and switchboard all ready and terminal outcome passed.

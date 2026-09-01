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
- **Status**: open (expected)
- **Severity**: low
- **Affects**: `docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`, `docs/internal/TESTING.md`, `docs/internal/SEAMS.md`
- **Symptom**: These documents describe the generated `notes` example domain because no real domain exists yet.
- **Impact**: A reader could mistake the example surface for the product surface. Mitigated by the planned-surface sections added to the reference documents and by the fencing the template already applies.
- **Evidence**: `template-manager orient switchboard` reports `example-domain-removed` incomplete.
- **Blocked on**: The first real vertical slice, then `template-manager detemplate switchboard`.

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

## Architecture Drift

Divergences between what the documents assert and what the code does. Checked
against the orientation gates.

| Drift | Documents say | Code says | Resolution |
|---|---|---|---|
| Domain implementation | Five domains with a fixed build order | Only the template's `notes` example exists | Expected. Documentation-first was deliberate; drift closes as slices land |
| Experience contract | Six product pages, three journeys | No matching routes or components exist | Expected. All specs carry `status: draft`, which the contract defines as intent authored ahead of the build |
| Reference docs | Describe `notes` endpoints and commands | Accurate to the code today | Resolves at `detemplate` |
| Dependencies | Nine scenario dependencies in `INTEGRATIONS.md` | `.vrooli/service.json` declares the same nine | **No drift** — verified 2026-09-01 |

## Cross-references

- `docs/internal/SECURITY.md` — the threat model these gaps sit in
- `docs/internal/PROGRESS.md` — what has actually been done
- `docs/internal/DECISIONS.md` — decisions that deliberately deferred some of these
- `docs/concepts/FLOWS.md` — deferred and unmodelled flows
- `PRD.md`, `requirements/` — unbuilt features, which do not belong here

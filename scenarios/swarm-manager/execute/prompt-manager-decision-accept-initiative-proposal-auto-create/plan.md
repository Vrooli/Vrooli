# Implementation Plan: Auto-Create Initiative on Accepting `initiative-proposal` Decisions

## Purpose

When an operator accepts a prompt-manager decision whose `context` is `initiative-proposal`, derive and create the corresponding `swarm-manager` initiative in one step — using structured metadata captured on the decision itself rather than hand-transcribed prose. Eliminate the second CLI call and the fidelity loss between "what the team agreed to" and "what was created."

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer interoperability-steer seam-discovery-and-enforcement boundary-of-responsibility-enforcement
```

Also read the just-landed sibling contract:

```bash
cat scenarios/prompt-manager/docs/reference/decision-modifications-contract.md
```

## Problem Statement

- `context: initiative-proposal` is the most common decision context in the morning vision walk; accept of one of these almost always implies "now create an initiative for this."
- Today the operator must:
  1. `prompt-manager team decision-accept ... --selected=A --notes="..."`
  2. `swarm-manager initiatives create --name=... --title=... --description=... --priority=...`
- Pain points (surfaced 2026-04-24 walk creating `web-console-readiness`):
  - **Two CLI calls** for the most common decision outcome.
  - **Hand-transcribed fidelity loss** — the initiative name/title/description/priority is re-authored in a separate JSON payload, which can drift from the decision's actual rationale and the just-landed `modifications` field (sibling F15).
  - No structured place to record `depends_on` at proposal time, so it gets bolted on after the fact.

## Greenfield Constraint

This is **greenfield** wiring of two scenarios. No backwards-compatibility shim for "old" decisions tagged `initiative-proposal` without metadata — the new metadata block is optional, and acceptance without it produces a clear error or skip (resolved by d5).

## Scope

### In scope
- New structured `initiative_metadata` block on prompt-manager decision records, populated via `decision-add` (and `decision-update`) when `context=initiative-proposal`.
- API schema extension on prompt-manager: `DecisionInitiativeMetadata` proto + handler validation + persistence.
- CLI flag(s) on `prompt-manager team decision-add` / `decision-update` for providing the metadata.
- New behavior on `prompt-manager team decision-accept` for `context=initiative-proposal`: trigger the downstream initiative-create.
- Cross-scenario seam between prompt-manager (decision side) and swarm-manager (initiatives side) — see d3.
- Propagation of the **selected option's rationale** + **`modifications`** (sibling F15) into the generated initiative's description.
- UI: structured form for `initiative_metadata` on decision-add for `initiative-proposal` decisions; preview block on decision-accept showing the initiative that will be created; success/failure indicator.
- CLI default render of `initiative_metadata` on `decision-show` for `initiative-proposal` decisions.

### Out of scope
- Automatic creation of initiative *members* (backlog items) — only the initiative itself. Member auto-population is a follow-on.
- Retroactive backfill of pre-existing `initiative-proposal` decisions.
- Editing `initiative_metadata` after the decision is accepted (immutable post-accept, mirroring the `modifications` rule from F15).
- Changes to reject / defer paths — auto-create only fires on **accept**.
- Changes to `swarm-manager initiatives create`'s validation rules; this item only invokes them.
- A new "team ↔ scenario" mapping system; for now, the `initiative_metadata` block carries the target scenario explicitly (swarm-manager).

## Cross-Initiative Implications

Sibling items in initiative `prompt-manager-decision-workflow-polish`:

| Sibling | Relationship |
|---|---|
| `execute/prompt-manager-decision-partial-accept-with-modifications` (F15, completed) | This item **consumes** the `modifications` field. Plan must propagate `excluded_clauses` / `additions` / `rationale` into the generated initiative description. |
| `fix/prompt-manager-decision-show-options-default-output` (F9) | Once that lands, ensure the `decision-show` default output also renders the `initiative_metadata` block. |
| `fix/prompt-manager-decision-accept-no-options-ergonomics` (F10) | If accepted, ensure `--selected=proposed`-style sugar still fires auto-create when context=initiative-proposal. |

Orchestrator: F14 (this item) sequences after F15 (done) and ideally after F9 / F10 so display + ergonomics are consistent.

## Current Technical Context

Targets (verify at execute-time):

- **Decision schema (prompt-manager):**
  - Proto + Go struct: `scenarios/prompt-manager/api/...` (decision record). The `Context string` field on the decision struct (`teams.go:194,228`) is what carries `initiative-proposal`.
  - Reference: `prompt-manager/cli/teams/teams.go` lines ~2870+ for `decision-accept` flag wiring and ~2630 for `decision-list --context=...` filter.
- **Initiative create API (swarm-manager):**
  - `scenarios/swarm-manager/api/internal/initiativereview/service.go` and adjacent.
  - CLI: `scenarios/swarm-manager/cli/cmd_initiatives.go`, `cli/domains/initiatives/register.go`.
- **Cross-scenario seam:** No existing direct call from prompt-manager → swarm-manager. The auto-create path needs a deliberate seam (d3).
- **Sibling F15 contract:** `scenarios/prompt-manager/docs/reference/decision-modifications-contract.md` defines the `modifications` shape this item consumes.

## Target End State

At decision-add time:

```bash
prompt-manager team decision-add <team-id> \
  --context=initiative-proposal \
  --topic="Web console readiness" \
  --options='[...]' \
  --initiative-metadata='{"name":"web-console-readiness","priority":5,"depends_on":["api-foundation"],"target_scenario":"swarm-manager"}'
```

At decision-accept time:

```bash
prompt-manager team decision-accept <team-id> <id> --selected=A
# auto-create fires automatically (per d4) →
# created initiative: web-console-readiness (priority=5, depends_on=[api-foundation])
# description: <decision topic> + <selected option rationale> + <modifications block, if any>
```

Same capability in API and UI.

## Open Decisions (Round 1)

See `workshop/round-001.json` for full options. Headline questions:

1. **d1 — Where does initiative metadata live?** Structured field on the decision record (recommended) vs. inferred from option labels.
2. **d2 — Trigger model on accept.** Always-on for `context=initiative-proposal` if metadata present (recommended) vs. opt-in `--auto-create-initiative` flag vs. interactive prompt.
3. **d3 — Cross-scenario seam.** prompt-manager API calls swarm-manager initiatives API directly via internal client (recommended) vs. CLI orchestrates two API calls vs. emit an event/queue.
4. **d4 — Failure semantics.** Decision still accepts but initiative-create returns warning (recommended) vs. transactional all-or-nothing vs. accept blocks until create succeeds.
5. **d5 — Decision tagged `initiative-proposal` but missing `initiative_metadata`.** Error on accept (recommended) vs. fall back to today's manual flow vs. silently create from topic + selected option.

## Implementation Strategy (phased — fills in once decisions resolve)

### Phase 1 — Decision schema (prompt-manager API)
- Define `DecisionInitiativeMetadata` proto:
  ```proto
  message DecisionInitiativeMetadata {
    string name = 1;            // initiative name (kebab-case)
    int32 priority = 2;         // 1-9, optional (default from server)
    repeated string depends_on = 3;
    string target_scenario = 4; // e.g. "swarm-manager"; defaults configurable
    string title = 5;           // optional override; default = decision topic
  }
  ```
- Field is optional; only meaningful when `context=initiative-proposal` (validated at write time).
- Immutable after first accept (mirrors F15 `modifications` rule).

### Phase 2 — CLI (prompt-manager)
- Add `--initiative-metadata='{...}'` and `--initiative-metadata-file=<path>` flags to `decision-add` and `decision-update`.
- Validate at boundary; reject metadata on decisions whose context is not `initiative-proposal` (or warn — TBD by d5).
- `decision-show` default output renders an `Initiative Metadata:` block when present.

### Phase 3 — Auto-create wiring (the seam)
- On `decision-accept` where `context=initiative-proposal`:
  - Resolve seam per d3.
  - Build initiative create payload from: `initiative_metadata.{name,priority,depends_on,target_scenario,title}`, `decision.topic` (fallback title), selected option rationale (description body), and `modifications` (appended).
  - Invoke create. Apply failure policy per d4.
- Surface the created initiative id/name in the decision-accept response.

### Phase 4 — UI parity
- Decision-add form: when `context=initiative-proposal` is selected, expose the structured form for `initiative_metadata` (name, priority, depends_on chips, target_scenario, optional title).
- Decision-accept view: show a "Will create initiative: X" preview block before submit; render result/failure inline.
- Decision-history / decision-detail: render the metadata block alongside the existing fields.

### Phase 5 — Documentation and follow-ups
- Author / extend a contract doc: `scenarios/prompt-manager/docs/reference/decision-initiative-proposal-contract.md` describing the metadata shape, the auto-create trigger, failure modes, and the relationship to `modifications`.
- Reference doc from proto comments and from the `modifications` contract (for cross-link).
- Notify F9 to render `initiative_metadata` in default `decision-show` output.

### Final — Cleanup & Verification
- Type/lint/test green across both scenarios.
- `vrooli scenario restart prompt-manager swarm-manager` (or each separately).
- End-to-end golden: decision-add → decision-accept → assert initiative exists with expected name/priority/description/depends_on.

## Contract Decisions (to harden as decisions resolve)

- New field name: `initiative_metadata` (distinct from `notes`, `modifications`).
- Three-surface parity required (API + CLI + UI ship together).
- Auto-create fires only on **accept**; not on reject/defer.
- `initiative_metadata` is immutable after first accept (consistent with F15).
- Generated initiative description = `<decision.topic>` + `<selected option label + rationale>` + `<modifications block, if any>`.
- Error model on auto-create failure: structured error mentioning both the decision id and the failed create payload, regardless of d4 resolution.

## Validation Policy

- `name` must match initiative-name regex (kebab-case); enforced at prompt-manager write time so failures surface early.
- `priority` (if set) bounded by swarm-manager initiative priority range.
- `target_scenario` validated against an allowlist (initially `["swarm-manager"]`).
- `depends_on` entries validated for format only at write time; existence-check happens at create time on the swarm-manager side (current behavior).
- `initiative_metadata` rejected on decisions whose `context != initiative-proposal`.

## Testing Plan

- **API unit tests (prompt-manager):**
  - `decision-add --context=initiative-proposal --initiative-metadata='{...}'` → persisted and round-trips.
  - Same flag on non-`initiative-proposal` context → rejected.
  - Malformed metadata → 400 with structured field-violation.
  - Mutation post-accept → rejected.
- **API integration tests (cross-scenario):**
  - Accept of `initiative-proposal` decision with metadata → swarm-manager initiative exists with derived fields, description includes selected option rationale + modifications.
  - Accept with metadata absent → behavior per d5.
  - swarm-manager create fails (e.g., duplicate name) → behavior per d4; decision state observable post-failure.
- **CLI integration tests:**
  - End-to-end add → accept → `swarm-manager initiatives get` returns the created initiative.
  - `--initiative-metadata-file=fixture.json` produces equivalent result to inline JSON.
- **UI tests:**
  - Component test: structured form only renders for `initiative-proposal` context.
  - View test: accept preview block + post-accept success/failure indicator.

## Rollout / Validation Checklist

- [ ] API test suites green in both scenarios.
- [ ] CLI round-trip on a real `dec-*` id produces a real initiative.
- [ ] UI manual test: full add → accept flow shows preview and outcome.
- [ ] Contract doc published in `scenarios/prompt-manager/docs/reference/`.
- [ ] Sibling F9 notified to include `Initiative Metadata` block in display fix.
- [ ] Sibling F10 notified to ensure `--selected=proposed` paths still trigger auto-create.
- [ ] Modifications from F15 verified to propagate into the generated initiative description.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Cross-scenario coupling makes prompt-manager API depend on swarm-manager runtime availability. | Seam choice (d3) localizes the dependency; failure policy (d4) prevents accept-path breakage. |
| Operator accepts a decision and the create silently fails. | Structured error in response; surface in CLI/UI; decision-show flags "auto-create failed: <reason>" until retried. |
| Metadata drifts from selected option's actual rationale. | Description body is **derived** from the selected option + modifications, not hand-typed at accept. |
| Future second target scenario (not swarm-manager) needs the same flow. | `target_scenario` field is already present in the metadata; allowlist starts narrow but is extensible. |
| `depends_on` entries reference initiatives that don't yet exist. | Validation matches existing swarm-manager behavior; not made stricter here. |

## Non-goals / Prohibited Patterns

- Do **not** parse `notes` prose to derive initiative fields — `initiative_metadata` is the only authoritative source.
- Do **not** auto-create on reject or defer.
- Do **not** edit `initiative_metadata` after first accept.
- Do **not** auto-create initiative *members* in this item (separate follow-on).

## Definition of Done

- [ ] `initiative_metadata` field round-trips end-to-end via API, CLI, UI.
- [ ] Accepting an `initiative-proposal` decision (per d4 / d5 outcomes) produces the corresponding swarm-manager initiative.
- [ ] Generated initiative carries selected option rationale + F15 modifications in its description.
- [ ] All five decisions in `workshop/round-001.json` resolved and reflected here.
- [ ] Contract doc landed and linked from proto comments.
- [ ] Three-surface parity verified.

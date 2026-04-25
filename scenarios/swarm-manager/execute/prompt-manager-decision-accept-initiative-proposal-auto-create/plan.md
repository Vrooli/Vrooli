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

This is **greenfield** wiring of two scenarios. No backwards-compatibility shim for "old" decisions tagged `initiative-proposal` without metadata — the new metadata block is optional, and acceptance without it errors per d5.

## Scope

### In scope
- New structured `initiative_metadata` block on prompt-manager decision records, populated via `decision-add` (and `decision-update`) when `context=initiative-proposal`.
- API schema extension on prompt-manager: `DecisionInitiativeMetadata` proto + handler validation + persistence.
- New `auto_create_status` field on the decision record (`unset | pending | created | failed`), plus `auto_create_error` and `auto_create_initiative_ref`.
- CLI flag(s) on `prompt-manager team decision-add` / `decision-update` for providing the metadata.
- Auto-create on `prompt-manager team decision-accept` for `context=initiative-proposal` (always-on per d2 when metadata present).
- Cross-scenario seam between prompt-manager and swarm-manager via `api-core/discovery` — see Phase 3.
- **Phase 0 prerequisite (per d6=A): expose swarm-manager initiatives over HTTP** — bundled into this item.
- **Server-side call site (per d7=A)**: prompt-manager API's decision-accept handler is the single point that invokes swarm-manager initiatives.
- Propagation of the **selected option's rationale** + **`modifications`** (sibling F15) into the generated initiative's description.
- UI: structured form for `initiative_metadata` on decision-add for `initiative-proposal` decisions; preview block on decision-accept showing the initiative that will be created; success/failure indicator with operator-facing workaround on failure.
- CLI default render of `initiative_metadata` and `auto_create_status` on `decision-show` for `initiative-proposal` decisions.

### Out of scope
- Automatic creation of initiative *members* (backlog items) — only the initiative itself. Member auto-population is a follow-on.
- Retroactive backfill of pre-existing `initiative-proposal` decisions.
- Editing `initiative_metadata` after the decision is accepted (immutable post-accept, mirroring the `modifications` rule from F15). Pre-accept editing via `decision-update` is permitted (see Validation Policy).
- Changes to reject / defer paths — auto-create only fires on **accept**.
- Changes to `swarm-manager initiatives create`'s validation rules; this item only invokes them.
- A new "team ↔ scenario" mapping system; for now, the `initiative_metadata` block carries the target scenario explicitly.
- **No first-class `decision-retry-auto-create` command (per d8=C).** On failure the operator runs `swarm-manager initiatives create` manually; the CLI/UI output explicitly prints that command line so the workaround is one copy-paste away.

## Cross-Initiative Implications

Sibling items in initiative `prompt-manager-decision-workflow-polish`:

| Sibling | Relationship |
|---|---|
| `execute/prompt-manager-decision-partial-accept-with-modifications` (F15, completed) | This item **consumes** the `modifications` field. Plan must propagate `excluded_clauses` / `additions` / `rationale` into the generated initiative description. |
| `fix/prompt-manager-decision-show-options-default-output` (F9) | Once that lands, ensure the `decision-show` default output also renders the `initiative_metadata` and `auto_create_status` blocks. |
| `fix/prompt-manager-decision-accept-no-options-ergonomics` (F10) | If accepted, ensure `--selected=proposed`-style sugar still fires auto-create when context=initiative-proposal. |

Orchestrator: F14 (this item) sequences after F15 (done) and ideally after F9 / F10 so display + ergonomics are consistent. Initiative `prompt-manager-decision-workflow-polish` has no upstream/downstream dependencies; sequencing is internal only.

## Resolved Decisions

### Round 1
- **d1=A** — `initiative_metadata` is a first-class structured field on the decision record, populated via `decision-add` / `decision-update`, immutable post-accept.
- **d2=A** — Auto-create is always-on when `context=initiative-proposal` AND `initiative_metadata` is present. No flag, no prompt. CLI/UI output must clearly indicate creation, surface get/edit commands, and remind the operator to enrich the initiative if needed (encoded in d9).
- **d3=Other (api-core/discovery)** — Cross-scenario calls go through `github.com/vrooli/api-core/discovery.ResolveScenarioURLDefault(ctx, "swarm-manager")`, then plain HTTP. Mirrors prompt-manager's existing skills-call pattern and workspace-sandbox's sandbox-resolution pattern.
- **d4=A** — On create failure, the decision still accepts. Response includes a structured warning. The decision record persists `auto_create_status: failed` with the reason. No partial rollback.
- **d5=A** — If `context=initiative-proposal` but `initiative_metadata` is absent at accept time, accept fails fast with: "add --initiative-metadata to the decision (or use decision-update) before accepting." No silent fallback.

### Round 2
- **d6=A** — Bundle the swarm-manager initiatives HTTP endpoint into this item's Phase 0. No prerequisite split; one shippable end-to-end behavior.
- **d7=A** — Server-side: prompt-manager API's decision-accept handler invokes swarm-manager initiatives via discovery. Single source of truth across UI/CLI/API.
- **d8=C** — **No new retry command.** On auto-create failure, the CLI/UI output instructs the operator to run `swarm-manager initiatives create ...` manually with the exact command line pre-filled from the decision's metadata. The decision's `auto_create_status` stays `failed` until the operator updates it via `decision-update --auto-create-status=created --auto-create-initiative-ref=<scenario>/<name>` (or the operator chooses to leave it as a record of the failed attempt).
- **d9=A** — Structured "Auto-Created Initiative" block in CLI/UI: name, priority, depends_on, get-command, edit-command, plus a "Reminder:" line to enrich the initiative. Failure case renders the same block shape with `Status: failed (<reason>)` plus the manual workaround command line (see d8).

## Current Technical Context

Targets (verify at execute-time):

- **Decision schema (prompt-manager):**
  - Proto + Go struct: `scenarios/prompt-manager/api/...` (decision record). The `Context string` field on the decision struct (`teams.go:194,228`) is what carries `initiative-proposal`.
  - Reference: `prompt-manager/cli/teams/teams.go` lines ~2870+ for `decision-accept` flag wiring and ~2630 for `decision-list --context=...` filter.
- **Cross-scenario seam (api-core):**
  - `github.com/vrooli/api-core/discovery.ResolveScenarioURLDefault(ctx, scenarioName) (string, error)`.
  - Existing usage in prompt-manager: `resolvePromptManagerBaseURL()` wrapper for skills endpoints. New wrapper to add: `resolveSwarmManagerBaseURL()` (or similar) for initiatives.
- **Initiative create (swarm-manager):**
  - CLI today: `scenarios/swarm-manager/cli/cmd_initiatives.go`, `cli/domains/initiatives/register.go`.
  - Service: `scenarios/swarm-manager/api/internal/initiativereview/service.go` and adjacent.
  - **Gap (Phase 0 work, per d6=A):** swarm-manager's HTTP server exposes profiles/tasks/runs/runners/investigation but NOT initiatives. Add `POST /api/v1/initiatives` and `GET /api/v1/initiatives/{name}` mirroring the CLI create/get behavior.
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

At decision-accept time (success):

```bash
prompt-manager team decision-accept <team-id> <id> --selected=A
# →
# Decision dec-abc123 accepted (selected: A).
# Auto-Created Initiative:
#   Name:        web-console-readiness
#   Priority:    5
#   Depends on:  api-foundation
#   Inspect:     swarm-manager initiatives get --name web-console-readiness
#   Edit:        swarm-manager initiatives update --name web-console-readiness ...
# Reminder: enrich the initiative (description, members, additional depends_on) as needed.
```

At decision-accept time (create failure, per d4 + d8=C + d9):

```bash
prompt-manager team decision-accept <team-id> <id> --selected=A
# →
# Decision dec-abc123 accepted (selected: A).
# Auto-Created Initiative:
#   Status: failed (initiative name "web-console-readiness" already exists)
#
# To create the initiative manually, run:
#   swarm-manager initiatives create \
#     --name=web-console-readiness \
#     --title="Web console readiness" \
#     --priority=5 \
#     --depends-on=api-foundation \
#     --description-file=/tmp/dec-abc123-initiative-description.md
#
# Once created, mark the decision as resolved:
#   prompt-manager team decision-update <team-id> dec-abc123 \
#     --auto-create-status=created \
#     --auto-create-initiative-ref=swarm-manager/web-console-readiness
```

The pre-filled command line uses the exact metadata captured on the decision; the description body is materialized to a tmp file (or printed inline if short) so the operator does not re-author it. Same capability and rendering in API and UI.

## Implementation Strategy (phased)

### Phase 0 — Prerequisite: swarm-manager initiatives HTTP endpoint (per d6=A)
- Add `POST /api/v1/initiatives` to swarm-manager's HTTP server, mirroring the CLI's create command (name, title, description, priority, depends_on, etc.). Validation parity with the CLI path; reuse the same service-layer create function.
- Add `GET /api/v1/initiatives/{name}` so the auto-create caller can verify creation and surface conflicts (409 on duplicate name).
- Authentication/authorization: match the existing swarm-manager HTTP surface conventions; no new auth model in this item.

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
- Add `auto_create_status` to the decision record: enum `unset | pending | created | failed`, plus `auto_create_error string` and `auto_create_initiative_ref string` (e.g. `"swarm-manager/web-console-readiness"`).
- Both fields are optional; `initiative_metadata` is only meaningful when `context=initiative-proposal` (validated at write time). Immutable after first accept.
- `decision-update` accepts `--auto-create-status` and `--auto-create-initiative-ref` post-accept (per d8=C: operator marks resolution after running the manual workaround). All other auto-create-* writes are server-managed.

### Phase 2 — CLI (prompt-manager)
- Add `--initiative-metadata='{...}'` and `--initiative-metadata-file=<path>` flags to `decision-add` and `decision-update`.
- Validate at boundary; reject metadata on decisions whose context is not `initiative-proposal`.
- Pre-accept: `decision-update --initiative-metadata=...` may overwrite the block any number of times. Post-accept: rejected with the F15-style immutability error.
- Add `--auto-create-status` and `--auto-create-initiative-ref` flags to `decision-update` for the d8=C manual-resolution path. Refuses to set `created` unless `auto_create_initiative_ref` is provided.
- `decision-show` default output renders an `Initiative Metadata:` block and an `Auto-Create Status:` block when present.

### Phase 3 — Auto-create wiring (the seam, server-side per d7=A)
- On `decision-accept` where `context=initiative-proposal`:
  1. Validate `initiative_metadata` is present (else fail per d5).
  2. Resolve the swarm-manager base URL via `discovery.ResolveScenarioURLDefault(ctx, metadata.target_scenario)`.
  3. Build the create payload from: `initiative_metadata.{name, priority, depends_on, target_scenario, title}`, `decision.topic` (fallback title), the selected option's `label` + `rationale` (description body), and `modifications` (appended).
  4. POST to `/api/v1/initiatives`.
  5. On success: persist `auto_create_status=created`, `auto_create_initiative_ref=<scenario>/<name>`. Return both the accepted decision and the created initiative reference in the response.
  6. On failure: persist `auto_create_status=failed`, `auto_create_error=<reason>`. Return the accepted decision plus a structured warning that includes the **pre-filled manual workaround** (the exact `swarm-manager initiatives create ...` command line, plus the `decision-update` call to mark resolution). The description body for the workaround is either inlined (short) or materialized to a tmp file path emitted by the server and surfaced by the CLI.
- CLI/UI render the response per the d9 output spec (structured "Auto-Created Initiative" block).

### Phase 4 — UI parity
- Decision-add form: when `context=initiative-proposal` is selected, expose the structured form for `initiative_metadata` (name, priority, depends_on chips, target_scenario, optional title).
- Decision-accept view: show a "Will create initiative: X" preview block before submit; render result/failure inline using the same structured fields as CLI.
- Decision-history / decision-detail: render the metadata block and `auto_create_status` alongside the existing fields. Failure state surfaces the manual workaround commands (copyable) and a "Mark as resolved" affordance bound to `decision-update --auto-create-status=created --auto-create-initiative-ref=...`.

### Phase 5 — Documentation and follow-ups
- Author / extend a contract doc: `scenarios/prompt-manager/docs/reference/decision-initiative-proposal-contract.md` describing the metadata shape, the auto-create trigger, failure modes, the d8=C manual-recovery flow, and the relationship to `modifications`.
- Reference doc from proto comments and from the `modifications` contract (for cross-link).
- Notify F9 to render `initiative_metadata` + `auto_create_status` in default `decision-show` output.

### Final — Cleanup & Verification
- Type/lint/test green across both scenarios.
- `vrooli scenario restart prompt-manager swarm-manager` (or each separately).
- End-to-end golden: decision-add → decision-accept → assert initiative exists with expected name/priority/description/depends_on; failure-path golden: simulate duplicate name → assert decision accepts with `auto_create_status=failed` → assert response includes the pre-filled `swarm-manager initiatives create ...` line and the `decision-update --auto-create-status=created` follow-up.

## Contract Decisions (hardened)

- New field name: `initiative_metadata` (distinct from `notes`, `modifications`).
- New status field: `auto_create_status` (`unset | pending | created | failed`) + `auto_create_error` + `auto_create_initiative_ref`.
- Three-surface parity required (API + CLI + UI ship together).
- Auto-create fires only on **accept**; not on reject/defer.
- Auto-create executes **server-side** in the prompt-manager API (per d7=A); CLI and UI do not orchestrate the cross-scenario call.
- `initiative_metadata` is immutable after first accept (consistent with F15). Mutable pre-accept via `decision-update`.
- `auto_create_status` and `auto_create_initiative_ref` are operator-writable via `decision-update` post-accept solely to record the d8=C manual-recovery outcome.
- Generated initiative description = `<decision.topic>` + `<selected option label + rationale>` + `<modifications block, if any>`.
- Cross-scenario seam: `api-core/discovery` URL resolution + plain HTTP; no new event bus, no shell-out, no embedded second binary.
- Failure-path UX (per d8=C + d9=A): API response carries the pre-filled `swarm-manager initiatives create ...` command and the `decision-update` follow-up line as structured fields; CLI and UI render them verbatim. **No** `decision-retry-auto-create` command.

## Validation Policy

- `name` must match initiative-name regex (kebab-case); enforced at prompt-manager write time so failures surface early.
- `priority` (if set) bounded by swarm-manager initiative priority range.
- `target_scenario` validated against an allowlist (initially `["swarm-manager"]`).
- `depends_on` entries validated for format only at write time; existence-check happens at create time on the swarm-manager side (current behavior).
- `initiative_metadata` rejected on decisions whose `context != initiative-proposal`.
- `initiative_metadata` may be set/replaced by `decision-update` any number of times **before** first accept; rejected after accept.
- `decision-update --auto-create-status=created` requires a non-empty `--auto-create-initiative-ref` in the same call. Status transitions are constrained to `failed → created` (the manual-recovery completion) and `failed → failed` (re-record an updated error). No other operator-driven transitions.

## Testing Plan

- **API unit tests (prompt-manager):**
  - `decision-add --context=initiative-proposal --initiative-metadata='{...}'` → persisted and round-trips.
  - Same flag on non-`initiative-proposal` context → rejected.
  - Malformed metadata → 400 with structured field-violation.
  - `decision-update --initiative-metadata=...` pre-accept → overwrites; post-accept → rejected.
  - `decision-update --auto-create-status=created --auto-create-initiative-ref=...` post-accept → permitted; without the ref → rejected; from non-failed status → rejected.
- **API unit tests (swarm-manager, Phase 0):**
  - `POST /api/v1/initiatives` mirrors CLI behavior for valid + invalid payloads.
  - `GET /api/v1/initiatives/{name}` returns the created initiative.
  - Duplicate name → 409 with structured error usable by prompt-manager's failure path.
- **API integration tests (cross-scenario):**
  - Accept of `initiative-proposal` decision with metadata → swarm-manager initiative exists with derived fields, description includes selected option rationale + modifications, decision has `auto_create_status=created` and `auto_create_initiative_ref` set.
  - Accept with metadata absent → fails per d5; decision unchanged.
  - swarm-manager create returns 409 (duplicate name) → decision accepted, `auto_create_status=failed`, error captured; response includes the pre-filled `swarm-manager initiatives create ...` workaround and `decision-update` follow-up; operator runs the workaround manually then `decision-update --auto-create-status=created --auto-create-initiative-ref=...` flips status.
  - swarm-manager unreachable → decision accepted, `auto_create_status=failed`, error captured; same workaround surfaced.
- **CLI integration tests:**
  - End-to-end add → accept → `swarm-manager initiatives get` returns the created initiative.
  - `--initiative-metadata-file=fixture.json` produces equivalent result to inline JSON.
  - Output snapshot test for the success render and the failure render (the failure snapshot must include the exact `swarm-manager initiatives create ...` line and the `decision-update --auto-create-status=created ...` line per d9).
  - `decision-update --auto-create-status=created --auto-create-initiative-ref=...` flips status; rejected without the ref or from a non-failed starting state.
- **UI tests:**
  - Component test: structured form only renders for `initiative-proposal` context.
  - View test: accept preview block + post-accept success/failure indicator + (in failure case) copyable workaround commands and a working "Mark as resolved" affordance bound to `decision-update`.

## Rollout / Validation Checklist

- [ ] API test suites green in both scenarios.
- [ ] CLI round-trip on a real `dec-*` id produces a real initiative.
- [ ] CLI failure-path round-trip produces `auto_create_status=failed` with the rendered output containing the pre-filled `swarm-manager initiatives create ...` workaround and `decision-update --auto-create-status=created` follow-up.
- [ ] Manual-recovery round-trip: run the printed workaround, then `decision-update --auto-create-status=created --auto-create-initiative-ref=...`, and confirm `decision-show` reflects the resolved state.
- [ ] UI manual test: full add → accept flow shows preview, outcome, and (in failure case) copyable workaround + working "Mark as resolved" affordance.
- [ ] Contract doc published in `scenarios/prompt-manager/docs/reference/`.
- [ ] Sibling F9 notified to include `Initiative Metadata` + `Auto-Create Status` blocks in the display fix.
- [ ] Sibling F10 notified to ensure `--selected=proposed` paths still trigger auto-create.
- [ ] Modifications from F15 verified to propagate into the generated initiative description.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| swarm-manager initiatives HTTP endpoint is non-trivial (auth, validation parity), inflating this item's effort. | Phase 0 explicitly scopes only `POST` + `GET` for the auto-create path; reuse existing service-layer create function. If scope balloons during Phase 0, surface for a possible mid-flight split. |
| Cross-scenario coupling makes prompt-manager API depend on swarm-manager runtime availability. | Failure policy (d4=A) prevents accept-path breakage; failure response carries a pre-filled manual workaround (d8=C + d9=A) so transient outages have a one-copy-paste recovery; `auto_create_status` makes the failure persistent and queryable. |
| Operator accepts a decision and the create silently fails. | `auto_create_status` persisted on the decision; structured CLI/UI render per d9; failure render includes the exact workaround command line. |
| Metadata drifts from selected option's actual rationale. | Description body is **derived** from the selected option + modifications, not hand-typed at accept. |
| Manual-recovery (d8=C) operators forget to run `decision-update --auto-create-status=created`, leaving stale `failed` status on resolved decisions. | The failure-render output includes the `decision-update ...` line right next to the `swarm-manager initiatives create ...` line; UI surfaces a "Mark as resolved" button; contract doc spells out the two-step recovery. |
| Future second target scenario (not swarm-manager) needs the same flow. | `target_scenario` field is already present in the metadata; allowlist starts narrow but is extensible. |
| `depends_on` entries reference initiatives that don't yet exist. | Validation matches existing swarm-manager behavior; not made stricter here. |
| Three surfaces (API/CLI/UI) drift on output rendering of the auto-create block. | d7=A (server-side call) + d9=A structured response: surfaces consume the same structured fields (including the pre-filled workaround strings on failure), only formatting differs. |

## Non-goals / Prohibited Patterns

- Do **not** parse `notes` prose to derive initiative fields — `initiative_metadata` is the only authoritative source.
- Do **not** auto-create on reject or defer.
- Do **not** edit `initiative_metadata` after first accept.
- Do **not** auto-create initiative *members* in this item (separate follow-on).
- Do **not** introduce a new event bus or messaging layer; the seam is `api-core/discovery` + HTTP, period.
- Do **not** add a `decision-retry-auto-create` command or any in-process retry loop (per d8=C). Failure recovery is operator-driven via the pre-filled workaround.

## Definition of Done

- [ ] `initiative_metadata` field round-trips end-to-end via API, CLI, UI.
- [ ] `auto_create_status` field persists and renders on `decision-show` and the UI.
- [ ] Accepting an `initiative-proposal` decision (with metadata) produces the corresponding swarm-manager initiative via the discovery seam.
- [ ] Generated initiative carries selected option rationale + F15 modifications in its description.
- [ ] Failure-path response carries a pre-filled `swarm-manager initiatives create ...` workaround and a `decision-update --auto-create-status=created ...` follow-up; CLI and UI render both verbatim/copyable.
- [ ] `decision-update --auto-create-status=created --auto-create-initiative-ref=...` post-accept flips a `failed` decision to `created`.
- [ ] All round-1 (d1–d5) and round-2 (d6–d9) decisions resolved and reflected here.
- [ ] Contract doc landed and linked from proto comments.
- [ ] Three-surface parity verified.

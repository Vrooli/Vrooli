# Decision Initiative-Proposal Auto-Create — Contract

## What this is

When a decision's `context` is `initiative-proposal`, accepting it means "now
create the corresponding swarm-manager initiative." This contract defines the
structured `initiative_metadata` block that drives auto-creation, the
`auto_create_status` lifecycle, the failure-recovery flow, and the cross-scenario
seam.

## Why this exists

Without it, the operator must run two CLI calls (decision-accept, then
swarm-manager initiatives create) and re-author the initiative's name / title /
description / priority by hand — losing fidelity with what the decision
actually approved. `initiative_metadata` is the single authoritative source.

## Shape

```jsonc
{
  "name":            "string",          // required; kebab-case (^[a-z0-9]+(-[a-z0-9]+)*$)
  "priority":        0,                  // 0 (unset) or 1-10
  "depends_on":      ["string", ...],   // optional initiative-name refs
  "target_scenario": "swarm-manager",   // optional; allowlisted; default = "swarm-manager"
  "title":           "string"           // optional override; default = decision.topic
}
```

- `name` is required and must match `^[a-z0-9]+(-[a-z0-9]+)*$`.
- `priority`, when set, must be in `[1, 10]`. `0` means "unset" — the
  swarm-manager service applies its own default.
- `depends_on` entries must be non-empty strings; existence is verified
  swarm-side at create time.
- `target_scenario` is validated against an allowlist (initially
  `["swarm-manager"]`).
- `title` falls back to `decision.topic` and then to `name` when empty.

`initiative_metadata` is **only valid** on decisions whose `context` is
`initiative-proposal`. The API rejects metadata on any other context.

## Auto-Create lifecycle

A decision carries three auto-create state fields:

| Field | Purpose |
|---|---|
| `auto_create_status` | `""` (unset) / `"pending"` / `"created"` / `"failed"` |
| `auto_create_initiative_ref` | `"<scenario>/<name>"` — populated on success |
| `auto_create_error` | failure reason — populated on failure |

The lifecycle:

1. **decision-add / decision-update (pre-accept):** operator attaches
   `initiative_metadata` via `--initiative-metadata='{...}'` or
   `--initiative-metadata-file=<path>`. Mutually exclusive flags. Replaceable
   any number of times before first accept.
2. **decision-accept:** if `context == initiative-proposal` and metadata is
   present, the prompt-manager API calls
   `POST <swarm-manager>/api/v1/initiatives` via `api-core/discovery`. The
   description body is derived from the decision's topic, the selected option's
   label + rationale, and the modifications block (per the
   [modifications contract](decision-modifications-contract.md)).
3. **success:** decision is accepted; `auto_create_status="created"` and
   `auto_create_initiative_ref="<scenario>/<name>"` are persisted; response
   carries the structured `auto_create_outcome` block for the surface to render.
4. **failure (per d4=A + d8=C):** decision is **still accepted** —
   `auto_create_status="failed"` and `auto_create_error=<reason>` are
   persisted. The response includes a pre-filled
   `swarm-manager initiatives create ...` workaround command line and a
   `prompt-manager team decision-update --auto-create-status=created
   --auto-create-initiative-ref=...` follow-up. The CLI/UI render both
   verbatim. The description body is materialised to a tmp file when long so
   the operator does not re-author it.
5. **manual recovery:** the operator runs the workaround command, then runs
   `decision-update --auto-create-status=created --auto-create-initiative-ref=...`
   to flip the persisted state. `failed → failed` is also permitted (re-record
   an updated error). No other transitions are allowed; there is **no**
   `decision-retry-auto-create` command.

## Semantic rules

1. **Authoritative source.** `initiative_metadata` is the only source of the
   initiative's identity. Do not parse `notes` or `description` to derive it.
2. **Auto-create fires only on accept.** Reject and defer paths do not call
   swarm-manager.
3. **Server-side execution.** The prompt-manager API's decision-accept handler
   is the single point that invokes swarm-manager. CLI and UI consume the
   structured response — they do not orchestrate the cross-scenario call.
4. **Accept-once immutability.** Once a decision is accepted, its
   `initiative_metadata` cannot change (mirrors the `modifications` rule).
   Pre-accept editing is permitted via `decision-update`.
5. **No silent fallback.** Accepting an `initiative-proposal` decision without
   metadata is rejected with: "add --initiative-metadata to the decision (or
   use decision-update) before accepting."
6. **Failure is non-fatal.** A swarm-manager outage or duplicate-name 409 does
   **not** block the accept — the decision still reaches `accepted` status, and
   the failure is queryable via `auto_create_status="failed"`.
7. **Members are out of scope.** Auto-create produces the initiative only;
   members (backlog items) are populated separately.

## Surfaces

- **API:** 
  - `POST /teams/{id}/decisions` and `PATCH /teams/{id}/decisions/{id}` accept
    `initiative_metadata` in the request body.
  - `PATCH /teams/{id}/decisions/{id}` returns
    `{ ...DecisionEntry, auto_create_outcome: AutoCreateOutcome }` when an
    `initiative-proposal` decision is accepted.
  - Cross-scenario call is `POST <swarm-manager>/api/v1/initiatives`, resolved
    via `discovery.ResolveScenarioURLDefault(ctx, target_scenario)`.
- **CLI:** 
  - `decision-add` / `decision-update`: `--initiative-metadata='{...}'`,
    `--initiative-metadata-file=<path>`.
  - `decision-update`: `--auto-create-status`, `--auto-create-initiative-ref`,
    `--auto-create-error` for the manual-recovery flow.
  - `decision-accept` and `decision-update` render the structured "Auto-Created
    Initiative" block on the response (success or failure with workaround).
  - `decision-show` renders `Initiative Metadata:` and `Auto-Create Status:`
    blocks when present.
- **UI:** 
  - DecisionLogView surfaces an outcome banner after acceptance with a
    copy-pasteable workaround command and a "Mark as resolved" affordance on
    failure. Persisted metadata + auto-create status render read-only on each
    decision card.

## Out of scope

- Auto-creation of initiative *members* (separate follow-on).
- Retroactive backfill of pre-existing `initiative-proposal` decisions.
- Editing `initiative_metadata` after first accept.
- A `decision-retry-auto-create` command or any in-process retry loop.
- A team↔scenario routing layer; `target_scenario` is carried explicitly.

## Why this contract is separate from proto/struct comments

The contract captures the **meaning** — what consumers should do with the data
— which is load-bearing across API, CLI, UI, swarm-manager, and downstream
operator workflows. Struct comments link here to ensure schema readers land on
this contract before writing consumer logic.

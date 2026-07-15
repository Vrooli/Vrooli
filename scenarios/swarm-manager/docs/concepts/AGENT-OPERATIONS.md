# Agent Operations

> **Status:** Canonical concept for the **declarative agent-operations** layer —
> the model that lets a backlog item or an initiative run a *named, versioned
> operation* whose implementing operating mode is chosen by data, not baked into
> code. This layer sits **above** operating modes: [`EXECUTION-MODES.md`](./EXECUTION-MODES.md)
> is the SSOT for how one methodology loop *runs*; this document is the SSOT for
> how work *selects* which loop runs, correlates the runs, and attributes them.
> The typed contracts and semantic validators live in
> `api/internal/agentops/` (schemas under `api/internal/agentops/schemas/*.json`);
> the wire vocabulary is `packages/proto/schemas/swarm-manager/v1/domain/agent_operations.proto`.

## Why this layer exists

Today every autonomous agent run against a backlog item or initiative is a
bespoke call site: research spawns an agent one way, workshop another, execution
another, review another (the Phase-1 [agent cutover ledger](../internal/AGENT-CUTOVER-LEDGER.md)
classified 14 such target-bound spawns). Each hard-codes *what* to run and *how*
to run it in Go. That makes the methodology illegible, unversioned, and
impossible to vary per target without editing code.

The declarative model separates the two questions:

- **What** is the operation? — a provider-neutral **operation contract**
  (`review-round`, `execution-run`, …) declaring required capabilities, typed
  inputs, typed results, standard outcomes, and evidence expectations.
- **How** is it run? — a **binding** selects an **operating mode** (at an exact
  revision) to implement the contract for a given target, resolved by
  deterministic precedence.

One coherent agentic unit = one operating-mode execution, correlated by a
durable **domain workflow instance**. Every execution pins an immutable
**provenance** record before it starts. State transitions choose a **domain
action** from a closed registry — never arbitrary code.

## The architecture decision table

These ten concepts are deliberately distinct. Conflating any two is the failure
this layer exists to prevent. Nouns are things; verbs are the registered actions
domain code performs — they never mix (there is no `backlog-item-workshop`).

| Concept | What it is | Kind | Authority (SSOT) | Example | Never |
|---|---|---|---|---|---|
| **Target kind** | The unit of work an operation runs against | data enum | operating-mode + agentops vocab (kept identical) | `backlog-item`, `initiative`, `plan-execution` | a verb; a mode; a plan-ref *field* |
| **Target capability** | A typed capability a target adapter *provides* | data (closed enum) + Go registry | `agentops.TargetCapabilities()` | `provides-review-artifacts` | invented per target |
| **Operation contract** | A named, versioned *what*: required capabilities, inputs, result, outcomes, evidence | data document | `operation-contract.schema.json` | `review-round@1.0.0` | naming a mode, func, service, path |
| **Mode definition + revision** | The *how*: a methodology loop as data, at an immutable revision | data folder | `EXECUTION-MODES.md` / `operating-mode.schema.json` | `holistic-loop@<digest>` | implementing a contract implicitly |
| **Binding** | Which mode+revision implements a contract for a scope | data document | `operation-binding.schema.json` | initiative-override → `holistic-loop@r1` | silently defaulting |
| **Domain action** | A registered, typed mutation domain code performs | closed Go registry + enum | `transition-policy.schema.json` `ActionName` | `complete-item`, `bind-plan` | an arbitrary Go/shell/path reference |
| **Transition policy** | Data selecting *which action* fires on *which outcome* from *which state* | data document | `transition-policy.schema.json` | `running + completed → open-review` | encoding behavior in data |
| **Workflow instance** | Durable correlation of a target's executions, decisions, timers, terminal state | domain document | `workflow-instance.schema.json` | `wf-<id>` beside the initiative | replacing domain truth |
| **Operating-mode execution** | One concrete run of a mode implementing an operation | runtime record | operating-mode engine | a phased-plan-drain run | spanning two operations |
| **Execution provenance** | The immutable record of everything that determined a run | immutable document | `execution-provenance.schema.json` | pinned digests + binding + target | being partial |
| **Evidence** | Canonical facts a completed operation is expected to have produced | data predicate | evidence ledger / contract `evidence_expectations` | `plan / authored` | a mode-specific vocabulary |

## How a run is decided (end to end)

1. A target (a backlog item or initiative) needs an **operation** run — say
   `review-round`.
2. The operation contract declares **required capabilities**
   (`provides-review-artifacts`). `CheckOperationTargetCompatibility` verifies
   the target kind provides them, failing **closed** with the missing set
   otherwise. Because both `backlog-item` and `initiative` provide
   `provides-review-artifacts`, one `review-round` contract is shareable across
   both.
3. A **binding** is resolved by deterministic precedence
   (`authorized-invocation` > `backlog-item-override` > `initiative-override` >
   `system-default`). The resolver snapshots the winning layer and the exact
   mode revision, and fails **closed** on every ambiguous/invalid state:
   - **absence** → typed `ErrNoBinding` (never an implicit default),
   - **invalid/disabled winner** → `ErrInvalidOverride` with **no** fallback to
     a lower layer,
   - **deleted revision** → `ErrDeletedRevision` (operator must re-bind),
   - **incompatible mode** → `ErrIncompatibleMode`.
4. An **execution provenance** record is pinned (operation+version, binding
   layer+owner, mode+revision, compiled-mode digest, prompt-catalog
   revision+digest, target, caller-input digest, policy revision, workflow id).
   A partial provenance can never authorize a run.
5. The chosen **operating mode** runs as **one execution**
   (`EXECUTION-MODES.md`). Its terminal outcome is one of the contract's
   declared **outcomes**.
6. The **domain workflow instance** correlates the execution (by idempotency
   key), and the **transition policy** selects the next **domain action** from
   the closed registry based on `(state, outcome)`. Domain code performs the
   action under its own invariants; data only *chose* it.

## Initiative member-item coordination is strategy, not a mode

The old phase-less `item-level` "operating mode" was a mode in name only — it
had no phase graph; it just meant "run each member item through its own
pipeline." That is not a methodology loop, so it is not a mode. In the
declarative model it is **member-item strategy configuration**
(`member-item-strategy.schema.json`) on the initiative's workflow instance: a
`strategy` (`parallel-items` | `sequential-items` | `prioritized-items`) plus
the per-item `item_operation` to run. A genuine initiative *loop* (holistic-loop)
remains a real operating mode with an `initiative` target. See
`EXECUTION-MODES.md` for the distinction between an initiative-target mode and
member-item coordination.

## Why data can never become arbitrary code

The one executable vocabulary a transition policy can name is the **closed
domain-action registry** (`ActionName`). A policy document has no
`command`/`handler`/`exec`/`path` field anywhere — every object is
`additionalProperties:false`, and action parameters are JSON scalars only (no
nested structures where a path or command could hide). Adding an action is a
domain-code change that registers a typed handler, never a data edit that
invents a name. This is what keeps data-authored methodology from turning into
data-authored code execution.

## References

- [`EXECUTION-MODES.md`](./EXECUTION-MODES.md) — the operating-mode engine SSOT (how one loop runs).
- [`../internal/AGENT-CUTOVER-LEDGER.md`](../internal/AGENT-CUTOVER-LEDGER.md) — the 14 target-bound behaviors this layer replaces.
- [`../operations/migration/LEGACY-MAPPING.md`](../operations/migration/LEGACY-MAPPING.md) — every Phase-1 legacy identity's destination in this model.
- `api/internal/agentops/schemas/*.json` — the authored contract schemas.
- `packages/proto/schemas/swarm-manager/v1/domain/agent_operations.proto` — the wire vocabulary.

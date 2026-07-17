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

Before the declarative-operations cutover, every autonomous agent run against a
backlog item or initiative was a bespoke call site: research spawned an agent one
way, workshop another, execution another, review another (the Phase-1
[agent cutover ledger](../internal/AGENT-CUTOVER-LEDGER.md) classified 14 such
target-bound spawns). Each hard-coded *what* to run and *how* to run it in Go,
which made the methodology illegible, unversioned, and impossible to vary per
target without editing code. That cutover is **complete**: those call sites are
gone, every autonomous spawn now flows through the generic operation runner, and
an architecture test fails the build if a domain package spawns directly. The
persisted-state migration that moved historical runs onto this model finished at
epoch 1 (125/125 staged and promoted, 0 quarantined); see
[Migration, as a completed fact](#migration-as-a-completed-fact).

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
| **Target kind** | The unit of work an operation runs against | data enum | operating-mode + agentops vocab (kept identical) | `backlog-item`, `initiative`, `plan-execution`, `scenario` | a verb; a mode; a plan-ref *field* |
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

## Recovery and reconciliation

The runtime is durable and fail-closed, so a crash or a lost delivery never
leaves silent corruption:

- **Workflow instances** are a durable canonical projection committed by
  compare-and-swap with atomic writes; a stale writer loses the swap rather than
  clobbering a newer commit.
- **Orphan-snapshot reconciliation** runs at startup
  (`opsrunner.ReconcileOrphanSnapshots`) and can be re-triggered on demand
  (`agent-operations reconcile`), reaping execution snapshots left behind when a
  process died mid-run.
- **Completion is single-authority.** A run reaches terminal state exclusively
  through the operation runner's commit path (the opsbridge completion router →
  commit-execution-round). There is no status-poller racing it.
- **Delivery is at-least-once against idempotent commits.** While an operation
  record is still `running`, the refresh driver keeps re-observing its round —
  including a round that is already terminal — so a lost delivery is always
  recovered on a later tick (`operatingmode.RefreshRound` re-fires the
  terminal-round observer; `CommitResult`/`CancelExecution` are idempotent).
- **Canceled rounds reap their operation record.** A cancel can arrive through
  any stop surface (execution cancel, operations bulk-stop, a raw agent-manager
  run stop). A canceled round is not a deliverable outcome, so the completion
  router reaps the running operation record via `opsrunner.CancelExecution`
  instead of committing a result — the workflow records the `canceled` outcome
  and the refresh driver stops polling the stopped run.
- **Uncorrelated active records fail closed.** An active execution record that
  carries a `RunID` but no operation correlation (`OpExecutionID`) is impossible
  after the migration, so the drain marks it failed and lands the item in
  `in_review` rather than stranding it — `execution/polling.go`
  `inspectRunningRecordsLocked`.

## Migration, as a completed fact

The persisted-state migration onto this model is **done** — it is history, not
active work. It promoted **epoch 1** with **125/125** legacy records staged and
promoted and **0** quarantined. The one-shot migrator tooling was deleted after
promotion; the evidence is preserved under
[`../operations/migration/`](../operations/migration/) (see
`P8C-MIGRATION-EVIDENCE.md`), and the live migration-status document is served
read-only through diagnostics (`GetMigrationStatus`, CLI
`agent-operations migration-status`). Imported legacy snapshots remain **readable
and labeled** `[legacy import]`, but are **refused for reproduction** — they have
no mode/binding provenance to pin.

## References

- [`EXECUTION-MODES.md`](./EXECUTION-MODES.md) — the operating-mode engine SSOT (how one loop runs).
- [`../internal/AGENT-CUTOVER-LEDGER.md`](../internal/AGENT-CUTOVER-LEDGER.md) — the 14 target-bound behaviors this layer replaces.
- [`../operations/migration/LEGACY-MAPPING.md`](../operations/migration/LEGACY-MAPPING.md) — every Phase-1 legacy identity's destination in this model.
- `api/internal/agentops/schemas/*.json` — the authored contract schemas.
- `packages/proto/schemas/swarm-manager/v1/domain/agent_operations.proto` — the wire vocabulary.

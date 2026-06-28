# Flows — Plan Manager

## Purpose Of This Document

The workflow and state-transition map for plan-manager. The records these flows
move through are defined in [`PLAN-MODEL.md`](PLAN-MODEL.md). plan-manager is a
*runtime*, so its flows are the product: guided authoring, guided execution with
just-in-time context, and the thin completion/handoff. These are the surfaces that
determine whether a small/local model can drive plan work.

## Flow Inventory

| Flow | Trigger | Owning domain | Terminal states |
|---|---|---|---|
| Authoring | `plan new` / `plan author` | `authoring` | plan `draft` finalized → `active` |
| Phase execution | `plan next` / `plan check` / phase transition | `execution` + `validation` | all phases `done` (or run ends partial) |
| Validation / staleness | on request, or on resume | `validation` | results + staleness tier returned |
| Execution-log capture | `log decision-add` / `finding-add` / `bug-add` / `record-add` / `note-add` | `log` | typed entry persisted; bugs/records forwarded downstream or left `pending` |
| Completion / handoff | `plan complete` | `execution` | canonical handoff assembled (with log summary); plan `complete` (full) or left resumable (partial) |

## Flow Details

### Authoring (Composer)

The wizard walks the **professional section sequence** in order, returning one
guided step at a time so a small/local model never submits a giant blob:

> purpose → problem/need → target outcome → scope → non-goals → assumptions →
> work posture checkpoint (autofilled, reviewed — not authored) → technical
> approach → constraints / prohibited approaches → references → regression anchor
> → global relevant context → validation strategy → definition of done → phases →
> final review.

Mandatory for implementation plans: problem/need, target outcome, technical
approach, validation strategy, and — per phase — ordered steps and phase
validation. Optional sections stay optional only when omission is genuinely safe.

1. Start a plan; the wizard walks sections in order and returns the API-owned
   guided step for the current section. The **work posture** step is autofilled
   from scenario maturity and only reviewed — the agent never writes the
   Greenfield block (see
   [PLAN-MODEL.md](PLAN-MODEL.md#work-posture--greenfieldbrownfield)).
2. As each section is reached, the wizard captures mechanical context behind
   seams: regression anchor (git-control-tower), code references (code-facts),
   and relevant-context candidate discovery (prompt-manager/search-hub/cli-health).
   Discovery stores pending candidates; the author accepts useful candidates into
   global or phase-scoped setup, or rejects noisy candidates with a reason. If a
   source is down, the candidate is marked degraded instead of being false-filled.
3. The author creates phase drafts through phase-native steps: title/intent,
   affected areas, ordered steps, expected outputs, references or an explicit
   `NO_CODE_REFS:` reason, phase-scoped relevant context, phase validation,
   acceptance, and optional risks/handoff notes. Each response returns a guided
   step so the agent does not need the full authoring skill in context. The
   renderer then projects these phase fields in a fixed review order (see
   [PLAN-MODEL.md](PLAN-MODEL.md#rendered-markdown--stable-review-artifact)).
4. After the mandatory sections and before phase work, `author continue` surfaces
   an explicit **global relevant-context checkpoint** (step kind
   `global_relevant_context`). The continue loop will not silently bypass
   plan-wide setup context: the agent resolves it by accepting/submitting at
   least one global context item, or by recording an explicit no-context reason
   (`author section-submit <session> --section relevant_context --content
   "NO_CONTEXT: <reason>"`).
5. The structure-validation gate refuses to finalize while a mandatory section is
   empty, a plan/phase has neither references nor a no-code reason, a phase's
   acceptance, ordered steps, or phase validation is empty, a phase's acceptance
   is a verbatim copy of its validation, or authored constraints contradict the
   derived work posture.
6. On finalize the plan moves `draft → active` and is persisted by `plans`.

### Phase execution (Runner)

1. `start`, `status`, `context`, `resume`, `continue`, and `next` return the
   current phase plus relevant-context setup items, reminders, last validation,
   current staleness, and a compact log summary (read from the `log` domain
   through the `LogLedger` seam) — injected just-in-time so the agent does not
   carry it. `context` is read-only; `resume` resolves an execution or plan and
   may move the pointer to an explicit phase without advancing past work.
   Once-per-execution first-run context is emitted exactly once even when the run
   is **created via `continue`/`resume`** (a brand-new run started through that
   path uses the start context mode, so `once_per_execution` items are not
   skipped).
2. The agent records its work products *as it goes* through the `log` domain
   (`log decision-add`, `log finding-add`, `log bug-add`, `log record-add`,
   `log note-add`) — these are typed ledger entries, not execution/phase fields.
   Findings file as candidate; bugs/records are forwarded downstream internally.
3. `check` runs the phase's computed baseline/validation set and returns results +
   staleness; phase status advances by typed transition or by inference once
   acceptance + validation pass.
4. Resume point and full/partial completion are computed from the phase-status set.

### Completion / handoff

1. `plan complete` runs a **thin** guided completion process emitting typed,
   Plan-Manager-local nudges — `record_finding`, `file_bug`, `capture_record`,
   `confirm_phase_status` — whose messages point at `plan-manager log ...`
   commands, never an external scenario CLI. It does not do heavy lifting.
2. The finalizer assembles the **canonical** handoff from captured state: phase
   status (full/partial + resume), validation, staleness, and a compact
   `LogSummary` + the run's `log_entries` read from the `log` domain through the
   `LogLedger` seam.
3. Findings are filed as **candidate / unvalidated** for operator triage /
   `log promote`; idempotency keys and attribution-keyed dedup avoid double-filing.
4. The agent's prose final message is **not** captured here — that is the
   orchestration layer's job (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).

## State Machines

**Plan lifecycle:**

```
draft ──finalize──▶ active ──all phases done──▶ complete
  │                   │                            │
  └────archive────────┴──────────archive──────────┴──▶ archived
```

**Phase status:**

```
todo ──start──▶ active ──acceptance + validation pass──▶ done
                  │
                  └──blocked (dependency/validation fails) ──▶ active (retry)
```

**Staleness (recomputed on demand / resume):**

```
fresh ──small diff in refs──▶ lightly_stale ──refs moved/deleted──▶ definitely_stale
```

## Maturity Ladder

- **L3 (now):** authoring, plans, execution, validation, completion/handoff, and
  candidate-finding triage flows are implemented and covered by service,
  handler, CLI, and integration tests.
- **L4 (future):** formal flow model for the phase + handoff state machines
  (resume idempotency, partial-completion correctness), added only when checked
  artifacts and replay tests exist.

## Production Shape

- Flows are synchronous request/response over Connect-RPC; no long-running
  background jobs in v1 (baseline runs are on-request, agent in the loop).
- Cross-scenario reads (code-facts, git-control-tower, validation sources) are
  best-effort and degrade gracefully — a flow never hard-fails because a composed
  source is down; it returns a marked gap.
- Idempotency: `status`/`context` are safe reads; `resume` is safe to repeat for
  the same target phase; `next` is the explicit pointer-advance operation.
  Status transitions are legal-move-checked so a retry cannot corrupt phase
  state.

## Deferred / Unmodeled Flows

- Prose-handoff capture and pass-along across loops — **not applicable** to
  plan-manager; owned by agent-manager/swarm-manager (it requires transcript
  access this scenario must not have).
- Background staleness sweeps / proactive re-validation — deferred; v1 computes
  staleness on demand and on resume, not on a timer.
- Consumer-inversion handshakes (swarm-manager/hygiene/`vrooli plans` delegating
  here) — deferred to OT-P2-002, after standalone proof.

## Cross-References

- [`PLAN-MODEL.md`](PLAN-MODEL.md) — the records these flows move
- [`DOMAINS.md`](DOMAINS.md) — which domain owns each flow
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — composed substrate + the handoff split
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry

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
| Completion / handoff | `plan complete` | `execution` | canonical handoff assembled; plan `complete` (full) or left resumable (partial) |

## Flow Details

### Authoring (Composer)

1. Start a plan; the wizard walks sections in order.
2. As each section is reached, the wizard **auto-fills** the mechanical ones —
   regression anchor (git-control-tower), required-reading (plan-skill-discovery),
   code references (code-facts) — each behind a seam that degrades to a marked gap
   if the source is down.
3. The structure-validation gate refuses to finalize while a mandatory section or
   a phase's acceptance is empty.
4. On finalize the plan moves `draft → active` and is persisted by `plans`.

### Phase execution (Runner)

1. `next` returns the earliest non-`done` phase plus its phase-scoped
   required-reading + reminders, last validation, and current staleness —
   injected just-in-time so the agent does not carry it.
2. The agent records decisions/findings via runner commands *as it goes*
   (in-flow capture).
3. `check` runs the phase's computed baseline/validation set and returns results +
   staleness; phase status advances by typed transition or by inference once
   acceptance + validation pass.
4. Resume point and full/partial completion are computed from the phase-status set.

### Completion / handoff

1. `plan complete` runs a **thin** guided completion process: nudges to record a
   finding, file any bugs, confirm phase status — it does not do heavy lifting.
2. The finalizer assembles the **canonical** handoff from in-flow captured state
   (decisions, validation, candidate findings, staleness, full/partial + resume).
3. Findings are filed as **candidate / unvalidated** for operator triage; the
   finalizer reconciles against run-id attribution to avoid double-filing.
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
- Idempotency: `next`/`check`/`status` are safe to repeat; status transitions are
  legal-move-checked so a retry cannot corrupt phase state.

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

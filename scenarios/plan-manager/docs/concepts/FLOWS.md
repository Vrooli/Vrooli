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

The session is a **form with a derived cursor, not a wizard with state**. The
guided `continue` loop remains the novice spine (one recommended action at a
time for small/local models), but nothing is order-gated: every guided step
carries a full-disclosure **checklist** (every requirement for the touched
scope with `filled`/`missing`/`violation` status), fields accept submission in
any order, and the batch `SubmitFields` RPC (`author submit --set …`,
`phase-submit --set …`, `phase-add --set …`) lands one field, a whole phase, or
the whole plan in one call with per-item accepted/rejected results. An agent
that already knows its content authors a complete N-phase plan in ≤ 3+N
mutation calls. Both context paths satisfy the context gates: typed
`context-submit` items and prose `relevant_context` notes / `NO_CONTEXT:`
reasons. Finalize reports honest persistence: the physical SQLite store path,
the stamped workspace, and the **computed** mirror publish result (`fresh` or
a loud `write_failed` warning — never a default `unknown`); re-running finalize
says `Already finalized` explicitly.

The guided spine walks the **professional section sequence** in order,
returning one recommended step at a time:

> purpose → problem/need → target outcome → scope → non-goals → assumptions →
> work posture checkpoint (autofilled, reviewed — not authored) → technical
> approach → constraints / prohibited approaches → **change boundary
> (`acceptance_allow`/`acceptance_deny`)** → references → regression anchor
> (boundary-native) → global relevant context → validation strategy → definition
> of done → phases → final review.

Mandatory for implementation plans: problem/need, target outcome, technical
approach, **change boundary** (satisfiable by an `OPERATOR_ONLY` reason),
validation strategy, and — per phase — ordered steps and phase validation. The
regression anchor is **boundary-native**: its affected scenarios and tiered
baseline/diff commands derive from the change boundary, so the boundary is
authored before the anchor. Optional sections stay optional only when omission is
genuinely safe.

1. Start a plan; the wizard walks sections in order and returns the API-owned
   guided step for the current section. The **work posture** step is autofilled
   from scenario maturity and only reviewed — the agent never writes the
   Greenfield block (see
   [PLAN-MODEL.md](PLAN-MODEL.md#work-posture--greenfieldbrownfield)).
2. As each section is reached, the wizard captures mechanical context behind
   seams: regression-anchor intent (derived from the change boundary) and
   prompt-manager skill-pack discovery. `author skill-pack` auto-adds returned
   skills as global setup context. Search-hub stays direct: the agent runs
   `search-hub query "<intent>" --type record,doc,skill` when docs/records/code
   context is useful, then submits only durable references or context.
3. The author creates phase drafts through phase-native steps: title/intent,
   affected areas, ordered steps, expected outputs, references or an explicit
   `NO_CODE_REFS:` reason, phase-scoped relevant context, phase validation,
   acceptance, and optional risks/handoff notes. Each response returns a guided
   step so the agent does not need the full authoring skill in context. The
   renderer then projects these phase fields in a fixed review order (see
   [PLAN-MODEL.md](PLAN-MODEL.md#rendered-markdown--stable-review-artifact)).
4. Global relevant context is advisory, not a hard navigation checkpoint. The
   guided step recommends `author skill-pack` and direct search-hub discovery,
   but finalization is blocked only by executable plan-shape gates.
5. The structure-validation gate refuses to finalize while a mandatory section is
   empty, a plan/phase has neither references nor a no-code reason, a phase's
   acceptance, ordered steps, or phase validation is empty, a phase's acceptance
   is a verbatim copy of its validation, or authored constraints contradict the
   derived work posture.
6. On finalize the plan moves `draft → active` and is persisted by `plans`.

#### Authoring response contract (small/local-model shaped)

Authoring commands fall into three response classes so a small-context agent
never has to carry the whole session graph:

1. **Mutation acknowledgement → focused progress + summary.** Normal mutations
   (`section-submit`, `phase-submit`, `phase-add`, `autofill`, `context-submit`,
   `context-update`, `context-remove`, `skill-pack`) **no longer
   return the full `AuthoringSession`**. They return a compact
   `AuthoringProgress` (session id, current section/phase,
   mandatory-sections filled/total, phases complete/total,
   `remaining_required_inputs[]`, `ready_to_finalize`), an
   `AuthoringMutationSummary` that names exactly what changed (object kind/id,
   field, a short human summary like "parsed 4 ordered steps"), the single
   changed object (the item/phase/candidate that was touched), any structure
   violations the change surfaced, and the next `GuidedStep`. `ready_to_finalize`
   is a *structurally ready* hint only — the command-reference seam still runs at
   Finalize and may surface late issues. The `GuidedStep` returned by a mutation
   resolves the **true** next step in the same order the continue loop uses (first
   unfilled mandatory section → global relevant-context checkpoint → first
   incomplete phase → outstanding violation → final review). It never reports
   `final_review` while global context is unresolved or a phase is incomplete — a
   mutation that happens to fill the last mandatory *section* does not jump to
   review.
2. **Continue / resume orientation.** `author continue` (`ContinueAuthoring`)
   returns the single current work item — exactly one of section *or* phase
   (both empty at the review/finalize step) — plus progress, violations, and the
   guided step. It does **not** return the full session.
3. **Explicit full-state reads.** `author get-session` (`GetSession`) returns the
   whole `AuthoringSession` graph; `author preview` (`PreviewPlan`) and
   `plans render` return the full rendered markdown. `author start`
   (`StartSession`) still returns the full session for initial hydration. The UI
   uses **read-after-write** — it issues a mutation, then calls `GetSession` to
   re-hydrate — instead of relying on a session echo from every mutation.

`author preview` applies the **same work-posture derivation** as finalize and
`plans render` (greenfield default, or brownfield from scenario maturity), so the
preview render and the persisted render agree (see
[`../internal/SEAMS.md`](../internal/SEAMS.md) — PosturePreparer).

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
3. Before a phase can be marked `done`, the runner requires a **phase feedback
   checkpoint**. The checkpoint is satisfied by phase-scoped durable feedback
   (decision/finding/bug/record) or by an explicit no-feedback note:
   `plan-manager log note-add <execution> --phase <phase> --title "Phase
   feedback reviewed: none" ...`. This keeps small agents from silently skipping
   feedback capture while avoiding fake findings.
4. The runner creates a producer ticket, displays its exact upstream start/wait
   commands, and later synchronizes typed terminal evidence. It never runs or
   waits for Git Control Tower/Test Genie itself. Phase status can advance only
   after the current execution and scope generation have synchronized PASS.
5. Resume point and full/partial completion are computed from the phase-status set.

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
4. Before normal completion, the runner requires a selector-free final ticket,
   producer-native wait, and synchronized clean full-inventory result. A partial
   handoff remains explicitly incomplete and resumable.
5. The agent's prose final message is **not** captured here — that is the
   orchestration layer's job (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).

## State Machines

### Rendered markdown mirror

Plan mutations persist the structured SQLite record first, then render and
publish the markdown mirror under the repo-contract runtime-home `plans` entry.
The mirror publish is crash-safe at the file level (temp file in the same
directory, then rename). Cross-resource atomicity is intentionally not promised:
if SQLite succeeds and the file write fails, the plan remains canonical and the
mirror metadata records `write_failed`.

`plans render` loads the canonical plan, reads the mirror when its hash and
renderer version match, and repairs from SQLite when the file is missing or
stale. This makes the absolute mirror path useful in terminals, Web Console
links, backups, and broken-server scenarios without making markdown editable
truth.

`plans reconcile` is the operator bulk path. With `--dry-run` it reads canonical
plans and fallback markdown sources, reports which mirrors would be repaired and
which source files would be imported, and performs no writes. The CLI accepts
`--workspace <path>` for repo-relative fallback scans; without it the scenario
CLI sends the current discovered Vrooli repo root when available. API callers
should pass `WorkspaceScope.root` explicitly when resolving `docs/plans` or
`plans` from a workspace. Without dry-run it repairs mirrors from SQLite and
imports markdown sources into new canonical records. Source files remain untouched
unless the caller opts into `--retire-sources`; with that flag,
ReconcilePlans removes only sources proven canonical, newly imported, or
duplicate after provenance/content checks. Parse failures and conflicts always
remain untouched for guided repair or relocation. Current mirror files under
runtime-home `plans` are recognized as projections and skipped rather than
re-imported as duplicate truth.

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
- Consumer-inversion handshakes (swarm-manager/hygiene delegating here) —
  deferred to OT-P2-002, after standalone proof. Root `vrooli plans` has been
  retired in favor of direct `plan-manager` usage.

## Cross-References

- [`PLAN-MODEL.md`](PLAN-MODEL.md) — the records these flows move
- [`DOMAINS.md`](DOMAINS.md) — which domain owns each flow
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — composed substrate + the handoff split
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry

---
name: "implementation-plan-execution"
description: "Execute a Plan Manager plan to its intent rather than its letter: read prior-plan status correctly, decide how far to diverge when the plan is wrong, extend the change boundary instead of writing workarounds, wait properly for long-running validation, and reserve 'blocked' for missing authority"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","execution","planning","implementation","scope","friction"]
  icon: "play"
  status: "active"
  revision: 1
  createdAt: "2026-08-07T00:00:00Z"
  updatedAt: "2026-08-07T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager"]
  origin:
    kind: "authored"
---
## Practice focus: Implementation Plan Execution

Execute a Plan Manager plan to its **intent**, not to its letter. A plan is a
compressed prediction written before the work started; the repository is the
authority on what is actually true. When the two disagree, serve the intent,
record the divergence, and keep going.

This skill is the execution-side counterpart to `implementation-plan-authoring`.
That skill decides what a plan should say. This one decides what to do when the
plan turns out to be wrong, incomplete, or blocked by something nobody predicted.

Required reading:
- `scenarios/plan-manager/docs/reference/cli-commands.md` — the `exec`, `log`,
  and `validate` command surface.

Optional reading:
- `docs/TESTING.md` — the wait protocol for long-running suites and baselines
  (§4 restates only the disposition, not the protocol).
- `prompt-manager skill read scientific-debugging` — when the friction is a
  defect whose cause is not yet understood.

---

### 1. When To Use This Skill

| Situation | Use this skill? | Why |
|---|---|---|
| An operator asked you to implement a named plan | Yes | This is the whole subject |
| You are resuming a partially executed plan | Yes | Divergence decisions recur on resume |
| You are authoring a plan | No | Use `implementation-plan-authoring` |
| You are executing one delegated slice under a slice budget | No | Use `swarm-manager-workflow-phased-plan-slice`; a delegated run has narrower authority on purpose (§6) |
| You are doing unplanned work | No | No plan means no plan-fidelity question |

---

### 2. Prior plans: status is not an ownership claim

Before starting, you will find other plans that look related. Read their status
correctly.

Plan status is **computed from the phase-status set**, not from anyone's
intent:

| Status | What it actually means | What it does NOT mean |
|---|---|---|
| `draft` | No phase has left `todo` | That the plan is unfinished, unvalidated, or still being authored. A finalized, validated, never-started plan reports `draft` forever. |
| `active` | At least one phase left `todo` | That anyone is working on it now. A run abandoned mid-phase reports `active` indefinitely. |
| `complete` | Every phase is `done` | — |

Neither `draft` nor `active` carries recency. Judge ownership from **last
activity**, not status: `plan-manager plans list` and `plans get` print the
decode and the age next to the status for exactly this reason.

**Do not report an existing plan as a reason to stop.** A stale `draft` or
`active` plan covering your subject is evidence to read, not a conflict to
escalate. Read it, reuse what is accurate, and say plainly in your report which
parts you took and which were stale.

---

### 3. The divergence ladder

Friction is expected. The question is never "is this in the plan?" — it is **"do
I understand this well enough to fix it correctly?"**

The ladder is keyed on how far the divergence sits from what the plan wrote
down. Higher tiers need more confidence, not more permission.

| Tier | Divergence | Rule |
|---|---|---|
| **T0** | Environment and operational friction: a stopped scenario, a stale binary, a missing baseline, a broken fixture, an approved dependency that is not installed | Fix it and continue. Never report T0 as a blocker. Restarting a scenario is not a scope decision. Self-repair authority is defined in `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md`. |
| **T1** | An edit outside `acceptance_allow` that a phase already in the plan needs in order to be implemented cleanly — a shared type, a proto, the API shape a handler depends on | Make the edit properly. Run `plan-manager exec boundary-extend <execution> --paths <glob> --reason <why>` first so validation scope follows, then record the divergence in Plan Manager's execution log. **A workaround that stays inside a stale boundary is the failure, not the fix.** |
| **T2** | A defect in a dependency, tool, or adjacent scenario that blocks the phase | Fix it when you understand the cause **and** the plan's stakes justify the detour (§5). Otherwise work around it, `log bug-add`, and continue. Say which you chose. |
| **T3** | Changing what the plan set out to do: adding, removing, or reordering phases; changing the target outcome; changing the chosen design | Do not do this silently. Finish everything else first, then record a candidate revision or a finding and report it. |

**Anti-gaming.** Extending the boundary is not a license to widen scope for
convenience. A T1 extension is justified only when a phase already in the plan
cannot be implemented cleanly without it. "While I was in there" is a T3 in
disguise.

`acceptance_deny` is never extendable. It is the authored prohibition, and
`boundary-extend` refuses it. If the clean change genuinely requires a denied
path, that is a T3: stop, report, and let the operator decide.

---

### 4. "Blocked" is a claim about authority, not difficulty

This is the single rule that most changes execution behavior.

> Report `blocked` only when proceeding requires a **decision, credential, or
> approval you do not have**. Friction you understand and can fix is not
> blocked. Work that is slow is not blocked. A test that fails for a reason you
> can diagnose is not blocked.

Three specific things that are **not** blockers:

- **A long-running command.** Baseline capture and suite validation routinely run
  for tens of minutes. Block ONCE on the producer's own wait verb and let it
  return. Do not poll in a loop — it wastes tokens and changes nothing. Do not
  re-run the command while a run is in flight. Do not decide from elapsed time
  that a run has stalled.
- **A failing test you have not read.** Read the failure first. A failure you can
  explain is work, not a wall.
- **A missing baseline or fixture.** Generate it. That is T0.

Before writing "blocked" in any report, name the specific decision you lack the
authority to make. If you cannot name one, you are not blocked.

---

### 5. Stakes modulate T2, not T0 or T1

A plan records the consequence of failing to deliver it (`implementation-plan-authoring`
§4 requires it). Read it. When it is absent, ask the operator or infer
conservatively from what the plan unblocks.

| Stakes | T2 posture |
|---|---|
| High — the plan gates revenue, a release, a commitment, or other work | Fix the blocking defect properly when you understand its cause. A durable fix beats a workaround that another agent inherits. |
| Ordinary | Work around it, file it with `log bug-add`, and finish the plan. Do not turn a feature plan into a dependency-repair project. |

T0 and T1 do not vary with stakes. Restarting a stopped scenario and touching
the proto that defines the API shape you are changing are correct at any stakes
level; a low-stakes plan does not earn dirtier code.

---

### 6. Boundary of this skill

**In scope:** deciding whether to diverge from a plan, how far, and what to
record; the disposition toward friction and long waits; reading prior-plan
status.

**Out of scope:**
- The mechanics of `exec` / `validate` / `log` — the CLI's `--help` and
  `cli-commands.md` own those, and a copied signature goes stale.
- Authoring or repairing plan content — `implementation-plan-authoring`.
- Delegated slice runs. `swarm-manager-workflow-phased-plan-slice` runs an
  untrusted agent against a slice budget and a fixed write scope; there,
  "stopping honestly beats overreaching" is correct, because the operator is not
  present to accept a scope expansion. **Do not apply this skill's T1 latitude to
  a delegated slice.** The two authority models are different on purpose.

---

### 7. Output Expectations

**Must produce:**
- Every phase either `done` with passing validation, or an explicit statement of
  what was not finished and why.
- An execution-log entry for every T1 and T2 divergence, naming what you
  changed outside the plan and what it served.
- A boundary extension for every edit made outside `acceptance_allow`, so no
  edit sits outside the validation oracle unrecorded.
- A final report that separates: what the plan predicted correctly, where it was
  wrong, and what you did about each.

**Must not produce:**
- A workaround adopted to stay inside a stale boundary.
- A `blocked` outcome that names no missing authority.
- Silent divergence — a plan executed differently than written, with nothing in
  the log ledger saying so.
- A phase marked `done` on evidence gathered before a boundary extension. A
  widened boundary invalidates the prior validation generation; re-run it.

---

### 8. Troubleshooting & Edge Cases

| Symptom | Likely cause | First move |
|---|---|---|
| `boundary-extend` refuses the path | The glob is covered by `acceptance_deny`, or would swallow a denied subtree | This is T3. Stop and report; do not route around the prohibition. |
| Phase validation says the scope generation is stale | The boundary or validation scope changed after the last run | Re-run phase validation. This is the intended cost of a widening, not an error. |
| A prior plan looks like it already does this | Its status is probably being misread (§2) | Check last activity and phase states, then reuse or supersede it explicitly. |
| The plan names a file or command that does not exist | The plan is stale, not the repo | Serve the intent against what the repo actually contains, and `log finding-add` the stale reference. |
| The plan's stakes are unrecorded | Authored before the field existed | Ask the operator once. Default to the ordinary-stakes T2 posture until answered. |
| A validation run has produced no output for a long time | Usually normal (§4) | Let the producer's wait verb return. Investigate only after it returns or times out. |

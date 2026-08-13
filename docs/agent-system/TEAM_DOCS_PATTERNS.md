# Team Docs Patterns

Reference for deciding where team-owned guidance and observations belong. Cite this file when designing a team, adding a plan of record, or deciding where an agent should write a finding.

This is canon. Cites `LAYERS.md` for where each layer lives, `PRIMITIVES.md` for definitions, and `INTAKE_PIPELINE.md` for the typed knowledge-flow pattern.

---

## The Fourth Surface: The Team Instrument

Three surfaces below hold a team's *words* — accepted truth, typed observations, and executable work. A mature team has a fourth that holds its **state and its error signal**: one scenario it reads to answer *what is the state of the world I own, and what should I do next?*

- **Who writes:** the scenario, from live joins. Never a person, and never the instrument's own denominators, which are owned by the capability owners it reads.
- **Who reads:** every member of the team, filtered to the rows their lane owns.
- **Lifespan:** none. Numerators are computed live and never stored, so a stale board is structurally impossible — it is either fresh or honestly unavailable.
- **Growth direction:** a new capability is a new row in a denominator, not new text in a member file. This is what makes a team get *smaller* as the system gets more capable.
- **Health signal:** one address. A team whose members are told to call several domain scenarios to learn their own team's state has no instrument, whatever it declares.
- **Declaration:** `team.json::instrument`. `status: "none"` with a dated `gapMarker` is a valid, honest value; silence is not.

The full contract — the six invariants, the two archetypes, the degradation contract, and what belongs in a denominator — is `path:docs/agent-system/TARGET_MODEL.md`. It is not restated here.

**The rule that decides between this surface and the three below:** if a scenario can answer it at read time, cite the query. Content with a status, a lifecycle, a counter, or a coverage figure belongs to the instrument; judgment frames and evidence belong to the surfaces below. That is the same read-time rule `OPERATING_GRAPHS.md` §"State belongs to scenarios" already states, applied to the question of *which surface* rather than *which document*.

---

## One Durable Truth Surface

Teams that produce durable intent use a **plan of record**. The plan of record is the authoritative documentation surface under `path:docs/<domain>/`.

- **Who writes:** operator-curated through an accepted Swarm Manager work item. Agents propose changes; they do not directly edit canon during normal team operation.
- **Who reads:** owning team, other teams, scenarios, and the operator.
- **Lifespan:** durable. Entries persist and evolve through explicit operator dispositions.
- **Growth direction:** grows only when the team's accepted truth expands.
- **Health signal:** stable, discoverable, and structured. High churn means the plan or its ownership boundary is unclear.
- **Shape:** manifest-declared modules. Use the local `manifest.json` and the shared `path:docs/agent-system/team-plan-of-record.manifest.json` contract as the source of truth.
- **Authority:** that base manifest is the contract; the Go structs in `path:scenarios/prompt-manager/api/memberflow/plan_of_record_manifest.go` are its parser, not a second authority. The JSON Schema that used to sit beside it was enforced by nothing and was deleted rather than kept as a third description of the same shape.

Use a plan of record when at least one of these holds:

1. Other teams or scenarios consume the material.
2. The operator needs a durable strategic frame across many entries.
3. The team needs accepted truth to anchor work and prevent drift.

If none hold, do not create documentation just to have a place to write. Use the team Source Ledger scope for observations and Swarm Manager for executable work.

---

## One Agent-Writable Substrate

Agents write observations, evidence, findings, drafts, and raw lessons to the team's **Source Ledger scope** through `source-ledger journal note` and scoped recall. They do not create sidecar markdown surfaces for working memory.

Typed knowledge topics cover the mutable side of team work:

| Need | Surface |
|---|---|
| Raw signal requiring routing | `*-inbox/<signal-type>/<slug>` |
| Bug report | `bug-inbox/<signal-type>/<slug>` |
| System friction, workaround, inefficiency, or process leak | `friction-inbox/<scope>/<slug>` |
| Research observation | domain topics such as `audience-scan/*`, `competitor-record/*`, `hook-record/*` |
| Audit output | `*-audit/*` |
| Run-derived lesson | `run-lesson-report/*` |
| In-flight artifact | `*-draft/*` |
| Proposed PoR edit surface | `*-canon/*` with `destination_kind = "por_file"` |

Every declared topic must identify its producer, reader, and downstream surface; durable team corpus is stored in the Source Ledger scope rather than a local append-only file.

---

## Promotion Rule

Typed evidence is promoted by its drainer/router, which picks exactly one outcome from the uniform action set defined in `INTAKE_PIPELINE.md` § Promotion / Routing (drop, canonical observation, Swarm Manager work, or a PoR/skill/CLI/Action update).

Agents may freely add typed observations when their member or writer skill declares the output. Promotion into durable truth or executable behavior requires the right gate:

| Destination | Gate |
|---|---|
| Plan of record | operator-curated edit with the approved work item's evidence |
| Skill guidance | Swarm Manager work item accepted by the owning operator |
| Action graduation / prose retirement | accepted Swarm Manager work item |
| CLI or scenario work | accepted work item routed through Swarm Manager |
| Missing capability | Swarm Manager work item with the blocked outcome and evidence |
| Team/member contract | accepted Swarm Manager work item owned by the team |
| Canonical knowledge observation | drainer retags or writes to declared destination prefix |

The router always chooses the smallest useful outcome; `SWARM_MANAGER_WORK.md` defines when a drain may execute an outcome directly versus leave it for operator disposition.

---

## Ledger write rule

Agent-written observations, evidence, findings, drafts, and raw lessons go to the
owning team's Source Ledger scope. A member may write only the scope and topic
surfaces declared by its team contract or writer skill. It does not create a local
append-only file or a second approval record. When the entry needs implementation or
operator judgment, file one evidence-backed Swarm Manager work item and use the
disposition on a later heartbeat.

---

## Traps To Avoid

1. **Ad hoc markdown memory.** If an agent needs to remember an observation, use a typed knowledge topic.
2. **Undrained topics.** A topic without a drainer, reader, or work consumer becomes invisible debt.
3. **Plan-of-record as scratchpad.** PoR is accepted truth, not a workspace.
4. **Double residency.** A concept has one authoritative home. A knowledge observation may cite PoR, but it does not duplicate the PoR's text.
5. **Work-item spam.** Reversible owned-data routing stays in the team ledger. File Swarm Manager work only when an outcome, evidence, or operator disposition is needed.
6. **Implementation by prose.** Stable deterministic work should move toward CLI contracts and Actions per `PROMOTION_LADDER.md`.

---

## Quick Checklist

When deciding where something belongs:

1. **Is it accepted durable truth?** Put it in the owning PoR through an operator-dispositioned work item.
2. **Is it raw or partially interpreted evidence?** Put it in a typed knowledge topic.
3. **Is it a bug?** Use `report-bug` into `bug-inbox/*`.
4. **Is it friction, workaround, inefficiency, missing process, or repeated manual pain?** Use `report-friction` into `friction-inbox/*`.
5. **Is an outcome missing or blocked?** File one evidence-backed Swarm Manager work item.
6. **Is it repeatable improvement?** File one bounded Swarm Manager work item or keep it in the team ledger until it is actionable.
7. **Does it require implementation work?** Route the work item through Swarm Manager.

---

## Related Skills

- `team-coordination-independent` / `team-coordination-leader-led` / `team-coordination-peer` — runtime coordination.
- `team-tool-mapping` — routing a shipped scenario into a team's instrument first, and per-member tool skills only as the fallback for a team that has none.
- `team-capability-consolidation` — turning hand-maintained state into a scenario and re-deriving the roster from what it now enforces.
- `documentation-health` — clarity discipline for durable docs.

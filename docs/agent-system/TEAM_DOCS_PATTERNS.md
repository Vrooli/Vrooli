# Team Docs Patterns

Reference for deciding where team-owned guidance and observations belong. Cite this file when designing a team, adding a plan of record, or deciding where an agent should write a finding.

This is canon. Cites `LAYERS.md` for where each layer lives, `PRIMITIVES.md` for definitions, and `INTAKE_PIPELINE.md` for the typed knowledge-flow pattern.

---

## One Durable Truth Surface

Teams that produce durable intent use a **plan of record**. The plan of record is the authoritative documentation surface under `path:docs/<domain>/`.

- **Who writes:** operator-curated via approved decisions. Agents propose changes; they do not directly edit canon during normal team operation.
- **Who reads:** owning team, other teams, scenarios, and the operator.
- **Lifespan:** durable. Entries persist and evolve through explicit decisions.
- **Growth direction:** grows only when the team's accepted truth expands.
- **Health signal:** stable, discoverable, and structured. High churn means the plan or its ownership boundary is unclear.
- **Shape:** manifest-declared modules. Use the local `manifest.json` and the shared `path:docs/agent-system/team-plan-of-record.manifest.json` contract as the source of truth.

Use a plan of record when at least one of these holds:

1. Other teams or scenarios consume the material.
2. The operator needs a durable strategic frame across many entries.
3. The team needs accepted truth to anchor decisions and prevent drift.

If none hold, do not create documentation just to have a place to write. Use typed knowledge topics, decisions, or swarm-manager work instead.

---

## One Agent-Writable Substrate

Agents write observations, evidence, findings, drafts, and raw lessons to **typed team knowledge topics** through `prompt-manager team knowledge-*` commands. They do not create sidecar markdown surfaces for working memory.

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

Every typed topic must have structural declarations in `topics.json`: who produces it, who drains or reads it, what taxonomy governs it, and what decisions or downstream surfaces it feeds.

---

## Promotion Rule

The promotion path is structural:

```text
typed knowledge topic
  -> drainer/router
  -> delete | canonical knowledge | decision | capability-gap | swarm-manager | PoR/skill/CLI/action update
```

Agents may freely add typed observations when their member or writer skill declares the output. Promotion into durable truth or executable behavior requires the right gate:

| Destination | Gate |
|---|---|
| Plan of record | accepted domain decision, then operator-curated edit |
| Skill guidance | `meta-self-improvement`, `skill-improvement`, or owning skill decision |
| Action graduation / prose retirement | `action-candidate` decision |
| CLI or scenario work | accepted decision routed to swarm-manager |
| Missing capability | `capability-gap` decision |
| Team/member contract | `team-structure-change` or owning team decision |
| Canonical knowledge observation | drainer retags or writes to declared destination prefix |

The router chooses the smallest useful outcome. A weak signal is deleted. A valid observation becomes canonical knowledge. A repeated pattern becomes a decision or capability gap. Implementation work routes through swarm-manager when it is not a fully specified direct write.

---

## Direct Writes

Direct writes are narrow. They are allowed only when `DECISIONS.md` says the accepted decision is fully specified, scoped to the owning team's surfaces, and does not create scenario, CLI, skill, action, infra, or cross-team implementation work.

Even direct writes produce an execution record in `decisions.jsonl`. The audit triangle is:

1. Decision records why the change was accepted.
2. Execution record records what artifact changed.
3. Git or the target artifact records how it changed.

---

## Traps To Avoid

1. **Ad hoc markdown memory.** If an agent needs to remember an observation, use a typed knowledge topic.
2. **Undrained topics.** A topic without a drainer, reader, or decision consumer becomes invisible debt.
3. **Plan-of-record as scratchpad.** PoR is accepted truth, not a workspace.
4. **Double residency.** A concept has one authoritative home. A knowledge observation may cite PoR, but it does not duplicate the PoR's text.
5. **Decision spam.** Reversible owned-data routing stays in knowledge. Decisions are for canon, ownership, blast radius, missing capabilities, and accepted behavior changes.
6. **Implementation by prose.** Stable deterministic work should move toward CLI contracts and Actions per `PROMOTION_LADDER.md`.

---

## Quick Checklist

When deciding where something belongs:

1. **Is it accepted durable truth?** Put it in the owning PoR via decision.
2. **Is it raw or partially interpreted evidence?** Put it in a typed knowledge topic.
3. **Is it a bug?** Use `report-bug` into `bug-inbox/*`.
4. **Is it friction, workaround, inefficiency, missing process, or repeated manual pain?** Use `report-friction` into `friction-inbox/*`.
5. **Is a capability missing and blocking work?** File `capability-gap`.
6. **Is it repeatable improvement but not blocking?** File `meta-self-improvement` or the owning improvement context.
7. **Does it require implementation work?** Route the accepted decision to swarm-manager.

---

## Related Skills

- `team-coordination-independent` / `team-coordination-leader-led` / `team-coordination-peer` — runtime coordination.
- `team-tool-mapping` — tool/skill assignment when team structure changes touch tool wiring.
- `documentation-health` — clarity discipline for durable docs.

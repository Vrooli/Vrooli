# Layers

The single canonical home for the layering rule. Other PoR files cite this one; skills cite this one; nobody restates it.

---

## The layering rule

Every piece of guidance in the agent system has exactly one correct home. The rule:

```
Truth lives in Plan of Record.
Judgment lives in Skills and operator dispositions.
Repeatable composition lives in governed Programs.
Single-command discovery lives in Actions.
Implementation and domain state live in Scenarios, exposed through CLIs and APIs.
Project work and disposition live in Swarm Manager; in-scope progress stays with that work.
Raw learning starts in inboxes and synthesis.
Identity stays in SOUL.md.
Ownership stays in team contracts and responsibilities.
```

No double residency. If the same definition appears in two homes, one is wrong; the migration plan picks one and retires the other.

---

## The classifier

When you have a paragraph and don't know where it belongs, ask what it is *saying*:

```
If it says what is true       -> Plan of Record.
If it says how to decide      -> Skill.
If it composes repeated calls -> governed Program.
If it wraps one command       -> Action.
If it owns state or invariants -> Scenario implementation.
If it records in-scope work   -> existing engagement's log.
If it needs new disposition   -> Swarm Manager backlog / capture.
If it is unverified or one-off -> inbox / synthesis (not permanent).
```

Use the classifier without inferring write authority. Observation-only work returns
its findings to the caller. [SCENARIO_DEVELOPMENT.md](SCENARIO_DEVELOPMENT.md) owns
the distinction between authorized implementation and new work disposition;
[PROMOTION_LADDER.md](PROMOTION_LADDER.md) owns graduation and retirement.

---

## Why the rule matters

The system's improvement velocity depends on this rule. When canon lives in the right home:

- Agents reading skills don't re-derive doctrine from prose. The skill cites canon, the agent reads canon once, the same definition steers every consumer.
- Audits become structural. The team-member capability audit (see `TEAM_MEMBER_ARCHITECTURE.md`) scores each layer independently because each layer has its own home.
- Retirement is mechanical. Once a CLI returns deterministic pass/fail for a workflow, the prose skill that tries to encode that workflow in words can be retired (per `PROMOTION_LADDER.md`).

When canon lives in the wrong home — typically when a skill restates doctrine that should live in PoR — the same paragraph drifts as different copies update at different rates. The 10-layer audit then turns into prose-grep, which misses everything.

---

## Where each layer lives in the file system

| Layer | Location | Examples |
|---|---|---|
| Plan of Record | `path:docs/<domain>/` and `path:docs/agent-system/` | `path:docs/monetization/`, `path:docs/marketing/evidence/research/README.md`, this file |
| Skills | Scenario-owned `skills/<skill-id>/SKILL.md` or Prompt Manager's `store/skills/packs/<pack>/<skill-id>/SKILL.md` | Scenario roles and cross-scenario methods; placement in `SKILL_AUTHORING.md` |
| Programs | `path:scenarios/<owner>/.vrooli/program-runtime/` | Typed composition contracts and sources; protocol owned by Program Runtime |
| Actions | `path:scenarios/prompt-manager/store/actions/<action-id>/` | `scenario.status.show`, framework-health |
| Scenario implementation | `path:scenarios/<scenario>/` and shared owner packages | Domain state and operations; CLI/API are invocation surfaces |
| Backlog | Swarm Manager backlog and captures | filed with `swarm-manager backlog create` or `captures create` |
| Inbox / synthesis | Source Ledger entries under a team scope | `research-inbox/<signal-type>/<slug>` |
| Identity | `path:store/agents/<id>/SOUL.md` | per-agent identity prose |
| Ownership | `path:store/teams/<team>/shared/TEAM.md`, `RESPONSIBILITIES.md`, `roles.json` | per-team contracts |
| Topic flow declarations | `path:store/teams/<team>/members/<member>/topics.json` (per-member); schema canon at `path:docs/agent-system/TOPICS_SCHEMA.md` | intake/output prefixes and taxonomy bindings |
| Signal taxonomies | `path:docs/<domain>/<id>-taxonomy.json` + `path:docs/<domain>/<NAME>_TAXONOMY.md` | per-domain signal vocabulary, dispatch table, evidence rules, destination schemas |

When a topic-prefix crosses team boundaries, the producer's taxonomy owns the front-matter schema; the consumer's taxonomy governs only its own routing. See `INTAKE_PIPELINE.md` § Cross-team schema ownership for the load-bearing rule and `TOPICS_SCHEMA.md` for the validator's resolution semantics.

---

## The lint rule

`team-member-capability-architecture-audit` flags as a smell ("skillless canon residue") any skill whose content includes:

- the layer mantra above (any paraphrase that names ≥3 of: PoR, Skill, Action, CLI, backlog, typed knowledge, identity, ownership)
- the classifier ("If it says X → Y") with ≥3 rows
- the promotion ladder steps (interim → CLI/tool → Action → retire) without citing `PROMOTION_LADDER.md`
- the 10-layer table without citing `TEAM_MEMBER_ARCHITECTURE.md`

Skills carrying canon residue must drop the prose and add `Required reading: docs/agent-system/<file>`. The PoR coherence test in `path:scenarios/prompt-manager/test/agent_system_canon_test.sh` enforces this.

# Layers

The single canonical home for the layering rule. Other PoR files cite this one; skills cite this one; nobody restates it.

---

## The layering rule

Every piece of guidance in the agent system has exactly one correct home. The rule:

```
Truth lives in Plan of Record.
Judgment lives in Skills and operator dispositions.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Swarm Manager backlog.
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
If it says what to run        -> Action.
If it says how it works       -> CLI implementation.
If it says what is missing    -> Swarm Manager backlog.
If it is unverified or one-off-> inbox / synthesis (not permanent).
```

Apply the classifier first. If the answer is "inbox / synthesis," the paragraph is debt — it exists because the permanent solution doesn't yet. The promotion ladder (`PROMOTION_LADDER.md`) describes how it eventually graduates or retires.

---

## Why the rule matters

The system's improvement velocity depends on this rule. When canon lives in the right home:

- Agents reading skills don't re-derive doctrine from prose. The skill cites canon, the agent reads canon once, the same definition steers every consumer.
- Audits become structural. The team-member capability audit (see `TEAM_MEMBER_ARCHITECTURE.md`) scores each layer independently because each layer has its own home.
- Retirement is mechanical. Once a CLI returns deterministic pass/fail for a workflow, the prose skill that tries to encode that workflow in words can be retired (per `PROMOTION_LADDER.md`).

When canon lives in the wrong home — typically when a skill restates doctrine that should live in PoR — the same paragraph drifts as different copies update at different rates. The 9-layer audit then turns into prose-grep, which misses everything.

---

## Where each layer lives in the file system

| Layer | Location | Examples |
|---|---|---|
| Plan of Record | `path:docs/<domain>/` and `path:docs/agent-system/` | `path:docs/monetization/`, `path:docs/marketing/evidence/research/README.md`, this file |
| Skills | `path:scenarios/prompt-manager/store/skills/packs/<pack>/<skill-id>/SKILL.md` | `signal-classifier`, `team-member-capability-architecture-audit` |
| Actions | `path:scenarios/prompt-manager/store/actions/<action-id>/` | `scenario.status.show`, framework-health |
| CLIs | `path:scenarios/<scenario>/cli/` and resource CLIs | `prompt-manager`, `swarm-manager`, `resource-postgres` |
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
- the 9-layer table without citing `TEAM_MEMBER_ARCHITECTURE.md`

Skills carrying canon residue must drop the prose and add `Required reading: docs/agent-system/<file>`. The PoR coherence test in `path:scenarios/prompt-manager/test/agent_system_canon_test.sh` enforces this.

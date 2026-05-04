# Meta-Optimization — Plan of Record

This folder is the **plan-of-record** for friction-side canon owned by the `meta-optimization` team: the friction-report taxonomy, routing rules, and any future shared docs that govern how cross-team friction signal flows into the team. The team's broader self-improvement canon lives in [`docs/agent-system/`](../agent-system/) (PRIMITIVES, LAYERS, PROMOTION_LADDER, TEAM_DOCS_PATTERNS, TEAM_MEMBER_ARCHITECTURE, INTAKE_PIPELINE, SKILL_AUTHORING, DEPRECATION_POLICY, REFERENCE_SCENARIOS); this folder is the friction-specific sister to that hub.

Maintained by the `meta-optimization` team. Operator-curated via `meta-self-improvement` decisions (owned by debt-curator).

See [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../agent-system/TEAM_DOCS_PATTERNS.md) for the pattern definition.

## Start here for agents

Use this README first, then choose the file that matches the work:

| Question | Start with |
|---|---|
| How do I report friction I just hit? | [`FRICTION_REPORT_TAXONOMY.md`](FRICTION_REPORT_TAXONOMY.md) — invoke `prompt-manager skill read report-friction` |
| Why is friction reporting cross-team rather than per-team? | this README §"Why cross-team" |
| What is the cross-team flow? | this README §"Cross-team flow" |
| What is the team's mission? | [`docs/agent-system/REFERENCE_SCENARIOS.md`](../agent-system/REFERENCE_SCENARIOS.md) and the `meta-optimization/team.json` mission field |

## Why cross-team

Friction signals — things that were missing, broken, confusing, slow, undocumented, or harder than they should have been — are observed by every agent on every team during normal work. Without a cross-team intake, those signals get trapped in each team's own knowledge log or notebook, and `meta-optimization` (whose mission is *"applying evolutionary pressure to Vrooli's dev meta-layer"*) loses the signal.

`friction-inbox/*` is the universal-source intake on `meta-optimization` that closes that loop. Any agent on any team files friction once via the `report-friction` skill; the `friction-curator` member triages and routes to the appropriate scoped friction topic. The scoped-topic owner (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) then synthesizes patterns and proposes durable fixes via their existing decision contexts.

This mirrors the bug-inbox pattern on scenario-qa. The two universal observation flows together establish a reusable architectural primitive — see [`docs/agent-system/TOPICS_SCHEMA.md`](../agent-system/TOPICS_SCHEMA.md) § Universal-source intakes.

## Cross-team flow

```
any-team/* ─[report-friction skill]──▶ meta-optimization/friction-inbox/<scope>/<slug>
                                                  │
                                                  ▼
                                  meta-optimization/friction-curator
                                                  │
       ┌──────────────┬──────────────┬────────────┴───────────┬──────────────┐
       ▼              ▼              ▼                        ▼              ▼
friction/toolchain  friction/        friction/prompt-         friction/      handoff to
  /<date>/<slug>    run-execution    team-agent-storage       recurring-    debt-curator
  (toolchain-       /<date>/<slug>   /<date>/<slug>           workaround    (unknown +
  validator)        (run-            (team-agent-             /<date>/<slug> reclass failed,
                    introspector)    optimizer)               (debt-curator) or overflow)
                                                  │
                                                  ▼
                                  friction-triage/<YYYY-MM-DD>
                                  (daily snapshot, supersedesPrevious)
```

The curator is a **router, not an analyst**. Synthesis stays with debt-curator. Deep root-cause analysis stays with the [`conversation-friction-analysis`](../../scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md) skill (used post-hoc on full conversation transcripts). The curator owns no decision contexts; routing is determinate from scope; capability-gaps and other decisions are still raised by the destination scoped-topic owners after they drain the routed entries.

## Folder map

| File | Purpose |
|---|---|
| [`FRICTION_REPORT_TAXONOMY.md`](FRICTION_REPORT_TAXONOMY.md) | Human-readable view of `friction-report-taxonomy.json`. Scopes, severities, schemas, action-selection rules, evidence rules, "what is NOT friction" guard. |
| [`friction-report-taxonomy.json`](friction-report-taxonomy.json) | Machine-readable taxonomy sidecar (loaded by the heartbeat builder; cited by `friction-curator/topics.json`). |

## Editing rules

- **Agents never write to these files directly.** All edits come through operator-approved decisions.
- **Edit context:** `meta-self-improvement` (owned by debt-curator on meta-optimization) covers `FRICTION_REPORT_TAXONOMY.md` updates and friction-report taxonomy schema changes. Scope additions go through the same decision flow with empirical evidence from `friction-triage/*` snapshots.
- **Operator executes edits** on decision acceptance. Commit messages cite the decision id.
- **Drafts are not canon.** Synthesis-in-flux content lives elsewhere (working notebooks or `docs/agent-system/drafts/`); files in this folder are stable PoR.

## Cross-references

- [`docs/agent-system/INTAKE_PIPELINE.md`](../agent-system/INTAKE_PIPELINE.md) — the inbox-router-drain pattern used by the friction-curator.
- [`docs/agent-system/TOPICS.md`](../agent-system/TOPICS.md) — registry of every active topic prefix; meta-optimization entries (including friction-inbox and friction-triage) live there.
- [`docs/agent-system/TOPICS_SCHEMA.md`](../agent-system/TOPICS_SCHEMA.md) — schema reference for `topics.json`; documents `source_team: "*"` (universal-source) semantics that friction-inbox uses.
- [`docs/scenario-qa/BUG_REPORT_TAXONOMY.md`](../scenario-qa/BUG_REPORT_TAXONOMY.md) — the sister universal observation flow on scenario-qa.
- [`docs/scenario-qa/README.md`](../scenario-qa/README.md) — pattern this folder mirrors (paired-doc-and-skill discipline applies).
- [`scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md) — universal writer skill.
- [`scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md) — deeper post-hoc analysis skill.

## Changelog

- **2026-05-03 — Friction observation flow landed.** Added `friction-curator` agent on meta-optimization, `friction-inbox/<scope>/<slug>` universal-source intake, `report-friction` writer skill, and `friction-triage/<YYYY-MM-DD>` daily snapshot topic. Heartbeat trigger paragraph in the Storage Map / Observe subsection now points all agents at `report-friction` for structural friction (with one-off-in-handoff guidance preserved). Producers should prefer the skill over ad-hoc friction notebook entries; the curator routes to the existing scoped friction sub-topics owned by toolchain-validator, run-introspector, team-agent-optimizer, and debt-curator. Sister flow to scenario-qa's bug-inbox; together they establish the universal-observation-flow primitive.

## Future PoR work

Flagged here so future operator-curated decisions can promote them when the substrate calls for it:

- **Friction-routing decision context.** If routing becomes non-determinate (e.g., a sixth scope arrives without a natural sub-member owner), the friction-curator may need a `friction-routing` decision context. Today routing is determinate; defer until real pressure shows.
- **Additional scoped friction topics.** If `unknown` scope exceeds 30% of total intake during the observation window (Phase M of the implementation plan), add new scoped friction topics on existing sub-members — candidates include `friction/skill/*` on skill-optimizer, `friction/docs/*` on a doc owner, `friction/policy/*` on meta-contrarian. Empirical, not speculative.
- **Universal observation flow primitive.** With two instances (bug-inbox, friction-inbox), the pattern becomes worth documenting in [`docs/agent-system/TOPICS_SCHEMA.md`](../agent-system/TOPICS_SCHEMA.md) as a named primitive with three parts: universal-source intake + writer skill + drainer agent + trigger paragraph.
- **Cross-team merge of duplicate friction.** Today the curator may flag `repeats-existing-friction-topic` honestly; future work may add automatic merge of duplicate inbox entries before routing.

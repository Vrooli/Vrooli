## Practice focus: Team Member Capability Architecture Audit

Audit whether a prompt-manager team member has the right capability structure around its work. Use this when a member is vague, workflow-heavy, repeatedly blocked, or dependent on external signals that are not captured cleanly.

Required reading:
- `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md` — the 9-layer model (definitions, score scale, smell catalogue)
- `docs/agent-system/INTAKE_PIPELINE.md` — the Intake → Collection → Analysis → Promotion pipeline
- `docs/agent-system/LAYERS.md` — the layering rule the audit applies
- `prompt-manager skill read capability-extraction`

Optional reading:
- `prompt-manager skill read team-tool-mapping`
- `prompt-manager skill read skill-authoring-practice`

---

### 1. When to Use This Methodology

Use this skill when evaluating a team member, role, or team slice and any of these are true:

| Trigger | Use this? | Typical owner |
|---|---|---|
| Member role is clear but the actual workflow is vague | Yes | `team-agent-optimizer` |
| HEARTBEAT.md contains detailed repeatable methodology | Yes | `team-agent-optimizer`, then `skill-optimizer` |
| Agent files explain a capability but no paired skill exists | Yes | `team-agent-optimizer` |
| Work depends on external/operator-fed/proactive signals but no intake path exists | Yes | `team-agent-optimizer` |
| Member repeatedly raises capability gaps caused by missing collection or tooling | Yes | `team-agent-optimizer` or `toolchain-validator` |
| A single existing skill needs prose cleanup | No | `skill-optimizer` |
| A deterministic command should become an Action | No | `skill-optimizer` action workflow |
| A one-off task needs execution | No | Use the task-specific skill |

The goal is not to make every member elaborate. The goal is to put each kind of instruction in the right layer so the system can improve it over time.

---

### 2. Layer Model

The 9-layer model — definitions, score scale, smell catalogue, and the layering rule — lives in `docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md`. Read it before scoring; do not re-derive the layers from this skill.

This skill scores the layers and routes findings; the layer definitions themselves are canon.

---

### 3. Audit Process

#### Phase 1: Gather the Target

Read only enough context to understand the member's current capability shape:

1. Agent files: `SOUL.md`, `AGENTS.md`, `TOOLS.md`, `agent.json`
2. Team files: `TEAM.md`, `roles.json`, `team.json`
3. Member files: `RESPONSIBILITIES.md`, `HEARTBEAT.md`, `last-handoff.md`, `topics.json`
4. Relevant shared state: decisions, knowledge, audit logs, typed evidence, plan-of-record hubs
5. Existing skill references from the member, team, docs, and graph node if available
6. Structural pipeline state — run once for the team and reuse the output:

   ```bash
   prompt-manager graph topics --team "<team>"           # validation findings (errors + warnings)
   prompt-manager graph drain-status --team "<team>"     # per-prefix queue depth + oldest age
   ```

   The validation report is the source of truth for the four pipeline-layer scores
   in Phase 2; the drain-status output feeds the `stalled_drain` / `piling_inbox`
   warnings copied into Section 5's Validation report subsection.

Use concrete evidence. Quote the current prose or name the missing file/skill/path.

#### Phase 2: Score the Nine Layers

For each layer, classify:

| Score | Meaning |
|---|---|
| `0 missing` | No visible structure |
| `1 weak` | Present but vague, stale, or implicit |
| `2 adequate` | Good enough for current usage |
| `3 strong` | Clear, reusable, and easy to optimize |
| `literal:n/a` | Not relevant for this member |

Do not penalize simple members for missing layers they do not need. For example, a pure reviewer may not need proactive collection. A market researcher probably does.

#### Phase 3: Identify Architecture Smells

Look for these recurring smells:

| Smell | Meaning | Likely fix |
|---|---|---|
| Vague capability | Member says what domain it works on but not how work enters or exits | Add intake and promotion guidance |
| Workflow in heartbeat | Repeatable method lives in `HEARTBEAT.md` | Extract or propose a focused skill |
| Planless skill | Skill exists but no plan-of-record doc says why/when it matters | Add or reference docs hub |
| Skillless canon | Plan-of-record doc exists but no executable skill applies it | Propose paired skill |
| Skillless canon residue | Skill restates canon that lives in `path:docs/agent-system/` (layer mantra, promotion ladder, 9-layer table, etc.) | Drop the prose; cite `docs/agent-system/<file>.md` |
| Mega-skill pressure | One skill handles many unrelated methods | Split into router plus method skills |
| Source ambiguity | External research required but source collection is unspecified | Add collection skill/tool/backlog |
| Passive-only intake | Operator can feed work, but proactive scan path is absent | Add proactive baseline or explicit non-goal |
| Proactive-only scan | Agent searches broadly but ignores operator-fed discoveries | Add inbox/intake from vision walk or team handoff |
| Promotion fog | No rule for observation vs typed evidence vs decision vs backlog | Add promotion/routing matrix |
| Dead-end gap | Member observes missing capability but cannot route it | Add capability-gap or owning optimizer path |

#### Phase 4: Choose the Smallest Useful Fix

Prefer the smallest change that repairs the architecture:

| Finding | Preferred proposal |
|---|---|
| Identity too long or procedural | `agent-improvement`: shrink identity; move method elsewhere |
| Member lane/write surface unclear | `agent-improvement` or `team-structure-change` |
| No plan-of-record home | `team-structure-change` or debt-curator follow-up |
| Missing repeatable method skill | route to `skill-optimizer` via decision or explicit handoff |
| One skill should split | route to `skill-optimizer`; name proposed split |
| Missing source collection tooling | `capability-gap` or team-structure proposal with backlog target |
| Missing operator-fed intake | add shared inbox/log or heartbeat intake step |
| Missing proactive baseline | add heartbeat step or method skill with collection mode |
| Missing promotion matrix | add member guidance or skill section |

Respect ownership. If you are `team-agent-optimizer`, do not author the skill yourself unless the operator explicitly asks. Propose the structural change and route skill work to `skill-optimizer`.

---

### 4. Pipeline scoring (Intake / Collection / Analysis / Promotion)

For members that process signals, evidence, or external information, the four pipeline layers (Intake, Collection, Analysis Method, Promotion / Routing) are derived **structurally** from the member's `topics.json` declarations rather than from prose grep. The pattern, the conventions, and the intake-router-drain mechanism live in `docs/agent-system/INTAKE_PIPELINE.md` — read it once.

Scoring rules (pipeline layers only; other layers stay prose-judgment):

| Layer | `0 missing` | `1 weak` | `2 adequate` | `3 strong` |
|---|---|---|---|---|
| Intake | no `intake[]` entries | one entry but no `taxonomy` (or `taxonomy` does not resolve) | at least one entry with a registered `taxonomy` | validation in `prompt-manager graph topics --team <name>` returns no smells for any intake prefix |
| Collection | no collection skill, no Action, no `external_producers[]` | `external_producers[]` declared but no procedure | a paired collection skill or Action exists | collection is exposed as an Action wrapping one CLI |
| Analysis | no method skill referenced from the taxonomy's dispatch | one method skill but combined with collection | a dedicated method skill is named | multiple method skills declared and the member dispatches to them by classifier-recommended signal type |
| Promotion / Routing | no `output[]`, no `decisions_owned[]`, `raises_capability_gaps: false` | `output[]` declared but no destinations or decisions | `output[]` + at least one of `decisions_owned[]` / `raises_capability_gaps` | `output[]` validates structurally (no orphan-output smells), `decisions_owned[]` are real, capability-gap path exists |

When the member legitimately has no pipeline (a pure reviewer, code-writer, or deterministic-CLI maintainer), score these four layers `literal:n/a` — `topics.json` should be `{}` (or omitted) as a positive declaration that there is no flow.

The audit's structural pipeline scores are populated by:

```bash
prompt-manager graph topics --team "<team>"           # validation report
prompt-manager graph drain-status --team "<team>"     # queue depth + age
```

Other layers (Identity, Ownership, Plan of Record, Skill Surface, Feedback Loop) remain prose-judgment per Phase 1–4 of `### 3. Audit Process`.

---

### 5. Output Contract

When using this skill, produce a concise audit in this shape:

```markdown
### Capability Architecture Audit: <team>/<member>

**Current capability:** <one sentence>

**Layer scores:**
- Identity: <0-3/n/a> - <evidence>
- Ownership: <0-3/n/a> - <evidence>
- Plan of Record: <0-3/n/a> - <evidence>
- Skill Surface: <0-3/n/a> - <evidence>
- Intake: <0-3/n/a> - <evidence>
- Collection: <0-3/n/a> - <evidence>
- Analysis Method: <0-3/n/a> - <evidence>
- Promotion / Routing: <0-3/n/a> - <evidence>
- Feedback Loop: <0-3/n/a> - <evidence>

**Primary smell:** <one smell from Section 3>

**Recommended fix:** <smallest useful structural change>

**Routing:**
- team-agent-optimizer: <agent/team prompt or contract change>
- skill-optimizer: <skill create/split/improve, if any>
- debt-curator: <plan-of-record or typed-evidence promotion, if any>
- capability-gap/backlog: <missing tool/scenario/action, if any>

**Validation report:** (copy verbatim from `prompt-manager graph topics --team <team>`)
- structural errors / warnings touching this member, including any
  `stalled_drain` or `piling_inbox` entries from the drain-status output.
- omit when the member's `topics.json` is `{}` and the report has no findings;
  in that case write `Validation report: clean (no flow declared).`

**Expected delta:** <what improves and how to measure it>
```

If no proposal is warranted, still record the strongest layer and the weakest layer so future audits have a baseline.

---

### 6. Worked Pattern: Marketing Researcher

The marketing researcher is a representative example, not a special case.

Likely diagnosis:
- Identity and ownership are mostly clear.
- Skill surface is weak if all research methods live in heartbeat prose or broad role language.
- Intake should support both operator-fed alpha from the morning vision walk and proactive baseline scans.
- Collection and analysis should not collapse into one vague "research" instruction forever.
- Promotion should distinguish observation, typed evidence, audience scan, decision, skill proposal, and capability-gap.

Likely target architecture:

```text
Research inbox / alpha intake
  -> research router skill
  -> focused method skills
  -> knowledge / decision / skill proposal / capability-gap
```

Example method skills:
- alpha extraction
- audience pain mining
- workflow deconstruction
- competitor positioning scan
- channel format scan
- hook pattern mining
- offer and funnel scan
- skill opportunity scan
- benchmark-adjacent scan

Do not hard-code this exact list into every researcher prompt. Use the audit to propose the smallest next step, then let skill and plan-of-record evolution proceed through the normal meta-optimization loop.

No known operational edge cases for standard usage.

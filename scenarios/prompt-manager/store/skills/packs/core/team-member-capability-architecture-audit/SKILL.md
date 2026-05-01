## Practice focus: Team Member Capability Architecture Audit

Audit whether a prompt-manager team member has the right capability structure around its work. Use this when a member is vague, workflow-heavy, repeatedly blocked, or dependent on external signals that are not captured cleanly.

Required reading:
- `prompt-manager skill read skill-principles`
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

Classify the member's capability across these layers:

| Layer | Belongs in | Good sign | Failure sign |
|---|---|---|---|
| Identity | `SOUL.md` and short agent prose | Compact enduring posture | Long task procedure or volatile rules |
| Ownership | team contract, role, `RESPONSIBILITIES.md` | Clear lane, decision contexts, write surfaces | "Help with X" with no decision/write boundary |
| Plan of Record | durable docs hub | Accepted strategy/canon has a discoverable home | Canon lives only in heartbeat prose or handoff |
| Skill Surface | focused skills | Repeatable workflows have one or more paired skills | One mega-skill or no skill for repeated work |
| Intake | team shared state, inbox, heartbeat, external handoff | Work can arrive through named channels | Operator discoveries disappear into conversation memory |
| Collection | tool skill, Action, CLI, scenario, or collection section | Evidence gathering is explicit and honest | "Research it" without source strategy |
| Analysis Method | practice skill | The reasoning method is reusable and inspectable | Every run reinvents the method |
| Promotion / Routing | contract, skill, decision guidance | Observation vs notebook vs decision vs backlog is explicit | Everything becomes a decision or nothing does |
| Feedback Loop | meta-optimization ownership | Skill/doc/tool gaps route to the right optimizer | Weakness is observed but has no improvement path |

Keep the layers separate:

```text
Truth lives in Plan of Record.
Judgment lives in Skills.
Execution lives in Actions and CLIs.
Implementation lives in scenario code.
Unbuilt work lives in backlog or capability-gap decisions.
Raw learning starts in notebooks and logs.
Identity stays in SOUL.md.
Ownership stays in team contracts and responsibilities.
```

---

### 3. Audit Process

#### Phase 1: Gather the Target

Read only enough context to understand the member's current capability shape:

1. Agent files: `SOUL.md`, `AGENTS.md`, `TOOLS.md`, `agent.json`
2. Team files: `TEAM.md`, `roles.json`, `team.json`
3. Member files: `RESPONSIBILITIES.md`, `HEARTBEAT.md`, `last-handoff.md`
4. Relevant shared state: decisions, knowledge, audit logs, notebooks, plan-of-record hubs
5. Existing skill references from the member, team, docs, and graph node if available

Use concrete evidence. Quote the current prose or name the missing file/skill/path.

#### Phase 2: Score the Nine Layers

For each layer, classify:

| Score | Meaning |
|---|---|
| `0 missing` | No visible structure |
| `1 weak` | Present but vague, stale, or implicit |
| `2 adequate` | Good enough for current usage |
| `3 strong` | Clear, reusable, and easy to optimize |
| `n/a` | Not relevant for this member |

Do not penalize simple members for missing layers they do not need. For example, a pure reviewer may not need proactive collection. A market researcher probably does.

#### Phase 3: Identify Architecture Smells

Look for these recurring smells:

| Smell | Meaning | Likely fix |
|---|---|---|
| Vague capability | Member says what domain it works on but not how work enters or exits | Add intake and promotion guidance |
| Workflow in heartbeat | Repeatable method lives in `HEARTBEAT.md` | Extract or propose a focused skill |
| Planless skill | Skill exists but no plan-of-record doc says why/when it matters | Add or reference docs hub |
| Skillless canon | Plan-of-record doc exists but no executable skill applies it | Propose paired skill |
| Mega-skill pressure | One skill handles many unrelated methods | Split into router plus method skills |
| Source ambiguity | External research required but source collection is unspecified | Add collection skill/tool/backlog |
| Passive-only intake | Operator can feed work, but proactive scan path is absent | Add proactive baseline or explicit non-goal |
| Proactive-only scan | Agent searches broadly but ignores operator-fed discoveries | Add inbox/intake from vision walk or team handoff |
| Promotion fog | No rule for observation vs notebook vs decision vs backlog | Add promotion/routing matrix |
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

### 4. Intake, Collection, Analysis, Promotion

For members that process signals, evidence, or external information, explicitly check this pipeline:

```text
Intake -> Collection -> Analysis Method -> Promotion / Routing
```

**Intake** asks: how does work enter the member's lane?
- operator-fed: vision walk, direct instruction, alpha inbox
- proactive: scheduled scan, known source list, telemetry
- cross-team: decisions, inbox messages, handoff, capability-gap
- internal: logs, knowledge, scenario metrics

**Collection** asks: how is source material gathered?
- supplied source refs
- web/manual research
- scenario API/CLI
- Action
- future capability-gap when source access is unavailable

**Analysis Method** asks: what reusable method interprets the material?
- audience pain mining
- workflow deconstruction
- competitor positioning scan
- hook pattern mining
- post-type discovery
- benchmark-adjacent scan
- domain-specific review rubric

**Promotion / Routing** asks: what happens to the output?
- ignore as low-signal
- append knowledge
- append notebook debt
- update working state
- propose plan-of-record change
- propose skill/action/scenario/backlog work
- route to another team/member

Combined skills are allowed early. A method skill may both collect and analyze when the source is simple. Split collection into its own tool/skill/action when source access becomes reusable, credentialed, scheduled, or deterministic.

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
- debt-curator: <plan-of-record/notebook promotion, if any>
- capability-gap/backlog: <missing tool/scenario/action, if any>

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
- Promotion should distinguish observation, notebook, audience scan, decision, skill proposal, and capability-gap.

Likely target architecture:

```text
Research inbox / alpha intake
  -> research router skill
  -> focused method skills
  -> knowledge / notebook / decision / skill proposal / capability-gap
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

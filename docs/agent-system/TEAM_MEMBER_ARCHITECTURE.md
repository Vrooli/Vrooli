# Team Member Architecture

The 9-layer model for evaluating whether a team member has the right capability structure around its work. This file defines the layers; the audit procedure that scores them lives in the `team-member-capability-architecture-audit` skill.

Cites `LAYERS.md`, `PRIMITIVES.md`, and `INTAKE_PIPELINE.md`. The intake-pipeline section here is a minimal slice; the full pattern lives in `INTAKE_PIPELINE.md`.

---

## The nine layers

Every member has nine independent capability layers. Each lives in a specific home (per `LAYERS.md`); each can be evaluated independently of the others.

| Layer | Belongs in | Good sign | Failure sign |
|---|---|---|---|
| **Identity** | `SOUL.md` and short agent prose | Compact enduring posture | Long task procedure or volatile rules |
| **Ownership** | team contract, role, `RESPONSIBILITIES.md` | Clear lane, decision contexts, write surfaces | "Help with X" with no decision/write boundary |
| **Plan of Record** | durable docs hub (`path:docs/<domain>/`) | Accepted strategy/canon has a discoverable home | Canon lives only in heartbeat prose or handoff |
| **Skill Surface** | focused skills | Repeatable workflows have one or more paired skills | One mega-skill or no skill for repeated work |
| **Intake** | shared state, inbox topic prefixes, heartbeat, external handoff, `topics.json` | Work can arrive through named, declared channels | Operator discoveries disappear into conversation memory |
| **Collection** | tool skill, Action, CLI, scenario, or collection section | Evidence gathering is explicit and honest | "Research it" without source strategy |
| **Analysis Method** | practice skill | The reasoning method is reusable and inspectable | Every run reinvents the method |
| **Promotion / Routing** | contract, skill, decision guidance, `topics.json` | Observation vs synthesis vs decision vs backlog is explicit | Everything becomes a decision or nothing does |
| **Feedback Loop** | meta-optimization ownership | Skill/doc/tool gaps route to the right optimizer | Weakness is observed but has no improvement path |

The principle behind the layers is the layering rule from `LAYERS.md`: each layer holds a different *kind* of guidance. Confusing the layers is the primary source of architecture smells.

---

## The four pipeline layers

Members that process signals, evidence, or external information rely on four layers in sequence:

```text
Intake -> Collection -> Analysis Method -> Promotion / Routing
```

These are the layers that the structural data layer (`topics.json`) makes machine-readable. Phase 5 of the agent-system migration plan rewires the audit skill so that scores for these four layers come from `topics.json` instead of prose grep.

### Intake

How does work enter the member's lane?

- operator-fed: vision walk, direct instruction, alpha inbox
- proactive: scheduled scan, known source list, telemetry
- cross-team: decisions, inbox messages, handoff, capability-gap
- internal: logs, knowledge, scenario metrics

Declared in `topics.json` as `intake[]` entries (each with a `prefix`, a `taxonomy` id, an optional `classifier_skill`, and an optional `source_team`) and `external_producers[]` (named non-team-member sources like `vision-walk` or `operator`).

### Collection

How is source material gathered?

- supplied source refs
- web/manual research
- scenario API/CLI
- Action
- future capability-gap when source access is unavailable

Detected from the presence of a paired collection skill, an Action, or — when neither exists — an explicit `capability-gap` claim. A member with no collection layer for a domain it claims to research is a failure sign.

### Analysis Method

What reusable method interprets the material?

- audience pain mining
- workflow deconstruction
- competitor positioning scan
- hook pattern mining
- post-type discovery
- benchmark-adjacent scan
- domain-specific review rubric

Combined skills are allowed early. A method skill may both collect and analyze when the source is simple. Split collection into its own tool/skill/Action when source access becomes reusable, credentialed, scheduled, or deterministic — that move is governed by `PROMOTION_LADDER.md`.

### Promotion / Routing

What happens to the output?

- ignore as low-signal
- append knowledge
- append synthesis (transient)
- update working state
- propose plan-of-record change
- propose skill / Action / scenario / backlog work
- route to another team/member

Declared in `topics.json` as `output[]` entries (each with a `prefix`, a `destination_kind`, and an optional `destination_team` / `destination_path`), `decisions_owned[]`, `decisions_consumed[]`, and `raises_capability_gaps`.

The full pattern, including the inbox-router-drain mechanic and topic-prefix conventions, lives in `INTAKE_PIPELINE.md`.

---

## Score scale (used by the audit skill)

Layers are scored on this scale:

| Score | Meaning |
|---|---|
| `0 missing` | No visible structure |
| `1 weak` | Present but vague, stale, or implicit |
| `2 adequate` | Good enough for current usage |
| `3 strong` | Clear, reusable, and easy to optimize |
| `literal:n/a` | Not relevant for this member |

Do not penalize simple members for missing layers they do not need. For example, a pure reviewer may not need proactive collection. A market researcher probably does.

---

## Architecture smells

Recurring smells that the audit skill flags. The fix column is a default; specific cases may warrant a different lane.

| Smell | Meaning | Likely fix |
|---|---|---|
| Vague capability | Member says what domain it works on but not how work enters or exits | Add intake and promotion guidance |
| Workflow in heartbeat | Repeatable method lives in `HEARTBEAT.md` | Extract or propose a focused skill |
| Planless skill | Skill exists but no plan-of-record doc says why/when it matters | Add or reference docs hub |
| Skillless canon | Plan-of-record doc exists but no executable skill applies it | Propose paired skill |
| Skillless canon residue | Skill restates canon that lives in PoR | Drop the prose; cite `path:docs/agent-system/<file>` |
| Mega-skill pressure | One skill handles many unrelated methods | Split into router plus method skills |
| Source ambiguity | External research required but source collection is unspecified | Add collection skill/tool/backlog |
| Passive-only intake | Operator can feed work, but proactive scan path is absent | Add proactive baseline or explicit non-goal |
| Proactive-only scan | Agent searches broadly but ignores operator-fed discoveries | Add inbox/intake from vision walk or team handoff |
| Promotion fog | No rule for observation vs synthesis vs decision vs backlog | Add `topics.json` declarations + promotion/routing matrix |
| Dead-end gap | Member observes missing capability but cannot route it | Add `raises_capability_gaps: true` and an owning optimizer path |

---

## Layer separation

The full canonical layering rule and classifier live in `LAYERS.md`. Audits apply that rule to assign findings to the smallest correct fix.

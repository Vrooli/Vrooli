# Decisions

The decision system is the agent system's commit log for plans. Every change in posture — a new audience definition, a deprecated skill, an action graduating from a CLI wrapper, a new SKU in the catalog — has a decision recording the why.

This file is canon. It defines the **decision contexts**, the **lifecycle**, the **routing policy** for what happens after acceptance, and the cross-cutting rules (capability-gap criteria, action graduation, stale-decision policy, cross-team output ownership, inbox backpressure). Cites `PRIMITIVES.md` for the bare definition, `INTAKE_PIPELINE.md` for how observations enter and reach a router, `PROMOTION_LADDER.md` for the prose → CLI → Action lifecycle, and `LAYERS.md` for where each artifact lives.

---

## 1. Decision contexts (the taxonomy)

A decision's `context` field names what kind of change it proposes. Each member declares the contexts it owns (`decisions_owned[]`) and the contexts whose acceptance changes its behavior (`decisions_consumed[]`) in `topics.json`.

The canonical contexts:

| Context | Used when | Owned by |
|---|---|---|
| `audience-update` | A new audience definition lands or an existing one shifts | marketing-crew researcher |
| `channel-strategy-update` | Channel posture changes | marketing-crew researcher |
| `content-publish-proposal` | New content draft proposed for publish | marketing-crew |
| `campaign-launch-proposal` | Campaign launches/changes | marketing-crew |
| `brand-guideline-update` | Brand canon evolves | marketing-crew |
| `notebook-promotion` / `notebook-retirement` | Working-notebook entry matures or expires | curator role on each team |
| `coverage-gap` | Deployed-SKU coverage is stale | marketing-crew |
| `audience-update` follow-ons (per team) | … | varies |
| `catalog-promotion` | Monetization SKU lands or shifts | monetization opportunity-scout / market-validator |
| `revenue-line-decision` | Subscription vs services line moves | monetization |
| `pricing-change` | Pricing posture changes | monetization |
| `meta-self-improvement` | A repeatable pattern wants to become a skill / scenario / config change | any member; audited by meta-optimization |
| `action-candidate` | A skill section meets the criteria to graduate to an Action | skill-optimizer |
| `action-improvement` / `action-deprecation` | An Action is changed or retired | skill-optimizer |
| `team-structure-change` | Team / member shape shifts | team-agent-optimizer |
| `agent-improvement` | Agent identity / methodology changes | team-agent-optimizer |
| `capability-gap` | A blocker is filed (see §5) | any member; consumed by director-swarm |
| `reliability-target-update` | Reliability target shifts | infra-health |
| `instrumentation-roadmap-update` | Instrumentation plan changes | infra-health |

This list grows. New contexts are introduced via `meta-self-improvement` decisions that propose adding the context (the system bootstraps its own context taxonomy through itself).

---

## 2. Lifecycle

A decision moves through these states:

```
proposed -> accepted -> executed -> superseded -> stale
              \
               -> rejected
```

| State | Set by | Meaning |
|---|---|---|
| `proposed` | the owning member | Sitting in the team's pending queue, awaiting operator (or approving-team) review |
| `accepted` | operator / approver | Decision is the live plan |
| `rejected` | operator / approver | Decision is closed; the proposed change does not happen |
| `executed` | the executing agent | A follow-on entry in `decisions.jsonl` recording the commit / artifact that made the change real (see §10) |
| `superseded` | a later decision in the same context | The newer decision must declare `supersedes: <id>` |
| `stale` | the team's stale-decision-handler | An accepted-but-not-executed (or accepted-and-since-irrelevant) decision past the staleness threshold; closed with rationale (see §8) |

A team's `operatingContract.governance.supersession.requiredBeforeNewDecision` enforces that two open decisions in the same context cannot exist at once. If a new proposal arrives while an open decision exists, the old one must be superseded or rejected first.

---

## 3. Direct-write vs swarm-manager — the routing policy

When a decision is **accepted**, its execution can land in one of two places:

- **Direct write** — the executing agent makes the change in-place. The decision itself, plus a follow-on `executed` record, is the audit trail.
- **Swarm-manager work** — the executing agent files a backlog item or initiative in `swarm-manager`; the change lands when that work is executed; commits reference both the decision and the swarm-manager item.

The choice is **scope-driven, not team-driven**.

### Direct write is allowed when **all** of these hold

1. **The decision text fully specifies the change.** The executing agent is transcribing, not designing. ("Update audience X's pain-point list to include Y" is direct-write-eligible. "Improve audience X's pain-point list" is not — that needs design work, which means swarm-manager.)
2. **The change is scoped to one of:** the team's own plan-of-record, the team's own knowledge entries, the team's own working-notebook drafts, the team's own member files (RESPONSIBILITIES.md, topics.json, etc.), or **an existing backlog item / initiative the team already owns** (updating planning artifacts to reflect a planning decision is itself a planning act, not new execution work).
3. **No new scenario, no new scenario CLI verb, no new or modified skill, no new or modified action, no infra change** is part of the change.
4. **No cross-team artifact** is touched. (If marketing's decision modifies marketing's own POR but reads from monetization's POR, that's still direct-write. If it modifies monetization's POR, that becomes a cross-team decision and routes to monetization's queue, not a direct write.)

### Swarm-manager-required when **any** of these hold

- The change creates or modifies a scenario, scenario CLI verb, skill, or action.
- The change touches infra, lifecycle, or shared resources.
- The change spans multiple files outside POR / knowledge / member files.
- The decision says *"build X"* rather than *"write X"* — i.e., the work is open-ended.
- The change is a bug fix in a scenario or shared platform.

### Why this split

Strict-everywhere creates ceremony for trivial edits and trains agents to either batch trivial work into oversized backlog items or to do it silently and lie. Loose-everywhere loses observability and makes it impossible to trace why a skill or action drifted.

Scope-based routing keeps about 80% of changes in the direct-write lane (cheap; decision-as-record covers it) and routes the 20% that need design and execution work through swarm-manager, where it can be staged, sequenced, and audited.

### Direct-write does not skip the audit

Every direct-write decision still produces a follow-on **execution record** in `decisions.jsonl` (see §10). The audit triangle is:

1. The decision itself (proposed → accepted) records the **why**.
2. The execution record (decisions.jsonl follow-on) records the **what** (commit hash, artifact path).
3. The git commit records the **how**.

Without (2), direct-write would lose traceability. Treat the execution record as non-optional.

---

## 4. Router execute-directly vs file-decision threshold

Routers (e.g., `marketing-research-router`, `monetization-opportunity-router`) classify inbox items and choose the smallest useful action. The **file-a-decision threshold** is the rule for when the router escalates rather than executing in place.

### A router may execute directly when **all** of these hold

1. **The action is reversible by the same team.** Adding a knowledge observation, retagging an inbox entry to a canonical surface prefix, dropping a duplicate — all reversible without coordination.
2. **The action is scoped to data the team owns.** Team knowledge entries, team inbox topics, team-internal skill execution.
3. **No POR write, no cross-team artifact, no new external surface** is created.

### The router files a decision instead when **any** hold

- The action would write to plan-of-record (any POR write — even the team's own — goes through a decision; POR is approval-gated by definition).
- The action touches another team's surfaces or output topics.
- The action's blast radius is larger than reversal-by-the-same-team can recover.
- The signal merits operator judgment (audience shift, pricing change, capability claim).

A router that hits the threshold files the decision, retags the inbox entry to indicate "decision filed, awaiting acceptance" (or deletes it if a new entry tracks the decision context), and moves on. It does not block waiting for acceptance.

### Action-creation authorization

Any team may **create and run an Action** without operator approval. Action creation is structurally low-risk: an Action wraps exactly one Vrooli-controlled CLI command, is validated by `prompt-manager action validate`, and inherits the safety properties of the underlying CLI. The same team can create the Action, run it, and document it without filing a decision, regardless of whether the Action will be reused cross-team.

The exception is **Action graduation** — converting a prose skill into an Action and retiring the prose. That is *not* the same as creating a new Action; it removes LLM oversight from a workflow that previously had it. See §6.

---

## 5. Decision vs capability-gap

When a member encounters something it cannot do, the choice between filing a `meta-self-improvement` decision and filing a `capability-gap` is structural:

| Filed thing | When | Example |
|---|---|---|
| `capability-gap` decision | The member is **blocked** because a CLI / scenario / source / tool / Action does not exist | "I cannot scan competitor pricing because no scenario covers credentialed scraping" |
| `meta-self-improvement` decision | The member sees a **repeatable pattern** that would be sharper as a skill / scenario / config / action change, but is not blocked right now | "I keep manually reformatting handoff entries; this should be a skill" |
| Backlog item (via swarm-manager) | The team has **decided** to build something specific | (After acceptance:) "Build `competitor-scrape` scenario" |

Phrased as: capability-gap = "we are blocked because X does not exist." Meta-self-improvement = "we work, but X would make us better." Backlog item = "we have decided to build X."

A capability-gap typically becomes a backlog item (see §7). A meta-self-improvement decision typically results in a skill/config/scenario change that may or may not require swarm-manager.

---

## 6. Capability-gap consumption

Capability-gaps are consumed by **director-swarm during vision-walk Phase 3**. The director-swarm member triaging the gap chooses a default routing:

| Gap shape | Default backlog item kind | Rationale |
|---|---|---|
| Scope unclear; feasibility uncertain; multiple plausible paths | `research` | Investigate and narrow before committing to build |
| Scope clear; maps to a known scenario or feature shape | `idea` | Direct path to feature work; scope is the proposal |
| Scope clear and points to a missing CLI verb on a known scenario | `idea` (scoped to that scenario) | The scenario is the home; add the verb there |
| Cross-cutting work that touches multiple scenarios or teams | `initiative` | Coupled work needs initiative-level coordination |

The gap-filer **may** hint a preferred kind in the decision body, but the consumer makes the call. If a `research` item produces a clear scope, it graduates into an `idea` item via a follow-on decision; the original research item retires.

For backlog item kinds (`idea | research | fix | execute | chore`) and the initiative-vs-backlog-item distinction, see `scenarios/swarm-manager/docs/concepts/`.

---

## 7. Action graduation gate

Promoting a prose skill (or a section of one) into an Action removes LLM oversight from that workflow permanently. This is the most consequential transition in the promotion ladder (`PROMOTION_LADDER.md`), and it is the one place where operator approval is **non-negotiable**.

### The gate

1. **`action-candidate` decision is required.** The skill-optimizer (or the owning member) files the decision with: baseline (prose section, token cost, manual-step count), expected delta, the exact CLI command the Action will wrap, validation evidence, and a measurement plan.
2. **Operator (or approving-team) accepts.** No team self-approves an action graduation, even within the team's own surfaces.
3. **Post-adoption measurement is recorded** as an `executed` follow-on decision: actual delta, run count, error rate after a defined observation window.

Unlike most direct-write decisions, an `action-candidate` decision execution still routes through swarm-manager (the Action authoring + skill prose retirement + measurement is non-trivial work).

### Why operator approval is non-negotiable

Action graduation is the system **forgetting how to think about a workflow**. If the underlying CLI regresses, the prose-retirement step has discarded the redundant judgment that would have caught it. The operator's accept-or-reject is the only check that the workflow is genuinely deterministic enough to deserve that.

---

## 8. Stale-decision policy

### Default: 14 heartbeats

A decision is stale when it has been **accepted but not executed** (or accepted, executed, and since rendered irrelevant by a context shift) for **14 heartbeats** of the owning team's cadence.

### Why heartbeats and not calendar time

A team that heartbeats hourly is doing fast-moving work; its decisions go stale faster. A team that heartbeats monthly is doing strategic work; its decisions stay relevant longer. Calendar time would mark every monthly-cadence decision as stale on every heartbeat ("two weeks elapsed → stale!") and every hourly-cadence team would never realistically catch up. Heartbeats are a self-correcting cadence: a team's own pace defines its own staleness.

### Owner: stale-decision-handler

Each team designates a member as `stale-decision-handler` in its operating contract. The contrarian is a common default, but not required — a team without a contrarian (e.g., infra-health) can designate any member with audit responsibilities. The handler reviews stale decisions each heartbeat and routes each to one of:

- `supersede` — file a replacement decision that supersedes the stale one
- `reject` — close it; no longer applicable
- `still-relevant-note` — annotate that the decision is still valid; the heartbeat counter resets

### Per-team override

A team may override the 14-heartbeat default in its `operatingContract.governance.staleDecisionPolicy`. Document the rationale (as monetization does: "tuned for 3/day operator review rate"). Until per-team variations show a pattern, stick to the default.

---

## 9. Cross-team output ownership

When team A produces knowledge that team B consumes (e.g., marketing's `monetization-benchmark-adjacent/*` entries that monetization's market-validator drains):

- **Producer owns the schema.** The shape, fields, and prefix taxonomy of the entry are defined where the producer writes them. If multiple producers write to the same prefix, they reference one shared schema doc (typically a section in the producer's POR hub).
- **Consumer owns the interpretation.** What the consumer does with each entry — what threshold counts as evidence, what triggers a decision, what gets dropped — is the consumer's call. The consumer does not edit the producer's schema; it adapts.
- **The cross-team edge is registered structurally.** The producer declares `output[].destination_team` in `topics.json`; the consumer declares `intake[].source_team`. The `prompt-manager graph topics` validation catches orphaned edges.

If producer and consumer disagree on schema, escalate via a `meta-self-improvement` decision (typically owned by meta-optimization, since it crosses team boundaries).

---

## 10. Decision-execution record (the third leg of the audit triangle)

Every accepted decision — direct-write or swarm-manager — produces a follow-on **execution record** appended to the team's `decisions.jsonl`:

```jsonc
{
  "kind": "decision-executed",
  "supersedes": null,
  "executes": "<original-decision-id>",
  "executedAt": "<ISO timestamp>",
  "executedBy": "<member-id>",
  "artifact": {
    "kind": "commit | swarm-manager-item | knowledge-entry | por-write",
    "ref": "<commit-sha | swarm-manager-id | knowledge-id | path>"
  },
  "delta": "<one-line summary of the actual change>"
}
```

The execution record closes the loop: decision (`proposed` → `accepted`) records the why; execution record records the what; git / swarm-manager / knowledge / POR records the how.

For direct-write decisions, the execution record is the **only** structured trace beyond git. Skipping it loses traceability and leaves direct-write changes effectively invisible to meta-optimization audits.

---

## 11. Inbox backpressure

Each team's inbox topics (`<inbox-name>/*` per `INTAKE_PIPELINE.md`) have a soft cap on **unrouted entries**. Default: **20 unrouted items**.

When the count of entries under `<inbox-name>/*` (the unrouted set, not yet retagged or deleted) reaches the cap, the team enters **drain-only mode**:

- Producers (vision-walk, cross-team handoff, baseline scans) writing to that inbox receive a soft warning and the producer should defer the new entry until the team is back below the cap.
- The router skill gets priority during the next heartbeat: drain takes precedence over collection / analysis.
- The team's stale-decision-handler reviews whether the backpressure indicates structural under-capacity (file a `meta-self-improvement` decision proposing more router capacity, a sub-router, or a cross-team handoff path) or transient burstiness (do nothing; it'll drain).

A team may override the 20-item soft cap in its operating contract for the same kind of cadence reasons as stale-decision policy. Default first; override only when you have a reason.

---

## 12. Quick decision tree

When a member sees something:

```
Is it raw observation / signal-shaped?
  Yes -> add to <inbox>/<signal-type>/<slug>; let router triage. STOP.
  No  -> continue.

Am I blocked because something does not exist?
  Yes -> file capability-gap decision. STOP.
  No  -> continue.

Is it a repeatable pattern that would be sharper with new structure?
  Yes -> file meta-self-improvement decision. STOP.
  No  -> continue.

Does it touch a domain I own (audience, catalog, brand, ...)?
  Yes -> file the owning context decision (audience-update, catalog-promotion, ...). STOP.
  No  -> handoff to the owning team via cross-team flow. STOP.
```

When a router classifies an item:

```
Stale / not actionable        -> drop (delete inbox entry)
Low-signal but valid          -> retag to canonical surface prefix (knowledge observation)
Capability already exists     -> run existing skill or action; output goes to knowledge
Trivial automation, no LLM    -> create + run new action; output goes to knowledge
Judgment / blast radius       -> file decision with the right context (see §1)
Blocked by missing capability -> file capability-gap (see §5)
```

When a decision is accepted:

```
Direct-write criteria met (§3) -> direct write + execution record in decisions.jsonl
Otherwise                      -> file backlog item or initiative in swarm-manager
                                  -> work executes -> commit references decision +
                                     swarm-manager item -> execution record appends
```

---

## Related canon

- `PRIMITIVES.md` — bare definitions of Decision, Capability-gap, Backlog item, Inbox / synthesis
- `INTAKE_PIPELINE.md` — how observations enter, get routed, and what each routing outcome means at the topic-prefix level
- `PROMOTION_LADDER.md` — the prose → CLI → Action lifecycle that `action-candidate` decisions drive
- `LAYERS.md` — where each artifact (decision, skill, action, knowledge, POR) lives
- `TEAM_DOCS_PATTERNS.md` — plan-of-record vs working-notebook patterns; informs which surfaces are direct-write-eligible
- `TEAM_MEMBER_ARCHITECTURE.md` — the 9-layer audit; layer 8 (Promotion / Routing) and layer 9 (Feedback Loop) consume this file

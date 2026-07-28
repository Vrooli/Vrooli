## Practice focus: Morning Vision Walk

Daily strategic sync and idea generation session — the single interface through which you steer the entire Vrooli project. This skill guides a structured conversation that covers decision triage, strategic review, and open-ended brainstorming, ensuring nothing falls through the cracks while preserving space for creative exploration.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

Optional reading:
- `prompt-manager skill read idea-workshop` (for ideation phase technique)
- `prompt-manager skill read swarm-manager-backlog-tools` (for creating backlog items in the action phase)

---

### **1. When to Use This Methodology**

Use Morning Vision Walk when:
- The user starts their daily session and wants to do their morning walk / strategic sync
- The user says things like "let's do a vision walk", "morning walk", "daily sync", "let's review things"
- The user wants a structured way to catch up on project state AND brainstorm

**Do NOT use** for:
- A specific brainstorming session about one idea (use idea-workshop instead)
- Implementation planning for a defined task (use the plan-manager authoring wizard via implementation-plan-authoring)
- Debugging, coding, or other tactical work

---

### **2. Philosophy and Context**

This skill exists because of a core insight about Vrooli:

**The project's biggest bottleneck is not execution — it's idea generation.** The swarm manager, director team, and agent infrastructure handle execution well. But new capabilities and strategic direction still originate primarily from the human. This skill exists to make that human input as high-leverage as possible by:

1. **Clearing the decision queue first** — so the human isn't carrying unresolved questions while trying to think creatively.
2. **Providing full project context** — so brainstorming is informed by current reality.
3. **Applying the "everything outside Vrooli is an error" frame** — every tool or service the human uses outside this project represents a capability gap and a potential new scenario. Daily chores and manual tasks are nucleation points for brainstorming new capabilities.
4. **Connecting ideas to the bigger picture** — every idea can potentially fit into a bundle (dev tools via LPBS, personal/household via Life OS) and onto the tech tree of all possible software.

**Meta-optimization context:** The meta-optimization team produces decisions about how Vrooli improves itself — skill conversions (prose → programmatic), agent/team structure changes, toolchain violations, run-derived lessons, debt promotion, and framework challenges. These are second-order but compounding — they make every future agent run cheaper and sharper. Phase 5.5 gives them dedicated air time so they aren't crowded out by first-order product decisions. `capability-gap` decisions raised by this team are an exception: they surface in Phase 3 alongside portfolio decisions because director-swarm consumes them.

**Marketing-crew context:** The marketing-crew team produces decisions about Vrooli's external voice — what gets published (content-publish-proposal), what campaigns launch (campaign-launch-proposal), when brand canon evolves (brand-guideline-update, audience-update, channel-update), where coverage on deployed SKUs is stale (coverage-gap), and when typed marketing-craft observations mature into permanent structure. Phase 5.3 handles these; `capability-gap` items raised by marketing-crew members are folded into Phase 3 alongside meta-optimization's (both have director-swarm as consumer).

**Ecosystem context:** How a scenario fits the whole is canonical at `path:docs/concepts/ECOSYSTEM.md` — two axes (functional role × interfaces) plus compound-value and monetization make up the full "ecosystem-fit" frame. When assessing or brainstorming a scenario, look past the app itself: which interface(s) does it serve or enable (direct UI, conversational/agentic, voice, programmatic, embodied), what functional role does it play (meta/self-improvement, interface-enabler, integration, product), and does it raise the system's multiplier or compound with other scenarios? **Bundle fit (below) is the monetization slice of this frame, not the whole of it.**

**Monetization context:** Vrooli's full monetization plan is canonical at `path:docs/monetization/` — see `STRATEGY.md` for principles, `CATALOG.md` for the SKU index, `TIERS.md` for delivery tiers, `REVENUE_LINES.md` for subscription-vs-services discipline. In brief: the business bundle (developer + solopreneur tools, including LPBS, Git Control Tower, and Web Console) is the first active bundle; the lifestyle bundle (personal + household) is the next candidate. Delivery tiers ladder from individual apps → self-hosted → hosted cloud → hardware (north-star only). Each new scenario brainstormed during this walk should be assessed for bundle fit (does it serve the business or lifestyle bundle?), role (headliner or depth?), and compound value within the ecosystem. The `monetization` team surfaces the concrete decisions that come out of tracking this plan.

**Infra-health context:** The infra-health team produces decisions about platform reliability and internal-code quality — runtime patterns surfaced from autoheal/system-monitor history (heal-loops, repeat failures, slow-restart trends, investigation clusters), internal-code audit findings across cli/lifecycle/setup/infra/harness, instrumentation gaps the team needs filled, cross-platform debt for tier-2+ deployment, and proposed reliability-target updates. Phase 5.7 handles these. `capability-gap` items raised by infra-health (typically missing CLI verbs on autoheal, system-monitor, or vrooli core) surface in Phase 3 alongside marketing-crew's and meta-optimization's, since director-swarm consumes them. Plan-of-record lives at `path:docs/infra-health/` (`strategy/RELIABILITY_TARGETS.md`, `evidence/INSTRUMENTATION_ROADMAP.md`, `evidence/CROSS_PLATFORM_LEDGER.md`).

**The long-term vision:** This morning walk should eventually be the *only* thing the human needs to do to steer the project. Everything else — execution, monitoring, deployment, testing — happens autonomously. The walk is where human judgment, creativity, and strategic thinking enter the system.

---

### **3. Prerequisites**

Before starting the conversation, read the vision walk prep deliverable:

```
Read the file: scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md
```

This file is generated daily at 5:00 AM by the vision-walk-prep agent and contains pre-compiled briefing data for all phases below. If the file is missing or stale (timestamp older than 36 hours), note this to the user and offer to gather the information live (this will be slower).

**Do not read the prep deliverable verbatim.** Use it as source material to have a natural conversation. Synthesize, prioritize, and present information conversationally.

**Check for a walk checkpoint.** If `last-handoff.md` contains a `## Walk Checkpoint` section, the previous walk diverged mid-session and did not complete. Before starting Phase 1, summarize the checkpoint to the user (phase left at, what was covered, what's pending, divergence scope + whether it resolved) and offer to resume from the pending phase rather than start fresh. See Section 5's "Explicit Divergence" pattern for how checkpoints are written and consumed.

---

### **4. The Process**

```
┌────────────────────────────────────────────────────────────────────────────┐
│                    MORNING VISION WALK FLOW                                │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ┌───────────┐   ┌──────────────┐   ┌───────────┐   ┌──────────────┐     │
│  │ 1. OPEN   │──▶│ 2. RETRO-    │──▶│ 3. PORT-  │──▶│ 4. STRATE-  │     │
│  │   FLOOR   │   │   SPECTIVE   │   │   FOLIO   │   │   GIST      │     │
│  └───────────┘   └──────────────┘   └───────────┘   └──────┬───────┘     │
│                                                              │             │
│  ┌───────────┐   ┌──────────────┐   ┌───────────┐          │             │
│  │ 5.5 META- │◀──┤ 5. MONE-     │◀──┤           │◀─────────┘             │
│  │ OPTIMIZE  │   │   TIZE       │   │           │                        │
│  └─────┬─────┘   └──────────────┘   └───────────┘                        │
│        │                                                                   │
│        ▼                                                                   │
│  ┌───────────┐   ┌──────────────┐                                         │
│  │ 6. CHORE  │──▶│ 7. BIG       │                                         │
│  │   AUDIT   │   │   PICTURE    │                                         │
│  └───────────┘   └──────┬───────┘                                         │
│                         │                                                  │
│                  ┌──────▼───────┐   ┌───────────┐                         │
│                  │ 8. ACTIONS   │──▶│ 9. WRAP-  │                         │
│                  │              │   │    UP     │                         │
│                  └──────────────┘   └───────────┘                         │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Open Floor**

**Entry criteria:** Session has started, user is ready.

**Actions:**
1. Greet the user warmly but briefly. This is a daily ritual, not a formal meeting.
2. Ask: "Anything urgent or on your mind that you want to tackle first?"
3. If yes, handle it — this could be a quick decision, a concern, or something they noticed. Let them lead.
4. If nothing urgent, transition naturally to Phase 2.

**Exit criteria:**
- [ ] User has raised any pressing concerns, or confirmed nothing urgent
- [ ] Any urgent items have been addressed or noted for a later phase

---

### **Phase 2: Retrospective**

**Entry criteria:** Open floor is clear.

**Actions:**
1. Using the prep deliverable's Retrospective section, present a concise summary of what happened in the past 24 hours.
2. Focus on **what changed**, not exhaustive status — completions, notable progress, new blockers, surprises.
3. Keep it to 1-2 minutes of speaking time. This is awareness-setting, not a deep review.
4. Ask if anything stands out or if they have feedback on any completed items.

**Exit criteria:**
- [ ] User has awareness of yesterday's activity
- [ ] Any feedback on completed work has been captured (note it for Phase 8 if it requires action)

---

### **Phase 3: Portfolio Decisions**

**Entry criteria:** Retrospective complete.

**Actions:**
1. Present pending portfolio decisions from the prep deliverable (max 3). Portfolio decisions here include `capability-gap` items raised by the meta-optimization team (run-introspector or toolchain-validator), the marketing-crew team, and the infra-health team — those live on their respective team queues but are portfolio decisions by design, so they're grouped here, not in Phases 5.3 / 5.5 / 5.7.
2. For each decision: state what's being decided, the recommended option, and why it matters — in conversational language, not formal decision-doc prose. If the decision has an attached contrarian challenge note (any team's `capability-gap` items may), present the skepticism alongside the recommendation so the operator sees both.
3. If the user makes a choice, execute it immediately on the correct team:
   ```bash
   # Director-swarm portfolio decisions
   prompt-manager team decision-accept director-swarm "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   # capability-gap decisions live on the team that raised them
   prompt-manager team decision-accept meta-optimization "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   prompt-manager team decision-accept marketing-crew "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   prompt-manager team decision-accept infra-health "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   ```
   Note: if `decision-accept` is not yet available in the CLI, that is a parity bug — file/surface it and escalate; do NOT work around it by calling the API directly from this skill.
4. If the user wants to defer, note it and move on.
5. If there are no pending decisions, say so briefly and move on.

**Guardrail:** Max 3 decisions in this phase. If there are more, the prep agent already selected the most impactful ones. Do not go hunting for additional decisions.

**Exit criteria:**
- [ ] All presented decisions have been addressed (accepted, rejected, or deferred)
- [ ] Or user wants to move on

---

### **Phase 4: Strategist Decisions**

**Entry criteria:** Portfolio decisions complete.

**Actions:**
1. Present pending strategist decisions from the prep deliverable (max 3).
2. These relate to Command Center metrics, outcome gaps, and strategic direction.
3. Present the prep deliverable's `### Prediction scores` section: matured-prediction verdicts (hit / miss / unmeasurable) and any systematic-misprediction signal — this is the calibration evidence the portfolio loop learns from. If it is empty or inventory-only, say so in one line and move on.
4. Same interaction pattern as Phase 3 — present, discuss, execute or defer.
5. If the strategist lane is paused, note it briefly and name which outcome categories still lack sensors per the Outcomes Charter §"Sensor map".

**Exit criteria:**
- [ ] Strategist decisions addressed, or section noted as not yet active

---

### **Phase 5: Monetization Decisions**

**Entry criteria:** Strategist decisions complete.

**Actions:**
1. Present any monetization team questions from the prep deliverable.
2. These may relate to bundle priorities, app rollout order, pricing strategy, or new revenue stream ideas.
3. Same interaction pattern — present, discuss, execute or defer.
4. If the monetization team is not yet active, note it briefly and move on.

**Exit criteria:**
- [ ] Monetization questions addressed, or section noted as not yet active

---

### **Phase 5.3: Marketing Decisions**

**Entry criteria:** Monetization decisions complete.

**Actions:**
1. Present pending marketing-crew decisions from the prep deliverable (max 3, diversified across contexts — not 3 publish-proposals in a row if other contexts have items).
2. For each decision: state what's being decided, the proposing member (brand-manager / subscription-advertiser / oss-advertiser / publisher / researcher / marketing-contrarian), the recommendation, and **any attached challenge notes from marketing-contrarian**. The contrarian's skepticism is first-class — present it, don't bury it.
3. Context-specific framings:
   - `content-publish-proposal` — "Member X drafted <artifact> for <audience / SKU>. Publish, hold for revision, or reject?" The linked draft is in the marketing-crew draft store; read it before deciding if the summary isn't enough.
   - `campaign-launch-proposal` — "Brand-manager proposes a campaign: <theme> targeting <audience> with launch window <date>. Approve (operator updates `docs/marketing/strategy/CAMPAIGNS.md`), defer, or reject?"
   - `brand-guideline-update` / `audience-update` / `channel-update` — "Proposed edit to plan-of-record: <brief>. Approve (operator edits `docs/marketing/<file>.md`) or reject?"
   - `coverage-gap` — "Deployed SKU <sku> has stale / missing marketing coverage. Direct the advertiser to prioritize refresh, or acknowledge and defer?"
   - `decision-rejection-proposed` — "Marketing-contrarian recommends rejecting or superseding <original-decision> for <failure-mode>. Agree, override, or defer?"
4. Execute the user's choice:
   ```bash
   prompt-manager team decision-accept marketing-crew "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   ```
5. For approved plan-of-record edits, the operator (or you, on the operator's direction) executes the actual file edits. Cite the decision id in the commit message.
6. If the user wants to defer, note it and move on.
7. If there are no pending decisions, say so briefly and move on.

**Disabled-team branch:** If `teams/marketing-crew/team.json` has `"enabled": false`, **or** the team is enabled but every member's heartbeat is disabled (check with `prompt-manager team heartbeat-list marketing-crew`), skip with: "Marketing-crew is not currently running. Once running, this phase will surface publish proposals, campaign launches, brand-canon edits, coverage gaps, and typed-learning decisions." The prep deliverable must distinguish *dormant* (heartbeats disabled, no recent runs) from *quiet* (heartbeats active but nothing raised) — only the latter is genuinely quiet.

**Guardrail:** Max 3 decisions. Prep agent has already prioritized.

**Exit criteria:**
- [ ] All presented marketing-crew decisions addressed (accepted, rejected, or deferred)
- [ ] Or section noted as disabled

---

### **Phase 5.5: Meta-Optimization Decisions**

**Entry criteria:** Marketing decisions complete.

**Actions:**
1. Present pending meta-optimization self-improvement decisions from the prep deliverable (max 3, category-diversified across debt / run-lessons / skills / agents-and-teams / toolchain / framework-meta).
2. For each decision: state what's being decided, the proposing member (skill-optimizer / team-agent-optimizer / run-introspector / toolchain-validator / debt-curator / contrarian), the recommendation, and **any attached contrarian challenge notes**. The contrarian's skepticism is a first-class signal here — present it, don't bury it.
3. For `decision-rejection-proposed` context, frame conversationally as: "Contrarian is recommending we reject or supersede [original decision X] because [failure mode tripped]. Agree, override, or defer?"
4. Execute the user's choice immediately:
   ```bash
   prompt-manager team decision-accept meta-optimization "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   ```
5. If the user wants to defer, note it and move on.
6. If there are no pending decisions, say so briefly and move on.

**Category orientation (first walk after the team is enabled — one-time explanation):**

| Category | What it means |
|---|---|
| Skill/Action conversions | Deterministic prose moving into Vrooli-controlled CLIs and Actions; judgment remains in skills |
| Agent / team structure | Changes to agent prompts, team coordination patterns, role additions/removals, deprecations |
| Run lessons | Durable lessons from specific agent-manager runs that warrant a skill/agent change |
| Toolchain violations | Issues the dev toolchain surfaced against the gold-star reference scenario |
| Debt promotions | Synthesis material from typed evidence topics mature enough to become permanent structure (Plan of Record under `path:docs/agent-system/`, skill, Action, CLI backlog, team-config change, or scenario feature) |
| Framework meta | Contrarian-identified failure modes not covered by the existing seven, or proposals to reject pending decisions |

**Guardrail:** Max 3 decisions in this phase. The prep agent has already diversified across categories; do not go hunting for more.

**Disabled-team branch:** If `teams/meta-optimization/team.json` has `"enabled": false`, **or** the team is enabled but every member's heartbeat is disabled (check with `prompt-manager team heartbeat-list meta-optimization`), skip with: "Meta-optimization team is not currently running. Once running, this phase will surface skill/agent/team/toolchain evolution proposals, run-derived lessons, and debt promotions." The prep deliverable must distinguish *dormant* (heartbeats disabled, no recent runs) from *quiet* (heartbeats active but nothing raised) — only the latter is genuinely quiet.

**Exit criteria:**
- [ ] All presented meta-optimization decisions addressed (accepted, rejected, or deferred)
- [ ] Or section noted as disabled

---

### **Phase 5.7: Infra-Health Decisions**

**Entry criteria:** Meta-optimization decisions complete.

**Actions:**
1. Present pending infra-health decisions from the prep deliverable (max 3, diversified across contexts — not 3 platform-code-findings in a row if other contexts have items).
2. For each decision: state what's being decided, the proposing member (runtime-health-scanner / platform-code-auditor / infra-contrarian), the recommendation, and **any attached challenge notes from infra-contrarian**. The contrarian's skepticism is first-class — present it, don't bury it.
3. Context-specific framings:
   - `runtime-health-finding` — "Runtime-health-scanner observed a pattern across <window>: <pattern>. Proposed action: <action>, as a swarm-manager backlog item of kind `fix` or `execute` (operator routes it there manually today). Approve, defer, or reject?"
   - `platform-code-finding` — "Platform-code-auditor graded <slice> as <grade> on <dimension>. Top finding at `<file:line>`: <pattern>. Proposed action: draft a swarm-manager backlog item of kind `fix` or `execute` (operator routes it there manually today). Approve, defer, or reject?"
   - `instrumentation-gap` — "<member> proposes Vrooli should be collecting <stat>. Without it, <which finding was blocked>. Approve (operator updates `evidence/INSTRUMENTATION_ROADMAP.md` and routes the build proposal), defer, or reject?"
   - `cross-platform-debt` — "Platform-code-auditor identified Linux-only assumption at `<file:line>`. Target tier: <tier>. Approve (operator appends to `evidence/CROSS_PLATFORM_LEDGER.md`), defer, or reject?"
   - `reliability-target-update` — "<member> proposes adjusting target for <component>: <old> → <new>. Reason: <30+ day baseline / consistently-missed-for-non-temporary>. Approve (operator updates `strategy/RELIABILITY_TARGETS.md`), defer, or reject?"
   - `framework-meta` — "Infra-contrarian proposes a new failure mode for the rubric: <name>. Examples: <list>. Approve (rubric expands), defer, or reject?"
   - `decision-rejection-proposed` — "Infra-contrarian recommends rejecting or superseding <original-decision> for <failure-mode>. Agree, override, or defer?"
4. Execute the user's choice:
   ```bash
   prompt-manager team decision-accept infra-health "<decision-id>" --selected "<option-key>" --notes "<user's reasoning>"
   ```
5. For approved `instrumentation-gap` / `cross-platform-debt` / `reliability-target-update` / `framework-meta` decisions, the operator (or you, on the operator's direction) executes the actual `docs/infra-health/` edits proposed in the decision's attached diff. Cite the decision id in the change line.
6. If the user wants to defer, note it and move on.
7. If there are no pending decisions, say so briefly and move on.

**Disabled-team branch:** If `teams/infra-health/team.json` has `"enabled": false`, **or** the team is enabled but every member's heartbeat is disabled (check with `prompt-manager team heartbeat-list infra-health`), skip with: "Infra-health team is not currently running. Once running, this phase will surface platform reliability findings, internal-code audit items, instrumentation gaps, and cross-platform debt." The prep deliverable must distinguish *dormant* (heartbeats disabled, no recent runs) from *quiet* (heartbeats active but nothing raised) — only the latter is genuinely quiet.

**Boundary callout (one-time, on first walk after team is enabled):** "Infra-health watches the platform itself — internal Vrooli code, lifecycle, and patterns across autoheal/system-monitor history. It does NOT replace scenario-qa (scenario code quality) or system-monitor (live alerts). It looks at the *aggregate* across days/weeks, so most findings are about repeat failures or trends, not individual incidents."

**Guardrail:** Max 3 decisions. Prep agent has already prioritized.

**Transition:** At this point, the decision-triage portion of the walk is complete. Signal the gear shift: "That covers the decisions waiting on you. Let's shift to the creative side — what's been happening outside Vrooli?"

**Exit criteria:**
- [ ] All presented infra-health decisions addressed (accepted, rejected, or deferred)
- [ ] Or section noted as disabled

---

### **Phase 6: Outside-Vrooli Signals**

**Entry criteria:** All decision phases complete. Transition to generative/creative mode.

**Actions:**
1. Using the prep deliverable's Life Audit section, reference any previous chore discussions for continuity.
2. Ask both prompts, leaving room for the user to answer either:
   - "What have you been doing outside of Vrooli in the past day or so? Any tools you used, tasks you did manually, things that felt like friction?"
   - "Any posts, bookmarks, workflows, skills, launches, competitor moves, marketing examples, or market patterns you saved or noticed?"
3. **Listen first.** Let the user talk through their day without interrupting.
4. Classify each item as one of two signal families:
   - **Life/tool friction:** chores, manual tasks, personal workflows, tools used outside Vrooli, or adoption gaps.
   - **Alpha extraction:** external posts, saved bookmarks, social threads, workflows, skills, marketing examples, lead-generation ideas, competitor moves, monetization facts, or market patterns.
5. For life/tool friction, briefly note:
   - Is there already a scenario that could handle this? If so, that's a gap in adoption, not capability.
   - Is there no scenario for this? That's a capability gap — a candidate for a new scenario.
6. For alpha extraction, preserve enough raw context for downstream agents:
   - source URL or platform if known
   - the user's raw note about why it seemed interesting
   - likely signal type: `workflow`, `skill`, `audience-pain`, `competitor`, `hook`, `channel-format`, `funnel`, `benchmark`, `capability-gap`, or `unknown`
   - likely owner if obvious; otherwise leave it for Phase 8 routing
7. Play back what you heard:
   - "So it sounds like [X, Y, Z] are areas where Vrooli isn't helping yet."
   - "And these external signals seem worth routing for follow-up: [A, B, C]."
8. Present any suggested prompts from the prep deliverable that the user didn't already cover.

**The frame to maintain:** Every non-recreational activity done outside Vrooli is an opportunity. External alpha is not evidence by itself; it is source material for the right downstream agent. The tone should be curious and constructive, like a partner noticing patterns, not a manager auditing time.

**Signal-inbox posture:** When the operator talks about saved posts or bookmarks, assume the future durable collection layer is signal-inbox. Do not invent platform-specific intake workflows during the walk. Capture the signal and route it; later agents can consume signal-inbox CLI exports when that scenario exists.

**Exit criteria:**
- [ ] User has discussed recent activities, frictions, or external signals outside Vrooli
- [ ] Capability gaps and alpha signals have been identified and acknowledged

---

### **Phase 7: Big Picture Ideation**

**Entry criteria:** Outside-Vrooli Signals has surfaced some raw material.

**Actions:**
1. Take the life/tool friction and capability gaps from Phase 6 and explore them:
   - What could be built now given current Vrooli capabilities?
   - What would require prerequisite scenarios first?
   - Which bundle would this fit into (dev bundle, Life OS, or a future bundle)?
   - Which interface(s) would it serve or enable, and what functional role would it play (meta / interface-enabler / integration / product)? (See `path:docs/concepts/ECOSYSTEM.md`.)
   - Does this unlock compound value (does building X make Y and Z possible)?
2. Take the alpha extraction signals and explore only enough to route them well:
   - Is this a marketing research signal, monetization signal, meta-optimization signal, product/scenario idea, or unclear strategic note?
   - Is the next step research, skill/action/scenario proposal, backlog item, or just preserving the source?
   - What raw context must be preserved so the downstream agent does not have to reconstruct the conversation?
3. If the user naturally gravitates toward a specific idea, follow them deeper — use the idea-workshop convergence patterns (listen, synthesize, sharpen, converge).
4. Read the prep deliverable's `### Self-Improvement Loop Ladder` and surface any loop whose claimed level changed since the last walk. A loop claiming above Level 0 must show its measured evidence — challenge the claim if none is attached. This grounds ideation in which self-improvement loops are actually running versus merely built.
5. If no strong ideas emerge from outside-Vrooli signals, use the prep deliverable's Big Picture Context to suggest:
   - Stalled initiatives that might benefit from fresh thinking
   - Unexplored areas of the bundle roadmap
   - Cross-cutting patterns noticed across recent work
6. **Tech tree integration (future):** When the tech-tree-designer scenario is available, use it here to map ideas onto the hierarchy, assess feasibility based on prerequisite nodes, and identify frontier opportunities. Until then, do this assessment conversationally.

**Convergence patterns (from idea-workshop):**

| Signal | Action |
|---|---|
| User is excited about a specific idea | Follow them deep — synthesize, ask sharpening questions |
| User is exploring broadly | Present options, help them compare |
| User seems stuck | Suggest angles from the prep deliverable or bundle roadmap |
| User says "that's enough" or shifts energy | Transition to Phase 8 |

**Guardrail:** Do not create backlog items during this phase. This is exploration time. Capture ideas mentally and hold them for Phase 8.

**Exit criteria:**
- [ ] At least one idea has been explored (even if briefly)
- [ ] User is ready to move to actions, or explicitly wants to wrap up

---

### **Phase 8: Actions**

**Entry criteria:** Ideation complete, user is ready to commit.

**Actions:**
1. Summarize everything discussed that could become actionable:
   - Decisions already executed in Phases 3-5 (just confirm these are done)
   - Feedback from Phase 2 that needs follow-up
   - Ideas from Phases 6-7 that the user wants to pursue
   - Alpha extraction signals that should be routed for downstream research or optimization
2. For each actionable item, ask: "Should this become a backlog item, research inbox entry, team knowledge note, or just a note for later?"
3. Route alpha extraction signals to the narrowest existing intake surface:

| Signal | Preferred destination |
|---|---|
| Marketing audience, channel, hook, post-type, workflow, competitor, funnel, or skill opportunity | `prompt-manager team knowledge-add marketing-crew --caller-note="vision-walk" --topic="research-inbox/<signal-type>/<short-slug>" --content "<raw operator note + flags>" --source="<url-if-known>"`. Signal-type ∈ `audience\|hook\|channel\|competitor\|workflow\|skill\|format\|funnel\|benchmark\|unknown`. |
| Monetization SKU/bundle/add-on/services-line/channel candidate, capability-arrival, customer-ask, competitor move, bundle hint, retention signal, or pricing benchmark | `prompt-manager team knowledge-add monetization --caller-note="vision-walk" --topic="opportunity-inbox/<signal-type>/<short-slug>" --content "<raw operator note + flags>" --source="<url-if-known>"`. Signal-type ∈ `competitor-move\|capability-arrival\|customer-ask\|channel\|bundle-hint\|retention-signal\|benchmark\|unknown`. The `signal-classifier` skill triages; promoted entries retag to `monetization/opportunity/<slug>` (SKU-shaped bets) or `monetization/market-scan/<slug>` (single-snapshot facts). |
| Monetization validation request — operator wants a comp captured, an assumption checked, or a competitor deep-dived | `prompt-manager team knowledge-add monetization --caller-note="vision-walk" --topic="validation-inbox/<request-type>/<short-slug>" --content "<request: who asked, what target, what urgency>" --source="<url-if-known>"`. Request-type ∈ `pricing-comp-needed\|assumption-check\|benchmark-staleness\|competitor-deep-dive\|channel-validation\|unknown`. The `signal-classifier` skill triages; converted entries retag to `monetization/market-scan/<slug>`. |
| Agent/team/skill/process improvement signal | `prompt-manager team knowledge-add meta-optimization --topic "vision-walk-record/alpha/<topic>" ...` if meta-optimization is active; otherwise director-swarm knowledge fallback. |
| Product/scenario idea ready for execution pipeline | Swarm-manager backlog item. |
| Unclear strategic residue or no owner exists yet | Director-swarm knowledge fallback. |
| Missing source collection, automation, CLI, or scenario blocks follow-up | Capability-gap decision or backlog item for the owning team. |

Research-inbox entries are knowledge entries on the `marketing-crew` team under a `research-inbox/<signal-type>/<slug>` topic. The CLI handles concurrency, retention, and listing — do not write JSONL files directly.

- Use `--caller-note="vision-walk"` so the source context is auditable. Knowledge entries are attributed automatically by runtime context; do not pass `--by` to `knowledge-add`.
- Put the operator's raw wording (and why it mattered) into `--content`, plus any honesty/confidence flags.
- Use `--source` for the original URL when available.
- Optional next method goes inline at the end of `--content` (e.g. `suggested-method: hook-pattern-mining`).

The router will retag (or delete) the entry once classified, so the inbox view (`team knowledge-list marketing-crew --topic-prefix=research-inbox/`) is always the unrouted set.

Opportunity-inbox entries follow the same pattern on the `monetization` team: knowledge entries under `opportunity-inbox/<signal-type>/<slug>`, written via the CLI (do not write JSONL files directly). The `signal-classifier` will retag promoted entries to `monetization/opportunity/<slug>` or `monetization/market-scan/<slug>`, or delete dropped ones, so `team knowledge-list monetization --topic-prefix=opportunity-inbox/` is always the unrouted set.

Validation-inbox entries are the same pattern for the market-validator's `validation-inbox/<request-type>/<slug>` topic. The `signal-classifier` triages and converts to `monetization/market-scan/<slug>` or raises a benchmark/pricing/financial-model-assumption decision. The inbox view (`team knowledge-list monetization --topic-prefix=validation-inbox/`) is always the unrouted set.

When signal-inbox exists, route bookmark-heavy alpha through that scenario's CLI/export path instead of manually creating platform-specific intake records. Until then, an inbox knowledge entry with source URL and raw note is the right shape.

4. For backlog items, use the swarm-manager CLI:
   ```bash
   swarm-manager backlog create --data '{
     "name": "<kebab-case-name>",
     "title": "<Title>",
     "kind": "idea",
     "priority": <1-5>,
     "tags": [<relevant-tags>],
     "description": "<description from discussion>"
   }'
   ```
5. For ideas that need more refinement before becoming backlog items and do not fit a team inbox, note them as knowledge entries for the next vision walk:
   ```bash
   prompt-manager team knowledge-add director-swarm --topic "vision-walk-record/chore-audit/<topic>" --content "<what was discussed and where it left off>"
   ```
6. Optionally kick off the idea hardening pipeline for created items:
   ```bash
   swarm-manager backlog research --kind idea --name "<name>" --data '{"mode":"clarify"}'
   ```

**Exit criteria:**
- [ ] All actionable items have been created, deferred, or noted
- [ ] User confirms nothing was missed

---

### **Phase 9: Wrap-up**

**Entry criteria:** Actions complete.

**Actions:**
1. Brief summary: "Today we [handled N decisions, reviewed yesterday's progress, explored ideas about X and Y, created Z backlog items]."
2. Ask: "Any final thoughts or feedback on how this session went?"
3. If the user has feedback about the walk itself (too long, wrong order, missing something), capture it as a knowledge entry for process improvement:
   ```bash
   prompt-manager team knowledge-add director-swarm --topic "vision-walk-record/process-feedback" --content "<feedback>"
   ```
4. Close warmly: "Have a good walk. The agents will take it from here."

**Exit criteria:**
- [ ] Session summarized
- [ ] Any process feedback captured
- [ ] User is done

---

### **5. Convergence Patterns**

#### **Phase Transition Detection**

| Signal | Action |
|---|---|
| User answers a decision quickly and cleanly | Move to next decision or next phase |
| User wants to discuss a decision at length | Allow it, but track time — gently suggest moving on after 5+ minutes on one item |
| User says "skip" or "next" or "move on" | Immediately transition to next phase |
| User says "let's brainstorm" or starts ideating | Jump to Phase 6 or 7 regardless of current phase |
| User says "I need to go" or "let's wrap up" | Jump to Phase 9 |
| User brings up something from a later phase early | Handle it now, mark that phase as partially complete |
| User proposes same-day execution of a discovered improvement | Evaluate Explicit Divergence criteria (below); if met, write checkpoint and diverge |

#### **Explicit Divergence**

The walk's standard shape is *triage now, capture ideas, defer action to Phase 8*. That works when discoveries are small. But sometimes a phase surfaces something important enough that same-day execution beats next-day backlog creation — the operator and agent still have full mental context, and deferring would force a future agent to reconstruct it from cold backlog entries.

For those cases, the walk supports an **Explicit Divergence**: leave the walk mid-session to plan/execute the discovery, then resume the walk later (same day or next day's prep).

**Criteria (both required):**
- **Mutual agreement** — operator and agent both say yes. Not one-sided. Agent may *propose* divergence but must not unilaterally abandon phases.
- **Bounded scope** — write the acceptance criterion for the divergence before leaving. Vague scope ("let's look into it") does not qualify; "plan is authored and ready for handoff" or "item X is merged and scenario restarts cleanly" does.

**No frequency guardrail.** If the walk regularly produces important-enough-to-act-on discoveries, that means the walk is working. Do not discourage divergence on the basis of "we diverged recently."

**Checkpoint protocol:**
1. Before leaving the walk, append a `## Walk Checkpoint (<ISO-timestamp with offset>)` section to `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md`.
2. The checkpoint must contain, in this order:
   - **Status line** — "Walk diverged mid-Phase-N by mutual agreement to …".
   - **Phases covered so far** — one bullet per completed/partial phase with the material outcome.
   - **Phases pending on resume** — one bullet per remaining phase with any relevant pointers from the prep deliverable.
   - **Process frictions captured so far** — numbered list, ready to file in Phase 8 when the walk resumes.
   - **Divergence scope (acceptance criterion)** — explicit pass/fail condition for when the divergence is complete.
   - **Resume protocol** — "re-read this file, skip already-covered phases, pick up at Phase N, re-evaluate <any decisions affected by the divergence>".
3. The `vision-walk-prep` agent preserves the `## Walk Checkpoint` section verbatim when regenerating `last-handoff.md` at 5:00 AM, until the walk resumes and the skill removes it.

**Resume protocol (also in Prerequisites):**
- At Phase 1 of any walk, check for a `## Walk Checkpoint` section. If present, summarize it to the operator and offer to resume. If they accept: skip covered phases, pick up from the stated resume point. If they decline (fresh walk wanted): delete the checkpoint section and proceed normally, noting that pending items from the prior walk may now be surfaced in fresh prep.
- When a resumed walk reaches Phase 9 (Wrap-up), remove the `## Walk Checkpoint` section as the final action so tomorrow's walk starts clean.

**When divergence produces artifacts (plans, merged PRs, backlog items):** cite them by path or ID in the checkpoint's Status line so resume knows what already shipped and does not double-file in Phase 8.

#### **Energy Management**

The walk should feel energizing, not draining. Monitor the user's engagement:

| Signal | Adjustment |
|---|---|
| Short, clipped answers | Reduce detail, move faster through phases |
| Excited, expansive responses | Give more room, especially in ideation phases |
| "I don't know" or "whatever you think" | Offer a recommendation and move on |
| Deep engagement with one topic | Let them explore — rigid phase order matters less than productive conversation |

---

### **6. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Reading the prep deliverable verbatim** | Boring, wastes time, user can read | Synthesize and converse naturally |
| **Spending all time on decisions** | Brainstorming gets squeezed out | Cap decisions at 3 per section, always reach Phase 6 |
| **Creating backlog items during brainstorming** | Premature commitment kills exploration | Capture ideas, formalize in Phase 8 |
| **Judging the user's chores** | Kills openness about what they do outside Vrooli | Frame as opportunity, not error |
| **Skipping phases because nothing's pending** | Misses the structure's value | Briefly note "nothing pending" and move on — the flow matters |
| **Over-engineering ideas during the walk** | Wrong time for implementation details | Keep at "what and why" level, defer "how" |
| **Not preserving alpha source context** | Downstream agents must reconstruct why a post, bookmark, or workflow mattered | Capture URL/platform, raw operator note, initial signal type, and likely owner before routing |
| **Not capturing life audit topics** | Loses continuity between walks | Always write knowledge entries or routed inbox records for outside-Vrooli signals |
| **Treating this as a status meeting** | It's a strategic sync + creative session | Lead with decisions, but the ideation is the most valuable part |

---

### **7. Boundaries**

This skill covers the **daily strategic sync ritual** — from triage through brainstorming to action capture.

**Does NOT cover:**
- **Deep idea refinement** — If an idea needs 30+ minutes of workshopping, transition to the idea-workshop skill
- **Implementation planning** — Use the plan-manager authoring wizard (implementation-plan-authoring) after an idea is in the backlog
- **Execution** — Agents handle execution; this walk is for steering
- **Tech tree design** — When available, the tech tree is consulted here but not designed or restructured during the walk

---

### **8. Output Expectations**

When running a Morning Vision Walk, you **must**:

1. Read the prep deliverable before starting (or note it's unavailable); if it contains a `## Walk Checkpoint` section, offer to resume before starting fresh (see Prerequisites + Section 5 "Explicit Divergence")
2. Cover all 12 phases in order — 1, 2, 3, 4, 5, 5.3, 5.5, 5.7, 6, 7, 8, 9 (skipping is fine if a section is empty, but acknowledge it; Explicit Divergence may exit the walk early per Section 5 — resume under the checkpoint protocol instead of treating remaining phases as skipped)
3. Execute decisions the user approves via CLI commands
4. Create backlog items for actionable ideas via swarm-manager CLI
5. Route alpha extraction signals to the narrowest existing inbox or knowledge surface, preserving source URL and raw operator note
6. Write knowledge entries for chore audit topics discussed when they do not fit a narrower intake surface
7. Summarize the session at wrap-up

You **should** also:
- Keep the overall session under 35-50 minutes of conversation
- Ensure brainstorming (Phases 6-7) gets at least 10 minutes even on heavy decision days
- Maintain a warm, collaborative tone — this is a daily ritual, not a performance review
- Note when sections are "not yet active" (strategist, monetization, marketing-crew, meta-optimization, infra-health) so the user knows what's coming

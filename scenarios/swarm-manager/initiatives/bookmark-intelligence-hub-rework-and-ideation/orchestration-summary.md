# Orchestration Summary — Bookmark Intelligence Hub Rework and Ideation Integration

## Context: how this initiative came to be

This initiative was authored during the **third explicit divergence of vision walk #4 (2026-04-27)**, mid-Phase-6 (chore audit). The chore-audit phase asked the operator what they'd been doing outside Vrooli; the operator surfaced a recurring pattern: they bookmark hundreds of items per day across X, TikTok, YouTube, etc., but almost none make it into Vrooli's idea pipeline because manual-paste-into-system has no ergonomic surface today. Even modest extraction rates from that bookmark stream would compound into a major idea-sourcing pipeline.

Investigation revealed an existing scenario at `scenarios/bookmark-intelligence-hub/` with substantial Go API + React UI infrastructure, multi-platform integration scaffolding (Reddit, X, TikTok), AI-powered categorization (Programming / Recipes / Fitness / Travel / News / Education / Entertainment), and an action-suggestion engine wired to other scenarios (recipe-book, workout-plan-generator, research-assistant, etc.). PRD last updated 2025-11-18.

But the existing design is **structurally over-built for automated scraping and structurally under-built for the manual-paste-first ideation flow the operator actually needs**. Three specific mismatches:

1. **Automated scraping is the core; manual paste is afterthought.** PRD assumes session cookies for X/TikTok and Huginn agents polling for new bookmarks. Operator cannot use X automation without risking a ban; BAS isn't yet mature enough for reliable web scraping. The v0 actually needed: manual paste-in-link + operator description + operator notes, with automated extraction as a *Phase 2 enhancement* once BAS or formal API access matures.

2. **Categories are content-shape, not Vrooli-use-shape.** Built-in categories classify bookmarks by *what they're about*. But the operator's actual examples sort by **how Vrooli should use the bookmark**:
   - *Direct scenario candidate* (e.g., a free-business-idea post → ship as one-off scenario when deployment ramp matures)
   - *Operational reference* (e.g., a marketplace-bidding tactics post → reference for a future flipping scenario)
   - *Marketing positioning material* (e.g., a homelab-built-with-Claude-Code post → bundle positioning material)
   - *Capability-class flag* (e.g., a designed-own-Kindle prompt-to-product post → long-term direction-flag for prompt-to-physical-product class of capabilities)
   - *Personal interest only* (filter out of the Vrooli flow)
   
   That's a fundamentally different taxonomy than "Programming / Recipes / Fitness." The existing taxonomy is "organize my bookmarks"; the new one is "feed Vrooli the right idea-shape per bookmark." The two should coexist — content-shape categories remain useful for downstream consumers (recipes flow to recipe-book, fitness to workout-plan-generator, etc.) — but Vrooli-use is the new orthogonal axis.

3. **No connection to vision-walk-prep.** The most strategic gap. Existing cross-scenario integrations are point-to-point (recipe-book consumes cooking content, workout-plan-generator consumes fitness content). There's **no integration into the brainstorming pipeline** — no path where bookmarks accumulate, get pre-processed by an agent, and surface as fresh idea-seeds in Phase 6/7 of the morning vision walk. That's the integration that compounds. Without it, the hub stays a content-organizer rather than an idea-mining engine.

## Why this matters strategically

The morning vision walk is Vrooli's load-bearing idea-injection mechanism — it's the single interface through which the operator steers the entire project (per the vision-walk skill's stated philosophy: *"the project's biggest bottleneck is not execution — it's idea generation"*). Today, ideas enter the walk only through what the operator volunteers in real-time during the chore-audit and big-picture phases. Bookmarks, which represent a continuous high-volume idea-stream the operator has *already curated implicitly* (they bookmarked it, so it caught their attention), are leaking out entirely. Closing this leak is one of the highest-leverage investments available right now.

## The illustrative palm-reading example

(Captured here as a worked example of the Vrooli-use taxonomy in action, NOT as a backlog item. The example does not become a scenario; it illustrates what kind of bookmark the new flow should handle.)

A user on X publicly shared a free business idea: take advantage of the new ChatGPT-images model to ship a palm-reading app — write the prompt, generate the marketing imagery, ship as iOS app, monetize. Operator bookmarked this post. Under the v0 of the rework:

1. Operator pastes the URL + a sentence noting *"palm reading app idea — direct scenario candidate, deferred until iOS deployment ramp matures."*
2. Manual classification: `Vrooli-use = scenario-candidate`. Classification taxonomy auto-suggests this based on the operator's note + the linked content (via BAS or another extraction path, if available; via operator's note alone if not).
3. The new ideation-extraction agent reads the entry, optionally investigates further (e.g., checks if a similar scenario already exists, what the deployment-ramp dependency looks like, what audience the resulting app would target), and surfaces it as an idea candidate to vision-walk-prep.
4. Vision-walk-prep includes the candidate in tomorrow's Big-picture-context section: *"Bookmark from N days ago: palm-reading-app concept. Operator notes: scenario-candidate, deferred until iOS deployment matures. Agent investigation: no existing scenario; iOS deployment ramp tracked under [initiative]; would target [audience]."*
5. Operator decides during the walk: pursue now / pursue when iOS ramp is ready / drop / refine. If pursue: spawn backlog item directly from the walk.

That's the loop. Scale this to dozens of bookmarks per week and the idea-pipeline shifts from "operator volunteers what they remember" to "operator curates from a pre-processed stream." Same operator effort; orders of magnitude more idea throughput.

## Architectural decisions baked in

These are not up for re-debate during execution — they're settled.

- **Drop Huginn integration entirely.** The existing `Huginn` resource integration is removed in this rework. Future automated bookmark extraction (when capability and audit-safety mature) routes through **Browser Automation Studio (BAS)** per the wrap-not-use principle. BAS is the canonical web-task substrate (per project memory `project_bas_primary_web_substrate.md`); this scenario does not roll its own.
- **Manual paste-first v0.** The CLI command, UI flow, and storage layer prioritize manual entry. Automated platform polling is removed from the v0 critical path. Operator pastes URL + optional description + notes; that's the supported entry method until BAS or formal API integration matures.
- **Dual classification axes.** Content-shape categories (Programming / Recipes / Fitness / etc.) are kept for downstream cross-scenario consumers (recipe-book, workout-plan-generator, etc.). The new Vrooli-use axis (scenario-candidate / operational-reference / marketing-positioning / capability-class-flag / personal-only-filter) is added orthogonally. A single bookmark gets both labels.
- **Categories register from scenarios, OR are managed centrally.** The exact mechanism is the design choice for work-item (b). Both shapes are viable; the decision is downstream. Constraint: scenarios that consume content-shape classifications (recipe-book, workout-plan-generator) must work in either model.
- **Wrap-not-use governs all external-tool integration.** Per project memory `project_wrap_not_use_principle.md`. Any future enhancement that involves calling an external service goes through a Vrooli scenario, not directly.
- **Vision-walk-prep integration is mandatory, not optional.** The whole point of this rework is to feed the morning walk. Work-item (d) is not deferrable.

## The four work items

### (a) BIH v0 redesign — manual-paste-first, dual-axis classification

Strip the Huginn integration and automated-polling defaults. Add manual-paste CLI command (`bookmark-intelligence-hub add --url <link> --notes "..."`) and matching UI flow. Add the Vrooli-use classification axis as a first-class field alongside the existing content-shape categories. Storage schema gets a new `vrooli_use` column (or equivalent). The manual-paste flow is the v0 supported path; automated extraction is removed from the critical path.

**Acceptance:** operator can paste a bookmark via CLI or UI in under 30 seconds; both classification axes are stored; existing content-shape integrations (to recipe-book, workout-plan-generator, etc., where those exist) continue to work.

### (b) Category-registration mechanism

Decide between scenarios self-registering categories with BIH, OR BIH managing categories centrally with scenarios consuming via API. Constraint: existing content-shape consumers (recipe-book, workout-plan-generator) must work in either model. The Vrooli-use taxonomy lives only in BIH; only BIH itself uses it.

**Acceptance:** a new scenario can register a content-shape category in one CLI call (or BIH-side admin call); BIH can route content-shape-categorized bookmarks to the right consumer.

### (c) New ideation-extraction agent

Reads bookmarks classified as `scenario-candidate` / `capability-class-flag` / `marketing-positioning`. For each, optionally investigates (e.g., checks for existing similar scenarios, identifies dependencies, infers audience from operator notes + linked content). Surfaces idea candidates to vision-walk-prep in a structured format. Does NOT make decisions; it's a pre-processor for the operator's walk.

**Acceptance:** running the agent against a corpus of bookmarks produces a structured candidate list; vision-walk-prep can consume the list and incorporate it into tomorrow's Big-picture-context.

### (d) Vision-walk-prep integration

Update the vision-walk-prep agent's HEARTBEAT to read from BIH (via API or CLI), pull surfaced idea candidates since the last walk, and incorporate them into the Big-picture-context section of the prep deliverable. The walk skill consumes them naturally during chore-audit / big-picture-ideation phases.

**Acceptance:** the morning of a walk, the prep deliverable includes a "Bookmark-derived idea candidates since last walk" subsection; the vision-walk skill surfaces these conversationally during Phase 6 or 7 without needing a code change to the walk skill.

## Dependencies and sequencing

- Work-item (a) is the foundation. (b), (c), (d) all assume the v0 redesign exists.
- (b) and (c) can run in parallel once (a) ships. (b) is more discovery-heavy (the registration mechanism design is non-trivial); (c) is more execution-heavy.
- (d) ships last because it depends on (c)'s output format being settled.
- BAS is referenced as a future enhancement substrate but is NOT a blocker for v0. Manual-paste covers v0.
- Workspace sandbox / GCT integration (per project memory `project_sandbox_purpose_accountability.md`) is not relevant to this initiative — these are content-flow integrations, not code-attribution integrations.

## What is OUT of scope for this initiative

- Building any of the ideated apps (palm-reading, etc.).
- Implementing automated platform scraping (X, Reddit, TikTok). When BAS matures, that's a separate enhancement initiative.
- Designing the Vrooli-use classification taxonomy in detail. Work-item (a) captures the requirement; the taxonomy specifics are the first deliverable of (a).
- Migrating existing bookmark data (if any). This rework is treated as a v0 reset; if migration is needed, surface during execution and decide.
- The flipping monetization concept, the homelab-as-bundle marketing positioning, and the prompt-to-product capability-class flag — those are tangentially related (they all came from the same chore-audit conversation) but are captured separately in their own homes (`docs/monetization/REVENUE_LINES.md`, typed marketing-craft observations, `docs/strategy/long-term-capability-flags.md` respectively).

## Pointers

- Vision walk #4 checkpoint: `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/last-handoff.md` — full conversation context including all three divergences.
- Existing scenario: `scenarios/bookmark-intelligence-hub/` — Go API + React UI, PRD at `scenarios/bookmark-intelligence-hub/PRD.md`. The rework keeps ~70% of v0 substrate and strips the Huginn layer.
- BAS scenario (canonical web-task substrate): `scenarios/browser-automation-studio/`.
- Vision-walk-prep agent: `scenarios/prompt-manager/store/teams/director-swarm/members/vision-walk-prep/HEARTBEAT.md` — needs the new integration step (work-item d).
- Wrap-not-use principle: project memory `project_wrap_not_use_principle.md`.
- BAS as primary substrate: project memory `project_bas_primary_web_substrate.md`.

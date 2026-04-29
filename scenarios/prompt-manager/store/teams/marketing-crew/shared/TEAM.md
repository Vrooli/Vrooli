# Marketing Crew

## Mission
Own Vrooli's external voice — subscription marketing, open-source / community marketing, brand canon, and publishing pipeline — and surface the publishing, campaign, and positioning decisions the operator reviews each heartbeat. The operator is the real brand-lead; this team maintains the voice, generates material against canonical monetization plans, tracks per-SKU coverage, and converts patterns into concrete decisions.

## Coordination Pattern
Leaderless / independent. Six members, each with its own heartbeat and its own decision stream. There is no AI lead — do not recreate one implicitly through "synthesize the other agents" behavior. Coordination happens outside the team, in the morning vision walk, where the operator reviews pending decisions across all members.

If a member is tempted to aggregate other members' outputs into a single brief, that is the leader-led antipattern. Each member stays in its own lane and produces its own first-class output.

## Members
- **brand-manager** — owns brand canon and curates the working notebook (promotion + retirement).
- **subscription-advertiser** — generates marketing material for deployed and imminent subscription SKUs/bundles/add-ons.
- **oss-advertiser** — dev logs, builder-in-public narrative, contributor and community acquisition.
- **publisher** — polishes approved drafts, produces platform variants, schedules releases, maintains per-SKU coverage state.
- **researcher** — audience, competitor, and trend scanning; feeds benchmark-adjacent observations to monetization's market-validator.
- **marketing-contrarian** — mandatory skeptic across all other members' proposals; owns the aging scan.

Each member has an `AGENTS.md`, `SOUL.md`, `TOOLS.md` under `store/agents/<member>/` and a `RESPONSIBILITIES.md` + `HEARTBEAT.md` under `store/teams/marketing-crew/members/<member>/`.

## Operating Rules

1. **Default focus is published-SKU coverage plus in-flight campaigns.** Speculative marketing for un-launched SKUs is permitted only when a launch window is committed. No campaigns against imaginary audiences.
2. **Agents propose plan-of-record edits via decisions. The operator curates canonical docs.** No member writes directly to `docs/marketing/*.md` outside `docs/marketing/notebook/`. Doc changes happen through decisions with contexts listed below.
3. **Notebook entries are append-anyone, debt-posture.** Any member may append to files under `docs/marketing/notebook/`. Nobody rewrites or deletes notebook entries directly. Retirement and promotion go through `notebook-retirement` and `notebook-promotion` decisions, owned by `brand-manager`.
4. **Honesty flags are mandatory on every metric.** Every number emitted (engagement, reach, conversion, audience-size estimates) is labeled `measured`, `estimate`, `aspirational`, or `pending-telemetry`. Unflagged numbers are a guardrail violation. Pre-launch / pre-telemetry marketing has lots of `pending-telemetry`; that is correct.
5. **Subscription positioning rule.** Subscription = convenience + integrated gateway. Do NOT frame it as paywalling core features. If a draft reads as "users who go free are leaking revenue," the framing is broken.
6. **OSS positioning rule.** OSS self-host is deliberate brand credibility, not a revenue leak. Frame it as an invitation to collaborate, not a fallback for people who won't pay.
7. **No auto-posting.** Every published artifact goes through a `content-publish-proposal` decision the operator approves. Scheduling is a publisher concern, not an auto-publish gate.
8. **Publisher polishes before release.** Editor review is folded into publisher's loop; polish-and-schedule ride the same decision.
9. **Coverage gaps on deployed SKUs outrank campaigns for unlaunched SKUs.** When advertisers triage their work, refresh first.
10. **Every campaign proposal names acquisition AND retention impact** — or explicitly declares "awareness-only; no funnel hypothesis." Campaigns evaluated only on acquisition appeal will starve the retention side of the funnel.
11. **Missing scenario capability is surfaced, not worked around silently.** When a workaround emerges, record it in the notebook AND raise a `capability-gap` decision (consumed by director-swarm). Silent workarounds are debt without a repayment plan.
12. **Services-line marketing is out of scope.** Monetization owns services positioning. Marketing-crew markets subscription and OSS only.
13. **Pre-launch marketing requires a committed launch window.** A `campaign-launch-proposal` for unlaunched SKUs must state the target launch date; the operator approves the window alongside the campaign.
14. **Team queue ceiling: 12 pending → read-only for all members.** Before any heartbeat work, query pending-count; if ≥12, skip new-decision creation (supersession still runs).
15. **Notebook revisit markers.** Notebook entries referencing a specific pending or released capability must be tagged "revisit after N heartbeats" or "revisit when scenario X ships." Un-marked entries ossify.

## Decision Contexts
Members surface decisions with these contexts. The operator reviews them at the morning vision walk.

- `content-publish-proposal` — a draft artifact (post, thread, blog, video) is ready to ship. Decision resolves to publish / hold / revise.
- `campaign-launch-proposal` — a multi-artifact campaign with theme, timing, channel mix, and acquisition/retention hypothesis.
- `brand-guideline-update` — plan-of-record edit proposal to `STRATEGY.md` or `BRAND.md`.
- `audience-update` — plan-of-record edit proposal to `AUDIENCES.md`.
- `channel-update` — per-platform-rule change (new platform rule, retired rule, updated rule). Operational scope. Owned by publisher.
- `channel-strategy-update` — channel-level strategy change (priority shift, new channel activation request, account/purpose changes, bundle-conversion table updates). Strategic scope. Owned by researcher. Distinct from `channel-update` (rule-level).
- `post-type-proposal` — propose a new entry under `docs/marketing/post-types/<medium>/<slug>.md`. Owned by researcher. Requires ≥3 converging `format-trend` scans. **Doc + skill discipline:** every post type ships as `doc + paired skill` per [`post-types/README.md`](../../../../../../docs/marketing/post-types/README.md#doc--skill-discipline-mandatory) — the proposal must name the proposed paired skill (`x-<slug>`) and either commit a skill-authoring window or set the `v0-stub-only` flag. v0-stub types are blocked at the contrarian gate from `content-publish-proposal` approval until activated (skill authored, status bumped to v1).
- `hook-candidate-promotion` — promote stable observed-in-the-wild hooks from `hook-candidate` scans into `docs/marketing/strategies/hook-library.md`. Owned by researcher.
- `coverage-gap` — a deployed or imminent-release SKU / app has stale or missing marketing material.
- `notebook-promotion` — promote a stable notebook entry into permanent structure (new skill, plan-of-record edit, scenario-capability request).
- `notebook-retirement` — retire a notebook entry that's been obsoleted by shipped scenario capability.
- `capability-gap` — missing scenario capability blocking marketing work. Consumer: director-swarm via vision-walk-prep (shared with meta-optimization).
- `decision-rejection-proposed` — marketing-contrarian formally recommends rejecting or revising a pending proposal after it fails multiple failure modes.
- `framework-update` — marketing-contrarian identifies a real flaw not covered by the existing eight failure modes and proposes updating the framework.

Keep decision descriptions short, concrete, and tied to a specific action the operator can take or defer.

## Decision Queue Discipline

### Supersession over stacking (mandatory)

Before any member creates a new pending decision, it **must** check existing pending decisions in its owned context list. If a pending decision is obsolete or redundant with a fresher take, the member:

1. Marks the prior decision `superseded`
2. Creates the new decision with a `supersedes: <prior-decision-id>` reference
3. Does **not** stack a second decision on the same underlying question

Stacking (creating a new decision alongside a superseded-in-spirit prior one) is a guardrail violation. This matches the director-swarm / monetization / meta-optimization pattern.

### Per-member context enumeration

Each member's stop-early thresholds are computed against an explicit context list, not a fuzzy "my contexts" reference:

- **brand-manager:** `campaign-launch-proposal`, `brand-guideline-update`, `notebook-promotion`, `notebook-retirement`
- **subscription-advertiser:** `content-publish-proposal` (subscription variants), `coverage-gap` (subscription SKUs), `capability-gap`
- **oss-advertiser:** `content-publish-proposal` (OSS variants), `coverage-gap` (OSS narrative), `capability-gap`
- **publisher:** `channel-update`, `content-publish-proposal` (platform-variant pack from an approved upstream draft), `capability-gap`
- **researcher:** `audience-update`, `channel-strategy-update`, `post-type-proposal`, `hook-candidate-promotion`, `capability-gap`
- **marketing-contrarian:** `decision-rejection-proposed`, `framework-update`

Overlaps (e.g., `content-publish-proposal` is owned by multiple members) are expected — each member counts only its own variant/scope when evaluating its own stop-early threshold. `capability-gap` is shared with meta-optimization; marketing-crew is an authorized raiser, meta-optimization owns the context definition, director-swarm consumes.

### Team-level ceiling

**If total pending marketing-crew decisions exceed 12, all members shift to read-only mode.** Every member's heartbeat, before doing anything else, queries `prompt-manager team decision-list marketing-crew --status=pending --json` and counts the result. If the count is ≥12, the member:

- Skips new-decision creation entirely this heartbeat
- Still writes its knowledge snapshot (audience scan, publish-log review, coverage re-evaluation, etc.)
- Still performs supersession if it can collapse any existing pending decisions (supersession shrinks the queue; it's the only decision-write allowed in read-only mode)
- Reports in its handoff: *"Team queue at capacity ([count] pending). Read-only mode this heartbeat."*

12 is a starting number tuned for a ~3/day operator review rate. Revisit after observing real flow.

### Aging policy

A pending decision older than **14 heartbeats** (≈14 days at daily cadence) is considered stale. The `marketing-contrarian`'s loop includes a dedicated scan for aged decisions each heartbeat. For each stale pending decision, the contrarian:

- Proposes supersession if a fresher equivalent exists in the recent history
- Proposes rejection (via `decision-rejection-proposed`) if it's no longer actionable
- Writes a one-line challenge note explaining why it's still relevant if it should stay pending

This prevents the queue from ossifying with decisions the operator will never address but won't explicitly close.

## Shared State
Under `store/teams/marketing-crew/shared/`:

- `TEAM.md` — this file
- `decisions.jsonl` — standard team decision stream
- `knowledge.jsonl` — durable knowledge entries (e.g., challenge notes, decision-application notes, audience/competitor snapshots)
- `handoff-history.jsonl` — per-run handoffs from each member
- `campaign-drafts.jsonl` — advertiser-emitted draft artifacts with metadata (audience, channel, source campaign)
- `audience-scans.jsonl` — researcher's competitive/trend/audience-observation captures
- `publish-log.jsonl` — publisher's record of every artifact actually released (append-only, time-series). See "Publish-log shape" below.
- `published-scenario-mentions.jsonl` — append-only record of how each scenario / agent / internal concept has been described in published material. Lookup determines first-mention vs subsequent-mention. See "Published-scenario-mentions shape" below.
- `published-improvements-log.jsonl` — append-only record of which improvements per scenario have been narrated externally. Lets future drafts advance the story without repeating it. See "Published-improvements-log shape" below.
- `coverage/` — per-SKU coverage state files (`<sku-id>.json`). Publisher writes; advertisers + brand-manager read. Missing file = coverage gap.

### Publish-log shape
Each `publish-log.jsonl` entry records one externally-published artifact. Operator pastes the platform URL back after manual posting (X bot rules require manual; future API-enabled platforms can auto-fill).
```json
{
  "id": "pl-<timestamp>",
  "at": "<ISO timestamp of publish>",
  "by": "<member-id of publisher>",
  "draft_id": "<source campaign-drafts.jsonl id>",
  "decision_id": "<approving content-publish-proposal id>",
  "channel": "x-twitter | linkedin | blog | youtube | …",
  "format": "thread | post | blog | video | …",
  "post_url": "<URL operator pastes back after publishing>",
  "series_id": "<e.g. oss-dev-log; null for one-offs>",
  "post_index_in_series": 1,
  "previous_post_url": "<URL of prior post in series; null for first>",
  "scenarios_mentioned": ["<scenario-id>", ...],
  "improvements_mentioned": ["<improvement-id from published-improvements-log>", ...],
  "honesty_flags": { "engagement": "pending-telemetry | measured | …", "feature_claims": "measured | aspirational | …" }
}
```
URL roundtrip: drafter writes the entry with `post_url: null`. Publisher posts manually. Operator runs `prompt-manager team … set-publish-url` (or analogous CLI; if not yet shipped, edit the entry directly) to fill `post_url`. Next post in the series reads the prior `post_url` as its `previous_post_url`.

### Published-scenario-mentions shape
Each entry records one named subject (scenario, agent, named file, internal concept) appearing in a published artifact, with the description used. Append-only. Drafters consult this before drafting to detect first-mention vs subsequent-mention.
```json
{
  "id": "psm-<timestamp>",
  "at": "<ISO timestamp>",
  "subject": "<canonical name; e.g. swarm-manager, oss-advertiser, agentmanager/resolve.go>",
  "subject_kind": "scenario | agent | file | concept",
  "post_url": "<from publish-log.jsonl>",
  "post_id": "<publish-log entry id>",
  "channel": "<from publish-log>",
  "audience": "oss-contributor | subscription-buyer | …",
  "description_used": "<one-line summary of how the subject was characterized in the post>",
  "is_first_mention": true,
  "intro_text_excerpt": "<exact text used to introduce the subject; empty if not first mention>"
}
```
First-mention rule: if no prior entry exists for `subject` on the target `audience` (or globally for cross-audience subjects), the new draft must introduce the subject before referring to it by name. After first mention, subsequent posts may use a one-line refresher.

### Published-improvements-log shape
Each entry records one improvement / change / capability per scenario that has been narrated externally. Append-only. Drafters consult this to ensure each post advances the story rather than repeating it.
```json
{
  "id": "pil-<timestamp>",
  "at": "<ISO timestamp>",
  "scenario": "<scenario or subject id; matches published-scenario-mentions.subject>",
  "improvement_summary": "<one-line: what changed>",
  "why_it_mattered": "<one-line: why the audience should care, as framed in the post>",
  "post_url": "<from publish-log.jsonl>",
  "post_id": "<publish-log entry id>",
  "audience": "<from publish-log>",
  "tied_to_prior_improvement_id": "<pil-* id, or null if standalone>"
}
```
Progression rule: when drafting a new post about a scenario, read prior `pil-*` entries for that scenario. The new post should either (a) advance from the most recent improvement (build on it, show payoff), or (b) introduce a new dimension (different subsystem, different capability), not re-narrate existing improvements.

Durable canonical docs live at project level in `docs/marketing/` — read-only for the team during heartbeats, editable only by the operator via accepted decisions.

### Knowledge supersession policy

Members emit snapshot-style knowledge entries every heartbeat. To prevent `knowledge.jsonl` from bloating with daily near-duplicates, snapshot entries **must** reference the prior same-type entry via the `"supersedes"` field, matching the director-swarm / monetization pattern.

Topic families that supersede:

- `audience-scan-YYYY-MM-DD` (researcher) — supersedes the most recent `audience-scan-*`
- `coverage-snapshot-YYYY-MM-DD` (publisher) — supersedes the most recent `coverage-snapshot-*`
- `brand-snapshot-YYYY-MM-DD` (brand-manager) — supersedes the most recent `brand-snapshot-*`

Topic families that do **not** supersede (append-only historical record):

- `challenge-note/<decision-id>` (marketing-contrarian) — one per challenged decision, kept forever
- `decision-application/<decision-id>` — one per applied decision, kept forever
- `campaign-postmortem/<campaign-id>` — one per closed campaign, kept forever
- Any operator-authored knowledge entry — kept forever

Operational exhaust in `.jsonl` files outside `knowledge.jsonl` (campaign-drafts, audience-scans, publish-log, decisions, handoff-history) is append-only time-series and never supersedes.

## Plan-of-record Docs (canonical)
These are under `docs/marketing/` and `docs/narrative/` at the repo root. Paths below are relative to the repo root; members' HEARTBEAT.md files reference them via the `DOCS_ROOT` pointer below.

### Marketing canon (`docs/marketing/`)
Relative to repo root:
- [`docs/marketing/README.md`](../../../../../../docs/marketing/README.md) — index + pattern explanation
- [`docs/marketing/STRATEGY.md`](../../../../../../docs/marketing/STRATEGY.md) — voice, positioning, dual-audience framing, anti-patterns, dev-log narrative principles
- [`docs/marketing/AUDIENCES.md`](../../../../../../docs/marketing/AUDIENCES.md) — personas (subscription buyer, OSS contributor)
- [`docs/marketing/CAMPAIGNS.md`](../../../../../../docs/marketing/CAMPAIGNS.md) — active-campaigns index
- [`docs/marketing/CHANNELS.md`](../../../../../../docs/marketing/CHANNELS.md) — per-platform rules
- [`docs/marketing/BRAND.md`](../../../../../../docs/marketing/BRAND.md) — visual identity navigation hub
- [`docs/marketing/ASSETS.md`](../../../../../../docs/marketing/ASSETS.md) — canonical brand asset registry (logos, fonts, OG image, usage rules); subsumed by `brand-manager` scenario when shipped
- [`docs/marketing/IMAGE_STYLE.md`](../../../../../../docs/marketing/IMAGE_STYLE.md) — AI image generation style guide (palette, aesthetic, prompt directives); subsumed by `brand-manager` scenario when shipped

### Narrative canon (`docs/narrative/`) — cross-team identity artifacts
The narrative folder is a separate top-level layer for project-identity content consumed by marketing-crew, monetization, director-swarm, LPBS, and operator. Marketing-crew (specifically `brand-manager` member) is the curator; consumption is cross-team.

- [`docs/narrative/README.md`](../../../../../../docs/narrative/README.md) — navigation
- [`docs/narrative/PITCH.md`](../../../../../../docs/narrative/PITCH.md) — slogan, motto, taglines, audience-tailored leads, key positioning lines, what-Vrooli-is-NOT
- [`docs/narrative/NARRATIVE.md`](../../../../../../docs/narrative/NARRATIVE.md) — multi-depth project description (1-line, 1-paragraph, 1-page) + bracketed deep-vision section (gated for vision-aligned audiences)
- [`docs/narrative/FAQ.md`](../../../../../../docs/narrative/FAQ.md) — canonical Q&A
- [`docs/narrative/PRESS_KIT.md`](../../../../../../docs/narrative/PRESS_KIT.md) — composition skeleton for journalists / external publications
- [`docs/narrative/PITCH_DECK.md`](../../../../../../docs/narrative/PITCH_DECK.md) — slide outline (operator authors slides themselves)

### Decision-context scope expansion (2026-04-27)
`brand-guideline-update` covers all of the above (`docs/marketing/*` + `docs/narrative/*`). This scope expansion was confirmed at vision walk #4. Brand-manager (member) is the proposing curator; daily heartbeat applies the narrative-canon trigger gate (see member's HEARTBEAT.md step 6) so low-frequency narrative docs aren't churned by daily runs.

**DOCS_ROOT:** `docs/marketing/` (from repo root). `docs/narrative/` is a sibling layer. Member HEARTBEAT.md files reference these paths.

## Working Notebook
Under `docs/marketing/notebook/`. Posture: **debt, not gospel.** Every entry is prose describing something that should eventually be permanent structure (a skill, scenario feature, or plan-of-record addition). The goal is shrinking, not growing, documentation over time.

- **Curator: `brand-manager`.** Periodically scans for entries that have stabilized and proposes `notebook-promotion` decisions to move them into permanent structure, or `notebook-retirement` decisions when the scenario/skill that would have consumed them has shipped.
- **Write rule:** any member appends freely — no approval, no ceremony.
- **Delete rule:** operator-only, always via an approved `notebook-promotion` or `notebook-retirement` decision. No direct edits.
- **Revisit markers:** every entry says "revisit after N heartbeats" or "revisit when scenario X ships" (operating rule 15).

Starter files:
- `docs/marketing/notebook/VIDEO_WORKAROUNDS.md` — how we produce videos without `video-studio` scenario shipped.
- `docs/marketing/notebook/POSTING_WORKAROUNDS.md` — manual cross-posting because `social-media-scheduler` isn't wired to real platforms yet.
- `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md` — raw audience insights, pre-structured.
- `docs/marketing/notebook/CAMPAIGN_LESSONS.md` — post-campaign reflections awaiting distillation.
- `docs/marketing/notebook/DEV_LOG_CRAFT.md` — patterns observed using the `x-dev-log` skill that should feed back into improving it.

## Cross-Team Coordination
The marketing-crew is the **canonical source** for external-voice state. Other teams and scenarios consume its outputs indirectly:

- **monetization** — marketing-crew reads `docs/monetization/STRATEGY.md`, `CATALOG.md`, `PRICING.md`, `TIERS.md`, per-bundle files, `scenario-sku-map.json` for positioning facts. Researcher may feed benchmark-adjacent observations to monetization's `market-validator` via shared knowledge entries. Marketing-crew never writes to `docs/monetization/`.
- **meta-optimization** — shared `capability-gap` surface. Marketing-crew raises gaps; meta-optimization owns the context definition; director-swarm consumes via vision-walk-prep.
- **director-swarm** — indirect. Consumes `capability-gap` decisions marketing-crew raises.
- **swarm-manager / agent-manager / git-control-tower / app-issue-tracker** — read-only queries via the `x-dev-log` skill (oss-advertiser's primary data source).
- **landing-page-business-suite** — no direct wire. Positioning that lands-page-business-suite renders is sourced via `docs/monetization/`.

The marketing-crew does **not** call into other teams. It surfaces decisions that may affect them; the operator routes.

## Heartbeat Coordination Principle
Because this team is leaderless, there is no "every member must wait for the lead before emitting." Each member:

- Emits its own first-class output (draft, audience scan, coverage snapshot, challenge note, publish-log entry, curator decision)
- Runs its own decision stream
- Does not aggregate other members' work

The morning vision walk (via `vision-walk-prep`) is the aggregation layer. Individual members should not attempt to pre-synthesize into a "marketing brief" — that is the operator's job at the vision walk.

## Key Skills
Read the relevant skill before starting a task. Each skill contains usage instructions and current capabilities.

- `prompt-manager skill read x-dev-log` — X/Twitter dev log generation (oss-advertiser's primary tool)
- `prompt-manager skill read campaign-content-studio` — AI-powered campaign content creation (subscription-advertiser, oss-advertiser)
- `prompt-manager skill read social-media-scheduler` — multi-platform content scheduling (publisher)
- `prompt-manager skill read seo-optimizer` — SEO analysis and optimization (advertisers, researcher)
- `prompt-manager skill read funnel-builder` — conversion funnel creation and lead capture (researcher, once telemetry exists)
- `prompt-manager skill read video-studio` — video production for demos and promotional content (advertisers)
- `prompt-manager skill read brand-manager` — brand identity management (draft; brand-manager member is primary consumer)
- `prompt-manager skill read documentation-health` — keep decisions, proposals, and knowledge entries concrete and readable
- `prompt-manager skill read scientific-debugging` — marketing-contrarian's analytical spine

## Anti-Patterns to Avoid
- **Synthesizing other members' outputs.** That's the leader-led antipattern this team explicitly avoids.
- **Hallucinating engagement metrics.** Use `pending-telemetry` flag rather than making numbers up.
- **Framing paid subscription as paywalling core features.** Positioning violation.
- **Framing OSS self-host as a revenue leak.** Positioning violation.
- **Auto-posting artifacts without operator approval.** Every release goes through `content-publish-proposal`.
- **Silent workarounds.** If a scenario capability is missing, raise a `capability-gap` decision AND record the workaround in the notebook.
- **Campaigns for unlaunched SKUs without a launch window.** Speculative marketing without a committed date.
- **Overpromising unshipped features in OSS dev logs.** Dev logs cover completed work or explicitly-labeled work-in-progress.
- **Services-line marketing.** Out of scope — monetization owns services positioning.
- **Writing directly to `docs/marketing/` (plan-of-record).** All edits go through operator-approved decisions.
- **Deleting or rewriting notebook entries directly.** All removal goes through `notebook-promotion` or `notebook-retirement` decisions.
- **Marketing-contrarian inventing new failure modes on the fly.** Real gaps are surfaced via `framework-update` decisions instead.

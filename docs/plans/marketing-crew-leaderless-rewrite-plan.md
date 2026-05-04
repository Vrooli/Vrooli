# Marketing Crew — Leaderless Rewrite Implementation Plan

## 1. Purpose

Rewrite the prompt-manager `marketing-crew` team to match the leaderless pattern the monetization and meta-optimization teams already use, and give it both a plan-of-record doc surface (`docs/marketing/`) and a working notebook (`docs/marketing/notebook/`).

When done: each marketing-crew member has its own heartbeat, produces its own first-class output, and surfaces decisions that the operator reviews at the morning vision walk via `vision-walk-prep`. The team can document brand/voice canon in plan-of-record, record tooling workarounds in the notebook, and propose `capability-gap` decisions when a missing scenario (e.g. video-studio) blocks its work.

---

## 2. Required Reading

Execute before starting implementation:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read team-shared-docs-design
prompt-manager skill read team-coordination-independent
prompt-manager skill read documentation-health
```

Reference docs to read before editing (read, don't copy):

- `scenarios/prompt-manager/store/teams/monetization/shared/TEAM.md` — closest structural precedent (leaderless, plan-of-record docs, contrarian).
- `scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM.md` — closest notebook precedent (working notebook + debt-curator).
- `docs/meta-optimization/README.md` — canonical working-notebook README shape.
- `docs/monetization/README.md` — canonical plan-of-record README shape.
- `scenarios/prompt-manager/store/agents/meta-contrarian/{AGENTS,SOUL,TOOLS}.md` — closest precedent for domain-scoped contrarian (we will create `marketing-contrarian` in the same shape).
- `scenarios/prompt-manager/store/teams/monetization/members/financial-tracker/HEARTBEAT.md` — richest heartbeat example (217 lines); our marketing members should land in a similar density range (100-200 lines each).
- `scenarios/prompt-manager/store/agents/vision-walk-prep/AGENTS.md` (phases 2-5) — shape of the integration phase we will add.

---

## 3. Problem Statement

The current `marketing-crew` team predates the leaderless pattern. It is in a structurally different generation than the other four teams.

**Evidence (current state, 2026-04-21):**

- `teams/marketing-crew/team.json` uses `"coordination.pattern": "leader-led"`, `"leadAgentId": "marketing-lead"`, `"runtime.mode": "single-process"`, `"decisionMode": "yolo"`. Monetization and meta-optimization both use `"independent"` / `"multi-process"` / `"approval"`.
- `org.json` has hierarchical edges (lead → 3 reports). Updated teams have `"edges": []`.
- `shared/` contains only `TEAM.md`. Monetization's `shared/` contains 8 files (decisions, knowledge, handoff, ledger, scans, etc.).
- Per-member `HEARTBEAT.md` files are 2-3 bullets. Monetization members have 100-200 line heartbeats with explicit decision contexts, stop-early thresholds, supersession rules, and output schemas.
- `shared/TEAM.md` has no decision contexts, no queue discipline, no anti-patterns, no cross-team coordination (beyond a read-CATALOG.md line).
- `vision-walk-prep` does not gather marketing decisions (phases 2/4/5 cover director-swarm, monetization, meta-opt).
- Most marketing tooling referenced by the team (video generation, multi-platform auto-posting, brand-manager CLI, social-media-scheduler wiring) is either not shipped or draft. The team needs a working notebook to record workarounds while those scenarios catch up.

**What the user asked for:**

1. Flip to leaderless; operator is effectively team-lead via vision-walk.
2. Members output decisions (not approval-gated drafts).
3. Plan-of-record for brand/voice/audience canon; working notebook for tooling workarounds.
4. 6 members: brand-manager, subscription-advertiser, oss-advertiser, publisher, researcher, marketing-contrarian.
5. brand-manager doubles as the working-notebook curator.
6. Per-SKU coverage state so the team tracks which deployed/imminent apps have fresh marketing material.
7. Missing-tooling decisions route through the existing `capability-gap` context (meta-optimization owns the context; director-swarm consumes).

---

## 4. Hard Rules (Greenfield Constraint)

These rules apply to every phase. Re-stated in Definition of Done.

1. **Greenfield.** Delete the current 4 members (`marketing-lead`, `content-creator`, `content-editor`, `market-researcher`) completely — both from `teams/marketing-crew/members/` and from `store/agents/` if not referenced by any other team. No compat shims, no renamed-role aliases, no `marketing-lead-legacy/`.
2. **No shared-contrarian reuse.** Marketing gets its own `marketing-contrarian` agent (mirroring the `contrarian` / `meta-contrarian` split). Do not point `marketing-crew` at the existing generic `contrarian` agent.
3. **Do not restart prompt-manager.** The operator is running this scenario; Claude only writes files. Operator restarts prompt-manager manually to pick up team-config changes.
4. **No direct writes to plan-of-record from agents.** Plan-of-record docs (`docs/marketing/*.md`, excluding `docs/marketing/notebook/`) are operator-curated via approved decisions, same rule as `docs/monetization/`.
5. **Notebook append-only from agents; operator-only deletes.** Same rule meta-opt uses for `docs/meta-optimization/`.
6. **Pre-existing issues in the touched surface must be fixed in this plan.** If a stale reference, dead link, or obsolete agent file in adjacent surfaces (e.g. `vision-walk-prep`) is noticed while doing the required edits, fix it in the same pass rather than deferring.

---

## 5. Scope

### In scope

- Rewrite `scenarios/prompt-manager/store/teams/marketing-crew/` fully (`team.json`, `roles.json`, `org.json`, `shared/TEAM.md`, delete old members, create 6 new members).
- Create 6 new agent definitions under `scenarios/prompt-manager/store/agents/` (`brand-manager`, `subscription-advertiser`, `oss-advertiser`, `publisher`, `researcher`, `marketing-contrarian`).
- Delete obsolete agent definitions (`marketing-lead`, `content-creator`, `content-editor`, `market-researcher`) **only if** they are not referenced by any other team. (Verification step in Phase 0.)
- Scaffold `docs/marketing/` plan-of-record tree with starter skeletons.
- Scaffold `docs/marketing/notebook/` working notebook with starter debt files and README.
- Wire `marketing-crew` into `vision-walk-prep` (new phase).
- Update meta-optimization's `capability-gap` context ownership list to include marketing-crew members as authorized raisers.
- Update cross-team readers that reference the old marketing member roles (primarily the monetization TEAM.md cross-team coordination section, which names `marketing-crew` generically — this likely needs no change, verify).
- Update `scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md`'s outbound-reads list (CATALOG.md and STRATEGY.md references) to match current docs/monetization layout.

### Out of scope

- Shipping any marketing-related scenario (video-studio, social-media-scheduler wiring, brand-manager scenario implementation). These remain `capability-gap` items the team raises after the rewrite.
- Enabling any of the 6 new members' heartbeats (`enabled: true`). All members ship with `enabled: false`; operator flips them on one at a time and observes.
- Migrating real existing content (drafts, past posts) into the new structure. The team starts empty.
- Changing any other team's coordination pattern or member list.
- Touching landing-page-business-suite, brand-manager scenario code, or any other scenario.
- Rewriting the x-dev-log, campaign-content-studio, seo-optimizer, social-media-scheduler, funnel-builder, video-studio, or brand-manager skills. (They are referenced from the new TOOLS.md files, not modified.)

---

## 6. Current Technical Context

### Files to delete (greenfield)

```
scenarios/prompt-manager/store/teams/marketing-crew/members/marketing-lead/
scenarios/prompt-manager/store/teams/marketing-crew/members/content-creator/
scenarios/prompt-manager/store/teams/marketing-crew/members/content-editor/
scenarios/prompt-manager/store/teams/marketing-crew/members/market-researcher/
```

Agent definitions to delete (verify un-referenced first — see Phase 0):

```
scenarios/prompt-manager/store/agents/marketing-lead/
scenarios/prompt-manager/store/agents/content-creator/
scenarios/prompt-manager/store/agents/content-editor/
scenarios/prompt-manager/store/agents/market-researcher/
```

### Files to rewrite

```
scenarios/prompt-manager/store/teams/marketing-crew/team.json
scenarios/prompt-manager/store/teams/marketing-crew/org.json
scenarios/prompt-manager/store/teams/marketing-crew/roles.json
scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md
```

### Files to create

**Team member dirs** (6):
```
scenarios/prompt-manager/store/teams/marketing-crew/members/brand-manager/{HEARTBEAT.md,RESPONSIBILITIES.md,heartbeat.json,logs/.gitkeep}
scenarios/prompt-manager/store/teams/marketing-crew/members/subscription-advertiser/{...}
scenarios/prompt-manager/store/teams/marketing-crew/members/oss-advertiser/{...}
scenarios/prompt-manager/store/teams/marketing-crew/members/publisher/{...}
scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/{...}
scenarios/prompt-manager/store/teams/marketing-crew/members/marketing-contrarian/{...}
```

**Agent definitions** (6):
```
scenarios/prompt-manager/store/agents/brand-manager/{AGENTS.md,SOUL.md,TOOLS.md,agent.json}
scenarios/prompt-manager/store/agents/subscription-advertiser/{...}
scenarios/prompt-manager/store/agents/oss-advertiser/{...}
scenarios/prompt-manager/store/agents/publisher/{...}
scenarios/prompt-manager/store/agents/researcher/{...}
scenarios/prompt-manager/store/agents/marketing-contrarian/{...}
```

> **Agent-ID collision check.** A `brand-manager` scenario already exists under `scenarios/brand-manager/`, and a `brand-manager` skill exists. Agent IDs are namespaced separately (under `store/agents/`), so an agent named `brand-manager` does not collide with either. Verify in Phase 0 by grepping for `brand-manager` string references to ensure no hard-coded lookup assumes agent-vs-scenario. Similarly for `researcher` (common word — check no other agent uses it).

**Team shared state**:
```
scenarios/prompt-manager/store/teams/marketing-crew/shared/decisions.jsonl           (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/knowledge.jsonl           (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/handoff-history.jsonl     (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/campaign-drafts.jsonl     (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/audience-scans.jsonl      (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/publish-log.jsonl         (empty)
scenarios/prompt-manager/store/teams/marketing-crew/shared/coverage/.gitkeep         (dir marker)
```

**Plan-of-record** (`docs/marketing/`):
```
docs/marketing/README.md
docs/marketing/STRATEGY.md
docs/marketing/AUDIENCES.md
docs/marketing/CAMPAIGNS.md
docs/marketing/CHANNELS.md
docs/marketing/BRAND.md
```

**Working notebook** (`docs/marketing/notebook/`):
```
docs/marketing/notebook/README.md
docs/marketing/notebook/VIDEO_WORKAROUNDS.md
docs/marketing/notebook/POSTING_WORKAROUNDS.md
docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md
docs/marketing/notebook/CAMPAIGN_LESSONS.md
docs/marketing/notebook/DEV_LOG_CRAFT.md
```

### Files to update (cross-surface)

```
scenarios/prompt-manager/store/agents/vision-walk-prep/AGENTS.md      (add marketing phase)
scenarios/prompt-manager/store/agents/vision-walk-prep/TOOLS.md       (add commands)
scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/SKILL.md   (add marketing block)
scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM.md  (add marketing-crew to capability-gap raisers list)
```

---

## 7. Target End State

### Team-level shape

`team.json`:
```json
{
  "kind": "team",
  "schemaVersion": 1,
  "id": "marketing-crew",
  "displayName": "Marketing Crew",
  "mission": "Own Vrooli's external voice — subscription marketing, open-source/community marketing, brand canon, and publishing pipeline — and surface the publishing, campaign, and positioning decisions the operator reviews each heartbeat.",
  "enabled": false,
  "runtime": { "mode": "multi-process" },
  "coordination": {
    "pattern": "independent",
    "reportingMode": "none",
    "messagingMode": "disabled",
    "capabilities": {
      "showOrgContext": false,
      "injectInbox": false,
      "allowPeerTriggers": false,
      "showTaskBoardGuidance": false,
      "showDecisionLogGuidance": true,
      "showKnowledgeLogGuidance": true,
      "requireHandoff": true
    }
  },
  "execution": { "queuePolicy": "bounded-parallel", "maxConcurrentRuns": 3 },
  "decisionMode": "approval",
  "shared": { "path": "shared/", "mountHint": "readWrite" },
  "revision": 4,
  "createdAt": "<preserve>",
  "updatedAt": "<today>"
}
```

`org.json`: `"edges": []`.

`roles.json`: 6 entries, one per member, one-sentence `description` each.

### Member list (6)

| Member | Decision contexts owned |
|---|---|
| `brand-manager` | `campaign-launch-proposal`, `brand-guideline-update`, `notebook-promotion`, `notebook-retirement` |
| `subscription-advertiser` | `content-publish-proposal` (subscription variant), `coverage-gap` (subscription SKUs), `capability-gap` |
| `oss-advertiser` | `content-publish-proposal` (OSS variant), `coverage-gap` (OSS narrative), `capability-gap` |
| `publisher` | `channel-update`, `content-publish-proposal` (platform-variant pack), `capability-gap` |
| `researcher` | `audience-update`, `capability-gap` |
| `marketing-contrarian` | `decision-rejection-proposed`, `framework-update` |

### Decision contexts (10 total, overlap expected)

1. `content-publish-proposal` — a draft artifact (post, thread, blog, video) is ready to ship. Decision resolves to publish / hold / revise.
2. `campaign-launch-proposal` — a multi-artifact campaign with a theme, timing, and channel mix.
3. `brand-guideline-update` — plan-of-record edit to `STRATEGY.md` or `BRAND.md`.
4. `audience-update` — plan-of-record edit to `AUDIENCES.md`.
5. `channel-update` — platform or per-platform-rule change (adding LinkedIn, retiring Mastodon, changing thread-length rules).
6. `coverage-gap` — a deployed or imminent-release SKU / app has stale or missing marketing material.
7. `notebook-promotion` — promote a stable notebook entry into permanent structure (new skill proposal, plan-of-record doc edit, scenario-capability request).
8. `notebook-retirement` — retire a notebook entry that's been obsoleted by shipped scenario capability.
9. `capability-gap` — missing scenario capability blocking marketing work (reuses meta-optimization's context; consumed by director-swarm via vision-walk-prep).
10. `decision-rejection-proposed` / `framework-update` — marketing-contrarian's two contexts.

### Queue discipline (copy from monetization / meta-opt)

- Team-level ceiling: **12 pending → read-only mode** for all members.
- Supersession over stacking: mandatory before new-decision creation.
- Per-member context enumeration (listed above — used for own stop-early threshold).
- Aging policy: **14 heartbeats** → marketing-contrarian scans for stale decisions and supersedes / proposes rejection / adds "still relevant" note.
- Knowledge supersession: snapshot-style entries supersede; append-only histories (challenge notes, campaign lessons) do not.

### Coverage state shape

`teams/marketing-crew/shared/coverage/<sku-id>.json`:

```json
{
  "sku_id": "business-bundle",
  "display_name": "Business Bundle",
  "last_touched": "2026-04-21",
  "status": "fresh | stale | missing",
  "channels": {
    "x-twitter": { "last_posted": "2026-04-18", "artifact_ref": "publish-log.jsonl#L42" },
    "blog": { "last_posted": null, "artifact_ref": null },
    "video": { "last_posted": null, "artifact_ref": null }
  },
  "next_review_date": "2026-05-05",
  "notes": "Post-launch marketing fresh; next refresh after v2 features ship."
}
```

SKU IDs correspond to `docs/monetization/scenario-sku-map.json` entries. Publisher writes; advertisers read to decide refresh targets. Missing file = no coverage = gap.

### Plan-of-record starter structure

- `docs/marketing/README.md` — index; names the two doc patterns; points at TEAM.md for the live rules.
- `docs/marketing/STRATEGY.md` — voice, positioning, dual-audience framing (subscription as convenience+gateway; OSS as deliberate credibility — **not** paywalling core), anti-patterns (overpromising, hype drift, hallucinated metrics, treating free as leak).
- `docs/marketing/AUDIENCES.md` — starter personas: subscription buyers (indie devs / solopreneurs / small teams), OSS contributors, cloud-hosted evaluators (placeholder; revisit when split).
- `docs/marketing/CAMPAIGNS.md` — index of active campaigns (empty at start; seeded by `campaign-launch-proposal` decisions).
- `docs/marketing/CHANNELS.md` — per-platform rules (X, blog, video placeholders with rules copied/refined from current marketing-crew TEAM.md).
- `docs/marketing/BRAND.md` — thin placeholder until `brand-manager` scenario ships; points at it.

### Working notebook starter structure

- `docs/marketing/notebook/README.md` — mirrors `docs/meta-optimization/README.md`: debt-posture explanation, curator identification (brand-manager), three-tier mental model, revisit-marker convention.
- `docs/marketing/notebook/VIDEO_WORKAROUNDS.md` — how we produce videos without `video-studio` scenario shipped.
- `docs/marketing/notebook/POSTING_WORKAROUNDS.md` — manual cross-posting because `social-media-scheduler` isn't wired.
- `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md` — raw audience insights, pre-structured.
- `docs/marketing/notebook/CAMPAIGN_LESSONS.md` — post-campaign reflections.
- `docs/marketing/notebook/DEV_LOG_CRAFT.md` — patterns observed while using the `x-dev-log` skill that should feed back into improving it.

All notebook files ship with an intentionally thin starter entry (a sentence or two stating the debt it's meant to capture) so the shape is visible but real content starts empty.

### Vision-walk-prep integration

Add phase 4.6 to `vision-walk-prep/AGENTS.md` (between monetization and meta-optimization):

> **4.6 Gather pending marketing decisions** — Query pending decisions across marketing-crew contexts: `content-publish-proposal`, `campaign-launch-proposal`, `brand-guideline-update`, `audience-update`, `channel-update`, `coverage-gap`, `notebook-promotion`, `notebook-retirement`. Fetch `challenge-report/<decision-id>` knowledge entries and attach inline to their target decisions. Select top 3 with context diversity (not 3 from one bucket). For each decision, record the proposing member. If the marketing-crew team has `"enabled": false` in `teams/marketing-crew/team.json`, note that and skip. `capability-gap` decisions raised by marketing-crew members are grouped with existing capability-gap items in phase 2 (they have the same consumer — director-swarm).

### Meta-optimization handoff for `capability-gap`

Add to `teams/meta-optimization/shared/TEAM.md` under the `capability-gap` context description:

> Authorized raisers: `run-introspector`, `toolchain-validator`, and marketing-crew members (`brand-manager`, `subscription-advertiser`, `oss-advertiser`, `publisher`, `researcher`). Marketing-crew raises it when a missing scenario capability blocks publishing or campaign work (e.g. video-studio scenario not shipped, social-media-scheduler not wired to real platforms). Consumer remains director-swarm.

---

## 8. Implementation Strategy (Phased)

Phases are ordered by safety — low-risk scaffolding first, cross-surface edits last. Each phase produces a committable unit.

### Phase 0 — Baseline verification and collision checks

Before any writes:

1. Confirm the 4 old member-agent definitions (`marketing-lead`, `content-creator`, `content-editor`, `market-researcher`) are **not referenced by any other team**:
   ```bash
   grep -rn "marketing-lead\|content-creator\|content-editor\|market-researcher" \
     scenarios/prompt-manager/store/teams/ scenarios/prompt-manager/store/agents/ \
     --include="*.json" --include="*.md" \
     | grep -v "/marketing-crew/" | grep -v "/agents/marketing-lead/" \
     | grep -v "/agents/content-creator/" | grep -v "/agents/content-editor/" \
     | grep -v "/agents/market-researcher/"
   ```
   Expected: zero matches. If matches appear, update those references as part of this plan (do not defer).

2. Confirm proposed new agent IDs do not collide with existing agents:
   ```bash
   for id in brand-manager subscription-advertiser oss-advertiser publisher researcher marketing-contrarian; do
     test -e "scenarios/prompt-manager/store/agents/$id" && echo "COLLISION: $id"
   done
   ```
   Expected: zero output. If any collide, document the collision and resolve (likely rename; consult operator).

3. Capture the current marketing-crew shape as a baseline snapshot for rollback reference:
   ```bash
   git status scenarios/prompt-manager/store/teams/marketing-crew/ docs/marketing/ 2>/dev/null
   ```
   No action needed beyond logging — the operator's git history is the rollback mechanism.

4. Capture current heartbeat-list output for the team to confirm it's clean before changes:
   ```bash
   prompt-manager team heartbeat-list marketing-crew
   ```

**Exit criterion:** no unexpected cross-references; no collisions; baseline logged.

### Phase 1 — Team config files

Rewrite these four files. Greenfield — full overwrite, no merge:

1. `teams/marketing-crew/team.json` — target shape in §7.
2. `teams/marketing-crew/org.json` — `{ "kind": "org-chart", "schemaVersion": 1, "teamId": "marketing-crew", "edges": [] }`.
3. `teams/marketing-crew/roles.json` — 6 role entries, descriptions matching the member-list table in §7.
4. `teams/marketing-crew/shared/TEAM.md` — new, structured like monetization's TEAM.md. Sections:
   - Mission
   - Coordination Pattern (leaderless / independent, explicit "no AI lead" note)
   - Members (one-line summaries)
   - Operating Rules (10-15 numbered rules — see §8.1 below)
   - Decision Contexts (10 contexts from §7 with short descriptions)
   - Decision Queue Discipline (supersession + per-member enumeration + 12-ceiling + 14-heartbeat aging — copy monetization's)
   - Shared State (file inventory)
   - Plan-of-record Docs (pointers to `docs/marketing/`)
   - Working Notebook (pointer to `docs/marketing/notebook/`; names brand-manager as curator; states append-anyone / no-direct-delete rule)
   - Cross-Team Coordination
   - Heartbeat Coordination Principle
   - Key Skills
   - Anti-Patterns to Avoid

**Exit criterion:** `prompt-manager team show marketing-crew` loads without error; no members present yet (they come in Phase 2).

#### 8.1 — Operating Rules to encode in TEAM.md

Minimum rule set (will refine wording during write):

1. Default focus is published-SKU coverage plus in-flight campaigns. Do not speculate on audiences for un-launched SKUs.
2. Agents propose plan-of-record edits via decisions. Operator curates canonical docs. No direct writes to `docs/marketing/` outside `notebook/`.
3. Notebook entries are append-anyone, debt-posture. Brand-manager curates retirement via `notebook-promotion` / `notebook-retirement` decisions.
4. Every metric emitted is labeled `measured`, `estimate`, `aspirational`, or `pending-telemetry`. Unflagged numbers are a guardrail violation. (Same rule as monetization; marketing has equivalent hallucination risk around engagement stats.)
5. Subscription marketing does **not** frame paid tier as paywalling core features. OSS self-host is brand credibility, not a leak. If copy reads as "users on free are leaking revenue," framing is broken.
6. OSS marketing does **not** overpromise unshipped features. Dev logs report on completed work or explicitly-labeled work-in-progress.
7. No auto-posting. Every published artifact goes through a `content-publish-proposal` decision the operator approves.
8. Publisher never publishes without editor-equivalent review (the polish step is folded into publisher's loop — both the polish and the schedule live in the same decision).
9. Coverage gaps on deployed SKUs are always higher priority than campaigns for unlaunched SKUs.
10. Every campaign proposal names both a subscription-acquisition and retention-impact hypothesis, OR explicitly declares "awareness-only; no funnel hypothesis." (Mirrors monetization's acquisition-AND-retention rule.)
11. Missing scenario capability blocking work is raised via `capability-gap`, not worked around silently. Workarounds also go in the notebook so they eventually promote into the gap resolution.
12. Services-line marketing is not in scope for this team. Monetization owns services positioning.
13. Pre-launch marketing requires operator approval of the launch date itself (decision context: `campaign-launch-proposal` with an explicit launch-window field).
14. Team queue ceiling: 12 pending → read-only mode for all members.
15. Notebook entries referencing a specific pending or released capability must be revisit-marked ("revisit after N heartbeats" or "revisit when scenario X ships").

### Phase 2 — New agent definitions

Create 6 agent directories under `store/agents/`. Each directory contains `agent.json`, `AGENTS.md`, `SOUL.md`, `TOOLS.md`.

**agent.json shape** (model this on existing agents):
```json
{
  "kind": "agent",
  "schemaVersion": 1,
  "id": "<agent-id>",
  "displayName": "<human name>",
  "revision": 1,
  "createdAt": "<today>",
  "updatedAt": "<today>"
}
```

Density target: AGENTS.md = 30-60 lines; SOUL.md = 20-40 lines; TOOLS.md = 30-60 lines. Mirror the shape visible in `store/agents/meta-contrarian/` (our closest precedent).

**Per-agent content sketch:**

#### 2a. `brand-manager` (team agent, not scenario)

- **SOUL.md:** owner of brand canon + dual-audience framing; curator of the working notebook (promotes/retires entries); strategic long-horizon voice (not per-post).
- **AGENTS.md workflow:** start of session → `prompt-manager team member-context marketing-crew brand-manager` → read last handoff → read `docs/marketing/STRATEGY.md` → scan notebook for ≥N-heartbeat-old entries → propose `notebook-promotion` or `notebook-retirement` decisions → scan pending `content-publish-proposal` decisions for voice/framing drift and challenge if found → propose `campaign-launch-proposal` / `brand-guideline-update` when warranted.
- **TOOLS.md:** `prompt-manager team decision-*`, `prompt-manager team knowledge-*`, filesystem reads on `docs/marketing/` and `docs/marketing/notebook/`. Skill references: `brand-manager` (draft — documents planned scenario), `campaign-content-studio`, `documentation-health`.

#### 2b. `subscription-advertiser`

- **SOUL.md:** generates marketing material for deployed and imminent SKUs/bundles/add-ons. Lives under the discipline that subscription = convenience+gateway, not paywall.
- **AGENTS.md workflow:** read coverage files under `shared/coverage/*.json` → identify stale/missing SKUs → read monetization `CATALOG.md` + per-bundle docs for facts → draft publish-proposal decisions → raise `coverage-gap` for stale SKUs that aren't being addressed → raise `capability-gap` when drafting needs a missing tool.
- **TOOLS.md:** `campaign-content-studio`, `seo-optimizer`, `x-dev-log` (for feature-shipped posts), `video-studio` (draft), `social-media-scheduler` (for scheduling hand-off to publisher).

#### 2c. `oss-advertiser`

- **SOUL.md:** builder-in-public voice; owns dev logs; contributor and community acquisition narrative. OSS self-host is brand credibility, framed as an invitation.
- **AGENTS.md workflow:** read last N heartbeats' worth of activity via `x-dev-log` skill → identify story arcs → draft dev-log publish-proposals → read `docs/marketing/AUDIENCES.md` for contributor-audience framing → raise `capability-gap` if x-dev-log skill or its data sources (git-control-tower, agent-manager, swarm-manager, app-issue-tracker) are unhealthy.
- **TOOLS.md:** `x-dev-log` primary, `campaign-content-studio` secondary, `video-studio` (draft), `seo-optimizer`.

#### 2d. `publisher`

- **SOUL.md:** pipeline operator. Polishes drafts, produces platform variants, schedules, owns `coverage/*.json` state.
- **AGENTS.md workflow:** read pending `content-publish-proposal` decisions → for each with operator approval, append to `publish-log.jsonl` and update corresponding `coverage/<sku-id>.json` → polish unpolished drafts attached to pending proposals → propose `channel-update` when platform rule drift is observed → raise `capability-gap` for missing publishing tooling (social-media-scheduler integration, multi-platform variant generation).
- **TOOLS.md:** `social-media-scheduler`, `seo-optimizer`, `documentation-health`, `campaign-content-studio` (for platform-variant generation).

#### 2e. `researcher`

- **SOUL.md:** audience/competitor/trend scanner. Honest about telemetry gap — does not hallucinate engagement numbers. Feeds benchmark-adjacent observations to monetization's market-validator via knowledge entries.
- **AGENTS.md workflow:** scan for competitor/trend signal → append to `audience-scans.jsonl` → propose `audience-update` decisions when persona revisions are warranted → cross-post benchmark observations to monetization `market-validator` via shared knowledge entry → raise `capability-gap` when research needs a tool that doesn't exist (e.g. competitive-intel scenario).
- **TOOLS.md:** `seo-optimizer` (competitor SEO analysis), `funnel-builder` (conversion/funnel analytics once telemetry exists), `systematic-exploration`.

#### 2f. `marketing-contrarian`

- **SOUL.md:** mandatory skeptic across all marketing-crew member proposals. Owns the aging scan. Domain-specific failure modes (not monetization's). Same structural role as `contrarian` / `meta-contrarian`.
- **AGENTS.md workflow:** team-ceiling check → fetch pending decisions → score against marketing failure-mode list → write `challenge-report/<decision-id>` knowledge entries → aging scan on decisions >14 heartbeats → supersession check on own prior decisions → optionally `decision-rejection-proposed` or `framework-update`.
- **Marketing failure-mode list** (to be stated in SOUL.md, mirrored in TEAM.md):
  1. Hype drift — overpromising features, claiming "soon" without a committed date.
  2. Voice drift — tone migrating from builder/honest to SaaS-marketer.
  3. Hallucinated engagement metrics — unflagged current-state numbers on views, conversions, or reach.
  4. Paywall framing — subscription described as blocking core rather than wrapping core in convenience.
  5. OSS-as-leak framing — free-tier users described as lost revenue.
  6. Coverage-gap-ignorance — new campaigns proposed while deployed SKUs have stale coverage.
  7. Acquisition-only hypothesis — campaign proposal without retention impact analysis, absent an explicit awareness-only flag.
  8. Capability-workaround-without-gap — agent workarounds in notebook without a matching `capability-gap` decision over time.

**Exit criterion:** all 6 agent directories exist and pass schema validation (see Phase 9).

### Phase 3 — Delete old members and old agent definitions

Only after Phase 0 confirmed zero cross-references:

1. Delete `teams/marketing-crew/members/{marketing-lead,content-creator,content-editor,market-researcher}/` directories entirely.
2. Delete `store/agents/{marketing-lead,content-creator,content-editor,market-researcher}/` directories entirely.

**Exit criterion:** `prompt-manager agent list` does not show the deleted agents; `prompt-manager team show marketing-crew` shows the new 6 members (from Phase 4 member-dir creation — these can interleave with Phase 4 if dependency order allows).

### Phase 4 — New member directories

Create `teams/marketing-crew/members/<name>/` for each of the 6 new members. Each contains:

1. `heartbeat.json` — shape:
   ```json
   {
     "kind": "heartbeat-config",
     "schemaVersion": 1,
     "teamId": "marketing-crew",
     "agentId": "<agent-id>",
     "enabled": false,
     "schedule": "0 */6 * * *",
     "revision": 1,
     "createdAt": "<today>",
     "updatedAt": "<today>"
   }
   ```
   (All start disabled per hard rule 3.)

2. `RESPONSIBILITIES.md` — 40-60 lines. Sections: Primary Duties, Owned Decision Contexts, Deliverables (files written, structure), Coordination Points, Honesty Flags & Guardrails, Available Skills.

3. `HEARTBEAT.md` — 100-200 lines, structured after monetization's financial-tracker heartbeat. Sections: Inputs (what to read), Team-ceiling Check, Workflow (numbered steps), Stop-Early Thresholds (per-context cap), Supersession Rules, Output Contract (what gets written where), Honesty Flags, Handoff Format.

4. `logs/.gitkeep` — placeholder directory for future execution logs.

**Exit criterion:** `prompt-manager team heartbeat-list marketing-crew` shows 6 entries, all disabled.

### Phase 5 — Team shared state skeleton

Create empty operational files under `teams/marketing-crew/shared/`:

- `decisions.jsonl`, `knowledge.jsonl`, `handoff-history.jsonl`, `campaign-drafts.jsonl`, `audience-scans.jsonl`, `publish-log.jsonl` — touch empty.
- `coverage/.gitkeep` — directory marker for per-SKU coverage files (files created lazily by publisher at runtime).

### Phase 6 — Plan-of-record docs

Create `docs/marketing/`:

- `README.md` — explains the doc surface, identifies two patterns (plan-of-record here; notebook under `notebook/`), names brand-manager as curator, explains that agents never edit these files directly — they propose via decisions.
- `STRATEGY.md` — voice, positioning, dual-audience framing, anti-patterns, links to monetization `STRATEGY.md` and `CATALOG.md`.
- `AUDIENCES.md` — one section per audience (subscription buyer, OSS contributor), each with a "last-revised" stamp and revisit marker.
- `CAMPAIGNS.md` — empty index, instructions for how new campaign files under `campaigns/<slug>.md` get added via `campaign-launch-proposal` decisions.
- `CHANNELS.md` — per-platform rules (X, blog, video placeholders; import current marketing-crew TEAM.md's platform guidelines section here as the starting content).
- `BRAND.md` — thin placeholder. Says: "When the `brand-manager` scenario ships, this file points at it. Until then, brand canon is prose: [placeholder for voice guidelines from STRATEGY.md]."

**Exit criterion:** all 6 files exist; each has a real (if thin) opening section; README cross-references are valid.

### Phase 7 — Working notebook

Create `docs/marketing/notebook/`:

- `README.md` — copy structural shape from `docs/meta-optimization/README.md`. Same three-tier mental model, same posture language ("debt, not gospel"), same "who writes what" split, but with brand-manager as curator and marketing-crew as owner. Adjust file-table rows for marketing-specific notebook files.
- 5 starter notebook files with the revisit-marker convention:
  - `VIDEO_WORKAROUNDS.md`
  - `POSTING_WORKAROUNDS.md`
  - `AUDIENCE_OBSERVATIONS.md`
  - `CAMPAIGN_LESSONS.md`
  - `DEV_LOG_CRAFT.md`

Each starter file is intentionally thin (2-5 lines explaining what the file is *for* and a "first entry pending" note). Real content arrives when agents run heartbeats.

### Phase 8 — Vision-walk-prep integration

Edit in order:

1. `scenarios/prompt-manager/store/agents/vision-walk-prep/AGENTS.md`:
   - Insert new phase 4.6 (marketing) between phase 4 (monetization) and phase 5 (meta-opt). Full text in §7 target end state.
   - Update any overview/summary paragraphs that enumerate phases.

2. `scenarios/prompt-manager/store/agents/vision-walk-prep/TOOLS.md`:
   - Add the decision-list commands for marketing-crew contexts.

3. `scenarios/prompt-manager/store/skills/packs/core/morning-vision-walk/SKILL.md`:
   - Add a marketing block mirroring the existing monetization/meta-opt blocks. Brief: where decisions come from, what the operator is choosing between.

### Phase 9 — Meta-opt capability-gap authorization update

Edit `scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM.md`:

- In the decision-contexts section where `capability-gap` is described, add the authorized-raisers note from §7.
- In the per-member context enumeration, add a footnote that marketing-crew members may also raise `capability-gap` (this is not ownership — meta-opt still owns the context definition — but the footnote clarifies the shared surface).

### Phase 10 — Validation (see §10 for detail)

Covered in Testing Plan section.

---

## 9. Contract Decisions

### Cross-team data contracts

1. **marketing-crew → monetization (read only):** marketing-crew reads `docs/monetization/STRATEGY.md`, `CATALOG.md`, `PRICING.md`, `TIERS.md`, per-bundle files, `scenario-sku-map.json`. Never writes to `docs/monetization/`.

2. **marketing-crew → meta-optimization (shared context):** marketing-crew members raise `capability-gap` decisions when missing scenario capability blocks their work. Context definition and consumer wiring stay owned by meta-optimization. Marketing-crew is an authorized raiser, not an owner.

3. **marketing-crew → director-swarm (indirect):** marketing-crew's `capability-gap` decisions flow to director-swarm via vision-walk-prep (existing pipe, no new wiring).

4. **marketing-crew → swarm-manager / agent-manager / git-control-tower / app-issue-tracker (read-only queries):** OSS-advertiser reads these for dev-log material via `x-dev-log` skill. No writes.

5. **marketing-crew → landing-page-business-suite:** none directly. Marketing-crew's plan-of-record may reference positioning that lands-page-business-suite consumes, but the consumption wire is via `docs/monetization/`.

6. **vision-walk-prep → marketing-crew:** read-only queries over pending decisions and `challenge-report/*` knowledge entries. No writes.

### Coverage file contract

- **Path:** `teams/marketing-crew/shared/coverage/<sku-id>.json`.
- **Writer:** `publisher` (append-then-rewrite: re-serializes the full file on every update).
- **Readers:** `subscription-advertiser`, `oss-advertiser`, `brand-manager`.
- **SKU ID source:** `docs/monetization/scenario-sku-map.json`.
- **Missing file semantics:** absence of a coverage file for a deployed SKU = coverage gap; publisher creates the file the first time it publishes material for that SKU.
- **OSS scope:** OSS-advertiser uses a synthetic SKU ID `oss-platform` for Vrooli-the-project-overall coverage state.

### Decision-supersession rules (inherited from monetization pattern)

- Before new-decision creation, every member checks pending decisions in its owned context list.
- Obsolete/redundant pending decisions get `status: superseded` and the new decision carries a `supersedes: <prior-id>` field.
- Stacking = guardrail violation.

### Notebook promotion rules

- `notebook-promotion` decision body must state: source file in notebook, proposed target surface (new skill / plan-of-record file / scenario capability), evidence the pattern has stabilized (N examples, N heartbeats).
- `notebook-retirement` decision body must state: source file, evidence the scenario/skill that would have consumed it has shipped (or evidence the pattern is obsoleted).
- Both decisions, on acceptance, cause the operator to edit the source file (delete entry + reorganize as needed). Acceptance never auto-edits the notebook.

---

## 10. Testing Plan

This is a team/config/doc rewrite. Most verification is schema/lint; there is minimal code change. Be honest about what's automated vs manual.

### 10.1 Automated checks

1. **Team config schema validation:**
   ```bash
   prompt-manager team show marketing-crew --dry-run 2>&1
   prompt-manager team heartbeat-list marketing-crew
   prompt-manager team roles marketing-crew
   prompt-manager team org-list marketing-crew
   ```
   Expected: all return success; no schema errors; 6 members listed; 6 roles listed; 0 org-chart edges.

2. **Agent definition schema validation:**
   ```bash
   for id in brand-manager subscription-advertiser oss-advertiser publisher researcher marketing-contrarian; do
     prompt-manager agent show $id || echo "FAIL: $id"
   done
   ```
   Expected: all succeed.

3. **Dry-run heartbeat rendering** — confirm each member's `member-context` output renders (no prompt template errors, no missing-file errors):
   ```bash
   for id in brand-manager subscription-advertiser oss-advertiser publisher researcher marketing-contrarian; do
     prompt-manager team member-context marketing-crew $id > /tmp/mc-$id.txt || echo "FAIL: $id"
   done
   wc -l /tmp/mc-*.txt
   ```
   Expected: 6 non-empty rendered prompts, no failures.

4. **Reference integrity:** no dangling references to deleted agents.
   ```bash
   grep -rn "marketing-lead\|content-creator\|content-editor\|market-researcher" \
     scenarios/prompt-manager/store/ docs/ \
     --include="*.json" --include="*.md"
   ```
   Expected: zero matches outside git-history and this plan file itself.

5. **Vision-walk-prep smoke test:**
   ```bash
   prompt-manager agent show vision-walk-prep
   grep -c "marketing-crew\|marketing decisions" \
     scenarios/prompt-manager/store/agents/vision-walk-prep/AGENTS.md \
     scenarios/prompt-manager/store/agents/vision-walk-prep/TOOLS.md
   ```
   Expected: non-zero matches in both files.

6. **Doc link validation** — every relative link in new `docs/marketing/*.md` and `docs/marketing/notebook/*.md` points at an existing file.
   ```bash
   # use whatever doc-link-checker is present; fallback:
   grep -rn '(\.\./\|\.\/' docs/marketing/ | grep -v "notebook/" | while read line; do
     # manual inspection per target; scripted check acceptable if a link-checker exists
     :
   done
   ```

7. **JSON validity:**
   ```bash
   for f in $(find scenarios/prompt-manager/store/teams/marketing-crew scenarios/prompt-manager/store/agents/{brand-manager,subscription-advertiser,oss-advertiser,publisher,researcher,marketing-contrarian} -name "*.json"); do
     jq empty "$f" || echo "INVALID: $f"
   done
   ```
   Expected: no `INVALID` output.

### 10.2 Manual / semi-automated checks

These cannot be meaningfully unit-tested; the verifier reads for correctness:

1. **TEAM.md completeness** — all 13 required sections present and non-empty (mission, coordination pattern, members, operating rules, decision contexts, queue discipline, shared state, plan-of-record docs, working notebook, cross-team coordination, heartbeat coordination principle, key skills, anti-patterns).
2. **Member HEARTBEAT.md density** — each is in the 100-200 line range, not a 2-bullet placeholder.
3. **SOUL.md distinctness** — each agent's SOUL.md has a unique identity statement; no two agents read as interchangeable.
4. **Capability-gap wiring** — meta-opt TEAM.md authorized-raisers note present; vision-walk-prep phase 2 unchanged (still consumes `capability-gap` for director-swarm).
5. **Notebook README mirrors meta-opt README structure** — posture language, three-tier model, curator named.

### 10.3 Not tested

- Heartbeat *execution* of the 6 new members — heartbeats ship disabled. Enabling and observing real output is follow-up operator work, not part of this plan's acceptance.
- End-to-end vision-walk integration against real pending decisions — no marketing decisions exist yet.
- Quality of the starter prose in plan-of-record / notebook — expected to be thin and iterated on later.

---

## 11. Rollout / Validation Checklist

Execute in order. Every item produces a visible pass/fail signal.

- [ ] Phase 0 grep for cross-references of the 4 old agents returns zero results outside marketing-crew's own directory.
- [ ] Phase 0 collision check for 6 new agent IDs returns zero results.
- [ ] Phase 1: `prompt-manager team show marketing-crew` returns new config (independent / multi-process / approval / revision 4).
- [ ] Phase 2: `prompt-manager agent show <id>` passes for all 6 new agents.
- [ ] Phase 3: old 4 member dirs and old 4 agent dirs are absent from the filesystem.
- [ ] Phase 4: `prompt-manager team heartbeat-list marketing-crew` shows 6 disabled entries.
- [ ] Phase 5: 6 empty .jsonl files exist under `teams/marketing-crew/shared/`; `coverage/` directory exists.
- [ ] Phase 6: 6 plan-of-record files exist under `docs/marketing/`; README lists all of them.
- [ ] Phase 7: 6 notebook files exist under `docs/marketing/notebook/`; README mirrors meta-opt posture.
- [ ] Phase 8: `vision-walk-prep/AGENTS.md` contains a phase 4.6 section naming marketing-crew; `morning-vision-walk` skill references marketing.
- [ ] Phase 9: meta-opt TEAM.md contains "Authorized raisers" line for `capability-gap` naming marketing-crew members.
- [ ] Phase 10.1 all 7 automated checks pass.
- [ ] Phase 10.2 all 5 manual checks pass under reviewer eyes.
- [ ] Operator restarts prompt-manager manually; `prompt-manager team show marketing-crew` still renders correctly after restart.
- [ ] Operator spot-enables **one** member heartbeat (recommend `researcher` as lowest-risk single-output starter), triggers it manually, and observes it produces sensible output against live `docs/monetization/` + notebook inputs. Spot-disable after confirming.

---

## 12. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A new agent ID collides with an existing one we missed | Low | High (team won't load) | Phase 0 grep + collision check; abort before writes if collision found. |
| Deleting `content-creator` / `content-editor` / `market-researcher` / `marketing-lead` breaks a cross-team wire we missed | Low | Medium (other team breaks) | Phase 0 cross-reference grep scoped beyond marketing-crew; fix inline before delete. |
| vision-walk-prep phase insertion breaks existing numbering / phase references inside the skill | Medium | Medium (morning walk renders wrong) | Use 4.6 (non-renumbering) instead of pushing 5 → 6; verify with `prompt-manager agent show vision-walk-prep` after edit. |
| Capability-gap context becomes noisy because marketing raises more gaps than meta-opt ever did | Medium | Low | Monitor. If noise becomes real, add a per-raiser sub-context (e.g., `capability-gap/marketing`) in a follow-up plan — not now. |
| Members' HEARTBEAT.md instructions are under-specified and the agents hallucinate their jobs on first-run | High | Medium (bad first outputs) | Ship with heartbeats disabled; enable one member at a time; observe and refine the HEARTBEAT.md before enabling the next. Build in the revisit-marker convention from day one. |
| Notebook becomes a growing debt pile if brand-manager never gets enabled | High | Medium (notebook rots) | `brand-manager` is the curator; if it isn't enabled, don't enable the other members either. State this in TEAM.md as an operational constraint. |
| Coverage file format proves wrong after first real use | Medium | Low | Operator edits format directly; file shape documented in TEAM.md and easy to revise. No downstream consumers outside the team at start. |
| User's prompt-manager session picks up the new team config before we've written all files | Medium | Low (temporary broken state) | Write in dependency order (agent defs before team refs them; team config last among the teams/ edits). Do not advise operator to restart mid-way. |
| OSS-advertiser and subscription-advertiser produce overlapping content on the same news item | Medium | Low | marketing-contrarian's failure-mode list includes cross-member-duplicate scan; `content-publish-proposal` decisions tagged with audience so duplicates are visible. |
| Researcher tries to report engagement metrics that don't exist | Medium | Medium (hallucinated numbers) | Operating rule 4 enforces honesty flags; marketing-contrarian failure mode 3 catches unflagged numbers. |

---

## 13. Non-goals / Prohibited Patterns

- **No marketing-lead.** Not renamed, not repurposed, not re-created. The operator is team lead via the morning walk.
- **No auto-posting.** Every artifact ships via an approved `content-publish-proposal` decision.
- **No `approval` on notebook appends.** Append-anyone; operator acts only on `notebook-promotion` / `notebook-retirement` decisions.
- **No compat shims for deleted agents.** No `/agents/marketing-lead` symlink, no deprecation note, no alias.
- **No telemetry-scenario authoring.** Same discipline as monetization — capability gaps are raised, not built.
- **No cross-writes.** marketing-crew never writes to `docs/monetization/`, `docs/meta-optimization/`, or any other team's shared state.
- **No mixing plan-of-record and notebook content.** An entry lives in exactly one place. Promotion deletes from notebook.
- **No skill rewrites in this plan.** x-dev-log, campaign-content-studio, brand-manager skill, etc. are referenced, not edited. Skill improvements land as separate `meta-self-improvement` or `skill-improvement` decisions via meta-optimization.
- **No marketing-contrarian inventing failure modes.** If it sees a real flaw outside the stated 8 failure modes, it proposes a `framework-update` decision (same discipline as monetization/meta-opt contrarians).
- **No enabling heartbeats as part of this plan.** Shipping disabled is the point — the operator enables one member at a time.

---

## 14. Definition of Done

All of these must be true:

1. `scenarios/prompt-manager/store/teams/marketing-crew/` contains: `team.json` (coordination.pattern = independent, runtime.mode = multi-process, decisionMode = approval, revision ≥ 4), `org.json` (edges = []), `roles.json` (6 roles), `shared/TEAM.md` (all 13 sections present, following monetization structural template).
2. `scenarios/prompt-manager/store/teams/marketing-crew/members/` contains exactly 6 subdirectories: `brand-manager`, `subscription-advertiser`, `oss-advertiser`, `publisher`, `researcher`, `marketing-contrarian`. Each with `heartbeat.json` (disabled), `RESPONSIBILITIES.md` (40-60 lines), `HEARTBEAT.md` (100-200 lines), `logs/`.
3. `scenarios/prompt-manager/store/agents/` contains exactly 6 new agent dirs for the members above. Each with `agent.json`, `AGENTS.md` (30-60 lines), `SOUL.md` (20-40 lines), `TOOLS.md` (30-60 lines).
4. Old dirs removed: `store/teams/marketing-crew/members/{marketing-lead,content-creator,content-editor,market-researcher}/` and `store/agents/{marketing-lead,content-creator,content-editor,market-researcher}/`.
5. `teams/marketing-crew/shared/` contains: `TEAM.md`, `decisions.jsonl`, `knowledge.jsonl`, `handoff-history.jsonl`, `campaign-drafts.jsonl`, `audience-scans.jsonl`, `publish-log.jsonl`, `coverage/`.
6. `docs/marketing/` contains: `README.md`, `STRATEGY.md`, `AUDIENCES.md`, `CAMPAIGNS.md`, `CHANNELS.md`, `BRAND.md`.
7. `docs/marketing/notebook/` contains: `README.md`, `VIDEO_WORKAROUNDS.md`, `POSTING_WORKAROUNDS.md`, `AUDIENCE_OBSERVATIONS.md`, `CAMPAIGN_LESSONS.md`, `DEV_LOG_CRAFT.md`.
8. `vision-walk-prep/AGENTS.md` includes phase 4.6 covering marketing-crew decisions; `vision-walk-prep/TOOLS.md` references the new queries; `morning-vision-walk` SKILL.md has a marketing block.
9. `teams/meta-optimization/shared/TEAM.md` has the authorized-raisers annotation under `capability-gap`.
10. All automated checks in §10.1 pass (team show, agent show, heartbeat-list, member-context render, reference-integrity grep, vision-walk-prep smoke, doc-link validation, JSON validity).
11. All manual checks in §10.2 pass under reviewer eyes.
12. No file outside the scope listed in §5 is modified.
13. No heartbeat is enabled. All 6 new `heartbeat.json` files have `"enabled": false`.
14. Greenfield-hard-rule compliance: no compat shims, no symlinks, no deprecated aliases, no `-legacy` dirs, no `TODO: remove` comments.
15. Operator restarts prompt-manager manually and confirms `prompt-manager team show marketing-crew` renders the new config (this step is operator-executed, not Claude-executed, per hard rule 3).

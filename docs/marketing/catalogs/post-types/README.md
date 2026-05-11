# Marketing Post Types — Plan of Record

This folder is the **strategic canon** for each kind of marketing post Vrooli produces. One file per post type, organized by primary medium.

These docs answer: *what is this type for, who is it aimed at, what is the conversion goal, what does a good one look like, what failure modes does the marketing-contrarian watch for?*

They do **not** answer: *here is the prompt, here is the round structure, here are the data sources to mine.* That is the role of the per-type **skill** under [`path:scenarios/prompt-manager/store/skills/packs/core/x-<type>/`](path:scenarios/prompt-manager/store/skills/packs/core/) (e.g., `x-dev-log`, `x-scenario-spotlight`). The skill is the *executable spec* an agent runs; the file in this folder is the *strategic canon* the operator reads to make decisions and the agent reads as required-reading context.

## Folder structure — by medium

Post types are grouped by their **primary medium** — what the reader/viewer/listener is fundamentally consuming. The medium is not the channel: a `text` post can cross-post to X, Threads, Bluesky, LinkedIn, and a blog; a `video` post can cross-post to TikTok, Reels, Shorts, YouTube, and embedded blog. The medium determines production tooling, asset requirements, and the skill's executable shape.

```
post-types/
  text/      # primary medium is the text post; assets support
  image/     # primary medium is one or more static images
  video/     # primary medium is moving video (or slideshow rendered as video)
```

## Doc + skill discipline (mandatory)

**Every post type ships as `doc + paired skill`.** This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other:

- **The doc** (in `path:docs/marketing/catalogs/post-types/<medium>/<slug>.md`) is the *strategic canon*: purpose, audience, conversion goal, asset weight, structure, hook patterns, type-specific contrarian failure modes. The operator reads it to make decisions; the marketing-contrarian uses it to score drafts; the agent reads it as required-reading context. Stable; changes only via `brand-guideline-update` decisions.
- **The skill** (in `path:scenarios/prompt-manager/store/skills/packs/core/x-<slug>/`) is the *executable spec*: which CLI calls to make, which scenarios to query, which JSON to assemble, which guardrails to apply. The agent runs it. Mutable; changes via skill-edit decisions.

The two halves carry **different content**: the doc holds *reasoning*, the skill holds *procedure*. A doc with no skill is a stale shrine — strategic canon nobody can act on. A skill with no doc is brittle — procedure that can't be checked against intent. Both halves required.

### Lifecycle: v0 (doc-only) → v1 (doc + skill, active)

A post-type entry can exist as a v0 stub before its skill is authored. v0 means **the strategic canon is documented but the type is not yet active in production**. Drafts of a v0 type cannot be approved through `content-publish-proposal` — there is no executable spec to run them against, and no hooks for the contrarian to score type-level failure modes consistently.

Activation (v0 → v1) requires:

1. The paired `x-<slug>` skill is authored and lives at `path:scenarios/prompt-manager/store/skills/packs/core/x-<slug>/SKILL.md`.
2. The skill's required-reading list cites the post-type doc (so the agent loads the canon before producing).
3. The doc's `Status:` line is bumped from `v0 (skeleton)` to `v1` and the `Paired skill:` line drops the `*(planned)*` annotation.
4. At least one member's `RESPONSIBILITIES.md` references the skill in its Available Skills table.

These four are checkable. The marketing-contrarian's review of any `content-publish-proposal` naming a recognized post type validates v1 status before allowing approval.

### Why one skill per post type — not a unified `x-post-author`

Skills compress over time, and **compression operates per-skill**. The skill-compression trajectory:

```
v1: long prose for an LLM to think through
       ↓
v2: shorter prose + a few specific CLI invocations
       ↓
vN: near-zero prose; CLI does the work
```

The meta-optimization team has a member that explicitly drives this compression — analyzing how each skill is used in production and proposing edits that push token-expensive reasoning out of the skill and into Vrooli's scenario CLIs. A skill compresses by binding to its specific substrate (the data sources it queries, the assets it composes, the failure modes it checks for). A *unified* `x-post-author` skill that branches on type would compress worse: its compression depends on whichever substrate is most generic, and post-type-specific reasoning that *should* fall out into a CLI command stays trapped in branching prose.

Authoring one skill per type — even when types share substrate — is the right shape because each skill can compress independently. As Vrooli's CLI absorbs more work (rich-media-studio, image-gen, video-gen, social-media-scheduler), each `x-<type>` skill thins out independently toward its own near-zero-prose endpoint.

This principle generalizes across teams (not just marketing) — see the meta-optimization team's charter for the cross-team articulation.

## Why per-entity files (not subsections of STRATEGY.md)

Per [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../../../agent-system/TEAM_DOCS_PATTERNS.md): monolithic plan-of-record docs invite accidental damage when adding/retiring one entity. Per-entity files keep edits scoped and let each post type evolve at its own pace. Cross-cutting structural rules (essay-shape, hook-vs-body asymmetry, intro-on-first-mention, inter-post linkage, recommendation-framing, competitive-comparison) live in [`../../methods/post-techniques/`](../../methods/post-techniques/) and are referenced from each post-type file rather than duplicated. Strategies that span multiple post-types (e.g., AI-UGC personas) live in [`../../strategy/patterns/`](../../strategy/patterns/).

## Decision tree — which post type when?

Ask the questions in order. The first "yes" picks the type.

1. **Is the operator trying to show project-wide progress / building-in-public narrative?** → [`doc:docs/marketing/catalogs/post-types/text/dev-log.md`](text/dev-log.md)
2. **Is the operator trying to get a specific person to use, sign up for, or buy a single Vrooli scenario as an app/product?** → [`doc:docs/marketing/catalogs/post-types/text/scenario-spotlight.md`](text/scenario-spotlight.md) (text-led, asset-supported)
3. **Is the operator trying to get developers to adopt Vrooli as an OSS framework for building agentic apps?** → `doc[future]:docs/marketing/catalogs/post-types/text/oss-framework.md` *(planned)*
4. **Is the operator trying to teach a complete use-case end-to-end (recipe + demo)?** → `doc[future]:docs/marketing/catalogs/post-types/text/use-case-tutorial.md` *(planned)*
5. **Is the operator trying to grab attention on a visual feed (Instagram, ad placements) with a single still image?** → [`doc:docs/marketing/catalogs/post-types/image/single-image-ad.md`](image/single-image-ad.md) or [`doc:docs/marketing/catalogs/post-types/image/infographic.md`](image/infographic.md)
6. **Is the operator trying to give general-audience tips that lead toward a Vrooli scenario in the last frame?** → [`doc:docs/marketing/catalogs/post-types/image/slideshow-tips-then-plug.md`](image/slideshow-tips-then-plug.md) (or `literal:slideshow-listicle` for a numbered-tips variation, or `doc[future]:docs/marketing/catalogs/post-types/video/slideshow-voiceover.md` if rendered as video with narration)
7. **Is the operator trying to produce short-form video for TikTok, Reels, or Shorts?** Pick from `video/`:
   - **Persona-actor narrating advice/story?** → [`doc:docs/marketing/catalogs/post-types/video/narrative-talking-head.md`](video/narrative-talking-head.md)
   - **Lifestyle / routine vignette?** → [`doc:docs/marketing/catalogs/post-types/video/day-in-life-ugc.md`](video/day-in-life-ugc.md)
   - **Pain → escalation → solution structure?** → [`doc:docs/marketing/catalogs/post-types/video/problem-agitate-solve.md`](video/problem-agitate-solve.md)
   - **Side-by-side or before/after comparison?** → [`doc:docs/marketing/catalogs/post-types/video/comparison-reel.md`](video/comparison-reel.md)
   - **Slideshow rendered with voiceover?** → [`doc:docs/marketing/catalogs/post-types/video/slideshow-voiceover.md`](video/slideshow-voiceover.md)
8. **Is the operator trying to produce a screen-recorded demo of a scenario at length (YouTube, blog embed)?** → [`doc:docs/marketing/catalogs/post-types/video/demo-recording.md`](video/demo-recording.md)
9. **None of the above** → likely doesn't need a structured post type. Capture as a one-off and let the marketing-crew researcher / publisher decide.

## Files in this folder

### `text/`

| File | Status | Primary skill | Primary lane/member |
|------|--------|---------------|----------------|
| [`text/dev-log.md`](text/dev-log.md) | v1 (extracted from `STRATEGY.md` on 2026-04-28; moved into `text/` 2026-04-28) | `x-dev-log` | OSS lane (`oss-advertiser`) |
| [`text/scenario-spotlight.md`](text/scenario-spotlight.md) | v1 (moved into `text/` 2026-04-28) | `x-scenario-spotlight` | Subscription lane (`subscription-advertiser`) |
| `text/oss-framework.md` | *planned — Post 3 from walk #5 (jcode-vs-Claude-Code-and-Codex framing) is a candidate reference* | `x-oss-framework` *(future)* | OSS lane (`oss-advertiser`) |
| `text/use-case-tutorial.md` | *planned — pending observation of 2-3 candidates in our own funnel* | `x-use-case-tutorial` *(future)* | Usually OSS lane (`oss-advertiser`); may split later |

### `image/`

| File | Status | Primary skill | Primary lane/member |
|------|--------|---------------|----------------|
| [`image/single-image-ad.md`](image/single-image-ad.md) | v0 (skeleton — awaiting first production run) | `x-single-image-ad` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`image/slideshow-listicle.md`](image/slideshow-listicle.md) | v0 (skeleton) | `x-slideshow-listicle` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`image/slideshow-tips-then-plug.md`](image/slideshow-tips-then-plug.md) | v0 (skeleton) | `x-slideshow-tips-then-plug` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`image/infographic.md`](image/infographic.md) | v0 (skeleton) | `x-infographic` *(future)* | Lane depends on subject: subscription for lifestyle/general, OSS for technical |

### `video/`

| File | Status | Primary skill | Primary lane/member |
|------|--------|---------------|----------------|
| [`video/narrative-talking-head.md`](video/narrative-talking-head.md) | v0 (skeleton) | `x-narrative-talking-head` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`video/day-in-life-ugc.md`](video/day-in-life-ugc.md) | v0 (skeleton) | `x-day-in-life-ugc` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`video/problem-agitate-solve.md`](video/problem-agitate-solve.md) | v0 (skeleton) | `x-problem-agitate-solve` *(future)* | Usually subscription lane (`subscription-advertiser`) |
| [`video/demo-recording.md`](video/demo-recording.md) | v0 (skeleton) | `x-demo-recording` *(future)* | Lane depends on demo subject; publisher handles release package |
| [`video/comparison-reel.md`](video/comparison-reel.md) | v0 (skeleton) | `x-comparison-reel` *(future)* | Lane depends on comparison subject |
| [`video/slideshow-voiceover.md`](video/slideshow-voiceover.md) | v0 (skeleton) | `x-slideshow-voiceover` *(future)* | Usually subscription lane (`subscription-advertiser`) |

## Cross-cutting techniques (shared across types)

Each post-type file references the techniques it depends on. The canonical home for each technique is [`../../methods/post-techniques/<name>.md`](../../methods/post-techniques/):

- Essay-shape per post (hook → intro → body → conclusion)
- Hook-vs-body length asymmetry
- Intro-on-first-mention (with `shared/published-scenario-mentions.jsonl` lookup)
- Inter-post linkage (incl. cross-platform amplification)
- No internal numbering externally
- Recommendation framing — third-party voice, applies when genuine third-party basis exists
- Competitive comparison — framing against named alternatives

For multi-frame and persona-actor content (image slideshows, video formats), see also:

- [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md) — disclosure rules and persona-actor account discipline.
- [`../rich-media/`](../rich-media/) — character / scene / product schemas that drive consistent multi-frame generation.
- [`../../strategy/patterns/hook-library.md`](../../strategy/patterns/hook-library.md) — proven hook patterns by platform/audience/outcome.

## Write rules

Same as the rest of `path:docs/marketing/`: agents never write directly; operator-curated via approved decisions.

- **New post-type proposals** come through the `post-type-proposal` decision context, owned by `researcher`. The proposal must include:
  1. The strategic canon content (purpose, audience, conversion goal, asset requirements, contrarian failure modes) — this is what authors the v0 doc.
  2. The proposed paired-skill name (`x-<slug>`) and which member(s) will consume it.
  3. The member's `RESPONSIBILITIES.md` Available Skills update.
  4. A commitment to author the skill within N heartbeats of doc acceptance, OR an explicit `v0-stub-only` flag if the type is being scaffolded for later activation.
- **Activation (v0 → v1)** is its own decision: when the paired skill is authored, the operator approves the activation by bumping the doc's `Status:` line and dropping the `*(planned)*` from `Paired skill:`. Activation is what makes the type usable in `content-publish-proposal` decisions.
- **Edits** to an existing post-type's strategic canon use `brand-guideline-update`. Skill-edits are skill-edit decisions on the skill side.
- **Type-specific platform-rule updates** use `channel-update`.

The marketing-contrarian validates v1 status as part of every `content-publish-proposal` review for a recognized type. Drafts of v0-stub types are blocked at the contrarian gate until the type is activated.

## Cross-references

- [`../../strategy/STRATEGY.md`](../../strategy/STRATEGY.md) — voice canon, dual-audience framing, anti-patterns (now eighteen named modes).
- [`../../methods/post-techniques/`](../../methods/post-techniques/) — cross-cutting voice and structure techniques.
- [`../../strategy/patterns/`](../../strategy/patterns/) — strategies that span multiple post-types (AI-UGC personas, hook library, funnel patterns).
- [`../rich-media/`](../rich-media/) — character / scene / product / prompt-template schemas for image and video generation.
- [`../../strategy/CHANNELS.md`](../../strategy/CHANNELS.md) — channel × format matrix showing which post-types map to which platforms.
- [`../../notebook/`](../../notebook/) — working notebook for patterns observed in production. Each post-type's promotion target may be either *this folder* (strategic refinements) or its paired skill (executable refinements).

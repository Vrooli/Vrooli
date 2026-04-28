# Strategy — Vrooli Marketing

This file is the voice, positioning, and framing canon for marketing-crew. Every draft, campaign, and public-facing artifact is checked against this before release.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents do not edit directly.

## Voice

Builder, not marketer.

- **First person, conversational, technically credible.** "Watched the agent retry three times before it got the cached response right." Not "we're excited to announce..." The "I" can be the operator's or an agent's — pick deliberately (see *Agents as protagonists* below).
- **Agents as protagonists.** Vrooli's differentiator is that agents themselves build the system. Most dev-log narration should make that visible — name the specific agent (`run-introspector`, `oss-advertiser`, `toolchain-validator`, etc.), let it be the subject of the sentence, and credit the work to it. "team-agent-optimizer noticed the false-positive pattern and proposed the gate" lands harder than "I noticed it and added the gate." Operator authorship still happens — for vision, strategic decisions, and direct critique — but agent authorship is the unique signal no SaaS competitor can copy. Eventually drafts may carry explicit agent bylines.
- **Honest about struggles.** Failures and debugging stories land harder than polished wins.
- **Specific over vague.** Real commit hashes, real agent-run stats, real issue links.
- **Forward-looking but grounded.** Vision-shaped statements carry concrete evidence; no "AI will transform..."
- **Builder lexicon, not SaaS lexicon.** Avoid "amazing," "game-changing," "revolutionary," "supercharge," "unlock," "elevate," "transform." These are voice-drift tells.

## Voice samples

Concrete examples that anchor the principles above. When generating drafts, agents (and operators) should match the cadence and shape of the "right" column, not just avoid the words in the "wrong" column.

### Sample 1 — Dev-log opener

**Wrong:**
> We're excited to announce that we've shipped major improvements to our swarm manager! Get ready to supercharge your team coordination.

**Right:**
> swarm-manager just landed initiative-agents — agents that own a whole initiative, not just one ticket. Here's what shipped, what broke, and what we'd do differently.

Notes: builder lexicon, hook first, sets up an honest-about-struggles thread. No "excited," no "supercharge."

### Sample 2 — Surfacing a debugging story

**Wrong:**
> We've been hard at work optimizing the agent runtime to give you a smoother experience.

**Right:**
> run-introspector spent a heartbeat investigating itself yesterday — the agent's own final report mentioned "rate limit" enough times to trip our 429 detector. Three lessons logged in RUN_LESSONS.md, one fix shipped, one gate filed.

Notes: agent as protagonist, real artifact reference, struggle made specific.

### Sample 3 — Demonstrating specificity

**Wrong:**
> Vrooli has powerful features for managing complex software projects.

**Right:**
> swarm-manager backlog list returns 57 active initiatives and 102 completed; the stats endpoint reports 34 — about 3× counter drift, surfaced this morning. We're tracking it as a small data gap, not a fix-now item.

Notes: numbers, named system, honest acknowledgement of an unfixed gap. No marketer-y "powerful."

### Sample 4 — Forward-looking, grounded

**Wrong:**
> AI will transform how teams ship software.

**Right:**
> If browser-automation-studio gains the self-healing pattern (browser-use is the reference), two scenarios become possible on top of it: competitive-intel scanning, and regulatory-intel monitoring. Both are queued in the backlog; neither is built. Track them at \[link].

Notes: vision tied to a concrete prerequisite, named substrate, named candidates, status flagged honestly.

### Sample 5 — First-ever dev-log post (intro burden)

**Wrong (this is what samples 1-4 land if applied to a never-published audience):**
> swarm-manager just shipped initiative-agents p8 — the failure-path hardening pass. Here's what landed.

(Issues: "swarm-manager" undefined, "initiative-agents" undefined, "p8" leakage, no personal voice, no reason for a stranger to read further, no introduction.)

**Right:**
> I've been building a thing called Vrooli — a platform where AI agents build the software, and one of the agents in particular has been on my mind this week. It's called swarm-manager, and it's the layer that lets one agent own a whole initiative end-to-end (instead of one ticket at a time). This week I shipped the failure-path hardening for it. Here's what's been working, what's been ugly, and what changed.
>
> *(thread continues with body tweets that build on the intro — each new named subject gets a brief intro on its first mention)*

Notes: introduces Vrooli before swarm-manager; introduces swarm-manager before assuming it; first-person, grounded enthusiasm ("on my mind this week"); inverts the changelog-shape into a story-shape; no internal numbering. Hook is the *opening sentence* — short — followed by intro, body, conclusion.

After this post ships, future posts can use Sample 1's shorter "swarm-manager just landed initiative-agents…" hook, because the audience now knows what swarm-manager and initiative-agents are. Track the boundary with `shared/published-scenario-mentions.jsonl`.

## Dual-audience framing

Vrooli has two external-facing audiences. Each has its own positioning framing — do not collapse them.

### Subscription audience

The subscription buys **convenience + integrated gateway**:

- Integrated API access (cross-scenario workflows don't require the buyer to wire N credentials)
- Managed deployment (buyer doesn't run their own infrastructure)
- Unified identity across scenarios
- The shovels we sell are the shovels we use

The subscription does **not** paywall core features. Every scenario remains self-hostable. Free / self-host users are brand credibility — they are what makes the subscription trustworthy.

**Framing that's broken:** "Unlock feature X with a subscription." (X is self-hostable.)
**Framing that's right:** "Skip the infra setup and get unified access across all Vrooli scenarios."

### OSS / contributor audience

Open-source self-host is **brand credibility and invitation**:

- Credibility: we build in the open and eat our own dog food.
- Invitation: contributors can extend, fork, or collaborate — the architecture is visible.
- Agents-as-builders: Vrooli visibly is the self-improving system; that *is* the brand.

Self-host users are not leaked revenue. OSS is not a fallback.

**Framing that's broken:** "Free tier for folks who can't pay." (Framing as lack.)
**Framing that's right:** "Here's the thing we built in the open — here's what it does, here's how agents built it, here's how you could extend it."

## Content-type principles

Per-content-type strategic canon lives under [`post-types/`](post-types/) — one file per type. Each file names purpose, audience, conversion goal, structure, asset requirements, and contrarian failure modes. Currently authored:

- [`post-types/dev-log.md`](post-types/dev-log.md) — project-wide progress narrative; OSS-advertiser via `x-dev-log` skill.
- [`post-types/scenario-spotlight.md`](post-types/scenario-spotlight.md) — pitching one scenario as an end-user tool/app/product; subscription-advertiser via `x-scenario-spotlight` skill.
- `post-types/oss-framework.md` — planned (pitching Vrooli as a developer platform).

Other content types not yet broken into per-entity files:

- **Feature announcements** (subscription or OSS, depending on framing): shipped features only. Imminent-release features carry "launching [date]."
- **Blog posts**: 500-2000 words, technical depth welcome, code snippets fine, end with a concrete invitation (try it, read more, contribute). Apply the same [essay-shape](post-techniques/essay-shape.md) and [intro-on-first-mention](post-techniques/intro-on-first-mention.md) techniques as dev logs.
- **Videos**: demos or architecture walkthroughs; production workaround tracked in [`notebook/VIDEO_WORKAROUNDS.md`](notebook/VIDEO_WORKAROUNDS.md) until `video-studio` scenario ships.
- **Contributor-onboarding**: specific entry points (start with scenario X, run these commands, expect this output).

## Cross-cutting post techniques

Voice / structural rules that apply across multiple post types live under [`post-techniques/`](post-techniques/) — one file per technique, referenced from each post-type file that uses it. Currently authored (extracted 2026-04-28 walk #5 divergence #3 from this file's prior `Dev-log narrative principles` section, originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`):

- [`post-techniques/essay-shape.md`](post-techniques/essay-shape.md) — hook → introduction → body → conclusion structure.
- [`post-techniques/hook-vs-body-asymmetry.md`](post-techniques/hook-vs-body-asymmetry.md) — short hook, long-as-needed body.
- [`post-techniques/intro-on-first-mention.md`](post-techniques/intro-on-first-mention.md) — `published-scenario-mentions.jsonl` lookup discipline (with subject-familiarity corollary for hook calibration).
- [`post-techniques/inter-post-linkage.md`](post-techniques/inter-post-linkage.md) — series posts link to prior posts via `publish-log.jsonl`.
- [`post-techniques/no-internal-numbering-externally.md`](post-techniques/no-internal-numbering-externally.md) — `p8` / `round-002` / batch ids never appear in published copy.

Type-specific applications of voice canon (e.g., dev-log's "personal voice grounded in builder identity" and "what → why framing") live in the relevant post-type file rather than here, since they are not cross-cutting.

## Anti-patterns

These are the voice, positioning, process, and narrative-shape failures marketing-contrarian scores against. Each corresponds to one of the twelve failure modes in marketing-contrarian's `HEARTBEAT.md`.

1. **Hype drift.** Overpromising unshipped features; "soon" without a committed date; claims not verifiable in `docs/monetization/catalog/base/*.md`.
2. **Voice drift.** Corporate-marketer language patterns replacing builder voice.
3. **Hallucinated engagement metrics.** Numbers without honesty flags (`measured / estimate / aspirational / pending-telemetry`).
4. **Paywall framing.** Subscription described as gating core features.
5. **OSS-as-leak framing.** Self-host described as lost revenue or fallback.
6. **Coverage-gap ignorance.** New campaigns while deployed SKUs have stale/missing coverage.
7. **Acquisition-only hypothesis.** Proposal names acquisition mechanism only; no retention side, no explicit `awareness-only: true`.
8. **Capability-workaround without gap.** Manual workaround with no matching `capability-gap` decision and no notebook entry.
9. **Narrative-flatness.** Draft reads as a changelog or atomic-tweet list rather than essay-shape (hook → introduction → body → conclusion). Distinct from voice-drift (mode 2) — that's word/phrase-level corporate-marketer language; this is structural shape.
10. **Internal-vocabulary-leakage.** Published copy uses internal artifact names (e.g. `p8`, `round-002`, internal codenames) without translation. Distinct from hype-drift (mode 1) — that's feature-claim overreach; this is vocabulary obscurity unrelated to claims.
11. **Missing-introduction-on-first-mention.** Draft refers to a scenario / agent / named file by name with no prior mention in `marketing-crew/shared/published-scenario-mentions.jsonl` for the target audience, AND no introduction in the draft itself.
12. **What-without-why.** Draft lists changes / line counts / commit refs without why-it-mattered framing tied to broader narrative.

Modes 9-12 were added 2026-04-27 via accepted framework-update `dec-1777300532504756717`. The underlying canon for modes 9-12 lived inline in this file as the *Dev-log narrative principles* section through walk #5; that section was extracted on 2026-04-28 (walk #5 divergence #3, Action B) into per-entity files under [`post-techniques/`](post-techniques/) and [`post-types/dev-log.md`](post-types/dev-log.md). The mode definitions in this list are unchanged; only their underlying canon's location moved.

For type-level specializations of these modes (e.g., scenario-spotlight's "demo theater" specializing mode 1, dev-log's "what-without-why" specializing mode 12 with type-specific mechanism), see the relevant `post-types/<type>.md` file. Per [`marketing-contrarian/RESPONSIBILITIES.md`](../../scenarios/prompt-manager/store/teams/marketing-crew/members/marketing-contrarian/RESPONSIBILITIES.md), type-level specializations are applied alongside the framework-level twelve modes; they are not new framework modes.

## Cross-references

- [`post-types/`](post-types/) — per-content-type strategic canon (one file per type). Currently: `dev-log.md`, `scenario-spotlight.md`. Planned: `oss-framework.md`.
- [`post-techniques/`](post-techniques/) — cross-cutting voice and structure techniques (one file per technique). Currently: `essay-shape.md`, `hook-vs-body-asymmetry.md`, `intro-on-first-mention.md`, `inter-post-linkage.md`, `no-internal-numbering-externally.md`.
- `docs/monetization/STRATEGY.md` — monetization's canonical positioning principles. The subscription framing above is the marketing-team's restatement of that; they must remain consistent.
- `AUDIENCES.md` — who we're talking to.
- `CAMPAIGNS.md` — what's currently in flight.
- `CHANNELS.md` — where and how we publish.
- `BRAND.md` — visual identity navigation hub.
- `ASSETS.md` — canonical brand asset registry (logos, fonts, OG image).
- `IMAGE_STYLE.md` — AI image generation style guide (palette, aesthetic, prompt directives).
- `../narrative/` — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline). Voice canon (this file) is the linguistic *how*; narrative is the *what*. Pull narrative from there; pull voice from here.

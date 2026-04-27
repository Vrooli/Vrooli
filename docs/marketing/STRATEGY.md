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

- **Dev logs** (OSS-advertiser via `x-dev-log`): see *Dev-log narrative principles* below — essay-shape (hook → intro → body → conclusion), 3-7 tweets per thread is a typical range *not* a structural cap, intro-on-first-mention, what→why framing, inter-post linkage, no internal numbering externally. Work-in-progress explicitly labeled.
- **Feature announcements** (subscription or OSS, depending on framing): shipped features only. Imminent-release features carry "launching [date]."
- **Blog posts**: 500-2000 words, technical depth welcome, code snippets fine, end with a concrete invitation (try it, read more, contribute). Same essay-shape and intro-on-first-mention rules as dev logs.
- **Videos**: demos or architecture walkthroughs; production workaround tracked in `notebook/VIDEO_WORKAROUNDS.md` until `video-studio` scenario ships.
- **Contributor-onboarding**: specific entry points (start with scenario X, run these commands, expect this output).

## Dev-log narrative principles

These principles extend the *Voice* and *Voice samples* sections above with structural requirements specifically for dev logs and series content. Added 2026-04-27 after vision walk #4 surfaced that voice canon alone was not catching narrative-shape failures.

### 1. Essay-shape per post
Every dev log is one essay split across the chosen format. **Required structure: hook → introduction → body → conclusion.** A thread is not a list of atomic tweets; a blog post is not a bulleted change-log. Each post must finish with a reason to return — what's coming next, where to find prior posts, how to follow.

### 2. Intro on first mention
Before referring to any scenario, agent, named file, or internal concept by name, check `shared/published-scenario-mentions.jsonl` (filtered to the target audience). If the subject has not been mentioned before in published material, the post must introduce it: one sentence covering what it is, why it exists, what it does at a high level. After first mention, subsequent posts may use a one-line refresher (e.g., "swarm-manager — the agent-orchestration substrate") instead of a full intro. **First-ever dev-log carries an outsized intro burden** because every concept is new to the audience; budget for it.

### 3. Personal voice grounded in builder identity
The *Voice* section above prescribes "first person, conversational, technically credible." Embodiment matters — a sentence that names an agent in third person is *not yet* personal voice. The first post of any series must especially feel like a software engineer talking about what they built — real grounded excitement tied to real work. The "I" can be the operator's or an agent's; pick deliberately, but pick.

### 4. What → why framing
Every change shown carries its own reason-to-care. "`resolve.go` is 52 lines + 202 test lines" is *what*. "`resolve.go` lets an initiative dispatch work to whichever agent the scenario expects, instead of hard-coding the wire — so adding a new scenario no longer means editing the dispatcher" is *why*. The why connects to broader narrative: last post's setup, next post's payoff, the project's vision. Show only changes that actually matter; do not narrate every commit. Use `shared/published-improvements-log.jsonl` to track what's already been narrated per scenario, so the next post advances the story rather than repeating it.

### 5. Hook-vs-body length asymmetry
X allows long posts now (with show-more gating); platform char limits no longer require uniform brevity. The first / hook tweet should be short to grab attention (around 280 chars on X). Body tweets carry the substance and may be longer when needed. Do not strip detail from body tweets to fit a uniform cap. Track lengths but apply the cap position-aware.

### 6. Inter-post linkage
Each post in a series connects to its predecessor. Mechanism: after publishing, the operator pastes the post URL back into `shared/publish-log.jsonl` (`post_url` field). The next post in the series cites that URL as `previous_post_url` (in the final reply, or as a visible link). Readers landing on the new post can find the prior chain. For the first post in a series, `previous_post_url` is null and the conclusion invites readers to follow for future posts.

### 7. No internal numbering externally
Internal artifacts (`p8`, `round-002`, `milestone-3`, batch ids) are operational vocabulary — they do not belong in published copy. The only sequential numbering visible externally is the dev-log post's own `post_index_in_series` (e.g., "post #1 in this dev-log series"), which signals to readers that other posts exist. If internal pass numbers leak externally, the audience cannot place them in any narrative they have access to.

### 8. Subject familiarity matters
A hook that works for an audience already familiar with the subject is *not* the same hook that works for an audience meeting it for the first time. Anti-pattern: hook assumes "swarm-manager" or "initiative-agents" is shared vocabulary on first publish. Pattern: hook either introduces the subject before the click-through, or hangs the click-through on a more universal frame (a problem, a question, a story) that the introduction follows. *Sample 1 below assumes the audience knows what swarm-manager is — appropriate for a subsequent post, not the first.*

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

Modes 9-12 were added 2026-04-27 via accepted framework-update `dec-1777300532504756717`. See [`Dev-log narrative principles`](#dev-log-narrative-principles) above for the underlying canon.

## Cross-references

- `docs/monetization/STRATEGY.md` — monetization's canonical positioning principles. The subscription framing above is the marketing-team's restatement of that; they must remain consistent.
- `AUDIENCES.md` — who we're talking to.
- `CAMPAIGNS.md` — what's currently in flight.
- `CHANNELS.md` — where and how we publish.
- `BRAND.md` — visual identity navigation hub.
- `ASSETS.md` — canonical brand asset registry (logos, fonts, OG image).
- `IMAGE_STYLE.md` — AI image generation style guide (palette, aesthetic, prompt directives).
- `../narrative/` — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline). Voice canon (this file) is the linguistic *how*; narrative is the *what*. Pull narrative from there; pull voice from here.

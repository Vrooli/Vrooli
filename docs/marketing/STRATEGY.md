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

- **Dev logs** (OSS-advertiser via `x-dev-log`): 3-7 tweets per thread, hook first, specific sourced claims. Work-in-progress explicitly labeled.
- **Feature announcements** (subscription or OSS, depending on framing): shipped features only. Imminent-release features carry "launching [date]."
- **Blog posts**: 500-2000 words, technical depth welcome, code snippets fine, end with a concrete invitation (try it, read more, contribute).
- **Videos**: demos or architecture walkthroughs; production workaround tracked in `notebook/VIDEO_WORKAROUNDS.md` until `video-studio` scenario ships.
- **Contributor-onboarding**: specific entry points (start with scenario X, run these commands, expect this output).

## Anti-patterns

These are the voice, positioning, and process failures marketing-contrarian scores against. Each corresponds to one of the eight failure modes in `TEAM.md`.

1. **Hype drift.** Overpromising unshipped features; "soon" without a committed date; claims not verifiable in `docs/monetization/catalog/base/*.md`.
2. **Voice drift.** Corporate-marketer language patterns replacing builder voice.
3. **Hallucinated engagement metrics.** Numbers without honesty flags (`measured / estimate / aspirational / pending-telemetry`).
4. **Paywall framing.** Subscription described as gating core features.
5. **OSS-as-leak framing.** Self-host described as lost revenue or fallback.
6. **Coverage-gap ignorance.** New campaigns while deployed SKUs have stale/missing coverage.
7. **Acquisition-only hypothesis.** Proposal names acquisition mechanism only; no retention side, no explicit `awareness-only: true`.
8. **Capability-workaround without gap.** Manual workaround with no matching `capability-gap` decision and no notebook entry.

## Cross-references

- `docs/monetization/STRATEGY.md` — monetization's canonical positioning principles. The subscription framing above is the marketing-team's restatement of that; they must remain consistent.
- `AUDIENCES.md` — who we're talking to.
- `CAMPAIGNS.md` — what's currently in flight.
- `CHANNELS.md` — where and how we publish.
- `BRAND.md` — visual and voice specifics (thin until brand-manager scenario ships).

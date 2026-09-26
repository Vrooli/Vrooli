# Strategy — Vrooli Marketing

This file is the voice, positioning, and framing canon for marketing-crew. Every draft, campaign, and public-facing artifact is checked against this before release.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents do not edit directly.

## Voice

Builder, not marketer.

- **First person, conversational, technically credible.** "Watched the agent retry three times before it got the cached response right." Not "we're excited to announce..." The "I" can be the operator's or an agent's — pick deliberately (see *Agents as protagonists* below).
- **Agents as protagonists.** Vrooli's differentiator is that agents themselves build the system. Most dev-log narration should make that visible — name the specific agent (`run-introspector`, `producer`, `toolchain-validator`, etc.), let it be the subject of the sentence, and credit the work to it. "team-agent-optimizer noticed the false-positive pattern and proposed the gate" lands harder than "I noticed it and added the gate." Operator authorship still happens — for vision, strategic decisions, and direct critique — but agent authorship is the unique signal no SaaS competitor can copy. Eventually drafts may carry explicit agent bylines.
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

After this post ships, future posts can use Sample 1's shorter "swarm-manager just landed initiative-agents…" hook, because the audience now knows what swarm-manager and initiative-agents are. Track the boundary with `shared/content-desk subject-familiarity records`.

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

Per-content-type strategic canon lives under [`../catalogs/post-types/`](../catalogs/post-types/) — one file per type. Each file names purpose, audience, conversion goal, structure, asset requirements, and contrarian failure modes. Currently authored:

- [`../catalogs/post-types/text/dev-log.md`](../catalogs/post-types/text/dev-log.md) — project-wide progress narrative; `producer` via `x-dev-log` skill.
- [`../catalogs/post-types/text/scenario-spotlight.md`](../catalogs/post-types/text/scenario-spotlight.md) — pitching one scenario as an end-user tool/app/product; subscription lane (`producer`) via `x-scenario-spotlight` skill.
- `../catalogs/post-types/text/oss-framework.md` — planned (pitching Vrooli as a developer platform).
- `path:../catalogs/post-types/image/` and `path:../catalogs/post-types/video/` — additional post-type entries by medium. See [`../catalogs/post-types/README.md`](../catalogs/post-types/README.md) for the decision tree and current coverage.

Other content types not yet broken into per-entity files:

- **Feature announcements** (subscription or OSS, depending on framing): shipped features only. Imminent-release features carry "launching [date]."
- **Blog posts**: 500-2000 words, technical depth welcome, code snippets fine, end with a concrete invitation (try it, read more, contribute). Apply the same [essay-shape](../methods/post-techniques/essay-shape.md) and [intro-on-first-mention](../methods/post-techniques/intro-on-first-mention.md) techniques as dev logs.
- **Videos**: demos or architecture walkthroughs; recurring manual production workarounds require typed marketing-craft observations or `capability-work` decisions until `video-studio` scenario ships.
- **Contributor-onboarding**: specific entry points (start with scenario X, run these commands, expect this output).

## Cross-cutting post techniques

Voice / structural rules that apply across multiple post types live under [`../methods/post-techniques/`](../methods/post-techniques/) — one file per technique, referenced from each post-type file that uses it. Currently authored (extracted 2026-04-28 walk #5 divergence #3 from this file's prior `Dev-log narrative principles` section, originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`):

- [`../methods/post-techniques/essay-shape.md`](../methods/post-techniques/essay-shape.md) — hook → introduction → body → conclusion structure.
- [`../methods/post-techniques/hook-vs-body-asymmetry.md`](../methods/post-techniques/hook-vs-body-asymmetry.md) — short hook, long-as-needed body.
- [`../methods/post-techniques/intro-on-first-mention.md`](../methods/post-techniques/intro-on-first-mention.md) — `content-desk subject-familiarity records` lookup discipline (with subject-familiarity corollary for hook calibration).
- [`../methods/post-techniques/inter-post-linkage.md`](../methods/post-techniques/inter-post-linkage.md) — series posts link to prior posts via `content-desk publish-history records`.
- [`../methods/post-techniques/no-internal-numbering-externally.md`](../methods/post-techniques/no-internal-numbering-externally.md) — `p8` / `round-002` / batch ids never appear in published copy.

Type-specific applications of voice canon (e.g., dev-log's "personal voice grounded in builder identity" and "what → why framing") live in the relevant post-type file rather than here, since they are not cross-cutting.

## Anti-patterns

These are the voice, positioning, process, and narrative-shape failures marketing-contrarian scores against. Each corresponds to one of the twelve failure modes in marketing-contrarian's `HEARTBEAT.md`.

1. **Hype drift.** Overpromising unshipped features; "soon" without a committed date; claims not verifiable in `docs/monetization/catalogs/skus/base/*.md`.
2. **Voice drift.** Corporate-marketer language patterns replacing builder voice.
3. **Hallucinated engagement metrics.** Numbers without honesty flags (`measured / estimate / aspirational / pending-telemetry`).
4. **Paywall framing.** Subscription described as gating core features.
5. **OSS-as-leak framing.** Self-host described as lost revenue or fallback.
6. **Coverage-gap ignorance.** New campaigns while deployed SKUs have stale/missing coverage.
7. **Acquisition-only hypothesis.** Proposal names acquisition mechanism only; no retention side, no explicit `awareness-only: true`.
8. **Capability-workaround without gap.** Manual workaround with no matching `capability-work` decision or typed observation.
9. **Narrative-flatness.** Draft reads as a changelog or atomic-tweet list rather than essay-shape (hook → introduction → body → conclusion). Distinct from voice-drift (mode 2) — that's word/phrase-level corporate-marketer language; this is structural shape.
10. **Internal-vocabulary-leakage.** Published copy uses internal artifact names (e.g. `p8`, `round-002`, internal codenames) without translation. Distinct from hype-drift (mode 1) — that's feature-claim overreach; this is vocabulary obscurity unrelated to claims.
11. **Missing-introduction-on-first-mention.** Draft refers to a scenario / agent / named file by name with no prior mention in `content-desk subject-familiarity records` for the target audience, AND no introduction in the draft itself.
12. **What-without-why.** Draft lists changes / line counts / commit refs without why-it-mattered framing tied to broader narrative.
13. **Persona-disclosure-violation.** Substantial AI-generated persona content posted without the platform's required AI-content label or without `#ad` for sponsorships. Scope: AI-UGC content per [`patterns/ai-ugc-personas.md`](patterns/ai-ugc-personas.md).
14. **Credential-claim-by-persona.** Persona claims a professional credential (doctor, therapist, lawyer, financial advisor, pharmacist, nurse practitioner, etc.) — explicit or implicit (e.g., professional-uniform visuals giving health advice). Hard reject; deception in regulated domains carries lawsuit risk.
15. **Real-person-impersonation.** Persona looks or sounds like a specific identifiable real person (celebrity, public figure, named competitor founder, anyone whose likeness is recognizable). Includes accidental drift; contrarian validates against a do-not-resemble registry per persona.
16. **Fabricated-real-customer-testimonial.** Post or persona claims to be a real Vrooli customer with a specific story when no such customer exists. Distinct from `recommendation-framing-without-basis` (mode 17 below; about *attribution*) — this is about *fabricating a customer*. Distinct from labeled AI-persona recommendation in persona's own voice (allowed under [`patterns/ai-ugc-personas.md`](patterns/ai-ugc-personas.md)).
17. **Recommendation-framing-without-basis.** Post uses third-party voice ("someone built this") without a real, identifiable third party. See [`../methods/post-techniques/recommendation-framing.md`](../methods/post-techniques/recommendation-framing.md). Distinct from mode 16 — this is attribution dishonesty in brand or operator voice; mode 16 is fabrication of a customer identity.
18. **Regulated-domain-advice-by-persona.** Persona gives advice in medical, financial, or legal domains regardless of explicit credential claim. Even with disclaimers, persona-voice content touching these domains is hazardous; redirect to brand-voice content or different topic.

Modes 9-12 were added 2026-04-27 via accepted framework-update `dec-1777300532504756717`. Modes 13-18 were added 2026-04-28 alongside the [`patterns/ai-ugc-personas.md`](patterns/ai-ugc-personas.md) authoring; the AI-UGC-specific modes (13-16, 18) draw their canonical rules from that strategy doc, and mode 17 consolidates the existing recommendation-framing-without-basis anti-pattern into the numbered framework. The underlying canon for modes 9-12 lived inline in this file as the *Dev-log narrative principles* section through walk #5; that section was extracted on 2026-04-28 (walk #5 divergence #3, Action B) into per-entity files under [`../methods/post-techniques/`](../methods/post-techniques/) and [`../catalogs/post-types/text/dev-log.md`](../catalogs/post-types/text/dev-log.md) (the `text/` subdirectory is the medium grouping introduced when post-types restructured by primary medium). The mode definitions in this list are unchanged; only their underlying canon's location moved.

For type-level specializations of these modes (e.g., scenario-spotlight's "demo theater" specializing mode 1, dev-log's "what-without-why" specializing mode 12 with type-specific mechanism), see the relevant `../catalogs/post-types/<type>.md` file. Per [`marketing-contrarian/RESPONSIBILITIES.md`](../../../scenarios/prompt-manager/store/teams/marketing-crew/members/marketing-contrarian/RESPONSIBILITIES.md), type-level specializations are applied alongside the framework-level twelve modes; they are not new framework modes.

## Source material discipline

Marketing-crew members regularly mine external content (viral posts, blog write-ups, competitor copy, video tutorials, course-funnel pitches) for structural-pattern alpha. This is high-value research input — *how* successful posts hook, structure, and convert is hard-earned knowledge, and most of it lives in production marketing the team did not produce. But external content also carries voice and tone that does not match Vrooli's `builder, not marketer` positioning, and uncritical mining leaks that voice into our drafts.

**The discipline:** mine the *structural pattern*, never the *tone*.

What's appropriate to extract from external content:

- ✅ Post structure (hook patterns, body shapes, conclusion mechanics).
- ✅ Audience-specific framings (what frame works for which persona).
- ✅ Conversion mechanics (call-to-action shapes, friction-reduction techniques, recommendation-framing patterns, competitive-comparison patterns).
- ✅ Asset patterns (when video helps, what kind of screen recording converts, brand-asset placement).
- ✅ Cross-platform amplification mechanics.

What is **not** appropriate to extract:

- ❌ Hyperbolic-marketer voice (course-funnel hype, "this changed my life," "you won't believe what happened next"), even when the structure works. Voice canon (above) is non-negotiable.
- ❌ Numbers as facts. Posts like "$15k/month, 5% close rate, 500 emails/day at 3% reply" are **upper-bound aspirational** and should be treated as hypothesis-generators, not benchmarks. Any number borrowed from external content lands as `feature_claims=overclaimed` until measured against our own data.
- ❌ Strategy claims at face value. "7,000 people are doing this and making $5,000-20,000/month" is unverified-third-party-claim — informative as a market-existence signal, not as a sizing input.
- ❌ Capability claims about competitors. Borrowed "X tool does Y" claims must be re-verified against the named tool's current docs before landing in any Vrooli post.

**Tagging convention:** when a marketing-crew member's knowledge entry or candidate-vertical-playbook draft references external content, the entry must carry an `unverified-third-party-claim` tag on every quantitative claim, an explicit citation (URL or post reference) for the source, and a freshness date (when was this read).

**Contrarian gate:** the marketing-contrarian validates that drafts using mined patterns:

1. Cite the structural source where the pattern came from.
2. Do not reproduce the source's voice / tone / hyperbolic framing.
3. Do not cite the source's numbers without independent measurement.
4. Apply our own honesty-flag schema (`feature_claims=measured | overclaimed | uncertain`, `data_source=verifiable | unverified-third-party-claim | aspirational`) regardless of how the source presented its claims.

A draft that fails any of these is rejected as `voice-drift` (mode 2) or `hype-drift` (mode 1) depending on the failure shape.

**Why this discipline exists:** at walk #5 the operator surfaced four viral marketing posts (Posts 1-4) as alpha sources. The structural patterns in them (recommendation framing, competitive comparison with multipliers, friction-reducing-hooks, tutorial-as-marketing) are genuinely useful and have been extracted to `../methods/post-techniques/` and `../catalogs/post-types/`. The numbers and tone in them are not appropriate for Vrooli copy. Codifying the discipline here ensures future research dumps are mined the same way without re-deriving the rule.

## Cross-references

- [`../catalogs/post-types/`](../catalogs/post-types/) — per-content-type strategic canon (one file per type). Currently: `dev-log.md`, `scenario-spotlight.md`. Planned: `oss-framework.md`, `use-case-tutorial.md`.
- [`../methods/post-techniques/`](../methods/post-techniques/) — cross-cutting voice and structure techniques (one file per technique). Currently: `essay-shape.md`, `hook-vs-body-asymmetry.md`, `intro-on-first-mention.md`, `inter-post-linkage.md`, `no-internal-numbering-externally.md`, `recommendation-framing.md`, `competitive-comparison.md`.
- `docs/monetization/strategy/STRATEGY.md` — monetization's canonical positioning principles. The subscription framing above is the marketing-team's restatement of that; they must remain consistent.
- `AUDIENCES.md` — who we're talking to.
- `CAMPAIGNS.md` — what's currently in flight.
- `CHANNELS.md` — where and how we publish.
- `BRAND.md` — visual identity navigation hub.
- `ASSETS.md` — canonical brand asset registry (logos, fonts, OG image).
- `IMAGE_STYLE.md` — AI image generation style guide (palette, aesthetic, prompt directives).
- `../../narrative/` — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline). Voice canon (this file) is the linguistic *how*; narrative is the *what*. Pull narrative from there; pull voice from here.

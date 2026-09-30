# Technique: Recommendation Framing

**Status:** v1. Authored 2026-04-28 (walk #5 divergence #4) consolidating prior references in `post-types/text/scenario-spotlight.md` (Recommendation variant) and `STRATEGY.md` (Anti-patterns: recommendation-framing-without-basis) into a canonical home.

## Rule

Use third-party voice ("someone built this thing that does X" / "here's a tool I came across" / "a friend recommended this") **only when there is a genuine third-party basis** — an external user, contributor, or operator-of-the-tool wrote about it; the scenario or feature was built by a contributor who is not the post author; the speaker is genuinely recommending someone else's work.

If the operator authored the thing being described and the post borrows third-party voice anyway, the framing reads slimy if discovered and erodes trust permanently. The contrarian rejects the draft.

## Scope: this technique vs. AI-UGC personas

This technique governs **third-party voice in posts produced by the brand or by the operator** — voice that *attributes* the recommendation to someone else. The third party must be real and identifiable; pseudonymous attribution is banned.

A separate, related practice — **AI-UGC personas** — covers content posted *from persona-actor accounts* that are openly labeled as AI-generated (per platform requirements). Persona-actors are not pseudonymous "real users" pretending to be human; they are transparently AI-generated personas with their own niche and content slate. Persona-voice recommendations from a labeled AI-UGC account are not subject to this technique's third-party-attribution rule, because the account itself is the speaker, not a fabricated stand-in. Persona-actor work is governed by [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md), which has its own rules: no professional-credential claims, no fabricated real-customer testimonials, no real-person impersonation, mandatory platform AI-content disclosure.

The distinction:

- **This technique** — brand or operator-authored post borrowing a third party's voice. Requires real, identifiable third party. Pseudonymous "someone said" is banned.
- **AI-UGC personas** — content posted from a labeled AI-persona account in the persona's own voice. Persona is not a stand-in for a fabricated real user; it's a transparently AI-generated voice with its own identity. Allowed under the rules in [`ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md).
- **Banned in both** — fabricated real-customer testimonial. A post or a persona claiming to be a specific real Vrooli customer with a specific story, when no such customer exists, is deception. This is distinct from a persona-actor recommending Vrooli in their own labeled-AI voice, which is allowed.

## Why it matters

Third-party-voice posts convert better than self-promotion. The reader's defensive filter for "someone is selling me something" doesn't engage as quickly when the post is framed as recommendation-of-someone-else's-thing rather than I-built-this. This is the empirical finding from observing viral marketing posts (e.g., the MapiLeads-recommendation post observed at walk #5 — "someone just built the most comprehensive B2B prospecting tool I've ever seen").

The conversion lift is real, but it's borrowed credibility. If the borrowing is genuine, the credibility transfers cleanly. If the borrowing is fabricated, the audience eventually notices (search "is X by Y?" → finds the operator authored it → trust collapses), and the lift inverts into long-term penalty.

The corresponding contrarian failure mode is **recommendation-framing-without-basis** (see `post-types/text/scenario-spotlight.md`'s contrarian failure-mode table). Distinct from voice-drift (corporate-marketer language) and capability-inflation (overclaiming features); this is specifically *attribution dishonesty*.

## When it applies

Genuine third-party basis includes:

- **External user or customer** wrote a post / review / case study about the scenario; we're amplifying or recommending it.
- **OSS contributor not the operator** built the scenario or feature; the post recommends *their* work.
- **Independent operator** of a Vrooli-hosted instance wrote about their setup; we're sharing their experience.
- **Trusted third-party publication / influencer** covered the work; we're quote-amplifying their coverage.

Fabricated or pseudo-bases that **do not** count:

- Operator wrote a fake recommendation post under a pseudonym presenting as a real human.
- A contributor's name is invoked but the post is operator-authored without their input.
- "Someone in our community" without a specific identifiable someone.
- Generic third-person ("people are saying...") without naming who.
- AI-persona-actor content where the persona claims to be a real Vrooli customer (this is the `fabricated-real-customer-testimonial` failure mode in [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md), distinct from persona-recommendation in persona's own labeled voice, which is allowed).

## How to apply it correctly

1. **Cite the source.** Link the original post / review / repo. The reader should be able to verify the third-party basis with one click.
2. **Use the third party's words where possible.** Quote-tweet, blockquote, screenshot. The recommendation is the third party's; the operator's job is to amplify, not ventriloquize.
3. **Disclose if a relationship exists.** If the third party is a contributor or partner, say so briefly. The credibility of recommendation framing depends on the audience's trust that the speaker is independent — disclosure protects that trust even when there's nominal independence.
4. **Apply the contrarian honesty flag.** A draft using this technique must carry `third_party_basis=genuine-third-party` (per `post-types/text/scenario-spotlight.md`'s honesty-flag schema). The contrarian must verify the basis before approval.

## What's allowed

- ✅ Recommending a contributor-built scenario in third-party voice with attribution and disclosure of contributor relationship.
- ✅ Quote-amplifying an external user's post about their experience with a Vrooli scenario.
- ✅ Posting a real review or case study with third-party voice, with the source linked.
- ✅ Sharing an independent operator's setup story in their own words.

## What's prohibited

- ❌ Operator-authored "someone built this" framing where the someone is the operator.
- ❌ Generic third-person attribution to no one in particular ("people are saying...", "I've heard that...").
- ❌ Pseudonymous or sock-puppet recommendation presenting as a real human.
- ❌ Recommendation framing without the contrarian's `third_party_basis=genuine-third-party` flag verified.
- ❌ Persona-actor content claiming to be a real Vrooli customer (allowed only as labeled-AI-persona recommendation per [`../../strategy/patterns/ai-ugc-personas.md`](../../strategy/patterns/ai-ugc-personas.md), which is a separate practice from this technique).

## Cross-references

- `STRATEGY.md` — voice canon; recommendation-framing-without-basis is a named anti-pattern.
- `post-types/text/scenario-spotlight.md` — Recommendation variant lists this technique as the conversion-rate-friendly framing; the contrarian failure-mode table includes the no-basis check.
- `post-techniques/competitive-comparison.md` — companion technique; recommendation framing of *someone else's tool framed against ours* (or vice versa) requires both techniques' rules to hold simultaneously.
- `post-techniques/intro-on-first-mention.md` — when the third party or their work is being introduced for the first time, that technique applies on top.
- `../../strategy/patterns/ai-ugc-personas.md` — separate practice for persona-actor content; persona recommendations in labeled-AI voice are governed by that doc, not by this technique.

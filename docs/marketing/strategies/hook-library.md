# Strategy: Hook Library

**Status:** v0 (skeleton — populated by researcher's `hook-candidate-promotion` decisions over time).

A curated library of proven hook patterns, tagged by platform / audience / outcome. Hooks are observed in the wild by `researcher`, captured in `audience-scans.jsonl` as `hook-candidate` scope entries, and promoted here when ≥3 stable observations support the hook + when a `hook-candidate-promotion` decision is approved.

This is a *library* (a list), not a *technique*. Hook *shape patterns* (essay-shape, hook-vs-body asymmetry) live in [`../post-techniques/`](../post-techniques/); specific *hook lines and templates* live here.

## How entries get here

Authoring discipline (per researcher's [`HEARTBEAT.md`](../../../scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/HEARTBEAT.md)):

1. Researcher captures `hook-candidate` scans — verbatim hooks observed in the wild, with platform + audience + source URL + freshness date.
2. After ≥3 stable hooks accumulate (or after a notable cluster), researcher proposes `hook-candidate-promotion`.
3. Operator approves; the operator (or brand-manager on their behalf) edits this file to add the hooks under the appropriate category.
4. Each entry retains its source citation per [STRATEGY.md's source-material discipline](../STRATEGY.md#source-material-discipline) — *mine the structural pattern, never the tone*.

Hooks the operator drafts directly are also accepted but should still be captured as scan entries first so the library remains traceable.

## Schema for each entry

```
- **Hook line / template:** "[verbatim or templated]"
  - **Platform(s):** X / TikTok / Reels / Shorts / etc.
  - **Audience:** [persona key from AUDIENCES.md]
  - **Post type(s):** [post-type slugs that this hook fits]
  - **Outcome observed:** [what happened in the wild — e.g., "high reply rate", "high save rate", "high watch-completion"]
  - **Honesty notes:** [does this hook require a specific verifiable claim? what fails if applied dishonestly?]
  - **Source:** [URL — original observation in researcher's audience-scans.jsonl or external reference]
  - **Captured:** YYYY-MM-DD
```

## Categories (populate as entries arrive)

### Friction-recognition openers

*"Have you ever ___ and ended up ___…"; openers that name a friction the audience has lived through. Strong on TikTok / Reels for problem-agitate-solve; strong on X for dev-log opens.*

(No entries yet.)

### Unexpected-contrast openers

*Two clauses that don't seem to belong together; the contrast hooks attention. Strong on X dev-log opens.*

(No entries yet.)

### Verified-multiplier hooks

*"X-times faster / cheaper / smaller than Y." Subject to the [hyperbolic-but-verifiable-multipliers sub-pattern](../post-techniques/competitive-comparison.md#sub-pattern-hyperbolic-but-verifiable-multipliers). Strong on comparison-reel and competitive scenario-spotlight.*

(No entries yet.)

### Friction-reducing-offer hooks

*"Here's what happens when ___" / "I stopped ___ and started ___." Strong on lifestyle-bundle TikTok and Reels.*

(No entries yet.)

### Specific-moment hooks

*A hook anchored to a concrete in-time moment ("This morning I noticed ___"). Strong on dev-log openers (Sample 1 and Sample 2 in [`../STRATEGY.md`](../STRATEGY.md) illustrate the shape).*

(No entries yet.)

### Story-cold-open hooks

*Persona starts mid-story; no intro. Strong on narrative-talking-head and day-in-life-ugc.*

(No entries yet.)

### Numbered-list cover hooks

*"5 ways I keep my house organized" / "7 dev tools that saved my week". Strong on slideshow-listicle and slideshow-tips-then-plug. Numbers between 5-10 perform; 3 reads thin, ≥11 loses.*

(No entries yet.)

## Anti-patterns (hooks the contrarian rejects)

- **Generic hype hooks** — "This will change your life", "You won't believe what happened next" — voice drift; auto-reject.
- **Multiplier hooks without citation** — "20x faster than X" without a one-click benchmark — `uncited-multiplier` failure mode.
- **Credential-implying hooks in persona voice** — "As a [profession], I ___" by an AI-persona that has no such credential — `credential-claim-by-persona` failure mode.
- **Fake-urgency hooks** — "Last chance to ___", "Only available today" — manufactured scarcity reads as low-trust marketing.
- **Curiosity-gap hooks without delivery** — hook promises a payoff the body doesn't deliver — burns audience trust.

## Cross-references

- [`../post-techniques/essay-shape.md`](../post-techniques/essay-shape.md) — where hooks fit in the post structure.
- [`../post-techniques/hook-vs-body-asymmetry.md`](../post-techniques/hook-vs-body-asymmetry.md) — hook-length discipline.
- [`../STRATEGY.md`](../STRATEGY.md) — voice canon (banned phrases that auto-disqualify hooks).
- Researcher's [`HEARTBEAT.md`](../../../scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/HEARTBEAT.md) — `hook-candidate` scan scope and `hook-candidate-promotion` decision context.

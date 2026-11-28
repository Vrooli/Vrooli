# Technique: No Internal Numbering Externally

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §7 on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

Internal artifacts (`p8`, `round-002`, `milestone-3`, batch ids) are operational vocabulary — they do not belong in published copy.

The only sequential numbering visible externally is the dev-log post's own `post_index_in_series` (e.g., "post #1 in this dev-log series"), which signals to readers that other posts exist. If internal pass numbers leak externally, the audience cannot place them in any narrative they have access to.

## Why it matters

Internal numbering is a tracking convenience for the operator and for swarm-manager: it correlates work to a specific pass / round / milestone in internal records. Externally, the numbers are noise — the audience has no `p7` to remember, no sense of how many `round-`s a workshop has, no model of how `milestone-3` relates to `milestone-2`. Leaking these numbers to readers signals two things, both bad: (a) the writer didn't translate from internal to external vocabulary; (b) the post is for-the-team-not-for-the-reader.

The corresponding contrarian failure mode is **internal-vocabulary-leakage** (mode 10 in `STRATEGY.md`'s Anti-patterns). Distinct from hype-drift (mode 1), which is feature-claim overreach; this is vocabulary obscurity unrelated to claims.

## What counts as internal numbering

- Pass identifiers: `p8`, `p18`, `p1.5`, `pass-3`.
- Round identifiers: `round-002`, `workshop-r4`, `feedback-round-3`.
- Milestone identifiers: `milestone-3`, `m2`, `phase-1.2`.
- Batch identifiers: any UUID-shaped or operator-generated batch tag.
- Internal version numbers tied to internal release cadence (different from public semver, which is fine).
- Initiative or backlog item names that include numeric suffixes used internally as ordering (`agent-sandbox-audit-foundation` is fine; `agent-sandbox-p3` is not).

## What's allowed externally

- `post_index_in_series` (e.g., "post #1 in this dev-log series") — the only sanctioned external numbering. Tells the reader other posts exist; orients them in the chain.
- Public version numbers / public semver releases.
- Real dates, real timestamps, real public commit hashes.
- Real commit / PR / issue counts ("6 commits 2026-04-26 → 2026-04-27") — these are evidence, not internal numbering.

## When it applies

- ✅ All published copy across all post types — `dev-log`, `scenario-spotlight`, `oss-framework` (planned), blog posts, LinkedIn, etc.
- ✅ Asset overlays — text in screen recordings or screenshots is published copy and must be sanitized too.
- ✅ Captions, alt text, social cards — same rule.
- ➖ Internal artifacts (handoffs, knowledge entries, retrospectives) — internal numbering is appropriate internally.

## Cross-references

- `STRATEGY.md` — Anti-patterns mode 10 (internal-vocabulary-leakage).
- `post-techniques/intro-on-first-mention.md` — companion rule about *named* internal vocabulary (scenario / agent / file names) that may be appropriate externally with introduction; this rule covers numerical vocabulary that's never appropriate externally regardless.
- `CHANNELS.md` — sanitization rules per platform; `x-dev-log` guardrails specifically reference path / email / credential sanitization, of which this is a kin.

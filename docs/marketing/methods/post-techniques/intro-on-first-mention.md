# Technique: Intro on First Mention

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §2 (and §8 — subject-familiarity, which is the corollary about hook calibration) on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

Before referring to any scenario, agent, named file, or internal concept by name, check `content-desk subject-familiarity records` (filtered to the target audience). If the subject has not been mentioned before in published material to that audience, the post must introduce it: one sentence covering what it is, why it exists, what it does at a high level. After first mention, subsequent posts may use a one-line refresher (e.g., "swarm-manager — the agent-orchestration substrate") instead of a full intro.

**First-ever post in any series carries an outsized intro burden** because every concept is new to the audience; budget for it.

### Subject-familiarity corollary (formerly §8)

A hook that works for an audience already familiar with the subject is *not* the same hook that works for an audience meeting it for the first time.

- **Anti-pattern:** hook assumes a name (e.g., "swarm-manager," "initiative-agents") is shared vocabulary on first publish.
- **Pattern:** hook either introduces the subject before the click-through, **or** hangs the click-through on a more universal frame (a problem, a question, a story) that the introduction follows.

`STRATEGY.md`'s Voice sample 1 ("swarm-manager just landed initiative-agents…") assumes the audience knows what swarm-manager is — appropriate for a subsequent post, not the first. Sample 5 illustrates the first-post version that introduces both Vrooli and swarm-manager before relying on either name.

## Why it matters

Internal vocabulary leakage and assumed-familiarity hooks are the two most common ways a technically-correct post lands flat with a new audience. The mechanism — `content-desk subject-familiarity records` filtered by audience — makes the check programmatic rather than memory-dependent. Skipping the check means the agent can produce a post that reads coherently to the operator (who knows the vocabulary) but is opaque to the reader.

The corresponding contrarian failure mode is **missing-introduction-on-first-mention** (mode 11 in `STRATEGY.md`'s Anti-patterns).

## When it applies

- ✅ `dev-log` — primary use case; every post checks.
- ✅ `scenario-spotlight` — every spotlight introduces the scenario unless `content-desk subject-familiarity records` shows a prior mention to the target audience persona.
- ✅ `oss-framework` (planned) — applies once authored.
- ✅ Blog posts and LinkedIn posts — same rule; same lookup.
- ➖ Internal-only artifacts (handoffs, knowledge entries) — not subject to this rule; internal vocabulary is fine internally.

## Mechanism

1. **Lookup.** Before drafting, query `content-desk subject-familiarity records` filtered by the target audience persona.
2. **Decision.**
   - First mention → full intro: one sentence covering *what / why / high-level what-it-does*.
   - Subsequent mention → one-line refresher acceptable; full intro optional.
3. **Hook calibration.** Choose hook shape based on familiarity:
   - Subject already familiar to audience → name-first hook (Sample 1 shape) is fine.
   - Subject new to audience → universal-frame hook (Sample 5 shape) followed by introduction in the body.
4. **After publish.** Append the publish event to `content-desk subject-familiarity records` so the next post sees the updated familiarity state.

## Cross-references

- `STRATEGY.md` — Voice sample 5 (first-ever dev-log) demonstrates the first-mention burden in concrete form.
- `STRATEGY.md` — Anti-patterns mode 11 (missing-introduction-on-first-mention).
- `post-techniques/no-internal-numbering-externally.md` — companion rule about internal vocabulary that's never appropriate externally regardless of familiarity.
- `post-techniques/inter-post-linkage.md` — companion rule about how series posts cite prior posts; downstream of this rule's familiarity tracking.

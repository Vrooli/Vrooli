# Technique: Essay-Shape Per Post

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §1 on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

Every post is one essay split across the chosen format. **Required structure: hook → introduction → body → conclusion.**

A thread is not a list of atomic tweets; a blog post is not a bulleted change-log. Each post must finish with a reason to return — what's coming next, where to find prior posts, how to follow.

## Why it matters

Atomic-tweet shape and changelog shape both fail readers in the same way: there is no narrative arc to retain attention from one element to the next. The hook earns the click; the introduction earns the second tweet; the body delivers what the introduction promised; the conclusion gives a reason to come back. Skipping any piece — especially the introduction or the conclusion — drops engagement.

The corresponding contrarian failure mode is **narrative-flatness** (mode 9 in `STRATEGY.md`'s Anti-patterns). Distinct from voice-drift (mode 2), which is word/phrase-level corporate-marketer language; this one is structural shape.

## When it applies

- ✅ `dev-log` — every dev-log post.
- ✅ `scenario-spotlight` — every spotlight post; the body is asset-led but still wrapped in essay shape.
- ✅ `oss-framework` (planned) — applies once authored.
- ✅ Blog posts of any type.
- ➖ Atomic single-tweet announcements (rare; usually a coverage-gap signal that a real post is needed instead).

## Hook patterns

The hook is the highest-leverage line in the post — it earns the click. A few recurring patterns work especially well; this is not an exhaustive list, just the ones we've observed converting:

- **Friction recognition.** "If you've ever ___ and ended up ___…" Names a specific friction the audience has felt. Used heavily in `scenario-spotlight` and any problem-led post.
- **Unexpected contrast.** "X just did Y — here's what happened next." Sets up a narrative arc the body delivers.
- **Verified multiplier.** "20x faster than [named alternative]." Only when the multiplier is measured, cited, and scope-honest — see [`competitive-comparison.md`](competitive-comparison.md) for the discipline.
- **Friction-reducing offer.** "Free, zero coding, just watch and copy." Drops the activation cost to near-zero. Used in tutorial-as-marketing and OSS-adoption framings (e.g., the agent-dev-team video course observed at walk #5). The friction-reducing hook converts strongly when the offer is *real* — copy that promises "zero coding" and then asks the reader to install seven dependencies erodes trust faster than no hook at all. Verify the offer matches the actual user experience before using.
- **Specific moment.** "Watched the agent retry three times before it got the cached response right." Anchors the post in a concrete event rather than an abstract claim.

The hook is intentionally short. The body delivers what the hook promised — see the companion [`hook-vs-body-asymmetry.md`](hook-vs-body-asymmetry.md) for the length-distribution discipline.

## Cross-references

- `STRATEGY.md` — Voice and Voice samples (especially Sample 5 — first-ever post applies essay-shape end-to-end).
- `STRATEGY.md` — Anti-patterns mode 9 (narrative-flatness): the failure-mode framing this technique prevents.
- `post-techniques/hook-vs-body-asymmetry.md` — companion technique on length distribution within the essay shape.
- `post-techniques/inter-post-linkage.md` — companion technique on how a post's conclusion links to the next post.

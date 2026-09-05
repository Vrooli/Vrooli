# Technique: Hook-vs-Body Length Asymmetry

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §5 on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

X allows long posts now (with show-more gating); platform char limits no longer require uniform brevity. The first / hook tweet should be short to grab attention (around 280 chars on X). Body tweets carry the substance and may be longer when needed. Do not strip detail from body tweets to fit a uniform cap. Track lengths but apply the cap position-aware.

## Why it matters

Pre-2024 X enforced a uniform 280-char cap, forcing writers to chop substance into atomic-tweet shape. That constraint is gone. Writers (and agents) who still apply uniform caps are leaving substance on the floor — *especially* in body tweets where a longer post earns the show-more click and rewards the reader with concrete detail. Hook stays short because attention is short at the hook position; body grows because attention has been earned by the time the reader is reading the body.

This technique also reinforces the [essay-shape](essay-shape.md) discipline by giving the body permission to be long enough to *be* a body rather than another hook.

## When it applies

- ✅ `dev-log` on X — primary use case. Body tweets routinely 400-600+ chars.
- ✅ `scenario-spotlight` on X — same principle; the body wraps the asset.
- ✅ `oss-framework` (planned) — applies once authored.
- ➖ LinkedIn / blog — these formats don't have the same per-element cap. Asymmetry is more about pacing than length there.
- ❌ Platforms with hard uniform per-post caps (e.g., Mastodon if instance-configured low) — work within the platform's actual limits.

## Practical guidance

- Hook tweet: aim ~280 chars on X. Punchy.
- Body tweets: write what the substance requires. If the reader has clicked through, they have committed to reading further.
- Track per-tweet character counts in the draft contract (e.g., `[210/204/228/233/166/200]`). Counts that scale up across the thread tend to indicate proper asymmetry; counts that stay flat at ~280 may indicate over-trimming.

## Cross-references

- `post-techniques/essay-shape.md` — the structural rule this technique works within.
- `STRATEGY.md` — Voice samples 1-5 demonstrate the asymmetry in shipped form.

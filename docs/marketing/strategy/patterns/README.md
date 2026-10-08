# Marketing Strategies — Plan of Record

This folder is the **strategic canon** for cross-cutting marketing strategies that don't fit cleanly into a single post-type, post-technique, or channel. One file per strategy.

These docs answer: *what is this strategic approach, when does it apply, when does it backfire, what discipline does the marketing-contrarian enforce against it?*

They do **not** answer: *here is the prompt; here is the round structure; here are the data sources to mine.* That belongs in skills under [`path:scenarios/prompt-manager/store/skills/packs/core/`](path:scenarios/prompt-manager/store/skills/packs/core/).

## Why a separate folder for strategies

Some marketing knowledge is neither tied to one post-type (so it doesn't belong under [`post-types/`](../../catalogs/post-types/)) nor a cross-cutting voice/structure rule (so it doesn't belong under [`post-techniques/`](../../methods/post-techniques/)). Examples: *AI-UGC personas* (a whole class of marketing approach with its own rules), *hook library* (a curated list of hooks, not itself a technique), *funnel patterns* (multi-step flows from post to product).

These get a third home: `strategies/`. Each strategy file is its own entity, scoped tight, referenced from whichever post-type or skill applies it.

## Files in this folder

| File | Status | Applies to |
|------|--------|-----------|
| [`ai-ugc-personas.md`](ai-ugc-personas.md) | v1 | Persona-actor accounts on TikTok, Instagram Reels, YouTube Shorts; any post-type produced in persona voice |
| [`hook-library.md`](hook-library.md) | v0 (skeleton — populated by the producer's `hook-candidate-promotion` decisions) | All post types as a reference library |
| [`funnel-patterns.md`](funnel-patterns.md) | v0 (skeleton — populated as the producer captures conversion paths) | All campaigns; cross-references with `funnel-builder` skill once telemetry exists |

## Write rules

Same as the rest of `path:docs/marketing/`: agents never write directly; operator-curated via approved decisions.

- `ai-ugc-personas.md` — `brand-guideline-update` work type.
- `hook-library.md` — `hook-candidate-promotion` work type (producer proposes; operator approves).
- `funnel-patterns.md` — `brand-guideline-update`.

## Cross-references

- [`../../catalogs/post-types/`](../../catalogs/post-types/) — strategies often inform multiple post-types; each post-type references the strategies it depends on.
- [`../../methods/post-techniques/`](../../methods/post-techniques/) — voice and structure techniques. A strategy may invoke multiple techniques.
- [`../STRATEGY.md`](../STRATEGY.md) — voice canon and anti-patterns. Strategies must not contradict it.
- [`../CHANNELS.md`](../CHANNELS.md) — channel rules, including AI-UGC disclosure expectations per platform.
- [`../../catalogs/rich-media/`](../../catalogs/rich-media/) — schemas for character/scene/product data that AI-UGC content depends on.

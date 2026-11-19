# Handoff — extraction candidates (System pane)

Status: **analysis + section complete, coordinator wiring pending.** `StudioExtract` is
built and tested in isolation; `StudioSystem.tsx` is untouched on purpose. The section
answers one deterministic question: *which repeated custom structure in this frame is
worth turning into a shared component?* It does not guess, score semantics, or invent a
percentage.

## Files

| File | Change |
|---|---|
| `ui/src/workbench/studio/extract.ts` | New pure module: shape signatures, candidate thresholds, nested-dedup, ranking |
| `ui/src/workbench/studio/extract.test.ts` | 9 unit tests (grouping, thresholds, exclusions, dedup, determinism, cap) |
| `ui/src/workbench/studio/StudioExtract.tsx` | New compact System-pane section; **Select example** calls `onSelect` |
| `ui/src/workbench/studio/StudioExtract.a11y.test.tsx` | 2 axe tests (populated list, honest empty state) |

No other file was touched. Existing `studio-*` / `wb-*` classes only; no new CSS, no
locale JSON edits. The coordinator runs `pnpm strings:catalog && pnpm strings:gen`.

## The signature

Shape, not copy. For one subtree:

```
type[propKey1,propKey2]{childSignature|childSignature|…}
```

- Prop **keys** are sorted and contribute only their names, never their values, so
  `pad: 16` and `pad: 24` match while `pad` vs `gap` do not.
- Child signatures keep document order: title-above-body is not body-above-title.
- Content and placement props never enter the shape: `id`, `text`, `value`, `label`,
  `hint`, `placeholder`, `alt`, `src`, `x`, `y`, `w`, `h`.
- Example: two pricing surfaces that differ only in headline copy, ids and padding
  produce `surface[pad]{text[]{…}|text[]{…}}` twice and group into one candidate.

Excluded roots (the node itself never becomes a candidate):

- every type in `RCL_PRIMITIVES` (`avatar`, `button`, `card`, `chip`, `input`, `meter`,
  `stat`) — already a shared asset;
- `page` and `slot` — structural.

Only the **root** decides. A custom `row` repeated inside two cards is still a
candidate, because extracting it is a real option; its id becomes the example.

## Thresholds

- `nodeCount >= 3` (the subtree, including its root). A repeated two-layer structure is
  filtered out as not worth a component.
- `occurrences >= 2`. A single instance is never a candidate.
- At most **5** candidates returned.

## Dedup (largest wins)

Candidates are considered by `nodeCount` descending. A candidate is dropped when every
one of its occurrences sits **at or inside** an occurrence of an already accepted
candidate: the larger structure already tells the same story, so the nested copy is
noise.

The brief described this as "drop a candidate whose example ids are a subset of an
accepted candidate's example ids". Root ids alone cannot express nesting — nested copies
have different ids — so the implementation walks the parent chain instead. The behaviour
is the same for the intended case (an identical inner block inside every copy of a larger
block is dropped) and it keeps the larger candidate. An inner structure that also repeats
outside the larger one keeps its own entry, because that is a second story.

## Ranking

1. `score` descending, where `score = (occurrences - 1) * nodeCount` — the rough
   duplication cost;
2. `nodeCount` descending;
3. `label` ascending.

`exampleIds` stays in document order, so `exampleIds[0]` is the first occurrence in the
frame.

## Integration (coordinator)

`StudioSystem` already receives `activeFrame: DesignFrame` and
`onSelectNode: (id: string) => void`. `StudioExtract`'s root is a
`<section className="studio-system-group">`, so it drops straight into the existing
`.studio-system` column.

Inside `<div className="studio-system">`, after the **Patterns** section and before the
**Pages** section (i.e. around the existing chevron at `StudioSystem.tsx:237`), add:

```tsx
import { StudioExtract } from "./StudioExtract";

// …

<div className="studio-system-arrow" aria-hidden>
  <ChevronRight size={12} />
</div>
<StudioExtract spec={activeFrame.spec} onSelect={onSelectNode} />
```

Do not pass `frames` or `coverage`: the analysis is deliberately single-frame and does
not blend layer share with extraction. `ChevronRight` is already imported in that file.

### Strings

New keys (created by `pnpm strings:catalog`, not hand-written):

- `workbench.studio.extractionCandidates`
- `workbench.studio.extractionCount`
- `workbench.studio.extractionExplain`
- `workbench.studio.extractionEmpty`
- `workbench.studio.extractionEmptyHint`
- `workbench.studio.extractionScore`
- `workbench.studio.extractionSelect`
- `workbench.studio.extractionSelectLabel`
- `workbench.studio.extractionUsage`

### Section contents

- Heading **Extraction candidates** + `{{count}} found`.
- One-line explanation: repeated custom structures of the same shape, ranked by
  duplication cost.
- Each entry: label (`surface · Starter + For one seat`), `cost {{score}}` chip,
  `×{{occurrences}} · {{count}} layers`, and a **Select example** button whose
  accessible name is `Select example of {{label}}`.
- Empty state: **No repeated custom structure in this frame.** plus **RCL primitives,
  pages and slots are excluded.**

## Validation

- `pnpm exec vitest run src/workbench/studio/extract.test.ts src/workbench/studio/StudioExtract.a11y.test.tsx`
  → 2 files / 11 tests green.
- `pnpm exec tsc --noEmit` → clean (new keys type-check through the `defaultValue`
  overload; they are not in `en.json` yet).
- `pnpm exec eslint` on the four files → clean.
- Not run (per brief): `pnpm build`, scenario restart, commit.

## Honest limits

- **One frame at a time.** Repeated structure across frames or designs is not visible
  yet; `PLAN.md` next-wave item 2 calls this the honest first version.
- **Shape, not semantics.** Two unrelated blocks with the same node types and prop keys
  group. That is the point — it is deterministic and inspectable — but it cannot tell a
  pricing card from a testimonial card.
- **Prop values are ignored.** A card with a different `pad` still matches; if padding
  should distinguish variants, that is a later, explicit rule.
- **Labels use the example's copy.** The signature is stable across copy changes; the
  human label is not (it quotes the first occurrence's text). Ranking tie-breaks can
  therefore move when copy changes, though membership does not.
- **No cross-design history, paint weight, or RCL impact estimate.** The score is layer
  count × extra copies, not saved bytes or minutes. No percentage is produced anywhere.
- **Fixed thresholds.** `3` layers / `2` occurrences / `5` results are constants in
  `extract.ts`, not settings.
- **Interpretation of "exclude"**: only the candidate *root* is excluded, so a custom
  structure nested inside an RCL primitive can still surface.

## Coordinator integration (landed)

- `StudioSystem` renders `<StudioExtract spec={activeFrame.spec} onSelect={onSelectNode} />` after
  the Pages section, before the coverage foot.
- The candidate label now prefers the authored layer role (`Group · Signals`) and strips the part
  after the first `·` before appending child text, so an imported card does not read
  `Group · Signals · Signals`. A regression test covers it.
- Layout fix: the list/item have `min-width:0` and the label ellipsizes; the "Select example"
  action stays inside the 245 px pane. Evidence: `studio-extract.png`.
- Live: the imported Northwind page produced one candidate, `Group · Signals + Every change gets a
  trail.` ×2, 3 layers, cost 3; selecting it selects `Group · Signals`.

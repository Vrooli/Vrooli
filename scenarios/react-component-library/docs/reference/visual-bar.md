# React Component Library visual bar

This bar is the visual acceptance standard for assets rendered by the preview
workbench. A screenshot is evidence only when it names the asset, viewport,
theme, story, and capture path. Passing a code or contract gate does not imply
that an asset passes this bar.

## Checks

- Layout: the asset uses the available stage width for its archetype, has no unintended clipping or overflow, and keeps related content grouped at every declared viewport.
- Layout: page, shell, frame, and pattern assets are not constrained by the primitive 760 px specimen well.
- Type scale: headings, labels, body text, and supporting text form a readable hierarchy with no truncation at the smallest declared viewport.
- Color: foreground, surface, border, focus, status, and interactive colors remain distinct in both themes and preserve semantic meaning without color alone.
- Control sizing: every interactive target is at least 44 CSS px on its touch axis, has a visible focus treatment, and keeps icon and label alignment stable.
- State: the declared default, loading, success, error, empty, disabled, and live interaction states are visually distinguishable and do not shift unrelated content.
- Motion: transitions are purposeful, bounded, and absent or reduced when the reduced-motion preference is active.
- Accessibility: landmarks, headings, labels, status announcements, focus order, keyboard operation, and contrast are observable in the rendered capture and accessibility tree.
- Responsiveness: content remains usable at 390x844, 834x1112, 1440x900, and 2560x1440 without a second page-level scrollbar.
- Theme parity: light and dark captures preserve hierarchy, affordance, contrast, and state meaning without relying on an accidental system theme.

## Capture matrix

Every inventory asset is captured at every combination below. The filename
format is `<viewport>-<theme>.jpg`.

| Viewport | CSS size | Themes |
| --- | --- | --- |
| mobile | 390x844 | light, dark |
| tablet | 834x1112 | light, dark |
| wide | 1440x900 | light, dark |
| ultrawide | 2560x1440 | light, dark |

The capture instrument uses the scenario shorthand URL and waits for the
preview page to settle. Theme is carried as an explicit URL parameter so the
capture record cannot silently claim that the browser's system preference was
the requested theme. If the running host does not apply that parameter, the
capture is marked `theme-unverified` in the verdict ledger and is not a pass.

## Verdict vocabulary

- `pass`: every applicable check is visibly satisfied and the capture path is present.
- `repair`: one or more checks fail; the asset must be repaired and recaptured.
- `theme-unverified`: the artifact exists but the requested theme was not applied; it cannot close the bar.
- `not-rendered`: the asset or route did not produce a usable specimen.

Verdicts are generated in the local `coverage/visual/verdicts.json` ledger.
The entire scenario `coverage/` tree is ignored because it contains generated
test and screenshot artifacts; it is not part of the component source or a
portable checked-in fixture. The baseline set is immutable within a capture
run, and final captures use the same matrix and asset names so local
before-and-after comparisons remain valid. Durable conclusions belong in a
report or plan log, with the local ledger treated as supporting evidence.

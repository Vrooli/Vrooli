# React Component Library visual bar

This bar is the visual acceptance standard for assets rendered by the preview
workbench. A screenshot is evidence only when it names the asset, viewport,
theme, story, and capture path. Passing a code or contract gate does not imply
that an asset passes this bar.

## Checks

| Check                                                                                                                                                                          | Class      | Evaluator / evidence rule                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ------------------------------------------------------------------------------------------------------------------------- |
| Layout: the asset uses the available stage width for its archetype, has no unintended clipping or overflow, and keeps related content grouped at every declared viewport.      | computed   | `content-not-clipped` compares client and scroll dimensions plus reachable overflow in the accessibility capture evidence |
| Layout: page, shell, frame, and pattern assets are not constrained by the primitive 760 px specimen well.                                                                      | unmeasured | No deterministic evaluator is registered yet                                                                              |
| Type scale: headings, labels, body text, and supporting text form a readable hierarchy with no truncation at the smallest declared viewport.                                   | unmeasured | Conformance measure planned; no credit until calibrated                                                                   |
| Color: foreground, surface, border, focus, status, and interactive colors remain distinct in both themes and preserve semantic meaning without color alone.                    | unmeasured | Capture evidence is attested until a computed contrast evaluator is calibrated                                            |
| Control sizing: every interactive target is at least 44 CSS px on its touch axis, has a visible focus treatment, and keeps icon and label alignment stable.                    | unmeasured | `tap-target-size` exists but is not yet calibrated for this bar                                                           |
| State: the declared default, loading, success, error, empty, disabled, and live interaction states are visually distinguishable and do not shift unrelated content.            | unmeasured | No deterministic state-diff evaluator is registered yet                                                                   |
| Motion: transitions are purposeful, bounded, and absent or reduced when the reduced-motion preference is active.                                                               | unmeasured | No deterministic motion evaluator is registered yet                                                                       |
| Accessibility: landmarks, headings, labels, status announcements, focus order, keyboard operation, and contrast are observable in the rendered capture and accessibility tree. | unmeasured | Existing accessibility claims remain separate from this bar until calibrated                                              |
| Responsiveness: content remains usable at 390x844, 834x1112, 1440x900, and 2560x1440 without a second page-level scrollbar.                                                    | unmeasured | Existing viewport claims remain separate from this bar until calibrated                                                   |
| Theme parity: light and dark captures preserve hierarchy, affordance, contrast, and state meaning without relying on an accidental system theme.                               | unmeasured | Theme application is checked; parity remains unmeasured                                                                   |

## Capture matrix

Every inventory asset is captured at every combination below. The filename
format is `<viewport>-<theme>.jpg`.

| Viewport  | CSS size  | Themes      |
| --------- | --------- | ----------- |
| mobile    | 390x844   | light, dark |
| tablet    | 834x1112  | light, dark |
| wide      | 1440x900  | light, dark |
| ultrawide | 2560x1440 | light, dark |

The capture instrument uses the scenario shorthand URL and waits for the
preview page to settle. Theme is carried as an explicit URL parameter so the
capture record cannot silently claim that the browser's system preference was
the requested theme. If the running host does not apply that parameter, the
capture is marked `theme-unverified` in the verdict ledger and is not a pass.
Animation and caret blinking are disabled, fixture data is seeded, and the
capture uses the pinned font set. These controls are part of reproducibility,
not visual polish.

The screenshot boundary is the isolated harness document's
`[data-preview-sheet]` element. The Playwright runner must screenshot that
locator exactly once per story/theme/viewport combination. A screenshot of the
Components workspace, its toolbar, or the iframe's full `#root` is diagnostic
only and cannot satisfy the visual bar. Frame-backed stories retain their
declared frame in the isolated capture, so the artifact proves the subject in
its intended composition context.

When several stories are selected, the runner may create one labeled contact
sheet for up to four stories. It first validates each story's isolated sheet,
then composes those rendered sheets into a single review artifact. The contact
sheet is evidence only when its manifest names every included story and records
the bounded sheet size.

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

## Retention and reproducibility contract

`scenarios/react-component-library/coverage/` is a local cache, not a
repository artifact. The `react-component-library coverage prune` command
enforces a maximum age of 14 days and a maximum total cache size of 2 GiB.
It always prints the files selected for deletion before applying deletion;
`--dry-run` is the default. `verdicts.json`, evaluator outputs, content hashes,
calibration verdicts, and timing records are never pruned. If the cache cannot
be reproduced with the same asset version, viewport, theme, kit, seeded data,
font set, and determinism controls, the verdict is invalid and the capture
instrument must be repaired.

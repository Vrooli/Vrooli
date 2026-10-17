# Component library gaps

Command Center is an ambient board with a zero-idle-chrome cycle rail, board
controls, and a full-bleed drawing surface. `AmbientDisplayShell` is the
closest library archetype, but it is pre-1.0 and cannot yet carry the board's
cycle controller and input-specific controls without changing the product's
ambient behavior. The scenario remains intentionally ejected until that
archetype is proven.

Two fit primitives are scenario-local for the same reason: `AutoScroll` (a
tall list that holds, steps row by row and returns to the top) and
`SupportingStrip` (measure, tighten, then page in place) both hold the board's
beat through `useBeatHold`, which is `CycleController` state. They are library
candidates once `CycleController` is; the geometry in `ui/src/lib/fit.ts` is
already pure and would move with them.

```shell-ejection
{"archetype":"ambient-display","reason":"Command Center is a full-bleed ambient board whose cycle rail, input-revealed controls, and drawing surface are scenario-owned. AmbientDisplayShell is pre-1.0 (0.1.2) and cannot carry the board controller without changing the product behavior; retain the working board and revisit after the archetype is proven.","files":["ui/src/components/AmbientShell.tsx","ui/src/components/AmbientDisplayShell.tsx","ui/src/components/AmbientCanvas.tsx","ui/src/components/BoardController.tsx"]}
```

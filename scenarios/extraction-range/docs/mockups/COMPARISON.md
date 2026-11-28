# Workbench comparison evidence

Current design target: [`workbench-target.svg`](workbench-target.svg), authored 2026-11-12 for refinement `range-workbench-concept-2026-11-12`.

## Build and behavior evidence

| Check | Result | Evidence |
|---|---|---|
| UI TypeScript and string catalog | Pass | `cd scenarios/extraction-range/ui && pnpm type-check` |
| Production UI build | Pass | `cd scenarios/extraction-range/ui && pnpm build` |
| Focused workbench and shell behavior | Pass | UI Vitest execution: 131 tests across 27 files; includes named-tool submission and scoreboard refresh. |
| Experience spec contract and capture join | Pass | `experience-manager spec validate extraction-range --json` reported `PASSED`, with no findings after the scenario owner restarted the updated UI/API proxy. |
| Desktop/phone capture | Pass | BAS captures are retained as PNGs in this directory; execution IDs: desktop `9763f98a-3ca3-43d8-a436-3686664dd0dd`, phone `84a462e7-7514-4b6b-92e7-705b0f2f0a37`, phone landscape `32e061db-c834-4781-b2e3-67f9f209d360`. Each capture reached `data-experience-state="ready"`. |
| API and browser console | Pass | Browser rendered 40 challenges, 10 families and 46 saved scores; `curl http://127.0.0.1:21976/levels` returned `200 application/json`. BAS console artifact reported no console output. |

## Visual comparison ledger

| Surface / state / viewport | Source/build | Reference and expected difference | Operated evidence | Remaining defect / disposition |
|---|---|---|---|---|
| Workbench, ready / desktop | UI build emitted `App-C5kkcyTr.js` and `index-BdPk9ugn.css` on 2026-11-12 | Target: simultaneous catalog, task form and scoreboard. Implementation uses the live 40-level catalog and durable score rows in place of the SVG's illustrative data. | [Desktop capture](captures/desktop-ready.png), BAS `9763f98a-3ca3-43d8-a436-3686664dd0dd`; challenge, score submission and scoreboard refresh also pass the UI interaction test. | Composition matches the target; long level names are deliberately ellipsized in the catalog, with full title and instructions visible on selection. |
| Workbench, ready / phone portrait | Same UI build | Target: challenge list first, drill into task and score, then switch to scoreboard. Implementation has separate phone-stage components using the same data and filters. | [Phone capture](captures/phone-ready.png), BAS `84a462e7-7514-4b6b-92e7-705b0f2f0a37`; the catalog and full capability remain reachable by tap. | Composition matches the target; long list labels truncate by design and remain fully readable in the selected challenge detail. |
| Workbench, ready / phone landscape (844 × 390) | Same UI build | Target is sequential below the width needed for desktop's three columns. Breakpoint now retains the mobile stage through 1279px. | [Phone-landscape capture](captures/phone-landscape-ready.png), BAS `32e061db-c834-4781-b2e3-67f9f209d360`; no horizontal document overflow finding in the experience gate. | Expected sequence preserved; no unresolved overflow. |
| Workbench, loading / initial request error / stale refresh | Same UI source | Contract binds one required `AsyncPanel` lifecycle region with loading, ready, empty, partial and error states. | `experience-manager spec validate extraction-range --json`: `PASSED`, zero findings; normal submit path is covered by the full UI test suite. | Request-error and retained-data states are implemented; no separate live failure injection was performed. |

The target SVG is an authored composition wireframe, not a product screenshot or operator ratification. The desktop and phone captures were compared against it; agent-authored refinements for available screen width and long-title containment are adopted. No image generation is used. The UI reaches L4 visual verification for these captured states; human concept/mockup ratification remains unrecorded, so the overall experiential maturity is reported as L3 pending those two human checkpoints.

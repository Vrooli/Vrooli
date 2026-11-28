# Experience Design

## Product concept

Extraction Range is a **field workbench for crawler builders**. Its main job is to let an operator move from a known challenge to a measured extractor result without losing track of which tool produced it. The interface uses familiar workbench structure: a challenge catalog, a focused task bench, and a durable results rail. Challenge IDs and measurements use compact technical typography; the owned target opens separately so the operator can inspect it while keeping the scorer ready.

The product is compared with a browser based QA and benchmark console: the useful bar is a searchable/scan-friendly test catalog, unambiguous task instructions, honest score definitions, and a result table that keeps tool attribution. It should feel like a focused engineering tool rather than a general health dashboard.

## Primary surface: Extraction Workbench (`/`)

### Shared content model

- **Challenge**: stable ID, family, difficulty, task, tags and owned target URL.
- **Submission**: tool/version, selected challenge ID, and a JSON array of extracted records.
- **Score**: correct/submitted/expected counts, precision, recall, F1, pass status and the tool identity.
- **Scoreboard**: one durable best result for each tool/challenge pair, with family filter and explicit empty state.
- **Page states**: initial loading and error/retry; filtered-empty catalog; empty scoreboard; partial refresh failure retaining loaded data; validation and submission error; submission in progress; scored result.

### Desktop composition

The primary page is a three-part work surface: a narrow challenge list at left, the active task and submission form in the broad center, and saved scores at right. All three remain visible so tool iteration does not require navigation. The selected target opens in a new tab. A family selector filters both catalog and scoreboard.

### Phone composition

Phone is sequential, not a squeezed split pane. A two-choice view switch moves between the challenge catalog and scoreboard; selecting a challenge drills into a full-width task and submission screen with a clear return action. The catalog is a dense list of tappable challenge rows. Tool identity, task text, payload editor, immediate score feedback and all score details remain available. Inputs use a 16px baseline and the submission action is a full-width touch target.

### Shell and visual system

The existing Vrooli Operational Console kit remains the source of tokens and primitives. This scenario composes a **range instrument panel** from those neutral work surfaces: blue selection/action, semantic green pass status, fine dividers, stable score numerals and deliberately quiet metadata. No decorative illustration is needed; the product's own challenge identifiers, score measures and three-part workbench provide the visual identity. A procedural visual asset is not used.

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | Existing operational console tokens preserve accessibility and system themes. |
| `density` | sidebar | Desktop needs simultaneous catalog, task and result context. |
| `mobileNav` | tabs | Two task-specific views preserve all functions without a table overflow. |
| `mainMode` | scroll | Catalog and scoreboard can grow; their own lists scroll without trapping the page. |

## Information architecture

| Surface | Route | Question answered |
|---|---|---|
| Extraction Workbench | `/` | Which challenge should I run, what does it require, and how did my tool score? |
| Notes example | `/notes` | What notes are stored by the generated example domain? |
| Settings | `/settings` | Which appearance and language preferences are active? |

## Accountable outcomes

1. A tool builder can select any of the catalog's ten families and see the task and owned practice target.
2. A tool builder can submit a named tool's result array, see an exact score, and recover from validation or network errors without losing the input.
3. A reviewer can compare saved best results across tools and families and see whether each result passed.

## Decisions and refinements

| ID | Source / author / date | Surfaces and states | Change and preserved behavior | Rationale / tradeoff | Evidence and disposition |
|---|---|---|---|---|---|
| `range-workbench-concept-2026-11-12` | F10 owner brief / agent-authored / 2026-11-12 | `/`, all catalog/submission/score states | Replace generic health-and-placeholder home with challenge workbench; preserve health endpoint and settings/notes routes elsewhere. | Makes the approved extraction task the first-useful action; allocates desktop width across simultaneous task regions and changes phone to sequential drill-in. | Current target recorded in `docs/mockups/`; code and validation evidence recorded in `docs/mockups/COMPARISON.md`. Adopted for this implementation; human concept/mockup ratification remains unrecorded. |

## Current maturity and evidence

- **L3 — concept + assets applied.** The workbench concept drives layout and vocabulary on the primary route. The interface uses the procedural range instrument panel (semantic status, dividers, typography and responsive composition), not raster art.
- Desktop, phone portrait and phone landscape captures were compared with the retained mockup; the machine experience contract validates cleanly. Overall maturity remains L3 because the skill's concept and mockup ratifications are human checkpoints and are not recorded yet.
- Machine-readable priorities, states, selectors and bindings live in `experience/pages/dashboard.json`.

## Cross-references

- `path:scenarios/extraction-range/experience/README.md` — contract authoring and validation.
- `path:scenarios/extraction-range/docs/guides/choosing-ui.md` — experience contract design.
- `path:scenarios/extraction-range/DESIGN.md` — shared design token and interaction contract.
- `path:scenarios/extraction-range/docs/mockups/` — composition target and comparison evidence.

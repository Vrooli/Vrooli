# Component fixtures and In Context implementation

Date: 2026-11-25  
Scope: React Component Library component page (`/assets/:id`), Fixtures and In Context tabs.

## Outcome

- **Fixtures** is a role-grouped specimen bench. Frames are measured and drawn at
  their real size, grouped by anatomy/axis/boundary, and exact visual duplicates
  are collapsed behind a disclosure.
- **In Context** reads typed slots from the seeded templates, ranks candidates,
  distinguishes exact/family/accepted roles, and keeps component fit separate
  from page health.
- **Context scenes** ship as Scene · Toolbar, Scene · Card, and Scene · Dialog
  footer.

Visible frame counts after duplicate collapse:

| Asset | Stories | Visible | Hidden redundant |
|---|---:|---:|---:|
| IconButton 3.1.8 | 17 | 14 | 3 |
| Card 1.3.1 | 7 | 4 | 3 |

Typed placement in Aperture:

| Asset | Auto slot | Score | Match |
|---|---|---:|---|
| IconButton | Section action | 87% | related region |
| Card | Metric or card | 84% | accepted by kind (warned) |

## Debt removed

- Deleted `FixtureCanvas`, its test, and `useFixturePositions`.
- Deleted `neutralizeSlots`; `TemplateRenderer` selects the active slot directly.
- Removed the fixed 320 × 220 white fixture frame and the generic `role: hero`
  accepts-everything fallback.
- Replaced the obsolete drag-to-arrange copy.
- Seed shelf is 19 curated workspaces at seed version 48; every state and page
  fallback spec has typed slots with authored fallback content.

## Validation

| Check | Result |
|---|---|
| UI tests (`cd ui && pnpm test`) | 180 files, 1,195 tests passed |
| UI lint (`cd ui && pnpm lint`) | passed |
| UI build (`cd ui && pnpm build`) | passed |
| Go (`cd api && go test -count=1 ./internal/workspaces/... ./handlers/preview/... ./internal/components/...`) | passed |
| Playwright 1440 × 1000 | bench and typed context captured |
| Playwright 390 × 844 | document width equals viewport for both tabs on both assets |

## Evidence location

Full visual evidence, the decision packet, and the implementation record are in
the experience-engineering operator experiment:

`~/.vrooli/research/topics/experience-engineering/operator-experiments/rcl-component-fixtures-context-20261125/`

The captured images and the self-contained HTML packet are intentionally not
tracked in this repository.

## Limits

The Aperture page still overflows at 390 px; that is a page-health finding, now
grouped separately from component fit. The Targets and Contrast instruments read
the page outside the component iframe. A scene is a compact section on a full
device page, not a separate renderer. The captures are layout evidence, not BAS
acceptance evidence.

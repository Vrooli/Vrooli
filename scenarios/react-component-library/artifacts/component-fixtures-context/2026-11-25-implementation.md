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

## Quiet check model adopted

The approved quiet pass replaced the split between experience claims in
Fixtures and eight live checks in In Context with one model:

- **Fixtures** shows a compact component-fit block for the selected frame:
  measured size, touch targets, text contrast, a type footnote and an evidence
  footnote. It links to In Context for page health.
- **In Context** leads with the same fit rows. Page, evidence and human checks
  are one `Page & evidence` disclosure, opened only when needed.
- The mode bar now says `Fit …` with a page-issue count; the old
  `{{passed}}/{{total}} claims pass` chip and the inspector percentage ring are
  gone.
- Per-row provenance labels (`measured`, `declared`, `n/a`) and tags
  (`declared`, `sample`, `page`) are gone. Type is informational, not scored.
- Evidence shows capture records, the last capture date and a Run evidence
  action. It no longer counts skipped captures as state coverage.
- Component-fit rows use the harness probe for control sizes and component
  text contrast; page health still measures the page.

## Debt removed

- Deleted `FixtureCanvas`, its test, and `useFixturePositions`.
- Deleted `neutralizeSlots`; `TemplateRenderer` selects the active slot directly.
- Removed the fixed 320 × 220 white fixture frame and the generic `role: hero`
  accepts-everything fallback.
- Replaced the obsolete drag-to-arrange copy.
- Seed shelf is 19 curated workspaces at seed version 48; every state and page
  fallback spec has typed slots with authored fallback content.

## Truthfulness follow-up

The first pass looked calm but still trusted data that could not move:

- The harness reported size, touch and type once, before the page theme or a
  props override. A surface toggle or edit left the old reading on screen. The
  harness now reports a measurement-only message after render and on request;
  `FixtureFrame` asks again after theme application.
- `measureComponentProbe` accepted visually hidden text (`Saving…` inside
  `rcl-visually-hidden`), so an icon-only story reported Arial 13.33px and
  widened its painted size to 55 × 40. Hidden nodes are excluded; the same
  story now measures 40 × 40 with no readable text.
- The In Context slot row used the placement profile (`140 × 44` for every
  IconButton) instead of the selected story. It now uses the measured paint,
  so Small is 36 × 36, Large 44 × 44 and Extra small 32 × 32. The placement
  explanation uses the same measured width.
- Experience status was transport reachability (`available`), while the check
  looked for a `fresh` value no producer emitted. Complete state coverage could
  never pass. The reader now derives `covered`, `partial`, `unmeasured`,
  `unavailable` or `not-configured` from declared states and passing captures;
  the UI uses one status helper in the header, drawer and checks.
- Fixtures and In Context each reproduced the touch/contrast/type thresholds.
  One set of row builders now owns those verdicts, including the icon-only
  "no readable text" case.
- Removed write-only check metadata (`ContextCheckRow.evidence`), the duplicate
  `ContextVerdict` type, dead summary claim counts, dead instrument helpers, and
  four `InContextReport` fields no production consumer read.

## Validation

| Check | Result |
|---|---|
| UI tests (`cd ui && pnpm test`) | 183 files, 1,215 tests passed |
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

The Aperture page still overflows at 390 px; that is a page-health finding,
grouped separately from component fit. The Targets and Contrast overlays remain
page instruments; the check rows use the component probe. A scene is a compact
section on a full device page, not a separate renderer. The captures are layout
evidence, not BAS acceptance evidence.

# Experience Audit

## Last Updated
2026-07-30

## Persona Map

| Persona | Primary Goal | Key Flows |
|---------|-------------|-----------|
| Developer building desktop app | Generate Electron wrapper for their scenario | Scenario selection → Template config → Generate → Build → Download |
| Agent automating desktop builds | Trigger desktop generation via API/CLI | CLI `generate` command or `POST /api/v1/desktop/generate` |
| Admin signing releases | Configure code signing for distribution | Signing page → Certificate setup → Signed build |

## View Structure

App uses `useUrlState` for routing with three view modes in [CODE: ui/src/App.tsx]:

| View | Purpose | Entry Point |
|------|---------|-------------|
| `inventory` | Browse available scenarios | Default landing view |
| `generator` | Multi-stage build pipeline UI | Select scenario from inventory |
| `signing` | Code signing configuration | Navigation action |

## Key UI Components

| Component | Purpose | Location |
|-----------|---------|----------|
| `ScenarioInventory` | Scenario browser with status badges | [CODE: ui/src/components/scenario-inventory/ScenarioInventory.tsx] |
| `GeneratorPage` | Multi-section pipeline: Preflight → Generate → Build → SmokeTest → Deploy | [CODE: ui/src/components/generator/ConnectionSectionRouter.tsx] |
| `SigningPage` | Code signing certificate management | [CODE: ui/src/components/signing/SigningPage.tsx] |
| `DocsPanel` | Integrated documentation sidebar | [CODE: ui/src/components/docs/DocsPanel.tsx] |
| `LiveDesktopDrawer` | VNC viewer for testing generated apps | [CODE: ui/src/components/livedesktop/LiveDesktopDrawer.tsx] |
| `ErrorBoundary` | Granular error recovery per section | [CODE: ui/src/components/ui/ErrorBoundary.tsx] |

## Friction Points

| Friction | Severity | Persona Affected | Notes |
|----------|----------|-------------------|-------|
| Pipeline stages are sequential with no skip option | Low | Developer | By design — each stage depends on prior output |
| Build failures can still require reading detailed logs | Low | Developer | Structured API recovery hints are combined into the UI error message; the pipeline error display still exposes raw diagnostics for advanced investigation |
| Code signing setup requires external certificates | Medium | Admin | P1 roadmap item — manual process today |
| No visual diff between template types before selection | Low | Developer | Template summaries shown; no live preview |

## Accessibility

| Pattern | Status | Notes |
|---------|--------|-------|
| `aria-*` attributes on interactive elements | Partial | Drawers, popovers, buttons have ARIA labels |
| Keyboard navigation | Partial | `tabIndex` management on buttons; drawers support Escape to close |
| Semantic HTML | Good | Proper heading hierarchy, form labels |
| Color contrast | Good | Dark theme with high-contrast text |
| Screen reader support | Improved | Component descriptions remain present; the fixed pipeline-status badge now uses `aria-live="polite"` so build transitions are announced |

## Recommendations

1. **Improve structured recovery presentation** — recovery hints now reach the user-facing error message; retain a dedicated, visually distinct remediation panel as a future UX refinement.
2. **Keep pipeline announcements concise** — the current polite live region announces status changes; test with screen-reader users before adding more announcements.
3. **Template preview** — show a visual preview or side-by-side comparison of template types during selection.

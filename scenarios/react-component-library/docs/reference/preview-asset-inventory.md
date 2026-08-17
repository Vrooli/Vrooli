# Preview asset inventory

This inventory is the bounded set of library assets directly composed by the
preview workbench source under `ui/src/features/components/**` and
`ui/src/pages/**`. Source annotations are the provenance input. An annotation
that does not resolve to a catalog asset remains listed so the repair phase
cannot silently omit it.

| Asset | Catalog status | Archetype | Source files |
| --- | --- | --- | --- |
| `data-display.code-block` | exists | primitive | `ui/src/features/components/ComponentEditorSource.tsx` |
| `data-display.data-table` | exists | primitive | `ui/src/features/components/ComponentsCard.tsx`, `ui/src/pages/CoveragePage.tsx` |
| `data-display.description-list` | exists | primitive | `ui/src/features/components/PropsExperimentPanel.tsx` |
| `data-display.tree-view` | exists | primitive | `ui/src/features/components/AdoptionFileTree.tsx` |
| `forms.input` | exists | primitive | `ui/src/features/components/IngestComponentForm.tsx` |
| `forms.select` | exists | primitive | `ui/src/features/components/ThemeSwitcher.tsx` |
| `navigation.page` | exists | shell | `ui/src/pages/ComponentDetailPage.tsx` |
| `overlays.dialog` | exists | overlay | `ui/src/features/components/CreateComponentDialog.tsx`, `ui/src/features/components/ComponentEditorTools.tsx` |
| `overlays.inspector-panel` | exists | overlay | `ui/src/features/components/InspectorPanel.tsx` |
| `overlays.popover` | exists | overlay | `ui/src/features/components/AnchoredMenu.tsx` |
| `manipulation.split-pane` | exists | pattern | `ui/src/features/components/ComponentEditorStage.tsx` |
| `navigation.master-detail` | exists | shell | `ui/src/features/components/ComponentEditor.tsx`, `ui/src/features/components/ComponentEditorController.tsx` |
| `templates.dashboard-page` | exists | page | `ui/src/pages/DashboardPage.tsx` |
| `preview.preview-dock` | exists | pattern | `ui/src/features/components/ComponentEditorController.tsx` |
| `preview.story-palette` | exists | pattern | `ui/src/features/components/ComponentEditorController.tsx` |
| `preview.inspector-drawer` | exists | overlay | `ui/src/features/components/ComponentEditorStage.tsx` |
| `preview.canvas-frame` | exists | pattern | `ui/src/features/components/ComponentEditorStage.tsx` |

## Annotation audit

The Phase 7 provenance sweep resolved the workbench annotations to released
catalog assets (`StatusBadge`, `ResponsivePanel`, `EmptyState`, `InspectorLayout`,
`PageFrame`, `DashboardPage`, `Page`, `SplitPane`, and `MasterDetail`). The
preview-specific library surfaces are indexed with versioned story contracts;
their current workbench compositions remain in the controller/stage while
adoption is staged. No unresolved annotation remains in the workbench source
set.

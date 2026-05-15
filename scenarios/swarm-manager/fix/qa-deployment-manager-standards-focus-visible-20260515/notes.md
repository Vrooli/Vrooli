## Problem

Deployment-manager readiness is red/yellow on standards. Fresh GCT job `a01fe0cf-1bd2-4002-b1d4-8f5130577caa` completed on 2026-05-15 and reported `standards.available=true`, `blockingViolations=0`, `warnings=100`, `totalViolations=100`. The visible top violations are all `Missing focus-visible styles`.

## Top Violations

1. `scenarios/deployment-manager/ui/src/App.tsx` - Missing focus-visible styles - add `focus-visible:ring-2 focus-visible:outline-none` to interactive elements, or use `[data-spatial-focus]` styling.
2. `scenarios/deployment-manager/ui/src/components/DependencyGraph.test.tsx` - Missing focus-visible styles - same recommendation.
3. `scenarios/deployment-manager/ui/src/components/DeploymentMonitor.test.tsx` - Missing focus-visible styles - same recommendation.
4. `scenarios/deployment-manager/ui/src/components/FitnessScoreBreakdown.test.tsx` - Missing focus-visible styles - same recommendation.
5. `scenarios/deployment-manager/ui/src/components/GuidedFlow.tsx` - Missing focus-visible styles - same recommendation.
6. `scenarios/deployment-manager/ui/src/components/LPBSReleaseConfigCard.test.tsx` - Missing focus-visible styles - same recommendation.
7. `scenarios/deployment-manager/ui/src/components/LPBSReleaseConfigCard.tsx` - Missing focus-visible styles - same recommendation.
8. `scenarios/deployment-manager/ui/src/components/Layout.test.tsx` - Missing focus-visible styles - same recommendation.
9. `scenarios/deployment-manager/ui/src/components/Layout.tsx` - Missing focus-visible styles - same recommendation.
10. `scenarios/deployment-manager/ui/src/components/ReleasesPanel.test.tsx` - Missing focus-visible styles - same recommendation.

## Impact

Deployment-manager governs release and approval operations. Missing keyboard/spatial focus indication makes those workflows harder to operate from keyboard, remote, kiosk, and iframe-hosted surfaces, and keeps QA readiness from clearing before downstream desktop-release work executes.

## Reproduction

Run:

```sh
git-control-tower review run deployment-manager --details=10 --json
```

Observed evidence: job `a01fe0cf-1bd2-4002-b1d4-8f5130577caa`, readiness `red`, standards warnings `100`, top violations all `Missing focus-visible styles`.

## Success Criteria

- Standards warnings for focus-visible styling are reduced from 100 to 0, or every remaining finding is a separately justified non-focus finding.
- Interactive deployment-manager controls expose visible focus state through `focus-visible:*` classes or `[data-spatial-focus]` styling.
- GCT standards dimension is green for deployment-manager.

## Proposed Action

1. Audit shared interactive primitives first: buttons, links, cards, menu items, tabs, selects, and custom clickable panels.
2. Add focus-visible and spatial-focus styling at the shared component layer where possible.
3. Fix remaining one-off interactive elements in the listed files.
4. Re-run GCT readiness and confirm standards warnings are cleared.

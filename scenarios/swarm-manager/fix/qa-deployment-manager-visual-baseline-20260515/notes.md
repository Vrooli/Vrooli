## Problem

Deployment-manager readiness is red on the visual dimension. Fresh GCT job `a01fe0cf-1bd2-4002-b1d4-8f5130577caa` and follow-up summary both reported `visual.available=true`, `screenshotCount=0`, `stale=false`. No current visual capture exists for deployment-manager, so GCT cannot prove the UI renders correctly.

## Top Violations

1. `scenarios/deployment-manager/ui/src/App.tsx` - no current visual capture exists for the app shell.
2. `scenarios/deployment-manager/ui/src/components/Layout.tsx` - high-traffic layout surface lacks visual readiness evidence.
3. `scenarios/deployment-manager/ui/src/components/GuidedFlow.tsx` - release workflow surface lacks visual readiness evidence.
4. `scenarios/deployment-manager/ui/src/components/DeploymentMonitor.tsx` - monitoring surface lacks visual readiness evidence.
5. `scenarios/deployment-manager/ui/src/components/ReleasesPanel.tsx` - release panel lacks visual readiness evidence.

## Impact

Deployment-manager release workflows are user-facing and operationally sensitive. Without at least one current screenshot capture, QA cannot catch blank rendering, overlap, broken iframe bridge layout, or viewport regressions before release-governance work builds on this surface.

## Reproduction

Run:

```sh
git-control-tower review run deployment-manager --details=10 --json
```

Observed evidence: job `a01fe0cf-1bd2-4002-b1d4-8f5130577caa`, readiness `red`, visual dimension `screenshotCount=0`.

## Success Criteria

- GCT review summary for deployment-manager reports visual green.
- `screenshotCount` is greater than 0.
- `latestCapture` is present and non-stale.
- Captures include at least the main deployment-manager app shell and one release workflow view at representative desktop viewport size.

## Proposed Action

1. Ensure deployment-manager UI can be launched under the review/capture harness.
2. Add or refresh BAS/browser visual capture coverage for the main app shell and release workflow panels.
3. Verify captures are stored where GCT visual readiness discovers them.
4. Re-run GCT readiness and confirm visual turns green.

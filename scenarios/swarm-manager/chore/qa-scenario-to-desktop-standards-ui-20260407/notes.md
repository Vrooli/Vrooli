# QA Recommendation

targetScenario: scenario-to-desktop
problemOrOpportunity: Standards warnings for UI conventions (drawer keydown listener and missing INTEROP-CRITICAL comments in vite config).
proposedAction:
- Move keydown listener from `ui/src/components/ui/drawer.tsx` into a dedicated hook under `ui/src/hooks/`.
- Add required `// INTEROP-CRITICAL` comments in `ui/vite.config.ts` above interop-sensitive settings.
- Re-run standards checks to confirm warnings cleared.
evidence: evidence/gct-review.json (GCT review summary)
riskLevel: low
executionModeHint: manual
createdByTeam: scenario-qa
sourceRunId: ba6373e2-b77d-42ed-b430-d7e8d5015cd5

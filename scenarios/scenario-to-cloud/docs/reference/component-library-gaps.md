# Scenario To Cloud component-library gaps

This product keeps its specialized workspace frame because the library shell does not
carry the simultaneous workflow regions described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Deployment wizard workspace owns dashboard, deployments, docs, wizard backtracking, health refresh, and a documentation sub-navigation. AppShell/2 cannot carry the wizard/documentation transition contract without moving product workflow controls into generic chrome.","files":["ui/src/components/layout/Layout.tsx","ui/src/components/docs/DocsSidebar.tsx","ui/src/hooks/useHashRouter.ts","ui/src/components/deployments/DeploymentsPage.tsx","ui/src/components/deployments/tabs/ConfirmationDialog.tsx","ui/src/components/deployments/tabs/SecretsTab.tsx","ui/src/components/wizard/InvestigationReport.tsx","ui/src/components/wizard/SpawnAgentButton.tsx","ui/src/components/wizard/StepPreflight.tsx"]}
```

## Deployment console (phase 19, 2026-09-09)

The console under `ui/src/components/deployments/console/` composes the
scenario's own primitives. Each shared asset that would have fit was
preflighted with `react-component-library adoptions preflight <asset>
scenario-to-cloud` before the local primitive was written; the verdicts are
recorded here so the library owner can close them.

| Need | Library asset | Preflight verdict (2026-09-09) | Decision |
|------|---------------|--------------------------------|----------|
| Destructive-action dialog with focus trap, Escape, focus return | `ResponsiveDialog` 1.3.1 | refused: `asset dependency react-component-library:GestureTokens@1.1.0 required by useOverlaySurface cannot be resolved` | local `DestructiveActionDialog` (own trap and focus return); the library's dependency graph is broken for this asset |
| Focus trap and focus return hooks | `useFocusTrap` 1.1.0, `useFocusReturn` 1.1.1 | `useFocusTrap` ok (i18n pass, selector pass); `useFocusReturn` `blocking: true`, maturity `scaffolded`, verdicts `not-measured` | not linked: linking adds `@vrooli/react-component-library` as a new governed package dependency for a 30-line trap, and the companion hook is blocked at the library's maturity floor |
| Status pills (health verdict, freshness, operation state) | `StatusBadge` 1.2.2, `StatusIndicator` 1.0.5 | 15 unsatisfied tokens each (`--border-hairline`, `--color-danger`, `--color-info`, `--color-muted-foreground`, `--color-success`, `--color-surface`, `--color-warning`, `--radius-pill`, `--space-*`, `--text-label*`); the token ramp in `ui/src/design-tokens.css` is empty | local `StatusPill` on the Tailwind palette; adopt once `adoptions tokens-sync` has a ramp to write into |
| Live announcements | `useAnnounce` 1.0.3, `LiveAnnouncer` 1.0.1 | `blocking: true`, maturity `scaffolded`, verdicts `not-measured` | local `LiveRegion` (`role="status"`, polite or assertive) |
| Reduced motion | `useReducedMotion` 1.0.1 | `blocking: true`, maturity `scaffolded` | CSS `motion-safe:animate-spin`; no JS needed |
| Definition rows, copy affordances, skeletons | `Skeleton` 1.0.6 (token-bound); no key/value or copy asset exists | not preflighted: token-bound or absent | local `KeyValue`, `CopyButton`, `SkeletonRows` |

Local forks retired by this phase: `ui/src/components/wizard/DeploymentProgress.tsx`
(percentage bar; deleted, replaced by the standing-driven `OperationPanel`)
and the redeploy modal with its `SwitchRow` and stepper composition inside
`DeploymentDetails.tsx` (replaced by the plan review and the advanced
disclosure). `tabs/ConfirmationDialog.tsx` remains because the secrets tab
still uses it; it should migrate to `DestructiveActionDialog` when that tab
is next touched.

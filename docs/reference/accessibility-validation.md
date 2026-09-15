# Accessibility Validation Ownership

Accessibility is a composed quality gate, not a standalone service or Test
Genie phase. Each evidence type has one active owner.

| Evidence | Owner | Gate | Unavailable evidence |
|---|---|---|---|
| Axe dependency, shared helper, and baseline test | UI Health | A scenario with a browser UI fails when its static harness contract is missing or malformed. | Not applicable to scenarios without a UI surface. |
| Live render and browser accessibility evidence | UI Health | Runtime evidence complements the static harness; it does not replace it. | `DEGRADED`, never passed. |
| Lighthouse accessibility scores for declared routes | Performance Health | A configured score below an accessibility error threshold fails. UI scenarios must declare a page plus accessibility error and warning thresholds. | Browser or Lighthouse unavailability is `DEGRADED`, never passed. |
| Palette and design-token contrast | Brand Manager | WCAG contrast calculations for authored colors and tokens. | Brand Manager does not validate ARIA, labels, or keyboard behavior. |
| Declared accessible name, keyboard reachability, reading order, and state affordances | Experience Manager | Optional experience claims reconcile against BAS accessibility-tree snapshots. | Capture is recorded as degraded/skipped evidence, never a satisfied claim. |

Use the React/Vite template's `ui/src/test-utils/a11y.ts` and
`AppShell.a11y.test.tsx` as the baseline harness. Add feature and interaction
tests where shell coverage is insufficient. Do not create a second compliance
runtime or an accessibility-specific Test Genie phase.

The retired Accessibility Compliance Hub has no active ownership, provider
registration, or control-plane entry.

## Evidence workflow

1. UI Health verifies the static axe harness and its runtime evidence.
2. Performance Health scores each declared UI route with Lighthouse.
3. Brand Manager verifies only the token/color contrast boundary.
4. Experience Manager optionally reconciles declared semantic intent from the
   same accessibility-tree evidence.

This split makes a failed result actionable: a missing test harness belongs to
UI Health, a route budget to Performance Health, a contrast token to Brand
Manager, and a declared semantic intent to Experience Manager.

# Deployment console experience specification

This document is the reconciled cloud experience specification for the
deployment page (`#/deployments/<id>`) and the wizard's plan review, apply
and operation views. It is written before the components and is the contract
the vitest, accessibility and BAS evidence under
the current scenario-to-cloud qualification evidence and the owner-produced
receipts for this plan's candidate.
proves. Requirements: STC-P0-038 (five operator questions), STC-P0-039
(accessible, interruption-safe recovery).

Doctrine (design section L): the browser renders API state; it never decides
policy. Every "can I do this" answer comes from an API field
(`plan.outcome`, `standing.next_action`, `verdict.compatible`, the authz
matrix scope on the route) or from a typed refusal
(`{error:{code,message,retryable,next_action,details}}`). No status-derived
`canX` booleans exist in the UI.

## 1. The five operator questions and the components that answer them

| # | Question | Component | API source | Selector prefix |
|---|----------|-----------|------------|-----------------|
| 1 | Which environment and machine am I controlling? | `console/IdentityHeader` | `GET /deployments/{id}` → `environment`, `target.{machine_id,node_id,enrollment_generation,transport,locator.host}`, `id`, `fence` | `console-identity-*` |
| 2 | Which release is desired and which is running? | `console/ReleasePanel` | desired: `POST /deployments/{id}/plan` (read scope, no effects) → `plan.{release_digest,configuration_digest,desired_revision,outcome}`; observed: `GET /deployments/{id}/health/observation` → `observation.{observed_release_digest,observed_configuration_digest}` | `console-release-*` |
| 3 | Is the application healthy, and how recent is that observation? | `console/HealthPanel` | `observation.{status,freshness,observed_at,producer_ref,checks[],partial,missing_dependencies}` | `console-health-*` |
| 4 | Which operation is running or waiting, and what happens next? | `console/OperationPanel` | `GET /operations/{id}` and `/wait?timeout=` (standing: `state`, `active_step`, `completed_steps`, `step_receipts`, `unknown_effects`, `next_action`, `reattach_command`, `error`); `GET /deployments/{id}/operations` for discovery; SSE `/deployments/{id}/progress?operation_id=` for step messages only | `console-operation-*` |
| 5 | What is the safe recovery or next action? | `console/RecoveryPanel` + `console/DestructiveActionDialog` | `GET /deployments/{id}/recovery-points`, `POST …/rollback-admission` (read scope) → `verdict`, `POST …/{rp}/restore` (destructive), `plan.handoff` / `needs_input` next_action for resume | `console-recovery-*` |

Desired release is never invented from the deployment record: if the plan
cannot be compiled (typed `closure_unavailable`, `needs_input`, denial) the
release panel says so and names the refusal.

The health panel never collapses status and freshness: "Healthy" with
freshness `stale` renders as stale evidence, and an unknown or unspecified
status never renders as healthy (fail closed, P16).

The operation panel shows completed steps and the active step. There is no
percentage bar; the only progress vocabulary is the standing's step list and
receipts.

## 2. Page-state inventory

Every state below is a rendered component state with a `data-state`
attribute and a vitest case. The page is a composition of panels, so panel
states compose: an identity header can be `ready` while the operation panel
is `interrupted`.

| State | Trigger | What the operator sees | Focus / announcement |
|-------|---------|------------------------|----------------------|
| `empty` | No operation has ever been admitted; no recovery points | Operation panel: "No operation is running" plus the primary action "Review plan". Recovery panel: "No recovery points" and the capture action or its refusal | none |
| `loading` | First fetch in flight | Skeleton rows sized like the ready layout; no spinner-only screens; primary actions are rendered disabled with `aria-busy` on the panel | none |
| `ready` | All observations current | Full panels; the primary action is the `next_action` when its kind is executable from the browser | none |
| `degraded` | Health `degraded`/`unhealthy`, freshness `stale`, `partial: true`, or health request failed | Health panel states the verdict and the missing dependencies; the release panel marks a desired/observed mismatch; nothing is hidden | polite live region announces the new verdict |
| `denied` | Typed `unauthenticated`, `forbidden_scope`, `forbidden_target`, `forbidden_origin` | `console/DeniedState` names the missing permission from `details.required_scope` (or the route's scope from the authz matrix when the refusal carries none), the target key for `forbidden_target`, and renders `next_action` as the remediation (sign in link or docs). The rest of the page stays readable | focus moves to the refusal heading |
| `interrupted` | Reload with a durable `{deployment_id, operation_id, plan_digest}` in sessionStorage, or a non-terminal operation found on `GET /deployments/{id}/operations` | Operation panel rehydrates from `GET /operations/{id}`, shows "Resumed from durable state", the reattach command with a copy button, and continues the server-side wait. Wizard: the deploy step reopens on the operation view, never on step one | assertive live region: "Operation resumed" |
| `failed-recovery` | Standing `failed_recovery` or `unknown_effects` non-empty | Operation panel lists each unknown effect with its `retry` and `next_action` text; the recovery panel becomes the primary region and its heading receives focus; destructive actions stay behind the preview dialog | assertive live region with the state |
| `needs-input` | `plan.outcome === "needs_input"` or typed `needs_input` refusal | The review renders the onboarding handoff (`next_action.label` linking to `next_action.reference`) and the `details.missing` list. No inline secret prompt exists in the console or the review | focus on the handoff link |
| `no-op` | `plan.outcome === "no_op"` | Review says nothing would change and disables apply with the reason | none |

## 3. Wizard flow

1. Manifest → Secrets → Build → Preflight (unchanged).
2. Deploy step, phase `review`: `POST /deployments` (with bundle and
   provided secrets) then `POST /deployments/{id}/plan`; `console/PlanReview`
   renders `preview.changes` (operation, effect, capability, verification,
   recovery, retry, cancel point), `preview.data_effects` (subject, effect),
   `preview.downtime` (`expected_seconds`, reason), `preview.recovery_strategy`,
   and `preview.shell_preview` collapsed under "Show equivalent commands".
3. Phase `apply`: `POST /deployments/{id}/plan/apply` with the reviewed
   `plan_digest` and a browser-minted UUID `request_key`. `plan_stale` and
   `plan_digest_mismatch` refusals re-open the review with the reason.
4. Phase `operation`: `console/OperationPanel` on the returned
   `operation_id`; the triple `{deployment_id, operation_id, plan_digest}`
   is written to `sessionStorage["stc.console.operation.<deployment_id>"]`
   before the first render of the operation view.
5. Reload during any phase after step 2 rehydrates from the durable record
   (`GET /operations/{id}`) and lands on the operation view. If the operation
   is terminal the view shows the result and the next action.

The legacy pipeline (`POST /deployments/{id}/execute`, which is durable and
returns `operation_id`) stays available from the deployment page under
"Advanced → Run pipeline with options" for forced bundle builds and
preflight; its operation is shown by the same operation panel.

## 4. Progressive disclosure

Always visible: identity header, release, health, operation, recovery.

Behind a single "Advanced surfaces" disclosure (`console-advanced-toggle`,
`aria-expanded`): Live State (processes, ports, system, caddy), Files,
Drift, Secrets, History, Investigations, Terminal, raw manifest / setup /
deploy result JSON and logs. Deep links (`?tab=terminal`) open the
disclosure automatically so shared URLs keep working.

Inside the operation panel: step receipts and unknown effects are listed;
raw receipt JSON is a collapsed block. Inside the review: the shell preview
is collapsed by default.

## 5. Actions and refusals

An action is rendered from an `ActionState`:

```ts
type ActionState = {
  id: string;
  label: string;
  available: boolean;
  reason?: { code: string; message: string };
  nextAction?: { owner?: string; kind?: string; reference?: string; label?: string };
  requiredScope?: string; // from the authz matrix route entry
};
```

Availability comes from `plan.outcome`, `standing.next_action`,
`verdict.compatible` or a typed refusal. Unavailable actions render as a
disabled control with the reason text next to it and the `next_action` as
the remediation link or button. Stop, Start and Run pipeline are always
rendered and the API's typed refusal is shown when it refuses.

Destructive actions (restore, rollback, run pipeline with a bundle rebuild,
cancel) open `DestructiveActionDialog`, which names the target key, the
affected data bindings (from the recovery point's `bindings` or the plan's
`data_effects`), the downtime, and requires the operator to type the
deployment's short id. The dialog is `role="dialog" aria-modal="true"`,
traps focus, closes on Escape and restores focus to the opener.

## 6. Local state preservation

Server observations refresh on their own cadence (health every 60 s,
standing through the server-side wait). The following local state survives
any refresh because it lives in component state that is not keyed by the
observation: the typed confirmation text in a dialog, expanded disclosures,
the selected recovery point, and the review scroll position. A vitest case
updates the mocked standing while a dialog has text typed and asserts the
text remains.

## 7. Accessibility contract

- Keyboard order follows reading order: header → release → health →
  operation → recovery → advanced. Every control is a `button`, `a` or
  form element; nothing relies on a click handler on a `div`.
- Focus restoration: closing a dialog returns focus to the opener; a denial
  moves focus to the refusal heading; resuming an interrupted operation
  moves focus to the operation heading.
- Status announcements: one polite live region for observation changes
  (health verdict, release mismatch) and one assertive live region for
  operation state changes (`console-operation-live`).
- Contrast: text tokens are `slate-100` on `slate-950/900` and status
  colours use the `-300` tints on `/15` backgrounds, which meet WCAG AA at
  the sizes used. axe (`expectNoA11yViolations`) runs on console, review,
  denied and recovery states.
- Reduced motion: no animation on state change; the only motion is the
  spinner, which is replaced by a static indicator under
  `prefers-reduced-motion: reduce` (`motion-safe:animate-spin`).
- Responsive: panels stack at 360 px, primary actions wrap and are never
  clipped (`flex-wrap`, `min-w-0`); two-column grid from 1024 px.

## 8. Evidence lanes

- vitest: every component state above, keyboard-only journeys with
  `userEvent.tab()`, axe on each surface.
- BAS (`bas/cases/19-console/`): observer and mutating journeys against the
  running scenario. Apply, restore, and cancel use the routed test-storage
  lease and must emit target/data assertions; observer-only rendering cases
  remain useful but cannot satisfy mutating qualification.
- Independent walkthrough (P19-A06, UX-08): pending EXT-08.

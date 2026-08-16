# Offer Desk UX architecture

Offer Desk is the operator console for a typed offer graph, machine-evaluable
triggers, promotion proposals, and a ranked board. It surfaces Money Ledger
actuals and posture but never stores accounting balances. Coverage, trigger
condition, and empirical actuals remain separate evidence classes.

## State vocabulary

The five pages use the shared `ExperienceSurface` state contract:

| State | Meaning | Rendering rule |
|---|---|---|
| default/idle | The surface is ready. | Show records and their evidence source. |
| loading/pending | Initial data is in flight. | Keep headings and announce busy state. |
| refreshing/syncing | A read is refreshing. | Retain known rows and show refresh status. |
| saving | A mutation is in flight. | Disable the submitted action. |
| success | A mutation completed. | Announce the result and refresh dependent views. |
| empty | No records are present. | Explain the first action; do not imply zero revenue. |
| partial/stale | Some evidence is incomplete or old. | Name the affected source, age, and consequence. |
| validation-error/request-error | A form or RPC failed. | Put a text error beside the action and retain context. |
| retrying/permission-denied/offline | Retry, role, or connectivity boundary. | Say what can be done next; never silently drop an action. |

On the board, “no actuals”, “zero actuals”, “blocked by a gate”, and “not yet
evaluated” are distinct. Source availability is not merged with trigger
condition, and neither is merged with empirical revenue.

## Board regions

The dashboard has three ranked regions:

1. **Fired triggers** — conditions currently eligible for operator attention.
2. **Blocked offers** — offers with a named refusal, missing fact, role, or
   source reason.
3. **Earning nothing** — active offers whose actuals are available and whose
   observed amount is zero.

The posture card separately shows Money Ledger source, posture age, goal
verdicts, evaluation freshness/condition, and the default-alive gap. The board
never converts unavailable actuals into zero and never hides the source reason.

## Surface specifications

| Route | Purpose and primary action | Data sources | Degradation and mobile strategy |
|---|---|---|---|
| `/` | Read ranked opportunities and posture. | Board RPC, Money Ledger position/goals, evaluation run. | Each source is labelled unavailable/partial/stale; cards stack on mobile. |
| `/offers` | Create nodes/edges, inspect lifecycle, and attempt promotion. | Catalog nodes/edges and gate proposals. | Refusal includes legal transition/remedy; the accessible offer table scrolls inside its container. |
| `/triggers` | Declare triggers, record facts, and dry-run evaluation. | Trigger/fact/evaluation RPCs. | UNKNOWN is distinct from unsatisfied; fact age and freshness remain visible. |
| `/proposals` | Accept or decline a promotion proposal with a reason. | Proposal list and catalog nodes. | Agent callers see the operator requirement; decline history remains visible. |
| `/settings` | Change locale/theme and inspect connection posture. | Local settings and Money Ledger connection status. | Controls remain stacked and keyboard reachable. |

The offer graph image is orientation only. The offer table carries the same
node/status/action information for screen readers, keyboard users, and narrow
viewports. Intended prices, when supplied on an edge, use a currency code and
remain distinct from charged or received amounts owned by upstream systems.

## Accessibility and responsive contract

All state-changing controls have machine-readable claims and explicit labels.
All forms associate labels with controls; all errors/statuses use text and live
regions. Compact actions retain a 44px target, bottom navigation clears the
safe area, and focus rings remain visible in light and dark themes. The page
body never scrolls horizontally: only explicitly wide tables do.

See [`../../experience/index.json`](../../experience/index.json),
`ui/src/consts/selectors.ts`, and
[`../reference/cli-commands.md`](../reference/cli-commands.md).

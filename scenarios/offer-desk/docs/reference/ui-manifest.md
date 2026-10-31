# Offer Desk UI surface reference

This scenario-specific reference describes the shipped console. It is not a
copy of the `react-vite` template manifest. The machine-readable selector
contract is
[`ui/src/consts/selectors.manifest.json`](../../ui/src/consts/selectors.manifest.json)
and the experience contract is [`../../experience/index.json`](../../experience/index.json).

## Surfaces

| Route | Surface test ID | Primary evidence |
|---|---|---|
| `/` | `page-dashboard` | fired/blocked/earning-nothing regions, source availability, evaluation condition, posture |
| `/offers` | `page-offers` | grouped relationship view (`catalog-grouped-view`), edge-count table, flat table toggle, lifecycle/refusal state, node/transition/edge forms |
| `/triggers` | `page-triggers` | trigger editor, dry-run verdict, fact trace/registry, freshness |
| `/proposals` | `page-proposals` | proposer, requested status, evidence, decline history, accept/decline actions |
| `/settings` | `page-settings` | theme/locale and ledger connection posture |

## Conventions

- BAS cases use `@selector/pages.*` tokens from `ui/src/consts/selectors.ts`.
  Rebuild `bas/registry.json` with `test-genie registry build` after case
  changes.
- Board regions preserve three evidence classes: ranking condition, trigger
  condition, and empirical actuals. A failed Money Ledger join is rendered as
  named unavailability rather than zero.
- Intended edge prices carry a currency code and remain separate from charged
  or received amounts. Status and direction are text, not colour alone.
- The grouped relationship view is derived from catalog verification and keeps
  the flat table reachable when verification is unavailable; duplicate
  identities are surfaced with an alert rather than silently merged.
- Forms have native labels, state-changing controls have machine claims, and
  compact actions keep the 44px mobile target.

## Component ownership

Pages live under `ui/src/pages`; shell/navigation under `ui/src/layout`;
shared controls under `ui/src/components`; Connect clients under `ui/src/api`;
selectors under `ui/src/consts`. The graph illustration is supplemental; the
semantic table is the canonical accessible representation.

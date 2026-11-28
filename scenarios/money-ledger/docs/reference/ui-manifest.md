# Money Ledger UI surface reference

This is a scenario reference, not a copy of the `react-vite` template
manifest. The machine-readable selector contract is
[`ui/src/consts/selectors.manifest.json`](../../ui/src/consts/selectors.manifest.json)
and the experience contract is [`../../experience/index.json`](../../experience/index.json).

## Surfaces

| Route | Surface test ID | Primary evidence |
|---|---|---|
| `/` | `page-dashboard` | `position-runway`, completeness, goal verdicts, position delta, `runway-burn-trend`, `runway-burn-table` |
| `/accounts` | `page-accounts` | `account-table`, balance basis/gap, paired transfer form |
| `/journal` | `page-journal` | `journal-table`, event basis/source, reversal action |
| `/adapters` | `page-adapters` | adapter availability, failure reason, last-success age |
| `/statements` | `page-statements` | period selector, category breakdown, coverage note, export |
| `/settings` | `page-settings` | theme and locale controls, goal summary |

## Conventions

- Use selector tokens from `selectors.ts`; do not bind workflows to incidental
  CSS or text. Regenerate the BAS registry with `test-genie registry build`
  after changing a case.
- A money value is displayed with its currency code, tabular numerals, sign,
  and visible basis/coverage. An unavailable value is textually unavailable,
  never a synthetic zero.
- The dashboard trend is backed by observed journal postings, supports 7/30/90
  day windows, and exposes the adopted Cartesian chart plus a visible table.
  Missing or partial data is a named gap; it is not interpolated.
- Forms use native labels and controls. Async success/error messages use
  status/alert semantics. Shared buttons and navigation retain the 44px mobile
  target and safe-area padding.

## Component ownership

Pages live under `ui/src/pages`; shell/navigation under `ui/src/layout`;
shared controls under `ui/src/components`; API clients under `ui/src/api`;
selectors under `ui/src/consts`. Product behavior remains in the page/API
layer and is covered by the scenario experience contract.

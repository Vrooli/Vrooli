# React UI usage-contract fleet

Last measured: 2026-11-15. Scope: **114 React UI trees, 8,802 production source files, 25 findings**. The scan inspects published RCL contracts at call sites. The per-contract downward ratchet is stored in `ui/usage-contract-ratchet.json` and enforced by `pnpm usage-contracts:fleet`.

| Contract | Initial findings | Current findings | Change |
|---|---:|---:|---:|
| AppShell | 2 | 1 | −1 |
| AppShell.items | 3 | 0 | −3 |
| Card | 1 | 1 | 0 |
| ContextMenu | 2 | 1 | −1 |
| ContextMenu.items | 2 | 0 | −2 |
| DataTable.columns | 1 | 0 | −1 |
| DataTable.rows | 16 | 0 | −16 |
| ExperienceSurface | 2 | 0 | −2 |
| FullPageDrawer | 4 | 0 | −4 |
| Overlay footer | 32 | 19 | −13 |
| PlanSummary.facts | 4 | 0 | −4 |
| RadioGroup.options | 13 | 0 | −13 |
| ResponsiveDialog | 1 | 0 | −1 |
| Select | 7 | 0 | −7 |
| Select.options | 19 | 0 | −19 |
| SettingsList | 3 | 3 | 0 |
| Tabs | 4 | 0 | −4 |
| Tabs.items | 3 | 0 | −3 |
| Timeline.events | 2 | 0 | −2 |
| **Total** | **121** | **25** | **−96** |

The per-contract ratchet compares against a 100-finding baseline and passes at 25 findings (−75). The scan command is `pnpm usage-contracts:fleet` from `scenarios/react-component-library/ui`; its JSON output includes the per-contract totals and ratchet comparison.

False-positive evidence is reported as three separate, unpaired snapshots. The initial systematic n=60 sample found 38 false positives (63.3%) at 121 findings. A later systematic n=50 sample found 18 false positives (36%) at 69 findings. After consumer edits reduced the fleet to 25 findings, the complete census found 1 false positive (4.0%); there were fewer than 50 findings to sample, so the census was not expanded with duplicate rows. The methods, finding labels and snapshot hashes are in `runs/ee-forcing-functions-20261115/evidence/usage-contract-fp-before-20261115.md`, `usage-contract-fp-sample-20261115.md`, and `usage-contract-fp-final-census-20261115.md`. Rates are descriptive, are not paired, and do not estimate false negatives.

An intermediate 26-finding scan failed the ratchet because Nutrition Planner's Today swap declared two primary actions. That call site was corrected to one primary; the final 25-finding fleet scan passed.

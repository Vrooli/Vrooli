# UI Spec Reconciliation

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

A page document declares intent in `experience/pages/`. Source bindings identify implementations; adoption provenance identifies released library usage. Sketch placement, declared library ownership and observed source evidence are separate fields. Verification does not turn local code into an adoption.

## Region-to-file join

The resolver accepts test IDs and bounded single-attribute selectors. It proves literal intrinsic JSX attributes, including bounded literal-prop forwarding through supported relative or library imports. Unsupported selectors, opaque wrappers, ambiguous matches and unproven forwarding retain explicit reasons. Source reachability is not proof that a conditional branch is visible in the current browser session.

The adoption-path join follows explicit provenance. Filename similarity is a labeled heuristic, never proven evidence. Tests: `api/internal/reconcile/resolver_test.go` and `packages/react-component-library/tooling/dom-bindings.test.mjs`.

## Region verdicts

| Verdict | Checked meaning |
| --- | --- |
| `matches` | Proven adoption agrees with the selected built asset/version. |
| `resolved-local` | Proven custom source implements the region, with no conflicting library placement. |
| `drifted` | Proven source disagrees with the selected adoption/placement. |
| `missing` | No matching implementation source was found. |
| `unverifiable` | A binding/provenance/placement cannot establish the required conclusion. |
| `extra` | Scanned slot source has no page-region mapping; advisory. |

Tests: `api/internal/reconcile/verdict_test.go`. Every result includes a reason. Local resolution is not a claim that all UX behavior is correct.

## Coverage

`built`, `declared` and `invented` describe sketch supply. `library_backed` and `local` count authored ownership declarations. `resolved` counts uniquely proven source bindings; `resolved_local` identifies successful custom implementations. A result can have source coverage without a built sketch asset.

A page with zero proven source bindings cannot pass. Required regions default to true; an explicit false remains optional. Tests `TestZeroResolvedRegionsNeverPass`, `TestResolvedCustomCodeHasItsOwnVerdictAndReason`, and `TestRegionRequiredDefaultsToSchemaContract` enforce those boundaries. Required failures still fail even when another region resolves.

## Rendered observation

Use `react-component-library page inspect <scenario> <route>` for browser DOM and source attribution, and `components test page:<Name> --version workspace` for declared page behavior. They answer different questions. See the [CLI reference](../reference/cli-commands.md#inspect-a-running-page).

## Design intent

Reuse a suitable library asset where it serves the page. Composed local pages remain valid source implementations and are reported honestly. Library adoption, page behavior and visual quality require their own evidence. Dated zero-coverage measurements from earlier design plans are superseded by the make-the-library-observable closing measurement; they are not current baseline facts.

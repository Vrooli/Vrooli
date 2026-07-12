# Validation Ownership

Validation findings are intentionally routed by concern. A provider may link to
another provider's evidence, but it must not duplicate its verdict.

| Concern | Authority | Evidence | Primary severity / presentation |
| --- | --- | --- | --- |
| Declared identity, `theme-color`, manifest values, and iOS status metadata | Brand Manager | `index.html`, web manifest, token contract | Required declaration drift; direct remediation |
| Rendered chrome/safe-area alignment and iframe-safe geometry | ui-health | Runtime screenshot, DOM, layout, viewport profile | Required runtime finding, grouped by viewport |
| Root height, scroll ownership, and cross-boundary scrolling | ui-health | Source plus runtime layout | Cross-boundary `scrollIntoView`/`window.scrollTo` advisory; scoped container `scrollTo` allowed |
| Stateful user intent such as restoration, selection scrolling, and tab transitions | Experience Manager | Explicit BAS interaction trace plus capture | Required claim failure with its state precondition |
| Governed component provenance and adoption freshness | react-component-library | Adoption inventory | Grouped governance advisory; never placed ahead of correctness blockers |
| Validation truth versus whether a run blocks policy | Test Genie | Provider result, evidence state, configured policy | Show provider truth first, then independent gate classification |

Autofix is limited to syntax-safe, template-recognized edits. Runtime layout,
scroll behavior, and chrome decisions remain detection-only because a mechanical
rewrite cannot establish intended ownership or interaction behavior.

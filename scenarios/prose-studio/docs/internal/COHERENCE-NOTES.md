# UI coherence notes

The surface migration keeps the existing semantic Tailwind tokens and shared `Card`, `Button`, and `EmptyState` primitives. Surface files compose those contracts and do not introduce raw palette values.

## Convergence scorecard

| Indicator | Result |
| --- | --- |
| Surface files with raw palette values | 0 in the four migrated pages |
| Legacy class contracts | 0 found in the four migrated pages |
| Core primitives | Existing token-backed primitives reused |
| High-traffic surfaces | Variation, styles, document, declarations split |
| Remaining debt | Typed live query clients and BAS fixtures |

The previous single `ProseSurfacePage` dispatcher remains as a compatibility adapter for tests and callers. Production routes now point directly to the four distinct surfaces.

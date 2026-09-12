Feature folders own the primary SDA product surfaces:

- `graph/` contains graph route composition, graph controls, D3 canvas, telemetry,
  selected-node inspection, and the current graph data hook.
- `deployment/` contains deployment readiness route composition, dashboard panels,
  metadata gap panels, and recommended workflow guidance.
- `catalog/` contains scenario catalog route composition, scan/optimize panels,
  and the current catalog data hook.

Shared UI primitives stay under `components/ui`. Cross-surface page state remains
in `app/routes.tsx` until React Router and React Query are adopted.

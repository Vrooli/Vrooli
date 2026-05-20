import { AppShell } from "./components/AppShell";
import { HealthCard } from "./features/health/HealthCard";

/**
 * PLACEHOLDER — REPLACE WHEN BUILDING THE REAL UX.
 *
 * This `App` is a minimum-viable scaffold so the template boots green.
 * It is not a reasonable end-state for your scenario. When you design
 * the real product:
 *   - Replace `AppShell` (or rewrite it heavily) with the real shell,
 *     navigation, and layout your scenario needs.
 *   - Replace this single-page composition with whatever surfaces
 *     your scenario actually has (router, pages, panels, dashboards).
 *   - Keep i18n, accessibility selectors, and design-token usage
 *     intact inside the new layout — those are durable seams, not
 *     placeholder choices.
 *
 * Pattern for adding a feature: create
 * `features/<name>/<Name>Card.tsx`, then import + render it. Deleting
 * a feature: delete the folder, remove the import + render. There is
 * no central registry to mutate.
 */
export default function App() {
  return (
    <AppShell>
      <HealthCard />
    </AppShell>
  );
}

import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { BackdropsPage } from "../pages/BackdropsPage";
import { CandidatesPage } from "../pages/CandidatesPage";
import { CatalogPage } from "../pages/CatalogPage";
import { ComposePage } from "../pages/ComposePage";
import { DashboardPage } from "../pages/DashboardPage";
import { PlacementsPage } from "../pages/PlacementsPage";
import { RemixPage } from "../pages/RemixPage";
import { SettingsPage } from "../pages/SettingsPage";
import { StylePage } from "../pages/StylePage";
import { SurfacesPage } from "../pages/SurfacesPage";
import { SweepPage } from "../pages/SweepPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Every route resolves to its own component. That was not true before: eleven
 * routes — catalog, surfaces, compose, placements, styles/:styleId, candidates,
 * backdrops, renders/:renderId and more — all resolved to a single 181-line
 * `WorkbenchPage` holding four hardcoded style rows and CSS-gradient
 * "specimens". Nine experience pages were declared and three components
 * existed, so the navigation promised a studio the app did not have.
 *
 * `routes.test.tsx` asserts the one-to-one mapping, so the collapse cannot
 * quietly return by someone reusing a component for a new route.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "catalog", element: <CatalogPage /> },
      { path: "styles/:styleId", element: <StylePage /> },
      { path: "sweep", element: <SweepPage /> },
      { path: "remix", element: <RemixPage /> },
      { path: "compose", element: <ComposePage /> },
      { path: "placements", element: <PlacementsPage /> },
      { path: "candidates", element: <CandidatesPage /> },
      { path: "backdrops", element: <BackdropsPage /> },
      { path: "surfaces", element: <SurfacesPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

// Opt in before React Router v7 makes these behaviors the defaults. Data
// routers own both flags; RouterProvider owns transition scheduling only.
const dataRouterFuture = {
  v7_relativeSplatPath: true,
  v7_startTransition: true,
};
const routerProviderFuture = { v7_startTransition: true };

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: dataRouterFuture });
  return <RouterProvider router={router} future={routerProviderFuture} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: dataRouterFuture });
  return <RouterProvider router={router} future={routerProviderFuture} />;
}

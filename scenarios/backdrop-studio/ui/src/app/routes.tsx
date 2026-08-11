import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { WorkbenchPage } from "../pages/WorkbenchPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "backdrops", element: <WorkbenchPage /> },
      { path: "backdrops/:backdropId", element: <WorkbenchPage /> },
      { path: "candidates", element: <WorkbenchPage /> },
      { path: "renders/:renderId", element: <WorkbenchPage /> },
      { path: "renders/:renderId/placements", element: <WorkbenchPage /> },
      { path: "catalog", element: <WorkbenchPage /> },
      { path: "compose", element: <WorkbenchPage /> },
      { path: "placements", element: <WorkbenchPage /> },
      { path: "placements/:candidateId", element: <WorkbenchPage /> },
      { path: "styles/:styleId", element: <WorkbenchPage /> },
      { path: "surfaces", element: <WorkbenchPage /> },
      { path: "surfaces/:surfaceId", element: <WorkbenchPage /> },
      { path: "sweep", element: <WorkbenchPage /> },
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

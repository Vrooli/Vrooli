import {
  createBrowserRouter,
  MemoryRouter,
  RouterProvider,
  useRoutes,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";

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
 * specific starting URL. This helper uses a declarative MemoryRouter because
 * these tests exercise routing, not loader/action request machinery.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  return <MemoryRouter initialEntries={initialEntries} future={dataRouterFuture}><TestRouteTree /></MemoryRouter>;
}

// A plain memory router is enough for route/shell tests and avoids constructing
// fetch Request objects during client navigation in the jsdom test environment.
// Production routing remains data-router based in AppRouter.
function TestRouteTree() { return useRoutes(routes); }

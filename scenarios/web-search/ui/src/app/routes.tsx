import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { DisputesPage } from "../pages/DisputesPage";
import { FindingsPage } from "../pages/FindingsPage";
import { OpsPage } from "../pages/OpsPage";
import { SearchPage } from "../pages/SearchPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. Kept module-private; tests construct routers through
 * the exported `AppRouter` / `TestAppRouter` components, which build from this
 * same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "search", element: <SearchPage /> },
      { path: "findings", element: <FindingsPage /> },
      { path: "disputes", element: <DisputesPage /> },
      { path: "ops", element: <OpsPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * React Router v7 future flags. Opting in early silences the v6 migration
 * console warnings (which the strict test-setup console guard treats as
 * failures) and makes the eventual v7 upgrade a no-op. Kept in one place so
 * every router construction site stays in sync.
 */
const ROUTER_FUTURE = {
  v7_startTransition: true,
  v7_relativeSplatPath: true,
} as const;

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_FUTURE} />;
}

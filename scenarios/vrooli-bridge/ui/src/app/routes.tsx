import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { RunsPage } from "../pages/RunsPage";
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
      { path: "runs", element: <RunsPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

// Opt into the React Router v7 behaviours now so the data routers don't emit
// future-flag console warnings (which the test harness treats as failures) and
// the eventual v7 upgrade is a no-op. For data routers, `v7_startTransition` is
// read off the <RouterProvider> `future` prop, while `v7_relativeSplatPath` is
// read off the router's own `future` config — so both placements are needed.
const ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
const PROVIDER_FUTURE = { v7_startTransition: true } as const;

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={PROVIDER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={PROVIDER_FUTURE} />;
}

import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { FleetPage } from "../pages/FleetPage";
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
      { path: "fleet", element: <FleetPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
/**
 * Opt into the React Router v7 transition behaviours now so the migration
 * warnings don't fire (and trip the strict console guard in tests). The two
 * flags are consumed in different places for data routers: `v7_relativeSplatPath`
 * is a `createRouter` option (a no-op for us — we have no splat routes), while
 * `v7_startTransition` is a `<RouterProvider>` prop (wraps router state updates
 * in `React.startTransition`).
 */
const CREATE_ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
const ROUTER_PROVIDER_FUTURE = { v7_startTransition: true } as const;

export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: CREATE_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: CREATE_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

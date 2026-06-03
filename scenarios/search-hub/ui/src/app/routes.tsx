import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SearchPage } from "../pages/SearchPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
// The route table is intentionally co-located with the router components it
// feeds (AppRouter / TestAppRouter); an array export isn't a primitive so
// allowConstantExport doesn't cover it. HMR of this config file is irrelevant.
// eslint-disable-next-line react-refresh/only-export-components
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <SearchPage /> },
      { path: "dashboard", element: <DashboardPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
// Opt into the React Router v7 behaviors now so v6 stops emitting future-flag
// deprecation warnings (the strict console guard in src/test-setup.ts fails any
// test that produces a console.warn). v7_relativeSplatPath is a data-router
// `future` option; v7_startTransition is a RouterProvider prop.
const DATA_ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
const ROUTER_PROVIDER_FUTURE = { v7_startTransition: true } as const;

export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: DATA_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: DATA_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

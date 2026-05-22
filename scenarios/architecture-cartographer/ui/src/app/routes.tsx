import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { NewTargetPage } from "../pages/NewTargetPage";
import { OverviewPage } from "../pages/OverviewPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TargetWorkspacePage } from "../pages/TargetWorkspacePage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Per-target sub-routes (graph, manifest, conflicts, apply, analytics) land
 * as children of `targets/:encodedPath` in later phases. The workspace
 * already renders an `<Outlet />` ready to host them.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "targets/new", element: <NewTargetPage /> },
      { path: "targets/:encodedPath", element: <TargetWorkspacePage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
// Opt into the v7 React Router future flags so the v6→v7 transition warnings
// stay silent in production renders and (critically) in tests, where the
// shared test-setup treats every console.warn as a hard failure.
//
// The flag set is split: `v7_startTransition` is read off the
// `RouterProvider` `future` prop; the remaining flags are read off the
// data router's `future` option (createBrowserRouter / createMemoryRouter).
const dataRouterFuture = {
  v7_relativeSplatPath: true,
  v7_fetcherPersist: true,
  v7_normalizeFormMethod: true,
  v7_partialHydration: true,
  v7_skipActionErrorRevalidation: true,
} as const;
const routerProviderFuture = { v7_startTransition: true } as const;

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

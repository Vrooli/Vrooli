/* eslint-disable react-refresh/only-export-components --
 * The route table (`routes`) is exported alongside the router components on
 * purpose so tests build an in-memory router from the same config. The
 * Fast-Refresh-only warning about mixed exports is accepted here. */
import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { OverviewPage } from "../pages/OverviewPage";
import { TargetsPage } from "../pages/TargetsPage";
import { DestinationsPage } from "../pages/DestinationsPage";
import { PlansPage } from "../pages/PlansPage";
import { RunsPage } from "../pages/RunsPage";
import { RestoresPage } from "../pages/RestoresPage";
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
      { index: true, element: <OverviewPage /> },
      { path: "targets", element: <TargetsPage /> },
      { path: "destinations", element: <DestinationsPage /> },
      { path: "plans", element: <PlansPage /> },
      { path: "runs", element: <RunsPage /> },
      { path: "restores", element: <RestoresPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, {
    future: { v7_relativeSplatPath: true },
  });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, {
    initialEntries,
    future: { v7_relativeSplatPath: true },
  });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

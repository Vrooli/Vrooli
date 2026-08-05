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
import { lazy } from "react";

import { AppShell } from "../layout/AppShell";

// Keep the first-paint overview in the entry bundle, while deferring the
// operator workflows until their route is visited. These pages pull in the
// heaviest dialogs, forms, and API surfaces; shipping them on `/` made the
// initial Lighthouse document pay for every workflow.
const OverviewPage = lazy(() => import("../pages/OverviewPage"));
const TargetsPage = lazy(() => import("../pages/TargetsPage"));
const DestinationsPage = lazy(() => import("../pages/DestinationsPage"));
const PlansPage = lazy(() => import("../pages/PlansPage"));
const RunsPage = lazy(() => import("../pages/RunsPage"));
const RestoresPage = lazy(() => import("../pages/RestoresPage"));
const DrillsPage = lazy(() => import("../pages/DrillsPage"));
const SettingsPage = lazy(() => import("../pages/SettingsPage"));

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
      { path: "drills", element: <DrillsPage /> },
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

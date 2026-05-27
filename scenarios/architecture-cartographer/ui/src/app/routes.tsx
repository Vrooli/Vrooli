import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { RouteErrorFallback } from "../components/RouteErrorFallback";
import { HistoryPage } from "../pages/HistoryPage";
import { NewTargetPage } from "../pages/NewTargetPage";
import { OverviewPage } from "../pages/OverviewPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TargetAnalyticsPage } from "../pages/TargetAnalyticsPage";
import { TargetApplyDomainPage } from "../pages/TargetApplyDomainPage";
import { TargetApplyPage } from "../pages/TargetApplyPage";
import { TargetConflictDetailPage } from "../pages/TargetConflictDetailPage";
import { TargetConflictsPage } from "../pages/TargetConflictsPage";
import { TargetGraphPage } from "../pages/TargetGraphPage";
import { TargetDomainsPage } from "../pages/TargetDomainsPage";
import { TargetWorkspacePage } from "../pages/TargetWorkspacePage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Per-target sub-routes for graph / domains / apply / analytics land in
 * later phases — conflicts is the first to ship and demonstrates the
 * nested-route + workspace-subnav pattern.
 */
// The route table is intentionally co-located with the router-provider
// components in this file; splitting it out solely to satisfy the HMR-only
// react-refresh rule would scatter the canonical config tests depend on.
// eslint-disable-next-line react-refresh/only-export-components
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    errorElement: <RouteErrorFallback />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "targets/new", element: <NewTargetPage /> },
      {
        path: "targets/:encodedPath",
        element: <TargetWorkspacePage />,
        children: [
          { path: "graph", element: <TargetGraphPage /> },
          { path: "domains", element: <TargetDomainsPage /> },
          { path: "conflicts", element: <TargetConflictsPage /> },
          { path: "conflicts/:conflictId", element: <TargetConflictDetailPage /> },
          { path: "apply", element: <TargetApplyPage /> },
          { path: "apply/:domainKey", element: <TargetApplyDomainPage /> },
          { path: "analytics", element: <TargetAnalyticsPage /> },
        ],
      },
      { path: "history", element: <HistoryPage /> },
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

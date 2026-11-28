import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";
import { lazy, Suspense, type ReactNode } from "react";

import { AppShell } from "../layout/AppShell";
import { RouteErrorFallback } from "../components/RouteErrorFallback";
import { OverviewPage } from "../pages/OverviewPage";

const HistoryPage = lazy(() => import("../pages/HistoryPage").then((m) => ({ default: m.HistoryPage })));
const NewTargetPage = lazy(() => import("../pages/NewTargetPage").then((m) => ({ default: m.NewTargetPage })));
const SettingsPage = lazy(() => import("../pages/SettingsPage").then((m) => ({ default: m.SettingsPage })));
const TargetAnalyticsPage = lazy(() =>
  import("../pages/TargetAnalyticsPage").then((m) => ({ default: m.TargetAnalyticsPage })),
);
const TargetApplyDomainPage = lazy(() =>
  import("../pages/TargetApplyDomainPage").then((m) => ({ default: m.TargetApplyDomainPage })),
);
const TargetApplyPage = lazy(() => import("../pages/TargetApplyPage").then((m) => ({ default: m.TargetApplyPage })));
const TargetCampaignDetailPage = lazy(() =>
  import("../pages/TargetCampaignDetailPage").then((m) => ({ default: m.TargetCampaignDetailPage })),
);
const TargetCampaignPage = lazy(() =>
  import("../pages/TargetCampaignPage").then((m) => ({ default: m.TargetCampaignPage })),
);
const TargetConflictDetailPage = lazy(() =>
  import("../pages/TargetConflictDetailPage").then((m) => ({ default: m.TargetConflictDetailPage })),
);
const TargetConflictsPage = lazy(() =>
  import("../pages/TargetConflictsPage").then((m) => ({ default: m.TargetConflictsPage })),
);
const TargetDomainsPage = lazy(() =>
  import("../pages/TargetDomainsPage").then((m) => ({ default: m.TargetDomainsPage })),
);
const TargetGraphPage = lazy(() => import("../pages/TargetGraphPage").then((m) => ({ default: m.TargetGraphPage })));
const TargetWorkspacePage = lazy(() =>
  import("../pages/TargetWorkspacePage").then((m) => ({ default: m.TargetWorkspacePage })),
);

function lazyElement(node: ReactNode) {
  return <Suspense fallback={null}>{node}</Suspense>;
}

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
      { path: "targets/new", element: lazyElement(<NewTargetPage />) },
      {
        path: "targets/:encodedPath",
        element: lazyElement(<TargetWorkspacePage />),
        children: [
          { path: "graph", element: lazyElement(<TargetGraphPage />) },
          { path: "domains", element: lazyElement(<TargetDomainsPage />) },
          { path: "conflicts", element: lazyElement(<TargetConflictsPage />) },
          { path: "conflicts/:conflictId", element: lazyElement(<TargetConflictDetailPage />) },
          { path: "campaign", element: lazyElement(<TargetCampaignPage />) },
          { path: "campaign/:campaignId", element: lazyElement(<TargetCampaignDetailPage />) },
          { path: "apply", element: lazyElement(<TargetApplyPage />) },
          { path: "apply/:domainKey", element: lazyElement(<TargetApplyDomainPage />) },
          { path: "analytics", element: lazyElement(<TargetAnalyticsPage />) },
        ],
      },
      { path: "history", element: lazyElement(<HistoryPage />) },
      { path: "settings", element: lazyElement(<SettingsPage />) },
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

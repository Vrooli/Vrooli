import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { OverviewPage } from "../pages/OverviewPage";
import { ExposurePage } from "../pages/ExposurePage";
import { RecoveryPage } from "../pages/RecoveryPage";
import { MetricsPage } from "../pages/MetricsPage";
import { AuditPage } from "../pages/AuditPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
/**
 * React Router v7 future flags. Opting in now silences the v6 migration
 * warnings (the test harness treats any console.warn as a failure) and locks
 * in the v7 behaviour we already rely on. Note the split: `v7_relativeSplatPath`
 * is a data-router (`createXRouter`) option, while `v7_startTransition` is a
 * `<RouterProvider>` prop.
 */
const DATA_ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
const ROUTER_PROVIDER_FUTURE = { v7_startTransition: true } as const;

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "exposure", element: <ExposurePage /> },
      { path: "recovery", element: <RecoveryPage /> },
      { path: "metrics", element: <MetricsPage /> },
      { path: "audit", element: <AuditPage /> },
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

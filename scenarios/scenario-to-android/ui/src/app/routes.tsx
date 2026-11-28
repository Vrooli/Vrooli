import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TargetMatrixPage } from "../pages/TargetMatrixPage";
import { RunReviewPage } from "../pages/RunReviewPage";
import { ReadinessPage } from "../pages/ReadinessPage";
import { Providers } from "./providers";
import { DistributionSurface, TargetDetailSurface, RunReviewSurface } from "../features/delivery/ExperienceSurfaces";

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
      { path: "targets", element: <TargetMatrixPage /> },
      { path: "runs", element: <RunReviewPage /> },
      { path: "runs/:runId", element: <RunReviewSurface /> },
      { path: "readiness", element: <ReadinessPage /> },
      { path: "distribution", element: <DistributionSurface /> },
      { path: "targets/:targetId", element: <TargetDetailSurface /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

// Opt in before React Router v7 makes these behaviors the defaults. Data
// routers own both flags; RouterProvider owns transition scheduling only.
const dataRouterFuture = {
  v7_relativeSplatPath: true,
  v7_startTransition: true,
};
const routerProviderFuture = { v7_startTransition: true };

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
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
  return (
    <Providers>
      <RouterProvider router={router} future={routerProviderFuture} />
    </Providers>
  );
}

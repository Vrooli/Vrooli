import { lazy, Suspense, type ReactNode } from "react";
import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { Providers } from "./providers";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

// Keep the shell and the first route small. Feature pages are independent
// delivery surfaces and should not make every initial dashboard load pay for
// flow authoring, evidence review, and settings code.
const DashboardPage = lazy(async () => ({ default: (await import("../pages/DashboardPage")).DashboardPage }));
const SettingsPage = lazy(async () => ({ default: (await import("../pages/SettingsPage")).SettingsPage }));
const FlowsPage = lazy(async () => ({ default: (await import("../pages/FlowsPage")).FlowsPage }));
const EvidencePage = lazy(async () => ({ default: (await import("../pages/EvidencePage")).EvidencePage }));
const DeviceDetailPage = lazy(async () => ({ default: (await import("../pages/DeviceDetailPage")).DeviceDetailPage }));

function PageLoading() {
  const { t } = useTranslation();
  return <div className="p-6 text-app-muted-foreground" role="status">{t(strings.health.loading)}</div>;
}

function lazyPage(element: ReactNode) {
  return <Suspense fallback={<PageLoading />}>{element}</Suspense>;
}

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
      { index: true, element: lazyPage(<DashboardPage />) },
      { path: "flows", element: lazyPage(<FlowsPage />) },
      { path: "evidence", element: lazyPage(<EvidencePage />) },
      { path: "settings", element: lazyPage(<SettingsPage />) },
      { path: "devices/:deviceId", element: lazyPage(<DeviceDetailPage />) },
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
  return (
    <Providers>
      <RouterProvider router={router} future={routerProviderFuture} />
    </Providers>
  );
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

import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";
import { getProxyInfo, type ProxyInfo } from "@vrooli/api-base";

import { AppShell } from "../layout/AppShell";
import { ConformancePage } from "../pages/ConformancePage";
import { DashboardPage } from "../pages/DashboardPage";
import { ProvidersPage } from "../pages/ProvidersPage";
import { RoutePreviewPage } from "../pages/RoutePreviewPage";
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
      { path: "providers", element: <ProvidersPage /> },
      { path: "routes/preview", element: <RoutePreviewPage /> },
      { path: "conformance", element: <ConformancePage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

const APP_SCENARIO = "ai-gateway";
const APP_PROXY_BASE = `/apps/${APP_SCENARIO}/proxy`;

/**
 * Path-based scenario proxies keep their prefix in the browser URL. Use the
 * injected owner metadata as React Router's basename only when it identifies
 * this app and agrees with the current location. Direct-root URLs stay `/`.
 */
export function resolveRouterBasename(
  pathname: string,
  proxyInfo: Pick<ProxyInfo, "appId" | "targetScenario" | "primary"> | null,
): string {
  if (!proxyInfo) return "/";

  const appId = proxyInfo.appId?.trim().toLowerCase();
  const targetScenario = proxyInfo.targetScenario?.trim().toLowerCase();
  const basePath = proxyInfo.primary?.path?.trim().replace(/\/+$/, "");
  if (appId !== APP_SCENARIO || targetScenario !== APP_SCENARIO || basePath !== APP_PROXY_BASE) {
    return "/";
  }

  if (pathname !== basePath && !pathname.startsWith(`${basePath}/`)) {
    return "/";
  }
  return basePath;
}

function getRouterBasename(): string {
  const pathname = typeof window === "undefined" ? "/" : window.location.pathname;
  return resolveRouterBasename(pathname, getProxyInfo());
}

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, {
    basename: getRouterBasename(),
    future: {
      v7_relativeSplatPath: true,
    },
  });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({
  initialEntries,
  basename = "/",
}: {
  initialEntries: string[];
  basename?: string;
}) {
  const router = createMemoryRouter(routes, {
    initialEntries,
    basename,
    future: {
      v7_relativeSplatPath: true,
    },
  });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

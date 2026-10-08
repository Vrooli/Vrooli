import { lazy, Suspense, type ComponentType, type ReactNode } from "react";
import type { RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { RouteFallback } from "./RouteFallback";

// Drill-down detail views are code-split: they carry the heavier per-entity
// rendering and are reached by navigation, not on first paint. The stale-chunk
// reload guard (installed in main.tsx) recovers tabs that lazy-load an old
// chunk after a redeploy.
const TemplateListPage = lazyPage(() => import("../pages/TemplateListPage"), "TemplateListPage");
const TemplateDetailPage = lazyPage(() => import("../pages/TemplateDetailPage"), "TemplateDetailPage");
const RunListPage = lazyPage(() => import("../pages/RunListPage"), "RunListPage");
const ValidationRunDetailPage = lazyPage(() => import("../pages/ValidationRunDetailPage"), "ValidationRunDetailPage");
const DebtListPage = lazyPage(() => import("../pages/DebtListPage"), "DebtListPage");
const DebtDetailPage = lazyPage(() => import("../pages/DebtDetailPage"), "DebtDetailPage");

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "templates", element: withSuspense(<TemplateListPage />) },
      { path: "templates/:templateId", element: withSuspense(<TemplateDetailPage />) },
      { path: "runs", element: withSuspense(<RunListPage />) },
      { path: "runs/:runId", element: withSuspense(<ValidationRunDetailPage />) },
      { path: "debt", element: withSuspense(<DebtListPage />) },
      { path: "debt/:debtKey", element: withSuspense(<DebtDetailPage />) },
    ],
  },
];

export const routerOptions = {
  future: {
    v7_relativeSplatPath: true,
  },
} as const;

export const routerProviderFuture = {
  v7_startTransition: true,
} as const;

// lazyPage adapts a named page export to the default-export shape React.lazy
// expects, so pages keep their named exports (consistent with the eager pages).
function lazyPage(
  loader: () => Promise<Record<string, ComponentType>>,
  name: string,
): ComponentType {
  return lazy(async () => {
    const component = (await loader())[name];
    if (!component) {
      throw new Error(`Route module is missing the '${name}' export`);
    }
    return { default: component };
  });
}

function withSuspense(node: ReactNode): ReactNode {
  return <Suspense fallback={<RouteFallback />}>{node}</Suspense>;
}

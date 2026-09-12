import { createElement } from "react";
import type { RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { AuditPage } from "../pages/AuditPage";
import { DriftPage } from "../pages/DriftPage";
import { ExposurePage } from "../pages/ExposurePage";
import { MetricsPage } from "../pages/MetricsPage";
import { OverviewPage } from "../pages/OverviewPage";
import { RecoveryPage } from "../pages/RecoveryPage";
import { SettingsPage } from "../pages/SettingsPage";

export const DATA_ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
export const ROUTER_PROVIDER_FUTURE = { v7_startTransition: true } as const;

export const routes: RouteObject[] = [
  {
    path: "/",
    element: createElement(AppShell),
    children: [
      { index: true, element: createElement(OverviewPage) },
      { path: "exposure", element: createElement(ExposurePage) },
      { path: "recovery", element: createElement(RecoveryPage) },
      { path: "metrics", element: createElement(MetricsPage) },
      { path: "audit", element: createElement(AuditPage) },
      { path: "drift", element: createElement(DriftPage) },
      { path: "settings", element: createElement(SettingsPage) },
    ],
  },
];

import { type RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { AuditPage } from "../pages/AuditPage";
import { BudgetsPage } from "../pages/BudgetsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { FleetPage } from "../pages/FleetPage";
import { ReadinessPage } from "../pages/ReadinessPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TracePage } from "../pages/TracePage";
import { TrendsPage } from "../pages/TrendsPage";

/**
 * Canonical route table. Lives in its own module (separate from the router
 * *components* in `routes.tsx`) so the route-component files keep fast-refresh
 * working — a file that exports both components and a config object trips the
 * react-refresh boundary rule.
 *
 * Routes map 1:1 to the six core performance workflows plus the overview and
 * settings. Add new pages by appending to the `children` array.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "audit", element: <AuditPage /> },
      { path: "trends", element: <TrendsPage /> },
      { path: "fleet", element: <FleetPage /> },
      { path: "trace", element: <TracePage /> },
      { path: "readiness", element: <ReadinessPage /> },
      { path: "budgets", element: <BudgetsPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

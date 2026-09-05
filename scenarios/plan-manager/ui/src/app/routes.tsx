import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { AuthoringPage } from "../pages/AuthoringPage";
import { DashboardPage } from "../pages/DashboardPage";
import { ExecutionPage } from "../pages/ExecutionPage";
import { PlanDetailPage } from "../pages/PlanDetailPage";
import { PlansPage } from "../pages/PlansPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TriagePage } from "../pages/TriagePage";
import { ValidationPage } from "../pages/ValidationPage";
import { VelocityPage } from "../pages/VelocityPage";

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
      { path: "plans", element: <PlansPage /> },
      { path: "plans/:planId", element: <PlanDetailPage /> },
      { path: "authoring", element: <AuthoringPage /> },
      { path: "execution", element: <ExecutionPage /> },
      { path: "validation", element: <ValidationPage /> },
      { path: "triage", element: <TriagePage /> },
      { path: "velocity", element: <VelocityPage /> },
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

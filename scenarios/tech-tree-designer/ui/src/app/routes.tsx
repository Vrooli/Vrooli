/* eslint-disable react-refresh/only-export-components, no-restricted-syntax */
import { lazy, Suspense } from "react";
import type { ReactNode } from "react";
import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";

const GraphPage = lazy(() => import("../features/graph/GraphPage").then((module) => ({ default: module.GraphPage })));
const OntologyPage = lazy(() => import("../features/ontology/OntologyPage").then((module) => ({ default: module.OntologyPage })));
const PlanningPage = lazy(() => import("../features/planning/PlanningPage").then((module) => ({ default: module.PlanningPage })));

function RouteLoader() {
  return <div className="rounded-lg border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">Loading surface…</div>;
}

function LazyPage({ children }: { children: ReactNode }) {
  return <Suspense fallback={<RouteLoader />}>{children}</Suspense>;
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
      { index: true, element: <DashboardPage /> },
      { path: "graph", element: <LazyPage><GraphPage /></LazyPage> },
      { path: "ontology", element: <LazyPage><OntologyPage /></LazyPage> },
      { path: "planning", element: <LazyPage><PlanningPage /></LazyPage> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

const routerFuture = {
  v7_relativeSplatPath: true,
} as const;

const providerFuture = {
  v7_startTransition: true,
} as const;

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: routerFuture });
  return <RouterProvider router={router} future={providerFuture} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: routerFuture });
  return <RouterProvider router={router} future={providerFuture} />;
}

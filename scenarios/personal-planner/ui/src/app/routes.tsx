import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";
import { lazy, Suspense, type ReactNode } from "react";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { ThemeProvider } from "../theme/ThemeProvider";

// Keep the first Today/Settings path lean. The deeper planning surfaces are
// independent route chunks so a new user does not download every form,
// timeline, review table, and goal editor before seeing the first useful view.
const PlanPage = lazy(() => import("../pages/PlanPage").then(({ PlanPage: page }) => ({ default: page })));
const GoalsPage = lazy(() => import("../pages/GoalsPage").then(({ GoalsPage: page }) => ({ default: page })));
const FocusPage = lazy(() => import("../pages/FocusPage").then(({ FocusPage: page }) => ({ default: page })));
const ReviewPage = lazy(() => import("../pages/ReviewPage").then(({ ReviewPage: page }) => ({ default: page })));

function DeferredRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<p className="route-loading" role="status">Loading surface…</p>}>{children}</Suspense>;
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
      { path: "plan", element: <DeferredRoute><PlanPage /></DeferredRoute> },
      { path: "goals", element: <DeferredRoute><GoalsPage /></DeferredRoute> },
      { path: "focus", element: <DeferredRoute><FocusPage /></DeferredRoute> },
      { path: "review", element: <DeferredRoute><ReviewPage /></DeferredRoute> },
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
    <ThemeProvider>
      <RouterProvider router={router} future={routerProviderFuture} />
    </ThemeProvider>
  );
}

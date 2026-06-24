import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { ConvergencePage } from "../pages/ConvergencePage";
import { DashboardPage } from "../pages/DashboardPage";
import { FocusPage } from "../pages/FocusPage";
import { SettingsPage } from "../pages/SettingsPage";
import { TrialsPage } from "../pages/TrialsPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
/**
 * React Router v7 future flags. Opting in early silences the v6 deprecation
 * warnings (the strict test-setup treats console.warn as a failure) and matches
 * the behavior the app will get on the v7 upgrade.
 */
export const ROUTER_FUTURE = {
  v7_startTransition: true,
  v7_relativeSplatPath: true,
} as const;

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "focus", element: <FocusPage /> },
      { path: "convergence", element: <ConvergencePage /> },
      { path: "trials", element: <TrialsPage /> },
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
  const router = createBrowserRouter(routes, { future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

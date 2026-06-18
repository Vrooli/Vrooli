import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { routerFutureFlags } from "./routerFuture";
import { AppShell } from "../layout/AppShell";
import { ActivityPage } from "../pages/ActivityPage";
import { HomePage } from "../pages/HomePage";
import { LibraryPage } from "../pages/LibraryPage";
import { ModelsPage } from "../pages/ModelsPage";
import { SettingsPage } from "../pages/SettingsPage";
import { WorkspacePage } from "../pages/WorkspacePage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Dual-mode: `/` Home + `/workspace` + `/library` are the Studio surfaces;
 * `/activity` + `/models` + `/settings` are the Console surfaces.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <HomePage /> },
      { path: "workspace", element: <WorkspacePage /> },
      { path: "library", element: <LibraryPage /> },
      { path: "activity", element: <ActivityPage /> },
      { path: "models", element: <ModelsPage /> },
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
  const router = createBrowserRouter(routes, { future: routerFutureFlags });
  // v7_startTransition is read from the RouterProvider prop (it governs React
  // rendering), separately from the data-router future above.
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, {
    initialEntries,
    future: routerFutureFlags,
  });
  return <RouterProvider router={router} future={{ v7_startTransition: true }} />;
}

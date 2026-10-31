import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { PosturePage } from "../pages/PosturePage";
import { DependenciesPage } from "../pages/DependenciesPage";
import { SecretsPage } from "../pages/SecretsPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
/**
 * Opt into the React Router v7 behaviors now so the v6 console warnings don't
 * fire (the test harness treats any `console.warn` as a failure). Kept in one
 * place so both the production and in-memory routers stay aligned.
 */
const ROUTER_FUTURE = {
  v7_startTransition: true,
  v7_relativeSplatPath: true,
} as const;

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <PosturePage /> },
      { path: "dependencies", element: <DependenciesPage /> },
      { path: "secrets", element: <SecretsPage /> },
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

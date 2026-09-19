import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
} from "react-router-dom";

import { DATA_ROUTER_FUTURE, ROUTER_PROVIDER_FUTURE, routes } from "./routeConfig";

/**
 * React Router v7 future flags. Opting in now silences the v6 migration
 * warnings (the test harness treats any console.warn as a failure) and locks
 * in the v7 behaviour we already rely on. Note the split: `v7_relativeSplatPath`
 * is a data-router (`createXRouter`) option, while `v7_startTransition` is a
 * `<RouterProvider>` prop.
 */
/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: DATA_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: DATA_ROUTER_FUTURE });
  return <RouterProvider router={router} future={ROUTER_PROVIDER_FUTURE} />;
}

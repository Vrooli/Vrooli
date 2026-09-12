import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
} from "react-router-dom";

import { routes } from "./routesConfig";
import { dataRouterFutureFlags, routerProviderFutureFlags } from "./routerFuture";

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production. The route *table* lives in `./routesConfig` so this file only
 * exports components (keeps fast-refresh happy).
 */
export function AppRouter() {
  // Re-create per mount so HMR / re-mounts pick up updated routes during dev
  // and so tests that manipulate `window.history` see fresh routing each time.
  const router = createBrowserRouter(routes, { future: dataRouterFutureFlags });
  return <RouterProvider router={router} future={routerProviderFutureFlags} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx` and the a11y tests.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, {
    initialEntries,
    future: dataRouterFutureFlags,
  });
  return <RouterProvider router={router} future={routerProviderFutureFlags} />;
}

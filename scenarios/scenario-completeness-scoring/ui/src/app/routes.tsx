import { createBrowserRouter, createMemoryRouter, RouterProvider } from "react-router-dom";

import { routes } from "./routeTable";

/**
 * Opt in to the v7 behaviors now so react-router stops emitting future-flag
 * warnings (the strict test console guard treats those as failures).
 */
const routerFuture = { v7_relativeSplatPath: true } as const;
const providerFuture = { v7_startTransition: true } as const;

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

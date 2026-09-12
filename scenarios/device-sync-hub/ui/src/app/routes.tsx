import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { TransferPage } from "../pages/TransferPage";
import { DevicesPage } from "../pages/DevicesPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. The index route is the split-screen Transfer surface;
 * Devices and Settings are secondary. Exported so tests construct an in-memory
 * router from the same config the production app uses.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <TransferPage /> },
      { path: "devices", element: <DevicesPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

/**
 * Production router (uses real browser history). Built lazily so module load
 * doesn't fail in test environments where `window.location` semantics differ
 * from production.
 */
// Opt into the v7 transition/relative-splat behaviour now so the router doesn't
// emit future-flag console warnings (which our test harness treats as failures)
// and so the eventual v7 upgrade is a no-op. For data routers, `v7_startTransition`
// is read off the <RouterProvider> `future` prop, while `v7_relativeSplatPath`
// is read off the router's own `future` config — so both placements are needed.
const ROUTER_FUTURE = { v7_relativeSplatPath: true } as const;
const PROVIDER_FUTURE = { v7_startTransition: true } as const;

export function AppRouter() {
  const router = createBrowserRouter(routes, { future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={PROVIDER_FUTURE} />;
}

/**
 * Test helper: render the same routes against an in-memory router with a
 * specific starting URL. Only used by `routes.test.tsx`.
 */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: ROUTER_FUTURE });
  return <RouterProvider router={router} future={PROVIDER_FUTURE} />;
}

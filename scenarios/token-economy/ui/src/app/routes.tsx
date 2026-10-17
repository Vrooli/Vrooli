import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { ApprovalsPage } from "../features/redemption/ApprovalsPage";
import { CatalogPage } from "../features/catalog/CatalogPage";
import { EarningPage } from "../features/earning/EarningPage";
import { GrantsPage } from "../features/grants/GrantsPage";
import { HoldersPage } from "../features/holders/HoldersPage";
import { HolderHistoryPage } from "../features/holders/HolderHistoryPage";
import { HolderHomePage } from "../features/holders/HolderHomePage";
import { HolderRewardsPage } from "../features/holders/HolderRewardsPage";
import { HolderShell } from "../features/holders/HolderShell";
import { JournalPage } from "../features/journal/JournalPage";
import { MintsPage } from "../features/mints/MintsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";

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
      { path: "tokens", element: <MintsPage /> },
      { path: "holders", element: <HoldersPage /> },
      { path: "earning", element: <EarningPage /> },
      { path: "grants", element: <GrantsPage /> },
      { path: "catalog", element: <CatalogPage /> },
      { path: "approvals", element: <ApprovalsPage /> },
      { path: "journal", element: <JournalPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
  {
    path: "/me",
    element: <HolderShell />,
    children: [
      { index: true, element: <HolderHomePage /> },
      { path: "history", element: <HolderHistoryPage /> },
      { path: "rewards", element: <HolderRewardsPage /> },
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
  return <RouterProvider router={router} future={routerProviderFuture} />;
}

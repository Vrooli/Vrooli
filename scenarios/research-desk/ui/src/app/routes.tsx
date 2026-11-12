import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { CommissionPage } from "../pages/CommissionPage";
import { CommissionsPage } from "../pages/CommissionsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { DeskPage } from "../pages/DeskPage";
import { FilePage } from "../pages/FilePage";
import { LaunchPage } from "../pages/LaunchPage";
import { LaunchStatusPage } from "../pages/LaunchStatusPage";
import { NeedsYouPage } from "../pages/NeedsYouPage";
import { ReviewPage } from "../pages/ReviewPage";
import { ReviewsPage } from "../pages/ReviewsPage";
import { TopicPage } from "../pages/TopicPage";
import { TopicsPage } from "../pages/TopicsPage";
import { ThemeProvider } from "../theme/ThemeProvider";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * The navigation (layout/navItems.tsx) holds the top-level destinations;
 * the rest are drill-ins from them.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "needs-you", element: <NeedsYouPage /> },
      { path: "topics", element: <TopicsPage /> },
      { path: "topics/:topic", element: <TopicPage /> },
      { path: "commissions", element: <CommissionsPage /> },
      { path: "commissions/:id", element: <CommissionPage /> },
      { path: "launch", element: <LaunchPage /> },
      { path: "launch/:id", element: <LaunchStatusPage /> },
      { path: "reviews", element: <ReviewsPage /> },
      { path: "reviews/:id", element: <ReviewPage /> },
      { path: "files/*", element: <FilePage /> },
      { path: "desk", element: <DeskPage /> },
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

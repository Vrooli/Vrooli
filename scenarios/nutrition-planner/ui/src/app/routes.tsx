import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { Navigate } from "react-router-dom";
import { SettingsPage } from "../pages/SettingsPage";
import { OnboardingPage } from "../features/onboarding/OnboardingPage";
import { TodayPage } from "../features/today/TodayPage";
import { WeekPage } from "../features/week/WeekPage";
import { GroceriesPage } from "../features/groceries/GroceriesPage";
import { NutritionPage } from "../features/nutrition/NutritionPage";
import { MealsPage } from "../features/meals/MealsPage";
import { ExplorePage } from "../features/explore/ExplorePage";
import { KitchenPage } from "../features/kitchen/KitchenPage";
import { DataTransferPage } from "../features/transfer/DataTransferPage";
import { RecipeDetailPage } from "../features/recipe/RecipeDetailPage";
import { CookSessionPage } from "../features/recipe/CookSessionPage";
import { ThemeProvider } from "../theme/ThemeProvider";

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
      { index: true, element: <TodayPage /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "setup", element: <OnboardingPage /> },
      { path: "today", element: <Navigate to="/" replace /> },
      { path: "recipes/:id/revisions/:revision", element: <RecipeDetailPage /> },
      { path: "cook/:sessionId", element: <CookSessionPage /> },
      { path: "week", element: <WeekPage /> },
      { path: "groceries", element: <GroceriesPage /> },
      { path: "meals", element: <MealsPage /> },
      { path: "meals/explore", element: <ExplorePage /> },
      { path: "kitchen", element: <KitchenPage /> },
      { path: "nutrition", element: <NutritionPage /> },
      { path: "transfer", element: <DataTransferPage /> },
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

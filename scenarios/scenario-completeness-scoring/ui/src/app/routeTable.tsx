import type { RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { NotesPage } from "../pages/NotesPage";
import { SettingsPage } from "../pages/SettingsPage";

/**
 * Canonical route table. Lives apart from the router components so this
 * file exports only data and `routes.tsx` exports only components
 * (react-refresh constraint). Tests build an in-memory router from the
 * same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "notes", element: <NotesPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

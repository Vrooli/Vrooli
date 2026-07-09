import type { RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { NotesPage } from "../pages/NotesPage"; // EXAMPLE-DOMAIN:notes
import { SettingsPage } from "../pages/SettingsPage";

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "notes", element: <NotesPage /> }, // EXAMPLE-DOMAIN:notes
      { path: "settings", element: <SettingsPage /> },
    ],
  },
];

export const routerOptions = {
  future: {
    v7_relativeSplatPath: true,
  },
} as const;

export const routerProviderFuture = {
  v7_startTransition: true,
} as const;

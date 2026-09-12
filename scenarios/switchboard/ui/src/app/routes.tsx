import { createBrowserRouter, createMemoryRouter, Navigate, RouterProvider, type RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { AgentDetailPage } from "../pages/AgentDetailPage";
import { AgentNewPage } from "../pages/AgentNewPage";
import { AgentsPage } from "../pages/AgentsPage";
import { CallPage } from "../pages/CallPage";
import { ChannelsPage } from "../pages/ChannelsPage";
import { ContactsPage } from "../pages/ContactsPage";
import { ConversationsPage } from "../pages/ConversationsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { ThemeProvider } from "../theme/ThemeProvider";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses. Every route here is a page
 * the experience contract declares; there are no placeholder surfaces.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "welcome", element: <Navigate to="/agents/new" replace /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "channels", element: <ChannelsPage /> },
      { path: "channels/:channelId", element: <ChannelsPage /> },
      { path: "agents", element: <AgentsPage /> },
      { path: "agents/new", element: <AgentNewPage /> },
      { path: "agents/:agentId", element: <AgentDetailPage /> },
      { path: "conversations", element: <ConversationsPage /> },
      { path: "conversations/:threadId", element: <ConversationsPage /> },
      { path: "call/:threadId", element: <CallPage /> },
      { path: "contacts", element: <ContactsPage /> },
      { path: "contacts/:contactId", element: <ContactsPage /> },
      { path: "*", element: <Navigate to="/" replace /> },
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

/** Production router (uses real browser history). */
export function AppRouter() {
  const router = createBrowserRouter(routes, { future: dataRouterFuture });
  return <RouterProvider router={router} future={routerProviderFuture} />;
}

/** Test helper: the same routes against an in-memory router. */
export function TestAppRouter({ initialEntries }: { initialEntries: string[] }) {
  const router = createMemoryRouter(routes, { initialEntries, future: dataRouterFuture });
  return (
    <ThemeProvider>
      <RouterProvider router={router} future={routerProviderFuture} />
    </ThemeProvider>
  );
}

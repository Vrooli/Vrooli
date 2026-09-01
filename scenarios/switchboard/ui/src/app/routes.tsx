import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
  type RouteObject,
} from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { SettingsPage } from "../pages/SettingsPage";
import { ChannelsPage } from "../pages/ChannelsPage";
import { AgentsPage } from "../pages/AgentsPage";
import { ConversationsPage } from "../pages/ConversationsPage";
import { ContactsPage } from "../pages/ContactsPage";
import { ThemeProvider } from "../theme/ThemeProvider";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExperienceSurface, type ExperienceSurfaceState } from "../components/experience/ExperienceSurface";

type SurfaceKey = (typeof strings.console.surface)[keyof typeof strings.console.surface];

function SurfacePage({ titleKey, region, descriptionKey, state = "ready" }: { titleKey: SurfaceKey; region: string; descriptionKey: SurfaceKey; state?: ExperienceSurfaceState }) {
  const { t } = useTranslation();
  return (
    <section aria-labelledby={`${region}-heading`} className="flex flex-col gap-4">
      <h2 id={`${region}-heading`} className="text-2xl font-semibold">{t(titleKey)}</h2>
      <p className="text-app-muted-foreground">{t(descriptionKey)}</p>
      <ExperienceSurface surfaceId={region} state={state} className="rounded-lg border p-6">
        {t(strings.console.surface.nothingConfigured)}
      </ExperienceSurface>
    </section>
  );
}

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
      { path: "welcome", element: <SurfacePage titleKey={strings.console.surface.welcome} region="draft-region" descriptionKey={strings.console.surface.welcomeDescription} /> },
      { path: "settings", element: <SettingsPage /> },
      { path: "channels", element: <ChannelsPage /> },
      { path: "channels/:channelId", element: <SurfacePage titleKey={strings.console.surface.channel} region="catalog-region" descriptionKey={strings.console.surface.channelDescription} state="empty" /> },
      { path: "agents", element: <AgentsPage /> },
      { path: "agents/:agentId", element: <SurfacePage titleKey={strings.console.surface.agent} region="grant-region" descriptionKey={strings.console.surface.agentDescription} /> },
      { path: "agents/new", element: <AgentsPage /> },
      { path: "conversations", element: <ConversationsPage /> },
      { path: "conversations/:threadId", element: <ConversationsPage /> },
      { path: "call/:threadId", element: <SurfacePage titleKey={strings.console.surface.call} region="transcript-region" descriptionKey={strings.console.surface.callDescription} /> },
      { path: "contacts", element: <ContactsPage /> },
      { path: "contacts/:contactId", element: <SurfacePage titleKey={strings.console.surface.contact} region="contact-region" descriptionKey={strings.console.surface.contactDescription} state="empty" /> },
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
  return <ThemeProvider><RouterProvider router={router} future={routerProviderFuture} /></ThemeProvider>;
}

import { Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { BottomNav } from "./BottomNav";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Responsive app shell. CSS grid with a header row and a (sidebar | content)
 * main row; mobile collapses the sidebar to a pinned bottom nav.
 *
 * This is the real shell — replaces the centered single-card placeholder.
 * Page content renders into the `<Outlet />`; routes are configured in
 * `app/routes.tsx`.
 */
export function AppShell() {
  const { t } = useTranslation();

  return (
    <div
      data-testid={selectors.layout.shell}
      className="flex min-h-full flex-col bg-app-background text-app-foreground"
    >
      <TopBar />
      <div className="flex min-h-0 flex-1">
        <Sidebar />
        <main
          data-testid={selectors.layout.main}
          aria-label={t(strings.layout.mainLabel)}
          className="min-w-0 flex-1 overflow-auto p-6"
        >
          <Outlet />
        </main>
      </div>
      <BottomNav />
    </div>
  );
}

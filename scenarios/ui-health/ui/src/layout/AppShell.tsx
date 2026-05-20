/**
 * AppShell — operational console shell.
 *
 * Desktop (≥ md): left sidebar (brand + primary nav), top bar (locale +
 * health + theme), main content. Pages can mount an `<InspectorPanel>` of
 * their own; the shell does not own one because it is feature-specific.
 *
 * Mobile (< md): sticky header (drawer trigger + brand + health pill +
 * theme), main content with bottom padding to clear the bottom nav, slide-in
 * drawer for full nav access, bottom nav for the most common destinations.
 */
import { useCallback, useState } from "react";
import { Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BottomNav } from "./BottomNav";
import { MobileDrawer } from "./MobileDrawer";
import { MobileHeader } from "./MobileHeader";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

export function AppShell() {
  const { t } = useTranslation();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const closeDrawer = useCallback(() => setDrawerOpen(false), []);
  const openDrawer = useCallback(() => setDrawerOpen(true), []);

  return (
    <div
      data-testid={selectors.layout.shell}
      className="flex min-h-screen w-full bg-app-background text-app-foreground"
    >
      <a
        href="#main-content"
        data-testid={selectors.layout.skipToContent}
        className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-50 focus:rounded-control focus:bg-app-primary focus:px-3 focus:py-2 focus:text-app-primary-foreground"
      >
        {t(strings.layout.skipToContent)}
      </a>

      <Sidebar />

      <div className="flex min-w-0 flex-1 flex-col">
        <MobileHeader onOpenDrawer={openDrawer} />
        <TopBar />
        <main
          id="main-content"
          data-testid={selectors.layout.main}
          aria-label={t(strings.layout.main)}
          className="pb-safe min-w-0 flex-1 overflow-auto px-4 py-4 pb-24 md:px-8 md:py-6 md:pb-8"
        >
          <Outlet />
        </main>
        <BottomNav />
      </div>

      <MobileDrawer open={drawerOpen} onClose={closeDrawer} />
    </div>
  );
}

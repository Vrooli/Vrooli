/**
 * AppShell — full-width operational shell.
 *
 * Desktop (≥ md): resizable left sidebar + main content. The sidebar is
 * driven by `useResizablePanel` so widths persist across reloads.
 * Mobile (< md): top header, bottom navigation, and a slide-in drawer
 * that hosts the same nav + component list. The dark/light theme is mirrored
 * on `<html>` by the surrounding `<ThemeProvider>`.
 *
 * Replaces the starter centered-card layout: no `max-w-xl`, no eyebrow,
 * no card wrapping page-level content.
 */
import { type ReactNode, useCallback, useRef, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { SidebarShell } from "../../../library/components/SidebarShell/versions/1.0.0/SidebarShell";
import { useIsMobile } from "../hooks/useMediaQuery";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useTranslation } from "../i18n";
import { Settings as SettingsIcon } from "lucide-react";
import { Link } from "react-router-dom";
import { ActiveWorkMenu } from "./ActiveWorkMenu";
import { MobileHeader } from "./MobileHeader";
import { SidebarContent } from "./Sidebar";
import { CatalogBrowser } from "../features/catalog/CatalogBrowser";

const SIDEBAR_STORAGE = "react-component-library.sidebar.width.v1";

interface Props {
  children?: ReactNode;
}

export function AppShell({ children }: Props) {
  const { t } = useTranslation();
  const location = useLocation();
  const shellRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLDivElement>(null);
  const isMobile = useIsMobile();
  const [drawerOpen, setDrawerOpen] = useState(false);

  const { size: sidebarWidth, resizeHandleProps } = useResizablePanel({
    containerRef: shellRef,
    targetRef: sidebarRef,
    minSize: 260,
    maxSize: 480,
    defaultSize: 300,
    adjacentMinSize: 420,
    handleWidth: 6,
    storageKey: SIDEBAR_STORAGE,
  });

  const closeDrawer = useCallback(() => setDrawerOpen(false), []);
  const openDrawer = useCallback(() => setDrawerOpen(true), []);

  const headerSlot = <div className="flex items-center gap-1"><ActiveWorkMenu /><Link to="/settings" aria-label={t("nav.settings", { defaultValue: "Settings" })} className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><SettingsIcon aria-hidden className="h-4 w-4" /></Link></div>;
  const isComponentDetail = /^\/assets\/[^/]+/.test(location.pathname);

  return (
    <div
      ref={shellRef}
      data-testid="app-shell"
      className="flex h-dvh min-h-0 w-full overflow-hidden bg-app-background text-app-foreground"
    >
      <SidebarShell
        ref={sidebarRef}
        mobileOpen={drawerOpen}
        onMobileClose={closeDrawer}
        mobileLabel={t("nav.drawerLabel", { defaultValue: "Navigation drawer" })}
        desktopLabel={t("nav.label", { defaultValue: "Primary navigation" })}
        closeLabel={t("nav.closeDrawer", { defaultValue: "Close navigation" })}
        mobileHeader={
          <span className="truncate text-sm font-semibold text-app-foreground">
            {t("app.brand", { defaultValue: "Component Library" })}
          </span>
        }
        width={isMobile ? undefined : sidebarWidth}
        resizeHandleProps={isMobile ? undefined : resizeHandleProps}
        contentClassName="flex"
      >
        <SidebarContent
          onNavigate={closeDrawer}
          headerSlot={headerSlot}
          inventorySlot={<CatalogBrowser compact onNavigate={closeDrawer} />}
        />
      </SidebarShell>

      <div className="flex min-w-0 flex-1 flex-col">
        <MobileHeader onOpenDrawer={openDrawer} />
        <main
          data-testid="app-main"
          className={
            isComponentDetail
              ? "pb-safe flex min-h-0 min-w-0 flex-1 flex-col overflow-auto p-0 pb-20 md:pb-0"
              : "pb-safe min-w-0 flex-1 overflow-auto px-4 py-4 pb-20 md:px-8 md:py-6 md:pb-8"
          }
        >
          {children ?? <Outlet />}
        </main>
      </div>
    </div>
  );
}

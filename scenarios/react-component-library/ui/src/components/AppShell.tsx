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
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { SidebarShell } from "../../../library/components/SidebarShell/versions/1.1.0/SidebarShell";
import { WorkspaceHeader } from "../../../library/components/WorkspaceHeader/versions/1.0.0/WorkspaceHeader";
import { useIsMobile } from "../hooks/useMediaQuery";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useTranslation } from "../i18n";
import { Menu, Settings as SettingsIcon } from "lucide-react";
import { Link } from "react-router-dom";
import { SidebarContent } from "./Sidebar";
import { ShellNavigationContext } from "./ShellNavigationContext";
import { CatalogBrowser } from "../features/catalog/CatalogBrowser";
import { CreateComponentDialog } from "../features/components/CreateComponentDialog";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

const SIDEBAR_STORAGE = "react-component-library.sidebar.width.v1";

interface Props {
  children?: ReactNode;
}

export function AppShell({ children }: Props) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const shellRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLDivElement>(null);
  const isMobile = useIsMobile();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [desktopSidebarCollapsed, setDesktopSidebarCollapsed] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [search, setSearch] = useState("");

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
  const sidebarCollapsed = isMobile ? !drawerOpen : desktopSidebarCollapsed;
  const openSidebar = useCallback(() => {
    if (isMobile) openDrawer();
    else setDesktopSidebarCollapsed(false);
  }, [isMobile, openDrawer]);

  const isComponentDetail = /^\/assets\/[^/]+/.test(location.pathname);
  const pageTitle = isComponentDetail ? t("catalog.title", { defaultValue: "Component Library" }) : location.pathname === "/settings" ? t("settings.title", { defaultValue: "Settings" }) : location.pathname === "/catalog" ? t("catalog.title", { defaultValue: "Library workspace" }) : t("app.brand", { defaultValue: "Component Library" });
  const pageDescription = isComponentDetail ? t("components.editor.subtitle", { defaultValue: "Source, preview, and viewport controls" }) : location.pathname === "/settings" ? t("settings.subtitle", { defaultValue: "Theme and locale preferences persist locally in your browser." }) : location.pathname === "/catalog" ? t("catalog.subtitle", { defaultValue: "Find reusable components and non-renderable hooks." }) : t("dashboard.subtitle", { defaultValue: "A clear view of your library's adoption and maintenance health." });

  return (
    <ShellNavigationContext.Provider value={{ sidebarCollapsed, openSidebar }}>
    <div
      ref={shellRef}
      data-testid="app-shell"
      className="flex h-dvh min-h-0 w-full overflow-hidden bg-app-background text-app-foreground"
    >
      <SidebarShell
        ref={sidebarRef}
        mobileOpen={drawerOpen}
        desktopCollapsed={desktopSidebarCollapsed}
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
          onCollapse={() => setDesktopSidebarCollapsed(true)}
          headerSlot={<Link to="/settings" aria-label={t("nav.settings", { defaultValue: "Settings" })} className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><SettingsIcon aria-hidden className="h-4 w-4" /></Link>}
          inventorySlot={<CatalogBrowser compact onNavigate={closeDrawer} />}
        />
      </SidebarShell>

      <div className="flex min-w-0 flex-1 flex-col">
        {!isComponentDetail && <WorkspaceHeader
          title={pageTitle}
          description={pageDescription}
          leading={sidebarCollapsed ? <button type="button" onClick={openSidebar} aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })} data-testid="workspace-header-open-sidebar" className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><Menu aria-hidden className="h-5 w-5" /></button> : undefined}
          actions={location.pathname !== "/settings" ? <><form onSubmit={(event) => { event.preventDefault(); navigate(`/catalog${search.trim() ? `?q=${encodeURIComponent(search.trim())}` : ""}`); }} className="hidden sm:block"><Input aria-label={t("catalog.search", { defaultValue: "Search catalog" })} value={search} onChange={(event) => setSearch(event.target.value)} placeholder={t("catalog.search", { defaultValue: "Search" })} className="h-9 w-44" /></form><span className="hidden text-xs text-app-muted-foreground md:inline">{t("dashboard.synced", { defaultValue: "Synced" })}</span><Button size="sm" onClick={() => setShowCreate(true)}>{t("dashboard.create", { defaultValue: "Create" })}</Button></> : undefined}
        />}
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
      {showCreate && <CreateComponentDialog onClose={() => setShowCreate(false)} />}
    </div>
    </ShellNavigationContext.Provider>
  );
}

/** @vrooliComponentSource react-component-library:AppShell */
import { type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { AppShell as LibraryAppShell } from "./AppShell/versions/1.0.0/AppShell";
import { SidebarShell } from "./SidebarShell";
import { WorkspaceHeader } from "./WorkspaceHeader";
import { useIsMobile } from "../hooks/useMediaQuery";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useTranslation } from "../i18n";
import { Menu, Settings as SettingsIcon } from "lucide-react";
import { Link } from "react-router-dom";
import { SidebarContent } from "./Sidebar";
import { ShellNavigationContext } from "./ShellNavigationContext";
import { CatalogBrowser } from "../features/catalog/CatalogBrowser";
import { CreateComponentDialog } from "../features/components/CreateComponentDialog";
import { Button } from "./Button";
import { Input } from "./Input";
import { ActionLauncher, type LauncherAction } from "./ActionLauncher";

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
  const [launcherAction, setLauncherAction] = useState<LauncherAction>(null);
  const [launcherAssetID, setLauncherAssetID] = useState("");
  const [launcherTarget, setLauncherTarget] = useState("");
  const [search, setSearch] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const requested = params.get("action");
    if (requested === "create") {
      setShowCreate(true);
      setLauncherAction(null);
      return;
    }
    if (requested === "extract" || requested === "adopt") {
      setLauncherAssetID(params.get("assetId") ?? "");
      setLauncherTarget(params.get("targetScenario") ?? "");
      setLauncherAction(requested);
    }
  }, [location.search]);

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
  const isCatalog = location.pathname === "/" || location.pathname === "/catalog";
  const pageTitle = isComponentDetail
    ? t("catalog.title", { defaultValue: "Component Library" })
    : location.pathname === "/settings"
      ? t("settings.title", { defaultValue: "Settings" })
      : location.pathname === "/coverage"
        ? "Catalog coverage"
        : location.pathname === "/capabilities"
          ? "Capability readiness"
          : isCatalog
            ? t("catalog.title", { defaultValue: "Library workspace" })
            : t("app.brand", { defaultValue: "Component Library" });
  const pageDescription = isComponentDetail
    ? t("components.editor.subtitle", { defaultValue: "Source, preview, and viewport controls" })
    : location.pathname === "/settings"
      ? t("settings.subtitle", {
          defaultValue: "Theme and locale preferences persist locally in your browser.",
        })
      : location.pathname === "/coverage"
        ? "Maturity distribution and ranked next work"
        : location.pathname === "/capabilities"
          ? "Integration readiness and recovery guidance"
          : isCatalog
            ? t("catalog.subtitle", {
                defaultValue: "Find reusable components and non-renderable hooks.",
              })
            : t("app.brand", { defaultValue: "Component Library" });

  const navigation = (
    <SidebarShell
      ref={sidebarRef}
      mode={desktopSidebarCollapsed && !isMobile ? "overlay" : "responsive"}
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
      width={isMobile || desktopSidebarCollapsed ? undefined : sidebarWidth}
      resizeHandleProps={isMobile ? undefined : resizeHandleProps}
      contentClassName="flex min-w-0"
    >
      <SidebarContent
        onNavigate={closeDrawer}
        onCollapse={() => setDesktopSidebarCollapsed(true)}
        headerSlot={
          <Link
            to="/settings"
            aria-label={t("nav.settings", { defaultValue: "Settings" })}
            className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            <SettingsIcon aria-hidden className="h-4 w-4" />
          </Link>
        }
        inventorySlot={<CatalogBrowser compact onNavigate={closeDrawer} />}
      />
    </SidebarShell>
  );

  const header = isComponentDetail ? null : (
    <WorkspaceHeader
      as="div"
      title={pageTitle}
      description={pageDescription}
      leading={
        sidebarCollapsed ? (
          <button
            type="button"
            onClick={openSidebar}
            aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
            data-testid="workspace-header-open-sidebar"
            className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            <Menu aria-hidden className="h-5 w-5" />
          </button>
        ) : undefined
      }
      actions={
        location.pathname !== "/settings" ? (
          <>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                navigate(
                  `/catalog${search.trim() ? `?q=${encodeURIComponent(search.trim())}` : ""}`,
                );
              }}
              className="hidden sm:block"
            >
              <Input
                aria-label={t("catalog.search", { defaultValue: "Search catalog" })}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t("catalog.search", { defaultValue: "Search" })}
                className="h-9 w-44"
              />
            </form>
            <span className="hidden text-xs text-app-muted-foreground md:inline">
              {t("dashboard.synced", { defaultValue: "Synced" })}
            </span>
            <Button size="sm" onClick={() => setShowCreate(true)}>
              {t("dashboard.create", { defaultValue: "Create" })}
            </Button>
          </>
        ) : undefined
      }
    />
  );

  return (
    <ShellNavigationContext.Provider value={{ sidebarCollapsed, openSidebar }}>
      <LibraryAppShell
        className="h-dvh min-h-0 w-full overflow-hidden"
        navigation={navigation}
        navigationMode="managed"
        navigationLabel={t("nav.label", { defaultValue: "Primary navigation" })}
        header={header}
        headerMode={isComponentDetail ? "hidden" : "visible"}
        mainMode={isComponentDetail ? "flush" : "padded"}
        mainClassName={
          isComponentDetail
            ? "pb-safe flex min-h-0 min-w-0 w-full max-w-full flex-1 flex-col overflow-auto pb-20 md:pb-0"
            : "pb-safe min-h-0 min-w-0 w-full max-w-full flex-1 overflow-auto pb-20 md:pb-0"
        }
      >
        {children ?? <Outlet />}
      </LibraryAppShell>
      {showCreate && <CreateComponentDialog onClose={() => setShowCreate(false)} />}
      <ActionLauncher
        action={launcherAction}
        onActionChange={setLauncherAction}
        onCreate={() => setShowCreate(true)}
        showTrigger={!isComponentDetail}
        initialAssetID={launcherAssetID}
        initialTarget={launcherTarget}
      />
    </ShellNavigationContext.Provider>
  );
}

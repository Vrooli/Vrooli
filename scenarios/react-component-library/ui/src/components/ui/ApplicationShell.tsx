import { type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { AppShell as LibraryAppShell } from "./AppShell/versions/1.0.0/AppShell";
import { SidebarShell } from "./SidebarShell/versions/1.2.0/SidebarShell";
import { BottomNav } from "./BottomNav/versions/1.3.0/BottomNav";
import { WorkspaceHeader } from "../WorkspaceHeader";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { useTranslation } from "../../i18n";
import { BarChart3, FolderTree, Menu, Settings as SettingsIcon, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { SidebarContent } from "../Sidebar";
import { ShellNavigationContext } from "../ShellNavigationContext";
import { CatalogBrowser } from "../../features/catalog/CatalogBrowser";
import { CreateComponentDialog } from "../../features/components/CreateComponentDialog";
import { Button } from "../Button";
import { IconButton } from "../IconButton";
import { Input } from "../Input";
import { ActionLauncher, type LauncherAction } from "../ActionLauncher";

const SIDEBAR_STORAGE = "react-component-library.sidebar.width.v2";

interface Props {
  children?: ReactNode;
}

export function ApplicationShell({ children }: Props) {
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
    minSize: 280,
    maxSize: 480,
    defaultSize: 340,
    adjacentMinSize: 440,
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
            <SettingsIcon aria-hidden className="h-icon-sm w-icon-sm" />
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
          /* The third copy of this control (see MobileHeader and Sidebar) now
             composes IconButton, so the sidebar re-open affordance inherits the
             shared control treatment instead of a local class list. */
          <IconButton
            onClick={openSidebar}
            aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
            data-testid="workspace-header-open-sidebar"
          >
            <Menu aria-hidden className="h-icon-md w-icon-md" />
          </IconButton>
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
                className="h-control-sm w-field-wide"
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
      <div ref={shellRef} className="h-dvh min-h-0 w-full">
        <LibraryAppShell
          className="h-full min-h-0 w-full overflow-hidden"
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
      </div>
      {isMobile ? (
        <BottomNav
          label={t("nav.label", { defaultValue: "Primary navigation" })}
          items={[
            {
              id: "catalog",
              label: "Catalog",
              href: "/catalog",
              active: isCatalog,
              icon: <FolderTree aria-hidden />,
            },
            {
              id: "coverage",
              label: "Coverage",
              href: "/coverage",
              active: location.pathname === "/coverage",
              icon: <BarChart3 aria-hidden />,
            },
            {
              id: "capabilities",
              label: "Capabilities",
              href: "/capabilities",
              active: location.pathname === "/capabilities",
              icon: <Sparkles aria-hidden />,
            },
          ]}
          onItemSelect={(item) => navigate(item.href ?? "/")}
        />
      ) : null}
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

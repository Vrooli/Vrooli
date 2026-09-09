import { type ReactNode, useCallback, useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { WorkspaceHeader } from "@vrooli/react-component-library/WorkspaceHeader/1";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { useTranslation } from "../../i18n";
import { BarChart3, FolderTree, Menu, Settings as SettingsIcon, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { ShellNavigationContext } from "../ShellNavigationContext";
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
  const isMobile = useIsMobile();
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

  const sidebarCollapsed = desktopSidebarCollapsed;
  const openSidebar = useCallback(() => setDesktopSidebarCollapsed(false), []);

  const isComponentDetail = /^\/assets\/[^/]+/.test(location.pathname);
  const isCatalog = ["/", "/catalog", "/components"].includes(location.pathname);
  const isDesign = location.pathname.startsWith("/design");
  const pageTitle = isDesign
    ? t("design.title")
    : isComponentDetail
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
  const pageDescription = isDesign
    ? t("design.subtitle")
    : isComponentDetail
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

  const header = isComponentDetail ? undefined : (
    <WorkspaceHeader
      as="div"
      title={pageTitle}
      description={pageDescription}
      leading={
        sidebarCollapsed ? (
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
        !isDesign && location.pathname !== "/settings" ? (
          <>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void navigate(
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
        brand={t("app.brand", { defaultValue: "Component Library" })}
        brandMark={<FolderTree aria-hidden />}
        brandHref="/"
        items={[
          { id: "design", label: t("design.title"), href: "/design", icon: <Sparkles aria-hidden />, current: isDesign },
          { id: "catalog", label: "Browse assets", shortLabel: "Assets", href: "/catalog", icon: <FolderTree aria-hidden />, current: isCatalog || isComponentDetail || location.pathname === "/components" },
          { id: "coverage", label: "Catalog coverage", shortLabel: "Coverage", href: "/coverage", icon: <BarChart3 aria-hidden />, current: location.pathname === "/coverage" },
          { id: "capabilities", label: "Capability readiness", shortLabel: "Capabilities", href: "/capabilities", icon: <Sparkles aria-hidden />, current: location.pathname === "/capabilities" },
        ]}
        renderLink={(_, { href, ...props }) => <Link {...props} to={href} />}
        onNavigate={(item) => { void navigate(item.href); }}
        navigationLabel={t("nav.label", { defaultValue: "Primary navigation" })}
        mobileNavigationLabel={t("nav.label", { defaultValue: "Primary navigation" })}
        density={desktopSidebarCollapsed ? "rail" : "sidebar"}
        mobileNav="tabs"
        sidebarStorageKey={SIDEBAR_STORAGE}
        className="h-dvh min-h-0 w-full overflow-hidden"
        header={header}
        mainMode={isComponentDetail ? "fill" : "scroll"}
        mainClassName={isComponentDetail ? "min-h-0 min-w-0 w-full flex flex-1 flex-col overflow-auto" : "min-h-0 min-w-0 w-full flex-1"}
        utility={
          <div className="flex items-center gap-space-2xs">
            <Link to="/settings" aria-label={t("nav.settings", { defaultValue: "Settings" })} className="touch-target inline-flex items-center justify-center rounded-control">
              <SettingsIcon aria-hidden className="h-icon-sm w-icon-sm" />
            </Link>
            {!isMobile && <IconButton aria-label={sidebarCollapsed ? "Expand navigation" : "Collapse navigation"} data-testid="sidebar-collapse" onClick={() => setDesktopSidebarCollapsed((value) => !value)}><Menu aria-hidden /></IconButton>}
          </div>
        }
      >
        {children ?? <Outlet />}
      </LibraryAppShell>
      {showCreate && <CreateComponentDialog onClose={() => setShowCreate(false)} />}
      <ActionLauncher
        action={launcherAction}
        onActionChange={setLauncherAction}
        onCreate={() => setShowCreate(true)}
        showTrigger={!isComponentDetail && !isDesign}
        initialAssetID={launcherAssetID}
        initialTarget={launcherTarget}
      />
    </ShellNavigationContext.Provider>
  );
}

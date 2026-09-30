import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from "@vrooli/react-component-library/AppShell/2";
import { Activity, Boxes, GaugeCircle, Layers, Settings as SettingsIcon } from "lucide-react";
import { NavLink, Outlet, useLocation } from "react-router-dom";

import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";
import { ErrorBoundary } from "./ErrorBoundary";
import { HealthPill } from "./HealthPill";
import { SidebarFlowList } from "./SidebarFlowList";
import { ThemeToggle } from "./ThemeToggle";

const SIDEBAR_STORAGE = "flow-verifier.sidebar.width.v1";

export function AppShell() {
  const { t } = useTranslation();
  const location = useLocation();

  const items: AppShellNavItem[] = [
    {
      id: "dashboard",
      label: t("nav.dashboard", { defaultValue: "Dashboard" }),
      href: ROUTES.dashboard,
      icon: <GaugeCircle aria-hidden className="h-4 w-4" />,
      current: location.pathname === ROUTES.dashboard,
      testId: "nav-dashboard",
    },
    {
      id: "scenarios",
      label: t("nav.scenarios", { defaultValue: "Scenarios" }),
      href: ROUTES.scenarios,
      icon: <Boxes aria-hidden className="h-4 w-4" />,
      current: location.pathname.startsWith("/scenarios"),
      testId: "nav-scenarios",
    },
    {
      id: "flows",
      label: t("nav.flows", { defaultValue: "All flows" }),
      href: ROUTES.flowsInventory,
      icon: <Layers aria-hidden className="h-4 w-4" />,
      current: location.pathname.startsWith("/flows"),
      testId: "nav-flows",
    },
    {
      id: "settings",
      label: t("nav.settings", { defaultValue: "Settings" }),
      href: ROUTES.settings,
      icon: <SettingsIcon aria-hidden className="h-4 w-4" />,
      current: location.pathname.startsWith("/settings"),
      testId: "nav-settings",
    },
  ];

  const renderLink = (item: AppShellNavItem, props: AppShellLinkProps) => (
    <NavLink
      to={item.href}
      className={props.className}
      aria-current={props["aria-current"]}
      aria-disabled={props["aria-disabled"]}
      data-testid={props["data-testid"]}
      onClick={props.onClick}
    >
      {props.children}
    </NavLink>
  );

  return (
    <LibraryAppShell
      brand={t("app.brand", { defaultValue: "Flow Studio" })}
      brandMark={<Activity aria-hidden className="h-4 w-4" />}
      brandHref={ROUTES.dashboard}
      items={items}
      renderLink={renderLink}
      density="sidebar"
      mobileNav="drawer"
      header={
        <div className="flex items-center gap-1">
          <HealthPill />
          <ThemeToggle />
        </div>
      }
      utility={<SidebarFlowList />}
      sidebarStorageKey={SIDEBAR_STORAGE}
      navigationLabel={t("nav.label", { defaultValue: "Primary navigation" })}
      mobileNavigationLabel={t("nav.label", { defaultValue: "Primary navigation" })}
      skipLabel={t("a11y.skipToMain", { defaultValue: "Skip to main content" })}
      menuLabel={t("nav.open", { defaultValue: "Open navigation" })}
      closeLabel={t("nav.close", { defaultValue: "Close navigation" })}
      testId="flow-verifier-app-shell"
    >
      <ErrorBoundary>
        <Outlet />
      </ErrorBoundary>
    </LibraryAppShell>
  );
}

import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import type { ReactNode } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import type { AppRoute } from "../app/routeDefinitions";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { routeDefinitions } from "../app/routeDefinitions";

export function AppShell({
  activeRoute,
  children,
  onNavigate
}: {
  activeRoute: AppRoute;
  children: ReactNode;
  onNavigate: (routeKey: AppRoute) => void;
}) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();

  return (
    <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
      brand={<div><span data-testid={selectors.app.title}>{t(strings.app.title)}</span><span className="sr-only">{t(strings.app.description)}</span></div>}
      brandHref="/"
      items={routeDefinitions.map((item) => ({ id: item.key, href: item.path, label: t(item.label), icon: <item.icon size={16} aria-hidden />, current: activeRoute === item.key || pathname === item.path, testId: selectors.layout.navLink(item.key) }))}
      renderLink={(item, { children: linkChildren, ...props }) => <button type="button" onClick={() => { onNavigate(item.id as AppRoute); navigate(item.href); }} {...props}>{linkChildren}</button>}
      onNavigate={(item) => onNavigate(item.id as AppRoute)}
      navigationLabel={strings.layout.sidebarLabel}
      mobileNavigationLabel={strings.layout.bottomNavLabel}
      sidebarStorageKey="scenario-dependency-analyzer.sidebar-width"
      testId={selectors.layout.shell}
    >
      <div className="flex flex-col gap-6"><Outlet />{children}</div>
    </LibraryAppShell>
  );
}

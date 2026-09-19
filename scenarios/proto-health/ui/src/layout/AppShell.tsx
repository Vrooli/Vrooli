import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { LayoutDashboard, Settings } from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { ShellControls } from "../components/ShellControls";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandHref="/"
      items={NAV_ITEMS.map(item => ({
        id: item.key, href: item.path, label: t(item.labelKey),
        icon: item.key === "dashboard" ? <LayoutDashboard aria-hidden="true" /> : <Settings aria-hidden="true" />,
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(item.path + "/"),
        testId: selectors.layout.sidebarLink({ key: item.key }),
      }))}
      utility={<ShellControls />}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find(entry => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={item => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      skipLabel={t(strings.layout.mainLabel)}
      sidebarStorageKey="proto-health.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

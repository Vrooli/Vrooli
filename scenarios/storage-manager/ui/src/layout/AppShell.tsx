import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { ThemeControl } from "../components/ThemeControl";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, iconForItem } from "./navItems";

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
        id: item.key, href: item.path, label: t(item.labelKey), icon: iconForItem(item),
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(item.path + "/"),
        testId: selectors.layout.navLink({ key: item.key }),
      }))}
      utility={<ThemeControl />}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find(entry => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={item => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarNavLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="storage-manager.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

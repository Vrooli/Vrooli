import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { ShellUtility } from "../components/ShellUtility";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

const MOBILE_KEYS = new Set(["dashboard", "validation", "search", "inventory", "reindex"]);

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.brand)}</span>}
      brandMark={<span aria-hidden>{t(strings.app.brandInitials)}</span>}
      brandHref="/"
      items={NAV_ITEMS.map(({ key, path, labelKey, icon: Icon, end }) => ({
        id: key, href: path, label: t(labelKey), icon: <Icon aria-hidden />,
        current: end ? pathname === path : pathname === path || pathname.startsWith(path + "/"),
        mobile: MOBILE_KEYS.has(key), testId: selectors.layout.navLink({ key }),
      }))}
      utility={<ShellUtility />}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find(entry => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={item => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      skipLabel={t(strings.layout.skipToContent)}
      sidebarStorageKey="ui-health.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

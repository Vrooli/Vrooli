import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { AppShell as LibraryAppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useAttention } from "../features/health/useAttention";
import { ShellUtility } from "../components/console/ShellUtility";
import { BrandMark } from "./BrandMark";
import { NAV_ITEMS, isNavItemActive, navIcon } from "./navItems";

// The library owns chrome; Switchboard configures a conversation workspace.
const SHELL = { density: "rail", mobileNav: "tabs", mainMode: "fill" } as const;

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const attention = useAttention();
  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.key,
    href: item.path,
    label: t(item.labelKey),
    shortLabel: t(item.shortLabelKey ?? item.labelKey),
    icon: navIcon(item.key),
    current: isNavItemActive(item, pathname),
    mobile: item.mobile,
    testId: selectors.layout.navLink({ key: item.key }),
    badge: item.key === "dashboard" && attention.pending > 0
      ? { value: attention.pending, tone: "warning", label: t(strings.console.attention.pendingCount, { count: attention.pending }) }
      : undefined,
  }));

  return (
    <LibraryAppShell
      {...SHELL}
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandMark={<BrandMark />}
      brandHref="/"
      items={items}
      utility={<ShellUtility />}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      skipLabel={t(strings.layout.skipToContent)}
      sidebarStorageKey="switchboard.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

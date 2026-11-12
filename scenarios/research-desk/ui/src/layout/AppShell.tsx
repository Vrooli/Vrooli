import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { AppShell as LibraryAppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";
import { useNeedsYou } from "../api/operator";
import { NAV_ITEMS, isNavItemActive } from "./navItems";

/**
 * The shell is the component library's; this file configures it and plugs in
 * the router. The desk is a phone-first lens: four tabs for what the operator
 * checks most (Now, Needs you, Topics, Commissions), with Launch, Reviews and
 * the Desk in the phone menu (Launch is also the first button on Now). Needs
 * you carries a count badge so waiting decisions are visible from every screen.
 */
const SHELL = {
  density: "sidebar",
  mobileNav: "menu-tabs",
  mainMode: "scroll",
} as const;

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const waiting = useNeedsYou().data?.items.filter((item) => !item.answer).length ?? 0;

  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.key,
    label: t(item.labelKey),
    href: item.path,
    icon: item.icon,
    current: isNavItemActive(item, pathname),
    mobile: item.mobile,
    badge: item.key === "needsYou" && waiting > 0 ? { value: waiting, tone: "warning", label: t(strings.layout.waiting, { count: waiting }) } : undefined,
    testId: selectors.layout.navLink({ key: item.key }),
  }));

  return (
    <LibraryAppShell
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandMark={<BrandMark />}
      brandHref="/"
      items={items}
      density={SHELL.density}
      mobileNav={SHELL.mobileNav}
      mainMode={SHELL.mainMode}
      renderLink={(item, { href, children, onClick, ...rest }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end === true} onClick={onClick} {...rest}>
          {children}
        </NavLink>
      )}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.navigationLabel)}
      mobileNavigationLabel={t(strings.layout.mobileNavigationLabel)}
      skipLabel={t(strings.layout.skipToContent)}
      menuLabel={t(strings.layout.openNavigation)}
      closeLabel={t(strings.layout.closeNavigation)}
      sidebarStorageKey="research-desk.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

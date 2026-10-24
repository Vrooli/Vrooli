import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { AppShell as LibraryAppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { BrandMark } from "./BrandMark";
import { NAV_ITEMS, isNavItemActive } from "./navItems";

/**
 * The shell is the component library's. This file configures it and plugs in
 * the router; it does not draw chrome.
 *
 * Decide these three settings in Gate 5 of `docs/START-HERE.md` and change
 * them here. Nothing else in the tree needs to know.
 *
 * - `density`: `"sidebar"` (icon + label, resizable) for a tool with several
 *   peer surfaces; `"rail"` (icon over a short label, narrow) when one surface
 *   needs the width.
 * - `mobileNav`: `"tabs"` for three to five destinations; `"drawer"` for more.
 * - `mainMode`: `"scroll"` pads and scrolls pages for you; `"fill"` hands a
 *   page the whole pane so it can pin its own header and composer.
 *
 * If the shell cannot do what your primary surface needs, do not fork it:
 * record the gap in `docs/reference/component-library-gaps.md` and eject with
 * `react-component-library adoptions eject --reason`.
 */
const SHELL = {
  density: "sidebar",
  mobileNav: "tabs",
  mainMode: "scroll",
} as const;

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();

  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.key,
    label: t(item.labelKey),
    href: item.path,
    icon: item.icon,
    current: isNavItemActive(item, pathname),
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
      sidebarStorageKey="{{SCENARIO_ID}}.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

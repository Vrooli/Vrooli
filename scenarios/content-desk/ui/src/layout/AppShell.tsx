import { Home, Settings } from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];

/**
 * Responsive app shell. CSS grid with a header row and a (sidebar | content)
 * main row; mobile collapses the sidebar to a pinned bottom nav.
 *
 * This is the real shell — replaces the centered single-card placeholder.
 * Page content renders into the `<Outlet />`; routes are configured in
 * `app/routes.tsx`.
 */
export function AppShell() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
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
        id: item.key,
        href: item.path,
        label: t(item.labelKey),
        icon: item.key === "settings" ? <Settings aria-hidden className="h-5 w-5" /> : <Home aria-hidden className="h-5 w-5" />,
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(item.path + "/"),
        testId: selectors.layout.sidebarLink({ key: item.key }),
      }))}
      utility={(
        <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
          <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
          <select
            value={choice}
            onChange={event => setTheme(event.target.value as ThemeChoice)}
            data-testid={selectors.theme.select}
            aria-label={t(strings.theme.switcherLabel)}
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
          >
            {THEME_CHOICES.map(theme => <option key={theme} value={theme}>{t(strings.theme.choice[theme])}</option>)}
          </select>
        </label>
      )}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find(entry => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={item => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="content-desk.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

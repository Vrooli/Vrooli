import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { LumeMark } from "../components/ui/lume-mark";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { THEME_CHOICES, THEME_CHOICE_LABEL } from "../theme/themeChoiceLabel";
import { NAV_ITEMS } from "./navItems";

function ShellUtility() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();

  return (
    <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
      <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
      <select
        value={choice}
        onChange={(event) => setTheme(event.target.value as ThemeChoice)}
        data-testid={selectors.theme.select}
        aria-label={t(strings.theme.switcherLabel)}
        className="rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
      >
        {THEME_CHOICES.map((theme) => (
          <option key={theme} value={theme}>{t(THEME_CHOICE_LABEL[theme])}</option>
        ))}
      </select>
    </label>
  );
}

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={<span data-testid={selectors.app.title} className="flex items-center gap-2"><LumeMark size={22} />{t(strings.app.title)}</span>}
      brandHref="/"
      items={NAV_ITEMS.map((item) => {
        const Icon = item.icon;
        return {
          id: item.key,
          href: item.path,
          label: t(item.labelKey),
          icon: <Icon size={16} aria-hidden />,
          current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`),
          testId: selectors.layout.navLink({ key: item.key }),
        };
      })}
      utility={<ShellUtility />}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>
      )}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="image-tools.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

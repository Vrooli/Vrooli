import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme } from "../theme/ThemeProvider";
import { NAV_ITEMS } from "./navItems";

const THEME_CHOICES = ["light", "dark", "system"] as const;

function LocaleUtility() {
  const { t } = useTranslation();
  const current = getCurrentLocale();
  return <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex gap-1">
    {SUPPORTED_LOCALES.map((code) => <button key={code} type="button" data-testid={selectors.locale.toggle({ code })} onClick={() => void setLocale(code)} aria-pressed={current === code}>{getLocaleConfig(code).nativeLabel}</button>)}
  </div>;
}
function ThemeUtility() {
  const { t } = useTranslation();
  const { choice, setTheme } = useTheme();
  return <div className="flex items-center gap-2">
    <LocaleUtility />
    <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
      <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
      <select value={choice} onChange={(event) => setTheme(event.target.value as (typeof THEME_CHOICES)[number])} data-testid={selectors.theme.select} aria-label={t(strings.theme.switcherLabel)} className="touch-target rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground">
        {THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(strings.theme.choice[theme])}</option>)}
      </select>
    </label>
  </div>;
}

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
    brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>} brandHref="/"
    items={NAV_ITEMS.map((item) => ({ id: item.key, href: item.path, label: t(item.labelKey), icon: <item.icon size={16} aria-hidden />, current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`), testId: selectors.layout.navLink({ key: item.key }) }))}
    utility={<ThemeUtility />}
    renderLink={(item, { href, children, ...props }) => <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>}
    onNavigate={(item) => navigate(item.href)} navigationLabel={t(strings.layout.sidebarLabel)} mobileNavigationLabel={t(strings.layout.bottomNavLabel)} sidebarStorageKey="vrooli-bridge.sidebar-width" testId={selectors.layout.shell}>
    <Outlet />
  </LibraryAppShell>;
}

import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { NAV_ITEMS } from "./navItems";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const THEME_LABELS = { light: strings.theme.choice.light, dark: strings.theme.choice.dark, system: strings.theme.choice.system } satisfies Record<ThemeChoice, string>;

function ShellUtility() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  return <div className="flex items-center gap-3">
    <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex gap-1">
      {SUPPORTED_LOCALES.map((locale) => <button key={locale} type="button" data-testid={selectors.locale.toggle({ code: locale })} onClick={() => void setLocale(locale)} aria-pressed={currentLocale === locale}>{getLocaleConfig(locale).nativeLabel}</button>)}
    </div>
    <label data-testid={selectors.theme.switcher}><span className="sr-only">{t(strings.theme.switcherLabel)}</span><select value={choice} onChange={(event) => setTheme(event.target.value as ThemeChoice)} data-testid={selectors.theme.select} aria-label={t(strings.theme.switcherLabel)}>{THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(THEME_LABELS[theme])}</option>)}</select></label>
  </div>;
}

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
    brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>} brandHref="/"
    items={NAV_ITEMS.map((item) => { const Icon = item.icon; return { id: item.key, href: item.path, label: t(item.labelKey), icon: <Icon size={16} aria-hidden />, current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`), testId: selectors.layout.navLink({ key: item.key }) }; })}
    utility={<ShellUtility />}
    renderLink={(item, { href, children, ...props }) => <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>}
    onNavigate={(item) => navigate(item.href)} navigationLabel={t(strings.layout.sidebarLabel)} mobileNavigationLabel={t(strings.layout.bottomNavLabel)} sidebarStorageKey="network-manager.sidebar-width" testId={selectors.layout.shell}>
    <Outlet />
  </LibraryAppShell>;
}

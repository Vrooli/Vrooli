import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

function LocaleUtility() {
  const { t } = useTranslation();
  const current = getCurrentLocale();
  return <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex gap-1">
    {SUPPORTED_LOCALES.map((locale) => <button key={locale} type="button" data-testid={selectors.locale.toggle({ code: locale })} onClick={() => void setLocale(locale)} aria-pressed={current === locale}>{getLocaleConfig(locale).nativeLabel}</button>)}
  </div>;
}

export function AppShell() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return <LibraryAppShell density="sidebar" mobileNav="tabs" mainMode="scroll"
    brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>} brandHref="/"
    items={NAV_ITEMS.map((item) => ({ id: item.key, href: item.path, label: t(item.labelKey), icon: <item.icon size={16} aria-hidden />, current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`), testId: selectors.layout.navLink({ key: item.key }) }))}
    utility={<LocaleUtility />}
    renderLink={(item, { href, children, ...props }) => <NavLink to={href} end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>}
    onNavigate={(item) => navigate(item.href)} navigationLabel={t(strings.layout.sidebarLabel)} mobileNavigationLabel={t(strings.layout.bottomNavLabel)} sidebarStorageKey="measures-health.sidebar-width" testId={selectors.layout.shell}>
    <Outlet />
  </LibraryAppShell>;
}

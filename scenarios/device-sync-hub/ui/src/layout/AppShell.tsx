import { ArrowLeftRight, HardDrive, Settings } from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { THEME_CHOICE_LABEL } from "../theme/themeStrings";
import { useRealtimeContext } from "../features/realtime/RealtimeProvider";
import { NAV_ITEMS } from "./navItems";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
const REALTIME_LABEL = {
  connecting: strings.realtime.connecting,
  open: strings.realtime.open,
  closed: strings.realtime.closed,
} as const;
const NAV_ICONS = {
  transfer: <ArrowLeftRight aria-hidden className="h-5 w-5" />,
  devices: <HardDrive aria-hidden className="h-5 w-5" />,
  settings: <Settings aria-hidden className="h-5 w-5" />,
} as const;

/** Library-owned shell adapter for the Device Sync Hub console. */
export function AppShell() {
  const { t } = useTranslation();
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const { status } = useRealtimeContext();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const statusDot = status === "open" ? "bg-app-success" : status === "connecting" ? "bg-app-warning" : "bg-app-muted-foreground";

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="fill"
      brand={<span data-testid={selectors.app.title}>{t(strings.app.title)}</span>}
      brandHref="/"
      items={NAV_ITEMS.map((item) => ({
        id: item.key,
        href: item.path,
        label: t(item.labelKey),
        icon: NAV_ICONS[item.key],
        current: item.end ? pathname === item.path : pathname === item.path || pathname.startsWith(`${item.path}/`),
        testId: selectors.layout.sidebarLink({ key: item.key }),
      }))}
      utility={(
        <div className="flex max-w-full flex-wrap items-center gap-2">
          <span data-testid={selectors.realtime.indicator} className="flex items-center gap-1.5 text-xs text-app-muted-foreground" aria-label={t(strings.realtime.label)} title={t(REALTIME_LABEL[status])}>
            <span className={`inline-block h-2 w-2 rounded-pill ${statusDot}`} aria-hidden="true" />
            <span className="hidden sm:inline">{t(REALTIME_LABEL[status])}</span>
          </span>
          <div role="group" aria-label={t(strings.locale.switcherLabel)} data-testid={selectors.locale.switcher} className="flex items-center gap-0.5 rounded-control border border-app-border bg-app-surface-muted p-0.5 text-xs">
            {SUPPORTED_LOCALES.map((locale) => (
              <button key={locale} type="button" data-testid={selectors.locale.toggle({ code: locale })} onClick={() => void setLocale(locale)} aria-pressed={currentLocale === locale} className={currentLocale === locale ? "min-h-11 min-w-11 rounded-control bg-app-primary px-2 py-1 font-medium text-app-primary-foreground" : "min-h-11 min-w-11 rounded-control px-2 py-1 text-app-muted-foreground hover:text-app-foreground"}>{getLocaleConfig(locale).nativeLabel}</button>
            ))}
          </div>
          <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
            <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
            <select value={choice} onChange={(event) => setTheme(event.target.value as ThemeChoice)} data-testid={selectors.theme.select} aria-label={t(strings.theme.switcherLabel)} className="min-h-11 rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground">
              {THEME_CHOICES.map((theme) => <option key={theme} value={theme}>{t(THEME_CHOICE_LABEL[theme])}</option>)}
            </select>
          </label>
        </div>
      )}
      renderLink={(item, { href, children, ...props }) => <NavLink to={href} end={NAV_ITEMS.find((entry) => entry.key === item.id)?.end} {...props}>{children}</NavLink>}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="device-sync-hub.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

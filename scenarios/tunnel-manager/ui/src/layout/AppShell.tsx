import { useQuery } from "@tanstack/react-query";
import {
  BarChart3,
  FileClock,
  GitCompareArrows,
  Globe2,
  LayoutDashboard,
  RotateCcw,
  Settings,
} from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

import { fetchHealth } from "../api/health";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { getCurrentLocale, getLocaleConfig, setLocale, SUPPORTED_LOCALES, useTranslation } from "../i18n";
import { StatusBadge } from "../components/ui/StatusBadge";
import type { ThemeChoice } from "../theme/themeContext";
import { THEME_CHOICE_LABEL } from "../theme/themeChoiceLabels";
import { useTheme } from "../theme/useTheme";
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
  const currentLocale = getCurrentLocale();
  const { choice, setTheme } = useTheme();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const healthQuery = useQuery({
    queryKey: ["api-health"],
    queryFn: fetchHealth,
    staleTime: 15_000,
    retry: 1,
  });
  const apiStatus = healthQuery.data?.status ?? (healthQuery.isError ? "unreachable" : "checking");
  const apiTone = healthQuery.isError ? "danger" : apiStatus === "healthy" ? "success" : "warning";
  const apiStatusLabel = apiStatus === "healthy"
    ? strings.health.statusHealthy
    : apiStatus === "unreachable"
      ? strings.health.statusUnreachable
      : strings.health.statusChecking;

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      brand={(
        <span className="flex min-w-0 flex-col">
          <span data-testid={selectors.app.title} className="truncate text-base font-semibold tracking-tight sm:text-lg">
            {t(strings.app.title)}
          </span>
          <span className="truncate text-[10px] font-medium uppercase tracking-[0.16em] text-app-muted-foreground">
            {t(strings.app.tagline)}
          </span>
        </span>
      )}
      brandHref="/"
      items={NAV_ITEMS.map((item) => ({
        id: item.key,
        href: item.path,
        label: t(item.labelKey),
        icon: iconForItem(item.key),
        current: item.end
          ? pathname === item.path
          : pathname === item.path || pathname.startsWith(`${item.path}/`),
        testId: selectors.layout.sidebarLink({ key: item.key }),
      }))}
      utility={(
        <div className="flex max-w-full flex-wrap items-center gap-2">
          <button
            type="button"
            className="rounded-pill focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            aria-label={`${t(strings.health.statusLabel)} ${t(apiStatusLabel)}`}
            onClick={() => void healthQuery.refetch()}
          >
            <StatusBadge tone={apiTone} className="min-h-11 items-center px-3" data-testid={selectors.health.statusBadge}>
              <span className="mr-1.5 inline-block h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
              {t(apiStatusLabel)}
            </StatusBadge>
          </button>
          <div
            role="group"
            aria-label={t(strings.locale.switcherLabel)}
            data-testid={selectors.locale.switcher}
            className="flex items-center gap-0.5 rounded-control border border-app-border bg-app-surface-muted p-0.5 text-xs"
          >
            {SUPPORTED_LOCALES.map((locale) => (
              <button
                key={locale}
                type="button"
                data-testid={selectors.locale.toggle({ code: locale })}
                onClick={() => void setLocale(locale)}
                aria-pressed={currentLocale === locale}
                className={currentLocale === locale
                  ? "min-h-11 min-w-11 rounded-control bg-app-primary px-2 py-1 font-medium text-app-primary-foreground"
                  : "min-h-11 min-w-11 rounded-control px-1.5 py-1 text-app-muted-foreground hover:text-app-foreground"}
              >
                {getLocaleConfig(locale).nativeLabel}
              </button>
            ))}
          </div>
          <label data-testid={selectors.theme.switcher} className="flex items-center gap-2 text-xs text-app-muted-foreground">
            <span className="sr-only">{t(strings.theme.switcherLabel)}</span>
            <select
              value={choice}
              onChange={(event) => setTheme(event.target.value as ThemeChoice)}
              data-testid={selectors.theme.select}
              aria-label={t(strings.theme.switcherLabel)}
              className="min-h-11 max-w-[7rem] rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground sm:max-w-none"
            >
              {THEME_CHOICES.map((theme) => (
                <option key={theme} value={theme}>{t(THEME_CHOICE_LABEL[theme])}</option>
              ))}
            </select>
          </label>
        </div>
      )}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink
          to={href}
          end={item.id === "brand" || NAV_ITEMS.find((entry) => entry.key === item.id)?.end}
          {...props}
        >
          {children}
        </NavLink>
      )}
      onNavigate={(item) => navigate(item.href)}
      navigationLabel={t(strings.layout.sidebarLabel)}
      mobileNavigationLabel={t(strings.layout.bottomNavLabel)}
      sidebarStorageKey="tunnel-manager.sidebar-width"
      testId={selectors.layout.shell}
    >
      <Outlet />
    </LibraryAppShell>
  );
}

function iconForItem(key: (typeof NAV_ITEMS)[number]["key"]) {
  const className = "h-5 w-5";
  switch (key) {
    case "overview": return <LayoutDashboard aria-hidden className={className} />;
    case "exposure": return <Globe2 aria-hidden className={className} />;
    case "recovery": return <RotateCcw aria-hidden className={className} />;
    case "metrics": return <BarChart3 aria-hidden className={className} />;
    case "audit": return <FileClock aria-hidden className={className} />;
    case "drift": return <GitCompareArrows aria-hidden className={className} />;
    case "settings": return <Settings aria-hidden className={className} />;
  }
}

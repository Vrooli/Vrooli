import { NavLink, useLocation } from "react-router-dom";
import { Target, Wrench, FileCog, Settings as SettingsIcon, ChevronsLeft, ChevronsRight } from "lucide-react";
import type { ReactNode } from "react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES, ROUTE_PATTERNS } from "../../routes.generated";
import { usePreferencesStore } from "../stores/preferencesStore";
import { cn } from "../lib/utils";

interface NavItem {
  to: string;
  patternPrefix: string;
  label: string;
  testId: string;
  icon: typeof Target;
}

// Inline literal references to nav label keys so the unused-keys lint
// rule sees a callsite per key. The values resolve at render time via
// the registry; here we just pin the key paths.
const NAV_LABEL_KEYS = [
  strings.nav.goldensLabel,
  strings.nav.skillsLabel,
  strings.nav.manifestsLabel,
  strings.nav.settingsLabel,
] as const;

/**
 * Persistent desktop sidebar. Hidden on mobile (the AppShell decides).
 * Collapse toggle persists via `preferencesStore`.
 */
export function Sidebar(): ReactNode {
  const { t } = useTranslation();
  const collapsed = usePreferencesStore((s) => s.sidebarCollapsed);
  const toggleSidebar = usePreferencesStore((s) => s.toggleSidebar);
  const location = useLocation();

  const navItems: readonly NavItem[] = [
    {
      to: ROUTES.goldensIndex,
      patternPrefix: ROUTE_PATTERNS.goldensIndex,
      label: t(NAV_LABEL_KEYS[0]),
      testId: selectors.nav.sidebarItemGoldens,
      icon: Target,
    },
    {
      to: ROUTES.skillsIndex,
      patternPrefix: ROUTE_PATTERNS.skillsIndex,
      label: t(NAV_LABEL_KEYS[1]),
      testId: selectors.nav.sidebarItemSkills,
      icon: Wrench,
    },
    {
      to: ROUTES.manifestsIndex,
      patternPrefix: ROUTE_PATTERNS.manifestsIndex,
      label: t(NAV_LABEL_KEYS[2]),
      testId: selectors.nav.sidebarItemManifests,
      icon: FileCog,
    },
    {
      to: ROUTES.settings,
      patternPrefix: ROUTE_PATTERNS.settings,
      label: t(NAV_LABEL_KEYS[3]),
      testId: selectors.nav.sidebarItemSettings,
      icon: SettingsIcon,
    },
  ];

  return (
    <aside
      data-testid={selectors.nav.sidebar}
      data-collapsed={collapsed}
      className={cn(
        "flex h-full shrink-0 flex-col border-r border-app-border bg-app-shell text-app-foreground transition-[width] duration-default",
        collapsed ? "w-16" : "w-60",
      )}
    >
      <div className="flex items-center gap-2 px-3 py-4">
        <span
          data-testid={selectors.nav.sidebarLogo}
          className="grid h-8 w-8 shrink-0 place-items-center rounded-control bg-app-accent text-xs font-bold text-app-primary-foreground"
        >
          {t(strings.nav.logo)}
        </span>
        {collapsed ? null : (
          <span className="truncate text-sm font-semibold text-app-foreground">
            {t(strings.app.eyebrow)}
          </span>
        )}
      </div>
      <nav className="flex-1 px-2">
        <ul className="flex flex-col gap-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive =
              item.patternPrefix === "/"
                ? location.pathname === "/"
                : location.pathname.startsWith(item.patternPrefix);
            return (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  data-testid={item.testId}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "flex items-center gap-3 rounded-control px-2 py-2 text-sm transition-colors",
                    isActive
                      ? "bg-app-accent/15 text-app-accent"
                      : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                  )}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  {collapsed ? null : <span className="truncate">{item.label}</span>}
                </NavLink>
              </li>
            );
          })}
        </ul>
      </nav>
      <div className="border-t border-app-border p-2">
        <button
          type="button"
          data-testid={selectors.nav.sidebarCollapseToggle}
          aria-label={t(strings.nav.menuToggle)}
          aria-pressed={collapsed}
          onClick={toggleSidebar}
          className="flex w-full items-center justify-center gap-2 rounded-control px-2 py-2 text-xs text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
        >
          {collapsed ? <ChevronsRight className="h-4 w-4" /> : <ChevronsLeft className="h-4 w-4" />}
        </button>
      </div>
    </aside>
  );
}

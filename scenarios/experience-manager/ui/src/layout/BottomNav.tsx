import {
  BarChart3,
  FileSearch,
  Gauge,
  Home,
  Settings,
  Wand2,
  type LucideIcon,
} from "lucide-react";
import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, type NavItem } from "./navItems";

const NAV_ICONS: Record<NavItem["key"], LucideIcon> = {
  fleet: Home,
  explorer: BarChart3,
  evidence: FileSearch,
  studio: Wand2,
  findings: Gauge,
  settings: Settings,
};

/**
 * Mobile bottom nav. Visible below the `md` breakpoint; on desktop, see
 * `Sidebar`. Same nav targets as `NAV_ITEMS`, rendered as a flex row pinned to
 * the viewport bottom.
 */
export function BottomNav() {
  const { t } = useTranslation();

  return (
    <nav
      data-testid={selectors.layout.bottomNav}
      aria-label={`${t(strings.layout.bottomNavLabel)} mobile`}
      className="fixed inset-x-0 bottom-0 z-40 flex items-stretch justify-around border-t border-app-border bg-app-surface/95 pb-[calc(max(env(safe-area-inset-bottom),24px)+1rem)] pt-2 shadow-lg shadow-black/10 backdrop-blur md:hidden"
    >
      {NAV_ITEMS.map((item) => {
        const Icon = NAV_ICONS[item.key];
        return (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.end}
            data-testid={selectors.layout.bottomNavLink({ key: item.key })}
            className={({ isActive }) =>
              [
                "flex min-h-12 min-w-0 flex-1 flex-col items-center justify-center gap-1 px-1 text-[0.68rem] font-medium leading-none",
                isActive ? "text-app-primary" : "text-app-muted-foreground",
              ].join(" ")
            }
          >
            <Icon className="size-5 shrink-0" aria-hidden="true" />
            <span className="block max-w-full truncate whitespace-nowrap">{t(item.labelKey)}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}

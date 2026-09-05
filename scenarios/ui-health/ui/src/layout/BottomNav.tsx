/**
 * BottomNav — pinned bottom navigation for mobile viewports. Mirrors the
 * primary destinations of `NAV_ITEMS` so the most common surfaces are
 * one tap away. Less-frequently-used destinations live in the drawer.
 */
import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS, type NavKey } from "./navItems";

const BOTTOM_KEYS: readonly NavKey[] = ["dashboard", "validation", "search", "inventory", "reindex"];

export function BottomNav() {
  const { t } = useTranslation();
  const items = NAV_ITEMS.filter((i) => (BOTTOM_KEYS as readonly string[]).includes(i.key));

  return (
    <nav
      data-testid={selectors.layout.bottomNav}
      aria-label={t(strings.layout.bottomNavLabel)}
      className="pb-safe fixed inset-x-0 bottom-0 z-30 flex border-t border-app-border bg-app-surface md:hidden"
    >
      {items.map((item) => {
        const Icon = item.icon;
        return (
          <NavLink
            key={item.key}
            to={item.path}
            end={item.end}
            data-testid={selectors.layout.bottomNavLink({ key: item.key })}
            className={({ isActive }) =>
              [
                "flex min-h-touch flex-1 flex-col items-center justify-center gap-0.5 py-2 text-[10px] font-medium",
                isActive ? "text-app-primary" : "text-app-muted-foreground hover:text-app-foreground",
              ].join(" ")
            }
          >
            <Icon aria-hidden className="h-5 w-5" />
            <span>{t(item.labelKey)}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}

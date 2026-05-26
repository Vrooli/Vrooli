import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

/**
 * Desktop sidebar. Hidden below the `md` breakpoint; on mobile, see
 * `BottomNav`.
 *
 * Keeps no internal state — the active link is driven by react-router. Replace
 * the nav items in `./navItems.ts` when this scenario's routes change.
 */
export function Sidebar() {
  const { t } = useTranslation();

  return (
    <nav
      data-testid={selectors.layout.sidebar}
      aria-label={t(strings.layout.sidebarLabel)}
      className="hidden h-full w-56 shrink-0 flex-col gap-1 border-r border-app-border bg-app-surface p-4 md:flex"
    >
      <p className="px-2 pb-2 text-xs uppercase tracking-wide text-app-muted-foreground">
        {t(strings.layout.sidebarLabel)}
      </p>
      {NAV_ITEMS.map((item) => {
        const Icon = item.icon;
        return (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.end}
            data-testid={selectors.layout.sidebarLink({ key: item.key })}
            className={({ isActive }) =>
              [
                "flex items-center gap-2.5 rounded-control px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-app-primary text-app-primary-foreground"
                  : "text-app-foreground hover:bg-app-surface-muted",
              ].join(" ")
            }
          >
            <Icon aria-hidden="true" className="h-4 w-4 shrink-0" />
            {t(item.labelKey)}
          </NavLink>
        );
      })}
    </nav>
  );
}

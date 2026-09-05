import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

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
      aria-label={t(strings.layout.bottomNavLabel)}
      className="flex shrink-0 items-stretch justify-around border-t border-app-border bg-app-surface pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      {NAV_ITEMS.map((item) => {
        const Icon = item.icon;
        return (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.end}
            aria-label={t(item.labelKey)}
            data-testid={selectors.layout.bottomNavLink({ key: item.key })}
            className={({ isActive }) =>
              [
                "flex min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-1 py-2",
                isActive ? "text-app-primary" : "text-app-muted-foreground",
              ].join(" ")
            }
          >
            <Icon aria-hidden="true" className="h-5 w-5 shrink-0" />
            <span className="w-full truncate text-center text-[10px] font-medium leading-tight">
              {t(item.labelKey)}
            </span>
          </NavLink>
        );
      })}
    </nav>
  );
}

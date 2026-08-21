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
      className="flex min-w-0 shrink-0 items-stretch justify-around overflow-hidden border-t border-app-border bg-app-surface md:hidden"
    >
      {NAV_ITEMS.map((item) => (
        <NavLink
          key={item.path}
          to={item.path}
          end={item.end}
          data-testid={selectors.layout.bottomNavLink({ key: item.key })}
          className={({ isActive }) =>
            [
              "min-w-0 flex flex-1 items-center justify-center break-words px-1 py-3 text-center text-[11px] font-medium",
              isActive ? "text-app-primary" : "text-app-muted-foreground",
            ].join(" ")
          }
        >
          {t(item.labelKey)}
        </NavLink>
      ))}
    </nav>
  );
}

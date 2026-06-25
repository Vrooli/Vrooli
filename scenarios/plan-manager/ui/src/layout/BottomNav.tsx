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
      className="flex shrink-0 items-stretch overflow-x-auto border-t border-app-border bg-app-surface md:hidden"
    >
      {NAV_ITEMS.map((item) => (
        <NavLink
          key={item.path}
          to={item.path}
          end={item.end}
          data-testid={selectors.layout.bottomNavLink({ key: item.key })}
          className={({ isActive }) =>
            [
              "flex min-w-[4.5rem] flex-1 items-center justify-center whitespace-nowrap px-3 py-3 text-xs font-medium",
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

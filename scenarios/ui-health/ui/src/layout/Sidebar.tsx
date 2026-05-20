/**
 * Sidebar — desktop primary navigation surface. Hidden below the `md`
 * breakpoint; on mobile, see `MobileDrawer` + `BottomNav`.
 */
import { Link, NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";
import { NAV_ITEMS } from "./navItems";

export function Sidebar() {
  const { t } = useTranslation();

  return (
    <aside
      data-testid={selectors.layout.sidebar}
      aria-label={t(strings.layout.sidebarLabel)}
      className="hidden h-screen w-60 shrink-0 flex-col border-r border-app-border bg-app-surface md:flex"
    >
      <div className="flex items-center gap-2 border-b border-app-border px-4 py-4">
        <Link
          to={ROUTES.dashboard}
          data-testid={selectors.app.brand}
          className="flex items-center gap-2 text-app-foreground"
        >
          <span
            aria-hidden
            className="inline-flex h-8 w-8 items-center justify-center rounded-control bg-app-primary text-sm font-semibold text-app-primary-foreground"
          >
            {t(strings.app.brandInitials)}
          </span>
          <span className="flex flex-col leading-tight">
            <span data-testid={selectors.app.title} className="text-sm font-semibold tracking-tight">
              {t(strings.app.brand)}
            </span>
            <span className="text-[10px] uppercase tracking-wide text-app-muted-foreground">
              {t(strings.app.eyebrow)}
            </span>
          </span>
        </Link>
      </div>

      <nav
        className="flex min-h-0 flex-1 flex-col gap-1 overflow-auto px-2 py-3"
        aria-label={t(strings.layout.sidebarLabel)}
      >
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.key}
              to={item.path}
              end={item.end}
              data-testid={selectors.layout.sidebarLink({ key: item.key })}
              className={({ isActive }) =>
                [
                  "flex items-center gap-2 rounded-control px-3 py-2 text-sm transition-colors",
                  isActive
                    ? "bg-app-surface-muted font-medium text-app-foreground"
                    : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                ].join(" ")
              }
            >
              <Icon aria-hidden className="h-4 w-4" />
              <span>{t(item.labelKey)}</span>
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}

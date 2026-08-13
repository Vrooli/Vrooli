/**
 * @vrooliComponentSource local:ProgramRuntimeSidebar
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption program-runtime:layout-sidebar
 * @vrooliComponentAppliedAt 2026-08-13T00:00:00Z
 *
 * Desktop-only navigation intentionally composes SidebarShell. BottomNav is
 * the governed mobile navigation counterpart; this persistent rail needs
 * vertical links and therefore keeps its own explicit local composition.
 */
import { NavLink } from "react-router-dom";

import { SidebarShell } from "../components/ui/sidebar-shell";
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
    <SidebarShell
      mode="persistent"
      mobileOpen={false}
      onMobileClose={() => undefined}
      mobileLabel={t(strings.layout.sidebarLabel)}
      desktopLabel={t(strings.layout.sidebarLabel)}
      closeLabel={t(strings.layout.sidebarLabel)}
      className="hidden w-56 md:flex"
      contentClassName="p-4"
    >
      <nav
        data-testid={selectors.layout.sidebar}
        aria-label={t(strings.layout.sidebarLabel)}
        className="flex flex-col gap-1"
      >
      <p className="px-2 pb-2 text-xs uppercase text-app-muted-foreground">
        {t(strings.layout.sidebarLabel)}
      </p>
      {NAV_ITEMS.map((item) => (
        <NavLink
          key={item.path}
          to={item.path}
          end={item.end}
          data-testid={selectors.layout.sidebarLink({ key: item.key })}
          className={({ isActive }) =>
            [
              "rounded-control px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-app-primary text-app-primary-foreground"
                : "text-app-foreground hover:bg-app-surface-muted",
            ].join(" ")
          }
        >
          {t(item.labelKey)}
        </NavLink>
      ))}
      </nav>
    </SidebarShell>
  );
}

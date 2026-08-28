/**
 * @vrooliComponentSource local:TemplateManagerSidebar
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption template-manager:layout-sidebar
 * @vrooliComponentAppliedAt 2026-07-09T00:00:00Z
 *
 * Desktop-only scenario navigation composed inside SidebarShell. The raw nav
 * markup is intentional because BottomNav governs the mobile tab bar, not the
 * persistent desktop rail.
 */
import { NavLink } from "react-router-dom";

import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/2";
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
      mobileLabel={t(strings.layout.mobileMenuLabel)}
      desktopLabel={t(strings.layout.sidebarLabel)}
      closeLabel={t(strings.layout.closeMenuLabel)}
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

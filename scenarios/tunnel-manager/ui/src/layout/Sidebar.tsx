import { NavLink } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { NAV_ITEMS } from "./navItems";

const NAV_GROUPS = ["operations", "diagnostics", "governance", "settings"] as const;
const NAV_GROUP_LABELS = {
  operations: strings.layout.navGroup.operations,
  diagnostics: strings.layout.navGroup.diagnostics,
  governance: strings.layout.navGroup.governance,
  settings: strings.layout.navGroup.settings,
} as const;

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
      {NAV_GROUPS.map((group) => (
        <div key={group} className="mt-3 first:mt-0">
          <p className="px-2 pb-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">{t(NAV_GROUP_LABELS[group])}</p>
          {NAV_ITEMS.filter((item) => item.group === group).map((item) => (
            <NavLink key={item.path} to={item.path} end={item.end} data-testid={selectors.layout.sidebarLink({ key: item.key })} className={({ isActive }) => ["rounded-control block px-3 py-2 text-sm font-medium transition-colors", isActive ? "bg-app-primary text-app-primary-foreground" : "text-app-foreground hover:bg-app-surface-muted"].join(" ")}>{t(item.labelKey)}</NavLink>
          ))}
        </div>
      ))}
    </nav>
  );
}

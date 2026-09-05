import { NavLink } from "react-router-dom";

import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/1";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useAttention } from "./useAttention";
import { navIcon } from "./navIcons";
import { NAV_ITEMS } from "./navItems";

/**
 * Desktop sidebar. The wrapper hides it below `md`; the RCL shell injects its
 * own `display:flex` rule, so the breakpoint has to live on a parent rather
 * than on the shell's className.
 */
export function Sidebar() {
  const { t } = useTranslation();
  const attention = useAttention();

  return (
    <div className="hidden min-h-0 md:flex">
      <SidebarShell
        mode="persistent"
        mobileOpen={false}
        onMobileClose={() => undefined}
        mobileLabel={t(strings.layout.sidebarLabel)}
        desktopLabel={t(strings.layout.sidebarLabel)}
        closeLabel={t(strings.layout.sidebarLabel)}
        className="w-56"
        contentClassName="flex flex-col p-3"
      >
        <nav data-testid={selectors.layout.sidebar} aria-label={t(strings.layout.sidebarLabel)} className="flex flex-1 flex-col gap-0.5">
          {NAV_ITEMS.map((item) => {
            const badge = item.key === "dashboard" ? attention.pending : item.key === "conversations" ? attention.pending : 0;
            return (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.end}
                data-testid={selectors.layout.sidebarLink({ key: item.key })}
                className={({ isActive }) =>
                  [
                    "group relative flex min-h-10 items-center gap-3 rounded-control px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-app-primary/10 text-app-primary before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-full before:bg-app-primary"
                      : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                  ].join(" ")
                }
              >
                {navIcon(item.key, "h-4.5 w-4.5 h-[18px] w-[18px] shrink-0")}
                <span className="truncate">{t(item.labelKey)}</span>
                {badge > 0 && item.key === "dashboard" ? (
                  <span
                    aria-label={t(strings.console.attention.pendingCount, { count: badge })}
                    className="ml-auto inline-flex min-w-5 items-center justify-center rounded-pill bg-app-warning px-1.5 py-0.5 font-mono text-[11px] font-semibold leading-none text-white"
                  >
                    {badge}
                  </span>
                ) : null}
              </NavLink>
            );
          })}
        </nav>
        <div className="mt-4 border-t border-app-border pt-3 text-xs text-app-muted-foreground">
          <div className="flex items-center gap-2 px-2">
            <span
              aria-hidden="true"
              className={["h-2 w-2 rounded-full", attention.apiOk === false ? "bg-app-danger" : attention.apiOk ? "bg-app-success" : "bg-app-border"].join(" ")}
            />
            <span>{attention.apiOk === false ? t(strings.console.shell.apiUnreachable) : t(strings.console.shell.apiConnected)}</span>
          </div>
        </div>
      </SidebarShell>
    </div>
  );
}

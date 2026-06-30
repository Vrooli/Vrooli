import { NavLink } from "react-router-dom";
import { cn } from "../../lib/utils";
import { NAV_ITEMS } from "./nav-items";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export function Sidebar() {
  const { t } = useTranslation();
  return (
    <aside
      className="sticky top-topbar hidden h-[calc(100%-theme(spacing.topbar))] w-sidebar shrink-0 border-r border-app-border bg-app-surface md:flex md:flex-col"
      aria-label={t(strings.shell.primaryNav)}
    >
      <nav className="flex-1 overflow-y-auto p-3">
        <ul className="flex flex-col gap-1">
          {NAV_ITEMS.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                end={item.to === "/"}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-2 rounded-control px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-app-primary/10 text-app-primary"
                      : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                  )
                }
              >
                <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                <span className="truncate">{t(item.labelKey)}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
      <footer className="border-t border-app-border p-3 text-[10px] uppercase tracking-wide text-app-muted-foreground">
        {t(strings.app.versionTag)}
      </footer>
    </aside>
  );
}

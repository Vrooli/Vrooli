import { NavLink } from "react-router-dom";
import { cn } from "../../lib/utils";
import { NAV_ITEMS } from "./nav-items";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export function MobileNav() {
  const { t } = useTranslation();
  const items = NAV_ITEMS.filter((i) => i.mobile);
  return (
    <nav
      aria-label={t(strings.shell.primaryNav)}
      className="pb-safe sticky bottom-0 z-sticky flex h-mobile-nav items-stretch border-t border-app-border bg-app-surface md:hidden"
    >
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.to === "/"}
          className={({ isActive }) =>
            cn(
              "flex flex-1 flex-col items-center justify-center gap-0.5 text-[11px] font-medium transition-colors",
              isActive ? "text-app-primary" : "text-app-muted-foreground hover:text-app-foreground",
            )
          }
        >
          <item.icon className="h-5 w-5" aria-hidden="true" />
          <span>{t(item.labelKey)}</span>
        </NavLink>
      ))}
    </nav>
  );
}

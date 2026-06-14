import type { AppRoute } from "../app/routeDefinitions";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { NAV_ITEMS } from "./navItems";

export function BottomNav({
  activeRoute,
  onNavigate
}: {
  activeRoute: AppRoute;
  onNavigate: (routeKey: AppRoute) => void;
}) {
  return (
    <nav
      aria-label={strings.layout.bottomNavLabel}
      className="fixed inset-x-0 bottom-0 z-20 grid grid-cols-4 border-t border-border/50 bg-background/95 backdrop-blur md:hidden"
      data-testid={selectors.layout.bottomNav}
    >
      {NAV_ITEMS.map((item) => (
        <button
          key={item.key}
          aria-current={activeRoute === item.key ? "page" : undefined}
          className={
            activeRoute === item.key
              ? "px-2 py-3 text-xs font-semibold text-primary"
              : "px-2 py-3 text-xs font-medium text-muted-foreground"
          }
          data-testid={selectors.layout.navLink(item.key)}
          onClick={() => onNavigate(item.key)}
          type="button"
        >
          {item.label}
        </button>
      ))}
    </nav>
  );
}

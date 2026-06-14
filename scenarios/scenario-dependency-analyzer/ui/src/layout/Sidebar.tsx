import type { AppRoute } from "../app/routeDefinitions";
import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { NAV_ITEMS } from "./navItems";

export function Sidebar({
  activeRoute,
  onNavigate
}: {
  activeRoute: AppRoute;
  onNavigate: (routeKey: AppRoute) => void;
}) {
  return (
    <nav
      aria-label={strings.layout.sidebarLabel}
      className="hidden w-64 shrink-0 flex-col gap-2 border-r border-border/50 bg-card/30 p-4 md:flex"
      data-testid={selectors.layout.sidebar}
    >
      <p className="px-2 pb-2 text-xs uppercase tracking-wide text-muted-foreground">
        {strings.layout.sidebarLabel}
      </p>
      {NAV_ITEMS.map((item) => (
        <Button
          key={item.key}
          aria-current={activeRoute === item.key ? "page" : undefined}
          className="justify-start"
          data-testid={selectors.layout.navLink(item.key)}
          onClick={() => onNavigate(item.key)}
          type="button"
          variant={activeRoute === item.key ? "default" : "ghost"}
        >
          {item.label}
        </Button>
      ))}
    </nav>
  );
}

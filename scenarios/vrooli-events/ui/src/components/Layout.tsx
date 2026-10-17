// DOC: docs/internal/EXPERIENCE-AUDIT.md#navigation-integrity
import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from "@vrooli/react-component-library/AppShell/2";
import { Heart, Activity } from "lucide-react";
import { Outlet, NavLink, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { STATUS_COLORS, HEALTH_POLL_INTERVAL_MS } from "../lib/constants";
import { NAV_ITEMS, ROUTES, type Route } from "../lib/router";
import { fetchHealth } from "../lib/api";
import { ErrorBoundary } from "./ErrorBoundary";

type HealthLevel = "healthy" | "degraded" | "unhealthy";
const HEALTH_LEVELS: readonly HealthLevel[] = ["healthy", "degraded", "unhealthy"];
const HEALTH_LEVEL_SET: ReadonlySet<string> = new Set<string>(HEALTH_LEVELS);
function isHealthLevel(s: string): s is HealthLevel { return HEALTH_LEVEL_SET.has(s); }

const ROUTE_SET: ReadonlySet<string> = new Set<string>(ROUTES);
function isRoute(s: string): s is Route { return ROUTE_SET.has(s); }

function toRoute(pathname: string): Route {
  const slug = pathname.replace(/^\//, "") || "stream";
  return isRoute(slug) ? slug : "stream";
}

export function Layout() {
  const location = useLocation();

  const currentPage = toRoute(location.pathname);

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const rawStatus = health?.status ?? "";
  const healthStatus: HealthLevel | "unknown" = isHealthLevel(rawStatus) ? rawStatus : "unknown";

  const statusColor = STATUS_COLORS[healthStatus] ?? STATUS_COLORS["unknown"];

  const items: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.id,
    href: `/${item.id}`,
    label: item.label,
    icon: <item.icon aria-hidden className="h-4 w-4" />,
    current: currentPage === item.id,
  }));
  const renderLink = (item: AppShellNavItem, { href: _href, children, ...props }: AppShellLinkProps) => (
    <NavLink to={item.href} {...props}>{children}</NavLink>
  );

  return (
    <LibraryAppShell
      brand="Vrooli Events"
      brandMark={<Activity aria-hidden className="h-4 w-4" />}
      brandHref="/stream"
      items={items}
      renderLink={renderLink}
      density="sidebar"
      mobileNav="drawer"
      mainMode="scroll"
      header={
        <div className="flex items-center gap-2 text-xs text-app-muted-foreground">
          <Heart className="h-3.5 w-3.5" aria-hidden />
          <span>System</span>
          <span role="status" className={`h-2 w-2 rounded-full ${statusColor}`} aria-label={healthStatus} />
        </div>
      }
      sidebarStorageKey="vrooli-events.sidebar-width"
      navigationLabel="Primary navigation"
      mobileNavigationLabel="Primary navigation"
      skipLabel="Skip to main content"
      menuLabel="Open navigation"
      closeLabel="Close navigation"
      testId="vrooli-events-app-shell"
    >
      <div>
        <ErrorBoundary fallbackMessage="This page encountered an error">
          <Outlet />
        </ErrorBoundary>
      </div>
    </LibraryAppShell>
  );
}

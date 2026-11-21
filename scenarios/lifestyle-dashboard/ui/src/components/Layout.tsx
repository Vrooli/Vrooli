/**
 * Main application layout with the governed shell and health utility.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Unified dashboard UI structure
 */
import { useQuery } from "@tanstack/react-query";
import { Outlet, NavLink, useLocation } from "react-router-dom";
import { Home, Heart, Activity, Settings, RefreshCw, Sun } from "lucide-react";
import { AppShell as LibraryAppShell, type AppShellNavItem } from "@vrooli/react-component-library/AppShell/2";

import { fetchHealth } from "../lib/api";

const NAV_ITEMS = [
  { id: "dashboard", path: "/", label: "Dashboard", icon: <Home aria-hidden="true" /> },
  { id: "domains", path: "/domains", label: "Domains", icon: <Heart aria-hidden="true" /> },
  { id: "events", path: "/events", label: "Events", icon: <Activity aria-hidden="true" /> },
  { id: "briefs", path: "/briefs", label: "Briefs", icon: <Sun aria-hidden="true" /> },
  { id: "settings", path: "/settings", label: "Settings", icon: <Settings aria-hidden="true" /> },
] as const;

export function Layout() {
  const location = useLocation();
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const shellItems: AppShellNavItem[] = NAV_ITEMS.map((item) => ({
    id: item.id,
    href: item.path,
    label: item.label,
    icon: item.icon,
    current: item.path === "/" ? location.pathname === "/" : location.pathname.startsWith(item.path),
  }));

  const healthLabel = healthQuery.data?.status === "ok"
    ? "Healthy"
    : healthQuery.error
      ? "Error"
      : "Checking...";

  return (
    <LibraryAppShell
      brand="Lifestyle Dashboard"
      brandMark={<Heart aria-hidden="true" />}
      items={shellItems}
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      header={(
        <div className="flex min-w-0 flex-1 items-center justify-between gap-4">
          <span className="truncate text-sm text-slate-400">Personal patterns, domains, events, and briefs</span>
          <div className="flex items-center gap-2 text-sm" role="status" aria-live="polite">
            <span className={`h-2 w-2 rounded-full ${healthQuery.data?.status === "ok" ? "bg-green-500" : healthQuery.error ? "bg-red-500" : "bg-yellow-500"}`} />
            <span className="text-slate-400">{healthLabel}</span>
          </div>
        </div>
      )}
      utility={(
        <button
          type="button"
          onClick={() => void healthQuery.refetch()}
          className="flex min-h-10 items-center gap-2 rounded-control px-2 text-sm text-slate-400 hover:bg-white/5 hover:text-white"
          title="Refresh health"
        >
          <RefreshCw className={`h-4 w-4 ${healthQuery.isFetching ? "animate-spin" : ""}`} />
          <span>Refresh</span>
        </button>
      )}
      renderLink={(item, { href, children, ...props }) => (
        <NavLink to={href} end={item.id === "dashboard"} {...props}>{children}</NavLink>
      )}
      onNavigate={() => undefined}
      sidebarStorageKey="lifestyle-dashboard.sidebar-width"
      testId="lifestyle-dashboard-shell"
      mainClassName="bg-slate-950 text-slate-50"
    >
      <div className="h-full overflow-auto">
        <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          <Outlet />
        </div>
        {healthQuery.data && (
          <footer className="mx-auto max-w-7xl px-4 pb-6 sm:px-6 lg:px-8">
            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <div className="flex flex-wrap items-center gap-4 text-sm text-slate-500">
                <span>Service: {healthQuery.data.service}</span>
                <span>•</span>
                <span>Version: {healthQuery.data.version || "1.0.0"}</span>
                <span>•</span>
                <span>Uptime: {Math.floor(healthQuery.data.uptime_seconds || 0)}s</span>
                {healthQuery.data.dependencies?.database && (
                  <>
                    <span>•</span>
                    <span>
                      Database: {healthQuery.data.dependencies.database.connected ? "connected" : "disconnected"}
                      {healthQuery.data.dependencies.database.latency_ms && (
                        <> ({healthQuery.data.dependencies.database.latency_ms.toFixed(1)}ms)</>
                      )}
                    </span>
                  </>
                )}
              </div>
            </div>
          </footer>
        )}
      </div>
    </LibraryAppShell>
  );
}

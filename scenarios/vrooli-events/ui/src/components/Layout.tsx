// DOC: docs/internal/EXPERIENCE-AUDIT.md#navigation-integrity
import { Heart } from "lucide-react";
import { Activity } from "lucide-react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { cn } from "../lib/utils";
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
  const navigate = useNavigate();

  const currentPage = toRoute(location.pathname);

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
  });

  const rawStatus = health?.status ?? "";
  const healthStatus: HealthLevel | "unknown" = isHealthLevel(rawStatus) ? rawStatus : "unknown";

  const statusColor = STATUS_COLORS[healthStatus] ?? STATUS_COLORS["unknown"];

  return (
    <div className="flex h-full bg-[var(--surface-base)] text-[var(--text-primary)]">
      <aside className="flex w-56 flex-col border-r border-[var(--border-default)] bg-[var(--surface-sidebar)]">
        <div className="flex items-center gap-2 border-b border-[var(--border-default)] px-4 py-4">
          <Activity className="h-5 w-5 text-[var(--text-accent)]" />
          <span className="text-sm font-semibold tracking-wide">Vrooli Events</span>
        </div>
        <nav className="flex-1 px-2 py-3">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            const active = currentPage === item.id;
            return (
              <button
                key={item.id}
                onClick={() => navigate(`/${item.id}`)}
                className={cn(
                  "mb-1 flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors",
                  active
                    ? "bg-white/10 text-white"
                    : "text-[var(--text-muted)] hover:bg-white/5 hover:text-[var(--text-secondary)]",
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </button>
            );
          })}
        </nav>
        <div className="border-t border-[var(--border-default)] px-4 py-3">
          <div className="flex items-center gap-2 text-xs text-[var(--text-muted)]">
            <Heart className="h-3.5 w-3.5" />
            <span>System</span>
            <span className={cn("ml-auto h-2 w-2 rounded-full", statusColor)} />
          </div>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <ErrorBoundary fallbackMessage="This page encountered an error">
          <Outlet />
        </ErrorBoundary>
      </main>
    </div>
  );
}

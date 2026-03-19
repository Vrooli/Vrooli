import { Activity, RefreshCw, AlertCircle, Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import type { ResourceHealthStatus } from "../../types";
import { fetchResourceHealth } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";

interface HealthDashboardProps {
  onNavigateToWizard?: () => void;
}

function HealthCard({ res }: { res: ResourceHealthStatus }) {
  return (
    <div
      data-testid={`health-card-${res.name}`}
      role="listitem"
      className={cn(
        "rounded-lg border bg-white/5 p-3 transition-all duration-150 sm:rounded-xl sm:p-4",
        "hover:scale-[1.01]",
        res.available
          ? "border-emerald-500/20 hover:border-emerald-500/40 hover:shadow-[0_0_12px_rgba(16,185,129,0.06)]"
          : "border-red-500/20 hover:border-red-500/40 hover:shadow-[0_0_12px_rgba(239,68,68,0.06)]"
      )}
    >
      <div className="flex items-center justify-between sm:justify-start sm:gap-2.5">
        <div className="flex items-center gap-2">
          <span
            data-testid={`status-indicator-${res.name}`}
            className={cn(
              "inline-block h-2 w-2 rounded-full sm:h-2.5 sm:w-2.5",
              res.available ? "bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.4)]" : "bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.4)]"
            )}
            role="img"
            aria-label={`${res.name} is ${res.available ? "healthy" : "unhealthy"}`}
          />
          <span className="text-sm font-medium sm:text-base">{res.name}</span>
        </div>
        <span className="rounded bg-white/5 px-1.5 py-0.5 text-xs text-slate-300 sm:hidden">{res.category}</span>
      </div>
      <div className="mt-1.5 hidden items-center gap-2 text-xs text-slate-300 sm:mt-2 sm:flex">
        <span className="rounded bg-white/5 px-1.5 py-0.5">{res.category}</span>
        <span>{res.status}</span>
      </div>
    </div>
  );
}

export function HealthDashboard({ onNavigateToWizard }: HealthDashboardProps = {}) {
  const { data, isLoading, error, dataUpdatedAt, refetch, isRefetching } = useQuery({
    queryKey: ["resource-health"],
    queryFn: fetchResourceHealth,
    refetchInterval: 30_000,
  });

  const resources = data?.resources ?? [];
  const healthyCount = data?.healthy_count ?? 0;
  const allHealthy = !isLoading && !error && healthyCount === resources.length;

  return (
    <div data-testid="health-dashboard">
      {/* Header - always rendered for heading hierarchy */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-semibold sm:text-2xl">Resource Health</h1>
          {!isLoading && !error && resources.length > 0 && (
            <p data-testid="health-summary" className="mt-1 text-sm text-slate-300">
              <span className={cn("font-medium", allHealthy ? "text-emerald-400" : "text-yellow-400")}>
                {healthyCount} of {resources.length}
              </span>
              {" "}resources healthy
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          {dataUpdatedAt > 0 && (
            <span className="text-xs text-slate-300" data-testid="health-last-checked">
              Last checked {new Date(dataUpdatedAt).toLocaleTimeString()} · auto-refreshes
            </span>
          )}
          {!isLoading && !error && resources.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => void refetch()}
              disabled={isRefetching}
              data-testid="health-refresh"
              aria-label="Refresh health data"
            >
              <RefreshCw className={cn("mr-1.5 h-3 w-3", isRefetching && "animate-spin")} aria-hidden="true" />
              {isRefetching ? "Refreshing..." : "Refresh"}
            </Button>
          )}
        </div>
      </div>

      {/* Body */}
      {isLoading && (
        <div data-testid="health-loading" className="flex flex-col items-center justify-center py-16 text-slate-300" role="status">
          <Loader2 className="h-8 w-8 animate-spin" aria-hidden="true" />
          <p className="mt-3 text-sm">Loading health data...</p>
        </div>
      )}

      {!isLoading && error && (
        <div data-testid="health-error" className="flex flex-col items-center justify-center py-16" role="alert">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
            <AlertCircle className="h-6 w-6 text-red-400" aria-hidden="true" />
          </div>
          <p className="mt-3 text-sm font-medium text-red-400">Failed to load health data</p>
          <p className="mt-1 max-w-xs text-center text-xs text-slate-300">
            {error instanceof Error ? error.message : "Unknown error"}
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void refetch()}
            className="mt-4"
            data-testid="health-retry"
            aria-label="Retry loading health data"
          >
            <RefreshCw className="mr-1.5 h-3 w-3" aria-hidden="true" />
            Retry
          </Button>
        </div>
      )}

      {!isLoading && !error && resources.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-slate-300" data-testid="health-empty">
          <Activity className="h-8 w-8" aria-hidden="true" />
          <p className="mt-3 text-sm font-medium">No resources detected</p>
          <p className="mt-1 text-xs text-slate-300">Complete the setup wizard to configure resources.</p>
          {onNavigateToWizard && (
            <Button
              variant="outline"
              size="sm"
              onClick={onNavigateToWizard}
              className="mt-4"
              data-testid="health-go-to-wizard"
            >
              Go to Setup Wizard
            </Button>
          )}
        </div>
      )}

      {!isLoading && !error && resources.length > 0 && (
        <div
          data-testid="health-grid"
          className="mt-4 grid gap-2 sm:mt-6 sm:gap-3 sm:grid-cols-2 lg:grid-cols-3"
          role="list"
          aria-label="Resource health status"
        >
          {resources.map((res) => (
            <HealthCard key={res.name} res={res} />
          ))}
        </div>
      )}
    </div>
  );
}

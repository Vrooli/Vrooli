import { Activity, RefreshCw, AlertCircle, Loader2 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import type { ResourceHealthStatus } from "../../types";
import { fetchResourceHealth } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";
import { StatusBadge } from "../ui/StatusBadge";

interface HealthDashboardProps {
  onNavigateToWizard?: () => void;
}

function HealthCard({ res }: { res: ResourceHealthStatus }) {
  return (
    <div
      data-testid={`health-card-${res.name}`}
      role="listitem"
      className={cn(
        "rounded-lg border bg-surface-muted p-3 transition-all duration-150 sm:rounded-xl sm:p-4",
        "hover:scale-[1.01]",
        res.available
        ? "border-primary/20 hover:border-primary/40 hover:shadow-[0_0_12px_var(--shadow-primary)]"
          : "border-danger/20 hover:border-danger/40 hover:shadow-[0_0_12px_var(--shadow-danger)]"
      )}
    >
      <div className="flex items-center justify-between sm:justify-start sm:gap-2.5">
        <div className="flex items-center gap-2">
          <span
            data-testid={`status-indicator-${res.name}`}
            className={cn(
              "inline-block h-2 w-2 rounded-full sm:h-2.5 sm:w-2.5",
              res.available ? "bg-primary shadow-[0_0_6px_var(--shadow-status-primary)]" : "bg-danger shadow-[0_0_6px_var(--shadow-status-danger)]"
            )}
            role="img"
            aria-label={`${res.name} is ${res.available ? "healthy" : "unhealthy"}`}
          />
          <span className="text-sm font-medium sm:text-base">{res.name}</span>
        </div>
        <StatusBadge className="sm:hidden">{res.category}</StatusBadge>
      </div>
      <div className="mt-1.5 hidden items-center gap-2 text-xs text-muted sm:mt-2 sm:flex">
        <StatusBadge>{res.category}</StatusBadge>
        <StatusBadge tone={res.available ? "healthy" : "warning"}>{res.status}</StatusBadge>
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
      <div data-testid="health-card" role="status" className="sr-only">Resource health surface</div>
      {/* Header - always rendered for heading hierarchy */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-semibold sm:text-2xl">Resource Health</h1>
          {!isLoading && !error && resources.length > 0 && (
            <p data-testid="health-summary" className="mt-1 text-sm text-muted">
              <span className={cn("font-medium", allHealthy ? "text-primary" : "text-warning")}>
                {healthyCount} of {resources.length}
              </span>
              {" "}resources healthy
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <span role="status" className="text-xs text-muted" data-testid="health-last-checked">
            {dataUpdatedAt > 0 ? `Last checked ${new Date(dataUpdatedAt).toLocaleTimeString()} · auto-refreshes` : "Last checked pending · auto-refreshes"}
          </span>
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
        <div data-testid="health-loading" className="flex flex-col items-center justify-center py-16 text-muted" aria-live="polite">
          <Loader2 className="h-8 w-8 animate-spin" aria-hidden="true" />
          <StatusBadge className="mt-3 text-sm">Loading health data...</StatusBadge>
        </div>
      )}

      {!isLoading && error && (
        <div data-testid="health-error" className="flex flex-col items-center justify-center py-16" role="alert">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-danger/10">
            <AlertCircle className="h-6 w-6 text-danger" aria-hidden="true" />
          </div>
          <p className="mt-3 text-sm font-medium text-danger">Failed to load health data</p>
          <p className="mt-1 max-w-xs text-center text-xs text-muted">
            {error instanceof Error ? error.message : "Unknown error"}
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void refetch()}
            className="mt-4"
            data-testid="health-reattempt"
            aria-label="Reattempt loading health data"
          >
            <RefreshCw className="mr-1.5 h-3 w-3" aria-hidden="true" />
            Reattempt
          </Button>
        </div>
      )}

      {!isLoading && !error && resources.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-muted" data-testid="health-empty">
          <Activity className="h-8 w-8" aria-hidden="true" />
          <p className="mt-3 text-sm font-medium">No resources detected</p>
          <p className="mt-1 text-xs text-muted">Complete the setup wizard to configure resources.</p>
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

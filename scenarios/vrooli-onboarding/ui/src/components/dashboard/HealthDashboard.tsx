import { Activity, RefreshCw, AlertCircle } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import type { ResourceHealthStatus } from "../../types";
import { fetchResourceHealth } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import { HealthIndicator } from "@vrooli/react-component-library/HealthIndicator/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { StatCard } from "@vrooli/react-component-library/StatCard/1";
import { i18n } from "../../i18n";

interface HealthDashboardProps {
  onNavigateToWizard?: () => void;
}

function HealthCard({ res }: { res: ResourceHealthStatus }) {
  return (
    <div
      data-testid={`health-card-${res.name}`}
      role="listitem"
      className={cn(
        "health-resource",
        res.available
        ? "border-primary/20 hover:border-primary/40 hover:shadow-[0_0_12px_var(--shadow-primary)]"
          : "border-danger/20 hover:border-danger/40 hover:shadow-[0_0_12px_var(--shadow-danger)]"
      )}
    >
      <div className="health-resource__identity">
        <div className="flex items-center gap-2">
          <span data-testid={`status-indicator-${res.name}`} role="img" aria-label={`${res.name} is ${res.available ? i18n.t("onboarding.dashboard.healthy").toLowerCase() : "unhealthy"}`}>
            <HealthIndicator state={res.available ? "healthy" : "blocked"} />
          </span>
          <span className="text-sm font-medium sm:text-base">{res.name}</span>
        </div>
        <StatusBadge>{res.category}</StatusBadge>
      </div>
      <StatusBadge tone={res.available ? "success" : "warning"}>{res.status}</StatusBadge>
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
    <div data-testid="health-dashboard" className="health-surface">
      <div data-testid="health-card" role="status" className="sr-only">{i18n.t("onboarding.dashboard.surface")}</div>
      {/* Header - always rendered for heading hierarchy */}
      <header className="surface-heading">
        <div>
          <p className="surface-eyebrow">{i18n.t("onboarding.dashboard.eyebrow")}</p>
          <h1>{i18n.t("onboarding.dashboard.heading")}</h1>
          {!isLoading && !error && resources.length > 0 && (
            <p data-testid="health-summary" className="mt-1 text-sm text-muted">
              <span className={cn("font-medium", allHealthy ? "text-primary" : "text-warning")}>
                {healthyCount} of {resources.length}
              </span>
              {allHealthy ? ` ${i18n.t("onboarding.dashboard.systemsReady")}` : ` ${i18n.t("onboarding.dashboard.needsAttention")}`}
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <span role="status" className="text-xs text-muted" data-testid="health-last-checked">
            {dataUpdatedAt > 0 ? i18n.t("onboarding.dashboard.lastChecked", { time: new Date(dataUpdatedAt).toLocaleTimeString() }) : i18n.t("onboarding.dashboard.pending")}
          </span>
          {!isLoading && !error && resources.length > 0 && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => void refetch()}
              disabled={isRefetching}
              data-testid="health-refresh"
              aria-label={i18n.t("onboarding.dashboard.refreshData")}
            >
              <RefreshCw className={cn("mr-1.5 h-3 w-3", isRefetching && "animate-spin")} aria-hidden="true" />
              {isRefetching ? i18n.t("onboarding.dashboard.refreshing") : i18n.t("onboarding.dashboard.refresh")}
            </Button>
          )}
        </div>
      </header>

      {!isLoading && !error && resources.length > 0 && (
        <div className="health-stats" data-testid="health-summary-cards">
          <StatCard label={i18n.t("onboarding.dashboard.healthy")} value={String(healthyCount)} tone={allHealthy ? "success" : "warning"} />
          <StatCard label={i18n.t("onboarding.dashboard.needsAttention")} value={String(resources.length - healthyCount)} tone={healthyCount === resources.length ? "success" : "warning"} />
          <StatCard label={i18n.t("onboarding.dashboard.resourcesChecked")} value={String(resources.length)} />
        </div>
      )}

      {/* Body */}
      <AsyncPanel
        surfaceId="resource-health"
        state={isLoading ? "loading" : error ? "error" : resources.length === 0 ? "empty" : "ready"}
        loading={<div data-testid="health-loading" className="health-loading" aria-live="polite"><div className="health-loading__stats"><Skeleton label={i18n.t("onboarding.dashboard.loadingSummary")} /><Skeleton /><Skeleton /></div><div className="health-loading__rows"><Skeleton label={i18n.t("onboarding.dashboard.loadingChecks")} /><Skeleton /><Skeleton /><Skeleton /></div></div>}
        error={<div data-testid="health-error" className="flex flex-col items-center justify-center py-16" role="alert"><AlertCircle className="h-6 w-6 text-danger" aria-hidden="true" /><p className="mt-3 text-sm font-medium text-danger">{i18n.t("onboarding.dashboard.failed")}</p><p className="mt-1 max-w-xs text-center text-xs text-muted">{error instanceof Error ? error.message : i18n.t("onboarding.dashboard.unknownError")}</p><Button variant="secondary" size="sm" onClick={() => void refetch()} className="mt-4" data-testid="health-reattempt" aria-label={i18n.t("onboarding.dashboard.reattemptLabel")}><RefreshCw className="mr-1.5 h-3 w-3" aria-hidden="true" />{i18n.t("onboarding.dashboard.reattempt")}</Button></div>}
        empty={<div className="flex flex-col items-center justify-center py-16 text-muted" data-testid="health-empty"><Activity className="h-8 w-8" aria-hidden="true" /><p className="mt-3 text-sm font-medium">{i18n.t("onboarding.dashboard.noResources")}</p><p className="mt-1 text-xs text-muted">{i18n.t("onboarding.dashboard.completeWizard")}</p>{onNavigateToWizard && <Button variant="secondary" size="sm" onClick={onNavigateToWizard} className="mt-4" data-testid="health-go-to-wizard">{i18n.t("onboarding.dashboard.goToWizard")}</Button>}</div>}
      >
        <div
          data-testid="health-grid"
          className="health-resources"
          role="list"
          aria-label={i18n.t("onboarding.dashboard.status")}
        >
          {resources.map((res) => (
            <HealthCard key={res.name} res={res} />
          ))}
        </div>
      </AsyncPanel>
    </div>
  );
}

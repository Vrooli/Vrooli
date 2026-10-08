import type { Ref } from "react";
import { Info, Loader2, RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { statusTone } from "../../theme/status";
import type { ScenarioDeploymentStatus } from "./deploymentStatus";
import { DeploymentStatusBadge, TierFitnessBadge } from "./DeploymentStatusBadge";

interface DeploymentStatusListProps {
  loading: boolean;
  loadingReports: boolean;
  onRefresh: () => void;
  onScan: (scenarioName: string, apply?: boolean) => void;
  onSearchChange: (value: string) => void;
  onSelectScenario: (status: ScenarioDeploymentStatus) => void;
  search: string;
  selectedScenarioName?: string;
  statuses: ScenarioDeploymentStatus[];
  statusRef: Ref<HTMLDivElement>;
  targetTier: string;
}

export function DeploymentStatusList({
  loading,
  loadingReports,
  onRefresh,
  onScan,
  onSearchChange,
  onSelectScenario,
  search,
  selectedScenarioName,
  statuses,
  statusRef,
  targetTier
}: DeploymentStatusListProps) {
  return (
    <Card className="border border-border/40 bg-background/40" ref={statusRef} id="deployment-status">
      <CardHeader className="flex flex-row items-center justify-between pb-1">
        <div>
          <CardTitle className="text-base">Scenario Deployment Status</CardTitle>
          <p className="text-xs text-muted-foreground">
            Click a row to open inline details (blockers, requirements, metadata gaps) for the selected tier.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={loading || loadingReports}
          className="gap-2"
        >
          {loading || loadingReports ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Refresh
        </Button>
      </CardHeader>
      <CardContent>
        <div className="mb-3 flex flex-wrap items-center gap-3">
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search scenarios..."
            className="h-9 w-full max-w-xs rounded border border-border/60 bg-background/60 px-3 text-sm text-foreground placeholder:text-muted-foreground"
            aria-label="Search scenarios"
          />
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Info className="h-3.5 w-3.5" aria-hidden="true" />
            <span>Sorted by urgency: critical → issues → not scanned → ready.</span>
          </div>
        </div>
        {loading || loadingReports ? (
          <div className="flex min-h-[200px] items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
          </div>
        ) : (
          <div className="space-y-2">
            {statuses.map((status) => {
              const tierAgg = status.lastReport?.aggregates?.[targetTier];
              const tierFitness = tierAgg?.fitness_score;
              const blockers = tierAgg?.blocking_dependencies || [];
              return (
                <div
                  key={status.scenario.name}
                  role="button"
                  tabIndex={0}
                  className={`flex w-full cursor-pointer items-center justify-between rounded-lg border border-border/40 bg-background/40 p-3 text-left transition-colors hover:bg-background/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background ${
                    selectedScenarioName === status.scenario.name ? "ring-2 ring-primary/50" : ""
                  }`}
                  onClick={() => onSelectScenario(status)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onSelectScenario(status);
                    }
                  }}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-foreground">
                        {status.scenario.display_name || status.scenario.name}
                      </p>
                      <DeploymentStatusBadge status={status.status} />
                    </div>
                    {status.scenario.description && (
                      <p className="mt-1 truncate text-xs text-muted-foreground">{status.scenario.description}</p>
                    )}
                    <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <span>
                        Last scan:{" "}
                        {status.scenario.last_scanned
                          ? new Date(status.scenario.last_scanned).toLocaleString()
                          : "Never"}
                      </span>
                      <span title="Best/worst across tiers">
                        Tier fitness: <TierFitnessBadge tierFitness={status.tierFitness} />
                      </span>
                      <span title={`Fitness for ${targetTier}`}>
                        {tierFitness !== undefined ? `${Math.round((tierFitness || 0) * 100)}% for ${targetTier}` : "No tier score"}
                      </span>
                      {blockers.length > 0 && <span className={statusTone("danger").text}>Blockers ({targetTier}): {blockers.length}</span>}
                      {status.missingMetadataCount > 0 && (
                        <span className={statusTone("warning").text}>Missing metadata: {status.missingMetadataCount}</span>
                      )}
                    </div>
                  </div>
                  <span className="ml-4 flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onScan(status.scenario.name, false);
                      }}
                      className="h-8 text-xs"
                      disabled={status.loading}
                    >
                      {status.loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Scan"}
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onScan(status.scenario.name, true);
                      }}
                      className="h-8 text-xs"
                      disabled={status.loading}
                    >
                      {status.loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Scan & Apply"}
                    </Button>
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

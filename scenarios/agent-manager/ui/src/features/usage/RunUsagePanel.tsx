import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Gauge } from "lucide-react";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import type { UsageBurner, UsageWindowAttribution } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { useUsageBurners, useUsageRecommendations } from "./api";
import { formatCount, formatPoints, formatTokens } from "../../lib/format";
import { windowKey, windowTitle } from "./format";
import { ActionList } from "./UsageActions";

function Tile({ label, value, caption }: { label: string; value: string; caption: string }) {
  return (
    <div className="rounded border border-border px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-lg font-semibold tabular-nums">{value}</div>
      <div className="text-xs text-muted-foreground">{caption}</div>
    </div>
  );
}

function RunWindowUsage({ attribution, burner }: { attribution: UsageWindowAttribution; burner: UsageBurner }) {
  const tokens = burner.tokens;
  return (
    <div className="space-y-2" data-testid="run-usage-window">
      <div className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
        <Tile label="Quota used" value={`≈ ${formatPoints(burner.points)}`} caption={`of ${windowTitle(attribution)}`} />
        <Tile label="Cached input" value={formatTokens(tokens?.cachedInput ?? 0n)} caption="re-read context" />
        <Tile label="New input" value={formatTokens((tokens?.input ?? 0n) + (tokens?.cacheCreation ?? 0n))} caption="not cached" />
        <Tile label="Output" value={formatTokens(tokens?.output ?? 0n)} caption={`${formatCount(burner.requests)} requests`} />
      </div>
      <p className="text-xs text-muted-foreground">
        {formatTokens(tokens?.total ?? 0n)} tokens in {burner.requests.toString()} requests, from the run's transcript{burner.model ? ` · model ${burner.model}` : ""}. Quota points are an estimate calibrated to provider readings.
      </p>
    </div>
  );
}

function RunUsageSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
      {Array.from({ length: 4 }, (_, index) => <Skeleton key={index} className="h-[4.25rem] rounded" />)}
    </div>
  );
}

interface RunUsagePanelProps {
  runId: string;
  /** The run's cost-event breakdown (rendered below the quota tiles), when it has any. */
  children?: ReactNode;
  hasCost: boolean;
  costLoading: boolean;
}

/**
 * Run detail › Usage: the run's tokens and estimated quota points per window,
 * its cost events, and the recommendations that name it. One empty message, and
 * only once both sources have answered with nothing.
 */
export function RunUsagePanel({ runId, children, hasCost, costLoading }: RunUsagePanelProps) {
  const burners = useUsageBurners("run", runId);
  const recommendations = useUsageRecommendations(runId);
  const rows = (burners.data?.windows ?? []).flatMap((attribution) => attribution.burners.filter((burner) => burner.key === runId).map((burner) => ({ attribution, burner })));
  const runActions = recommendations.data?.actions ?? [];
  return (
    <section aria-labelledby="run-usage-heading" className="space-y-4" data-testid="run-usage-panel">
      <h3 id="run-usage-heading" className="sr-only">Quota usage</h3>
      <DataState
        compact
        subject="Run usage"
        loading={burners.isLoading || costLoading}
        error={burners.error}
        isEmpty={rows.length === 0 && !hasCost}
        skeleton={<RunUsageSkeleton />}
        onRetry={() => void burners.refetch()}
        empty={<EmptyState density="compact" icon={<Gauge aria-hidden="true" />} title="No usage recorded for this run" description="No token counters or cost events were captured for this run, and it used no quota inside a current provider window." />}
      >
        {rows.map(({ attribution, burner }) => <RunWindowUsage key={windowKey(attribution)} attribution={attribution} burner={burner} />)}
        {children}
      </DataState>
      {runActions.length > 0 ? (
        <div className="space-y-1">
          <div className="flex items-center justify-between gap-2">
            <h4 className="text-sm font-semibold">Recommendations for this run</h4>
            <Link to="/usage" className="inline-flex min-h-11 items-center text-sm text-primary sm:min-h-8">All recommendations →</Link>
          </div>
          <ActionList actions={runActions} />
        </div>
      ) : null}
    </section>
  );
}

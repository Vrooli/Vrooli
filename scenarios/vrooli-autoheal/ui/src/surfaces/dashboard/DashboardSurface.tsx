import { Activity, AlertCircle, AlertTriangle, CheckCircle, ChevronDown, ChevronRight, HardDrive } from "lucide-react";
import { CheckCard, EventsTimeline, PlatformInfo, SummaryCard, SystemProtection, UptimeStats } from "./components";
import { Card } from "../../shared/ui/primitives";
import type { CheckCategory, HealthResult, StatusResponse } from "../../lib/api";

export interface EnrichedCheck extends HealthResult {
  title?: string;
  description?: string;
  importance?: string;
  category?: CheckCategory;
  intervalSeconds?: number;
}

export interface CollapsedGroups {
  critical: boolean;
  warning: boolean;
  ok: boolean;
}

interface DashboardSurfaceProps {
  data: StatusResponse | undefined;
  checksMetadataCount: number;
  enrichedChecks: EnrichedCheck[];
  groupedChecks: {
    critical: EnrichedCheck[];
    warning: EnrichedCheck[];
    ok: EnrichedCheck[];
  };
  collapsedGroups: CollapsedGroups;
  onToggleGroup: (group: keyof CollapsedGroups) => void;
  autoRefresh: boolean;
  autoRefreshIntervalSeconds: number;
  onShowTrends: () => void;
  onSelectCheck: (checkId: string) => void;
}

export function DashboardSurface({
  data,
  checksMetadataCount,
  enrichedChecks,
  groupedChecks,
  collapsedGroups,
  onToggleGroup,
  autoRefresh,
  autoRefreshIntervalSeconds,
  onShowTrends,
  onSelectCheck,
}: DashboardSurfaceProps) {
  return (
    <>
      <div className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
        <SummaryCard
          title="Total Checks"
          value={data?.summary.total || 0}
          icon={HardDrive}
          tone="neutral"
        />
        <SummaryCard
          title="Healthy"
          value={data?.summary.ok || 0}
          icon={CheckCircle}
          tone="success"
        />
        <SummaryCard
          title="Warnings"
          value={data?.summary.warning || 0}
          icon={AlertTriangle}
          tone="warning"
        />
        <SummaryCard
          title="Critical"
          value={data?.summary.critical || 0}
          icon={AlertCircle}
          tone="danger"
        />
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        <div className="space-y-4 md:col-span-2">
          <h2 className="flex items-center gap-2 text-lg font-medium">
            <Activity size={20} className="text-accent-primary" />
            Health Checks
            {enrichedChecks.length === 0 && checksMetadataCount > 0 && (
              <span className="text-sm font-normal text-text-muted/80">
                ({checksMetadataCount} registered - click &quot;Run Tick&quot; to execute)
              </span>
            )}
          </h2>

          {groupedChecks.critical.length > 0 && (
            <div className="space-y-2">
              <button
                onClick={() => onToggleGroup("critical")}
                className="flex w-full items-center gap-2 text-left text-sm font-medium text-red-400 transition-colors hover:text-red-300"
              >
                {collapsedGroups.critical ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
                <AlertCircle size={16} />
                <span>Critical Issues</span>
                <span className="ml-auto text-xs font-normal text-red-500/80">
                  {groupedChecks.critical.length} {groupedChecks.critical.length === 1 ? "check" : "checks"}
                </span>
              </button>
              {!collapsedGroups.critical && (
                <div className="ml-1 space-y-2">
                  {groupedChecks.critical.map((check) => (
                    <CheckCard key={check.checkId} check={check} onInfoClick={onSelectCheck} />
                  ))}
                </div>
              )}
            </div>
          )}

          {groupedChecks.warning.length > 0 && (
            <div className="space-y-2">
              <button
                onClick={() => onToggleGroup("warning")}
                className="flex w-full items-center gap-2 text-left text-sm font-medium text-amber-400 transition-colors hover:text-amber-300"
              >
                {collapsedGroups.warning ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
                <AlertTriangle size={16} />
                <span>Warnings</span>
                <span className="ml-auto text-xs font-normal text-amber-500/80">
                  {groupedChecks.warning.length} {groupedChecks.warning.length === 1 ? "check" : "checks"}
                </span>
              </button>
              {!collapsedGroups.warning && (
                <div className="ml-1 space-y-2">
                  {groupedChecks.warning.map((check) => (
                    <CheckCard key={check.checkId} check={check} onInfoClick={onSelectCheck} />
                  ))}
                </div>
              )}
            </div>
          )}

          {groupedChecks.ok.length > 0 && (
            <div className="space-y-2">
              <button
                onClick={() => onToggleGroup("ok")}
                className="flex w-full items-center gap-2 text-left text-sm font-medium text-emerald-400 transition-colors hover:text-emerald-300"
              >
                {collapsedGroups.ok ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
                <CheckCircle size={16} />
                <span>Healthy</span>
                <span className="ml-auto text-xs font-normal text-emerald-500/80">
                  {groupedChecks.ok.length} {groupedChecks.ok.length === 1 ? "check" : "checks"}
                </span>
              </button>
              {!collapsedGroups.ok && (
                <div className="ml-1 space-y-2">
                  {groupedChecks.ok.map((check) => (
                    <CheckCard key={check.checkId} check={check} onInfoClick={onSelectCheck} />
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="space-y-4">
          <SystemProtection />
          <UptimeStats onShowTrends={onShowTrends} />
          {data?.platform && <PlatformInfo platform={data.platform} />}

          <Card className="p-4">
            <p className="text-sm text-text-muted">
              Last updated:{" "}
              <span className="text-text-primary">
                {data?.timestamp ? new Date(data.timestamp).toLocaleTimeString() : "Never"}
              </span>
            </p>
            {autoRefresh && (
              <p className="mt-1 text-xs text-text-muted/80">
                Auto-refresh every {autoRefreshIntervalSeconds}s
              </p>
            )}
          </Card>
        </div>
      </div>

      <div className="mt-6">
        <EventsTimeline />
      </div>
    </>
  );
}

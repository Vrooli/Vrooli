import { Activity, AlertCircle, AlertTriangle, CheckCircle, ChevronDown, ChevronRight, HardDrive, HelpCircle } from "lucide-react";
import { memo, Profiler } from "react";
import { CheckCard, EventsTimeline, PlatformInfo, SummaryCard, SystemProtection, UptimeStats } from "./components";
import { Card } from "../../shared/ui/primitives";
import { selectors } from "../../consts/selectors";
import type { CheckCategory, HealthResult, StatusResponse } from "../../lib/api";
import { onProfilerRender } from "../../lib/profiler";

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
  notApplicable: boolean;
}

type GroupKey = keyof CollapsedGroups;

const GROUP_CONFIG: Record<
  GroupKey,
  {
    title: string;
    icon: typeof AlertCircle;
    headerClass: string;
    countClass: string;
  }
> = {
  critical: {
    title: "Critical Issues",
    icon: AlertCircle,
    headerClass: "bg-accent-danger/10 text-accent-danger hover:bg-accent-danger/20",
    countClass: "text-accent-danger/80",
  },
  warning: {
    title: "Warnings",
    icon: AlertTriangle,
    headerClass: "bg-accent-warning/10 text-accent-warning hover:bg-accent-warning/20",
    countClass: "text-accent-warning/80",
  },
  ok: {
    title: "Healthy",
    icon: CheckCircle,
    headerClass: "bg-accent-success/10 text-accent-success hover:bg-accent-success/20",
    countClass: "text-accent-success/80",
  },
  notApplicable: {
    title: "Not Applicable",
    icon: HelpCircle,
    headerClass: "bg-surface-overlay/60 text-text-muted hover:bg-surface-overlay",
    countClass: "text-text-muted",
  },
};

interface CheckGroupSectionProps {
  group: GroupKey;
  checks: EnrichedCheck[];
  collapsed: boolean;
  onToggleGroup: (group: GroupKey) => void;
  onSelectCheck: (checkId: string) => void;
}

const CheckGroupSection = memo(function CheckGroupSection({
  group,
  checks,
  collapsed,
  onToggleGroup,
  onSelectCheck,
}: CheckGroupSectionProps) {
  if (checks.length === 0) {
    return null;
  }

  const config = GROUP_CONFIG[group];
  const Icon = config.icon;

  return (
    <div className="space-y-1.5 sm:space-y-2">
      <button
        onClick={() => onToggleGroup(group)}
        className={`flex min-w-0 w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm font-medium transition-colors ${config.headerClass}`}
      >
        {collapsed ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
        <Icon size={16} className="shrink-0" />
        <span className="min-w-0 truncate">{config.title}</span>
        <span className={`ml-auto text-xs font-normal ${config.countClass}`}>
          {checks.length} {checks.length === 1 ? "check" : "checks"}
        </span>
      </button>
      {!collapsed && (
        <div className="min-w-0 rounded-lg border border-border-default/70 bg-surface-elevated/40 divide-y divide-border-default/70 sm:space-y-2 sm:divide-y-0 sm:border-none sm:bg-transparent">
          {checks.map((check) => (
            <CheckCard
              key={check.checkId}
              check={check}
              onInfoClick={onSelectCheck}
              mobileListItem
            />
          ))}
        </div>
      )}
    </div>
  );
});

interface DashboardSurfaceProps {
  data: StatusResponse | undefined;
  checksMetadataCount: number;
  enrichedChecks: EnrichedCheck[];
  groupedChecks: {
    critical: EnrichedCheck[];
    warning: EnrichedCheck[];
    ok: EnrichedCheck[];
    notApplicable: EnrichedCheck[];
  };
  collapsedGroups: CollapsedGroups;
  onToggleGroup: (group: keyof CollapsedGroups) => void;
  autoRefresh: boolean;
  autoRefreshIntervalSeconds: number;
  onShowTrends: () => void;
  onSelectCheck: (checkId: string) => void;
}

function DashboardSurfaceImpl({
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
    <div className="min-w-0">
      <div
        className="mb-5 grid min-w-0 grid-cols-2 gap-3 sm:mb-6 sm:gap-4 md:grid-cols-4"
        data-testid={selectors.summary.grid}
      >
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
        <SummaryCard
          title="Not Applicable"
          value={data?.summary.notApplicable || 0}
          icon={HelpCircle}
          tone="neutral"
        />
      </div>

      <div className="grid gap-5 sm:gap-6 md:grid-cols-3">
        <div className="space-y-3 sm:space-y-4 md:col-span-2">
          <h2 className="flex flex-wrap items-center gap-2 text-lg font-medium">
            <Activity size={20} className="text-accent-primary" />
            Health Checks
            {enrichedChecks.length === 0 && checksMetadataCount > 0 && (
              <span className="text-xs font-normal text-text-muted/80 sm:text-sm">
                ({checksMetadataCount} registered - click &quot;Run Tick&quot; to execute)
              </span>
            )}
          </h2>

          {(Object.keys(GROUP_CONFIG) as GroupKey[]).map((group) => (
            <CheckGroupSection
              key={group}
              group={group}
              checks={groupedChecks[group]}
              collapsed={collapsedGroups[group]}
              onToggleGroup={onToggleGroup}
              onSelectCheck={onSelectCheck}
            />
          ))}
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
        <Profiler id="EventsTimeline" onRender={onProfilerRender}>
          <EventsTimeline />
        </Profiler>
      </div>
    </div>
  );
}

export function DashboardSurface(props: DashboardSurfaceProps) {
  return (
    <Profiler id="DashboardSurface" onRender={onProfilerRender}>
      <DashboardSurfaceImpl {...props} />
    </Profiler>
  );
}

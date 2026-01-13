/**
 * Horizontal pipeline overview showing status of all stages.
 * Displays: Bundle → Preflight → Generate → Build → Smoke Test
 */

import { ChevronRight, Package, Shield, Cog, Hammer, TestTube } from "lucide-react";
import { StageStatusBadge, type StageStatusType } from "./StageStatusBadge";
import type { ValidationStatus, StageStatus } from "../../lib/api";
import { cn } from "../../lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";

interface PipelineStatusOverviewProps {
  validationStatus: ValidationStatus | null;
  className?: string;
  compact?: boolean;
}

const STAGE_ORDER = ["bundle", "preflight", "generate", "build", "smoke_test"] as const;

const stageConfig: Record<
  (typeof STAGE_ORDER)[number],
  { label: string; Icon: typeof Package }
> = {
  bundle: { label: "Bundle", Icon: Package },
  preflight: { label: "Preflight", Icon: Shield },
  generate: { label: "Generate", Icon: Cog },
  build: { label: "Build", Icon: Hammer },
  smoke_test: { label: "Smoke Test", Icon: TestTube },
};

interface StageNodeProps {
  stageName: (typeof STAGE_ORDER)[number];
  stageStatus: StageStatus | undefined;
  compact?: boolean;
}

function StageNode({ stageName, stageStatus, compact }: StageNodeProps) {
  const config = stageConfig[stageName];
  const status: StageStatusType = (stageStatus?.status as StageStatusType) || "none";
  const { Icon } = config;

  const statusColors: Record<StageStatusType, string> = {
    valid: "border-green-600 bg-green-950/30 text-green-400",
    stale: "border-yellow-600 bg-yellow-950/30 text-yellow-400",
    invalid: "border-red-600 bg-red-950/30 text-red-400",
    none: "border-slate-600 bg-slate-950/30 text-slate-400",
  };

  const content = (
    <div
      className={cn(
        "flex items-center gap-2 rounded-lg border px-3 py-2 transition-colors",
        statusColors[status],
        compact ? "px-2 py-1" : ""
      )}
    >
      <Icon className={cn("flex-shrink-0", compact ? "h-3.5 w-3.5" : "h-4 w-4")} />
      {!compact && (
        <span className="text-xs font-medium whitespace-nowrap">
          {config.label}
        </span>
      )}
    </div>
  );

  if (compact) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent side="bottom" className="text-xs">
            <div className="font-medium">{config.label}</div>
            <div className="text-slate-400 capitalize">{status}</div>
            {stageStatus?.staleness_reason && (
              <div className="mt-1 text-yellow-300 text-[10px]">
                {stageStatus.staleness_reason}
              </div>
            )}
            {stageStatus?.last_run && (
              <div className="mt-1 text-slate-500 text-[10px]">
                Last run: {new Date(stageStatus.last_run).toLocaleString()}
              </div>
            )}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return content;
}

export function PipelineStatusOverview({
  validationStatus,
  className,
  compact = false,
}: PipelineStatusOverviewProps) {
  const stages = validationStatus?.stages || {};

  return (
    <div className={cn("flex items-center flex-wrap gap-1", className)}>
      {STAGE_ORDER.map((stageName, index) => (
        <div key={stageName} className="flex items-center">
          <StageNode
            stageName={stageName}
            stageStatus={stages[stageName]}
            compact={compact}
          />
          {index < STAGE_ORDER.length - 1 && (
            <ChevronRight className="mx-1 h-4 w-4 text-slate-600" />
          )}
        </div>
      ))}
    </div>
  );
}

interface PipelineStatusSummaryProps {
  validationStatus: ValidationStatus | null;
  className?: string;
}

/**
 * Compact summary showing overall pipeline status.
 */
export function PipelineStatusSummary({
  validationStatus,
  className,
}: PipelineStatusSummaryProps) {
  if (!validationStatus) {
    return (
      <div className={cn("flex items-center gap-2 text-xs text-slate-400", className)}>
        <span>Pipeline:</span>
        <StageStatusBadge status="none" showLabel size="sm" />
      </div>
    );
  }

  const overallStatus = validationStatus.overall_status as StageStatusType;
  const staleCount = Object.values(validationStatus.stages || {}).filter(
    (s) => s.status === "stale"
  ).length;
  const validCount = Object.values(validationStatus.stages || {}).filter(
    (s) => s.status === "valid"
  ).length;

  return (
    <div className={cn("flex items-center gap-2 text-xs", className)}>
      <span className="text-slate-400">Pipeline:</span>
      <StageStatusBadge status={overallStatus || "none"} showLabel size="sm" />
      {staleCount > 0 && (
        <span className="text-yellow-400/80">({staleCount} stale)</span>
      )}
      {validCount > 0 && staleCount === 0 && (
        <span className="text-green-400/80">({validCount} cached)</span>
      )}
    </div>
  );
}

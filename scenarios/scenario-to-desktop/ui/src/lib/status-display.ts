/**
 * Shared status display utilities for pipeline and stage statuses.
 * Extracted from SectionHeader.tsx and SidebarHeader.tsx to eliminate duplication.
 */

import type { LucideIcon } from "lucide-react";
import {
  Loader2,
  CheckCircle2,
  XCircle,
  Circle,
  Clock,
  AlertCircle,
  Play,
} from "lucide-react";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** Pipeline status types (superset of stage statuses) */
export type PipelineStatus =
  | StageStatus
  | "idle"
  | "starting"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | null
  | undefined;

/** Status display configuration */
export interface StatusDisplayConfig {
  label: string;
  icon: LucideIcon;
  className: string;
}

/** Badge-style status display with separate badge and icon classes */
export interface BadgeStatusDisplay {
  label: string;
  icon: LucideIcon;
  badgeClass: string;
  iconClass: string;
}

/** Base status display configuration for stages */
export const STAGE_STATUS_CONFIG: Partial<
  Record<StageStatus, StatusDisplayConfig>
> = {
  [StageStatus.PENDING]: {
    label: "Pending",
    icon: Clock,
    className: "text-slate-400",
  },
  [StageStatus.RUNNING]: {
    label: "Running",
    icon: Clock,
    className: "text-blue-400 animate-pulse",
  },
  [StageStatus.COMPLETED]: {
    label: "Completed",
    icon: CheckCircle2,
    className: "text-green-400",
  },
  [StageStatus.FAILED]: {
    label: "Failed",
    icon: XCircle,
    className: "text-red-400",
  },
  [StageStatus.SKIPPED]: {
    label: "Skipped",
    icon: Circle,
    className: "text-slate-400",
  },
} as const;

/** Badge-style status display for section headers */
export const HEADER_STATUS_CONFIG: Partial<
  Record<StageStatus, BadgeStatusDisplay>
> = {
  [StageStatus.PENDING]: {
    label: "Pending",
    icon: Circle,
    badgeClass: "bg-slate-500/20 text-slate-400 border-slate-500/30",
    iconClass: "text-slate-500",
  },
  [StageStatus.RUNNING]: {
    label: "Running",
    icon: Loader2,
    badgeClass: "bg-blue-500/20 text-blue-400 border-blue-500/30",
    iconClass: "text-blue-400 animate-spin",
  },
  [StageStatus.COMPLETED]: {
    label: "Completed",
    icon: CheckCircle2,
    badgeClass: "bg-green-500/20 text-green-400 border-green-500/30",
    iconClass: "text-green-400",
  },
  [StageStatus.FAILED]: {
    label: "Failed",
    icon: XCircle,
    badgeClass: "bg-red-500/20 text-red-400 border-red-500/30",
    iconClass: "text-red-400",
  },
  [StageStatus.SKIPPED]: {
    label: "Skipped",
    icon: Circle,
    badgeClass: "bg-slate-500/20 text-slate-400 border-slate-500/30",
    iconClass: "text-slate-500",
  },
} as const;

/**
 * Get stage status display configuration with optional custom labels.
 * @param status - Stage status
 * @param labels - Optional custom labels for specific statuses
 */
export function getStageStatusDisplay(
  status: StageStatus | undefined,
  labels?: Partial<Record<StageStatus, string>>,
): StatusDisplayConfig {
  const s = status ?? StageStatus.PENDING;
  const base =
    STAGE_STATUS_CONFIG[s] ?? STAGE_STATUS_CONFIG[StageStatus.PENDING];
  if (!base) throw new Error("Missing pending stage status configuration");
  const customLabel = labels?.[s];
  if (customLabel) {
    return { ...base, label: customLabel };
  }
  return base;
}

/**
 * Get pipeline status display for sidebar header.
 * Handles additional statuses like 'idle', 'starting', 'cancelled', and null.
 */
export function getPipelineStatusDisplay(
  status: PipelineStatus,
): StatusDisplayConfig {
  switch (status) {
    case "idle":
      // Pipeline created but not started - ready for user to configure and run
      return {
        label: "Ready",
        icon: Play,
        className: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
      };
    case StageStatus.RUNNING:
    case "running":
    case "starting":
      return {
        label: "Running",
        icon: Loader2,
        className: "bg-blue-500/20 text-blue-400 border-blue-500/30",
      };
    case StageStatus.COMPLETED:
    case "completed":
      return {
        label: "Completed",
        icon: CheckCircle2,
        className: "bg-green-500/20 text-green-400 border-green-500/30",
      };
    case StageStatus.FAILED:
    case "failed":
      return {
        label: "Failed",
        icon: AlertCircle,
        className: "bg-red-500/20 text-red-400 border-red-500/30",
      };
    case "cancelled":
      return {
        label: "Cancelled",
        icon: AlertCircle,
        className: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30",
      };
    case StageStatus.PENDING:
      return {
        label: "Pending",
        icon: Circle,
        className: "bg-slate-500/20 text-slate-400 border-slate-500/30",
      };
    default:
      return {
        label: "No Pipeline",
        icon: Play,
        className: "bg-slate-500/20 text-slate-400 border-slate-500/30",
      };
  }
}

/** Format stage name for display (capitalize first letter) */
export function formatStageName(stage: StageName | null): string {
  switch (stage) {
    case StageName.RESOLVE_DEPLOYMENT:
      return "Resolve deployment";
    case StageName.BUNDLE:
      return "Bundle";
    case StageName.PREFLIGHT:
      return "Preflight";
    case StageName.GENERATE:
      return "Generate";
    case StageName.BUILD:
      return "Build";
    case StageName.SMOKE_TEST:
      return "Smoke test";
    case StageName.DEPLOY:
      return "Deploy";
    default:
      return "";
  }
}

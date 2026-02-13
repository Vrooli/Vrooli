/**
 * Domain constants for Swarm Manager
 *
 * This module contains constants related to domain concepts.
 * Presentation-layer code should import these rather than defining their own.
 */

import { Circle, CheckCircle, AlertCircle, type LucideIcon } from "lucide-react";
import { formatDisplayText } from "../lib";
import type {
  BacklogKind,
  BacklogResearchTarget,
  BacklogStatus,
  ExecutionMode,
  ExecutionStatus,
  ScenarioStatus,
} from "./domain";

// ============================================================================
// Backlog Status Display
// ============================================================================

export const BACKLOG_KINDS: BacklogKind[] = ["idea", "research", "fix", "execute"];

export const BACKLOG_KIND_LABELS: Record<BacklogKind, string> = {
  idea: "Idea",
  research: "Research",
  fix: "Fix",
  execute: "Execute",
};

export const BACKLOG_RESEARCH_TARGETS: BacklogResearchTarget[] = [
  "idea",
  "fix",
  "execute",
  "unspecified",
];

export const BACKLOG_STATUSES: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

export const BACKLOG_RESEARCH_TARGET_LABELS: Record<BacklogResearchTarget, string> = {
  idea: "Idea",
  fix: "Fix",
  execute: "Execute",
  unspecified: "Unspecified",
};

/**
 * Maps backlog status to tailwind background color classes
 */
export const BACKLOG_STATUS_COLORS: Record<BacklogStatus, string> = {
  backlog: "bg-slate-600",
  researching: "bg-blue-600",
  ready: "bg-green-600",
  queued: "bg-yellow-600",
  in_progress: "bg-purple-600",
  completed: "bg-emerald-600",
  archived: "bg-gray-600",
};

/**
 * Formats a backlog status for display (converts underscores/hyphens to spaces, capitalizes).
 * Delegates to the shared formatDisplayText utility to prevent duplication.
 */
export function formatBacklogStatus(status: BacklogStatus): string {
  return formatDisplayText(status);
}

// ============================================================================
// Scenario Status Display
// ============================================================================

/**
 * Maps scenario status to lucide icons
 */
export const SCENARIO_STATUS_ICONS: Record<ScenarioStatus, LucideIcon> = {
  running: CheckCircle,
  stopped: Circle,
  error: AlertCircle,
  unknown: Circle,
};

export const SCENARIO_STATUSES: ScenarioStatus[] = ["running", "stopped", "error", "unknown"];

/**
 * Maps scenario status to tailwind text color classes
 */
export const SCENARIO_STATUS_COLORS: Record<ScenarioStatus, string> = {
  running: "text-green-400",
  stopped: "text-slate-400",
  error: "text-red-400",
  unknown: "text-slate-500",
};

// ============================================================================
// Execution Status Display
// ============================================================================

export const EXECUTION_STATUSES: ExecutionStatus[] = [
  "pending",
  "scheduled",
  "running",
  "completed",
  "failed",
  "canceled",
];

export const EXECUTION_MODES: ExecutionMode[] = ["manual", "scheduled", "yolo"];

/**
 * Maps execution status to tailwind background color classes.
 */
export const EXECUTION_STATUS_COLORS: Record<ExecutionStatus, string> = {
  pending: "bg-slate-500",
  scheduled: "bg-blue-500",
  running: "bg-cyan-500",
  completed: "bg-emerald-500",
  failed: "bg-red-500",
  canceled: "bg-amber-500",
};

/**
 * Formats an execution status for display.
 */
export function formatExecutionStatus(status: ExecutionStatus): string {
  return formatDisplayText(status);
}

/**
 * Formats an execution mode for display.
 */
export function formatExecutionMode(mode: ExecutionMode): string {
  if (mode === "yolo") {
    return "Start now";
  }
  return formatDisplayText(mode);
}

/**
 * Domain constants for Swarm Manager
 *
 * This module contains constants related to domain concepts.
 * Presentation-layer code should import these rather than defining their own.
 */

import { Activity, Bug, Circle, CheckCircle, AlertCircle, Cpu, Lightbulb, MessageSquare, Package, Rocket, Search, Target, Wrench, Zap, type LucideIcon } from "lucide-react";
import { formatDisplayText } from "../lib";
import type {
  AgentActivityStatus,
  AgentRunStatus,
  BacklogKind,
  BacklogStatus,
  CaptureStatus,
  ExecutionMode,
  ExecutionStatus,
  InitiativeStatus,
  ScenarioStatus,
} from "./domain";

// ============================================================================
// Backlog Status Display
// ============================================================================

export const BACKLOG_KINDS: BacklogKind[] = ["idea", "research", "fix", "execute", "chore"];

export const BACKLOG_KIND_LABELS: Record<BacklogKind, string> = {
  idea: "Idea",
  research: "Research",
  fix: "Fix",
  execute: "Execute",
  chore: "Chore",
};

export const BACKLOG_KIND_ICONS: Record<BacklogKind, LucideIcon> = {
  idea: Lightbulb,
  research: Search,
  fix: Bug,
  execute: Rocket,
  chore: Wrench,
};

export const BACKLOG_STATUSES: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "failed",
  "archived",
];

/** Statuses a user can manually set. Excludes queued/in_progress (managed by execution system). */
export const USER_SETTABLE_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready", "failed", "completed", "archived"];

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
  failed: "bg-red-600",
  archived: "bg-gray-600",
};

/**
 * Maps backlog status to semi-transparent chip color classes (bg + text).
 * Used for clickable dependency chips where the full-opacity badge would be too heavy.
 */
export const BACKLOG_STATUS_CHIP_COLORS: Record<BacklogStatus, string> = {
  backlog: "bg-slate-600/20 text-slate-300",
  researching: "bg-blue-600/20 text-blue-400",
  ready: "bg-green-600/20 text-green-400",
  queued: "bg-yellow-600/20 text-yellow-300",
  in_progress: "bg-purple-600/20 text-purple-400",
  completed: "bg-emerald-600/20 text-emerald-400",
  failed: "bg-red-600/20 text-red-400",
  archived: "bg-gray-600/20 text-gray-400",
};

/**
 * Formats a backlog status for display (converts underscores/hyphens to spaces, capitalizes).
 * Delegates to the shared formatDisplayText utility to prevent duplication.
 */
export function formatBacklogStatus(status: BacklogStatus): string {
  return formatDisplayText(status);
}

// ============================================================================
// Entity Type Display
// ============================================================================

/**
 * Canonical entity type identifiers used across the UI.
 * Must stay in sync with GraphEntityType in surfaces/graph/types.ts.
 */
export type EntityType =
  | "backlog"
  | "initiative"
  | "scenario"
  | "capture"
  | "execution"
  | "agent-run"
  | "agent-activity";

/**
 * Single source of truth for entity type icons.
 * Graph nodes, detail page headers, and any other entity-type-aware UI
 * should import from here rather than defining their own icon mappings.
 */
export const ENTITY_TYPE_ICONS: Record<EntityType, LucideIcon> = {
  backlog: Lightbulb,
  initiative: Target,
  scenario: Package,
  capture: MessageSquare,
  execution: Zap,
  "agent-run": Activity,
  "agent-activity": Cpu,
};

/**
 * Human-readable labels for entity types.
 */
export const ENTITY_TYPE_LABELS: Record<EntityType, string> = {
  backlog: "Backlog",
  initiative: "Initiative",
  scenario: "Scenario",
  capture: "Capture",
  execution: "Execution",
  "agent-run": "Run",
  "agent-activity": "Activity",
};

// ============================================================================
// Capture Status Display
// ============================================================================

export const CAPTURE_STATUSES: CaptureStatus[] = ["classifying", "classified", "failed"];

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
// Initiative Status Display
// ============================================================================

export const INITIATIVE_STATUSES: InitiativeStatus[] = ["active", "completed", "archived"];

/**
 * Maps initiative status to semi-transparent chip color classes (bg + text).
 */
export const INITIATIVE_STATUS_CHIP_COLORS: Record<string, string> = {
  active: "bg-sky-500/15 text-sky-400",
  completed: "bg-green-500/15 text-green-400",
  archived: "bg-slate-500/15 text-slate-500",
};

// ============================================================================
// Execution Status Display
// ============================================================================

export const EXECUTION_STATUSES: ExecutionStatus[] = [
  "pending",
  "starting",
  "running",
  "needs_review",
  "validating",
  "needs_fixup",
  "completed",
  "failed",
  "canceled",
];

export const EXECUTION_MODES: ExecutionMode[] = ["manual", "yolo"];

/**
 * Maps execution status to tailwind background color classes.
 */
export const EXECUTION_STATUS_COLORS: Record<ExecutionStatus, string> = {
  pending: "bg-slate-500",
  starting: "bg-violet-500",
  running: "bg-cyan-500",
  needs_review: "bg-yellow-500",
  validating: "bg-indigo-500",
  needs_fixup: "bg-orange-500",
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
    return "Auto";
  }
  return formatDisplayText(mode);
}

// ============================================================================
// Agent Status Display
// ============================================================================

export const AGENT_ACTIVITY_STATUSES: AgentActivityStatus[] = [
  "pending", "starting", "running", "needs_review", "complete", "failed", "cancelled", "unspecified",
];

export const AGENT_RUN_STATUSES: AgentRunStatus[] = [
  "pending", "starting", "running", "needs_review", "complete", "failed", "cancelled", "unspecified",
];

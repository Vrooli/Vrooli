/**
 * Domain constants for Swarm Manager
 *
 * This module contains constants related to domain concepts.
 * Presentation-layer code should import these rather than defining their own.
 */

import { Activity, AlertCircle, Archive, ArrowRight, Bot, Bug, Check, CheckCircle, CheckCircle2, Circle, ClipboardCheck, Cpu, Flag, Lightbulb, Link2, MessageCircleQuestion, MessageSquare, Package, PenLine, Play, Rocket, RotateCcw, Search, Send, Sparkles, Trophy, Wrench, Zap, type LucideIcon } from "lucide-react";
import { formatDisplayText } from "../lib";
import type { PlanGateKind } from "../surfaces/plan/types";
import type { BacklogKind, BacklogStatus } from "./backlog";
import type { CaptureStatus } from "./capture";
import type { ScenarioStatus } from "./scenario";
import type { ExecutionMode, ExecutionStatus } from "./execution";
import type { AgentActivityStatus, AgentRunStatus } from "./agent";

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

// The status vocabulary and its classification (which statuses are terminal,
// user-settable, queueable, …) are generated from the server's SSOT table.
// Re-exported here so the many existing importers of these names from
// types/constants keep working. To add a status, edit
// api/internal/backlogstatus/statuses.go and run `make gen-status`.
export {
  BACKLOG_STATUSES,
  BACKLOG_STATUS_LABELS,
  USER_SETTABLE_STATUSES,
  TERMINAL_STATUSES,
  RESOLVED_STATUSES,
  QUEUEABLE_BACKLOG_STATUSES,
  IN_FLIGHT_STATUSES,
  REVIEW_STATUSES,
} from "./backlog-status.generated";

/**
 * Maps backlog status to tailwind background color classes
 */
export const BACKLOG_STATUS_COLORS: Record<BacklogStatus, string> = {
  suggested: "bg-fuchsia-600",
  backlog: "bg-slate-600",
  researching: "bg-blue-600",
  ready: "bg-green-600",
  queued: "bg-yellow-600",
  in_progress: "bg-purple-600",
  in_review: "bg-amber-600",
  review_pending: "bg-cyan-600",
  completed: "bg-emerald-600",
  failed: "bg-red-600",
  needs_followup: "bg-orange-600",
  // Muted grey: dropped work is settled, not an outcome to draw the eye.
  dropped: "bg-zinc-600",
};

/**
 * Maps backlog status to semi-transparent chip color classes (bg + text).
 * Used for clickable dependency chips where the full-opacity badge would be too heavy.
 */
export const BACKLOG_STATUS_CHIP_COLORS: Record<BacklogStatus, string> = {
  suggested: "bg-fuchsia-500/20 text-fuchsia-300",
  backlog: "bg-slate-600/20 text-slate-300",
  researching: "bg-blue-600/20 text-blue-400",
  ready: "bg-green-600/20 text-green-400",
  queued: "bg-yellow-600/20 text-yellow-300",
  in_progress: "bg-purple-600/20 text-purple-400",
  in_review: "bg-amber-500/20 text-amber-400",
  review_pending: "bg-cyan-500/20 text-cyan-400",
  completed: "bg-emerald-600/20 text-emerald-400",
  failed: "bg-red-600/20 text-red-400",
  needs_followup: "bg-orange-500/20 text-orange-400",
  dropped: "bg-zinc-600/20 text-zinc-400",
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
 * The graph-node subset must stay in sync with GraphEntityType in
 * surfaces/graph/types.ts. "goal" is UI-only (goals are not graph nodes),
 * so it lives here for icon/label resolution but is absent from
 * GraphEntityType and ENTITY_REGISTRY.
 */
export type EntityType =
  | "backlog"
  | "goal"
  | "scenario"
  | "capture"
  | "execution"
  | "agent-run"
  | "agent-activity";

/**
 * Single source of truth for entity type icons.
 * Graph nodes, detail page headers, and any other entity-type-aware UI
 * should import from here rather than defining their own icon mappings.
 *
 * Every goal surface resolves its icon from this registry rather than
 * hardcoding an icon locally.
 */
export const ENTITY_TYPE_ICONS: Record<EntityType, LucideIcon> = {
  backlog: Lightbulb,
  goal: Trophy,
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
  goal: "Goal",
  scenario: "Scenario",
  capture: "Capture",
  execution: "Execution",
  "agent-run": "Run",
  "agent-activity": "Activity",
};

export type SidebarEntityType =
  | "backlog"
  | "captures"
  | "goals"
  | "executions"
  | "sessions";

/**
 * Icons for sidebar tabs. Entity-backed tabs derive from ENTITY_TYPE_ICONS so
 * a tab and its entity can never drift apart; the remaining tabs (operating
 * modes, sessions) have no entity-type equivalent and carry their own.
 */
export const SIDEBAR_TAB_ICONS: Record<SidebarEntityType, LucideIcon> = {
  backlog: ENTITY_TYPE_ICONS.backlog,
  captures: ENTITY_TYPE_ICONS.capture,
  goals: ENTITY_TYPE_ICONS.goal,
  executions: ENTITY_TYPE_ICONS.execution,
  sessions: Bot,
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
// Review Classification Display
// ============================================================================

export const REVIEW_CLASSIFICATION_LABELS: Record<string, string> = {
  ready: "Healthy",
  ready_with_notes: "Minor Issues",
  needs_work: "Needs Work",
  not_assessable: "Inconclusive",
};

export const REVIEW_CLASSIFICATION_COLORS: Record<string, string> = {
  ready: "bg-emerald-500/20 text-emerald-400",
  ready_with_notes: "bg-amber-500/20 text-amber-400",
  needs_work: "bg-red-500/20 text-red-400",
  not_assessable: "bg-slate-500/20 text-slate-400",
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

export const AGENT_ACTIVITY_STATUSES = [
  "pending", "starting", "running", "needs_review", "complete", "failed", "cancelled", "unspecified",
] as const satisfies readonly AgentActivityStatus[];

export const AGENT_RUN_STATUSES: AgentRunStatus[] = [
  "pending", "starting", "running", "needs_review", "complete", "failed", "cancelled", "unspecified",
];

// ============================================================================
// Plan Gate Display
// ============================================================================

/**
 * Display labels for plan-board gate kinds.
 *
 * The wire enum is deliberately stable — `workshop` still names "this item's
 * canonical plan needs authoring, validation, or fresh acceptance". Only the
 * operator-facing word changed to "Plan", so the label lives here rather than
 * being read off the enum at the render site.
 */
export const PLAN_GATE_LABELS: Record<PlanGateKind, string> = {
  decide: "Decide",
  proposal: "Proposal",
  review: "Review",
  classify: "Classify",
  workshop: "Plan",
};

/**
 * Labels for the `suggested` hint carried by a plan gate.
 *
 * Backlog-derived gates send `author-plan` / `accept-plan` / `validate-plan`;
 * goal-derived gates send the raw action name (`workshop`). Anything
 * unrecognised falls back to the gate-kind label.
 */
export const PLAN_GATE_SUGGESTION_LABELS: Record<string, string> = {
  "author-plan": "Author plan",
  "accept-plan": "Accept plan",
  "validate-plan": "Validate plan",
  workshop: "Plan",
};

// ============================================================================
// Next Action Display
// ============================================================================

/**
 * Icons for operator next-actions, keyed by `NextActionID`.
 *
 * Next-action labels arrive from the API per item, so any button rendering one
 * must resolve its icon here — otherwise the label ships without the leading
 * icon every other button in the app carries.
 */
export const NEXT_ACTION_ICONS: Record<string, LucideIcon> = {
  decide: MessageCircleQuestion,
  review: ClipboardCheck,
  accept_plan: CheckCircle2,
  author_plan: Sparkles,
  repair_plan: Wrench,
  plan_goal: Sparkles,
  run: Play,
  dispatch_followup: Send,
  author_followup: PenLine,
  resolve_dependencies: Link2,
  accept_suggestion: Check,
  retry: RotateCcw,
  archive: Archive,
  close_out: Flag,
};

/** Resolve a next-action icon, falling back to a neutral "proceed" glyph. */
export function nextActionIcon(actionId: string | undefined): LucideIcon {
  return (actionId && NEXT_ACTION_ICONS[actionId]) || ArrowRight;
}

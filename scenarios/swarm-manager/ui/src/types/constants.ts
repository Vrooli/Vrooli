/**
 * Domain constants for Swarm Manager
 *
 * This module contains constants related to domain concepts.
 * Presentation-layer code should import these rather than defining their own.
 */

import { Circle, CheckCircle, AlertCircle, type LucideIcon } from "lucide-react";
import { formatDisplayText } from "../lib";
import type { IdeaStatus, ScenarioStatus } from "./domain";

// ============================================================================
// Idea Status Display
// ============================================================================

/**
 * Maps idea status to tailwind background color classes
 */
export const IDEA_STATUS_COLORS: Record<IdeaStatus, string> = {
  backlog: "bg-slate-600",
  researching: "bg-blue-600",
  ready: "bg-green-600",
  queued: "bg-yellow-600",
  in_progress: "bg-purple-600",
  completed: "bg-emerald-600",
  archived: "bg-gray-600",
};

/**
 * Formats an idea status for display (converts underscores/hyphens to spaces, capitalizes).
 * Delegates to the shared formatDisplayText utility to prevent duplication.
 */
export function formatIdeaStatus(status: IdeaStatus): string {
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

/**
 * Maps scenario status to tailwind text color classes
 */
export const SCENARIO_STATUS_COLORS: Record<ScenarioStatus, string> = {
  running: "text-green-400",
  stopped: "text-slate-400",
  error: "text-red-400",
  unknown: "text-slate-500",
};

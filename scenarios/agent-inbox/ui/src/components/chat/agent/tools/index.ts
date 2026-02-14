/**
 * Tool component registry for agent event rendering.
 *
 * Maps tool names to specialized display components. Each component
 * receives an AgentEvent (tool_call) and an optional result (tool_result),
 * and renders a professional, collapsible card with tool-specific formatting.
 *
 * The `runnerType` parameter allows future per-runner routing if runners
 * emit conflicting event shapes for the same tool name.
 *
 * Tool name → Component mapping:
 * ─────────────────────────────────────────────────────
 *   Bash                      → AgentBashCard
 *   Read, Write, Edit         → AgentFileToolCard
 *   Glob, Grep                → AgentFileToolCard
 *   WebFetch, WebSearch       → AgentWebFetchCard
 *   Task                      → AgentTaskCard
 *   (everything else)         → AgentToolCallCard (generic fallback)
 */

import type { ComponentType } from "react";
import type { AgentEvent } from "../../../../lib/api";
import { AgentBashCard } from "./AgentBashCard";
import { AgentFileToolCard } from "./AgentFileToolCard";
import { AgentWebFetchCard } from "./AgentWebFetchCard";
import { AgentTaskCard } from "./AgentTaskCard";
import AgentToolCallCard from "../AgentToolCallCard";

/** Props shared by all tool card components. */
export interface ToolCardProps {
  event: AgentEvent;
  result?: AgentEvent;
}

type ToolCardComponent = ComponentType<ToolCardProps>;

/**
 * Static registry mapping tool names to their display component.
 * Add new entries here when supporting additional tool types.
 */
const TOOL_REGISTRY: Record<string, ToolCardComponent> = {
  // Shell execution
  Bash: AgentBashCard,

  // File operations
  Read: AgentFileToolCard,
  Write: AgentFileToolCard,
  Edit: AgentFileToolCard,
  Glob: AgentFileToolCard,
  Grep: AgentFileToolCard,

  // Web tools
  WebFetch: AgentWebFetchCard,
  WebSearch: AgentWebFetchCard,

  // Subagent/task spawning
  Task: AgentTaskCard,
};

/**
 * Returns the appropriate component for rendering a tool call event.
 *
 * @param toolName - The tool_name from the agent event (e.g. "Bash", "Read")
 * @param _runnerType - Optional runner type for future per-runner routing
 * @returns The component to render the tool call, or the generic fallback
 */
export function getToolComponent(
  toolName: string | undefined,
  _runnerType?: string,
): ToolCardComponent {
  if (!toolName) return AgentToolCallCard;
  return TOOL_REGISTRY[toolName] ?? AgentToolCallCard;
}

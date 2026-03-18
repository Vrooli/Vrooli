/**
 * Hook for grouping agent events by type for display.
 *
 * Groups tool_call and tool_result events together using tool_call_id
 * for reliable correlation, with a name+proximity fallback for events
 * that lack the ID.
 */

import { useMemo } from "react";
import type { AgentEvent } from "../../../lib/api";

export interface GroupedEvent {
  type: "message" | "tool" | "status" | "error" | "compaction" | "raw";
  event: AgentEvent;
  result?: AgentEvent;
}

export function useGroupedEvents(events: AgentEvent[]): GroupedEvent[] {
  return useMemo(() => {
    // Build a map of tool_call_id -> tool_result event for reliable matching
    const resultsByCallId = new Map<string, AgentEvent>();
    // Fallback: match by tool name + proximity for events without tool_call_id
    const resultsByNameFallback = new Map<string, AgentEvent>();

    // First pass: index tool results
    events.forEach((event, index) => {
      if (event.type !== "tool_result") return;

      // Prefer tool_call_id-based matching
      if (event.tool_call_id) {
        resultsByCallId.set(event.tool_call_id, event);
        return;
      }

      // Fallback: match by tool name + proximity (within 10 events)
      if (event.tool_name) {
        for (let i = index - 1; i >= Math.max(0, index - 10); i--) {
          const prevEvent = events[i];
          if (prevEvent === undefined) continue;
          if (
            prevEvent.type === "tool_call" &&
            prevEvent.tool_name === event.tool_name &&
            !resultsByNameFallback.has(prevEvent.id)
          ) {
            resultsByNameFallback.set(prevEvent.id, event);
            break;
          }
        }
      }
    });

    // Second pass: create grouped items
    const grouped: GroupedEvent[] = [];
    const renderedResultIds = new Set<string>();

    events.forEach(event => {
      if (event.type === "message") {
        grouped.push({ type: "message", event });
      } else if (event.type === "tool_call") {
        // Try tool_call_id first, then fallback map
        const result = (event.tool_call_id && resultsByCallId.get(event.tool_call_id))
          || resultsByNameFallback.get(event.id);
        if (result) {
          renderedResultIds.add(result.id);
        }
        grouped.push({ type: "tool", event, result });
      } else if (event.type === "tool_result") {
        // Skip if already rendered with a tool_call
        if (!renderedResultIds.has(event.id)) {
          // Orphan result - render as standalone
          grouped.push({ type: "tool", event, result: event });
        }
      } else if (event.type === "compaction") {
        grouped.push({ type: "compaction", event });
      } else if (event.type === "status") {
        grouped.push({ type: "status", event });
      } else if (event.type === "error") {
        grouped.push({ type: "error", event });
      } else if (event.type === "log" || event.type === "metric") {
        // Skip log events (internal debug info) and metric events (shown in header)
        return;
      } else {
        // All other event types: render as generic raw card
        grouped.push({ type: "raw", event });
      }
    });

    return grouped;
  }, [events]);
}

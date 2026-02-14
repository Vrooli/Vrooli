import { useRef, useEffect, useMemo } from "react";
import type { AgentEvent } from "../../../lib/api";
import AgentMessageBubble from "./AgentMessageBubble";
import AgentToolCallCard from "./AgentToolCallCard";
import AgentRawEventCard from "./AgentRawEventCard";
import { getToolComponent } from "./tools";

/** Parsed metric from raw_data JSON. */
export interface AgentMetric {
  name: string;
  value: number;
  unit: string;
  tags?: Record<string, string>;
}

interface AgentEventListProps {
  /** List of events to render */
  events: AgentEvent[];
  /** Whether to auto-scroll to bottom on new events */
  autoScroll?: boolean;
  /** Runner type for tool-specific rendering (e.g. "claude_code", "codex") */
  runnerType?: string;
}

/**
 * Renders a list of agent events as chat messages and tool calls.
 * Groups tool_call and tool_result events together using tool_call_id
 * for reliable correlation, with a name+proximity fallback for events
 * that lack the ID.
 */
export function AgentEventList({ events, autoScroll = true, runnerType }: AgentEventListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const prevEventCountRef = useRef(events.length);

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    if (autoScroll && events.length > prevEventCountRef.current) {
      containerRef.current?.scrollTo({
        top: containerRef.current.scrollHeight,
        behavior: "smooth"
      });
    }
    prevEventCountRef.current = events.length;
  }, [events.length, autoScroll]);

  // Group tool calls with their results
  const groupedEvents = useMemo(() => {
    // Build a map of tool_call_id → tool_result event for reliable matching
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
    const grouped: Array<{
      type: "message" | "tool" | "status" | "error" | "raw";
      event: AgentEvent;
      result?: AgentEvent;
    }> = [];

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
      } else if (event.type === "status") {
        // Status events are shown in the header, skip inline
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

  if (events.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500">
        <p>Waiting for agent response...</p>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="flex-1 overflow-y-auto p-4 space-y-4">
      {groupedEvents.map((item, index) => {
        switch (item.type) {
          case "message":
            return <AgentMessageBubble key={item.event.id || index} event={item.event} />;

          case "tool": {
            const ToolComponent = getToolComponent(item.event.tool_name, runnerType);
            return (
              <ToolComponent
                key={item.event.id || index}
                event={item.event}
                result={item.result}
              />
            );
          }

          case "status":
            // Status events are typically shown in the header, skip inline
            return null;

          case "error":
            return (
              <div
                key={item.event.id || index}
                className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-300"
              >
                <div className="text-sm font-medium">Error</div>
                <div className="text-sm">{item.event.content}</div>
              </div>
            );

          case "raw":
            return <AgentRawEventCard key={item.event.id || index} event={item.event} />;

          default:
            return null;
        }
      })}
    </div>
  );
}

export default AgentEventList;

import { useRef, useEffect, useMemo } from "react";
import type { AgentEvent } from "../../../lib/api";
import AgentMessageBubble from "./AgentMessageBubble";
import AgentToolCallCard from "./AgentToolCallCard";

interface AgentEventListProps {
  /** List of events to render */
  events: AgentEvent[];
  /** Whether to auto-scroll to bottom on new events */
  autoScroll?: boolean;
}

/**
 * Renders a list of agent events as chat messages and tool calls.
 * Groups tool_call and tool_result events together for better UX.
 */
export function AgentEventList({ events, autoScroll = true }: AgentEventListProps) {
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
    const toolResults = new Map<string, AgentEvent>();

    // First pass: collect tool results by tool name + close proximity
    // (Simple heuristic - match by tool name and within 5 events)
    events.forEach((event, index) => {
      if (event.type === "tool_result" && event.tool_name) {
        // Find the most recent unmatched tool_call with same name
        for (let i = index - 1; i >= Math.max(0, index - 10); i--) {
          const prevEvent = events[i];
          if (prevEvent === undefined) continue;
          if (
            prevEvent.type === "tool_call" &&
            prevEvent.tool_name === event.tool_name &&
            !toolResults.has(prevEvent.id)
          ) {
            toolResults.set(prevEvent.id, event);
            break;
          }
        }
      }
    });

    // Second pass: create grouped items
    const grouped: Array<{
      type: "message" | "tool" | "status" | "error";
      event: AgentEvent;
      result?: AgentEvent;
    }> = [];

    const renderedResultIds = new Set<string>();

    events.forEach(event => {
      if (event.type === "message") {
        grouped.push({ type: "message", event });
      } else if (event.type === "tool_call") {
        const result = toolResults.get(event.id);
        if (result) {
          renderedResultIds.add(result.id);
        }
        grouped.push({ type: "tool", event, result });
      } else if (event.type === "tool_result") {
        // Skip if already rendered with a tool_call
        if (!renderedResultIds.has(event.id)) {
          // Orphan result - render as standalone
          grouped.push({ type: "tool", event: event, result: event });
        }
      } else if (event.type === "status") {
        grouped.push({ type: "status", event });
      } else if (event.type === "error") {
        grouped.push({ type: "error", event });
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

          case "tool":
            return (
              <AgentToolCallCard
                key={item.event.id || index}
                event={item.event}
                result={item.result}
              />
            );

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

          default:
            return null;
        }
      })}
    </div>
  );
}

export default AgentEventList;

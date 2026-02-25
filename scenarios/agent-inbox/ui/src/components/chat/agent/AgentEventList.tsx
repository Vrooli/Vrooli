import { useRef, useEffect, useMemo, useState } from "react";
import type { AgentEvent, Message } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import AgentMessageBubble from "./AgentMessageBubble";
import AgentCompactionCard from "./AgentCompactionCard";
import AgentRawEventCard from "./AgentRawEventCard";
import { getToolComponent } from "./tools";
import { MarkdownRenderer } from "../../markdown/MarkdownRenderer";
import { User } from "lucide-react";

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
  /** Message rendering style (bubble or compact) */
  viewMode?: ViewMode;
  /** Initial messages (e.g. user's prompt) to render before agent events */
  initialMessages?: Message[];
  /** ID of a message to scroll to and highlight (from search navigation) */
  scrollToMessageId?: string | null;
  /** Called after scroll-to-message completes or gives up */
  onScrollComplete?: () => void;
}

/**
 * Renders a list of agent events as chat messages and tool calls.
 * Groups tool_call and tool_result events together using tool_call_id
 * for reliable correlation, with a name+proximity fallback for events
 * that lack the ID.
 */
export function AgentEventList({
  events,
  autoScroll = true,
  runnerType,
  viewMode = "bubble",
  initialMessages,
  scrollToMessageId,
  onScrollComplete,
}: AgentEventListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const prevEventCountRef = useRef(events.length);
  const isCompact = viewMode === "compact";

  // Ref map for scroll-to-message support (keyed by message ID or event ID)
  const elementRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Track which element is highlighted (from search result navigation)
  const [highlightedId, setHighlightedId] = useState<string | null>(null);

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

  // Scroll to target message/event when scrollToMessageId is set.
  // Uses retry polling because the target element may not be in the DOM yet.
  useEffect(() => {
    if (!scrollToMessageId) return;

    let attempts = 0;
    const maxAttempts = 30; // ~3 seconds
    let timerId: number;
    let highlightTimerId: number;

    const tryScroll = () => {
      const el = elementRefs.current.get(scrollToMessageId);
      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" });
        setHighlightedId(scrollToMessageId);
        highlightTimerId = window.setTimeout(() => {
          setHighlightedId(null);
          onScrollComplete?.();
        }, 2000);
        return;
      }
      attempts++;
      if (attempts < maxAttempts) {
        timerId = window.setTimeout(tryScroll, 100);
      } else {
        onScrollComplete?.();
      }
    };

    tryScroll();
    return () => {
      clearTimeout(timerId);
      clearTimeout(highlightTimerId);
    };
  }, [scrollToMessageId, onScrollComplete]);

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
      type: "message" | "tool" | "status" | "error" | "compaction" | "raw";
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
      } else if (event.type === "compaction") {
        grouped.push({ type: "compaction", event });
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

  if ((!initialMessages || initialMessages.length === 0) && events.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-500">
        <p>Waiting for agent response...</p>
      </div>
    );
  }

  const highlightClass = "ring-4 ring-yellow-400 rounded-xl bg-yellow-400/20 shadow-[0_0_15px_rgba(250,204,21,0.4)]";

  return (
    <div
      ref={containerRef}
      className={`flex-1 w-full min-w-0 max-w-full overflow-y-auto overflow-x-auto ${isCompact ? "p-3 space-y-2" : "p-4 space-y-4"}`}
      data-testid="agent-event-list"
    >
      {/* Render initial messages (user's prompt) before agent events */}
      {initialMessages?.map((msg) => {
        const isHighlighted = msg.id === highlightedId;
        return (
          <div
            key={msg.id}
            ref={(el) => {
              if (el) elementRefs.current.set(msg.id, el);
              else elementRefs.current.delete(msg.id);
            }}
            className={`transition-all duration-300 ${isHighlighted ? highlightClass : ""}`}
          >
            {isCompact ? (
              <div className="border-l-2 border-l-blue-500 pl-3 py-1">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-medium text-blue-400">You</span>
                  <span className="text-xs text-zinc-500">
                    {new Date(msg.created_at).toLocaleTimeString()}
                  </span>
                </div>
                <div className="text-sm text-zinc-100">
                  <MarkdownRenderer content={msg.content} />
                </div>
              </div>
            ) : (
              <div className="flex gap-3 flex-row-reverse min-w-0 max-w-full w-full">
                <div className="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center bg-blue-600">
                  <User className="h-4 w-4 text-white" />
                </div>
                <div className="w-0 min-w-0 flex-1 max-w-full rounded-2xl px-4 py-2 bg-blue-600 text-white rounded-br-md overflow-x-auto">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs text-blue-200">You</span>
                  </div>
                  <div className="min-w-0 max-w-full text-sm break-words [overflow-wrap:anywhere]">
                    <MarkdownRenderer content={msg.content} />
                  </div>
                  <div className="text-xs mt-1 text-blue-200">
                    {new Date(msg.created_at).toLocaleTimeString()}
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      })}

      {/* Render grouped agent events */}
      {groupedEvents.map((item, index) => {
        switch (item.type) {
          case "message":
            return (
              <AgentMessageBubble
                key={item.event.id || index}
                event={item.event}
                viewMode={viewMode}
              />
            );

          case "tool": {
            const ToolComponent = getToolComponent(item.event.tool_name, runnerType);
            return (
              <ToolComponent
                key={item.event.id || index}
                event={item.event}
                result={item.result}
                viewMode={viewMode}
              />
            );
          }

          case "compaction":
            return (
              <AgentCompactionCard
                key={item.event.id || index}
                event={item.event}
                viewMode={viewMode}
              />
            );

          case "status":
            // Status events are typically shown in the header, skip inline
            return null;

          case "error":
            return (
              <div
                key={item.event.id || index}
                className={isCompact
                  ? "px-2 py-1 rounded border border-red-500/25 bg-red-500/5 text-red-300"
                  : "p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-300"
                }
              >
                <div className="text-sm font-medium">Error</div>
                <div className="text-sm">{item.event.content}</div>
              </div>
            );

          case "raw":
            return (
              <AgentRawEventCard
                key={item.event.id || index}
                event={item.event}
                viewMode={viewMode}
              />
            );

          default:
            return null;
        }
      })}
    </div>
  );
}

export default AgentEventList;

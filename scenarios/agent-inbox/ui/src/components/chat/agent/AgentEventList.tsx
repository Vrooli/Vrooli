import { useRef, useEffect, useState } from "react";
import type { AgentEvent, Message } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import AgentMessageBubble from "./AgentMessageBubble";
import AgentCompactionCard from "./AgentCompactionCard";
import AgentRawEventCard from "./AgentRawEventCard";
import { getToolComponent } from "./tools";
import { MarkdownRenderer } from "../../markdown/MarkdownRenderer";
import { User } from "lucide-react";
import { useGroupedEvents } from "./useGroupedEvents";

/** Parsed metric from raw_data JSON. */
export interface AgentMetric {
  name: string;
  value: number;
  unit: string;
  tags?: Record<string, string>;
}

interface AgentEventListProps {
  events: AgentEvent[];
  autoScroll?: boolean;
  runnerType?: string;
  viewMode?: ViewMode;
  initialMessages?: Message[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
}

/**
 * Renders a list of agent events as chat messages and tool calls.
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

  const elementRefs = useRef<Map<string, HTMLDivElement>>(new Map());
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

  // Scroll to target message/event
  useEffect(() => {
    if (!scrollToMessageId) return;

    let attempts = 0;
    const maxAttempts = 30;
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

  const groupedEvents = useGroupedEvents(events);

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

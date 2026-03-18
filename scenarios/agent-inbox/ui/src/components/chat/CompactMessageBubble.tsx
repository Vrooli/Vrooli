import { Wrench } from "lucide-react";
import type { Message, ToolCall, ToolCallRecord } from "../../lib/api";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import { MarkdownRenderer, CodeBlock } from "../markdown";
import { ToolCallItem } from "./ToolCallItem";
import { MessageAttachments } from "./MessageAttachments";
import { formatTime } from "./messageListTypes";

interface CompactMessageBubbleProps {
  message: Message;
  isUser: boolean;
  isTool: boolean;
  hasToolCalls: boolean;
  toolCallRecordMap: Map<string, ToolCallRecord>;
  asyncOperationMap: Map<string, AsyncStatusUpdate>;
  onOpenAsyncDrawer?: (operation: AsyncStatusUpdate) => void;
  highlightClass: string;
  renderActions: (position: "user" | "assistant" | "tool") => React.ReactNode;
  forwardedRef: React.ForwardedRef<HTMLDivElement>;
}

export function CompactMessageBubble({
  message,
  isUser,
  isTool,
  hasToolCalls,
  toolCallRecordMap,
  asyncOperationMap,
  onOpenAsyncDrawer,
  highlightClass,
  renderActions,
  forwardedRef,
}: CompactMessageBubbleProps) {
  const roleLabel = isUser ? "You" : isTool ? "Tool" : "Assistant";
  const roleColor = isUser ? "text-indigo-400" : isTool ? "text-amber-400" : "text-emerald-400";
  const borderColor = isUser ? "border-l-indigo-500" : isTool ? "border-l-amber-500" : "border-l-emerald-500";

  return (
    <div
      ref={forwardedRef}
      className={`group transition-all duration-300 border-l-2 ${borderColor} pl-3 py-1 ${highlightClass}`}
      data-testid={`message-${message.id}`}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className={`text-xs font-medium ${roleColor}`}>{roleLabel}</span>
        <span className="text-xs text-slate-500 dark:text-slate-600">{formatTime(message.created_at)}</span>
        <div className="flex-1" />
        {renderActions(isUser ? "user" : isTool ? "tool" : "assistant")}
      </div>
      {isTool ? (
        <div className="max-h-80 overflow-y-auto">
          <CodeBlock code={(() => {
            const content = typeof message.content === "string"
              ? message.content
              : (message.content ? JSON.stringify(message.content, null, 2) : "");
            try {
              const parsed: unknown = JSON.parse(content);
              return JSON.stringify(parsed, null, 2);
            } catch {
              return content;
            }
          })()} language="json" />
        </div>
      ) : (
        <div className="text-sm text-slate-700 dark:text-slate-200">
          <MessageAttachments attachments={message.attachments} isUser={isUser} compact />
          <MarkdownRenderer content={message.content} />
        </div>
      )}
      {hasToolCalls && (
        <div className="mt-2 pl-2 border-l border-amber-500/30">
          <div className="text-xs text-amber-500 dark:text-amber-400 mb-1 flex items-center gap-1">
            <Wrench className="h-3 w-3" />
            Using tools
          </div>
          {(message.tool_calls ?? []).map((tc: ToolCall) => (
            <ToolCallItem
              key={tc.id}
              toolCall={tc}
              record={toolCallRecordMap.get(tc.id)}
              asyncOperation={asyncOperationMap.get(tc.id)}
              variant="compact"
              onOpenAsyncDrawer={onOpenAsyncDrawer}
            />
          ))}
        </div>
      )}
    </div>
  );
}

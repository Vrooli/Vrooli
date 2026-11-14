import { User, Bot, Wrench } from "lucide-react";
import type { Message, ToolCall, ToolCallRecord } from "../../lib/api";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import { MarkdownRenderer, CodeBlock } from "../markdown";
import { ToolCallItem } from "./ToolCallItem";
import { MessageAttachments } from "./MessageAttachments";
import { formatTime } from "./messageListTypes";

interface BubbleMessageBubbleProps {
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

/** Format tool content as JSON if possible */
function formatToolContent(content: unknown): string {
  if (typeof content === "string") {
    try {
      const parsed: unknown = JSON.parse(content);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return content;
    }
  } else if (content && typeof content === "object") {
    return JSON.stringify(content, null, 2);
  }
  if (content === null || content === undefined) {
    return "";
  }
  if (typeof content === "number" || typeof content === "boolean" || typeof content === "bigint") {
    return String(content);
  }
  return JSON.stringify(content);
}

export function BubbleMessageBubble({
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
}: BubbleMessageBubbleProps) {
  // Tool response messages
  if (isTool) {
    return (
      <div ref={forwardedRef} className={`group flex justify-start transition-all duration-300 ${highlightClass}`} data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%] min-w-0">
          <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
            <Wrench className="h-4 w-4 text-amber-500 dark:text-amber-400" />
          </div>
          <div className="text-slate-600 dark:text-slate-300 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs text-amber-600 dark:text-amber-400">Tool Result</span>
            </div>
            <div className="max-h-80 overflow-y-auto">
              <CodeBlock code={formatToolContent(message.content)} language="json" />
            </div>
            <p className="text-xs mt-1.5 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
          </div>
        </div>
      </div>
    );
  }

  // Assistant message with tool calls
  if (hasToolCalls) {
    return (
      <div ref={forwardedRef} className={`group flex justify-start transition-all duration-300 ${highlightClass}`} data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%] min-w-0">
          <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
            <Bot className="h-4 w-4 text-indigo-400" />
          </div>
          <div className="space-y-2 min-w-0">
            {(message.content || (message.attachments && message.attachments.length > 0)) && (
              <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200 min-w-0">
                <div className="flex items-center justify-end gap-2 mb-1">
                  {renderActions("assistant")}
                </div>
                <MessageAttachments attachments={message.attachments} />
                {message.content && <MarkdownRenderer content={message.content} />}
              </div>
            )}
            <div className="bg-amber-50 dark:bg-slate-800/60 border border-amber-200 dark:border-amber-500/20 rounded-xl px-4 py-3">
              <div className="text-xs text-amber-600 dark:text-amber-400 mb-2 flex items-center gap-1.5">
                <Wrench className="h-3 w-3" />
                Using tools
              </div>
              {(message.tool_calls ?? []).map((tc: ToolCall) => (
                <ToolCallItem
                  key={tc.id}
                  toolCall={tc}
                  record={toolCallRecordMap.get(tc.id)}
                  asyncOperation={asyncOperationMap.get(tc.id)}
                  variant="bubble"
                  onOpenAsyncDrawer={onOpenAsyncDrawer}
                />
              ))}
              <p className="text-xs mt-2 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Regular user/assistant messages
  return (
    <div
      ref={forwardedRef}
      className={`group flex ${isUser ? "justify-end" : "justify-start"} transition-all duration-300 ${highlightClass}`}
      data-testid={`message-${message.id}`}
    >
      <div className={`flex gap-3 max-w-[85%] min-w-0 ${isUser ? "flex-row-reverse" : ""}`}>
        {/* Avatar */}
        <div
          className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${
            isUser ? "bg-indigo-600" : "bg-indigo-500/20"
          }`}
        >
          {isUser ? (
            <User className="h-4 w-4 text-white" />
          ) : (
            <Bot className="h-4 w-4 text-indigo-400" />
          )}
        </div>

        {/* Message Content */}
        <div
          className={`rounded-2xl px-4 py-3 ${
            isUser
              ? "bg-indigo-600 text-white rounded-tr-md"
              : "bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-200 rounded-tl-md"
          } min-w-0`}
        >
          <div className={`flex items-center ${isUser ? "justify-start" : "justify-end"} gap-2 mb-1`}>
            {renderActions(isUser ? "user" : "assistant")}
          </div>
          <MessageAttachments attachments={message.attachments} isUser={isUser} />
          <MarkdownRenderer content={message.content} />
          <p
            className={`text-xs mt-1.5 ${
              isUser ? "text-indigo-200/60" : "text-slate-400 dark:text-slate-500"
            }`}
          >
            {formatTime(message.created_at)}
          </p>
        </div>
      </div>
    </div>
  );
}

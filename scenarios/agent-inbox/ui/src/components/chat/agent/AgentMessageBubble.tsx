import { Bot, User, Copy, Check } from "lucide-react";
import { useCallback, useState } from "react";
import type { AgentEvent } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import { MarkdownRenderer } from "../../markdown/MarkdownRenderer";
import { Tooltip } from "../../ui/tooltip";
import { useToast } from "../../ui/toast";

interface AgentMessageBubbleProps {
  event: AgentEvent;
  viewMode?: ViewMode;
}

/**
 * Renders a message event from an agent run as a chat bubble.
 */
export function AgentMessageBubble({ event, viewMode = "bubble" }: AgentMessageBubbleProps) {
  const isUser = event.role === "user";
  const isSystem = event.role === "system";
  const isCompact = viewMode === "compact";
  const { addToast } = useToast();
  const [copied, setCopied] = useState(false);

  const roleLabel = isUser ? "You" : isSystem ? "System" : "Agent";

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(event.content || "");
      setCopied(true);
      addToast("Copied to clipboard", "success");
      setTimeout(() => setCopied(false), 1500);
    } catch {
      addToast("Failed to copy", "error");
    }
  }, [event.content, addToast]);

  if (isCompact) {
    const borderColor = isUser ? "border-l-blue-500" : isSystem ? "border-l-zinc-500" : "border-l-emerald-500";
    const roleColor = isUser ? "text-blue-400" : isSystem ? "text-zinc-400" : "text-emerald-400";

    return (
      <div className={`group transition-all duration-200 border-l-2 ${borderColor} pl-3 py-1 min-w-0 max-w-full overflow-x-auto`}>
        <div className="flex items-center gap-2 mb-1">
          <span className={`text-xs font-medium ${roleColor}`}>{roleLabel}</span>
          <span className="text-xs text-zinc-500">{new Date(event.timestamp).toLocaleTimeString()}</span>
          <div className="flex-1" />
          <Tooltip content={copied ? "Copied" : "Copy message"} side="top">
            <button
              onClick={() => { void handleCopy(); }}
              className="p-1 rounded text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800/70 transition-colors"
              aria-label="Copy message"
              data-testid={`agent-message-copy-${event.id}`}
            >
              {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
            </button>
          </Tooltip>
        </div>

        {isSystem ? (
          <p className="text-sm italic text-zinc-300">{event.content}</p>
        ) : (
          <div className="min-w-0 max-w-full text-sm text-zinc-100 break-words [overflow-wrap:anywhere]">
            <MarkdownRenderer content={event.content} />
          </div>
        )}
      </div>
    );
  }

  return (
    <div className={`group flex gap-3 min-w-0 max-w-full w-full ${isUser ? "flex-row-reverse" : ""}`}>
      {/* Avatar */}
      <div
        className={`
          flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center
          ${isUser ? "bg-blue-600" : isSystem ? "bg-zinc-600" : "bg-emerald-600"}
        `}
      >
        {isUser ? (
          <User className="h-4 w-4 text-white" />
        ) : (
          <Bot className="h-4 w-4 text-white" />
        )}
      </div>

      {/* Message bubble */}
      <div
        className={`
          w-0 min-w-0 flex-1 max-w-full rounded-2xl px-4 py-2 overflow-x-auto
          ${isUser
            ? "bg-blue-600 text-white rounded-br-md"
            : isSystem
              ? "bg-zinc-700 text-zinc-300 rounded-bl-md"
              : "bg-zinc-800 text-zinc-100 rounded-bl-md"
          }
        `}
      >
        <div className="flex items-center gap-2 mb-1">
          <span className={`text-xs ${isUser ? "text-blue-200" : "text-zinc-400"}`}>
            {roleLabel}
          </span>
          <div className="flex-1" />
          <Tooltip content={copied ? "Copied" : "Copy message"} side="top">
            <button
              onClick={() => { void handleCopy(); }}
              className={`p-1 rounded transition-colors ${
                isUser
                  ? "text-blue-200 hover:text-white hover:bg-blue-500/50"
                  : "text-zinc-500 hover:text-zinc-300 hover:bg-zinc-700"
              }`}
              aria-label="Copy message"
              data-testid={`agent-message-copy-${event.id}`}
            >
              {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
            </button>
          </Tooltip>
        </div>

        {isSystem ? (
          <p className="text-sm italic">{event.content}</p>
        ) : (
          <div className="min-w-0 max-w-full text-sm break-words [overflow-wrap:anywhere]">
            <MarkdownRenderer content={event.content} />
          </div>
        )}

        {/* Timestamp */}
        <div className={`text-xs mt-1 ${isUser ? "text-blue-200" : "text-zinc-500"}`}>
          {new Date(event.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
}

export default AgentMessageBubble;

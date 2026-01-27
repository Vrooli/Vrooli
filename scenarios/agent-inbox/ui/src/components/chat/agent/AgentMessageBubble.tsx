import { Bot, User } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";
import { MarkdownRenderer } from "../../markdown/MarkdownRenderer";

interface AgentMessageBubbleProps {
  event: AgentEvent;
}

/**
 * Renders a message event from an agent run as a chat bubble.
 */
export function AgentMessageBubble({ event }: AgentMessageBubbleProps) {
  const isUser = event.role === "user";
  const isSystem = event.role === "system";

  return (
    <div className={`flex gap-3 ${isUser ? "flex-row-reverse" : ""}`}>
      {/* Avatar */}
      <div
        className={`
          flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center
          ${isUser ? "bg-blue-600" : isSystem ? "bg-zinc-600" : "bg-purple-600"}
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
          max-w-[80%] rounded-2xl px-4 py-2
          ${isUser
            ? "bg-blue-600 text-white rounded-br-md"
            : isSystem
              ? "bg-zinc-700 text-zinc-300 rounded-bl-md"
              : "bg-zinc-800 text-zinc-100 rounded-bl-md"
          }
        `}
      >
        {isSystem ? (
          <p className="text-sm italic">{event.content}</p>
        ) : (
          <div className="prose prose-invert prose-sm max-w-none">
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

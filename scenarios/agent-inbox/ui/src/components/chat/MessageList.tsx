import { useEffect, useRef } from "react";
import { Loader2, User, Bot, Terminal } from "lucide-react";
import type { Message } from "../../lib/api";

interface MessageListProps {
  messages: Message[];
  isGenerating: boolean;
  streamingContent: string;
  viewMode: "bubble" | "terminal";
}

export function MessageList({
  messages,
  isGenerating,
  streamingContent,
  viewMode,
}: MessageListProps) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamingContent]);

  if (messages.length === 0 && !isGenerating) {
    return (
      <div className="flex-1 flex items-center justify-center p-8" data-testid="empty-messages">
        <div className="text-center max-w-sm">
          <div className="w-16 h-16 rounded-2xl bg-indigo-500/20 flex items-center justify-center mx-auto mb-4">
            <Bot className="h-8 w-8 text-indigo-400" />
          </div>
          <h3 className="text-lg font-medium text-white mb-2">Start a conversation</h3>
          <p className="text-sm text-slate-500">
            Type a message below to begin chatting with the AI assistant. Ask questions, request
            help with tasks, or just have a conversation.
          </p>
        </div>
      </div>
    );
  }

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  if (viewMode === "terminal") {
    return (
      <div className="flex-1 overflow-y-auto p-4 font-mono text-sm" data-testid="message-list">
        {messages.map((message) => (
          <div key={message.id} className="mb-3" data-testid={`message-${message.id}`}>
            <span className="text-slate-500">[{formatTime(message.created_at)}]</span>{" "}
            <span
              className={
                message.role === "user"
                  ? "text-green-400"
                  : message.role === "assistant"
                  ? "text-blue-400"
                  : "text-yellow-400"
              }
            >
              {message.role === "user" ? "you" : message.role === "assistant" ? "ai" : "system"}:
            </span>{" "}
            <span className="text-slate-200 whitespace-pre-wrap">{message.content}</span>
          </div>
        ))}

        {isGenerating && (
          <div className="mb-3" data-testid="streaming-message">
            <span className="text-slate-500">[now]</span>{" "}
            <span className="text-blue-400">ai:</span>{" "}
            <span className="text-slate-200">
              {streamingContent || (
                <span className="text-slate-500 animate-pulse">thinking...</span>
              )}
            </span>
          </div>
        )}
        <div ref={endRef} />
      </div>
    );
  }

  // Bubble mode (default)
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4" data-testid="message-list">
      {messages.map((message) => (
        <MessageBubble key={message.id} message={message} />
      ))}

      {isGenerating && (
        <div className="flex justify-start" data-testid="streaming-message">
          <div className="flex gap-3 max-w-[85%]">
            <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
              <Bot className="h-4 w-4 text-indigo-400" />
            </div>
            <div className="bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-200">
              {streamingContent ? (
                <p className="whitespace-pre-wrap">{streamingContent}</p>
              ) : (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin text-indigo-400" />
                  <span className="text-sm text-slate-400">Thinking...</span>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
      <div ref={endRef} />
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  if (isSystem) {
    return (
      <div className="flex justify-center" data-testid={`message-${message.id}`}>
        <div className="bg-slate-800/50 rounded-lg px-4 py-2 text-sm text-slate-400 italic max-w-[80%]">
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`flex ${isUser ? "justify-end" : "justify-start"}`}
      data-testid={`message-${message.id}`}
    >
      <div className={`flex gap-3 max-w-[85%] ${isUser ? "flex-row-reverse" : ""}`}>
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
              : "bg-slate-800 text-slate-200 rounded-tl-md"
          }`}
        >
          <p className="whitespace-pre-wrap">{message.content}</p>
          <p
            className={`text-xs mt-1.5 ${
              isUser ? "text-indigo-200/60" : "text-slate-500"
            }`}
          >
            {formatTime(message.created_at)}
          </p>
        </div>
      </div>
    </div>
  );
}

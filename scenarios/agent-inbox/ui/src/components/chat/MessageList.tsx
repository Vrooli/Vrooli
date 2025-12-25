import { useEffect, useRef } from "react";
import { Loader2, User, Bot, Terminal, Wrench, CheckCircle2, XCircle, Play } from "lucide-react";
import type { Message, ToolCall } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";

interface MessageListProps {
  messages: Message[];
  isGenerating: boolean;
  streamingContent: string;
  viewMode: "bubble" | "terminal";
  activeToolCalls?: ActiveToolCall[];
}

export function MessageList({
  messages,
  isGenerating,
  streamingContent,
  viewMode,
  activeToolCalls = [],
}: MessageListProps) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamingContent, activeToolCalls]);

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
          <TerminalMessage key={message.id} message={message} formatTime={formatTime} />
        ))}

        {/* Active tool calls during generation */}
        {activeToolCalls.map((tc) => (
          <div key={tc.id} className="mb-3" data-testid={`tool-call-${tc.id}`}>
            <span className="text-slate-500">[now]</span>{" "}
            <span className="text-amber-400">tool:</span>{" "}
            <span className="text-slate-200">
              {tc.status === "running" ? (
                <span className="animate-pulse">Running {tc.name}...</span>
              ) : tc.status === "completed" ? (
                <span className="text-green-400">✓ {tc.name} completed</span>
              ) : (
                <span className="text-red-400">✗ {tc.name} failed: {tc.error}</span>
              )}
            </span>
          </div>
        ))}

        {isGenerating && !activeToolCalls.length && (
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

      {/* Active tool calls during generation */}
      {activeToolCalls.length > 0 && (
        <div className="flex justify-start" data-testid="active-tool-calls">
          <div className="flex gap-3 max-w-[85%]">
            <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
              <Wrench className="h-4 w-4 text-amber-400" />
            </div>
            <div className="bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-200 space-y-2">
              {activeToolCalls.map((tc) => (
                <div key={tc.id} className="flex items-center gap-2">
                  {tc.status === "running" ? (
                    <>
                      <Play className="h-4 w-4 text-amber-400 animate-pulse" />
                      <span className="text-sm">Running <code className="bg-slate-700 px-1.5 py-0.5 rounded">{tc.name}</code>...</span>
                    </>
                  ) : tc.status === "completed" ? (
                    <>
                      <CheckCircle2 className="h-4 w-4 text-green-400" />
                      <span className="text-sm text-green-400">{tc.name} completed</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="h-4 w-4 text-red-400" />
                      <span className="text-sm text-red-400">{tc.name} failed</span>
                    </>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {isGenerating && !activeToolCalls.length && (
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

function TerminalMessage({ message, formatTime }: { message: Message; formatTime: (s: string) => string }) {
  const roleConfig = {
    user: { color: "text-green-400", label: "you" },
    assistant: { color: "text-blue-400", label: "ai" },
    system: { color: "text-yellow-400", label: "system" },
    tool: { color: "text-amber-400", label: "tool" },
  };

  const config = roleConfig[message.role] || roleConfig.system;

  // For tool messages, show a more compact format
  if (message.role === "tool") {
    return (
      <div className="mb-3" data-testid={`message-${message.id}`}>
        <span className="text-slate-500">[{formatTime(message.created_at)}]</span>{" "}
        <span className={config.color}>{config.label}:</span>{" "}
        <span className="text-slate-200 whitespace-pre-wrap">
          <code className="text-xs bg-slate-800 px-1 rounded">{message.content.slice(0, 200)}{message.content.length > 200 ? "..." : ""}</code>
        </span>
      </div>
    );
  }

  // For assistant messages with tool calls
  if (message.role === "assistant" && message.tool_calls && message.tool_calls.length > 0) {
    return (
      <div className="mb-3" data-testid={`message-${message.id}`}>
        <span className="text-slate-500">[{formatTime(message.created_at)}]</span>{" "}
        <span className={config.color}>{config.label}:</span>{" "}
        {message.content && (
          <span className="text-slate-200 whitespace-pre-wrap">{message.content}</span>
        )}
        <div className="ml-4 mt-1 text-amber-400">
          {message.tool_calls.map((tc: ToolCall) => (
            <div key={tc.id} className="text-xs">
              → Calling <code className="bg-slate-800 px-1 rounded">{tc.function.name}</code>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="mb-3" data-testid={`message-${message.id}`}>
      <span className="text-slate-500">[{formatTime(message.created_at)}]</span>{" "}
      <span className={config.color}>{config.label}:</span>{" "}
      <span className="text-slate-200 whitespace-pre-wrap">{message.content}</span>
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";
  const isTool = message.role === "tool";
  const hasToolCalls = message.role === "assistant" && message.tool_calls && message.tool_calls.length > 0;

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

  // Tool response messages
  if (isTool) {
    return (
      <div className="flex justify-start" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
            <Wrench className="h-4 w-4 text-amber-400" />
          </div>
          <div className="bg-slate-800/60 border border-amber-500/20 rounded-2xl rounded-tl-md px-4 py-3 text-slate-300">
            <div className="text-xs text-amber-400 mb-1">Tool Result</div>
            <pre className="text-xs whitespace-pre-wrap overflow-x-auto max-h-40 overflow-y-auto">
              {message.content.length > 500 ? message.content.slice(0, 500) + "..." : message.content}
            </pre>
            <p className="text-xs mt-1.5 text-slate-500">{formatTime(message.created_at)}</p>
          </div>
        </div>
      </div>
    );
  }

  // Assistant message with tool calls
  if (hasToolCalls) {
    return (
      <div className="flex justify-start" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
            <Bot className="h-4 w-4 text-indigo-400" />
          </div>
          <div className="space-y-2">
            {message.content && (
              <div className="bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-200">
                <p className="whitespace-pre-wrap">{message.content}</p>
              </div>
            )}
            <div className="bg-slate-800/60 border border-amber-500/20 rounded-xl px-4 py-3">
              <div className="text-xs text-amber-400 mb-2 flex items-center gap-1.5">
                <Wrench className="h-3 w-3" />
                Using tools
              </div>
              {message.tool_calls!.map((tc: ToolCall) => (
                <div key={tc.id} className="flex items-center gap-2 text-sm text-slate-300">
                  <Play className="h-3 w-3 text-amber-400" />
                  <code className="bg-slate-700 px-1.5 py-0.5 rounded text-xs">{tc.function.name}</code>
                </div>
              ))}
              <p className="text-xs mt-2 text-slate-500">{formatTime(message.created_at)}</p>
            </div>
          </div>
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

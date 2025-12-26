import { useEffect, useRef, forwardRef } from "react";
import { Loader2, User, Bot, Wrench, CheckCircle2, XCircle, Play } from "lucide-react";
import type { Message, ToolCall } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";

interface MessageListProps {
  messages: Message[];
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls?: ActiveToolCall[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
}

export function MessageList({
  messages,
  isGenerating,
  streamingContent,
  activeToolCalls = [],
  scrollToMessageId,
  onScrollComplete,
}: MessageListProps) {
  const endRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Scroll to specific message when navigating from search
  useEffect(() => {
    if (scrollToMessageId) {
      const messageEl = messageRefs.current.get(scrollToMessageId);
      if (messageEl) {
        // Scroll into view with highlight effect
        messageEl.scrollIntoView({ behavior: "smooth", block: "center" });
        // Add highlight class temporarily
        messageEl.classList.add("ring-2", "ring-yellow-400", "ring-offset-2");
        setTimeout(() => {
          messageEl.classList.remove("ring-2", "ring-yellow-400", "ring-offset-2");
          onScrollComplete?.();
        }, 2000);
      } else {
        // Message not found, just clear the state
        onScrollComplete?.();
      }
    }
  }, [scrollToMessageId, onScrollComplete, messages]);

  // Auto-scroll to end for new messages/streaming
  useEffect(() => {
    if (!scrollToMessageId) {
      endRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, streamingContent, activeToolCalls, scrollToMessageId]);

  if (messages.length === 0 && !isGenerating) {
    return (
      <div className="flex-1 flex items-center justify-center p-8" data-testid="empty-messages">
        <div className="text-center max-w-sm">
          <div className="w-16 h-16 rounded-2xl bg-indigo-500/20 flex items-center justify-center mx-auto mb-4">
            <Bot className="h-8 w-8 text-indigo-400" />
          </div>
          <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-2">Start a conversation</h3>
          <p className="text-sm text-slate-500 dark:text-slate-400">
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

  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4" data-testid="message-list">
      {messages.map((message) => (
        <MessageBubble
          key={message.id}
          message={message}
          ref={(el) => {
            if (el) messageRefs.current.set(message.id, el);
            else messageRefs.current.delete(message.id);
          }}
        />
      ))}

      {/* Active tool calls during generation */}
      {activeToolCalls.length > 0 && (
        <div className="flex justify-start" data-testid="active-tool-calls">
          <div className="flex gap-3 max-w-[85%]">
            <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
              <Wrench className="h-4 w-4 text-amber-400" />
            </div>
            <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200 space-y-2">
              {activeToolCalls.map((tc) => (
                <div key={tc.id} className="flex items-center gap-2">
                  {tc.status === "running" ? (
                    <>
                      <Play className="h-4 w-4 text-amber-500 dark:text-amber-400 animate-pulse" />
                      <span className="text-sm">Running <code className="bg-slate-200 dark:bg-slate-700 px-1.5 py-0.5 rounded">{tc.name}</code>...</span>
                    </>
                  ) : tc.status === "completed" ? (
                    <>
                      <CheckCircle2 className="h-4 w-4 text-green-500 dark:text-green-400" />
                      <span className="text-sm text-green-600 dark:text-green-400">{tc.name} completed</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="h-4 w-4 text-red-500 dark:text-red-400" />
                      <span className="text-sm text-red-600 dark:text-red-400">{tc.name} failed</span>
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
            <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200">
              {streamingContent ? (
                <p className="whitespace-pre-wrap">{streamingContent}</p>
              ) : (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin text-indigo-500 dark:text-indigo-400" />
                  <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
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

const MessageBubble = forwardRef<HTMLDivElement, { message: Message }>(function MessageBubble(
  { message },
  ref
) {
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
      <div ref={ref} className="flex justify-center transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="bg-slate-200/50 dark:bg-slate-800/50 rounded-lg px-4 py-2 text-sm text-slate-500 dark:text-slate-400 italic max-w-[80%]">
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>
      </div>
    );
  }

  // Tool response messages
  if (isTool) {
    return (
      <div ref={ref} className="flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
            <Wrench className="h-4 w-4 text-amber-500 dark:text-amber-400" />
          </div>
          <div className="bg-amber-50 dark:bg-slate-800/60 border border-amber-200 dark:border-amber-500/20 rounded-2xl rounded-tl-md px-4 py-3 text-slate-600 dark:text-slate-300">
            <div className="text-xs text-amber-600 dark:text-amber-400 mb-1">Tool Result</div>
            <pre className="text-xs whitespace-pre-wrap overflow-x-auto max-h-40 overflow-y-auto">
              {message.content.length > 500 ? message.content.slice(0, 500) + "..." : message.content}
            </pre>
            <p className="text-xs mt-1.5 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
          </div>
        </div>
      </div>
    );
  }

  // Assistant message with tool calls
  if (hasToolCalls) {
    return (
      <div ref={ref} className="flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
            <Bot className="h-4 w-4 text-indigo-400" />
          </div>
          <div className="space-y-2">
            {message.content && (
              <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200">
                <p className="whitespace-pre-wrap">{message.content}</p>
              </div>
            )}
            <div className="bg-amber-50 dark:bg-slate-800/60 border border-amber-200 dark:border-amber-500/20 rounded-xl px-4 py-3">
              <div className="text-xs text-amber-600 dark:text-amber-400 mb-2 flex items-center gap-1.5">
                <Wrench className="h-3 w-3" />
                Using tools
              </div>
              {message.tool_calls!.map((tc: ToolCall) => (
                <div key={tc.id} className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
                  <Play className="h-3 w-3 text-amber-500 dark:text-amber-400" />
                  <code className="bg-amber-100 dark:bg-slate-700 px-1.5 py-0.5 rounded text-xs">{tc.function.name}</code>
                </div>
              ))}
              <p className="text-xs mt-2 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={ref}
      className={`flex ${isUser ? "justify-end" : "justify-start"} transition-all duration-300`}
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
              : "bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-200 rounded-tl-md"
          }`}
        >
          <p className="whitespace-pre-wrap">{message.content}</p>
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
});

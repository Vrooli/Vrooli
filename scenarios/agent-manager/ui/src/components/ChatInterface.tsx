import { useState, useRef, useEffect, useMemo } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { Send, Loader2, User, Bot, Copy, Check, AlertCircle } from "lucide-react";
import { Button } from "./ui/button";
import { Textarea } from "./ui/textarea";
import { cn, formatDate } from "../lib/utils";
import { MarkdownRenderer } from "./markdown";
import type { Run, RunEvent } from "../types";
import { RunEventType, RunStatus } from "../types";

interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
}

interface ChatInterfaceProps {
  run: Run;
  events: RunEvent[];
  eventsLoading: boolean;
  onContinue: (message: string) => Promise<void>;
  /** The initial task prompt that started this run */
  initialPrompt?: string;
}

export function ChatInterface({ run, events, eventsLoading, onContinue, initialPrompt }: ChatInterfaceProps) {
  const [inputMessage, setInputMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [copyStatus, setCopyStatus] = useState<Record<string, "idle" | "copied">>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Extract all messages from events, with initial prompt as first message
  const messages = useMemo(() => {
    const result: ChatMessage[] = [];

    // Add the initial task prompt as the first user message
    if (initialPrompt) {
      result.push({
        id: "initial-prompt",
        role: "user",
        content: initialPrompt,
        timestamp: run.createdAt ? new Date(timestampMs(run.createdAt)) : new Date(),
      });
    }

    for (const event of events) {
      if (event.data.case !== "message") continue;
      const payload = event.data.value as { role?: string; content?: string };
      const role = (payload.role || "").toLowerCase();
      if (role !== "user" && role !== "assistant") continue;

      result.push({
        id: event.id,
        role: role as "user" | "assistant",
        content: payload.content || "",
        timestamp: event.timestamp ? new Date(timestampMs(event.timestamp)) : new Date(),
      });
    }
    return result;
  }, [events, initialPrompt, run.createdAt]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length]);

  // Check if the run is actively generating
  const isGenerating = useMemo(() => {
    return run.status === RunStatus.RUNNING || run.status === RunStatus.STARTING || run.status === RunStatus.PENDING;
  }, [run.status]);

  // Check if continuation is available
  const canContinue = useMemo(() => {
    // Must have a session ID
    if (!run.sessionId) return false;
    // Must not be currently running
    if (isGenerating) {
      return false;
    }
    return true;
  }, [run.sessionId, isGenerating]);

  // Auto-scroll when generating state changes (show skeleton)
  useEffect(() => {
    if (isGenerating) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [isGenerating]);

  const handleSend = async () => {
    if (!inputMessage.trim() || !canContinue || sending) return;

    setSending(true);
    try {
      await onContinue(inputMessage.trim());
      setInputMessage("");
    } catch (err) {
      console.error("Failed to continue run:", err);
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleCopy = async (messageId: string, content: string) => {
    try {
      await navigator.clipboard.writeText(content);
      setCopyStatus((prev) => ({ ...prev, [messageId]: "copied" }));
      setTimeout(() => {
        setCopyStatus((prev) => ({ ...prev, [messageId]: "idle" }));
      }, 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  if (eventsLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading conversation...</span>
      </div>
    );
  }

  if (messages.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        No messages in this conversation
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 min-h-0 overflow-y-auto space-y-4 mb-4">
        {messages.map((message) => (
          <div
            key={message.id}
            className={cn(
              "flex gap-3",
              message.role === "user" ? "flex-row-reverse" : "flex-row"
            )}
          >
            {/* Avatar */}
            <div
              className={cn(
                "flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center",
                message.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground"
              )}
            >
              {message.role === "user" ? (
                <User className="h-4 w-4" />
              ) : (
                <Bot className="h-4 w-4" />
              )}
            </div>

            {/* Message bubble */}
            <div
              className={cn(
                "group relative max-w-[80%] rounded-lg px-4 py-2",
                message.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted"
              )}
            >
              {/* Copy button */}
              <button
                onClick={() => handleCopy(message.id, message.content)}
                className={cn(
                  "absolute top-2 opacity-0 group-hover:opacity-100 transition-opacity",
                  message.role === "user" ? "left-2" : "right-2"
                )}
                title="Copy message"
              >
                {copyStatus[message.id] === "copied" ? (
                  <Check className="h-3 w-3" />
                ) : (
                  <Copy className="h-3 w-3" />
                )}
              </button>

              {/* Content */}
              <div className="text-sm">
                <MarkdownRenderer content={message.content} />
              </div>

              {/* Timestamp */}
              <div
                className={cn(
                  "text-[10px] mt-1",
                  message.role === "user"
                    ? "text-primary-foreground/70"
                    : "text-muted-foreground"
                )}
              >
                {formatDate(message.timestamp)}
              </div>
            </div>
          </div>
        ))}

        {/* Loading skeleton when agent is generating */}
        {isGenerating && (
          <div className="flex gap-3 flex-row">
            {/* Avatar */}
            <div className="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center bg-muted text-muted-foreground">
              <Bot className="h-4 w-4" />
            </div>

            {/* Skeleton bubble */}
            <div className="max-w-[80%] rounded-lg px-4 py-3 bg-muted">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Generating response...</span>
              </div>
              {/* Animated skeleton bars */}
              <div className="mt-2 space-y-2">
                <div className="h-3 bg-muted-foreground/20 rounded animate-pulse w-[200px]" />
                <div className="h-3 bg-muted-foreground/20 rounded animate-pulse w-[160px]" />
                <div className="h-3 bg-muted-foreground/20 rounded animate-pulse w-[180px]" />
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      {canContinue ? (
        <div className="border-t border-border pt-4">
          <div className="flex gap-2">
            <Textarea
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type your follow-up message..."
              className="min-h-[60px] resize-none"
              disabled={sending}
            />
            <Button
              onClick={handleSend}
              disabled={!inputMessage.trim() || sending}
              className="self-end"
            >
              {sending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Press Enter to send, Shift+Enter for new line
          </p>
        </div>
      ) : (
        <div className="border-t border-border pt-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <AlertCircle className="h-4 w-4" />
            {!run.sessionId ? (
              <span>Session not available for this run</span>
            ) : run.status === RunStatus.RUNNING ? (
              <span>Run is in progress - wait for completion to continue</span>
            ) : (
              <span>Continuation not available</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

import { useState, useRef, useEffect, useMemo } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { Send, Loader2, User, Bot, Copy, Check, AlertCircle, Trash2, Paperclip } from "lucide-react";
import { Button } from "./ui/button";
import { Textarea } from "./ui/textarea";
import { cn } from "../lib/utils";
import { formatStandardDateTime } from "../lib/dateTime";
import { MarkdownRenderer } from "./markdown";
import { useAttachments } from "../hooks/useAttachments";
import { AttachmentPreview } from "./AttachmentPreview";
import type { Run, RunEvent } from "../types";
import { RunStatus } from "../types";
import type { MessageAttachmentInfo } from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";

interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
  attachments?: MessageAttachmentInfo[];
}

/** Strip empty <context type="image" ...></context> tags from message content */
function stripImageContextTags(content: string): string {
  return content.replace(/<context\s+[^>]*type="image"[^>]*>\s*<\/context>/g, "").trim();
}

interface ChatInterfaceProps {
  run: Run;
  events: RunEvent[];
  eventsLoading: boolean;
  onContinue: (message: string, attachmentIds?: string[]) => Promise<void>;
  onDeleteMessage: (eventId: string) => Promise<void>;
}

export function ChatInterface({
  run,
  events,
  eventsLoading,
  onContinue,
  onDeleteMessage,
}: ChatInterfaceProps) {
  const [inputMessage, setInputMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [copyStatus, setCopyStatus] = useState<Record<string, "idle" | "copied">>({});
  const [continueError, setContinueError] = useState<string | null>(null);
  const [revealedMessages, setRevealedMessages] = useState<Record<string, boolean>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { attachments, addAttachment, removeAttachment, clearAttachments, isUploading, getUploadedIds } = useAttachments();

  const deletedMessages = useMemo(() => {
    const deleted = new Set<string>();
    for (const event of events) {
      if (event.data.case !== "messageDeleted") continue;
      const payload = event.data.value as { targetEventId?: string };
      if (payload.targetEventId) {
        deleted.add(payload.targetEventId);
      }
    }
    return deleted;
  }, [events]);

  // Extract all messages from events
  const messages = useMemo(() => {
    const result: ChatMessage[] = [];

    for (const event of events) {
      if (event.data.case !== "message") continue;
      const payload = event.data.value as { role?: string; content?: string; attachments?: MessageAttachmentInfo[] };
      const role = (payload.role || "").toLowerCase();
      if (role !== "user" && role !== "assistant") continue;

      result.push({
        id: event.id,
        role: role as "user" | "assistant",
        content: stripImageContextTags(payload.content || ""),
        timestamp: event.timestamp ? new Date(timestampMs(event.timestamp)) : new Date(),
        attachments: payload.attachments,
      });
    }
    return result;
  }, [events]);

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
    return run.actions?.canContinue ?? false;
  }, [run.actions?.canContinue]);

  // Auto-scroll when generating state changes (show skeleton)
  useEffect(() => {
    if (isGenerating) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [isGenerating]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    const styles = window.getComputedStyle(textarea);
    const lineHeight = Number.parseFloat(styles.lineHeight || "20");
    const padding =
      Number.parseFloat(styles.paddingTop || "0") + Number.parseFloat(styles.paddingBottom || "0");
    const maxHeight = lineHeight * 10 + padding;
    const nextHeight = Math.min(textarea.scrollHeight, maxHeight);

    textarea.style.height = `${nextHeight}px`;
    textarea.style.overflowY = textarea.scrollHeight > maxHeight ? "auto" : "hidden";
  }, [inputMessage]);

  const handleSend = async () => {
    const hasText = inputMessage.trim().length > 0;
    const hasAttachments = attachments.length > 0;
    if ((!hasText && !hasAttachments) || !canContinue || sending || isUploading) return;

    setSending(true);
    setContinueError(null);
    try {
      const ids = getUploadedIds();
      await onContinue(inputMessage.trim(), ids.length > 0 ? ids : undefined);
      setInputMessage("");
      clearAttachments();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to continue run";
      setContinueError(message);
      console.error("Failed to continue run:", err);
    } finally {
      setSending(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      addAttachment(file);
      e.target.value = ""; // Reset so same file can be selected again
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

  const handleDelete = async (messageId: string) => {
    if (!window.confirm("Delete this message? It will be hidden but remain in the run history.")) {
      return;
    }
    try {
      await onDeleteMessage(messageId);
      setRevealedMessages((prev) => {
        if (!prev[messageId]) return prev;
        const next = { ...prev };
        delete next[messageId];
        return next;
      });
    } catch (err) {
      console.error("Failed to delete message:", err);
    }
  };

  const toggleReveal = (messageId: string) => {
    setRevealedMessages((prev) => ({ ...prev, [messageId]: !prev[messageId] }));
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
          (() => {
            const isDeleted = deletedMessages.has(message.id);
            const isRevealed = revealedMessages[message.id] ?? false;
            const showContent = !isDeleted || isRevealed;

            return (
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
                "flex flex-col gap-1",
                message.role === "user" ? "items-end" : "items-start"
              )}
            >
              <div
                className={cn(
                  "max-w-[95%] rounded-lg px-4 py-2",
                  message.role === "user"
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted"
                )}
              >
                {showContent ? (
                  <div className="text-sm">
                    {/* Render image attachments as thumbnails */}
                    {message.attachments && message.attachments.length > 0 && (
                      <div className="flex flex-wrap gap-2 mb-2">
                        {message.attachments.map((att) => (
                          <a
                            key={att.id}
                            href={att.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="block"
                          >
                            <img
                              src={att.url}
                              alt={att.fileName}
                              className="max-w-[200px] max-h-[200px] rounded object-cover border border-border"
                            />
                          </a>
                        ))}
                      </div>
                    )}
                    {message.content && <MarkdownRenderer content={message.content} />}
                  </div>
                ) : (
                  <div className="text-sm italic text-muted-foreground">
                    Message deleted
                  </div>
                )}

                {/* Timestamp */}
                <div
                  className={cn(
                    "text-[10px] mt-1",
                    message.role === "user"
                      ? "text-primary-foreground/70"
                      : "text-muted-foreground"
                  )}
                >
                  {formatStandardDateTime(message.timestamp)}
                </div>
              </div>

              <div
                className={cn(
                  "flex items-center gap-1",
                  message.role === "user" ? "justify-end" : "justify-start"
                )}
              >
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => handleCopy(message.id, message.content)}
                  disabled={!showContent}
                  title="Copy message"
                >
                  {copyStatus[message.id] === "copied" ? (
                    <Check className="h-3 w-3" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </Button>
                {!isDeleted ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive"
                    onClick={() => handleDelete(message.id)}
                    title="Delete message"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                ) : (
                  <Button
                    variant="link"
                    size="sm"
                    className="px-2"
                    onClick={() => toggleReveal(message.id)}
                  >
                    {isRevealed ? "Hide message" : "Show message"}
                  </Button>
                )}
              </div>
            </div>
          </div>
            );
          })()
        ))}

        {/* Loading skeleton when agent is generating */}
        {isGenerating && (
          <div className="flex gap-3 flex-row">
            {/* Avatar */}
            <div className="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center bg-muted text-muted-foreground">
              <Bot className="h-4 w-4" />
            </div>

            {/* Skeleton bubble */}
            <div className="max-w-[95%] rounded-lg px-4 py-3 bg-muted">
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
          {/* Attachment preview */}
          {attachments.length > 0 && (
            <AttachmentPreview
              attachments={attachments}
              onRemove={removeAttachment}
              isUploading={isUploading}
            />
          )}
          <div className="flex gap-2">
            {/* Image upload button */}
            <Button
              variant="ghost"
              size="icon"
              className="self-end"
              onClick={() => fileInputRef.current?.click()}
              title="Attach image"
            >
              <Paperclip className="h-4 w-4" />
            </Button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              onChange={handleFileSelect}
              className="hidden"
            />
            <Textarea
              ref={textareaRef}
              value={inputMessage}
              onChange={(e) => {
                setInputMessage(e.target.value);
                if (continueError) {
                  setContinueError(null);
                }
              }}
              onKeyDown={handleKeyDown}
              placeholder="Type your follow-up message..."
              className="min-h-[60px] resize-none"
              disabled={sending}
            />
            <Button
              onClick={handleSend}
              disabled={(!inputMessage.trim() && attachments.length === 0) || sending || isUploading}
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
          {continueError ? (
            <div className="mt-2 flex items-center gap-2 text-xs text-destructive">
              <AlertCircle className="h-3 w-3" />
              <span>{continueError}</span>
            </div>
          ) : null}
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

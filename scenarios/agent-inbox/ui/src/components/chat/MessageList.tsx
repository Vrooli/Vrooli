import { useEffect, useRef, useState, forwardRef, useCallback, useMemo } from "react";
import {
  Loader2, User, Bot, Wrench, CheckCircle2, XCircle, Play,
  Copy, Volume2, VolumeX, RefreshCw, Pencil, Trash2, GitBranch,
  ChevronDown, ChevronUp, ShieldAlert
} from "lucide-react";
import type { Message, ToolCall, ToolCallRecord } from "../../lib/api";
import type { ActiveToolCall, PendingApproval } from "../../hooks/useCompletion";
import type { ViewMode } from "../settings/Settings";
import { Tooltip } from "../ui/tooltip";
import { useToast } from "../ui/toast";
import { VersionPicker } from "./VersionPicker";
import { PendingApprovalCard } from "./PendingApprovalCard";
import { getSiblingInfo, getPreviousSibling, getNextSibling } from "../../lib/messageTree";

interface MessageListProps {
  messages: Message[];
  /** All messages including non-visible branches (for sibling computation) */
  allMessages?: Message[];
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls?: ActiveToolCall[];
  /** Persisted tool call records with status/result info */
  toolCallRecords?: ToolCallRecord[];
  /** Pending tool call approvals */
  pendingApprovals?: PendingApproval[];
  /** Whether we're waiting for user to approve pending tool calls */
  awaitingApprovals?: boolean;
  /** Whether an approval is being processed */
  isProcessingApproval?: boolean;
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
  viewMode?: ViewMode;
  /** Called when user requests regeneration of an assistant message */
  onRegenerateMessage?: (messageId: string) => void;
  /** Called when user selects a different branch/version */
  onSelectBranch?: (messageId: string) => void;
  /** Called when user wants to fork the conversation from a specific message */
  onForkConversation?: (messageId: string) => void;
  /** Called when user approves a pending tool call */
  onApproveTool?: (toolCallId: string) => void;
  /** Called when user rejects a pending tool call */
  onRejectTool?: (toolCallId: string, reason?: string) => void;
  /** Whether regeneration is in progress */
  isRegenerating?: boolean;
  /** Whether forking is in progress */
  isForking?: boolean;
}

export function MessageList({
  messages,
  allMessages,
  isGenerating,
  streamingContent,
  activeToolCalls = [],
  toolCallRecords = [],
  pendingApprovals = [],
  awaitingApprovals = false,
  isProcessingApproval = false,
  scrollToMessageId,
  onScrollComplete,
  viewMode = "bubble",
  onRegenerateMessage,
  onSelectBranch,
  onForkConversation,
  onApproveTool,
  onRejectTool,
  isRegenerating = false,
  isForking = false,
}: MessageListProps) {
  // Use allMessages for sibling computation, fallback to visible messages
  const messagesForSiblings = allMessages ?? messages;
  const endRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Create lookup map from tool_call_id to ToolCallRecord for persisted tool calls
  // IDs from OpenRouter are strings like "call_abc123", stored as-is in both
  // messages.tool_calls and tool_calls.id. Normalize by removing dashes for
  // backward compatibility with any legacy UUID-formatted records.
  const toolCallRecordMap = useMemo(() => {
    const map = new Map<string, ToolCallRecord>();
    for (const record of toolCallRecords) {
      // Store with both original and normalized ID for robust lookup
      map.set(record.id, record);
      const normalizedId = record.id.replace(/-/g, "");
      if (normalizedId !== record.id) {
        map.set(normalizedId, record);
      }
    }
    return map;
  }, [toolCallRecords]);

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

  const isCompact = viewMode === "compact";

  return (
    <div className={`flex-1 overflow-y-auto p-4 ${isCompact ? "space-y-2" : "space-y-4"}`} data-testid="message-list">
      {messages.map((message) => (
        <MessageBubble
          key={message.id}
          message={message}
          viewMode={viewMode}
          allMessages={messagesForSiblings}
          toolCallRecordMap={toolCallRecordMap}
          onRegenerate={onRegenerateMessage}
          onSelectBranch={onSelectBranch}
          onFork={onForkConversation}
          isRegenerating={isRegenerating}
          isForking={isForking}
          ref={(el) => {
            if (el) messageRefs.current.set(message.id, el);
            else messageRefs.current.delete(message.id);
          }}
        />
      ))}

      {/* Active tool calls during generation */}
      {activeToolCalls.length > 0 && (
        isCompact ? (
          <div className="border-l-2 border-l-amber-500 pl-3 py-1" data-testid="active-tool-calls">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs font-medium text-amber-400">Tool</span>
            </div>
            <div className="space-y-1">
              {activeToolCalls.map((tc) => (
                <div key={tc.id} className="flex items-center gap-2 text-sm">
                  {tc.status === "running" ? (
                    <>
                      <Play className="h-3 w-3 text-amber-500 dark:text-amber-400 animate-pulse" />
                      <span className="text-slate-700 dark:text-slate-300">Running <code className="bg-slate-200 dark:bg-slate-700 px-1 py-0.5 rounded text-xs">{tc.name}</code>...</span>
                    </>
                  ) : tc.status === "completed" ? (
                    <>
                      <CheckCircle2 className="h-3 w-3 text-green-500 dark:text-green-400" />
                      <span className="text-green-600 dark:text-green-400">{tc.name} completed</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="h-3 w-3 text-red-500 dark:text-red-400" />
                      <span className="text-red-600 dark:text-red-400">{tc.name} failed</span>
                    </>
                  )}
                </div>
              ))}
            </div>
          </div>
        ) : (
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
        )
      )}

      {/* Pending Approvals */}
      {pendingApprovals.length > 0 && onApproveTool && onRejectTool && (
        <div className="space-y-2" data-testid="pending-approvals">
          {awaitingApprovals && (
            <div className="flex items-center gap-2 py-2">
              <ShieldAlert className="h-4 w-4 text-yellow-400" />
              <span className="text-sm text-yellow-400">Awaiting approval to continue</span>
            </div>
          )}
          {pendingApprovals.map((approval) => (
            <PendingApprovalCard
              key={approval.id}
              approval={approval}
              onApprove={onApproveTool}
              onReject={onRejectTool}
              isProcessing={isProcessingApproval}
            />
          ))}
        </div>
      )}

      {isGenerating && !activeToolCalls.length && (
        isCompact ? (
          <div className="border-l-2 border-l-emerald-500 pl-3 py-1" data-testid="streaming-message">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs font-medium text-emerald-400">Assistant</span>
              {!streamingContent && (
                <Loader2 className="h-3 w-3 animate-spin text-emerald-400" />
              )}
            </div>
            {streamingContent ? (
              <div className="text-sm text-slate-700 dark:text-slate-200 whitespace-pre-wrap">{streamingContent}</div>
            ) : (
              <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
            )}
          </div>
        ) : (
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
        )
      )}
      <div ref={endRef} />
    </div>
  );
}

// Action button component for consistent styling
interface ActionButtonProps {
  icon: React.ReactNode;
  tooltip: string;
  onClick: () => void;
  isActive?: boolean;
  className?: string;
}

function ActionButton({ icon, tooltip, onClick, isActive, className }: ActionButtonProps) {
  return (
    <Tooltip content={tooltip} side="top">
      <button
        onClick={onClick}
        className={`p-1.5 rounded-md transition-colors ${
          isActive
            ? "bg-indigo-500/20 text-indigo-400"
            : "hover:bg-white/10 text-slate-400 hover:text-slate-200"
        } ${className || ""}`}
        aria-label={tooltip}
      >
        {icon}
      </button>
    </Tooltip>
  );
}

interface MessageBubbleProps {
  message: Message;
  viewMode: ViewMode;
  /** All messages for computing siblings */
  allMessages: Message[];
  /** Map of tool_call_id to ToolCallRecord for status lookup */
  toolCallRecordMap: Map<string, ToolCallRecord>;
  /** Called when user requests regeneration */
  onRegenerate?: (messageId: string) => void;
  /** Called when user selects a different branch */
  onSelectBranch?: (messageId: string) => void;
  /** Called when user wants to fork the conversation from this message */
  onFork?: (messageId: string) => void;
  /** Whether regeneration is in progress */
  isRegenerating?: boolean;
  /** Whether forking is in progress */
  isForking?: boolean;
}

const MessageBubble = forwardRef<HTMLDivElement, MessageBubbleProps>(function MessageBubble(
  { message, viewMode, allMessages, toolCallRecordMap, onRegenerate, onSelectBranch, onFork, isRegenerating = false, isForking = false },
  ref
) {
  const { addToast } = useToast();
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";
  const isSystem = message.role === "system";
  const isTool = message.role === "tool";
  const hasToolCalls = message.role === "assistant" && message.tool_calls && message.tool_calls.length > 0;
  const isCompact = viewMode === "compact";

  // Compute sibling info for version picker
  const siblingInfo = useMemo(() => {
    if (!isAssistant || !allMessages || allMessages.length === 0) {
      return { current: 1, total: 1, siblings: [] };
    }
    return getSiblingInfo(allMessages, message.id);
  }, [isAssistant, allMessages, message.id]);

  const hasSiblings = siblingInfo.total > 1;

  // State for tool result expansion
  const [isExpanded, setIsExpanded] = useState(false);
  // State for text-to-speech
  const [isSpeaking, setIsSpeaking] = useState(false);

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  // Copy message content to clipboard
  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content);
      addToast("Copied to clipboard", "success");
    } catch {
      addToast("Failed to copy", "error");
    }
  }, [message.content, addToast]);

  // Text-to-speech for assistant messages
  const handleReadAloud = useCallback(() => {
    if (isSpeaking) {
      window.speechSynthesis.cancel();
      setIsSpeaking(false);
      return;
    }

    if (!window.speechSynthesis) {
      addToast("Text-to-speech not supported in this browser", "error");
      return;
    }

    const utterance = new SpeechSynthesisUtterance(message.content);
    utterance.onend = () => setIsSpeaking(false);
    utterance.onerror = () => {
      setIsSpeaking(false);
      addToast("Speech synthesis failed", "error");
    };

    window.speechSynthesis.speak(utterance);
    setIsSpeaking(true);
  }, [message.content, isSpeaking, addToast]);

  // Cancel speech when component unmounts
  useEffect(() => {
    return () => {
      if (isSpeaking) {
        window.speechSynthesis.cancel();
      }
    };
  }, [isSpeaking]);

  // Placeholder actions that show "coming soon" toast
  const handleComingSoon = useCallback((feature: string) => {
    addToast(`${feature} coming soon`, "info");
  }, [addToast]);

  // Regenerate handler - calls the actual regeneration function
  const handleRegenerate = useCallback(() => {
    if (onRegenerate && isAssistant) {
      onRegenerate(message.id);
    } else {
      handleComingSoon("Regenerate");
    }
  }, [onRegenerate, isAssistant, message.id, handleComingSoon]);

  const handleEdit = useCallback(() => handleComingSoon("Edit message"), [handleComingSoon]);
  const handleDelete = useCallback(() => handleComingSoon("Delete message"), [handleComingSoon]);

  // Fork conversation handler - calls the actual fork function
  const handleFork = useCallback(() => {
    if (onFork) {
      onFork(message.id);
    } else {
      handleComingSoon("Fork conversation");
    }
  }, [onFork, message.id, handleComingSoon]);

  // Version picker navigation handlers
  const handlePreviousVersion = useCallback(() => {
    console.log("[VersionPicker] Previous clicked", {
      messageId: message.id,
      hasOnSelectBranch: !!onSelectBranch,
      allMessagesCount: allMessages.length,
      siblingInfo,
    });
    if (!onSelectBranch) {
      console.log("[VersionPicker] No onSelectBranch handler!");
      return;
    }
    const prevSibling = getPreviousSibling(allMessages, message.id);
    console.log("[VersionPicker] Previous sibling:", prevSibling);
    if (prevSibling) {
      console.log("[VersionPicker] Selecting branch:", prevSibling.id);
      onSelectBranch(prevSibling.id);
    }
  }, [onSelectBranch, allMessages, message.id, siblingInfo]);

  const handleNextVersion = useCallback(() => {
    console.log("[VersionPicker] Next clicked", {
      messageId: message.id,
      hasOnSelectBranch: !!onSelectBranch,
    });
    if (!onSelectBranch) return;
    const nextSibling = getNextSibling(allMessages, message.id);
    console.log("[VersionPicker] Next sibling:", nextSibling);
    if (nextSibling) {
      onSelectBranch(nextSibling.id);
    }
  }, [onSelectBranch, allMessages, message.id]);

  // Render action buttons for a message
  const renderActions = (position: "user" | "assistant" | "tool") => {
    const iconSize = "h-3.5 w-3.5";

    if (position === "user") {
      return (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <ActionButton icon={<Copy className={iconSize} />} tooltip="Copy" onClick={handleCopy} />
          <ActionButton icon={<Pencil className={iconSize} />} tooltip="Edit" onClick={handleEdit} />
          <ActionButton icon={<Trash2 className={iconSize} />} tooltip="Delete" onClick={handleDelete} />
          <ActionButton
            icon={isForking ? <Loader2 className={`${iconSize} animate-spin`} /> : <GitBranch className={iconSize} />}
            tooltip={isForking ? "Forking..." : "Fork from here"}
            onClick={handleFork}
            className={isForking ? "cursor-not-allowed" : ""}
          />
        </div>
      );
    }

    if (position === "assistant") {
      return (
        <div className="flex items-center gap-0.5">
          {/* Version picker - always visible when there are siblings */}
          {hasSiblings && (
            <VersionPicker
              current={siblingInfo.current}
              total={siblingInfo.total}
              onPrevious={handlePreviousVersion}
              onNext={handleNextVersion}
              disabled={isRegenerating}
              className="mr-1"
            />
          )}
          {/* Action buttons - visible on hover */}
          <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <ActionButton icon={<Copy className={iconSize} />} tooltip="Copy" onClick={handleCopy} />
            <ActionButton
              icon={isSpeaking ? <VolumeX className={iconSize} /> : <Volume2 className={iconSize} />}
              tooltip={isSpeaking ? "Stop reading" : "Read aloud"}
              onClick={handleReadAloud}
              isActive={isSpeaking}
            />
            <ActionButton
              icon={isRegenerating ? <Loader2 className={`${iconSize} animate-spin`} /> : <RefreshCw className={iconSize} />}
              tooltip={isRegenerating ? "Regenerating..." : "Regenerate"}
              onClick={handleRegenerate}
              className={isRegenerating ? "cursor-not-allowed" : ""}
            />
            <ActionButton
              icon={isForking ? <Loader2 className={`${iconSize} animate-spin`} /> : <GitBranch className={iconSize} />}
              tooltip={isForking ? "Forking..." : "Fork from here"}
              onClick={handleFork}
              className={isForking ? "cursor-not-allowed" : ""}
            />
          </div>
        </div>
      );
    }

    // Tool messages
    return (
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
        <ActionButton icon={<Copy className={iconSize} />} tooltip="Copy" onClick={handleCopy} />
        <ActionButton
          icon={isExpanded ? <ChevronUp className={iconSize} /> : <ChevronDown className={iconSize} />}
          tooltip={isExpanded ? "Collapse" : "Expand"}
          onClick={() => setIsExpanded(!isExpanded)}
          isActive={isExpanded}
        />
      </div>
    );
  };

  // Get tool content - truncated or full based on expansion
  const getToolContent = () => {
    if (isExpanded || message.content.length <= 500) {
      return message.content;
    }
    return message.content.slice(0, 500) + "...";
  };

  // System messages - same in both modes
  if (isSystem) {
    return (
      <div ref={ref} className="flex justify-center transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className={`bg-slate-200/50 dark:bg-slate-800/50 rounded-lg px-4 py-2 text-sm text-slate-500 dark:text-slate-400 italic ${isCompact ? "w-full text-left" : "max-w-[80%]"}`}>
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>
      </div>
    );
  }

  // Compact mode rendering
  if (isCompact) {
    const roleLabel = isUser ? "You" : isTool ? "Tool" : "Assistant";
    const roleColor = isUser ? "text-indigo-400" : isTool ? "text-amber-400" : "text-emerald-400";
    const borderColor = isUser ? "border-l-indigo-500" : isTool ? "border-l-amber-500" : "border-l-emerald-500";

    return (
      <div
        ref={ref}
        className={`group transition-all duration-300 border-l-2 ${borderColor} pl-3 py-1`}
        data-testid={`message-${message.id}`}
      >
        <div className="flex items-center gap-2 mb-1">
          <span className={`text-xs font-medium ${roleColor}`}>{roleLabel}</span>
          <span className="text-xs text-slate-500 dark:text-slate-600">{formatTime(message.created_at)}</span>
          <div className="flex-1" />
          {renderActions(isUser ? "user" : isTool ? "tool" : "assistant")}
        </div>
        {isTool ? (
          <pre className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap overflow-x-auto">
            {getToolContent()}
          </pre>
        ) : (
          <div className="text-sm text-slate-700 dark:text-slate-200 whitespace-pre-wrap">{message.content}</div>
        )}
        {hasToolCalls && (
          <div className="mt-2 pl-2 border-l border-amber-500/30">
            <div className="text-xs text-amber-500 dark:text-amber-400 mb-1 flex items-center gap-1">
              <Wrench className="h-3 w-3" />
              Using tools
            </div>
            {message.tool_calls!.map((tc: ToolCall) => {
              const record = toolCallRecordMap.get(tc.id);
              const status = record?.status || "pending";
              const errorMessage = record?.error_message;
              const isFailed = status === "failed" || status === "rejected" || status === "cancelled";
              return (
                <div key={tc.id} className="text-xs text-slate-600 dark:text-slate-400">
                  <div className="flex items-center gap-2">
                    {status === "completed" ? (
                      <CheckCircle2 className="h-3 w-3 text-green-500 dark:text-green-400" />
                    ) : isFailed ? (
                      <XCircle className="h-3 w-3 text-red-500 dark:text-red-400" />
                    ) : status === "running" ? (
                      <Loader2 className="h-3 w-3 text-amber-500 dark:text-amber-400 animate-spin" />
                    ) : status === "pending_approval" ? (
                      <ShieldAlert className="h-3 w-3 text-yellow-500 dark:text-yellow-400" />
                    ) : (
                      <Play className="h-3 w-3 text-amber-500 dark:text-amber-400" />
                    )}
                    <code className="bg-slate-200 dark:bg-slate-700 px-1 py-0.5 rounded">{tc.function.name}</code>
                    {status === "completed" && <span className="text-green-600 dark:text-green-400">completed</span>}
                    {isFailed && <span className="text-red-600 dark:text-red-400">{status}</span>}
                  </div>
                  {isFailed && errorMessage && (
                    <div className="ml-5 mt-1 text-red-500 dark:text-red-400 bg-red-500/10 rounded px-2 py-1 break-words">
                      {errorMessage}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    );
  }

  // Bubble mode: Tool response messages
  if (isTool) {
    return (
      <div ref={ref} className="group flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
            <Wrench className="h-4 w-4 text-amber-500 dark:text-amber-400" />
          </div>
          <div className="bg-amber-50 dark:bg-slate-800/60 border border-amber-200 dark:border-amber-500/20 rounded-2xl rounded-tl-md px-4 py-3 text-slate-600 dark:text-slate-300">
            <div className="flex items-center justify-between gap-2 mb-1">
              <span className="text-xs text-amber-600 dark:text-amber-400">Tool Result</span>
              {renderActions("tool")}
            </div>
            <pre className="text-xs whitespace-pre-wrap overflow-x-auto max-h-40 overflow-y-auto">
              {getToolContent()}
            </pre>
            <p className="text-xs mt-1.5 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
          </div>
        </div>
      </div>
    );
  }

  // Bubble mode: Assistant message with tool calls
  if (hasToolCalls) {
    return (
      <div ref={ref} className="group flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
            <Bot className="h-4 w-4 text-indigo-400" />
          </div>
          <div className="space-y-2">
            {message.content && (
              <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200">
                <div className="flex items-center justify-end gap-2 mb-1">
                  {renderActions("assistant")}
                </div>
                <p className="whitespace-pre-wrap">{message.content}</p>
              </div>
            )}
            <div className="bg-amber-50 dark:bg-slate-800/60 border border-amber-200 dark:border-amber-500/20 rounded-xl px-4 py-3">
              <div className="text-xs text-amber-600 dark:text-amber-400 mb-2 flex items-center gap-1.5">
                <Wrench className="h-3 w-3" />
                Using tools
              </div>
              {message.tool_calls!.map((tc: ToolCall) => {
                const record = toolCallRecordMap.get(tc.id);
                const status = record?.status || "pending";
                const errorMessage = record?.error_message;
                const isFailed = status === "failed" || status === "rejected" || status === "cancelled";
                return (
                  <div key={tc.id} className="text-sm text-slate-600 dark:text-slate-300">
                    <div className="flex items-center gap-2">
                      {status === "completed" ? (
                        <CheckCircle2 className="h-3 w-3 text-green-500 dark:text-green-400" />
                      ) : isFailed ? (
                        <XCircle className="h-3 w-3 text-red-500 dark:text-red-400" />
                      ) : status === "running" ? (
                        <Loader2 className="h-3 w-3 text-amber-500 dark:text-amber-400 animate-spin" />
                      ) : status === "pending_approval" ? (
                        <ShieldAlert className="h-3 w-3 text-yellow-500 dark:text-yellow-400" />
                      ) : (
                        <Play className="h-3 w-3 text-amber-500 dark:text-amber-400" />
                      )}
                      <code className="bg-amber-100 dark:bg-slate-700 px-1.5 py-0.5 rounded text-xs">{tc.function.name}</code>
                      {status === "completed" && <span className="text-xs text-green-600 dark:text-green-400">completed</span>}
                      {isFailed && <span className="text-xs text-red-600 dark:text-red-400">{status}</span>}
                    </div>
                    {isFailed && errorMessage && (
                      <div className="ml-5 mt-1 text-xs text-red-500 dark:text-red-400 bg-red-500/10 rounded px-2 py-1 break-words">
                        {errorMessage}
                      </div>
                    )}
                  </div>
                );
              })}
              <p className="text-xs mt-2 text-slate-400 dark:text-slate-500">{formatTime(message.created_at)}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Bubble mode: Regular user/assistant messages
  return (
    <div
      ref={ref}
      className={`group flex ${isUser ? "justify-end" : "justify-start"} transition-all duration-300`}
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
          {/* Action buttons row */}
          <div className={`flex items-center ${isUser ? "justify-start" : "justify-end"} gap-2 mb-1`}>
            {renderActions(isUser ? "user" : "assistant")}
          </div>
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

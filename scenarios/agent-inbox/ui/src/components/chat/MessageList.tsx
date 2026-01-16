import { memo, useEffect, useRef, useState, forwardRef, useCallback, useMemo } from "react";
import {
  Loader2, User, Bot, Wrench, CheckCircle2, XCircle, Play,
  Copy, Volume2, VolumeX, RefreshCw, Pencil, Trash2, GitBranch,
  ChevronDown, ChevronUp, ShieldAlert
} from "lucide-react";
import { resolveAttachmentUrl, type Attachment, type Message, type ToolCall, type ToolCallRecord } from "../../lib/api";
import type { ActiveToolCall, PendingApproval } from "../../hooks/useCompletion";
import type { ViewMode } from "../settings/Settings";
import { Tooltip } from "../ui/tooltip";
import { useToast } from "../ui/toast";
import { VersionPicker } from "./VersionPicker";
import { PendingApprovalCard } from "./PendingApprovalCard";
import { getSiblingInfo, getPreviousSibling, getNextSibling } from "../../lib/messageTree";
import { MarkdownRenderer, CodeBlock } from "../markdown";

// Stable empty arrays for default prop values
// CRITICAL: Using `= []` in destructuring creates a NEW array on every render,
// which changes references and triggers infinite re-render loops via useMemo dependencies
const EMPTY_IMAGES: string[] = [];
const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
const EMPTY_TOOL_RECORDS: ToolCallRecord[] = [];
const EMPTY_APPROVALS: PendingApproval[] = [];
const EMPTY_SIBLINGS: Message[] = [];

// Stable default for sibling info to prevent creating new objects in useMemo
// CRITICAL: Returning { siblings: [] } creates a NEW array each time, which changes
// references and triggers useCallback dependencies like handlePreviousVersion to recreate
const DEFAULT_SIBLING_INFO = { current: 1, total: 1, siblings: EMPTY_SIBLINGS };

// Stable empty map for sibling info when there are no messages
// CRITICAL: Using `new Map()` inside a component creates new reference each render
const EMPTY_SIBLING_MAP: Map<string, { current: number; total: number; siblings: Message[] }> = new Map();

interface MessageListProps {
  messages: Message[];
  /** All messages including non-visible branches (for sibling computation) */
  allMessages?: Message[];
  isGenerating: boolean;
  streamingContent: string;
  /** AI-generated images during streaming (before they're saved as attachments) */
  generatedImages?: string[];
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
  /** Called when user wants to edit a user message */
  onEditMessage?: (message: Message) => void;
  /** Called when user approves a pending tool call */
  onApproveTool?: (toolCallId: string) => void;
  /** Called when user rejects a pending tool call */
  onRejectTool?: (toolCallId: string, reason?: string) => void;
  /** Whether regeneration is in progress */
  isRegenerating?: boolean;
  /** Whether forking is in progress */
  isForking?: boolean;
}

// DEBUG: Track renders globally to detect loops
let messageListRenderCount = 0;
// DEBUG: Track hook execution order to detect conditional hook issues
let currentRenderHookIndex = 0;
// DEBUG: Track history of render hook counts to detect mismatch
const renderHookHistory: Array<{ renderNum: number; hookCount: number; messagesLength: number; tookEarlyReturn: boolean; completed: boolean }> = [];
// DEBUG: Track if a render is in progress (to detect interruptions)
let renderInProgress: { renderNum: number; globalSeq: number; startTime: number } | null = null;
// DEBUG: Track the last messages reference to detect changes
let lastMessagesRef: unknown = null;

// Inner implementation of MessageList - will be wrapped with memo
function MessageListInner({
  messages,
  allMessages,
  isGenerating,
  streamingContent,
  generatedImages = EMPTY_IMAGES,
  activeToolCalls = EMPTY_TOOL_CALLS,
  toolCallRecords = EMPTY_TOOL_RECORDS,
  pendingApprovals = EMPTY_APPROVALS,
  awaitingApprovals = false,
  isProcessingApproval = false,
  scrollToMessageId,
  onScrollComplete,
  viewMode = "bubble",
  onRegenerateMessage,
  onSelectBranch,
  onForkConversation,
  onEditMessage,
  onApproveTool,
  onRejectTool,
  isRegenerating = false,
  isForking = false,
}: MessageListProps) {
  // DEBUG: Track renders and reset hook counter
  messageListRenderCount++;
  currentRenderHookIndex = 0;
  const globalSeq = typeof window !== 'undefined' && window.__getNextRenderSeq__ ? window.__getNextRenderSeq__() : 0;

  // DEBUG: Check if previous render was interrupted
  if (renderInProgress) {
    console.log(`[MessageList] !!! PREVIOUS RENDER #${renderInProgress.renderNum} (globalSeq: ${renderInProgress.globalSeq}) WAS INTERRUPTED !!! It never completed.`);
  }

  // Mark this render as in progress
  renderInProgress = { renderNum: messageListRenderCount, globalSeq, startTime: Date.now() };

  // DEBUG: Check if messages reference changed
  const messagesRefChanged = messages !== lastMessagesRef;
  const previousMessagesLength = Array.isArray(lastMessagesRef) ? (lastMessagesRef as unknown[]).length : 'N/A';
  lastMessagesRef = messages;

  // DEBUG: Log render start with critical state that could affect hook paths
  console.log(`[MessageList] ===== RENDER #${messageListRenderCount} (global seq: ${globalSeq}) START =====`, {
    messagesLength: messages?.length,
    messagesIsArray: Array.isArray(messages),
    messagesUndefined: messages === undefined,
    messagesNull: messages === null,
    allMessagesLength: allMessages?.length,
    toolCallRecordsLength: toolCallRecords?.length,
    isGenerating,
    // Check early return condition RIGHT AT START
    wouldEarlyReturn: messages?.length === 0 && !isGenerating,
    // Track reference changes
    messagesRefChanged,
    previousMessagesLength,
  });

  // Use allMessages for sibling computation, fallback to visible messages
  const messagesForSiblings = allMessages ?? messages;

  // DEBUG: Log hook #1
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useRef(endRef)`);
  const endRef = useRef<HTMLDivElement>(null);

  // DEBUG: Log hook #2
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useRef(messageRefs)`);
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Ref for debouncing auto-scroll to prevent rapid scroll triggers during streaming
  // DEBUG: Log hook #3
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useRef(scrollTimeoutRef)`);
  const scrollTimeoutRef = useRef<number | null>(null);

  // Clear stale message refs when messages change to prevent memory leaks
  // and stale data issues when switching between chats
  // DEBUG: Log hook #4
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useEffect(clearStaleRefs)`);
  useEffect(() => {
    const currentMessageIds = new Set(messages.map((m) => m.id));
    // Remove refs for messages that are no longer in the list
    for (const id of messageRefs.current.keys()) {
      if (!currentMessageIds.has(id)) {
        messageRefs.current.delete(id);
      }
    }
  }, [messages]);

  // Create lookup map from tool_call_id to ToolCallRecord for persisted tool calls
  // IDs from OpenRouter are strings like "call_abc123", stored as-is in both
  // messages.tool_calls and tool_calls.id. Normalize by removing dashes for
  // backward compatibility with any legacy UUID-formatted records.
  // DEBUG: Log hook #5
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useMemo(toolCallRecordMap) - BEFORE`);
  const toolCallRecordMap = useMemo(() => {
    console.log("[MessageList] INSIDE toolCallRecordMap useMemo, toolCallRecords:", toolCallRecords.length);
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
  console.log("[MessageList] AFTER toolCallRecordMap useMemo (hook #5 complete)");

  // OPTIMIZATION: Precompute siblingInfo for all assistant messages at the list level.
  // Previously, each MessageBubble computed getSiblingInfo(allMessages, message.id),
  // which built a message map and did filtering N times for N messages.
  // By computing once here and passing down, we reduce O(N²) to O(N) complexity.
  // CRITICAL: Return stable EMPTY_SIBLING_MAP when no messages to prevent new reference.
  // DEBUG: Log hook #6
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useMemo(siblingInfoMap) - BEFORE`);
  const siblingInfoMap = useMemo(() => {
    console.log("[MessageList] INSIDE siblingInfoMap useMemo, messagesForSiblings:", messagesForSiblings?.length, "messages:", messages.length);
    if (!messagesForSiblings || messagesForSiblings.length === 0) {
      return EMPTY_SIBLING_MAP;
    }
    const map = new Map<string, { current: number; total: number; siblings: Message[] }>();
    // Only compute for assistant messages (they're the only ones with version pickers)
    for (const message of messages) {
      if (message.role === "assistant") {
        map.set(message.id, getSiblingInfo(messagesForSiblings, message.id));
      }
    }
    return map;
  }, [messagesForSiblings, messages]);
  console.log("[MessageList] AFTER siblingInfoMap useMemo (hook #6 complete)");

  // Scroll to specific message when navigating from search
  // DEBUG: Log hook #7
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useEffect(scrollToMessage)`);
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
  // DEBOUNCED: Use timeout to prevent rapid scroll triggers during streaming which can
  // interact with browser reflow and cause React reconciliation issues.
  // Uses stable primitive dependencies instead of array references to prevent
  // unnecessary effect runs.
  // DEBUG: Log hook #8
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useEffect(autoScroll)`);
  useEffect(() => {
    if (!scrollToMessageId) {
      // Clear any pending scroll
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
      // Debounce the scroll with a small delay
      scrollTimeoutRef.current = window.setTimeout(() => {
        endRef.current?.scrollIntoView({ behavior: "smooth" });
      }, 50);
    }
    return () => {
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, [messages.length, Boolean(streamingContent), activeToolCalls.length, scrollToMessageId]);

  const isCompact = viewMode === "compact";

  // Filter out tool messages whose results are already displayed inline in the
  // preceding assistant message's ToolCallItem. Only show tool messages as a
  // fallback when there's no ToolCallRecord with a result.
  // DEBUG: Log hook #9
  currentRenderHookIndex++;
  console.log(`[MessageList] Hook #${currentRenderHookIndex}: useMemo(filteredMessages) - ABOUT TO CALL useMemo, render #${messageListRenderCount}, globalSeq: ${globalSeq}`);

  const filteredMessages = useMemo(() => {
    console.log("[MessageList] INSIDE filteredMessages useMemo callback executing, messages:", messages?.length, "isArray:", Array.isArray(messages));
    return messages.filter((message) => {
      // Only filter tool messages
      if (message.role !== "tool" || !message.tool_call_id) {
        return true;
      }
      // Check if we have a record with a result for this tool call
      const record = toolCallRecordMap.get(message.tool_call_id);
      // Hide if record exists with a result (it's already shown inline)
      if (record && record.status === "completed" && record.result) {
        return false;
      }
      // Show as fallback when no record or no result
      return true;
    });
  }, [messages, toolCallRecordMap]);

  // Record this render in history
  renderHookHistory.push({ renderNum: messageListRenderCount, hookCount: currentRenderHookIndex, messagesLength: messages?.length ?? -1, tookEarlyReturn: false, completed: true });
  renderInProgress = null; // Mark render as completed
  console.log(`[MessageList] ===== RENDER #${messageListRenderCount} COMPLETE ===== Total hooks: ${currentRenderHookIndex}. History now:`, renderHookHistory.slice(-5));

  // IMPORTANT: Early return MUST be AFTER all hooks to satisfy React's Rules of Hooks.
  // Previously this was between hooks #8 and #9, causing Error #310 when messages
  // changed from 0 to 1 (hook count changed from 8 to 9).
  if (messages.length === 0 && !isGenerating) {
    console.log(`[MessageList] Empty state - returning empty UI, render #${messageListRenderCount}`);
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

  return (
    <div className={`flex-1 overflow-y-auto p-4 ${isCompact ? "space-y-2" : "space-y-4"}`} data-testid="message-list">
      {filteredMessages.map((message) => (
        <MessageBubble
          key={message.id}
          message={message}
          viewMode={viewMode}
          allMessages={messagesForSiblings}
          siblingInfo={siblingInfoMap.get(message.id) ?? DEFAULT_SIBLING_INFO}
          toolCallRecordMap={toolCallRecordMap}
          onRegenerate={onRegenerateMessage}
          onSelectBranch={onSelectBranch}
          onFork={onForkConversation}
          onEdit={onEditMessage}
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
              {!streamingContent && generatedImages.length === 0 && (
                <Loader2 className="h-3 w-3 animate-spin text-emerald-400" />
              )}
            </div>
            {/* Show generated images during streaming */}
            {generatedImages.length > 0 && (
              <div className="flex flex-wrap gap-2 mb-2">
                {generatedImages.map((imgUrl, idx) => (
                  <img
                    key={idx}
                    src={resolveAttachmentUrl(imgUrl)}
                    alt={`Generated image ${idx + 1}`}
                    className="max-w-[150px] max-h-[150px] rounded-lg object-contain border border-slate-200 dark:border-slate-700"
                  />
                ))}
              </div>
            )}
            {streamingContent ? (
              <div className="text-sm text-slate-700 dark:text-slate-200">
                <MarkdownRenderer content={streamingContent} isStreaming />
              </div>
            ) : generatedImages.length === 0 ? (
              <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
            ) : null}
          </div>
        ) : (
          <div className="flex justify-start" data-testid="streaming-message">
            <div className="flex gap-3 max-w-[85%]">
              <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
                <Bot className="h-4 w-4 text-indigo-400" />
              </div>
              <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200">
                {/* Show generated images during streaming */}
                {generatedImages.length > 0 && (
                  <div className="flex flex-wrap gap-2 mb-3">
                    {generatedImages.map((imgUrl, idx) => (
                      <img
                        key={idx}
                        src={resolveAttachmentUrl(imgUrl)}
                        alt={`Generated image ${idx + 1}`}
                        className="max-w-[300px] max-h-[300px] rounded-lg object-contain border border-slate-200 dark:border-slate-700"
                      />
                    ))}
                  </div>
                )}
                {streamingContent ? (
                  <MarkdownRenderer content={streamingContent} isStreaming />
                ) : generatedImages.length === 0 ? (
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin text-indigo-500 dark:text-indigo-400" />
                    <span className="text-sm text-slate-500 dark:text-slate-400">Thinking...</span>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        )
      )}
      <div ref={endRef} />
    </div>
  );
}

// Export MessageList with simple memo wrapper (no custom comparison)
// The parent component (ChatView) is responsible for passing stable props
export const MessageList = memo(MessageListInner);

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

// Component for displaying a single tool call with expandable result
interface ToolCallItemProps {
  toolCall: ToolCall;
  record?: ToolCallRecord;
  variant: "compact" | "bubble";
}

function ToolCallItem({ toolCall, record, variant }: ToolCallItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const status = record?.status || "pending";
  const errorMessage = record?.error_message;
  const result = record?.result;
  const isFailed = status === "failed" || status === "rejected" || status === "cancelled";
  const hasResult = status === "completed" && result;

  const isCompact = variant === "compact";
  const textSize = isCompact ? "text-xs" : "text-sm";
  const iconSize = isCompact ? "h-3 w-3" : "h-3 w-3";
  const codeClass = isCompact
    ? "bg-slate-200 dark:bg-slate-700 px-1 py-0.5 rounded"
    : "bg-amber-100 dark:bg-slate-700 px-1.5 py-0.5 rounded text-xs";

  // Try to format result as JSON if possible
  // Note: result might be a string (from DB) or an object (from SSE streaming)
  const formatResult = (resultData: unknown): string => {
    if (typeof resultData === "string") {
      // If it's a string, try to parse and re-stringify for formatting
      try {
        const parsed = JSON.parse(resultData);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return resultData;
      }
    } else if (resultData && typeof resultData === "object") {
      // If it's already an object, just stringify it
      return JSON.stringify(resultData, null, 2);
    }
    return String(resultData ?? "");
  };

  return (
    <div className={`${textSize} text-slate-600 dark:text-slate-400`}>
      <div className="flex items-center gap-2">
        {status === "completed" ? (
          <CheckCircle2 className={`${iconSize} text-green-500 dark:text-green-400`} />
        ) : isFailed ? (
          <XCircle className={`${iconSize} text-red-500 dark:text-red-400`} />
        ) : status === "running" ? (
          <Loader2 className={`${iconSize} text-amber-500 dark:text-amber-400 animate-spin`} />
        ) : status === "pending_approval" ? (
          <ShieldAlert className={`${iconSize} text-yellow-500 dark:text-yellow-400`} />
        ) : (
          <Play className={`${iconSize} text-amber-500 dark:text-amber-400`} />
        )}
        <code className={codeClass}>{toolCall.function.name}</code>
        {status === "completed" && (
          <span className={`${isCompact ? "" : "text-xs"} text-green-600 dark:text-green-400`}>completed</span>
        )}
        {isFailed && (
          <span className={`${isCompact ? "" : "text-xs"} text-red-600 dark:text-red-400`}>{status}</span>
        )}
        {hasResult && (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="ml-auto flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
          >
            {isExpanded ? (
              <>
                <ChevronUp className="h-3 w-3" />
                <span>Hide</span>
              </>
            ) : (
              <>
                <ChevronDown className="h-3 w-3" />
                <span>Show result</span>
              </>
            )}
          </button>
        )}
      </div>
      {isFailed && errorMessage && (
        <div className="ml-5 mt-1 text-xs text-red-500 dark:text-red-400 bg-red-500/10 rounded px-2 py-1 break-words">
          {typeof errorMessage === "string" ? errorMessage : JSON.stringify(errorMessage)}
        </div>
      )}
      {hasResult && isExpanded && (
        <div className="ml-5 mt-1 max-h-80 overflow-y-auto">
          <CodeBlock code={formatResult(result)} language="json" />
        </div>
      )}
    </div>
  );
}

// Component for displaying message attachments (images)
interface MessageAttachmentsProps {
  attachments?: Attachment[];
  isUser?: boolean;
  compact?: boolean;
}

function MessageAttachments({ attachments, isUser = false, compact = false }: MessageAttachmentsProps) {
  const [expandedImage, setExpandedImage] = useState<string | null>(null);

  if (!attachments || attachments.length === 0) {
    return null;
  }

  // Filter to only show images
  const images = attachments.filter(att =>
    att.content_type?.startsWith("image/")
  );

  if (images.length === 0) {
    return null;
  }

  const imageSize = compact ? "max-w-[150px] max-h-[150px]" : "max-w-[300px] max-h-[300px]";

  return (
    <>
      <div className={`flex flex-wrap gap-2 ${compact ? "mt-1 mb-1" : "mt-2 mb-2"}`}>
        {images.map((attachment) => {
          const resolvedUrl = resolveAttachmentUrl(attachment.url);
          return (
            <button
              key={attachment.id}
              onClick={() => setExpandedImage(resolvedUrl || null)}
              className={`relative group/img rounded-lg overflow-hidden border ${
                isUser
                  ? "border-indigo-400/30 hover:border-indigo-300/50"
                  : "border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500"
              } transition-colors cursor-pointer`}
            >
              <img
                src={resolvedUrl}
                alt={attachment.file_name}
                className={`${imageSize} object-cover`}
                loading="lazy"
              />
              <div className="absolute inset-0 bg-black/0 group-hover/img:bg-black/10 transition-colors" />
            </button>
          );
        })}
      </div>

      {/* Expanded image modal */}
      {expandedImage && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
          onClick={() => setExpandedImage(null)}
        >
          <button
            className="absolute top-4 right-4 p-2 text-white hover:text-gray-300 transition-colors"
            onClick={() => setExpandedImage(null)}
            aria-label="Close"
          >
            <XCircle className="h-8 w-8" />
          </button>
          <img
            src={expandedImage}
            alt="Expanded view"
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-lg"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </>
  );
}

interface MessageBubbleProps {
  message: Message;
  viewMode: ViewMode;
  /** All messages for computing siblings (used for navigation) */
  allMessages: Message[];
  /** Precomputed sibling info from parent (avoids recomputation per bubble) */
  siblingInfo: { current: number; total: number; siblings: Message[] };
  /** Map of tool_call_id to ToolCallRecord for status lookup */
  toolCallRecordMap: Map<string, ToolCallRecord>;
  /** Called when user requests regeneration */
  onRegenerate?: (messageId: string) => void;
  /** Called when user selects a different branch */
  onSelectBranch?: (messageId: string) => void;
  /** Called when user wants to fork the conversation from this message */
  onFork?: (messageId: string) => void;
  /** Called when user wants to edit a user message */
  onEdit?: (message: Message) => void;
  /** Whether regeneration is in progress */
  isRegenerating?: boolean;
  /** Whether forking is in progress */
  isForking?: boolean;
}

// Inner component for MessageBubble - will be wrapped with memo
const MessageBubbleInner = forwardRef<HTMLDivElement, MessageBubbleProps>(function MessageBubbleInner(
  { message, viewMode, allMessages, siblingInfo, toolCallRecordMap, onRegenerate, onSelectBranch, onFork, onEdit, isRegenerating = false, isForking = false },
  ref
) {
  const { addToast } = useToast();
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";
  const isSystem = message.role === "system";
  const isTool = message.role === "tool";
  const hasToolCalls = message.role === "assistant" && message.tool_calls && message.tool_calls.length > 0;
  const isCompact = viewMode === "compact";

  // siblingInfo is now passed from parent (MessageList) to avoid N calls to getSiblingInfo.
  // This reduces O(N²) complexity to O(N) for sibling computation.
  const hasSiblings = siblingInfo.total > 1;

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

  // Edit handler - only for user messages
  const handleEdit = useCallback(() => {
    if (onEdit && isUser) {
      onEdit(message);
    } else {
      handleComingSoon("Edit message");
    }
  }, [onEdit, isUser, message, handleComingSoon]);

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
  // NOTE: siblingInfo is NOT in dependencies - it was only used for debug logging
  // and including it caused callback recreation cascades (siblingInfo object changes often)
  const handlePreviousVersion = useCallback(() => {
    if (!onSelectBranch) {
      return;
    }
    const prevSibling = getPreviousSibling(allMessages, message.id);
    if (prevSibling) {
      onSelectBranch(prevSibling.id);
    }
  }, [onSelectBranch, allMessages, message.id]);

  const handleNextVersion = useCallback(() => {
    if (!onSelectBranch) return;
    const nextSibling = getNextSibling(allMessages, message.id);
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

    // Tool messages - CodeBlock handles copy button, so no actions needed
    return null;
  };

  // System messages - same in both modes
  if (isSystem) {
    return (
      <div ref={ref} className="flex justify-center transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className={`bg-slate-200/50 dark:bg-slate-800/50 rounded-lg px-4 py-2 text-sm text-slate-500 dark:text-slate-400 italic ${isCompact ? "w-full text-left" : "max-w-[80%]"}`}>
          <MarkdownRenderer content={message.content} />
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
          <div className="max-h-80 overflow-y-auto">
            <CodeBlock code={(() => {
              // Defensive: ensure content is a string
              const content = typeof message.content === "string"
                ? message.content
                : (message.content ? JSON.stringify(message.content, null, 2) : "");
              try {
                const parsed = JSON.parse(content);
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
            {message.tool_calls!.map((tc: ToolCall) => (
              <ToolCallItem
                key={tc.id}
                toolCall={tc}
                record={toolCallRecordMap.get(tc.id)}
                variant="compact"
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  // Bubble mode: Tool response messages
  if (isTool) {
    // Format tool content as JSON if possible
    // Defensive: content might be undefined, null, or even an object in edge cases
    const formatToolContent = (content: unknown): string => {
      if (typeof content === "string") {
        try {
          const parsed = JSON.parse(content);
          return JSON.stringify(parsed, null, 2);
        } catch {
          return content;
        }
      } else if (content && typeof content === "object") {
        return JSON.stringify(content, null, 2);
      }
      return String(content ?? "");
    };

    return (
      <div ref={ref} className="group flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
            <Wrench className="h-4 w-4 text-amber-500 dark:text-amber-400" />
          </div>
          <div className="text-slate-600 dark:text-slate-300">
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

  // Bubble mode: Assistant message with tool calls
  if (hasToolCalls) {
    return (
      <div ref={ref} className="group flex justify-start transition-all duration-300" data-testid={`message-${message.id}`}>
        <div className="flex gap-3 max-w-[85%]">
          <div className="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center shrink-0">
            <Bot className="h-4 w-4 text-indigo-400" />
          </div>
          <div className="space-y-2">
            {(message.content || (message.attachments && message.attachments.length > 0)) && (
              <div className="bg-slate-100 dark:bg-slate-800 rounded-2xl rounded-tl-md px-4 py-3 text-slate-700 dark:text-slate-200">
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
              {message.tool_calls!.map((tc: ToolCall) => (
                <ToolCallItem
                  key={tc.id}
                  toolCall={tc}
                  record={toolCallRecordMap.get(tc.id)}
                  variant="bubble"
                />
              ))}
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
});

// Wrap MessageBubble with simple memo (no custom comparison)
const MessageBubble = memo(MessageBubbleInner) as typeof MessageBubbleInner;

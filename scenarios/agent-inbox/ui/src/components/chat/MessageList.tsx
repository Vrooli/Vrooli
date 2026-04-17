import { memo, useMemo } from "react";
import { Bot } from "lucide-react";
import type { Message, ToolCallRecord } from "../../lib/api";
import type { ActiveToolCall, PendingApproval } from "../../hooks/useCompletion";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ViewMode } from "../settings/Settings";
import { getSiblingInfo } from "../../lib/messageTree";
import { MessageBubble } from "./MessageBubble";
import { ActiveToolCallsDisplay, PendingApprovalsDisplay, StreamingMessageDisplay } from "./StreamingMessage";
import { useScrollManagement } from "./useScrollManagement";
import {
  EMPTY_IMAGES,
  EMPTY_TOOL_CALLS,
  EMPTY_TOOL_RECORDS,
  EMPTY_APPROVALS,
  EMPTY_ASYNC_OPS,
  DEFAULT_SIBLING_INFO,
  EMPTY_SIBLING_MAP,
  EMPTY_ASYNC_OP_MAP,
  EMPTY_MESSAGES,
} from "./messageListTypes";

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
  /** Async operation status updates for async tools */
  asyncOperations?: AsyncStatusUpdate[];
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
  /** Called when user wants to open the async drawer for an operation */
  onOpenAsyncDrawer?: (operation: AsyncStatusUpdate) => void;
  /** Whether regeneration is in progress */
  isRegenerating?: boolean;
  /** Whether forking is in progress */
  isForking?: boolean;
}

// Inner implementation of MessageList - will be wrapped with memo
function MessageListInner({
  messages,
  allMessages,
  isGenerating,
  streamingContent,
  generatedImages = EMPTY_IMAGES,
  activeToolCalls = EMPTY_TOOL_CALLS,
  toolCallRecords = EMPTY_TOOL_RECORDS,
  asyncOperations = EMPTY_ASYNC_OPS,
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
  onOpenAsyncDrawer,
  isRegenerating = false,
  isForking = false,
}: MessageListProps) {
  const messagesForSiblings = allMessages ?? messages;

  const { endRef, messageRefs, highlightedMessageId } = useScrollManagement({
    messages,
    streamingContent,
    activeToolCalls,
    scrollToMessageId,
    onScrollComplete,
  });

  // Create lookup map from tool_call_id to ToolCallRecord for persisted tool calls
  const toolCallRecordMap = useMemo(() => {
    const map = new Map<string, ToolCallRecord>();
    for (const record of toolCallRecords) {
      map.set(record.id, record);
      const normalizedId = record.id.replace(/-/g, "");
      if (normalizedId !== record.id) {
        map.set(normalizedId, record);
      }
    }
    return map;
  }, [toolCallRecords]);

  // Create lookup map from tool_call_id to AsyncStatusUpdate
  const asyncOperationMap = useMemo(() => {
    if (asyncOperations.length === 0) return EMPTY_ASYNC_OP_MAP;
    const map = new Map<string, AsyncStatusUpdate>();
    for (const op of asyncOperations) {
      map.set(op.tool_call_id, op);
    }
    return map;
  }, [asyncOperations]);

  // Precompute siblingInfo for all assistant messages at the list level
  const siblingInfoMap = useMemo(() => {
    if (messagesForSiblings.length === 0) return EMPTY_SIBLING_MAP;
    const map = new Map<string, { current: number; total: number; siblings: Message[] }>();
    for (const message of messages) {
      if (message.role === "assistant") {
        map.set(message.id, getSiblingInfo(messagesForSiblings, message.id));
      }
    }
    return map;
  }, [messagesForSiblings, messages]);

  const isCompact = viewMode === "compact";

  // Filter out tool messages whose results are already displayed inline
  const filteredMessages = useMemo(() => {
    if (messages.length === 0) return EMPTY_MESSAGES;
    return messages.filter((message) => {
      if (message.role !== "tool" || !message.tool_call_id) return true;
      const record = toolCallRecordMap.get(message.tool_call_id);
      if (record && record.status === "completed" && record.result) return false;
      return true;
    });
  }, [messages, toolCallRecordMap]);

  // IMPORTANT: Early return MUST be AFTER all hooks to satisfy React's Rules of Hooks.
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

  return (
    <div className={`flex-1 overflow-y-auto overflow-x-hidden p-4 ${isCompact ? "space-y-2" : "space-y-4"}`} data-testid="message-list">
      {filteredMessages.map((message) => (
        <MessageBubble
          key={message.id}
          message={message}
          viewMode={viewMode}
          allMessages={messagesForSiblings}
          siblingInfo={siblingInfoMap.get(message.id) ?? DEFAULT_SIBLING_INFO}
          toolCallRecordMap={toolCallRecordMap}
          asyncOperationMap={asyncOperationMap}
          onRegenerate={onRegenerateMessage}
          onSelectBranch={onSelectBranch}
          onFork={onForkConversation}
          onEdit={onEditMessage}
          onOpenAsyncDrawer={onOpenAsyncDrawer}
          isRegenerating={isRegenerating}
          isForking={isForking}
          isHighlighted={message.id === highlightedMessageId}
          ref={(el) => {
            if (el) messageRefs.current.set(message.id, el);
            else messageRefs.current.delete(message.id);
          }}
        />
      ))}

      <ActiveToolCallsDisplay activeToolCalls={activeToolCalls} isCompact={isCompact} />

      {pendingApprovals.length > 0 && onApproveTool && onRejectTool && (
        <PendingApprovalsDisplay
          pendingApprovals={pendingApprovals}
          awaitingApprovals={awaitingApprovals}
          isProcessingApproval={isProcessingApproval}
          onApproveTool={onApproveTool}
          onRejectTool={onRejectTool}
        />
      )}

      {isGenerating && !activeToolCalls.length && (
        <StreamingMessageDisplay
          streamingContent={streamingContent}
          generatedImages={generatedImages}
          isCompact={isCompact}
        />
      )}

      <div ref={endRef} />
    </div>
  );
}

// Export MessageList with simple memo wrapper (no custom comparison)
// The parent component (ChatView) is responsible for passing stable props
export const MessageList = memo(MessageListInner);

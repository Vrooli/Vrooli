import { useMemo, useRef } from "react";
import { Loader2 } from "lucide-react";
import { ErrorBoundary } from "../ErrorBoundary";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageInput, type MessagePayload } from "./MessageInput";
import { AsyncOperationsPanel } from "./AsyncOperationCard";
import type { ChatWithMessages, Model, Label, Message } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ViewMode } from "../settings/Settings";
import { computeVisibleMessages } from "../../lib/messageTree";

// Stable empty arrays for default prop values and useMemo returns
// CRITICAL: Using `= []` or `return []` creates a NEW array on every render/recalculation,
// which changes references and triggers infinite re-render loops via useMemo dependencies
const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
const EMPTY_IMAGES: string[] = [];
const EMPTY_ASYNC_OPS: AsyncStatusUpdate[] = [];
const EMPTY_MESSAGES: Message[] = [];
const EMPTY_TOOL_RECORDS: import("../../lib/api").ToolCallRecord[] = [];

interface ChatViewProps {
  chatData: ChatWithMessages | null;
  models: Model[];
  labels: Label[];
  isLoading: boolean;
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls?: ActiveToolCall[];
  generatedImages?: string[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
  onSendMessage: (payload: MessagePayload) => void;
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDeleteChat: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
  viewMode?: ViewMode;
  // Branching operations
  onRegenerateMessage?: (messageId: string) => void;
  onSelectBranch?: (messageId: string) => void;
  onForkConversation?: (messageId: string) => void;
  isRegenerating?: boolean;
  isForking?: boolean;
  // Edit operations
  editingMessage?: Message | null;
  onEditMessage?: (message: Message) => void;
  onCancelEdit?: () => void;
  onSubmitEdit?: (payload: MessagePayload) => void;
  // Async operations
  asyncOperations?: AsyncStatusUpdate[];
  onCancelAsyncOperation?: (toolCallId: string) => Promise<void>;
  // Template activation (for template-to-tool linking)
  onTemplateActivated?: (templateId: string, toolIds: string[]) => Promise<void>;
  /** Currently active template ID (for UI indicator) */
  activeTemplateId?: string | null;
  /** Callback to deactivate the active template */
  onTemplateDeactivate?: () => void;
}

export function ChatView({
  chatData,
  models,
  labels,
  isLoading,
  isGenerating,
  streamingContent,
  activeToolCalls = EMPTY_TOOL_CALLS,
  generatedImages = EMPTY_IMAGES,
  scrollToMessageId,
  onScrollComplete,
  onSendMessage,
  onUpdateChat,
  onToggleRead,
  onToggleStar,
  onToggleArchive,
  onDeleteChat,
  onAssignLabel,
  onRemoveLabel,
  viewMode,
  onRegenerateMessage,
  onSelectBranch,
  onForkConversation,
  isRegenerating = false,
  isForking = false,
  editingMessage,
  onEditMessage,
  onCancelEdit,
  onSubmitEdit,
  asyncOperations = EMPTY_ASYNC_OPS,
  onCancelAsyncOperation,
  onTemplateActivated,
  activeTemplateId,
  onTemplateDeactivate,
}: ChatViewProps) {
  // DEBUG: Track renders
  const renderCount = useRef(0);
  renderCount.current++;
  const globalSeq = typeof window !== 'undefined' && window.__getNextRenderSeq__ ? window.__getNextRenderSeq__() : 0;
  console.log(`[ChatView] ===== Render #${renderCount.current} (global seq: ${globalSeq}) START =====`, {
    chatId: chatData?.chat?.id,
    messageCount: chatData?.messages?.length,
    messagesIsArray: Array.isArray(chatData?.messages),
    messagesUndefined: chatData?.messages === undefined,
    messagesNull: chatData?.messages === null,
    chatDataExists: !!chatData,
    isLoading,
    isGenerating,
  });

  // Compute visible messages based on the active branch
  // This filters the full message tree to only show the active path
  // NOTE: Must be called before any early returns to satisfy React's rules of hooks

  // Memoize allMessages to avoid creating new array references on each render.
  // CRITICAL: We must return EMPTY_MESSAGES for BOTH undefined AND empty arrays.
  // The nullish coalescing operator (??) only handles null/undefined, but [] is truthy.
  // Without this check, an empty messages array from the API would create a new
  // reference on every render, potentially causing infinite re-render loops (React #310).
  console.log("[ChatView] BEFORE allMessages useMemo, chatData?.messages:", chatData?.messages?.length);
  const allMessages = useMemo(() => {
    console.log("[ChatView] INSIDE allMessages useMemo");
    const messages = chatData?.messages;
    if (!messages || messages.length === 0) {
      console.log("[ChatView] allMessages returning EMPTY_MESSAGES");
      return EMPTY_MESSAGES;
    }
    console.log("[ChatView] allMessages returning messages array, length:", messages.length);
    return messages;
  }, [chatData?.messages]);
  console.log("[ChatView] AFTER allMessages useMemo, result length:", allMessages.length);
  const activeLeafId = chatData?.chat?.active_leaf_message_id ?? null;

  // CRITICAL: Must use stable EMPTY_MESSAGES, not inline [] which creates new reference each time
  // NOTE: Do NOT include `chatData` in dependencies! The computation only uses `allMessages` and
  // `activeLeafId`. Including `chatData` causes unnecessary recalculations whenever ANY chatData
  // property changes (tool_call_records, chat name, etc.), potentially creating cascading re-renders.
  console.log("[ChatView] BEFORE visibleMessages useMemo");
  const visibleMessages = useMemo(() => {
    console.log("[ChatView] INSIDE visibleMessages useMemo, allMessages:", allMessages.length, "activeLeafId:", activeLeafId);
    if (allMessages.length === 0) return EMPTY_MESSAGES;
    return computeVisibleMessages(allMessages, activeLeafId ?? undefined);
  }, [allMessages, activeLeafId]);
  console.log("[ChatView] AFTER visibleMessages useMemo, result length:", visibleMessages.length);

  // NOTE: We previously used useDeferredValue here to try to reduce render storms, but it caused
  // a critical bug: the deferred values would lag behind, creating a mismatch where ChatView had
  // messages but MessageList received an empty array. This caused "too many re-renders" errors
  // because MessageList's useMemo calculations would see inconsistent state (isGenerating=true
  // but messages=[]).

  // Memoize tool call records to prevent cascading re-renders in MessageList
  // CRITICAL: The API returns a NEW empty array [] on each response. The default parameter
  // `= EMPTY_TOOL_RECORDS` in MessageList only applies for `undefined`, NOT for `[]`.
  // Without this memoization, each query cache update creates new array references,
  // causing useMemo dependency chains to recalculate and potentially infinite render loops.
  console.log("[ChatView] BEFORE stableToolCallRecords useMemo");
  const stableToolCallRecords = useMemo(() => {
    console.log("[ChatView] INSIDE stableToolCallRecords useMemo, records:", chatData?.tool_call_records?.length);
    const records = chatData?.tool_call_records;
    if (!records || records.length === 0) {
      return EMPTY_TOOL_RECORDS;
    }
    return records;
  }, [chatData?.tool_call_records]);
  console.log("[ChatView] AFTER stableToolCallRecords useMemo");

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-950" data-testid="chat-view-loading">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-400 mx-auto mb-3" />
          <p className="text-sm text-slate-500">Loading conversation...</p>
        </div>
      </div>
    );
  }

  if (!chatData) {
    return null;
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-slate-950" data-testid="chat-view">
      <ErrorBoundary name="ChatHeader">
        <ChatHeader
          chat={chatData.chat}
          models={models}
          labels={labels}
          onUpdateChat={onUpdateChat}
          onToggleRead={onToggleRead}
          onToggleStar={onToggleStar}
          onToggleArchive={onToggleArchive}
          onDelete={onDeleteChat}
          onAssignLabel={onAssignLabel}
          onRemoveLabel={onRemoveLabel}
        />
      </ErrorBoundary>

      {/* Async Operations Panel - shows active long-running tool operations */}
      {asyncOperations.length > 0 && (
        <ErrorBoundary name="AsyncOperationsPanel">
          <AsyncOperationsPanel
            operations={asyncOperations}
            onCancel={onCancelAsyncOperation}
          />
        </ErrorBoundary>
      )}

      <ErrorBoundary name="MessageList">
        <MessageList
          messages={visibleMessages}
          allMessages={allMessages}
          isGenerating={isGenerating}
          streamingContent={streamingContent}
          activeToolCalls={activeToolCalls}
          generatedImages={generatedImages}
          toolCallRecords={stableToolCallRecords}
          scrollToMessageId={scrollToMessageId}
          onScrollComplete={onScrollComplete}
          viewMode={viewMode}
          onRegenerateMessage={onRegenerateMessage}
          onSelectBranch={onSelectBranch}
          onForkConversation={onForkConversation}
          onEditMessage={onEditMessage}
          isRegenerating={isRegenerating}
          isForking={isForking}
        />
      </ErrorBoundary>

      <div className="border-t border-white/10 bg-slate-950/50">
        <ErrorBoundary name="MessageInput">
          <MessageInput
            onSend={onSendMessage}
            isLoading={isGenerating}
            currentModel={models.find((m) => m.id === chatData.chat.model) || null}
            chatId={chatData.chat.id}
            chatWebSearchDefault={chatData.chat.web_search_enabled || false}
            editingMessage={editingMessage}
            onCancelEdit={onCancelEdit}
            onSubmitEdit={onSubmitEdit}
            onTemplateActivated={onTemplateActivated}
            activeTemplateId={activeTemplateId}
            onTemplateDeactivate={onTemplateDeactivate}
          />
        </ErrorBoundary>
      </div>
    </div>
  );
}

import { useMemo } from "react";
import { Loader2 } from "lucide-react";
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
}: ChatViewProps) {
  // Compute visible messages based on the active branch
  // This filters the full message tree to only show the active path
  // NOTE: Must be called before any early returns to satisfy React's rules of hooks

  // Memoize allMessages to avoid creating new array references on each render
  // when chatData?.messages is undefined (which would cause infinite re-renders)
  // CRITICAL: Must use stable EMPTY_MESSAGES, not inline [] which creates new reference each time
  const allMessages = useMemo(
    () => chatData?.messages ?? EMPTY_MESSAGES,
    [chatData?.messages]
  );
  const activeLeafId = chatData?.chat?.active_leaf_message_id ?? null;

  // CRITICAL: Must use stable EMPTY_MESSAGES, not inline [] which creates new reference each time
  const visibleMessages = useMemo(() => {
    if (!chatData || allMessages.length === 0) return EMPTY_MESSAGES;
    return computeVisibleMessages(allMessages, activeLeafId ?? undefined);
  }, [chatData, allMessages, activeLeafId]);

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
      <ChatHeader
        chat={chatData.chat}
        models={models}
        labels={labels}
        onUpdateChat={(data) => onUpdateChat(data)}
        onToggleRead={onToggleRead}
        onToggleStar={onToggleStar}
        onToggleArchive={onToggleArchive}
        onDelete={onDeleteChat}
        onAssignLabel={onAssignLabel}
        onRemoveLabel={onRemoveLabel}
      />

      {/* Async Operations Panel - shows active long-running tool operations */}
      {asyncOperations.length > 0 && (
        <AsyncOperationsPanel
          operations={asyncOperations}
          onCancel={onCancelAsyncOperation}
        />
      )}

      <MessageList
        messages={visibleMessages}
        allMessages={allMessages}
        isGenerating={isGenerating}
        streamingContent={streamingContent}
        activeToolCalls={activeToolCalls}
        generatedImages={generatedImages}
        toolCallRecords={chatData.tool_call_records}
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

      <div className="border-t border-white/10 bg-slate-950/50">
        <MessageInput
          onSend={onSendMessage}
          isLoading={isGenerating}
          currentModel={models.find((m) => m.id === chatData.chat.model) || null}
          chatId={chatData.chat.id}
          chatWebSearchDefault={chatData.chat.web_search_enabled || false}
          editingMessage={editingMessage}
          onCancelEdit={onCancelEdit}
          onSubmitEdit={onSubmitEdit}
        />
      </div>
    </div>
  );
}

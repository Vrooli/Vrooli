import { useMemo, useState, useCallback } from "react";
import { Loader2 } from "lucide-react";
import { ErrorBoundary } from "../ErrorBoundary";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import type { MessagePayload } from "./MessageInput";
import { AsyncStatusBar } from "./AsyncStatusBar";
import { AsyncOperationDrawer } from "./AsyncOperationDrawer";
import { AgentEventList } from "./agent/AgentEventList";
import type { AsyncResultReference } from "./AsyncResultChip";
import type { ChatWithMessages, Model, Label, Message } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ViewMode } from "../settings/Settings";
import { computeVisibleMessages } from "../../lib/messageTree";
import { useAgentChatMode } from "./useAgentChatMode";
import { ChatViewFooter } from "./ChatViewFooter";

// Stable empty arrays for default prop values
const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
const EMPTY_IMAGES: string[] = [];
const EMPTY_ASYNC_OPS: AsyncStatusUpdate[] = [];
const EMPTY_MESSAGES: Message[] = [];
const EMPTY_TOOL_RECORDS: import("../../lib/api").ToolCallRecord[] = [];
const EMPTY_ASYNC_REFS: AsyncResultReference[] = [];

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
  onRegenerateMessage?: (messageId: string) => void;
  onSelectBranch?: (messageId: string) => void;
  onForkConversation?: (messageId: string) => void;
  isRegenerating?: boolean;
  isForking?: boolean;
  editingMessage?: Message | null;
  onEditMessage?: (message: Message) => void;
  onCancelEdit?: () => void;
  onSubmitEdit?: (payload: MessagePayload) => void;
  asyncOperations?: AsyncStatusUpdate[];
  activeAsyncOperations?: AsyncStatusUpdate[];
  completedAsyncOperations?: AsyncStatusUpdate[];
  onCancelAsyncOperation?: (toolCallId: string) => Promise<void>;
  onRefreshAsyncOperation?: (toolCallId: string) => Promise<AsyncStatusUpdate>;
  onFetchAsyncHistory?: () => Promise<void>;
  hasMoreAsyncHistory?: boolean;
  asyncReferences?: AsyncResultReference[];
  onInsertAsyncReference?: (operation: AsyncStatusUpdate) => void;
  onRemoveAsyncReference?: (toolCallId: string) => void;
  onTemplateActivated?: (templateId: string) => Promise<void>;
  activeTemplateId?: string | null;
  onTemplateDeactivate?: () => void;
  onRefreshChat?: () => void;
  onOpenAgentSettings?: () => void;
  onBackToList?: () => void;
  isMobile?: boolean;
  onOpenSidebar?: () => void;
}

export function ChatView({
  chatData, models, labels, isLoading, isGenerating, streamingContent,
  activeToolCalls = EMPTY_TOOL_CALLS, generatedImages = EMPTY_IMAGES,
  scrollToMessageId, onScrollComplete, onSendMessage, onUpdateChat,
  onToggleRead, onToggleStar, onToggleArchive, onDeleteChat,
  onAssignLabel, onRemoveLabel, viewMode,
  onRegenerateMessage, onSelectBranch, onForkConversation,
  isRegenerating = false, isForking = false,
  editingMessage, onEditMessage, onCancelEdit, onSubmitEdit,
  asyncOperations = EMPTY_ASYNC_OPS,
  activeAsyncOperations = EMPTY_ASYNC_OPS,
  completedAsyncOperations = EMPTY_ASYNC_OPS,
  onCancelAsyncOperation, onRefreshAsyncOperation,
  onFetchAsyncHistory, hasMoreAsyncHistory = false,
  asyncReferences: _asyncReferences = EMPTY_ASYNC_REFS,
  onInsertAsyncReference, onRemoveAsyncReference: _onRemoveAsyncReference,
  onTemplateActivated, activeTemplateId, onTemplateDeactivate,
  onRefreshChat, onOpenAgentSettings: _onOpenAgentSettings,
  onBackToList, isMobile, onOpenSidebar,
}: ChatViewProps) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedOperation, setSelectedOperation] = useState<AsyncStatusUpdate | null>(null);
  const [statusBarCollapsed, setStatusBarCollapsed] = useState(false);

  const agent = useAgentChatMode({ chatData, onRefreshChat, onSendLlmMessage: onSendMessage });

  const handleOpenDrawer = useCallback((operation?: AsyncStatusUpdate) => {
    setSelectedOperation(operation ?? null);
    setDrawerOpen(true);
  }, []);

  const handleInsertReference = useCallback((operation: AsyncStatusUpdate) => {
    onInsertAsyncReference?.(operation);
    setDrawerOpen(false);
  }, [onInsertAsyncReference]);

  const handleRefreshOperation = useCallback(async (toolCallId: string) => {
    if (onRefreshAsyncOperation) await onRefreshAsyncOperation(toolCallId);
  }, [onRefreshAsyncOperation]);

  const handleCancelOperation = useCallback(async (toolCallId: string) => {
    if (onCancelAsyncOperation) await onCancelAsyncOperation(toolCallId);
  }, [onCancelAsyncOperation]);

  const handleLoadMoreHistory = useCallback(async () => {
    if (onFetchAsyncHistory) await onFetchAsyncHistory();
  }, [onFetchAsyncHistory]);

  const allMessages = useMemo(() => {
    const messages = chatData?.messages;
    if (!messages || messages.length === 0) return EMPTY_MESSAGES;
    return messages;
  }, [chatData?.messages]);
  const activeLeafId = chatData?.chat.active_leaf_message_id ?? null;

  const visibleMessages = useMemo(() => {
    if (allMessages.length === 0) return EMPTY_MESSAGES;
    return computeVisibleMessages(allMessages, activeLeafId ?? undefined);
  }, [allMessages, activeLeafId]);

  const stableToolCallRecords = useMemo(() => {
    const records = chatData?.tool_call_records;
    if (!records || records.length === 0) return EMPTY_TOOL_RECORDS;
    return records;
  }, [chatData?.tool_call_records]);

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

  if (!chatData) return null;

  return (
    <div className="flex-1 flex flex-col min-h-0 min-w-0 bg-slate-950" data-testid="chat-view">
      <ErrorBoundary name="ChatHeader">
        <ChatHeader
          chat={chatData.chat} models={models} labels={labels}
          chatMode={agent.chatMode} onUpdateChat={onUpdateChat}
          onToggleRead={onToggleRead} onToggleStar={onToggleStar}
          onToggleArchive={onToggleArchive} onDelete={onDeleteChat}
          onAssignLabel={onAssignLabel} onRemoveLabel={onRemoveLabel}
          isAgentActive={agent.isAgentActive} agentStatus={agent.agentStatus}
          agentMetrics={agent.agentMetrics} agentError={agent.agentError}
          onStopAgent={() => { void agent.handleStopAgent(); }} onBackToList={onBackToList}
          isMobile={isMobile} onOpenSidebar={onOpenSidebar}
          hasMessages={visibleMessages.length > 0}
          onModeChange={agent.setChatMode} onOpenAgentSettings={_onOpenAgentSettings}
        />
      </ErrorBoundary>

      {!agent.isAgentActive && (activeAsyncOperations.length > 0 || completedAsyncOperations.length > 0) && (
        <ErrorBoundary name="AsyncStatusBar">
          <AsyncStatusBar
            operations={asyncOperations}
            completedCount={completedAsyncOperations.length}
            onRefresh={handleRefreshOperation} onCancel={handleCancelOperation}
            onOpenDrawer={handleOpenDrawer} isCollapsed={statusBarCollapsed}
            onToggleCollapse={() => setStatusBarCollapsed(!statusBarCollapsed)}
          />
        </ErrorBoundary>
      )}

      <AsyncOperationDrawer
        isOpen={drawerOpen} onClose={() => setDrawerOpen(false)}
        operation={selectedOperation} completedOperations={completedAsyncOperations}
        onRefresh={handleRefreshOperation} onCancel={handleCancelOperation}
        onInsertReference={handleInsertReference}
        onLoadMoreHistory={handleLoadMoreHistory} hasMoreHistory={hasMoreAsyncHistory}
      />

      {agent.isAgentActive ? (
        <ErrorBoundary name="AgentEventList">
          <AgentEventList
            events={agent.agentEvents} autoScroll={!scrollToMessageId}
            viewMode={viewMode} initialMessages={visibleMessages}
            scrollToMessageId={scrollToMessageId} onScrollComplete={onScrollComplete}
          />
        </ErrorBoundary>
      ) : (
        <ErrorBoundary name="MessageList">
          <MessageList
            messages={visibleMessages} allMessages={allMessages}
            isGenerating={isGenerating} streamingContent={streamingContent}
            activeToolCalls={activeToolCalls} generatedImages={generatedImages}
            toolCallRecords={stableToolCallRecords} asyncOperations={asyncOperations}
            scrollToMessageId={scrollToMessageId} onScrollComplete={onScrollComplete}
            viewMode={viewMode} onRegenerateMessage={onRegenerateMessage}
            onSelectBranch={onSelectBranch} onForkConversation={onForkConversation}
            onEditMessage={onEditMessage} onOpenAsyncDrawer={handleOpenDrawer}
            isRegenerating={isRegenerating} isForking={isForking}
          />
        </ErrorBoundary>
      )}

      <ChatViewFooter
        agent={agent} isGenerating={isGenerating} models={models}
        chatModel={chatData.chat.model} chatId={chatData.chat.id}
        chatWebSearchEnabled={chatData.chat.web_search_enabled || false}
        editingMessage={editingMessage} onCancelEdit={onCancelEdit}
        onSubmitEdit={onSubmitEdit} onTemplateActivated={onTemplateActivated}
        activeTemplateId={activeTemplateId} onTemplateDeactivate={onTemplateDeactivate}
      />
    </div>
  );
}

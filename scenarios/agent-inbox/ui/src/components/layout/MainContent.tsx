import { Suspense, lazy } from "react";
import { ErrorBoundary } from "../ErrorBoundary";
import { EmptyState } from "../chat/EmptyState";
import type { ViewMode } from "../settings/Settings";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { AsyncResultReference } from "../chat/AsyncResultChip";
import type { MessagePayload } from "../chat/MessageInput";
import type { AgentStartConfig } from "../chat/AgentStartModal";
import type { AgentRunSummary } from "../../lib/api";
import type { ChatWithMessages, Label, Message } from "../../lib/api-types";
import type { Model } from "../../lib/api-models";

const LazyChatView = lazy(async () => {
  const module = await import("../chat/ChatView");
  return { default: module.ChatView };
});

interface MainContentProps {
  selectedChatId: string | null;
  chatData: ChatWithMessages | null | undefined;
  models: Model[];
  labels: Label[];
  loadingChat: boolean;
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls: ActiveToolCall[];
  generatedImages: string[];
  scrollToMessageId: string | null;
  chatViewCallbacks: {
    handleScrollComplete: () => void;
    handleUpdateChatFromView: (data: Record<string, unknown>) => void;
    handleToggleReadFromView: () => void;
    handleToggleStarFromView: () => void;
    handleToggleArchiveFromView: () => void;
    handleDeleteChatFromView: () => void;
    handleAssignLabelFromView: (labelId: string) => void;
    handleRemoveLabelFromView: (labelId: string) => void;
    handleRegenerateMessageFromView: (messageId: string) => void;
    handleSubmitEditFromView: (payload: MessagePayload) => void;
    handleRefreshChat: () => void;
  };
  sendMessage: (payload: MessagePayload) => void;
  viewMode: ViewMode;
  selectBranch: (messageId: string) => void;
  forkConversation: (messageId: string) => void;
  isRegenerating: boolean;
  isForking: boolean;
  editingMessage: Message | null;
  setEditingMessage: (msg: Message | null) => void;
  cancelEdit: () => void;
  asyncOperations: AsyncStatusUpdate[];
  activeAsyncOperations: AsyncStatusUpdate[];
  completedAsyncOperations: AsyncStatusUpdate[];
  cancelAsyncOperation: (toolCallId: string) => Promise<void>;
  refreshAsyncOperation: (toolCallId: string) => Promise<AsyncStatusUpdate>;
  handleFetchAsyncHistory: () => Promise<void>;
  hasMoreAsyncHistory: boolean;
  asyncReferences: AsyncResultReference[];
  handleInsertAsyncReference: (op: AsyncStatusUpdate) => void;
  handleRemoveAsyncReference: (toolCallId: string) => void;
  handleTemplateActivated: (templateId: string, toolIds: string[]) => Promise<void>;
  activeTemplateId: string | null;
  onTemplateDeactivate: () => void;
  handleOpenAgentSettings: () => void;
  handleBackToList: () => void;
  isMobile: boolean;
  setSidebarOpen: (open: boolean) => void;
  chatListOpen: boolean;
  // EmptyState props
  createChatWithMessage: (payload: MessagePayload) => void;
  handleStartAgentChat: (payload: MessagePayload, config: AgentStartConfig) => void;
  handleAttachRunFromEmpty: (run: AgentRunSummary) => void;
  isCreatingChat: boolean;
}

export function MainContent({
  selectedChatId,
  chatData,
  models,
  labels,
  loadingChat,
  isGenerating,
  streamingContent,
  activeToolCalls,
  generatedImages,
  scrollToMessageId,
  chatViewCallbacks,
  sendMessage,
  viewMode,
  selectBranch,
  forkConversation,
  isRegenerating,
  isForking,
  editingMessage,
  setEditingMessage,
  cancelEdit,
  asyncOperations,
  activeAsyncOperations,
  completedAsyncOperations,
  cancelAsyncOperation,
  refreshAsyncOperation,
  handleFetchAsyncHistory,
  hasMoreAsyncHistory,
  asyncReferences,
  handleInsertAsyncReference,
  handleRemoveAsyncReference,
  handleTemplateActivated,
  activeTemplateId,
  onTemplateDeactivate,
  handleOpenAgentSettings,
  handleBackToList,
  isMobile,
  setSidebarOpen,
  chatListOpen,
  createChatWithMessage,
  handleStartAgentChat,
  handleAttachRunFromEmpty,
  isCreatingChat,
}: MainContentProps) {
  return (
    <div
      className={`flex-1 flex flex-col min-h-0 min-w-0 ${
        chatListOpen && selectedChatId ? "hidden lg:flex" : "flex"
      }`}
    >
      <ErrorBoundary name="ChatContent">
        {selectedChatId ? (
          <Suspense fallback={(
            <div className="flex-1 flex items-center justify-center bg-slate-950">
              <span className="text-sm text-slate-400">Loading chat...</span>
            </div>
          )}>
            <LazyChatView
              key={selectedChatId}
              chatData={chatData?.chat.id === selectedChatId ? chatData : null}
              models={models}
              labels={labels}
              isLoading={loadingChat || (!!selectedChatId && chatData?.chat.id !== selectedChatId)}
              isGenerating={isGenerating}
              streamingContent={streamingContent}
              activeToolCalls={activeToolCalls}
              generatedImages={generatedImages}
              scrollToMessageId={scrollToMessageId}
              onScrollComplete={chatViewCallbacks.handleScrollComplete}
              onSendMessage={sendMessage}
              onUpdateChat={chatViewCallbacks.handleUpdateChatFromView}
              onToggleRead={chatViewCallbacks.handleToggleReadFromView}
              onToggleStar={chatViewCallbacks.handleToggleStarFromView}
              onToggleArchive={chatViewCallbacks.handleToggleArchiveFromView}
              onDeleteChat={chatViewCallbacks.handleDeleteChatFromView}
              onAssignLabel={chatViewCallbacks.handleAssignLabelFromView}
              onRemoveLabel={chatViewCallbacks.handleRemoveLabelFromView}
              viewMode={viewMode}
              onRegenerateMessage={chatViewCallbacks.handleRegenerateMessageFromView}
              onSelectBranch={selectBranch}
              onForkConversation={forkConversation}
              isRegenerating={isRegenerating}
              isForking={isForking}
              editingMessage={editingMessage}
              onEditMessage={setEditingMessage}
              onCancelEdit={cancelEdit}
              onSubmitEdit={chatViewCallbacks.handleSubmitEditFromView}
              asyncOperations={asyncOperations}
              activeAsyncOperations={activeAsyncOperations}
              completedAsyncOperations={completedAsyncOperations}
              onCancelAsyncOperation={cancelAsyncOperation}
              onRefreshAsyncOperation={refreshAsyncOperation}
              onFetchAsyncHistory={handleFetchAsyncHistory}
              hasMoreAsyncHistory={hasMoreAsyncHistory}
              asyncReferences={asyncReferences}
              onInsertAsyncReference={handleInsertAsyncReference}
              onRemoveAsyncReference={handleRemoveAsyncReference}
              onTemplateActivated={handleTemplateActivated}
              activeTemplateId={activeTemplateId}
              onTemplateDeactivate={onTemplateDeactivate}
              onRefreshChat={chatViewCallbacks.handleRefreshChat}
              onOpenAgentSettings={handleOpenAgentSettings}
              onBackToList={handleBackToList}
              isMobile={isMobile}
              onOpenSidebar={() => setSidebarOpen(true)}
            />
          </Suspense>
        ) : (
          <EmptyState
            onStartChat={createChatWithMessage}
            onStartAgentChat={handleStartAgentChat}
            onAttachRun={handleAttachRunFromEmpty}
            onOpenAgentSettings={handleOpenAgentSettings}
            isCreating={isCreatingChat}
            models={models}
            isMobile={isMobile}
            onOpenSidebar={() => setSidebarOpen(true)}
          />
        )}
      </ErrorBoundary>
    </div>
  );
}

import { useMemo, useState, useCallback, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { ErrorBoundary } from "../ErrorBoundary";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageInput, type MessagePayload } from "./MessageInput";
import { AsyncStatusBar } from "./AsyncStatusBar";
import { AsyncOperationDrawer } from "./AsyncOperationDrawer";
import type { ChatMode } from "./ModeSelector";
import { AgentStartModal, type AgentStartConfig } from "./AgentStartModal";
import { AttachRunModal } from "./AttachRunModal";
import { AgentEventList, type AgentMetric } from "./agent/AgentEventList";
import type { AsyncResultReference } from "./AsyncResultChip";
import type { ChatWithMessages, Model, Label, Message, AgentChatConfig } from "../../lib/api";
import { startAgentMode, sendAgentMessage, stopAgentMode, attachAgentRun, AgentModeError } from "../../lib/api";
import type { AgentRunSummary } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import type { ViewMode } from "../settings/Settings";
import { computeVisibleMessages } from "../../lib/messageTree";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import { useAgentWebSocket } from "../../hooks/useAgentWebSocket";

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
  activeAsyncOperations?: AsyncStatusUpdate[];
  completedAsyncOperations?: AsyncStatusUpdate[];
  onCancelAsyncOperation?: (toolCallId: string) => Promise<void>;
  onRefreshAsyncOperation?: (toolCallId: string) => Promise<AsyncStatusUpdate>;
  onFetchAsyncHistory?: () => Promise<void>;
  hasMoreAsyncHistory?: boolean;
  // Async result references
  asyncReferences?: AsyncResultReference[];
  onInsertAsyncReference?: (operation: AsyncStatusUpdate) => void;
  onRemoveAsyncReference?: (toolCallId: string) => void;
  // Template activation (for template-to-tool linking)
  onTemplateActivated?: (templateId: string, toolIds: string[]) => Promise<void>;
  /** Currently active template ID (for UI indicator) */
  activeTemplateId?: string | null;
  /** Callback to deactivate the active template */
  onTemplateDeactivate?: () => void;
  // Agent mode
  /** Callback to refresh chat data after agent mode changes */
  onRefreshChat?: () => void;
  /** Open settings focused on agent tab */
  onOpenAgentSettings?: () => void;
  /** Mobile: go back to chat list */
  onBackToList?: () => void;
  /** Whether the viewport is mobile-sized */
  isMobile?: boolean;
  /** Mobile: open the sidebar */
  onOpenSidebar?: () => void;
}

// Stable empty array for async references
const EMPTY_ASYNC_REFS: AsyncResultReference[] = [];

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
  activeAsyncOperations = EMPTY_ASYNC_OPS,
  completedAsyncOperations = EMPTY_ASYNC_OPS,
  onCancelAsyncOperation,
  onRefreshAsyncOperation,
  onFetchAsyncHistory,
  hasMoreAsyncHistory = false,
  asyncReferences: _asyncReferences = EMPTY_ASYNC_REFS,
  onInsertAsyncReference,
  onRemoveAsyncReference: _onRemoveAsyncReference,
  onTemplateActivated,
  activeTemplateId,
  onTemplateDeactivate,
  onRefreshChat,
  onOpenAgentSettings: _onOpenAgentSettings,
  onBackToList,
  isMobile,
  onOpenSidebar,
}: ChatViewProps) {
  // Async operations drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedOperation, setSelectedOperation] = useState<AsyncStatusUpdate | null>(null);
  const [statusBarCollapsed, setStatusBarCollapsed] = useState(false);

  // Agent mode state
  const { settings: agentSettings } = useAgentSettings();
  const [chatMode, setChatMode] = useState<ChatMode>(chatData?.chat?.chat_mode || "llm");
  const [isStartingAgent, setIsStartingAgent] = useState(false);
  const [showAgentStartModal, setShowAgentStartModal] = useState(false);
  const [pendingAgentMessage, setPendingAgentMessage] = useState<string>("");
  const [agentError, setAgentError] = useState<{ message: string; recovery?: string } | null>(null);
  const [queuedMessage, setQueuedMessage] = useState<MessagePayload | null>(null);
  const [showAttachModal, setShowAttachModal] = useState(false);
  const [isAttaching, setIsAttaching] = useState(false);

  // Sync chatMode with server state when chat data loads or changes.
  // Without this, chatMode stays "llm" because useState initializer runs once
  // when chatData is still null (loading), and never updates when it arrives.
  useEffect(() => {
    if (chatData?.chat?.chat_mode) {
      setChatMode(chatData.chat.chat_mode as ChatMode);
    }
  }, [chatData?.chat?.chat_mode]);

  // Check if agent is currently active
  const isAgentActive = chatData?.chat?.chat_mode === "agent" && !!chatData?.chat?.agent_run_id;

  // Stable callback for agent status changes.
  // Must be wrapped in useCallback to avoid creating a new function reference
  // on every render, which would destabilize useAgentWebSocket's polling loop.
  const handleAgentStatusChange = useCallback((newStatus: import("../../lib/api").AgentModeStatus) => {
    // Refresh chat when agent completes or fails
    if (["complete", "failed", "cancelled"].includes(newStatus.status || "")) {
      onRefreshChat?.();
    }
  }, [onRefreshChat]);

  // Agent WebSocket for real-time events
  const {
    events: agentEvents,
    status: agentStatus,
    isConnected: _isAgentConnected,
    error: _agentWsError,
    refresh: refreshAgentEvents
  } = useAgentWebSocket({
    chatId: chatData?.chat?.id || null,
    runId: chatData?.chat?.agent_run_id || null,
    enabled: isAgentActive,
    onStatusChange: handleAgentStatusChange
  });

  // Whether the agent is actively processing (blocks sending but not typing)
  const agentBusy = isAgentActive && !!agentStatus?.status
    && ["pending", "starting", "running"].includes(agentStatus.status);

  // Aggregate metric events for display in the status header
  const agentMetrics = useMemo((): AgentMetric[] => {
    if (!agentEvents || agentEvents.length === 0) return [];
    const metrics: AgentMetric[] = [];
    for (const ev of agentEvents) {
      if (ev.type !== "metric" || !ev.raw_data) continue;
      try {
        const parsed = JSON.parse(ev.raw_data) as Record<string, unknown>;
        if (parsed.name && typeof parsed.value === "number") {
          metrics.push({
            name: parsed.name as string,
            value: parsed.value,
            unit: (parsed.unit as string) || "",
            tags: parsed.tags as Record<string, string> | undefined,
          });
        }
      } catch {
        // Skip unparseable metric events
      }
    }
    return metrics;
  }, [agentEvents]);

  // Handle starting agent mode
  const handleStartAgent = useCallback(async (config: AgentStartConfig) => {
    if (!chatData?.chat?.id || !pendingAgentMessage) return;

    setIsStartingAgent(true);
    setAgentError(null);

    try {
      const agentConfig: AgentChatConfig = {
        message: pendingAgentMessage,
        runner_type: config.runner_type,
        project_path: config.project_path,
        model: config.model || undefined,
        max_turns: config.max_turns || undefined
      };

      await startAgentMode(chatData.chat.id, agentConfig);
      setShowAgentStartModal(false);
      setPendingAgentMessage("");
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) {
        setAgentError({ message: e.message, recovery: e.recovery });
      } else {
        setAgentError({ message: e instanceof Error ? e.message : "Failed to start agent" });
      }
    } finally {
      setIsStartingAgent(false);
    }
  }, [chatData?.chat?.id, pendingAgentMessage, onRefreshChat]);

  // Handle sending message in agent mode
  const handleSendAgentMessage = useCallback(async (message: string) => {
    if (!chatData?.chat?.id) return;

    // If not in agent mode yet, show the start modal
    if (!isAgentActive) {
      setPendingAgentMessage(message);
      setShowAgentStartModal(true);
      return;
    }

    // Continue existing run
    try {
      setAgentError(null);
      await sendAgentMessage(chatData.chat.id, message);
      refreshAgentEvents();
    } catch (e) {
      if (e instanceof AgentModeError) {
        setAgentError({ message: e.message, recovery: e.recovery });
      } else {
        setAgentError({ message: e instanceof Error ? e.message : "Failed to send message" });
      }
    }
  }, [chatData?.chat?.id, isAgentActive, refreshAgentEvents]);

  // Auto-send queued message when agent finishes
  useEffect(() => {
    if (!agentBusy && queuedMessage) {
      const payload = queuedMessage;
      setQueuedMessage(null);
      handleSendAgentMessage(payload.content);
    }
  }, [agentBusy, queuedMessage, handleSendAgentMessage]);

  // Handle stopping agent
  const handleStopAgent = useCallback(async () => {
    if (!chatData?.chat?.id) return;

    try {
      await stopAgentMode(chatData.chat.id);
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) {
        setAgentError({ message: e.message, recovery: e.recovery });
      } else {
        setAgentError({ message: e instanceof Error ? e.message : "Failed to stop agent" });
      }
    }
  }, [chatData?.chat?.id, onRefreshChat]);

  // Handle attaching an existing run
  const handleAttachRun = useCallback(async (run: AgentRunSummary) => {
    if (!chatData?.chat?.id) return;

    setIsAttaching(true);
    setAgentError(null);

    try {
      await attachAgentRun(chatData.chat.id, run.run_id, run.task_id);
      setShowAttachModal(false);
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) {
        setAgentError({ message: e.message, recovery: e.recovery });
      } else {
        setAgentError({ message: e instanceof Error ? e.message : "Failed to attach run" });
      }
    } finally {
      setIsAttaching(false);
    }
  }, [chatData?.chat?.id, onRefreshChat]);

  // Handle sending message - route to agent or LLM based on mode
  const handleSendMessage = useCallback((payload: MessagePayload) => {
    if (chatMode === "agent") {
      if (agentBusy) {
        // Queue the message to auto-send when the agent finishes
        setQueuedMessage(payload);
      } else {
        handleSendAgentMessage(payload.content);
      }
    } else {
      onSendMessage(payload);
    }
  }, [chatMode, agentBusy, handleSendAgentMessage, onSendMessage]);

  // Handle opening the drawer for a specific operation or history view
  const handleOpenDrawer = useCallback((operation?: AsyncStatusUpdate) => {
    setSelectedOperation(operation ?? null);
    setDrawerOpen(true);
  }, []);

  // Handle inserting an async reference
  const handleInsertReference = useCallback((operation: AsyncStatusUpdate) => {
    onInsertAsyncReference?.(operation);
    setDrawerOpen(false);
  }, [onInsertAsyncReference]);

  // Handle refresh with Promise wrapper
  const handleRefreshOperation = useCallback(async (toolCallId: string) => {
    if (onRefreshAsyncOperation) {
      await onRefreshAsyncOperation(toolCallId);
    }
  }, [onRefreshAsyncOperation]);

  // Handle cancel with Promise wrapper
  const handleCancelOperation = useCallback(async (toolCallId: string) => {
    if (onCancelAsyncOperation) {
      await onCancelAsyncOperation(toolCallId);
    }
  }, [onCancelAsyncOperation]);

  // Handle load more history
  const handleLoadMoreHistory = useCallback(async () => {
    if (onFetchAsyncHistory) {
      await onFetchAsyncHistory();
    }
  }, [onFetchAsyncHistory]);
  // Compute visible messages based on the active branch
  // This filters the full message tree to only show the active path
  // NOTE: Must be called before any early returns to satisfy React's rules of hooks

  // Memoize allMessages to avoid creating new array references on each render.
  // CRITICAL: We must return EMPTY_MESSAGES for BOTH undefined AND empty arrays.
  // The nullish coalescing operator (??) only handles null/undefined, but [] is truthy.
  // Without this check, an empty messages array from the API would create a new
  // reference on every render, potentially causing infinite re-render loops (React #310).
  const allMessages = useMemo(() => {
    const messages = chatData?.messages;
    if (!messages || messages.length === 0) {
      return EMPTY_MESSAGES;
    }
    return messages;
  }, [chatData?.messages]);
  const activeLeafId = chatData?.chat?.active_leaf_message_id ?? null;

  // CRITICAL: Must use stable EMPTY_MESSAGES, not inline [] which creates new reference each time
  // NOTE: Do NOT include `chatData` in dependencies! The computation only uses `allMessages` and
  // `activeLeafId`. Including `chatData` causes unnecessary recalculations whenever ANY chatData
  // property changes (tool_call_records, chat name, etc.), potentially creating cascading re-renders.
  const visibleMessages = useMemo(() => {
    if (allMessages.length === 0) return EMPTY_MESSAGES;
    return computeVisibleMessages(allMessages, activeLeafId ?? undefined);
  }, [allMessages, activeLeafId]);

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
  const stableToolCallRecords = useMemo(() => {
    const records = chatData?.tool_call_records;
    if (!records || records.length === 0) {
      return EMPTY_TOOL_RECORDS;
    }
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

  if (!chatData) {
    return null;
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 min-w-0 bg-slate-950" data-testid="chat-view">
      <ErrorBoundary name="ChatHeader">
        <ChatHeader
          chat={chatData.chat}
          models={models}
          labels={labels}
          chatMode={chatMode}
          onUpdateChat={onUpdateChat}
          onToggleRead={onToggleRead}
          onToggleStar={onToggleStar}
          onToggleArchive={onToggleArchive}
          onDelete={onDeleteChat}
          onAssignLabel={onAssignLabel}
          onRemoveLabel={onRemoveLabel}
          isAgentActive={isAgentActive}
          agentStatus={agentStatus}
          agentMetrics={agentMetrics}
          agentError={agentError}
          onStopAgent={handleStopAgent}
          onBackToList={onBackToList}
          isMobile={isMobile}
          onOpenSidebar={onOpenSidebar}
          hasMessages={visibleMessages.length > 0}
          onModeChange={setChatMode}
          onOpenAgentSettings={_onOpenAgentSettings}
        />
      </ErrorBoundary>

      {/* Async Status Bar - compact view of active/recent operations (only in LLM mode) */}
      {!isAgentActive && (activeAsyncOperations.length > 0 || completedAsyncOperations.length > 0) && (
        <ErrorBoundary name="AsyncStatusBar">
          <AsyncStatusBar
            operations={asyncOperations}
            completedCount={completedAsyncOperations.length}
            onRefresh={handleRefreshOperation}
            onCancel={handleCancelOperation}
            onOpenDrawer={handleOpenDrawer}
            isCollapsed={statusBarCollapsed}
            onToggleCollapse={() => setStatusBarCollapsed(!statusBarCollapsed)}
          />
        </ErrorBoundary>
      )}

      {/* Async Operation Drawer - for details and history */}
      <AsyncOperationDrawer
        isOpen={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        operation={selectedOperation}
        completedOperations={completedAsyncOperations}
        onRefresh={handleRefreshOperation}
        onCancel={handleCancelOperation}
        onInsertReference={handleInsertReference}
        onLoadMoreHistory={handleLoadMoreHistory}
        hasMoreHistory={hasMoreAsyncHistory}
      />

      {/* Main content: Agent events or normal message list */}
      {isAgentActive ? (
        <ErrorBoundary name="AgentEventList">
          <AgentEventList
            events={agentEvents}
            autoScroll={!scrollToMessageId}
            viewMode={viewMode}
            initialMessages={visibleMessages}
            scrollToMessageId={scrollToMessageId}
            onScrollComplete={onScrollComplete}
          />
        </ErrorBoundary>
      ) : (
        <ErrorBoundary name="MessageList">
          <MessageList
            messages={visibleMessages}
            allMessages={allMessages}
            isGenerating={isGenerating}
            streamingContent={streamingContent}
            activeToolCalls={activeToolCalls}
            generatedImages={generatedImages}
            toolCallRecords={stableToolCallRecords}
            asyncOperations={asyncOperations}
            scrollToMessageId={scrollToMessageId}
            onScrollComplete={onScrollComplete}
            viewMode={viewMode}
            onRegenerateMessage={onRegenerateMessage}
            onSelectBranch={onSelectBranch}
            onForkConversation={onForkConversation}
            onEditMessage={onEditMessage}
            onOpenAsyncDrawer={handleOpenDrawer}
            isRegenerating={isRegenerating}
            isForking={isForking}
          />
        </ErrorBoundary>
      )}

      <div className="border-t border-white/10 bg-slate-950/50">
        {queuedMessage && (
          <div className="flex items-center justify-between px-4 py-2 bg-blue-500/10 border-b border-blue-500/20">
            <span className="text-xs text-blue-300">
              Message queued — will send when agent finishes
            </span>
            <button
              onClick={() => setQueuedMessage(null)}
              className="text-xs text-blue-400 hover:text-blue-200"
            >
              Cancel
            </button>
          </div>
        )}
        <ErrorBoundary name="MessageInput">
          <MessageInput
            onSend={handleSendMessage}
            isLoading={isGenerating || isStartingAgent}
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

      {/* Agent start modal */}
      <AgentStartModal
        isOpen={showAgentStartModal}
        onClose={() => {
          setShowAgentStartModal(false);
          setPendingAgentMessage("");
        }}
        onStart={handleStartAgent}
        defaultSettings={agentSettings}
        isLoading={isStartingAgent}
      />

      {/* Attach run modal */}
      <AttachRunModal
        isOpen={showAttachModal}
        onClose={() => setShowAttachModal(false)}
        onAttach={handleAttachRun}
        isLoading={isAttaching}
      />
    </div>
  );
}

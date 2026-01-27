import { useState, useCallback } from "react";
import { ModeSelector, type ChatMode } from "./ModeSelector";
import { AgentStartModal, type AgentStartConfig } from "./AgentStartModal";
import { AgentStatusIndicator } from "./agent/AgentStatusIndicator";
import { AgentEventList } from "./agent/AgentEventList";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import { useAgentWebSocket } from "../../hooks/useAgentWebSocket";
import {
  startAgentMode,
  sendAgentMessage,
  stopAgentMode,
  type Chat,
  type AgentChatConfig,
  type AgentModeStatus
} from "../../lib/api";

interface AgentModeWrapperProps {
  /** Current chat */
  chat: Chat | null;
  /** Children (MessageList, MessageInput) to render when in LLM mode */
  children: React.ReactNode;
  /** Called when the chat mode changes (to update chat state) */
  onModeChange?: (mode: ChatMode) => void;
  /** Callback when chat needs refresh after agent mode changes */
  onRefreshChat?: () => void;
}

/**
 * Wrapper component that handles agent mode state and rendering.
 * Shows the normal chat UI when in LLM mode, and the agent event stream
 * when in agent mode.
 */
export function AgentModeWrapper({
  chat,
  children,
  onModeChange,
  onRefreshChat
}: AgentModeWrapperProps) {
  const { settings } = useAgentSettings();
  const [mode, setMode] = useState<ChatMode>(chat?.chat_mode || "llm");
  const [isStartingAgent, setIsStartingAgent] = useState(false);
  const [showStartModal, setShowStartModal] = useState(false);
  const [pendingMessage, setPendingMessage] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  // Determine if agent is currently active
  const isAgentActive = chat?.chat_mode === "agent" && !!chat?.agent_run_id;

  // Agent WebSocket for real-time events
  const {
    events,
    status,
    isConnected,
    error: wsError,
    refresh: refreshEvents
  } = useAgentWebSocket({
    chatId: chat?.id || null,
    runId: chat?.agent_run_id || null,
    enabled: isAgentActive,
    onStatusChange: (newStatus: AgentModeStatus) => {
      // Refresh chat when agent completes or fails
      if (["complete", "failed", "cancelled"].includes(newStatus.status || "")) {
        onRefreshChat?.();
      }
    }
  });

  // Handle mode change
  const handleModeChange = useCallback((newMode: ChatMode) => {
    setMode(newMode);
    onModeChange?.(newMode);
  }, [onModeChange]);

  // Handle starting agent mode
  const handleStartAgent = useCallback(async (config: AgentStartConfig) => {
    if (!chat?.id || !pendingMessage) return;

    setIsStartingAgent(true);
    setError(null);

    try {
      const agentConfig: AgentChatConfig = {
        message: pendingMessage,
        runner_type: config.runner_type,
        project_path: config.project_path,
        model: config.model || undefined,
        max_turns: config.max_turns || undefined
      };

      await startAgentMode(chat.id, agentConfig);
      setShowStartModal(false);
      setPendingMessage("");
      onRefreshChat?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to start agent");
    } finally {
      setIsStartingAgent(false);
    }
  }, [chat?.id, pendingMessage, onRefreshChat]);

  // Handle sending agent message
  const handleSendAgentMessage = useCallback(async (message: string) => {
    if (!chat?.id) return;

    // If not in agent mode yet, show the start modal
    if (!isAgentActive) {
      setPendingMessage(message);
      setShowStartModal(true);
      return;
    }

    // Continue existing run
    try {
      setError(null);
      await sendAgentMessage(chat.id, message);
      refreshEvents();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to send message");
    }
  }, [chat?.id, isAgentActive, refreshEvents]);

  // Handle stopping agent
  const handleStopAgent = useCallback(async () => {
    if (!chat?.id) return;

    try {
      await stopAgentMode(chat.id);
      onRefreshChat?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to stop agent");
    }
  }, [chat?.id, onRefreshChat]);

  // Render based on mode
  return (
    <div className="flex flex-col h-full">
      {/* Mode selector in header area */}
      {chat && (
        <div className="flex items-center gap-2 px-4 py-2 border-b border-zinc-800">
          <ModeSelector
            mode={mode}
            onModeChange={handleModeChange}
            disabled={isStartingAgent}
            isAgentActive={isAgentActive}
          />
          {error && (
            <span className="text-xs text-red-400">{error}</span>
          )}
          {wsError && isAgentActive && (
            <span className="text-xs text-yellow-400">Connection issue: {wsError}</span>
          )}
        </div>
      )}

      {/* Agent status bar when active */}
      {isAgentActive && status && (
        <AgentStatusIndicator
          status={status.status}
          phase={status.phase}
          progress={status.progress_percent}
          errorMsg={status.error_msg}
          onStop={handleStopAgent}
        />
      )}

      {/* Main content area */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {isAgentActive ? (
          // Agent mode: show event stream
          <AgentEventList events={events} autoScroll={true} />
        ) : (
          // LLM mode: show normal chat
          children
        )}
      </div>

      {/* Agent start modal */}
      <AgentStartModal
        isOpen={showStartModal}
        onClose={() => {
          setShowStartModal(false);
          setPendingMessage("");
        }}
        onStart={handleStartAgent}
        defaultSettings={settings}
        isLoading={isStartingAgent}
      />
    </div>
  );
}

export default AgentModeWrapper;

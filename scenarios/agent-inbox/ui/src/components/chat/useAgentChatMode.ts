/**
 * useAgentChatMode - Agent mode state and actions for ChatView.
 *
 * Extracted from ChatView.tsx. Manages agent start/stop, message sending in
 * agent mode, run attachment, queued messages, and WebSocket event handling.
 */
import { useState, useCallback, useEffect, useMemo } from "react";
import type { ChatMode } from "./ModeSelector";
import type { AgentMetric } from "./agent/AgentEventList";
import type { MessagePayload } from "./MessageInput";
import type { ChatWithMessages, AgentChatConfig, AgentRunSummary } from "../../lib/api";
import {
  startAgentMode,
  sendAgentMessage,
  stopAgentMode,
  attachAgentRun,
  AgentModeError,
} from "../../lib/api";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import { useAgentWebSocket } from "../../hooks/useAgentWebSocket";

export interface UseAgentChatModeOptions {
  chatData: ChatWithMessages | null;
  onRefreshChat?: () => void;
  onSendLlmMessage: (payload: MessagePayload) => void;
}

export function useAgentChatMode({
  chatData,
  onRefreshChat,
  onSendLlmMessage,
}: UseAgentChatModeOptions) {
  const { settings: agentSettings } = useAgentSettings();
  const [chatMode, setChatMode] = useState<ChatMode>(chatData?.chat.chat_mode || "llm");
  const [isStartingAgent, setIsStartingAgent] = useState(false);
  const [showAgentStartModal, setShowAgentStartModal] = useState(false);
  const [pendingAgentMessage, setPendingAgentMessage] = useState("");
  const [agentError, setAgentError] = useState<{ message: string; recovery?: string } | null>(null);
  const [queuedMessage, setQueuedMessage] = useState<MessagePayload | null>(null);
  const [showAttachModal, setShowAttachModal] = useState(false);
  const [isAttaching, setIsAttaching] = useState(false);

  // Sync chatMode with server state
  useEffect(() => {
    if (chatData?.chat.chat_mode) {
      setChatMode(chatData.chat.chat_mode as ChatMode);
    }
  }, [chatData?.chat.chat_mode]);

  const isAgentActive = chatData?.chat.chat_mode === "agent" && !!chatData.chat.agent_run_id;

  const handleAgentStatusChange = useCallback((newStatus: import("../../lib/api").AgentModeStatus) => {
    if (["complete", "failed", "cancelled"].includes(newStatus.status || "")) {
      onRefreshChat?.();
    }
  }, [onRefreshChat]);

  const {
    events: agentEvents,
    status: agentStatus,
    isConnected: _isAgentConnected,
    error: _agentWsError,
    refresh: refreshAgentEvents,
  } = useAgentWebSocket({
    chatId: chatData?.chat.id || null,
    runId: chatData?.chat.agent_run_id || null,
    enabled: isAgentActive,
    onStatusChange: handleAgentStatusChange,
  });

  const agentBusy = isAgentActive && !!agentStatus?.status
    && ["pending", "starting", "running"].includes(agentStatus.status);

  const agentMetrics = useMemo((): AgentMetric[] => {
    if (agentEvents.length === 0) return [];
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
      } catch { /* skip */ }
    }
    return metrics;
  }, [agentEvents]);

  const handleStartAgent = useCallback(async (config: import("./AgentStartModal").AgentStartConfig) => {
    if (!chatData?.chat.id || !pendingAgentMessage) return;
    setIsStartingAgent(true);
    setAgentError(null);
    try {
      const agentConfig: AgentChatConfig = {
        message: pendingAgentMessage,
        project_path: config.project_path,
      };
      await startAgentMode(chatData.chat.id, agentConfig);
      setShowAgentStartModal(false);
      setPendingAgentMessage("");
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) setAgentError({ message: e.message, recovery: e.recovery });
      else setAgentError({ message: e instanceof Error ? e.message : "Failed to start agent" });
    } finally {
      setIsStartingAgent(false);
    }
  }, [chatData?.chat.id, pendingAgentMessage, onRefreshChat]);

  const handleSendAgentMessage = useCallback(async (message: string, attachmentIds?: string[]) => {
    if (!chatData?.chat.id) return;
    if (!isAgentActive) {
      setPendingAgentMessage(message);
      setShowAgentStartModal(true);
      return;
    }
    try {
      setAgentError(null);
      await sendAgentMessage(chatData.chat.id, message, attachmentIds);
      void refreshAgentEvents();
    } catch (e) {
      if (e instanceof AgentModeError) setAgentError({ message: e.message, recovery: e.recovery });
      else setAgentError({ message: e instanceof Error ? e.message : "Failed to send message" });
    }
  }, [chatData?.chat.id, isAgentActive, refreshAgentEvents]);

  useEffect(() => {
    if (!agentBusy && queuedMessage) {
      const payload = queuedMessage;
      setQueuedMessage(null);
      void handleSendAgentMessage(payload.content, payload.attachmentIds);
    }
  }, [agentBusy, queuedMessage, handleSendAgentMessage]);

  const handleStopAgent = useCallback(async () => {
    if (!chatData?.chat.id) return;
    try {
      await stopAgentMode(chatData.chat.id);
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) setAgentError({ message: e.message, recovery: e.recovery });
      else setAgentError({ message: e instanceof Error ? e.message : "Failed to stop agent" });
    }
  }, [chatData?.chat.id, onRefreshChat]);

  const handleAttachRun = useCallback(async (run: AgentRunSummary) => {
    if (!chatData?.chat.id) return;
    setIsAttaching(true);
    setAgentError(null);
    try {
      await attachAgentRun(chatData.chat.id, run.run_id, run.task_id);
      setShowAttachModal(false);
      onRefreshChat?.();
    } catch (e) {
      if (e instanceof AgentModeError) setAgentError({ message: e.message, recovery: e.recovery });
      else setAgentError({ message: e instanceof Error ? e.message : "Failed to attach run" });
    } finally {
      setIsAttaching(false);
    }
  }, [chatData?.chat.id, onRefreshChat]);

  const handleSendMessage = useCallback((payload: MessagePayload) => {
    if (chatMode === "agent") {
      if (agentBusy) setQueuedMessage(payload);
      else void handleSendAgentMessage(payload.content, payload.attachmentIds);
    } else {
      onSendLlmMessage(payload);
    }
  }, [chatMode, agentBusy, handleSendAgentMessage, onSendLlmMessage]);

  return {
    chatMode,
    setChatMode,
    isAgentActive,
    agentBusy,
    agentEvents,
    agentStatus,
    agentMetrics,
    agentError,
    agentSettings,
    isStartingAgent,
    showAgentStartModal,
    setShowAgentStartModal,
    pendingAgentMessage,
    setPendingAgentMessage,
    queuedMessage,
    setQueuedMessage,
    showAttachModal,
    setShowAttachModal,
    isAttaching,
    handleStartAgent,
    handleSendAgentMessage,
    handleStopAgent,
    handleAttachRun,
    handleSendMessage,
  };
}

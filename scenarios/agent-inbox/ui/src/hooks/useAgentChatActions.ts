import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useToast } from "../components/ui/toast";
import {
  startAgentMode,
  attachAgentRun,
  deleteChat as deleteChatAPI,
  createChat as createChatAPI,
  AgentModeError,
} from "../lib/api";
import type { AgentRunSummary } from "../lib/api";
import type { AgentStartConfig } from "../components/chat/AgentStartModal";
import type { MessagePayload } from "../components/chat/MessageInput";
import { getDefaultModel } from "../components/settings/Settings";

export function useAgentChatActions({
  selectChat,
}: {
  selectChat: (chatId: string) => void;
}) {
  const queryClient = useQueryClient();
  const { addToast } = useToast();

  // Handle starting an agent chat with message and config from EmptyState
  const handleStartAgentChat = useCallback(
    async (payload: MessagePayload, config: AgentStartConfig) => {
      const hasContent = payload.content.trim();
      if (!hasContent) return;

      let chatId: string | undefined;
      try {
        // Create chat in agent mode with default model
        const defaultModel = getDefaultModel();
        const newChat = await createChatAPI({ model: defaultModel, chat_mode: "agent" });
        chatId = newChat.id;

        // Select the new chat
        selectChat(chatId);

        // Start agent mode with the first message
        await startAgentMode(chatId, {
          message: payload.content.trim(),
          runner_type: config.runner_type,
          project_path: config.project_path,
          model: config.model || undefined,
          max_turns: config.max_turns || undefined,
        });

        // Refresh chat data to get updated agent state
        void queryClient.invalidateQueries({ queryKey: ["chats"] });
        void queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      } catch (error) {
        console.error("Failed to create agent chat:", error);
        // Clean up the partially-created chat so the user isn't left with a broken empty chat
        if (chatId) {
          try { await deleteChatAPI(chatId); } catch { /* best effort */ }
          selectChat("");
          void queryClient.invalidateQueries({ queryKey: ["chats"] });
        }
        // Surface the error to the user
        const msg = error instanceof AgentModeError
          ? error.message
          : error instanceof Error ? error.message : "Failed to start agent chat";
        addToast(msg, "error", 8000);
      }
    },
    [selectChat, queryClient, addToast]
  );

  // Handle attaching an existing run from EmptyState (creates a chat first)
  const handleAttachRunFromEmpty = useCallback(
    async (run: AgentRunSummary) => {
      let chatId: string | undefined;
      try {
        const defaultModel = getDefaultModel();
        const newChat = await createChatAPI({ model: defaultModel, chat_mode: "agent" });
        chatId = newChat.id;
        selectChat(chatId);

        await attachAgentRun(chatId, run.run_id, run.task_id);

        void queryClient.invalidateQueries({ queryKey: ["chats"] });
        void queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      } catch (error) {
        console.error("Failed to attach run:", error);
        if (chatId) {
          try { await deleteChatAPI(chatId); } catch { /* best effort */ }
          selectChat("");
          void queryClient.invalidateQueries({ queryKey: ["chats"] });
        }
        const msg = error instanceof AgentModeError
          ? error.message
          : error instanceof Error ? error.message : "Failed to attach run";
        addToast(msg, "error", 8000);
      }
    },
    [selectChat, queryClient, addToast]
  );

  return {
    handleStartAgentChat,
    handleAttachRunFromEmpty,
  };
}

/**
 * useChatActions - Message operations for the useChats hook.
 *
 * Handles sending messages, regeneration, editing, branching, forking,
 * and tool approval/rejection. All operations coordinate with the
 * CacheUpdateManager for batched invalidations during streaming.
 */
import { useState, useCallback } from "react";
import {
  addMessage,
  createChat,
  regenerateMessage as apiRegenerateMessage,
  editMessage as apiEditMessage,
  type StreamingEvent,
  type Message,
  type Chat,
} from "../lib/api";
import type { CompletionState, CompletionActions } from "./useCompletionTypes";
import type { CacheUpdateManager } from "../lib/cache";
import { getDefaultModel } from "../components/settings/Settings";
import type { MessagePayload } from "../components/chat/MessageInput";

export interface UseChatActionsOptions {
  selectedChatId: string | null;
  setSelectedChatId: (id: string | null) => void;
  onChatChange?: (chatId: string | null) => void;
  completion: CompletionState & CompletionActions;
  cacheManager: CacheUpdateManager;
  chats: Chat[];
  toggleReadMutate: (params: { chatId: string; value?: boolean }) => void;
  selectBranchMutate: (params: { chatId: string; messageId: string }) => void;
  forkChatMutate: (params: { chatId: string; messageId: string }) => void;
}

export function useChatActions({
  selectedChatId,
  setSelectedChatId,
  onChatChange,
  completion,
  cacheManager,
  chats,
  toggleReadMutate,
  selectBranchMutate,
  forkChatMutate,
}: UseChatActionsOptions) {
  // State for regeneration streaming
  const [isRegenerating, setIsRegenerating] = useState(false);
  const [regeneratingContent, setRegeneratingContent] = useState("");

  // State for message editing
  const [editingMessage, setEditingMessage] = useState<Message | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  // Regenerate a message (ChatGPT-style branching)
  const regenerateMessage = useCallback(
    async (chatId: string, messageId: string) => {
      if (isRegenerating || completion.isGenerating) return;
      setIsRegenerating(true);
      setRegeneratingContent("");
      const abortController = new AbortController();
      cacheManager.startStreaming(chatId);
      try {
        await apiRegenerateMessage(chatId, messageId, {
          stream: true,
          signal: abortController.signal,
          onChunk: (content: string) => { setRegeneratingContent((prev) => prev + content); },
          onEvent: (event: StreamingEvent) => { if (event.type === "error") console.error("Regeneration error:", event.error); },
        });
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") console.log("Regeneration aborted");
        else console.error("Regeneration failed:", error);
      } finally {
        setIsRegenerating(false);
        setRegeneratingContent("");
        await cacheManager.endStreaming(chatId);
      }
    },
    [isRegenerating, completion.isGenerating, cacheManager]
  );

  // Select a different message branch
  const selectBranch = useCallback(
    (messageId: string) => {
      if (!selectedChatId) return;
      selectBranchMutate({ chatId: selectedChatId, messageId });
    },
    [selectedChatId, selectBranchMutate]
  );

  // Edit a message (creates sibling and triggers new AI response)
  const editMessageAndComplete = useCallback(
    async (messageId: string, payload: MessagePayload) => {
      if (!selectedChatId || isEditing || completion.isGenerating) return;
      setIsEditing(true);
      setEditingMessage(null);
      cacheManager.startStreaming(selectedChatId);
      try {
        await apiEditMessage(selectedChatId, messageId, {
          content: payload.content.trim(),
          attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
          web_search: payload.webSearchEnabled ? true : undefined,
        });
        await cacheManager.invalidateChat(selectedChatId, ["chats"]);
        await completion.runCompletion(selectedChatId);
      } catch (error) {
        console.error("Edit message failed:", error);
      } finally {
        setIsEditing(false);
        await cacheManager.endStreaming(selectedChatId);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, isEditing, completion.runCompletion, cacheManager]
  );

  const cancelEdit = useCallback(() => { setEditingMessage(null); }, []);

  // Send message and run completion
  const sendMessageAndComplete = useCallback(
    async (chatId: string, payload: MessagePayload) => {
      const newMessage = await addMessage(chatId, {
        role: "user",
        content: payload.content.trim(),
        attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
        web_search: payload.webSearchEnabled ? true : undefined,
        skill_ids: payload.skillIds && payload.skillIds.length > 0 ? payload.skillIds : undefined,
      });
      cacheManager.optimisticAddMessage(chatId, newMessage, newMessage.id);
      await new Promise<void>((resolve) => { setTimeout(() => { resolve(); }, 0); });
      cacheManager.startStreaming(chatId);
      try {
        await completion.runCompletion(chatId, { skills: payload.skills });
      } catch (error) {
        console.error("Chat completion failed:", error);
      } finally {
        await cacheManager.endStreaming(chatId);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [completion.runCompletion]
  );

  // Create a new chat and immediately send a message
  const createChatWithMessage = useCallback(
    async (payload: MessagePayload) => {
      const hasContent = payload.content.trim() || payload.attachmentIds.length > 0;
      if (!hasContent || completion.isGenerating) return;
      try {
        const defaultModel = getDefaultModel();
        const newChat = await createChat({ model: defaultModel });
        setSelectedChatId(newChat.id);
        onChatChange?.(newChat.id);
        await sendMessageAndComplete(newChat.id, payload);
      } catch (error) {
        console.error("Failed to create chat with message:", error);
      }
    },
    [completion.isGenerating, sendMessageAndComplete, onChatChange, setSelectedChatId]
  );

  // Send message to existing chat
  const sendMessage = useCallback(
    async (payload: MessagePayload) => {
      const hasContent = payload.content.trim() || payload.attachmentIds.length > 0;
      if (!selectedChatId || !hasContent || completion.isGenerating) return;
      await sendMessageAndComplete(selectedChatId, payload);
    },
    [selectedChatId, completion.isGenerating, sendMessageAndComplete]
  );

  // Select chat and mark as read
  const selectChat = useCallback(
    (chatId: string) => {
      const newId = chatId || null;
      setSelectedChatId(newId);
      onChatChange?.(newId);
      const chat = chats.find((c) => c.id === chatId);
      if (chat && !chat.is_read) toggleReadMutate({ chatId, value: true });
    },
    [chats, toggleReadMutate, onChatChange, setSelectedChatId]
  );

  // Fork conversation from a specific message
  const forkConversation = useCallback(
    (messageId: string) => {
      if (!selectedChatId) return;
      forkChatMutate({ chatId: selectedChatId, messageId });
    },
    [selectedChatId, forkChatMutate]
  );

  // Tool approval actions
  const approveTool = useCallback(
    async (toolCallId: string) => {
      if (!selectedChatId) return;
      const result = await completion.approveTool(selectedChatId, toolCallId);
      await cacheManager.invalidateChat(selectedChatId, ["chats"]);
      if (result.auto_continued) { /* no extra invalidation needed */ }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, completion.approveTool, cacheManager]
  );

  const rejectTool = useCallback(
    async (toolCallId: string, reason?: string) => {
      if (!selectedChatId) return;
      await completion.rejectTool(selectedChatId, toolCallId, reason);
      await cacheManager.invalidateChat(selectedChatId);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, completion.rejectTool, cacheManager]
  );

  return {
    // Regeneration
    isRegenerating,
    regeneratingContent,
    regenerateMessage,
    // Branching
    selectBranch,
    // Editing
    editingMessage,
    isEditing,
    setEditingMessage,
    editMessageAndComplete,
    cancelEdit,
    // Sending
    sendMessage,
    createChatWithMessage,
    // Selection
    selectChat,
    // Fork
    forkConversation,
    // Tool approval
    approveTool,
    rejectTool,
  };
}

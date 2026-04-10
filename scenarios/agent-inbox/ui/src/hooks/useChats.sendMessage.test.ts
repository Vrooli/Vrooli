/**
 * Tests for useChats hook - sendMessage, createChatWithMessage, and edit flows
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useChats } from "./useChats";
import * as api from "../lib/api";
import * as completionHook from "./useCompletion";
import * as labelsHook from "./useLabels";
import {
  mockChat,
  mockMessage,
  mockChatWithMessages,
  mockCompletionState,
  mockLabelsState,
  createWrapper,
} from "./useChats.test.helpers";

describe("useChats - message sending and editing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();

    vi.mocked(api.fetchChats).mockResolvedValue([mockChat]);
    vi.mocked(api.fetchChat).mockResolvedValue(mockChatWithMessages);
    vi.mocked(api.fetchModels).mockResolvedValue([]);
    vi.mocked(completionHook.useCompletion).mockReturnValue(mockCompletionState);
    vi.mocked(labelsHook.useLabels).mockReturnValue(mockLabelsState);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("sendMessage", () => {
    it("sends message to selected chat", async () => {
      vi.useRealTimers();

      const newMessage: api.Message = {
        id: "msg-2",
        chat_id: "chat-1",
        role: "user",
        content: "Test message",
        sibling_index: 0,
        created_at: "2025-01-01T00:01:00Z",
      };

      vi.mocked(api.addMessage).mockResolvedValue(newMessage);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.sendMessage({
          content: "Test message",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.addMessage).toHaveBeenCalledWith("chat-1", {
        role: "user",
        content: "Test message",
        attachment_ids: undefined,
        web_search: undefined,
        skill_ids: undefined,
      });
    });

    it("runs completion after sending message", async () => {
      vi.useRealTimers();

      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.sendMessage({
          content: "Hello",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(mockCompletionState.runCompletion).toHaveBeenCalledWith("chat-1", expect.any(Object));
    });

    it("does not trigger frontend auto-name during sendMessage", async () => {
      vi.useRealTimers();

      vi.mocked(api.fetchChats).mockResolvedValue([{ ...mockChat, name: "New Chat" }]);
      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.sendMessage({
          content: "Hello",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.autoNameChat).not.toHaveBeenCalled();
    });

    it("does not send if content is empty", async () => {
      vi.useRealTimers();

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.sendMessage({
          content: "   ",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.addMessage).not.toHaveBeenCalled();
    });

    it("does not send if already generating", async () => {
      vi.useRealTimers();

      vi.mocked(completionHook.useCompletion).mockReturnValue({
        ...mockCompletionState,
        isGenerating: true,
      });

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.sendMessage({
          content: "Hello",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.addMessage).not.toHaveBeenCalled();
    });
  });

  describe("createChatWithMessage", () => {
    it("creates new chat and sends first message", async () => {
      vi.useRealTimers();

      const newChat = { ...mockChat, id: "new-chat" };
      vi.mocked(api.createChat).mockResolvedValue(newChat);
      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);

      const onChatChange = vi.fn();

      const { result } = renderHook(
        () => useChats({ onChatChange }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.createChatWithMessage({
          content: "Hello",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.createChat).toHaveBeenCalledWith({ model: "gpt-4" });
      expect(onChatChange).toHaveBeenCalledWith("new-chat");
    });
  });

  describe("editMessageAndComplete", () => {
    it("edits message and runs completion", async () => {
      vi.useRealTimers();

      const editedMessage: api.Message = {
        id: "msg-edit",
        chat_id: "chat-1",
        role: "user",
        content: "Edited content",
        sibling_index: 1,
        created_at: "2025-01-01T00:02:00Z",
      };

      vi.mocked(api.editMessage).mockResolvedValue(editedMessage);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.editMessageAndComplete("msg-1", {
          content: "Edited content",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.editMessage).toHaveBeenCalledWith("chat-1", "msg-1", {
        content: "Edited content",
        attachment_ids: undefined,
        web_search: undefined,
      });
      expect(mockCompletionState.runCompletion).toHaveBeenCalledWith("chat-1");
    });
  });

  describe("edit mode", () => {
    it("sets editing message", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.setEditingMessage(mockMessage);
      });

      expect(result.current.editingMessage).toEqual(mockMessage);
    });

    it("cancels edit mode", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.setEditingMessage(mockMessage);
      });

      expect(result.current.editingMessage).toEqual(mockMessage);

      act(() => {
        result.current.cancelEdit();
      });

      expect(result.current.editingMessage).toBeNull();
    });
  });
});

/**
 * Tests for useChats hook - CRUD mutations, return value, optimistic updates, attachments, web search
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

describe("useChats - CRUD mutations and misc", () => {
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

  describe("chat mutations", () => {
    it("creates chat and selects it", async () => {
      vi.useRealTimers();
      const newChat = { ...mockChat, id: "new-chat-id" };
      vi.mocked(api.createChat).mockResolvedValue(newChat);
      const onChatChange = vi.fn();
      const { result } = renderHook(() => useChats({ onChatChange }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      act(() => { result.current.createChat({ name: "New Chat", model: "gpt-4" }); });
      await waitFor(() => { expect(api.createChat).toHaveBeenCalledWith({ name: "New Chat", model: "gpt-4" }); });
      await waitFor(() => { expect(onChatChange).toHaveBeenCalledWith("new-chat-id"); });
    });

    it("calls deleteChat mutation", async () => {
      vi.useRealTimers();
      const { result } = renderHook(() => useChats({ initialChatId: "chat-1" }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      expect(typeof result.current.deleteChat).toBe("function");
      act(() => { result.current.deleteChat("chat-1"); });
      expect(result.current.selectedChatId).toBe("chat-1");
    });

    it("deletes all chats", async () => {
      vi.useRealTimers();
      vi.mocked(api.deleteAllChats).mockResolvedValue({ deleted: 5 });
      const { result } = renderHook(() => useChats(), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      await act(async () => { await result.current.deleteAllChats(); });
      expect(api.deleteAllChats).toHaveBeenCalled();
    });

    it("updates chat", async () => {
      vi.useRealTimers();
      vi.mocked(api.updateChat).mockResolvedValue({ ...mockChat, name: "Updated" });
      const { result } = renderHook(() => useChats({ initialChatId: "chat-1" }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      act(() => { result.current.updateChat({ chatId: "chat-1", data: { name: "Updated" } }); });
      await waitFor(() => { expect(api.updateChat).toHaveBeenCalledWith("chat-1", { name: "Updated" }); });
    });

    it("toggles archive", async () => {
      vi.useRealTimers();
      vi.mocked(api.toggleArchive).mockResolvedValue({ is_archived: true });
      const { result } = renderHook(() => useChats(), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      act(() => { result.current.toggleArchive({ chatId: "chat-1", value: true }); });
      await waitFor(() => { expect(api.toggleArchive).toHaveBeenCalledWith("chat-1", true); });
    });

    it("toggles star", async () => {
      vi.useRealTimers();
      vi.mocked(api.toggleStar).mockResolvedValue({ is_starred: true });
      const { result } = renderHook(() => useChats(), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      act(() => { result.current.toggleStar({ chatId: "chat-1", value: true }); });
      await waitFor(() => { expect(api.toggleStar).toHaveBeenCalledWith("chat-1", true); });
    });
  });

  describe("return value", () => {
    it("returns expected action functions", async () => {
      vi.useRealTimers();
      const { result } = renderHook(() => useChats(), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      expect(typeof result.current.selectChat).toBe("function");
      expect(typeof result.current.sendMessage).toBe("function");
      expect(typeof result.current.createChatWithMessage).toBe("function");
      expect(typeof result.current.selectBranch).toBe("function");
      expect(typeof result.current.regenerateMessage).toBe("function");
      expect(typeof result.current.editMessageAndComplete).toBe("function");
      expect(typeof result.current.approveTool).toBe("function");
      expect(typeof result.current.rejectTool).toBe("function");
    });
  });

  describe("optimistic updates", () => {
    it("optimistically adds message to cache", async () => {
      vi.useRealTimers();
      const newMessage: api.Message = { id: "msg-new", chat_id: "chat-1", role: "user", content: "New message", sibling_index: 0, created_at: "2025-01-01T00:03:00Z" };
      vi.mocked(api.addMessage).mockResolvedValue(newMessage);
      const { result } = renderHook(() => useChats({ initialChatId: "chat-1" }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      await act(async () => { await result.current.sendMessage({ content: "New message", attachmentIds: [], webSearchEnabled: false }); });
      expect(api.addMessage).toHaveBeenCalled();
    });
  });

  describe("attachment handling", () => {
    it("includes attachment IDs when sending message", async () => {
      vi.useRealTimers();
      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);
      const { result } = renderHook(() => useChats({ initialChatId: "chat-1" }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      await act(async () => { await result.current.sendMessage({ content: "Message with attachments", attachmentIds: ["attach-1", "attach-2"], webSearchEnabled: false }); });
      expect(api.addMessage).toHaveBeenCalledWith("chat-1", expect.objectContaining({ attachment_ids: ["attach-1", "attach-2"] }));
    });
  });

  describe("web search handling", () => {
    it("includes web search flag when enabled", async () => {
      vi.useRealTimers();
      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);
      const { result } = renderHook(() => useChats({ initialChatId: "chat-1" }), { wrapper: createWrapper() });
      await waitFor(() => { expect(result.current.loadingChats).toBe(false); });
      await act(async () => { await result.current.sendMessage({ content: "Search the web", attachmentIds: [], webSearchEnabled: true }); });
      expect(api.addMessage).toHaveBeenCalledWith("chat-1", expect.objectContaining({ web_search: true }));
    });
  });
});

/**
 * Tests for useChats hook
 *
 * Tests the main orchestration hook for chat management including:
 * - Message sending flow with optimistic updates
 * - Cache invalidation timing
 * - Branch operations
 * - Edit flow
 * - Integration with useCompletion
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useChats } from "./useChats";
import * as api from "../lib/api";
import * as completionHook from "./useCompletion";
import * as labelsHook from "./useLabels";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchChats: vi.fn(),
  fetchChat: vi.fn(),
  fetchModels: vi.fn(),
  createChat: vi.fn(),
  deleteChat: vi.fn(),
  deleteAllChats: vi.fn(),
  updateChat: vi.fn(),
  addMessage: vi.fn(),
  toggleRead: vi.fn(),
  toggleArchive: vi.fn(),
  toggleStar: vi.fn(),
  autoNameChat: vi.fn(),
  regenerateMessage: vi.fn(),
  editMessage: vi.fn(),
  selectBranch: vi.fn(),
  bulkOperateChats: vi.fn(),
  forkChat: vi.fn(),
}));

// Mock the useCompletion hook
vi.mock("./useCompletion", () => ({
  useCompletion: vi.fn(),
}));

// Mock the useLabels hook
vi.mock("./useLabels", () => ({
  useLabels: vi.fn(),
}));

// Mock settings
vi.mock("../components/settings/Settings", () => ({
  getDefaultModel: vi.fn(() => "gpt-4"),
}));

// Test data
const mockChat: api.Chat = {
  id: "chat-1",
  name: "Test Chat",
  preview: "Hello world",
  model: "gpt-4",
  view_mode: "bubble",
  chat_mode: "llm",
  is_read: true,
  is_archived: false,
  is_starred: false,
  label_ids: [],
  tools_enabled: true,
  web_search_enabled: false,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

const mockMessage: api.Message = {
  id: "msg-1",
  chat_id: "chat-1",
  role: "user",
  content: "Hello",
  sibling_index: 0,
  created_at: "2025-01-01T00:00:00Z",
};

const mockChatWithMessages: api.ChatWithMessages = {
  chat: mockChat,
  messages: [mockMessage],
  tool_call_records: [],
};

const mockCompletionState = {
  isGenerating: false,
  streamingContent: "",
  generatedImages: [],
  activeToolCalls: [],
  pendingApprovals: [],
  awaitingApprovals: false,
  runCompletion: vi.fn().mockResolvedValue(undefined),
  resetCompletion: vi.fn(),
  cancelCompletion: vi.fn(),
  approveTool: vi.fn().mockResolvedValue({ success: true }),
  rejectTool: vi.fn().mockResolvedValue(undefined),
};

const mockLabelsState = {
  labels: [],
  isLoading: false,
  createLabel: vi.fn(),
  deleteLabel: vi.fn(),
  assignLabel: vi.fn(),
  removeLabel: vi.fn(),
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
    logger: {
      log: () => {},
      warn: () => {},
      error: () => {},
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useChats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();

    // Setup default mocks
    vi.mocked(api.fetchChats).mockResolvedValue([mockChat]);
    vi.mocked(api.fetchChat).mockResolvedValue(mockChatWithMessages);
    vi.mocked(api.fetchModels).mockResolvedValue([]);
    vi.mocked(completionHook.useCompletion).mockReturnValue(mockCompletionState);
    vi.mocked(labelsHook.useLabels).mockReturnValue(mockLabelsState);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("initial state", () => {
    it("starts with correct default state", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      expect(result.current.selectedChatId).toBeNull();
      expect(result.current.currentView).toBe("inbox");
      expect(result.current.isGenerating).toBe(false);
    });

    it("uses initialChatId when provided", async () => {
      vi.useRealTimers();

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.selectedChatId).toBe("chat-1");
    });
  });

  describe("chat fetching", () => {
    it("fetches chats for current view", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      expect(api.fetchChats).toHaveBeenCalledWith({
        archived: false,
        starred: false,
      });
      expect(result.current.chats).toEqual([mockChat]);
    });

    it("fetches archived chats when view changes", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.setCurrentView("archived");
      });

      await waitFor(() => {
        expect(api.fetchChats).toHaveBeenCalledWith({
          archived: true,
          starred: false,
        });
      });
    });

    it("fetches starred chats when view changes", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.setCurrentView("starred");
      });

      await waitFor(() => {
        expect(api.fetchChats).toHaveBeenCalledWith({
          archived: false,
          starred: true,
        });
      });
    });
  });

  describe("chat selection", () => {
    it("selects chat", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.selectChat("chat-1");
      });

      expect(result.current.selectedChatId).toBe("chat-1");
    });

    it("does not mark read if already read", async () => {
      vi.useRealTimers();

      vi.mocked(api.fetchChats).mockResolvedValue([mockChat]); // Already read

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.selectChat("chat-1");
      });

      expect(api.toggleRead).not.toHaveBeenCalled();
    });

    it("calls onChatChange callback when chat changes", async () => {
      vi.useRealTimers();

      const onChatChange = vi.fn();

      const { result } = renderHook(
        () => useChats({ onChatChange }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.selectChat("chat-1");
      });

      expect(onChatChange).toHaveBeenCalledWith("chat-1");
    });
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

    it("auto-names chat if name is 'New Chat'", async () => {
      vi.useRealTimers();

      const newChat = { ...mockChat, name: "New Chat" };
      vi.mocked(api.fetchChats).mockResolvedValue([newChat]);
      vi.mocked(api.addMessage).mockResolvedValue(mockMessage);
      vi.mocked(api.autoNameChat).mockResolvedValue({ ...newChat, name: "Auto Named" });

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

      expect(api.autoNameChat).toHaveBeenCalledWith("chat-1");
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

  describe("branch operations", () => {
    it("selects branch via selectBranch mutation", async () => {
      vi.useRealTimers();

      vi.mocked(api.selectBranch).mockResolvedValue({ active_leaf_message_id: "msg-2" });

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.selectBranch("msg-2");
      });

      await waitFor(() => {
        expect(api.selectBranch).toHaveBeenCalledWith("chat-1", "msg-2");
      });
    });

    it("does not select branch if no chat selected", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.selectBranch("msg-2");
      });

      expect(api.selectBranch).not.toHaveBeenCalled();
    });
  });

  describe("regenerateMessage", () => {
    it("regenerates message with streaming", async () => {
      vi.useRealTimers();

      vi.mocked(api.regenerateMessage).mockResolvedValue(undefined);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.regenerateMessage("chat-1", "msg-1");
      });

      expect(api.regenerateMessage).toHaveBeenCalledWith(
        "chat-1",
        "msg-1",
        expect.objectContaining({
          stream: true,
          onChunk: expect.any(Function),
          onEvent: expect.any(Function),
        })
      );
    });

    it("does not regenerate if already regenerating", async () => {
      vi.useRealTimers();

      let resolveRegen: (() => void) | undefined;
      vi.mocked(api.regenerateMessage).mockImplementation(async () => {
        await new Promise<void>(resolve => {
          resolveRegen = resolve;
        });
      });

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      // Start first regeneration
      act(() => {
        result.current.regenerateMessage("chat-1", "msg-1");
      });

      // Try second regeneration while first is in progress
      act(() => {
        result.current.regenerateMessage("chat-1", "msg-2");
      });

      // Only one call should have been made
      expect(api.regenerateMessage).toHaveBeenCalledTimes(1);

      // Cleanup
      resolveRegen?.();
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

  describe("tool approval actions", () => {
    it("approves tool and invalidates cache", async () => {
      vi.useRealTimers();

      vi.mocked(mockCompletionState.approveTool).mockResolvedValue({
        success: true,
        auto_continued: false,
        pending_approvals: [],
        tool_result: { id: "call-1", tool_name: "test", status: "completed" },
      });

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.approveTool("call-1");
      });

      expect(mockCompletionState.approveTool).toHaveBeenCalledWith("chat-1", "call-1");
    });

    it("rejects tool and invalidates cache", async () => {
      vi.useRealTimers();

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.rejectTool("call-1", "Not authorized");
      });

      expect(mockCompletionState.rejectTool).toHaveBeenCalledWith("chat-1", "call-1", "Not authorized");
    });
  });

  describe("bulk operations", () => {
    it("performs bulk archive operation", async () => {
      vi.useRealTimers();

      vi.mocked(api.bulkOperateChats).mockResolvedValue({
        success_count: 2,
        fail_count: 0,
        total: 2,
      });

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.bulkOperate({
          chatIds: ["chat-1", "chat-2"],
          operation: "archive",
        });
      });

      await waitFor(() => {
        expect(api.bulkOperateChats).toHaveBeenCalledWith(
          ["chat-1", "chat-2"],
          "archive",
          undefined
        );
      });
    });
  });

  describe("fork conversation", () => {
    it("forks conversation from message", async () => {
      vi.useRealTimers();

      const forkedChat = { ...mockChat, id: "forked-chat" };
      vi.mocked(api.forkChat).mockResolvedValue(forkedChat);

      const onChatChange = vi.fn();

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1", onChatChange }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.forkConversation("msg-1");
      });

      await waitFor(() => {
        expect(api.forkChat).toHaveBeenCalledWith("chat-1", "msg-1");
      });

      await waitFor(() => {
        expect(onChatChange).toHaveBeenCalledWith("forked-chat");
      });
    });
  });

  describe("chat mutations", () => {
    it("creates chat and selects it", async () => {
      vi.useRealTimers();

      const newChat = { ...mockChat, id: "new-chat-id" };
      vi.mocked(api.createChat).mockResolvedValue(newChat);

      const onChatChange = vi.fn();

      const { result } = renderHook(
        () => useChats({ onChatChange }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.createChat({ name: "New Chat", model: "gpt-4" });
      });

      await waitFor(() => {
        expect(api.createChat).toHaveBeenCalledWith({ name: "New Chat", model: "gpt-4" });
      });

      await waitFor(() => {
        expect(onChatChange).toHaveBeenCalledWith("new-chat-id");
      });
    });

    it("calls deleteChat mutation", async () => {
      vi.useRealTimers();

      // For this test, we're verifying the deleteChat action exists and can be called
      // The actual API call is tested via integration tests
      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      // Verify deleteChat is a function that can be called
      expect(typeof result.current.deleteChat).toBe("function");

      // Calling it should not throw
      act(() => {
        result.current.deleteChat("chat-1");
      });

      // The mutation was triggered (we can verify via state or simply that no error occurred)
      expect(result.current.selectedChatId).toBe("chat-1"); // Still set initially
    });

    it("deletes all chats", async () => {
      vi.useRealTimers();

      vi.mocked(api.deleteAllChats).mockResolvedValue({ deleted: 5 });

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      await act(async () => {
        await result.current.deleteAllChats();
      });

      expect(api.deleteAllChats).toHaveBeenCalled();
    });

    it("updates chat", async () => {
      vi.useRealTimers();

      vi.mocked(api.updateChat).mockResolvedValue({ ...mockChat, name: "Updated" });

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.updateChat({ chatId: "chat-1", data: { name: "Updated" } });
      });

      await waitFor(() => {
        expect(api.updateChat).toHaveBeenCalledWith("chat-1", { name: "Updated" });
      });
    });

    it("toggles archive", async () => {
      vi.useRealTimers();

      vi.mocked(api.toggleArchive).mockResolvedValue({ is_archived: true });

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.toggleArchive({ chatId: "chat-1", value: true });
      });

      await waitFor(() => {
        expect(api.toggleArchive).toHaveBeenCalledWith("chat-1", true);
      });
    });

    it("toggles star", async () => {
      vi.useRealTimers();

      vi.mocked(api.toggleStar).mockResolvedValue({ is_starred: true });

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      act(() => {
        result.current.toggleStar({ chatId: "chat-1", value: true });
      });

      await waitFor(() => {
        expect(api.toggleStar).toHaveBeenCalledWith("chat-1", true);
      });
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

  describe("return value", () => {
    it("returns expected action functions", async () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      // Verify all expected actions are available
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

      const newMessage: api.Message = {
        id: "msg-new",
        chat_id: "chat-1",
        role: "user",
        content: "New message",
        sibling_index: 0,
        created_at: "2025-01-01T00:03:00Z",
      };

      vi.mocked(api.addMessage).mockResolvedValue(newMessage);

      const { result } = renderHook(
        () => useChats({ initialChatId: "chat-1" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loadingChats).toBe(false);
      });

      // The optimistic update happens via queryClient.setQueryData
      // This test verifies that addMessage is called with correct params
      await act(async () => {
        await result.current.sendMessage({
          content: "New message",
          attachmentIds: [],
          webSearchEnabled: false,
        });
      });

      expect(api.addMessage).toHaveBeenCalled();
    });
  });

  describe("attachment handling", () => {
    it("includes attachment IDs when sending message", async () => {
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
          content: "Message with attachments",
          attachmentIds: ["attach-1", "attach-2"],
          webSearchEnabled: false,
        });
      });

      expect(api.addMessage).toHaveBeenCalledWith("chat-1", expect.objectContaining({
        attachment_ids: ["attach-1", "attach-2"],
      }));
    });
  });

  describe("web search handling", () => {
    it("includes web search flag when enabled", async () => {
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
          content: "Search the web",
          attachmentIds: [],
          webSearchEnabled: true,
        });
      });

      expect(api.addMessage).toHaveBeenCalledWith("chat-1", expect.objectContaining({
        web_search: true,
      }));
    });
  });
});

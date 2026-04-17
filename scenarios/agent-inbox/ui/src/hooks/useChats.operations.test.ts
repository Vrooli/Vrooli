/**
 * Tests for useChats hook - Branch ops, regenerate, tool approval, bulk ops, fork
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

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
vi.mock("./useCompletion", () => ({ useCompletion: vi.fn() }));
vi.mock("./useLabels", () => ({ useLabels: vi.fn() }));
vi.mock("../components/settings/Settings", () => ({
  getDefaultModel: vi.fn(() => "gpt-4"),
}));

import { useChats } from "./useChats";
import * as api from "../lib/api";
import * as completionHook from "./useCompletion";
import * as labelsHook from "./useLabels";
import {
  mockChat,
  mockMessage as _mockMessage,
  mockChatWithMessages,
  mockCompletionState,
  mockLabelsState,
  createWrapper,
} from "./useChats.test.helpers";

describe("useChats - operations", () => {
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

      act(() => {
        void result.current.regenerateMessage("chat-1", "msg-1");
      });

      act(() => {
        void result.current.regenerateMessage("chat-1", "msg-2");
      });

      expect(api.regenerateMessage).toHaveBeenCalledTimes(1);

      resolveRegen?.();
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
});

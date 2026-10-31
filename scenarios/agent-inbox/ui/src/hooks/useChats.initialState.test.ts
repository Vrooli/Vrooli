/**
 * Tests for useChats hook - Initial state, chat fetching, and chat selection
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
  mockChatWithMessages,
  mockCompletionState,
  mockLabelsState,
  createWrapper,
} from "./useChats.test.helpers";

describe("useChats - initial state and fetching", () => {
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

  describe("initial state", () => {
    it("starts with correct default state", () => {
      vi.useRealTimers();

      const { result } = renderHook(() => useChats(), {
        wrapper: createWrapper(),
      });

      expect(result.current.selectedChatId).toBeNull();
      expect(result.current.currentView).toBe("inbox");
      expect(result.current.isGenerating).toBe(false);
    });

    it("uses initialChatId when provided", () => {
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
});

/**
 * Tests for useCompletion hook - Approve/reject, reset, cleanup, error handling, stability
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useCompletion } from "./useCompletion";
import * as api from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  completeChat: vi.fn(),
  approveToolCall: vi.fn(),
  rejectToolCall: vi.fn(),
}));

describe("useCompletion - approval, reset, and errors", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("approveTool", () => {
    it("calls API and updates state on approval", async () => {
      vi.useRealTimers();
      const mockResult: api.ApprovalResult = {
        success: true,
        tool_result: { id: "call_123", tool_name: "run-agent", status: "completed", result: '{"success": true}' },
        pending_approvals: [],
        auto_continued: false,
      };
      vi.mocked(api.approveToolCall).mockResolvedValue(mockResult);
      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        options?.onEvent?.({ type: "tool_pending_approval", tool_call_id: "call_123", tool_name: "run-agent", arguments: "{}" });
        options?.onEvent?.({ type: "awaiting_approvals" });
        return Promise.resolve();
      });
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.runCompletion("chat-123"); });
      await act(async () => {
        const approvalResult = await result.current.approveTool("chat-123", "call_123");
        expect(approvalResult).toEqual(mockResult);
      });
      expect(api.approveToolCall).toHaveBeenCalledWith("call_123", "chat-123");
    });

    it("clears awaitingApprovals when auto_continued", async () => {
      vi.useRealTimers();
      vi.mocked(api.approveToolCall).mockResolvedValue({
        success: true, tool_result: { id: "call_123", tool_name: "run-agent", status: "completed" },
        pending_approvals: [], auto_continued: true,
      });
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.approveTool("chat-123", "call_123"); });
      expect(result.current.awaitingApprovals).toBe(false);
    });

    it("clears awaitingApprovals when no more pending", async () => {
      vi.useRealTimers();
      vi.mocked(api.approveToolCall).mockResolvedValue({
        success: true, tool_result: { id: "call_123", tool_name: "run-agent", status: "completed" },
        pending_approvals: [], auto_continued: false,
      });
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.approveTool("chat-123", "call_123"); });
      expect(result.current.awaitingApprovals).toBe(false);
    });
  });

  describe("rejectTool", () => {
    it("calls API and updates state on rejection", async () => {
      vi.useRealTimers();
      vi.mocked(api.rejectToolCall).mockResolvedValue(undefined);
      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        options?.onEvent?.({ type: "tool_pending_approval", tool_call_id: "call_456", tool_name: "dangerous-tool", arguments: "{}" });
        options?.onEvent?.({ type: "awaiting_approvals" });
        return Promise.resolve();
      });
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.runCompletion("chat-123"); });
      await act(async () => { await result.current.rejectTool("chat-123", "call_456", "Not authorized"); });
      expect(api.rejectToolCall).toHaveBeenCalledWith("call_456", "chat-123", "Not authorized");
    });

    it("clears awaitingApprovals when last pending is rejected", async () => {
      vi.useRealTimers();
      vi.mocked(api.rejectToolCall).mockResolvedValue(undefined);
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.rejectTool("chat-123", "call_456"); });
      expect(result.current.awaitingApprovals).toBe(false);
    });
  });

  describe("resetCompletion", () => {
    it("clears all state and aborts request", async () => {
      vi.useRealTimers();
      let resolveCompletion: (() => void) | undefined;
      vi.mocked(api.completeChat).mockImplementation(async () => {
        await new Promise<void>(resolve => { resolveCompletion = resolve; });
      });
      const { result } = renderHook(() => useCompletion());
      act(() => { void result.current.runCompletion("chat-123"); });
      await waitFor(() => { expect(result.current.isGenerating).toBe(true); });
      act(() => { result.current.resetCompletion(); });
      expect(result.current.isGenerating).toBe(false);
      expect(result.current.streamingContent).toBe("");
      expect(result.current.activeToolCalls).toEqual([]);
      expect(result.current.pendingApprovals).toEqual([]);
      expect(result.current.awaitingApprovals).toBe(false);
      resolveCompletion?.();
    });
  });

  describe("cleanup on unmount", () => {
    it("aborts in-flight request when hook unmounts", async () => {
      vi.useRealTimers();
      let wasAborted = false;
      let resolvePromise: (() => void) | undefined;
      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        options?.signal?.addEventListener("abort", () => { wasAborted = true; });
        await new Promise<void>((resolve) => {
          resolvePromise = resolve;
          options?.signal?.addEventListener("abort", () => resolve());
        });
      });
      const { result, unmount } = renderHook(() => useCompletion());
      act(() => { void result.current.runCompletion("chat-123"); });
      unmount();
      await new Promise(resolve => setTimeout(resolve, 10));
      expect(wasAborted).toBe(true);
      resolvePromise?.();
    });
  });

  describe("error handling", () => {
    it("handles non-abort errors gracefully", async () => {
      vi.useRealTimers();
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const testError = new Error("Network error");
      vi.mocked(api.completeChat).mockRejectedValue(testError);
      const { result } = renderHook(() => useCompletion());
      await act(async () => {
        try { await result.current.runCompletion("chat-123"); } catch (e) { expect(e).toBe(testError); }
      });
      expect(result.current.isGenerating).toBe(false);
      expect(consoleSpy).toHaveBeenCalledWith("Chat completion failed:", testError);
      consoleSpy.mockRestore();
    });

    it("handles AbortError silently without logging", async () => {
      vi.useRealTimers();
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const abortError = new Error("Aborted");
      abortError.name = "AbortError";
      vi.mocked(api.completeChat).mockRejectedValue(abortError);
      const { result } = renderHook(() => useCompletion());
      await act(async () => { await result.current.runCompletion("chat-123"); });
      expect(consoleSpy).not.toHaveBeenCalled();
      consoleSpy.mockRestore();
    });
  });

  describe("return value stability", () => {
    it("returns memoized object to prevent unnecessary re-renders", () => {
      vi.useRealTimers();
      vi.mocked(api.completeChat).mockResolvedValue(undefined);
      const { result, rerender } = renderHook(() => useCompletion());
      const firstRender = result.current;
      rerender();
      const secondRender = result.current;
      expect(firstRender.runCompletion).toBe(secondRender.runCompletion);
      expect(firstRender.cancelCompletion).toBe(secondRender.cancelCompletion);
      expect(firstRender.resetCompletion).toBe(secondRender.resetCompletion);
      expect(firstRender.approveTool).toBe(secondRender.approveTool);
      expect(firstRender.rejectTool).toBe(secondRender.rejectTool);
    });
  });
});

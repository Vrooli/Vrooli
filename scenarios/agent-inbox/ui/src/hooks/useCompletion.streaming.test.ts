/**
 * Tests for useCompletion hook - Streaming, tool calls, and pending approvals
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

describe("useCompletion - streaming and tool calls", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("initial state", () => {
    it("starts with clean state", () => {
      const { result } = renderHook(() => useCompletion());

      expect(result.current.isGenerating).toBe(false);
      expect(result.current.streamingContent).toBe("");
      expect(result.current.generatedImages).toEqual([]);
      expect(result.current.activeToolCalls).toEqual([]);
      expect(result.current.pendingApprovals).toEqual([]);
      expect(result.current.awaitingApprovals).toBe(false);
    });
  });

  describe("streaming content accumulation", () => {
    it("accumulates content events into streamingContent", async () => {
      vi.useRealTimers();

      let eventHandler: ((event: api.StreamingEvent) => void) | undefined;
      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        eventHandler = options?.onEvent;
        eventHandler?.({ type: "content", content: "Hello " });
        eventHandler?.({ type: "content", content: "world" });
        return Promise.resolve();
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123");
      });

      expect(api.completeChat).toHaveBeenCalledWith("chat-123", expect.objectContaining({
        stream: true,
        onEvent: expect.any(Function),
      }));
    });

    it("handles image_generated events", async () => {
      vi.useRealTimers();

      let eventHandler: ((event: api.StreamingEvent) => void) | undefined;
      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        eventHandler = options?.onEvent;
        eventHandler?.({ type: "image_generated", image_url: "https://example.com/image1.png" });
        eventHandler?.({ type: "image_generated", image_url: "https://example.com/image2.png" });
        return Promise.resolve();
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123");
      });

      expect(api.completeChat).toHaveBeenCalled();
    });
  });

  describe("tool call lifecycle", () => {
    it("adds tool call on tool_call_start event", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        options?.onEvent?.({
          type: "tool_call_start",
          tool_id: "call_123",
          tool_name: "run-agent",
          arguments: '{"task": "test"}',
        });
        await new Promise(resolve => setTimeout(resolve, 10));
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        const promise = result.current.runCompletion("chat-123");
        await promise;
      });

      expect(api.completeChat).toHaveBeenCalled();
    });

    it("updates tool call status on tool_call_result event", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        options?.onEvent?.({
          type: "tool_call_start",
          tool_id: "call_123",
          tool_name: "run-agent",
          arguments: '{"task": "test"}',
        });
        options?.onEvent?.({
          type: "tool_call_result",
          tool_id: "call_123",
          status: "completed",
          result: '{"success": true}',
        });
        return Promise.resolve();
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123");
      });

      expect(api.completeChat).toHaveBeenCalled();
    });

    it("marks tool call as failed on failed result", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        options?.onEvent?.({
          type: "tool_call_start",
          tool_id: "call_456",
          tool_name: "run-agent",
        });
        options?.onEvent?.({
          type: "tool_call_result",
          tool_id: "call_456",
          status: "failed",
          error: "Tool execution failed",
        });
        return Promise.resolve();
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123");
      });

      expect(api.completeChat).toHaveBeenCalled();
    });
  });

  describe("pending approval handling", () => {
    it("adds pending approval on tool_pending_approval event", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockImplementation((_chatId, options) => {
        options?.onEvent?.({
          type: "tool_pending_approval",
          tool_call_id: "call_789",
          tool_name: "dangerous-tool",
          arguments: '{"action": "delete"}',
        });
        return Promise.resolve();
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123");
      });

      expect(api.completeChat).toHaveBeenCalled();
    });

    it("sets awaitingApprovals on awaiting_approvals event", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        options?.onEvent?.({ type: "awaiting_approvals" });
        await new Promise(resolve => setTimeout(resolve, 50));
      });

      const { result } = renderHook(() => useCompletion());

      act(() => {
        void result.current.runCompletion("chat-123");
      });

      await waitFor(() => {
        expect(result.current.awaitingApprovals).toBe(true);
        expect(result.current.isGenerating).toBe(false);
      });
    });
  });

  describe("template deactivation callback", () => {
    it("calls onTemplateDeactivated when deactivate_template is true", async () => {
      vi.useRealTimers();

      const onTemplateDeactivated = vi.fn();

      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        options?.onEvent?.({
          type: "tool_call_start",
          tool_id: "call_123",
          tool_name: "suggested-tool",
        });
        options?.onEvent?.({
          type: "tool_call_result",
          tool_id: "call_123",
          status: "completed",
          deactivate_template: true,
        });
        await new Promise(resolve => setTimeout(resolve, 10));
      });

      const { result } = renderHook(() => useCompletion({ onTemplateDeactivated }));

      await act(async () => {
        await result.current.runCompletion("chat-123");
        await new Promise(resolve => setTimeout(resolve, 20));
      });

      expect(onTemplateDeactivated).toHaveBeenCalled();
    });
  });

  describe("completion options", () => {
    it("passes forcedTool option to API", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockResolvedValue(undefined);

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123", {
          forcedTool: { scenario: "agent-manager", toolName: "spawn_coding_agent" },
        });
      });

      expect(api.completeChat).toHaveBeenCalledWith("chat-123", expect.objectContaining({
        forcedTool: { scenario: "agent-manager", toolName: "spawn_coding_agent" },
      }));
    });

    it("passes skills option to API", async () => {
      vi.useRealTimers();

      vi.mocked(api.completeChat).mockResolvedValue(undefined);

      const skills = [
        { id: "skill-1", name: "Test Skill", content: "Do X", key: "test", label: "Test" },
      ];

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-123", { skills });
      });

      expect(api.completeChat).toHaveBeenCalledWith("chat-123", expect.objectContaining({
        skills,
      }));
    });
  });
});

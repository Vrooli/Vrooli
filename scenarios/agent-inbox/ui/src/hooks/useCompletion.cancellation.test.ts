/**
 * Tests for useCompletion hook - Request cancellation and request ID guard
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useCompletion } from "./useCompletion";
import * as api from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  completeChat: vi.fn(),
  approveToolCall: vi.fn(),
  rejectToolCall: vi.fn(),
}));

describe("useCompletion - cancellation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("request cancellation", () => {
    it("cancels in-flight request on cancelCompletion", async () => {
      vi.useRealTimers();

      let _abortSignal: AbortSignal | undefined;
      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        _abortSignal = options?.signal;
        await new Promise((_, reject) => {
          options?.signal?.addEventListener("abort", () => {
            reject(new DOMException("Aborted", "AbortError"));
          });
        });
      });

      const { result } = renderHook(() => useCompletion());

      act(() => {
        result.current.runCompletion("chat-123");
      });

      act(() => {
        result.current.cancelCompletion();
      });

      expect(result.current.isGenerating).toBe(false);
    });

    it("aborts previous request when starting new one", async () => {
      vi.useRealTimers();

      let callCount = 0;
      const abortSignals: AbortSignal[] = [];

      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        callCount++;
        if (options?.signal) {
          abortSignals.push(options.signal);
        }
        await new Promise(resolve => setTimeout(resolve, 10));
      });

      const { result } = renderHook(() => useCompletion());

      await act(async () => {
        await result.current.runCompletion("chat-1");
      });

      await act(async () => {
        await result.current.runCompletion("chat-2");
      });

      expect(callCount).toBe(2);
    });

    it("prevents overlapping completions", async () => {
      vi.useRealTimers();

      let callCount = 0;
      let resolveFirst: (() => void) | undefined;

      vi.mocked(api.completeChat).mockImplementation(async () => {
        callCount++;
        if (callCount === 1) {
          await new Promise<void>(resolve => {
            resolveFirst = resolve;
          });
        }
      });

      const { result } = renderHook(() => useCompletion());

      act(() => {
        result.current.runCompletion("chat-1");
      });

      act(() => {
        result.current.runCompletion("chat-2");
      });

      expect(callCount).toBe(1);

      resolveFirst?.();
    });
  });

  describe("request ID guard", () => {
    it("ignores stale events from cancelled requests", async () => {
      vi.useRealTimers();

      let firstEventHandler: ((event: api.StreamingEvent) => void) | undefined;
      let callCount = 0;

      vi.mocked(api.completeChat).mockImplementation(async (_chatId, options) => {
        callCount++;
        if (callCount === 1) {
          firstEventHandler = options?.onEvent;
          await new Promise(() => {});
        } else {
          // Second request completes immediately
        }
      });

      const { result } = renderHook(() => useCompletion());

      act(() => {
        result.current.runCompletion("chat-1");
      });

      act(() => {
        result.current.cancelCompletion();
      });

      act(() => {
        firstEventHandler?.({ type: "content", content: "Stale content" });
      });

      expect(result.current.streamingContent).toBe("");
    });
  });
});

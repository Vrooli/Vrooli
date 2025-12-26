/**
 * useCompletion - Manages AI chat completion state including streaming and tool calls.
 *
 * This hook encapsulates the complexity of:
 * - Streaming content from AI responses
 * - Tool call execution and status tracking
 * - Generation state management
 * - Request cancellation on unmount or new request
 *
 * TEMPORAL FLOW DESIGN:
 * - Each completion request has a unique ID to prevent race conditions
 * - AbortController enables cancellation on unmount or new request
 * - State updates are guarded by request ID to prevent stale updates
 * - Cleanup runs on unmount to abort in-flight requests
 *
 * SEAM: This hook provides a clean boundary for AI completion logic.
 * For testing, the streaming behavior can be mocked by providing
 * a custom `completeChat` function via dependency injection or
 * by mocking the api module.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { completeChat, StreamingEvent } from "../lib/api";

export interface ActiveToolCall {
  id: string;
  name: string;
  arguments: string;
  status: "running" | "completed" | "failed";
  result?: string;
  error?: string;
}

export interface CompletionState {
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls: ActiveToolCall[];
}

export interface CompletionActions {
  runCompletion: (chatId: string) => Promise<void>;
  resetCompletion: () => void;
  cancelCompletion: () => void;
}

// Generate unique request IDs to correlate state updates
let requestIdCounter = 0;
function generateRequestId(): number {
  return ++requestIdCounter;
}

export function useCompletion(): CompletionState & CompletionActions {
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [activeToolCalls, setActiveToolCalls] = useState<ActiveToolCall[]>([]);

  // Track current request to prevent stale updates from race conditions
  const currentRequestIdRef = useRef<number>(0);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Cleanup on unmount - cancel any in-flight request
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
    };
  }, []);

  // Create event handler with request ID guard
  const createEventHandler = useCallback((requestId: number) => {
    return (event: StreamingEvent) => {
      // Guard: Only process events for the current request
      if (currentRequestIdRef.current !== requestId) {
        return; // Stale event from cancelled request
      }

      switch (event.type) {
        case "content":
          if (event.content) {
            setStreamingContent((prev) => prev + event.content);
          }
          break;

        case "tool_call_start":
          if (event.tool_id && event.tool_name) {
            setActiveToolCalls((prev) => [
              ...prev,
              {
                id: event.tool_id!,
                name: event.tool_name!,
                arguments: event.arguments || "{}",
                status: "running",
              },
            ]);
          }
          break;

        case "tool_call_result":
          if (event.tool_id) {
            setActiveToolCalls((prev) =>
              prev.map((tc) =>
                tc.id === event.tool_id
                  ? {
                      ...tc,
                      status: event.status === "completed" ? "completed" : "failed",
                      result: event.result,
                      error: event.error,
                    }
                  : tc
              )
            );
          }
          break;

        case "tool_calls_complete":
          // Tool calls are done, follow-up response may come
          break;

        case "error":
          console.error("Streaming error:", event.error);
          break;
      }
    };
  }, []);

  const cancelCompletion = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    // Invalidate current request ID to reject any pending state updates
    currentRequestIdRef.current = 0;
    setIsGenerating(false);
    setStreamingContent("");
    setActiveToolCalls([]);
  }, []);

  const runCompletion = useCallback(
    async (chatId: string) => {
      // Cancel any existing request before starting new one
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller and request ID
      const abortController = new AbortController();
      abortControllerRef.current = abortController;
      const requestId = generateRequestId();
      currentRequestIdRef.current = requestId;

      // Reset state for new request
      setIsGenerating(true);
      setStreamingContent("");
      setActiveToolCalls([]);

      try {
        await completeChat(chatId, {
          stream: true,
          onEvent: createEventHandler(requestId),
          signal: abortController.signal,
        });

        // Only reset state if this is still the active request
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          setStreamingContent("");
          setActiveToolCalls([]);
        }
      } catch (error) {
        // Only handle errors for the active request
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          setStreamingContent("");
          setActiveToolCalls([]);

          // Don't throw on abort - it's intentional cancellation
          if (error instanceof Error && error.name === "AbortError") {
            return;
          }

          console.error("Chat completion failed:", error);
          throw error;
        }
      }
    },
    [createEventHandler]
  );

  const resetCompletion = useCallback(() => {
    cancelCompletion();
  }, [cancelCompletion]);

  return {
    // State
    isGenerating,
    streamingContent,
    activeToolCalls,

    // Actions
    runCompletion,
    resetCompletion,
    cancelCompletion,
  };
}

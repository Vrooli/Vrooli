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
import { completeChat, approveToolCall, rejectToolCall, StreamingEvent } from "../lib/api";

export interface ActiveToolCall {
  id: string;
  name: string;
  arguments: string;
  status: "running" | "completed" | "failed" | "pending_approval";
  result?: string;
  error?: string;
}

export interface PendingApproval {
  id: string;
  toolName: string;
  arguments: string;
  startedAt: string;
}

export interface CompletionState {
  isGenerating: boolean;
  streamingContent: string;
  generatedImages: string[]; // AI-generated image URLs during streaming
  activeToolCalls: ActiveToolCall[];
  pendingApprovals: PendingApproval[];
  awaitingApprovals: boolean;
}

export interface CompletionActions {
  runCompletion: (chatId: string) => Promise<void>;
  resetCompletion: () => void;
  cancelCompletion: () => void;
  approveTool: (chatId: string, toolCallId: string) => Promise<void>;
  rejectTool: (chatId: string, toolCallId: string, reason?: string) => Promise<void>;
}

// Generate unique request IDs to correlate state updates
let requestIdCounter = 0;
function generateRequestId(): number {
  return ++requestIdCounter;
}

export function useCompletion(): CompletionState & CompletionActions {
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [generatedImages, setGeneratedImages] = useState<string[]>([]);
  const [activeToolCalls, setActiveToolCalls] = useState<ActiveToolCall[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>([]);
  const [awaitingApprovals, setAwaitingApprovals] = useState(false);
  const [isProcessingApproval, setIsProcessingApproval] = useState(false);

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

        case "image_generated":
          if (event.image_url) {
            setGeneratedImages((prev) => [...prev, event.image_url!]);
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

        case "tool_pending_approval":
          // A tool requires approval before execution
          if (event.tool_call_id && event.tool_name) {
            setPendingApprovals((prev) => [
              ...prev,
              {
                id: event.tool_call_id!,
                toolName: event.tool_name!,
                arguments: event.arguments || "{}",
                startedAt: new Date().toISOString(),
              },
            ]);
            // Also add to active tool calls with pending status
            setActiveToolCalls((prev) => [
              ...prev,
              {
                id: event.tool_call_id!,
                name: event.tool_name!,
                arguments: event.arguments || "{}",
                status: "pending_approval",
              },
            ]);
          }
          break;

        case "awaiting_approvals":
          // Signal that we're waiting for user to approve pending tool calls
          setAwaitingApprovals(true);
          setIsGenerating(false);
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
    setGeneratedImages([]);
    setActiveToolCalls([]);
    setPendingApprovals([]);
    setAwaitingApprovals(false);
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
      setGeneratedImages([]);
      setActiveToolCalls([]);
      setPendingApprovals([]);
      setAwaitingApprovals(false);

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
          setGeneratedImages([]);
          setActiveToolCalls([]);
          // Note: Don't reset pendingApprovals here as we might be awaiting approvals
        }
      } catch (error) {
        // Only handle errors for the active request
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          setStreamingContent("");
          setGeneratedImages([]);
          setActiveToolCalls([]);
          setPendingApprovals([]);
          setAwaitingApprovals(false);

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

  // Approve a pending tool call and optionally trigger auto-continue
  const approveTool = useCallback(
    async (chatId: string, toolCallId: string) => {
      setIsProcessingApproval(true);
      try {
        const result = await approveToolCall(toolCallId, chatId);

        // Remove from pending approvals
        setPendingApprovals((prev) => prev.filter((p) => p.id !== toolCallId));

        // Update active tool call status
        setActiveToolCalls((prev) =>
          prev.map((tc) =>
            tc.id === toolCallId
              ? { ...tc, status: "completed" as const, result: result.tool_result?.result }
              : tc
          )
        );

        // If auto-continued, a new completion was triggered
        // The UI should refetch messages to get the new response
        if (result.auto_continued) {
          // Reset state for the new completion that was triggered
          setAwaitingApprovals(false);
        } else if (result.pending_approvals?.length === 0) {
          // No more pending approvals
          setAwaitingApprovals(false);
        }

        return result;
      } finally {
        setIsProcessingApproval(false);
      }
    },
    []
  );

  // Reject a pending tool call
  const rejectTool = useCallback(
    async (chatId: string, toolCallId: string, reason?: string) => {
      setIsProcessingApproval(true);
      try {
        await rejectToolCall(toolCallId, chatId, reason);

        // Remove from pending approvals
        setPendingApprovals((prev) => prev.filter((p) => p.id !== toolCallId));

        // Update active tool call status
        setActiveToolCalls((prev) =>
          prev.map((tc) =>
            tc.id === toolCallId
              ? { ...tc, status: "failed" as const, error: reason || "Rejected by user" }
              : tc
          )
        );

        // If no more pending approvals, clear the awaiting state
        setPendingApprovals((current) => {
          if (current.length === 0) {
            setAwaitingApprovals(false);
          }
          return current;
        });
      } finally {
        setIsProcessingApproval(false);
      }
    },
    []
  );

  const resetCompletion = useCallback(() => {
    cancelCompletion();
  }, [cancelCompletion]);

  return {
    // State
    isGenerating,
    streamingContent,
    generatedImages,
    activeToolCalls,
    pendingApprovals,
    awaitingApprovals,

    // Actions
    runCompletion,
    resetCompletion,
    cancelCompletion,
    approveTool,
    rejectTool,
  };
}

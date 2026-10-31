/**
 * useCompletion - Manages AI chat completion state including streaming and tool calls.
 *
 * This hook encapsulates the complexity of:
 * - Streaming content from AI responses
 * - Tool call execution and status tracking
 * - Generation state management
 * - Request cancellation on unmount or new request
 *
 * Implementation is split across:
 * - useCompletionTypes.ts: Shared types and constants
 * - useCompletionStreaming.ts: SSE event handler factory
 * - useCompletion.ts (this file): Main composition hook
 *
 * SEAM: This hook provides a clean boundary for AI completion logic.
 */
import { useState, useCallback, useRef, useEffect, startTransition, useMemo } from "react";
import { completeChat, approveToolCall, rejectToolCall } from "../lib/api";
import {
  type ActiveToolCall,
  type PendingApproval,
  type CompletionState,
  type CompletionActions,
  type CompletionOptions,
  EMPTY_IMAGES,
  EMPTY_TOOL_CALLS,
  EMPTY_APPROVALS,
  generateRequestId,
} from "./useCompletionTypes";
import { useStreamingEventHandler } from "./useCompletionStreaming";

// Re-export types for consumers
export type { ActiveToolCall, PendingApproval, CompletionState, CompletionActions, CompletionOptions };

export function useCompletion(): CompletionState & CompletionActions {
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [generatedImages, setGeneratedImages] = useState<string[]>(EMPTY_IMAGES);
  const [activeToolCalls, setActiveToolCalls] = useState<ActiveToolCall[]>(EMPTY_TOOL_CALLS);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>(EMPTY_APPROVALS);
  const [awaitingApprovals, setAwaitingApprovals] = useState(false);
  const [_isProcessingApproval, setIsProcessingApproval] = useState(false);

  const currentRequestIdRef = useRef<number>(0);
  const abortControllerRef = useRef<AbortController | null>(null);
  const isCompletionInFlightRef = useRef(false);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
        abortControllerRef.current = null;
      }
    };
  }, []);

  // Streaming event handler from extracted module
  const createEventHandler = useStreamingEventHandler(
    currentRequestIdRef,
    {
      setStreamingContent,
      setGeneratedImages,
      setActiveToolCalls,
      setPendingApprovals,
      setAwaitingApprovals,
      setIsGenerating,
    },
  );

  const cancelCompletion = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    currentRequestIdRef.current = 0;
    isCompletionInFlightRef.current = false;
    setIsGenerating(false);
    startTransition(() => {
      setStreamingContent("");
      setGeneratedImages(EMPTY_IMAGES);
      setActiveToolCalls(EMPTY_TOOL_CALLS);
      setPendingApprovals(EMPTY_APPROVALS);
      setAwaitingApprovals(false);
    });
  }, []);

  const runCompletion = useCallback(
    async (chatId: string, options?: CompletionOptions) => {
      if (isCompletionInFlightRef.current) return;
      isCompletionInFlightRef.current = true;

      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const abortController = new AbortController();
      abortControllerRef.current = abortController;
      const requestId = generateRequestId();
      currentRequestIdRef.current = requestId;

      setIsGenerating(true);
      startTransition(() => {
        setStreamingContent("");
        setGeneratedImages(EMPTY_IMAGES);
        setActiveToolCalls(EMPTY_TOOL_CALLS);
        setPendingApprovals(EMPTY_APPROVALS);
        setAwaitingApprovals(false);
      });

      try {
        await completeChat(chatId, {
          stream: true,
          onEvent: createEventHandler(requestId),
          signal: abortController.signal,
          skills: options?.skills,
        });

        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          startTransition(() => {
            setStreamingContent("");
            setGeneratedImages(EMPTY_IMAGES);
            setActiveToolCalls(EMPTY_TOOL_CALLS);
          });
        }
      } catch (error) {
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          startTransition(() => {
            setStreamingContent("");
            setGeneratedImages(EMPTY_IMAGES);
            setActiveToolCalls(EMPTY_TOOL_CALLS);
            setPendingApprovals(EMPTY_APPROVALS);
            setAwaitingApprovals(false);
          });

          if (error instanceof Error && error.name === "AbortError") return;
          console.error("Chat completion failed:", error);
          throw error;
        }
      } finally {
        isCompletionInFlightRef.current = false;
      }
    },
    [createEventHandler]
  );

  // Approve a pending tool call
  const approveTool = useCallback(
    async (chatId: string, toolCallId: string) => {
      setIsProcessingApproval(true);
      try {
        const result = await approveToolCall(toolCallId, chatId);

        setPendingApprovals((prev) => prev.filter((p) => p.id !== toolCallId));
        setActiveToolCalls((prev) =>
          prev.map((tc) =>
            tc.id === toolCallId
              ? { ...tc, status: "completed" as const, result: result.tool_result.result }
              : tc
          )
        );

        if (result.auto_continued) {
          setAwaitingApprovals(false);
        } else if (result.pending_approvals.length === 0) {
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

        let newPendingApprovals: PendingApproval[] = [];
        setPendingApprovals((prev) => {
          newPendingApprovals = prev.filter((p) => p.id !== toolCallId);
          return newPendingApprovals;
        });

        setActiveToolCalls((prev) =>
          prev.map((tc) =>
            tc.id === toolCallId
              ? { ...tc, status: "failed" as const, error: reason || "Rejected by user" }
              : tc
          )
        );

        if (newPendingApprovals.length === 0) {
          setAwaitingApprovals(false);
        }
      } finally {
        setIsProcessingApproval(false);
      }
    },
    []
  );

  const resetCompletion = useCallback(() => {
    cancelCompletion();
  }, [cancelCompletion]);

  return useMemo(
    () => ({
      isGenerating,
      streamingContent,
      generatedImages,
      activeToolCalls,
      pendingApprovals,
      awaitingApprovals,
      runCompletion,
      resetCompletion,
      cancelCompletion,
      approveTool,
      rejectTool,
    }),
    [
      isGenerating,
      streamingContent,
      generatedImages,
      activeToolCalls,
      pendingApprovals,
      awaitingApprovals,
      runCompletion,
      resetCompletion,
      cancelCompletion,
      approveTool,
      rejectTool,
    ]
  );
}

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
import { useState, useCallback, useRef, useEffect, startTransition, useMemo } from "react";
import { completeChat, approveToolCall, rejectToolCall, StreamingEvent, ApprovalResult, SkillPayloadForAPI } from "../lib/api";

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

export interface CompletionOptions {
  forcedTool?: { scenario: string; toolName: string };
  skills?: SkillPayloadForAPI[];
}

export interface CompletionActions {
  runCompletion: (chatId: string, options?: CompletionOptions) => Promise<void>;
  resetCompletion: () => void;
  cancelCompletion: () => void;
  approveTool: (chatId: string, toolCallId: string) => Promise<ApprovalResult>;
  rejectTool: (chatId: string, toolCallId: string, reason?: string) => Promise<void>;
}

export interface UseCompletionOptions {
  /** Called when a template's suggested tool is used (template should be deactivated) */
  onTemplateDeactivated?: () => void;
}

// Generate unique request IDs to correlate state updates
let requestIdCounter = 0;
function generateRequestId(): number {
  return ++requestIdCounter;
}

// Stable empty arrays to prevent infinite re-render loops
// CRITICAL: Using inline [] creates new array on every setState call,
// which changes references and triggers useMemo/useCallback dependencies
const EMPTY_IMAGES: string[] = [];
const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
const EMPTY_APPROVALS: PendingApproval[] = [];

export function useCompletion(options?: UseCompletionOptions): CompletionState & CompletionActions {
  const { onTemplateDeactivated } = options || {};
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [generatedImages, setGeneratedImages] = useState<string[]>(EMPTY_IMAGES);
  const [activeToolCalls, setActiveToolCalls] = useState<ActiveToolCall[]>(EMPTY_TOOL_CALLS);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>(EMPTY_APPROVALS);
  const [awaitingApprovals, setAwaitingApprovals] = useState(false);
  const [_isProcessingApproval, setIsProcessingApproval] = useState(false);

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
  // CRITICAL: SSE events from EventSource are NOT automatically batched by React 18.
  // We use startTransition to mark state updates as non-urgent, which allows React
  // to batch them more effectively and prevents "too many re-renders" errors during
  // rapid streaming updates.
  const createEventHandler = useCallback((requestId: number) => {
    return (event: StreamingEvent) => {
      // Guard: Only process events for the current request
      if (currentRequestIdRef.current !== requestId) {
        return; // Stale event from cancelled request
      }

      switch (event.type) {
        case "content":
          if (event.content) {
            // Content updates are frequent but non-urgent - use transition
            startTransition(() => {
              setStreamingContent((prev) => prev + event.content);
            });
          }
          break;

        case "image_generated":
          if (event.image_url) {
            const imageUrl = event.image_url;
            startTransition(() => {
              setGeneratedImages((prev) => [...prev, imageUrl]);
            });
          }
          break;

        case "tool_call_start":
          if (event.tool_id && event.tool_name) {
            const toolId = event.tool_id;
            const toolName = event.tool_name;
            startTransition(() => {
              setActiveToolCalls((prev) => [
                ...prev,
                {
                  id: toolId,
                  name: toolName,
                  arguments: event.arguments || "{}",
                  status: "running",
                },
              ]);
            });
          }
          break;

        case "tool_call_result":
          if (event.tool_id) {
            startTransition(() => {
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
            });
            // Check if a template's suggested tool was used
            // CRITICAL: Defer this callback to avoid React Error #310
            // ("Cannot update a component while rendering a different component").
            // The callback triggers parent state updates via useActiveTemplate.deactivate(),
            // which would otherwise happen during the child's render phase.
            if (event.deactivate_template && onTemplateDeactivated) {
              queueMicrotask(() => {
                onTemplateDeactivated();
              });
            }
          }
          break;

        case "tool_calls_complete":
          // Tool calls are done, follow-up response may come
          break;

        case "tool_pending_approval":
          // A tool requires approval before execution
          if (event.tool_call_id && event.tool_name) {
            const toolCallId = event.tool_call_id;
            const toolName = event.tool_name;
            const toolArgs = event.arguments || "{}";
            startTransition(() => {
              setPendingApprovals((prev) => [
                ...prev,
                {
                  id: toolCallId,
                  toolName: toolName,
                  arguments: toolArgs,
                  startedAt: new Date().toISOString(),
                },
              ]);
              // Also add to active tool calls with pending status
              setActiveToolCalls((prev) => [
                ...prev,
                {
                  id: toolCallId,
                  name: toolName,
                  arguments: toolArgs,
                  status: "pending_approval",
                },
              ]);
            });
          }
          break;

        case "awaiting_approvals":
          // Signal that we're waiting for user to approve pending tool calls
          // These are urgent state changes that affect UI flow
          setAwaitingApprovals(true);
          setIsGenerating(false);
          break;

        case "error":
          console.error("Streaming error:", event.error);
          break;
      }
    };
  }, [onTemplateDeactivated]);

  // Track if a completion is currently in flight to prevent overlapping requests
  // NOTE: Defined here before cancelCompletion so it can be referenced
  const isCompletionInFlightRef = useRef(false);

  const cancelCompletion = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    // Invalidate current request ID to reject any pending state updates
    currentRequestIdRef.current = 0;
    // Clear in-flight flag since we're cancelling
    isCompletionInFlightRef.current = false;
    // Batch all state resets to minimize render cycles
    // isGenerating is urgent, others can be deferred
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
      // Prevent overlapping completions which can cause render storms
      // during rapid state transitions (e.g., fresh chat message send)
      if (isCompletionInFlightRef.current) {
        return;
      }
      isCompletionInFlightRef.current = true;

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
      // CRITICAL: isGenerating is urgent (affects UI immediately), but other resets
      // can be batched via startTransition to prevent render storms during the
      // critical fresh-chat-to-generating transition.
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
          forcedTool: options?.forcedTool,
          skills: options?.skills,
        });

        // Only reset state if this is still the active request
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          // Batch non-urgent resets via transition
          startTransition(() => {
            setStreamingContent("");
            setGeneratedImages(EMPTY_IMAGES);
            setActiveToolCalls(EMPTY_TOOL_CALLS);
            // Note: Don't reset pendingApprovals here as we might be awaiting approvals
          });
        }
      } catch (error) {
        // Only handle errors for the active request
        if (currentRequestIdRef.current === requestId) {
          setIsGenerating(false);
          // Batch non-urgent resets via transition
          startTransition(() => {
            setStreamingContent("");
            setGeneratedImages(EMPTY_IMAGES);
            setActiveToolCalls(EMPTY_TOOL_CALLS);
            setPendingApprovals(EMPTY_APPROVALS);
            setAwaitingApprovals(false);
          });

          // Don't throw on abort - it's intentional cancellation
          if (error instanceof Error && error.name === "AbortError") {
            return;
          }

          console.error("Chat completion failed:", error);
          throw error;
        }
      } finally {
        // Always clear the in-flight flag when completion ends
        isCompletionInFlightRef.current = false;
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

        // Remove from pending approvals and check if we should clear awaiting state
        // CRITICAL: Do NOT call setState inside another setState's updater function.
        // That's a React anti-pattern that can cause "too many re-renders" errors.
        // Instead, we calculate the new state and then update both values.
        let newPendingApprovals: PendingApproval[] = [];
        setPendingApprovals((prev) => {
          newPendingApprovals = prev.filter((p) => p.id !== toolCallId);
          return newPendingApprovals;
        });

        // Update active tool call status
        setActiveToolCalls((prev) =>
          prev.map((tc) =>
            tc.id === toolCallId
              ? { ...tc, status: "failed" as const, error: reason || "Rejected by user" }
              : tc
          )
        );

        // Clear awaiting state if no more pending approvals
        // This is safe to do outside the updater since we captured the new value above
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

  // CRITICAL: Memoize the return object to prevent creating new object references
  // on every render. Without this, every render creates a new object that triggers
  // re-renders in consuming components (useChats, App.tsx), potentially causing
  // "too many re-renders" errors during rapid state transitions.
  return useMemo(
    () => ({
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

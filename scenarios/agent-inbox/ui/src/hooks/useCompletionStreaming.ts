/**
 * useCompletionStreaming - SSE event handler factory for streaming completions.
 *
 * Extracted from useCompletion.ts. Handles all streaming event types
 * (content, tool_call_start, tool_call_result, tool_pending_approval, etc.)
 * using React startTransition for non-urgent batched updates.
 */
import { useCallback, startTransition, type Dispatch, type SetStateAction } from "react";
import type { StreamingEvent } from "../lib/api";
import type { ActiveToolCall, PendingApproval } from "./useCompletionTypes";
import type { MutableRefObject } from "react";

export interface StreamingSetters {
  setStreamingContent: Dispatch<SetStateAction<string>>;
  setGeneratedImages: Dispatch<SetStateAction<string[]>>;
  setActiveToolCalls: Dispatch<SetStateAction<ActiveToolCall[]>>;
  setPendingApprovals: Dispatch<SetStateAction<PendingApproval[]>>;
  setAwaitingApprovals: Dispatch<SetStateAction<boolean>>;
  setIsGenerating: Dispatch<SetStateAction<boolean>>;
}

export function useStreamingEventHandler(
  currentRequestIdRef: MutableRefObject<number>,
  setters: StreamingSetters,
  onTemplateDeactivated?: () => void,
) {
  const {
    setStreamingContent,
    setGeneratedImages,
    setActiveToolCalls,
    setPendingApprovals,
    setAwaitingApprovals,
    setIsGenerating,
  } = setters;

  return useCallback((requestId: number) => {
    return (event: StreamingEvent) => {
      // Guard: Only process events for the current request
      if (currentRequestIdRef.current !== requestId) return;

      switch (event.type) {
        case "content":
          if (event.content) {
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
            if (event.deactivate_template && onTemplateDeactivated) {
              queueMicrotask(() => {
                onTemplateDeactivated();
              });
            }
          }
          break;

        case "tool_calls_complete":
          break;

        case "tool_pending_approval":
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
          setAwaitingApprovals(true);
          setIsGenerating(false);
          break;

        case "error":
          console.error("Streaming error:", event.error);
          break;
      }
    };
  }, [
    currentRequestIdRef,
    setStreamingContent,
    setGeneratedImages,
    setActiveToolCalls,
    setPendingApprovals,
    setAwaitingApprovals,
    setIsGenerating,
    onTemplateDeactivated,
  ]);
}

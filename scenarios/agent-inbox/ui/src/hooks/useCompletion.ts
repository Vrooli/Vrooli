/**
 * useCompletion - Manages AI chat completion state including streaming and tool calls.
 *
 * This hook encapsulates the complexity of:
 * - Streaming content from AI responses
 * - Tool call execution and status tracking
 * - Generation state management
 *
 * SEAM: This hook provides a clean boundary for AI completion logic.
 * For testing, the streaming behavior can be mocked by providing
 * a custom `completeChat` function via dependency injection or
 * by mocking the api module.
 */
import { useState, useCallback } from "react";
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
}

export function useCompletion(): CompletionState & CompletionActions {
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");
  const [activeToolCalls, setActiveToolCalls] = useState<ActiveToolCall[]>([]);

  const handleStreamingEvent = useCallback((event: StreamingEvent) => {
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
  }, []);

  const runCompletion = useCallback(
    async (chatId: string) => {
      setIsGenerating(true);
      setStreamingContent("");
      setActiveToolCalls([]);

      try {
        await completeChat(chatId, {
          stream: true,
          onEvent: handleStreamingEvent,
        });
      } catch (error) {
        console.error("Chat completion failed:", error);
        throw error;
      } finally {
        setIsGenerating(false);
        setStreamingContent("");
        setActiveToolCalls([]);
      }
    },
    [handleStreamingEvent]
  );

  const resetCompletion = useCallback(() => {
    setIsGenerating(false);
    setStreamingContent("");
    setActiveToolCalls([]);
  }, []);

  return {
    // State
    isGenerating,
    streamingContent,
    activeToolCalls,

    // Actions
    runCompletion,
    resetCompletion,
  };
}

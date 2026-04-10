import { useState, useCallback, useEffect } from "react";
import type { AsyncStatusUpdate } from "./useAsyncStatus";

export interface AsyncReference {
  tool_call_id: string;
  tool_name: string;
  status: string;
  summary: string;
}

export interface AsyncReferencesState {
  asyncReferences: AsyncReference[];
  handleInsertAsyncReference: (op: AsyncStatusUpdate) => void;
  handleRemoveAsyncReference: (toolCallId: string) => void;
  asyncHistoryOffset: number;
  hasMoreAsyncHistory: boolean;
  handleFetchAsyncHistory: () => Promise<void>;
}

export function useAsyncReferences(
  selectedChatId: string | null,
  fetchAsyncHistory: (limit: number, offset: number) => Promise<{ operations: unknown[]; hasMore: boolean }>
): AsyncReferencesState {
  const [asyncReferences, setAsyncReferences] = useState<AsyncReference[]>([]);
  const [asyncHistoryOffset, setAsyncHistoryOffset] = useState(0);
  const [hasMoreAsyncHistory, setHasMoreAsyncHistory] = useState(true);

  // Handle inserting an async result reference
  const handleInsertAsyncReference = useCallback((op: AsyncStatusUpdate) => {
    const summarizeResult = (result: unknown, maxLength = 100): string => {
      if (result === null || result === undefined) return "No result data";
      if (typeof result === "string") {
        return result.length > maxLength ? result.slice(0, maxLength - 3) + "..." : result;
      }
      if (typeof result === "object") {
        const obj = result as Record<string, unknown>;
        if (typeof obj.message === "string") {
          return obj.message.length > maxLength ? obj.message.slice(0, maxLength - 3) + "..." : obj.message;
        }
        if (typeof obj.summary === "string") {
          return obj.summary.length > maxLength ? obj.summary.slice(0, maxLength - 3) + "..." : obj.summary;
        }
        if (Array.isArray(obj.files)) {
          return `Created ${obj.files.length} file${obj.files.length !== 1 ? "s" : ""}`;
        }
      }
      return "Result available";
    };

    setAsyncReferences((prev) => [
      ...prev.filter((r) => r.tool_call_id !== op.tool_call_id),
      {
        tool_call_id: op.tool_call_id,
        tool_name: op.tool_name,
        status: op.status,
        summary: summarizeResult(op.result, 100),
      },
    ]);
  }, []);

  // Handle removing an async result reference
  const handleRemoveAsyncReference = useCallback((toolCallId: string) => {
    setAsyncReferences((prev) => prev.filter((r) => r.tool_call_id !== toolCallId));
  }, []);

  // Handle fetching more async history
  const handleFetchAsyncHistory = useCallback(async () => {
    const result = await fetchAsyncHistory(20, asyncHistoryOffset);
    setAsyncHistoryOffset((prev) => prev + result.operations.length);
    setHasMoreAsyncHistory(result.hasMore);
  }, [fetchAsyncHistory, asyncHistoryOffset]);

  // Reset async references when chat changes
  useEffect(() => {
    setAsyncReferences([]);
    setAsyncHistoryOffset(0);
    setHasMoreAsyncHistory(true);
  }, [selectedChatId]);

  return {
    asyncReferences,
    handleInsertAsyncReference,
    handleRemoveAsyncReference,
    asyncHistoryOffset,
    hasMoreAsyncHistory,
    handleFetchAsyncHistory,
  };
}

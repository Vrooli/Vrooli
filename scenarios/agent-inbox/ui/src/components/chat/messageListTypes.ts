import type { Message } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useCompletion";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";

// Stable empty arrays for default prop values
// CRITICAL: Using `= []` in destructuring creates a NEW array on every render,
// which changes references and triggers infinite re-render loops via useMemo dependencies
export const EMPTY_IMAGES: string[] = [];
export const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
export const EMPTY_TOOL_RECORDS: import("../../lib/api").ToolCallRecord[] = [];
export const EMPTY_APPROVALS: import("../../hooks/useCompletion").PendingApproval[] = [];
export const EMPTY_SIBLINGS: Message[] = [];
export const EMPTY_ASYNC_OPS: AsyncStatusUpdate[] = [];

// Stable default for sibling info to prevent creating new objects in useMemo
// CRITICAL: Returning { siblings: [] } creates a NEW array each time, which changes
// references and triggers useCallback dependencies like handlePreviousVersion to recreate
export const DEFAULT_SIBLING_INFO = { current: 1, total: 1, siblings: EMPTY_SIBLINGS };

// Stable empty map for sibling info when there are no messages
// CRITICAL: Using `new Map()` inside a component creates new reference each render
export const EMPTY_SIBLING_MAP: Map<string, { current: number; total: number; siblings: Message[] }> = new Map();

// Stable empty map for async operations
export const EMPTY_ASYNC_OP_MAP: Map<string, AsyncStatusUpdate> = new Map();

// Stable empty array for filtered messages when messages is empty
export const EMPTY_MESSAGES: Message[] = [];

export function formatTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

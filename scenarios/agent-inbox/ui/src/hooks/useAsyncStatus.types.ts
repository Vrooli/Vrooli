/**
 * Types and type guards for the useAsyncStatus hook.
 */

import { resolveApiBase } from "@vrooli/api-base";

export const API_BASE = resolveApiBase({ appendSuffix: true });

/** Status update received via SSE */
export interface AsyncStatusUpdate {
  tool_call_id: string;
  chat_id: string;
  tool_name: string;
  status: string;
  progress?: number;
  message?: string;
  phase?: string;
  result?: unknown;
  error?: string;
  is_terminal: boolean;
  updated_at: string;
}

/** History response from the API */
export interface AsyncHistoryResponse {
  operations: AsyncStatusUpdate[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

/** Options for the useAsyncStatus hook */
export interface UseAsyncStatusOptions {
  /** Whether to auto-connect when chatId is provided (default: true) */
  autoConnect?: boolean;
}

/** Return type for the useAsyncStatus hook */
export interface UseAsyncStatusReturn {
  /** All operations (active + recently completed) */
  operations: AsyncStatusUpdate[];
  /** Only non-terminal operations */
  activeOperations: AsyncStatusUpdate[];
  /** Only terminal operations */
  completedOperations: AsyncStatusUpdate[];
  /** Whether SSE connection is established */
  isConnected: boolean;
  /** Connection error message, if any */
  error: string | null;
  /** Cancel a running operation */
  cancelOperation: (toolCallId: string) => Promise<void>;
  /** Force-refresh a specific operation (immediate status poll) */
  refreshOperation: (toolCallId: string) => Promise<AsyncStatusUpdate>;
  /** Fetch completed operation history from server */
  fetchHistory: (limit?: number, offset?: number) => Promise<{
    operations: AsyncStatusUpdate[];
    total: number;
    hasMore: boolean;
  }>;
  /** Number of non-terminal operations */
  activeCount: number;
}

// Stable empty array for when there are no operations
// CRITICAL: Using inline [] creates new array on every render, causing infinite re-render loops
export const EMPTY_OPERATIONS: AsyncStatusUpdate[] = [];

export function isAsyncStatusUpdate(v: unknown): v is AsyncStatusUpdate {
  return typeof v === 'object' && v !== null
    && typeof (v as Record<string, unknown>).tool_call_id === 'string'
    && typeof (v as Record<string, unknown>).is_terminal === 'boolean';
}

export function isAsyncHistoryResponse(v: unknown): v is AsyncHistoryResponse {
  return typeof v === 'object' && v !== null
    && Array.isArray((v as Record<string, unknown>).operations);
}

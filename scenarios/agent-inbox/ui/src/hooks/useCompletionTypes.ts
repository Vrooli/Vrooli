/**
 * Shared types and constants for the useCompletion hook family.
 *
 * Extracted from useCompletion.ts for modularity.
 */
import type { SkillPayloadForAPI } from "../lib/api";

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
  generatedImages: string[];
  activeToolCalls: ActiveToolCall[];
  pendingApprovals: PendingApproval[];
  awaitingApprovals: boolean;
}

export interface CompletionOptions {
  skills?: SkillPayloadForAPI[];
}

export interface CompletionActions {
  runCompletion: (chatId: string, options?: CompletionOptions) => Promise<void>;
  resetCompletion: () => void;
  cancelCompletion: () => void;
  approveTool: (chatId: string, toolCallId: string) => Promise<import("../lib/api").ApprovalResult>;
  rejectTool: (chatId: string, toolCallId: string, reason?: string) => Promise<void>;
}

// Stable empty arrays to prevent infinite re-render loops
export const EMPTY_IMAGES: string[] = [];
export const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
export const EMPTY_APPROVALS: PendingApproval[] = [];

// Generate unique request IDs to correlate state updates
let requestIdCounter = 0;
export function generateRequestId(): number {
  return ++requestIdCounter;
}

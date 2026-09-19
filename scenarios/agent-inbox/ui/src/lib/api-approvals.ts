/**
 * Tool approval and pending approval API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { ToolCallRecord } from "./api-types";

// =============================================================================
// Pending Approvals
// =============================================================================

export interface PendingApproval {
  id: string;
  tool_name: string;
  arguments: string;
  status: string;
  started_at: string;
}

export interface ApprovalResult {
  success: boolean;
  tool_result: {
    id: string;
    tool_name: string;
    status: string;
    result?: string;
  };
  pending_approvals: PendingApproval[];
  auto_continued: boolean;
}

export async function fetchChatToolCalls(chatId: string): Promise<ToolCallRecord[]> {
  const url = buildApiUrl(`/chats/${chatId}/tool-calls`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch tool calls: ${res.status}`);
  }

  return jsonResponse<ToolCallRecord[]>(res);
}

/**
 * Get pending tool call approvals for a chat.
 * @param chatId - Chat ID
 */
export async function getPendingApprovals(chatId: string): Promise<PendingApproval[]> {
  const url = buildApiUrl(`/chats/${chatId}/pending-approvals`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get pending approvals: ${res.status}`);
  }

  const data = await jsonResponse<{ pending_approvals: PendingApproval[] }>(res);
  return data.pending_approvals;
}

/**
 * Approve a pending tool call.
 * @param toolCallId - Tool call ID
 * @param chatId - Chat ID for validation
 */
export async function approveToolCall(toolCallId: string, chatId: string): Promise<ApprovalResult> {
  const url = buildApiUrl(`/tool-calls/${toolCallId}/approve?chat_id=${chatId}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to approve tool call: ${errorText}`);
  }

  return jsonResponse<ApprovalResult>(res);
}

/**
 * Reject a pending tool call.
 * @param toolCallId - Tool call ID
 * @param chatId - Chat ID for validation
 * @param reason - Optional rejection reason
 */
export async function rejectToolCall(toolCallId: string, chatId: string, reason?: string): Promise<void> {
  const url = buildApiUrl(`/tool-calls/${toolCallId}/reject?chat_id=${chatId}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason })
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to reject tool call: ${errorText}`);
  }
}

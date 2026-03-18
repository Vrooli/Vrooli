/**
 * Tool approval and pending approval API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { ApprovalOverride } from "./api-types";

// =============================================================================
// Tool Approval Override
// =============================================================================

/**
 * Set the approval override for a tool.
 * @param scenario - Scenario name
 * @param toolName - Tool name
 * @param approvalOverride - Approval override value ("" | "require" | "skip")
 * @param chatId - Optional chat ID for chat-specific configuration
 */
export async function setToolApproval(
  scenario: string,
  toolName: string,
  approvalOverride: ApprovalOverride,
  chatId?: string
): Promise<void> {
  const url = buildApiUrl("/tools/config/approval", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      scenario,
      tool_name: toolName,
      approval_override: approvalOverride
    })
  });

  if (!res.ok) {
    throw new Error(`Failed to set tool approval: ${res.status}`);
  }
}

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

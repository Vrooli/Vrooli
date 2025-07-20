/**
 * Tool Approval Types
 * 
 * Unified type definitions for tool approval system used across
 * API endpoints, socket events, and internal services.
 */

import { type EventTypes } from "../consts/socketEvents.js";
import type { SocketEventPayloads } from "../consts/socketEvents.js";

/**
 * Extract socket event payload types for tool approvals
 */
export type ToolApprovalGrantedPayload = SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_GRANTED];
export type ToolApprovalRejectedPayload = SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_REJECTED];
export type ToolApprovalRequiredPayload = SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_REQUIRED];
export type ToolApprovalTimeoutPayload = SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_TIMEOUT];

/**
 * API input type for responding to tool approval requests
 * Used by the respondToToolApproval endpoint
 */
export interface RespondToToolApprovalInput {
    /** ID of the conversation/chat containing the tool request */
    conversationId: string;
    /** Unique ID of the pending approval */
    pendingId: string;
    /** Whether the tool use is approved */
    approved: boolean;
    /** Optional reason for rejection (only used when approved=false) */
    reason?: string;
}

/**
 * Result type for tool approval API responses
 */
export interface RespondToToolApprovalResult {
    __typename: "Success";
    success: boolean;
    error?: string;
}

/**
 * Internal type for user approval responses
 * Used by approval system services
 */
export interface UserApprovalResponse {
    /** Unique ID of the pending approval */
    pendingId: string;
    /** Whether the tool use is approved */
    approved: boolean;
    /** ID of the user making the decision */
    userId: string;
    /** Optional reason for rejection */
    reason?: string;
}

/**
 * Result of approval processing
 * Returned by approval system after processing
 */
export interface ApprovalResult {
    /** Whether the tool was approved */
    approved: boolean;
    /** Optional reason for the decision */
    reason?: string;
    /** Time taken to make the decision (ms) */
    approvalDuration?: number;
    /** ID of user who approved (if approved) */
    approvedBy?: string;
    /** ID of user who rejected (if rejected) */
    rejectedBy?: string;
}

/**
 * Utility type guards
 */
export function isToolApprovalGranted(
    payload: ToolApprovalGrantedPayload | ToolApprovalRejectedPayload,
): payload is ToolApprovalGrantedPayload {
    return "approvedBy" in payload && !("reason" in payload);
}

export function isToolApprovalRejected(
    payload: ToolApprovalGrantedPayload | ToolApprovalRejectedPayload,
): payload is ToolApprovalRejectedPayload {
    return "reason" in payload && !("approvedBy" in payload);
}

/**
 * Utility to convert between conversationId and chatId naming
 * (Since socket events use chatId but API uses conversationId)
 */
export function toSocketPayload(
    input: RespondToToolApprovalInput & { toolCallId: string; toolName: string; callerBotId: string },
    userId: string,
): ToolApprovalGrantedPayload | ToolApprovalRejectedPayload {
    const base = {
        chatId: input.conversationId, // Convert naming convention
        pendingId: input.pendingId,
        toolCallId: input.toolCallId,
        toolName: input.toolName,
        callerBotId: input.callerBotId,
    };

    if (input.approved) {
        return {
            ...base,
            approvedBy: userId,
        } as ToolApprovalGrantedPayload;
    } else {
        return {
            ...base,
            reason: input.reason,
        } as ToolApprovalRejectedPayload;
    }
}

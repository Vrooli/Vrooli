/**
 * Tool Approval System Integration
 * 
 * Tool approval system integrated with event architecture.
 * Handles approval requests, user responses, and timeout management using
 * the unified event system with barrier synchronization.
 */

import { nanoid } from "nanoid";
import { type Logger } from "winston";
import { getEventBus } from "./eventBus.js";
import {
    type EventMetadata,
    type EventSource,
    type ToolEvent,
    type ToolEventData,
} from "./types.js";

/**
 * Tool call request for approval processing
 */
export interface ToolCallRequest {
    id: string;
    name: string;
    parameters: Record<string, unknown>;
    callerBotId: string;
    conversationId: string;
    approvalTimeoutMs: number;
    autoRejectOnTimeout: boolean;
    estimatedCredits: string;
    riskLevel: "low" | "medium" | "high" | "critical";
}

/**
 * Result of approval processing
 */
export interface ApprovalResult {
    approved: boolean;
    reason?: string;
    approvalDuration?: number;
    approvedBy?: string;
    rejectedBy?: string;
}

/**
 * User approval response
 */
export interface UserApprovalResponse {
    pendingId: string;
    approved: boolean;
    userId: string;
    reason?: string;
}

/**
 * Tool approval system integrated with event architecture.
 * Handles approval requests, user responses, and timeout management.
 */
export class ApprovalSystem {
    private readonly pendingApprovals = new Map<string, {
        toolCall: ToolCallRequest;
        pendingId: string;
        requestedAt: Date;
        timeoutId: NodeJS.Timeout;
        resolve: (result: ApprovalResult) => void;
        reject: (error: Error) => void;
    }>();

    constructor(
        private readonly logger: Logger,
    ) { }

    /**
     * Process tool approval request as blocking event
     */
    async processApprovalRequest(toolCall: ToolCallRequest): Promise<ApprovalResult> {
        const pendingId = nanoid();
        const startTime = Date.now();

        this.logger.info("[ApprovalSystem] Processing tool approval request", {
            toolCallId: toolCall.id,
            toolName: toolCall.name,
            pendingId,
            callerBotId: toolCall.callerBotId,
            conversationId: toolCall.conversationId,
            estimatedCredits: toolCall.estimatedCredits,
            riskLevel: toolCall.riskLevel,
        });

        return new Promise<ApprovalResult>((resolve, reject) => {
            // Create timeout handler
            const timeoutId = setTimeout(() => {
                this.handleApprovalTimeout(pendingId, startTime);
            }, toolCall.approvalTimeoutMs);

            // Store pending approval
            this.pendingApprovals.set(pendingId, {
                toolCall,
                pendingId,
                requestedAt: new Date(),
                timeoutId,
                resolve,
                reject,
            });

            // Create approval required event
            const approvalEvent: ToolEvent = {
                id: nanoid(),
                type: "tool/approval_required",
                timestamp: new Date(),
                source: this.createEventSource("tool-approval-system") as EventSource & { tier: 3 },
                correlationId: toolCall.id,
                data: this.createApprovalEventData(toolCall, pendingId),
                metadata: this.createApprovalEventMetadata(toolCall),
            };

            // Publish the approval event
            getEventBus().publish(approvalEvent)
                .then(result => {
                    if (!result.success) {
                        this.cleanupPendingApproval(pendingId);
                        reject(result.error || new Error("Failed to publish approval event"));
                    }
                })
                .catch(error => {
                    this.cleanupPendingApproval(pendingId);
                    reject(error);
                });
        });
    }

    /**
     * Handle user approval response
     */
    async handleUserApprovalResponse(response: UserApprovalResponse): Promise<void> {
        const pending = this.pendingApprovals.get(response.pendingId);
        if (!pending) {
            this.logger.warn("[ApprovalSystem] Received response for unknown pending approval", {
                pendingId: response.pendingId,
                userId: response.userId,
            });
            return;
        }

        const approvalDuration = Date.now() - pending.requestedAt.getTime();

        this.logger.info("[ApprovalSystem] Received user approval response", {
            pendingId: response.pendingId,
            approved: response.approved,
            userId: response.userId,
            reason: response.reason,
            approvalDuration,
            toolName: pending.toolCall.name,
        });

        // Clean up pending approval
        this.cleanupPendingApproval(response.pendingId);

        // Create result
        const result: ApprovalResult = {
            approved: response.approved,
            reason: response.reason,
            approvalDuration,
            approvedBy: response.approved ? response.userId : undefined,
            rejectedBy: response.approved ? undefined : response.userId,
        };

        // Emit appropriate event
        const eventType = response.approved ? "tool/approval_granted" : "tool/approval_rejected";
        const responseEvent: ToolEvent = {
            id: nanoid(),
            type: eventType,
            timestamp: new Date(),
            source: this.createEventSource("tool-approval-system") as EventSource & { tier: 3 },
            correlationId: pending.toolCall.id,
            data: {
                ...this.createApprovalEventData(pending.toolCall, response.pendingId),
                approvedBy: result.approvedBy,
                rejectedBy: result.rejectedBy,
                reason: response.reason,
                approvalDuration,
            },
            metadata: {
                deliveryGuarantee: "reliable",
                priority: "medium",
                userId: response.userId,
                conversationId: pending.toolCall.conversationId,
            },
        };

        await getEventBus().publish(responseEvent);

        // Resolve the promise
        pending.resolve(result);
    }

    /**
     * Get pending approvals for a conversation
     */
    getPendingApprovals(conversationId: string): Array<{
        pendingId: string;
        toolCall: ToolCallRequest;
        requestedAt: Date;
        timeoutAt: Date;
    }> {
        const pending: Array<{
            pendingId: string;
            toolCall: ToolCallRequest;
            requestedAt: Date;
            timeoutAt: Date;
        }> = [];

        for (const [pendingId, approval] of this.pendingApprovals) {
            if (approval.toolCall.conversationId === conversationId) {
                pending.push({
                    pendingId,
                    toolCall: approval.toolCall,
                    requestedAt: approval.requestedAt,
                    timeoutAt: new Date(approval.requestedAt.getTime() + approval.toolCall.approvalTimeoutMs),
                });
            }
        }

        return pending.sort((a, b) => a.requestedAt.getTime() - b.requestedAt.getTime());
    }

    /**
     * Cancel a pending approval (e.g., when conversation is stopped)
     */
    async cancelPendingApproval(pendingId: string, reason: string): Promise<void> {
        const pending = this.pendingApprovals.get(pendingId);
        if (!pending) {
            this.logger.warn("[ApprovalSystem] Tried to cancel unknown pending approval", {
                pendingId,
                reason,
            });
            return;
        }

        this.logger.info("[ApprovalSystem] Cancelling pending approval", {
            pendingId,
            reason,
            toolName: pending.toolCall.name,
            callerBotId: pending.toolCall.callerBotId,
        });

        // Clean up
        this.cleanupPendingApproval(pendingId);

        // Emit cancellation event
        const cancellationEvent: ToolEvent = {
            id: nanoid(),
            type: "tool/approval_cancelled",
            timestamp: new Date(),
            source: this.createEventSource("tool-approval-system") as EventSource & { tier: 3 },
            correlationId: pending.toolCall.id,
            data: {
                ...this.createApprovalEventData(pending.toolCall, pendingId),
                cancellationReason: reason,
            },
            metadata: {
                deliveryGuarantee: "reliable",
                priority: "medium",
                conversationId: pending.toolCall.conversationId,
            },
        };

        await getEventBus().publish(cancellationEvent);

        // Reject the promise
        pending.reject(new Error(`Approval cancelled: ${reason}`));
    }

    /**
     * Get approval system metrics
     */
    getMetrics(): {
        pendingApprovals: number;
        totalApprovalRequests: number;
        averageApprovalTime: number;
        approvalRate: number;
        timeoutRate: number;
    } {
        // TODO: Implement metrics tracking
        return {
            pendingApprovals: this.pendingApprovals.size,
            totalApprovalRequests: 0,
            averageApprovalTime: 0,
            approvalRate: 0,
            timeoutRate: 0,
        };
    }

    /**
     * Private helper methods
     */

    private handleApprovalTimeout(pendingId: string, startTime: number): void {
        const pending = this.pendingApprovals.get(pendingId);
        if (!pending) return;

        const timeoutDuration = Date.now() - startTime;

        this.logger.warn("[ApprovalSystem] Tool approval timed out", {
            pendingId,
            toolName: pending.toolCall.name,
            timeoutDuration,
            autoRejectOnTimeout: pending.toolCall.autoRejectOnTimeout,
        });

        // Clean up
        this.cleanupPendingApproval(pendingId);

        // Emit timeout event
        const timeoutEvent: ToolEvent = {
            id: nanoid(),
            type: "tool/approval_timeout",
            timestamp: new Date(),
            source: this.createEventSource("tool-approval-system") as EventSource & { tier: 3 },
            correlationId: pending.toolCall.id,
            data: {
                ...this.createApprovalEventData(pending.toolCall, pendingId),
                timeoutDuration,
                autoRejected: pending.toolCall.autoRejectOnTimeout,
            },
            metadata: {
                deliveryGuarantee: "reliable",
                priority: "high",
                conversationId: pending.toolCall.conversationId,
            },
        };

        getEventBus().publish(timeoutEvent).catch(error => {
            this.logger.error("[ApprovalSystem] Failed to emit timeout event", {
                pendingId,
                error: error instanceof Error ? error.message : String(error),
            });
        });

        // Resolve with timeout result
        const result: ApprovalResult = {
            approved: !pending.toolCall.autoRejectOnTimeout,
            reason: pending.toolCall.autoRejectOnTimeout ? "Auto-rejected on timeout" : "Timed out but keeping pending",
            approvalDuration: timeoutDuration,
        };

        if (pending.toolCall.autoRejectOnTimeout) {
            pending.resolve(result);
        } else {
            // Keep pending for manual resolution
            pending.reject(new Error("Approval timed out"));
        }
    }

    private cleanupPendingApproval(pendingId: string): void {
        const pending = this.pendingApprovals.get(pendingId);
        if (pending) {
            clearTimeout(pending.timeoutId);
            this.pendingApprovals.delete(pendingId);
        }
    }

    private createEventSource(component: string): EventSource {
        return {
            tier: "cross-cutting",
            component,
            instanceId: nanoid(),
        };
    }

    private createApprovalEventData(toolCall: ToolCallRequest, pendingId: string): ToolEventData {
        return {
            toolName: toolCall.name,
            toolCallId: toolCall.id,
            parameters: toolCall.parameters,
            pendingId,
            callerBotId: toolCall.callerBotId,
            approvalTimeoutAt: Date.now() + toolCall.approvalTimeoutMs,
        };
    }

    private createApprovalEventMetadata(toolCall: ToolCallRequest): EventMetadata {
        return {
            deliveryGuarantee: "reliable", // Approval events need reliable delivery
            priority: this.getPriorityFromRiskLevel(toolCall.riskLevel),
            conversationId: toolCall.conversationId,
            tags: [
                "tool-approval",
                toolCall.riskLevel,
                `tool-${toolCall.name}`,
                `bot-${toolCall.callerBotId}`,
            ],
        };
    }

    private getPriorityFromRiskLevel(riskLevel: string): "low" | "medium" | "high" | "critical" {
        switch (riskLevel) {
            case "critical": return "critical";
            case "high": return "high";
            case "medium": return "medium";
            case "low": return "low";
            default: return "medium";
        }
    }
}

import { type Logger } from "winston";
import { type IntelligentEvent } from "../events/eventBus.js";

/**
 * Tool Approval Manager - Human-in-the-loop approval workflows
 * 
 * Manages approval workflows for potentially dangerous or high-cost tool executions,
 * providing human oversight while maintaining system performance.
 */
export class ToolApprovalManager {
    private readonly logger: Logger;
    private readonly eventBus: any; // Will be properly typed
    private readonly pendingApprovals = new Map<string, PendingApproval>();
    private readonly approvalHandlers = new Map<string, ApprovalHandler>();
    private readonly approvalPolicies = new Map<string, ApprovalPolicy>();
    
    // Configuration
    private readonly DEFAULT_APPROVAL_TIMEOUT = 300000; // 5 minutes
    private readonly MAX_PENDING_APPROVALS = 1000;
    private readonly APPROVAL_REMINDER_INTERVAL = 60000; // 1 minute

    constructor(eventBus: any, logger?: Logger) {
        this.eventBus = eventBus;
        this.logger = logger || console as any;
        
        // Initialize default policies
        this.initializeDefaultPolicies();
        
        // Start approval reminder system
        this.startApprovalReminders();
    }

    /**
     * Process a tool approval request
     */
    async processApprovalRequest(event: IntelligentEvent): Promise<void> {
        const approvalId = this.generateApprovalId(event);
        
        try {
            // Check if approval is already pending
            if (this.pendingApprovals.has(approvalId)) {
                this.logger.debug(`[ToolApprovalManager] Approval already pending`, {
                    approvalId,
                    eventId: event.id,
                });
                return;
            }

            // Check approval policy
            const policy = this.determineApprovalPolicy(event);
            if (!policy.required) {
                // Auto-approve based on policy
                await this.publishApprovalEvent(approvalId, event, "auto_approved", "System", "Auto-approved by policy");
                return;
            }

            // Create pending approval
            const pendingApproval: PendingApproval = {
                id: approvalId,
                eventId: event.id,
                event,
                policy,
                status: "pending",
                createdAt: new Date(),
                timeoutAt: new Date(Date.now() + (policy.timeoutMs || this.DEFAULT_APPROVAL_TIMEOUT)),
                requiredApprovers: policy.requiredApprovers,
                approvals: [],
                rejections: [],
                lastReminderAt: null,
            };

            // Check capacity
            if (this.pendingApprovals.size >= this.MAX_PENDING_APPROVALS) {
                this.logger.warn(`[ToolApprovalManager] Too many pending approvals, auto-rejecting`, {
                    approvalId,
                    pendingCount: this.pendingApprovals.size,
                });
                
                await this.rejectApproval(
                    approvalId,
                    "system",
                    "System overloaded - too many pending approvals",
                );
                return;
            }

            this.pendingApprovals.set(approvalId, pendingApproval);

            this.logger.info(`[ToolApprovalManager] Created approval request`, {
                approvalId,
                eventId: event.id,
                eventType: event.type,
                policyName: policy.name,
                requiredApprovers: policy.requiredApprovers,
                timeoutAt: pendingApproval.timeoutAt,
            });

            // Publish approval required event
            await this.publishApprovalRequiredEvent(pendingApproval);

        } catch (error) {
            this.logger.error(`[ToolApprovalManager] Error processing approval request`, {
                approvalId,
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });

            // Auto-reject on error
            await this.rejectApproval(
                approvalId,
                "system",
                `Approval processing error: ${error instanceof Error ? error.message : String(error)}`,
            );
        }
    }

    /**
     * Grant approval from a user
     */
    async grantApproval(
        approvalId: string,
        approvedBy: string,
        reason?: string,
        conditions?: string[],
    ): Promise<ApprovalResult> {
        const pending = this.pendingApprovals.get(approvalId);
        if (!pending) {
            return {
                success: false,
                error: "Approval request not found or already processed",
            };
        }

        if (pending.status !== "pending") {
            return {
                success: false,
                error: `Approval already ${pending.status}`,
            };
        }

        // Check if user is authorized to approve
        if (!this.isAuthorizedApprover(approvedBy, pending.policy)) {
            return {
                success: false,
                error: "User not authorized to approve this request",
            };
        }

        // Add approval
        pending.approvals.push({
            approvedBy,
            reason: reason || "No reason provided",
            conditions: conditions || [],
            timestamp: new Date(),
        });

        this.logger.info(`[ToolApprovalManager] Approval granted`, {
            approvalId,
            approvedBy,
            reason,
            approvalsCount: pending.approvals.length,
            required: pending.policy.threshold,
        });

        // Check if we have enough approvals
        if (pending.approvals.length >= pending.policy.threshold) {
            return await this.finalizeApproval(approvalId, true);
        }

        return {
            success: true,
            message: `Approval recorded. ${pending.policy.threshold - pending.approvals.length} more approvals needed.`,
        };
    }

    /**
     * Reject approval from a user
     */
    async rejectApproval(
        approvalId: string,
        rejectedBy: string,
        reason: string,
    ): Promise<ApprovalResult> {
        const pending = this.pendingApprovals.get(approvalId);
        if (!pending) {
            return {
                success: false,
                error: "Approval request not found or already processed",
            };
        }

        if (pending.status !== "pending") {
            return {
                success: false,
                error: `Approval already ${pending.status}`,
            };
        }

        // Check if user is authorized (anyone in required approvers can reject)
        if (rejectedBy !== "system" && !this.isAuthorizedApprover(rejectedBy, pending.policy)) {
            return {
                success: false,
                error: "User not authorized to reject this request",
            };
        }

        // Add rejection
        pending.rejections.push({
            rejectedBy,
            reason,
            timestamp: new Date(),
        });

        this.logger.info(`[ToolApprovalManager] Approval rejected`, {
            approvalId,
            rejectedBy,
            reason,
        });

        // Finalize rejection
        return await this.finalizeApproval(approvalId, false);
    }

    /**
     * Get pending approval by ID
     */
    getPendingApproval(approvalId: string): PendingApproval | null {
        return this.pendingApprovals.get(approvalId) || null;
    }

    /**
     * Get all pending approvals for a user
     */
    getPendingApprovalsForUser(userId: string): PendingApproval[] {
        const userApprovals: PendingApproval[] = [];
        
        for (const approval of this.pendingApprovals.values()) {
            if (approval.status === "pending" && 
                approval.requiredApprovers.includes(userId)) {
                userApprovals.push(approval);
            }
        }
        
        return userApprovals.sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());
    }

    /**
     * Get approval statistics
     */
    getApprovalStatistics(): ApprovalStatistics {
        const pending = Array.from(this.pendingApprovals.values());
        const total = pending.length;
        const byStatus = {
            pending: pending.filter(a => a.status === "pending").length,
            approved: pending.filter(a => a.status === "approved").length,
            rejected: pending.filter(a => a.status === "rejected").length,
            timeout: pending.filter(a => a.status === "timeout").length,
        };

        // Calculate average processing time for completed approvals
        const completed = pending.filter(a => a.status !== "pending");
        const avgProcessingTime = completed.length > 0 
            ? completed.reduce((sum, a) => {
                const processingTime = (a.completedAt || new Date()).getTime() - a.createdAt.getTime();
                return sum + processingTime;
            }, 0) / completed.length
            : 0;

        return {
            total,
            byStatus,
            avgProcessingTimeMs: avgProcessingTime,
            oldestPendingAge: this.getOldestPendingAge(),
            generatedAt: new Date(),
        };
    }

    /**
     * Register an approval policy
     */
    registerApprovalPolicy(policy: ApprovalPolicy): void {
        this.approvalPolicies.set(policy.name, policy);
        
        this.logger.info(`[ToolApprovalManager] Registered approval policy`, {
            policyName: policy.name,
            requiredApprovers: policy.requiredApprovers,
            threshold: policy.threshold,
        });
    }

    /**
     * Private helper methods
     */
    private generateApprovalId(event: IntelligentEvent): string {
        return `approval_${event.id}_${Date.now()}`;
    }

    private determineApprovalPolicy(event: IntelligentEvent): ApprovalPolicy {
        // Check for explicit tool approval events
        if (event.type === "tool/approval_required") {
            const toolName = event.data?.toolName || "unknown";
            const estimatedCost = parseFloat(event.data?.estimatedCost || "0");
            
            // High-cost tools require approval
            if (estimatedCost > 10000) {
                return this.approvalPolicies.get("high_cost_tools") || this.getDefaultPolicy();
            }
            
            // Financial tools require approval
            if (toolName.includes("financial") || toolName.includes("trade")) {
                return this.approvalPolicies.get("financial_tools") || this.getDefaultPolicy();
            }
            
            // Medical tools require approval
            if (toolName.includes("medical") || toolName.includes("health")) {
                return this.approvalPolicies.get("medical_tools") || this.getDefaultPolicy();
            }
        }

        // Check security level
        if (event.securityLevel === "secret" || event.securityLevel === "confidential") {
            return this.approvalPolicies.get("high_security") || this.getDefaultPolicy();
        }

        // Human approval explicitly required
        if (event.humanApprovalRequired) {
            return this.approvalPolicies.get("explicit_approval") || this.getDefaultPolicy();
        }

        // Default - no approval required
        return this.approvalPolicies.get("auto_approve") || this.getNoApprovalPolicy();
    }

    private isAuthorizedApprover(userId: string, policy: ApprovalPolicy): boolean {
        return policy.requiredApprovers.includes(userId) || 
               policy.requiredApprovers.includes("any_admin");
    }

    private async finalizeApproval(approvalId: string, approved: boolean): Promise<ApprovalResult> {
        const pending = this.pendingApprovals.get(approvalId);
        if (!pending) {
            return {
                success: false,
                error: "Approval not found",
            };
        }

        pending.status = approved ? "approved" : "rejected";
        pending.completedAt = new Date();

        const eventType = approved ? "tool/approval_granted" : "tool/approval_rejected";
        const eventData = {
            approvalId,
            originalEventId: pending.eventId,
            toolName: pending.event.data?.toolName,
            approvals: pending.approvals,
            rejections: pending.rejections,
            finalStatus: pending.status,
            processingDuration: pending.completedAt.getTime() - pending.createdAt.getTime(),
        };

        // Publish approval result event
        await this.publishApprovalEvent(
            approvalId,
            pending.event,
            eventType,
            approved ? pending.approvals[pending.approvals.length - 1].approvedBy : pending.rejections[pending.rejections.length - 1].rejectedBy,
            approved ? "Approval granted" : "Approval rejected",
        );

        // Clean up
        this.pendingApprovals.delete(approvalId);

        const message = approved 
            ? "Approval granted successfully"
            : "Approval rejected";

        return {
            success: true,
            message,
            finalStatus: pending.status,
        };
    }

    private async publishApprovalRequiredEvent(pending: PendingApproval): Promise<void> {
        const eventData = {
            approvalId: pending.id,
            originalEventId: pending.eventId,
            toolName: pending.event.data?.toolName,
            toolArguments: pending.event.data?.toolArguments,
            estimatedCost: pending.event.data?.estimatedCost,
            requiredApprovers: pending.requiredApprovers,
            threshold: pending.policy.threshold,
            timeoutAt: pending.timeoutAt,
            policyName: pending.policy.name,
        };

        // Publish to event bus
        await this.eventBus.publishTierEvent(
            pending.event.tier,
            "tool/approval_required",
            eventData,
            {
                deliveryGuarantee: "reliable",
                priority: "high",
            },
        );
    }

    private async publishApprovalEvent(
        approvalId: string,
        originalEvent: IntelligentEvent,
        eventType: string,
        userId: string,
        reason: string,
    ): Promise<void> {
        const eventData = {
            approvalId,
            originalEventId: originalEvent.id,
            userId,
            reason,
            timestamp: new Date(),
        };

        await this.eventBus.publishTierEvent(
            originalEvent.tier,
            eventType,
            eventData,
            {
                deliveryGuarantee: "reliable",
                priority: "medium",
            },
        );
    }

    private initializeDefaultPolicies(): void {
        // Auto-approve policy (no approval required)
        this.registerApprovalPolicy({
            name: "auto_approve",
            required: false,
            requiredApprovers: [],
            threshold: 0,
            timeoutMs: 1000,
            autoRejectOnTimeout: false,
        });

        // Explicit approval policy
        this.registerApprovalPolicy({
            name: "explicit_approval",
            required: true,
            requiredApprovers: ["any_admin"],
            threshold: 1,
            timeoutMs: this.DEFAULT_APPROVAL_TIMEOUT,
            autoRejectOnTimeout: true,
        });

        // High-cost tools policy
        this.registerApprovalPolicy({
            name: "high_cost_tools",
            required: true,
            requiredApprovers: ["cost_controller", "finance_manager"],
            threshold: 1,
            timeoutMs: this.DEFAULT_APPROVAL_TIMEOUT,
            autoRejectOnTimeout: true,
        });

        // Financial tools policy
        this.registerApprovalPolicy({
            name: "financial_tools",
            required: true,
            requiredApprovers: ["portfolio_manager", "risk_analyst"],
            threshold: 1,
            timeoutMs: this.DEFAULT_APPROVAL_TIMEOUT,
            autoRejectOnTimeout: true,
        });

        // Medical tools policy
        this.registerApprovalPolicy({
            name: "medical_tools",
            required: true,
            requiredApprovers: ["medical_director", "compliance_officer"],
            threshold: 2, // Requires 2 approvals for medical
            timeoutMs: this.DEFAULT_APPROVAL_TIMEOUT,
            autoRejectOnTimeout: true,
        });

        // High security policy
        this.registerApprovalPolicy({
            name: "high_security",
            required: true,
            requiredApprovers: ["security_officer", "admin"],
            threshold: 1,
            timeoutMs: this.DEFAULT_APPROVAL_TIMEOUT,
            autoRejectOnTimeout: true,
        });
    }

    private getDefaultPolicy(): ApprovalPolicy {
        return this.approvalPolicies.get("explicit_approval")!;
    }

    private getNoApprovalPolicy(): ApprovalPolicy {
        return this.approvalPolicies.get("auto_approve")!;
    }

    private startApprovalReminders(): void {
        setInterval(async () => {
            await this.sendApprovalReminders();
            await this.processTimeouts();
        }, this.APPROVAL_REMINDER_INTERVAL);
    }

    private async sendApprovalReminders(): Promise<void> {
        const now = new Date();
        
        for (const approval of this.pendingApprovals.values()) {
            if (approval.status !== "pending") continue;
            
            const timeSinceCreated = now.getTime() - approval.createdAt.getTime();
            const timeSinceReminder = approval.lastReminderAt 
                ? now.getTime() - approval.lastReminderAt.getTime()
                : Infinity;
            
            // Send reminder if it's been 2 minutes since creation and 1 minute since last reminder
            if (timeSinceCreated > 120000 && timeSinceReminder > 60000) {
                await this.publishApprovalReminderEvent(approval);
                approval.lastReminderAt = now;
            }
        }
    }

    private async processTimeouts(): Promise<void> {
        const now = new Date();
        const timedOut: string[] = [];
        
        for (const [approvalId, approval] of this.pendingApprovals) {
            if (approval.status === "pending" && now > approval.timeoutAt) {
                timedOut.push(approvalId);
            }
        }
        
        for (const approvalId of timedOut) {
            if (this.pendingApprovals.get(approvalId)?.policy.autoRejectOnTimeout) {
                await this.rejectApproval(approvalId, "system", "Approval timeout");
            }
        }
    }

    private async publishApprovalReminderEvent(approval: PendingApproval): Promise<void> {
        const eventData = {
            approvalId: approval.id,
            toolName: approval.event.data?.toolName,
            timeRemaining: approval.timeoutAt.getTime() - Date.now(),
            requiredApprovers: approval.requiredApprovers,
            approvalsReceived: approval.approvals.length,
            approvalsRequired: approval.policy.threshold,
        };

        await this.eventBus.publishTierEvent(
            approval.event.tier,
            "tool/approval_reminder",
            eventData,
            {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
            },
        );
    }

    private getOldestPendingAge(): number | null {
        const pending = Array.from(this.pendingApprovals.values())
            .filter(a => a.status === "pending");
        
        if (pending.length === 0) return null;
        
        const oldest = pending.reduce((oldest, current) => 
            current.createdAt < oldest.createdAt ? current : oldest,
        );
        
        return Date.now() - oldest.createdAt.getTime();
    }
}

/**
 * Supporting interfaces
 */
export interface PendingApproval {
    id: string;
    eventId: string;
    event: IntelligentEvent;
    policy: ApprovalPolicy;
    status: "pending" | "approved" | "rejected" | "timeout";
    createdAt: Date;
    completedAt?: Date;
    timeoutAt: Date;
    requiredApprovers: string[];
    approvals: ApprovalGrant[];
    rejections: ApprovalRejection[];
    lastReminderAt: Date | null;
}

export interface ApprovalPolicy {
    name: string;
    required: boolean;
    requiredApprovers: string[];
    threshold: number;
    timeoutMs: number;
    autoRejectOnTimeout: boolean;
}

interface ApprovalGrant {
    approvedBy: string;
    reason: string;
    conditions: string[];
    timestamp: Date;
}

interface ApprovalRejection {
    rejectedBy: string;
    reason: string;
    timestamp: Date;
}

export interface ApprovalResult {
    success: boolean;
    message?: string;
    error?: string;
    finalStatus?: string;
}

interface ApprovalHandler {
    userId: string;
    capabilities: string[];
    handler: (approval: PendingApproval) => Promise<void>;
}

interface ApprovalStatistics {
    total: number;
    byStatus: {
        pending: number;
        approved: number;
        rejected: number;
        timeout: number;
    };
    avgProcessingTimeMs: number;
    oldestPendingAge: number | null;
    generatedAt: Date;
}
import { type ToolApprovalManager } from "../../../../services/execution/cross-cutting/approval/toolApprovalManager.js";
import { type EventPublisher } from "../../../../services/execution/shared/EventPublisher.js";
import { logger } from "../../../../services/logger.js";

export interface ApprovalScenario {
    name: string;
    description: string;
    toolRequest: ToolRequest;
    expectedOutcome: ApprovalOutcome;
    userResponses?: UserResponse[];
    policyOverrides?: PolicyOverride[];
}

export interface ToolRequest {
    toolId: string;
    action: string;
    parameters: any;
    context: {
        agentId: string;
        userId?: string;
        sensitivity: "low" | "medium" | "high" | "critical";
        reason: string;
    };
}

export interface ApprovalOutcome {
    approved: boolean;
    reason?: string;
    conditions?: string[];
    modifications?: any;
}

export interface UserResponse {
    timing: "immediate" | "delayed";
    action: "approve" | "deny" | "modify";
    modifications?: any;
    reason?: string;
}

export interface PolicyOverride {
    policy: string;
    value: any;
}

export class ApprovalWorkflowFixture {
    private approvalManager: ToolApprovalManager;
    private eventPublisher: EventPublisher;
    private pendingApprovals: Map<string, PendingApproval> = new Map();
    private approvalHistory: ApprovalRecord[] = [];

    constructor(approvalManager: ToolApprovalManager, eventPublisher: EventPublisher) {
        this.approvalManager = approvalManager;
        this.eventPublisher = eventPublisher;
        this.setupEventHandlers();
    }

    /**
     * Test a complete approval scenario
     */
    async testApprovalScenario(scenario: ApprovalScenario): Promise<ApprovalTestResult> {
        logger.info("Testing approval scenario", { scenario: scenario.name });

        // Apply policy overrides if any
        if (scenario.policyOverrides) {
            await this.applyPolicyOverrides(scenario.policyOverrides);
        }

        // Submit tool request for approval
        const requestId = await this.submitToolRequest(scenario.toolRequest);

        // Simulate user responses if provided
        if (scenario.userResponses) {
            await this.simulateUserResponses(requestId, scenario.userResponses);
        }

        // Wait for approval decision
        const result = await this.waitForApprovalDecision(requestId);

        // Verify outcome matches expectations
        const outcomeMatches = this.verifyOutcome(result, scenario.expectedOutcome);

        return {
            scenario: scenario.name,
            requestId,
            result,
            outcomeMatches,
            timeline: this.getApprovalTimeline(requestId),
            metrics: this.getApprovalMetrics(requestId),
        };
    }

    /**
     * Test automatic approval based on policies
     */
    async testAutomaticApproval(
        toolId: string,
        context: any,
    ): Promise<boolean> {
        const request: ToolRequest = {
            toolId,
            action: "execute",
            parameters: {},
            context: {
                agentId: "test-agent",
                sensitivity: "low",
                reason: "Automatic approval test",
                ...context,
            },
        };

        const requestId = await this.submitToolRequest(request);
        const result = await this.waitForApprovalDecision(requestId, 1000); // Short timeout for automatic

        return result.approved && result.automatic;
    }

    /**
     * Test approval escalation
     */
    async testEscalation(
        initialRequest: ToolRequest,
        escalationTrigger: EscalationTrigger,
    ): Promise<EscalationResult> {
        const requestId = await this.submitToolRequest(initialRequest);

        // Trigger escalation condition
        await this.triggerEscalation(requestId, escalationTrigger);

        // Wait for escalation to complete
        const escalationResult = await this.waitForEscalation(requestId);

        return {
            requestId,
            escalated: escalationResult.escalated,
            escalationLevel: escalationResult.level,
            finalApprover: escalationResult.approver,
            timeline: this.getApprovalTimeline(requestId),
        };
    }

    /**
     * Test conditional approval with runtime checks
     */
    async testConditionalApproval(
        request: ToolRequest,
        conditions: ApprovalCondition[],
    ): Promise<ConditionalApprovalResult> {
        // Set up conditional approval
        await this.setupConditionalApproval(request.toolId, conditions);

        const requestId = await this.submitToolRequest(request);

        // Execute with condition monitoring
        const result = await this.executeWithConditions(requestId);

        return {
            requestId,
            approved: result.approved,
            conditionsMet: result.conditionResults,
            violations: result.violations,
            remediation: result.remediation,
        };
    }

    /**
     * Test approval learning and adaptation
     */
    async testApprovalLearning(
        requests: ToolRequest[],
        feedbackGenerator: (result: any) => ApprovalFeedback,
    ): Promise<LearningResult> {
        const results: ApprovalDecision[] = [];
        const feedbacks: ApprovalFeedback[] = [];

        // Process multiple requests
        for (const request of requests) {
            const requestId = await this.submitToolRequest(request);
            const result = await this.waitForApprovalDecision(requestId);
            results.push(result);

            // Generate and submit feedback
            const feedback = feedbackGenerator(result);
            await this.submitFeedback(requestId, feedback);
            feedbacks.push(feedback);
        }

        // Analyze learning
        const learning = await this.analyzeApprovalLearning(results, feedbacks);

        return {
            totalRequests: requests.length,
            approvalRate: results.filter(r => r.approved).length / results.length,
            learningMetrics: learning,
            policyEvolution: await this.getPolicyEvolution(),
        };
    }

    private setupEventHandlers() {
        // Listen for approval requests
        this.eventPublisher.on("approval:requested", (event: any) => {
            this.pendingApprovals.set(event.requestId, {
                requestId: event.requestId,
                request: event.request,
                timestamp: new Date(),
                status: "pending",
            });
        });

        // Listen for approval decisions
        this.eventPublisher.on("approval:decided", (event: any) => {
            const pending = this.pendingApprovals.get(event.requestId);
            if (pending) {
                pending.status = "decided";
                pending.decision = event.decision;
                pending.decisionTime = new Date();
            }

            this.approvalHistory.push({
                requestId: event.requestId,
                request: pending?.request,
                decision: event.decision,
                timestamp: new Date(),
                duration: pending ? new Date().getTime() - pending.timestamp.getTime() : 0,
            });
        });
    }

    private async submitToolRequest(request: ToolRequest): Promise<string> {
        const requestId = `test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        await this.approvalManager.requestApproval({
            id: requestId,
            toolId: request.toolId,
            action: request.action,
            parameters: request.parameters,
            context: request.context,
            timestamp: new Date(),
        });

        return requestId;
    }

    private async simulateUserResponses(
        requestId: string,
        responses: UserResponse[],
    ) {
        for (const response of responses) {
            // Apply timing
            if (response.timing === "delayed") {
                await new Promise(resolve => setTimeout(resolve, 2000));
            }

            // Simulate user action
            switch (response.action) {
                case "approve":
                    await this.approvalManager.approveRequest(requestId, {
                        userId: "test-user",
                        reason: response.reason,
                    });
                    break;

                case "deny":
                    await this.approvalManager.denyRequest(requestId, {
                        userId: "test-user",
                        reason: response.reason,
                    });
                    break;

                case "modify":
                    await this.approvalManager.modifyRequest(requestId, {
                        userId: "test-user",
                        modifications: response.modifications,
                        reason: response.reason,
                    });
                    break;
            }
        }
    }

    private async waitForApprovalDecision(
        requestId: string,
        timeout = 10000,
    ): Promise<ApprovalDecision> {
        const startTime = Date.now();

        while (Date.now() - startTime < timeout) {
            const pending = this.pendingApprovals.get(requestId);
            if (pending?.status === "decided" && pending.decision) {
                return pending.decision;
            }

            await new Promise(resolve => setTimeout(resolve, 100));
        }

        throw new Error(`Approval decision timeout for request: ${requestId}`);
    }

    private verifyOutcome(
        actual: ApprovalDecision,
        expected: ApprovalOutcome,
    ): boolean {
        if (actual.approved !== expected.approved) return false;

        if (expected.reason && actual.reason !== expected.reason) return false;

        if (expected.conditions) {
            const actualConditions = actual.conditions || [];
            if (!expected.conditions.every(c => actualConditions.includes(c))) {
                return false;
            }
        }

        if (expected.modifications) {
            if (!this.deepEqual(actual.modifications, expected.modifications)) {
                return false;
            }
        }

        return true;
    }

    private getApprovalTimeline(requestId: string): ApprovalEvent[] {
        const events: ApprovalEvent[] = [];
        const record = this.approvalHistory.find(r => r.requestId === requestId);

        if (record) {
            events.push({
                type: "requested",
                timestamp: record.timestamp,
                details: { request: record.request },
            });

            if (record.decision) {
                events.push({
                    type: "decided",
                    timestamp: new Date(record.timestamp.getTime() + record.duration),
                    details: { decision: record.decision },
                });
            }
        }

        return events;
    }

    private getApprovalMetrics(requestId: string): ApprovalMetrics {
        const record = this.approvalHistory.find(r => r.requestId === requestId);

        if (!record) {
            return {
                decisionTime: 0,
                automatic: false,
                escalated: false,
                modified: false,
            };
        }

        return {
            decisionTime: record.duration,
            automatic: record.duration < 100, // Automatic if very fast
            escalated: record.decision?.escalated || false,
            modified: record.decision?.modifications !== undefined,
        };
    }

    private async applyPolicyOverrides(overrides: PolicyOverride[]) {
        for (const override of overrides) {
            await this.approvalManager.updatePolicy(override.policy, override.value);
        }
    }

    private async triggerEscalation(
        requestId: string,
        trigger: EscalationTrigger,
    ) {
        switch (trigger.type) {
            case "timeout":
                await new Promise(resolve => setTimeout(resolve, trigger.value));
                break;

            case "sensitivity":
                await this.approvalManager.updateRequestSensitivity(requestId, trigger.value);
                break;

            case "conflict":
                await this.approvalManager.reportConflict(requestId, trigger.value);
                break;
        }
    }

    private async waitForEscalation(requestId: string): Promise<any> {
        // Simplified - wait for escalation event
        return new Promise(resolve => {
            this.eventPublisher.once("approval:escalated", (event: any) => {
                if (event.requestId === requestId) {
                    resolve(event);
                }
            });
        });
    }

    private async setupConditionalApproval(
        toolId: string,
        conditions: ApprovalCondition[],
    ) {
        await this.approvalManager.setToolConditions(toolId, conditions);
    }

    private async executeWithConditions(requestId: string): Promise<any> {
        return this.approvalManager.executeWithConditions(requestId);
    }

    private async submitFeedback(requestId: string, feedback: ApprovalFeedback) {
        await this.approvalManager.recordFeedback(requestId, feedback);
    }

    private async analyzeApprovalLearning(
        results: ApprovalDecision[],
        feedbacks: ApprovalFeedback[],
    ): Promise<LearningMetrics> {
        // Analyze patterns in approvals and feedback
        const approvalPatterns = this.findApprovalPatterns(results);
        const feedbackTrends = this.analyzeFeedbackTrends(feedbacks);

        return {
            patterns: approvalPatterns,
            trends: feedbackTrends,
            adaptationRate: this.calculateAdaptationRate(results, feedbacks),
            accuracyImprovement: this.calculateAccuracyImprovement(feedbacks),
        };
    }

    private findApprovalPatterns(results: ApprovalDecision[]): any[] {
        // Simplified pattern detection
        return [];
    }

    private analyzeFeedbackTrends(feedbacks: ApprovalFeedback[]): any[] {
        // Simplified trend analysis
        return [];
    }

    private calculateAdaptationRate(
        results: ApprovalDecision[],
        feedbacks: ApprovalFeedback[],
    ): number {
        // Simplified calculation
        return 0.5;
    }

    private calculateAccuracyImprovement(feedbacks: ApprovalFeedback[]): number {
        // Simplified calculation
        return 0.1;
    }

    private async getPolicyEvolution(): Promise<PolicyEvolution[]> {
        // Return policy changes over time
        return [];
    }

    private deepEqual(a: any, b: any): boolean {
        return JSON.stringify(a) === JSON.stringify(b);
    }
}

// Type definitions
interface PendingApproval {
    requestId: string;
    request: ToolRequest;
    timestamp: Date;
    status: "pending" | "decided";
    decision?: ApprovalDecision;
    decisionTime?: Date;
}

interface ApprovalRecord {
    requestId: string;
    request?: ToolRequest;
    decision: ApprovalDecision;
    timestamp: Date;
    duration: number;
}

interface ApprovalDecision {
    approved: boolean;
    reason?: string;
    conditions?: string[];
    modifications?: any;
    automatic?: boolean;
    escalated?: boolean;
}

interface ApprovalTestResult {
    scenario: string;
    requestId: string;
    result: ApprovalDecision;
    outcomeMatches: boolean;
    timeline: ApprovalEvent[];
    metrics: ApprovalMetrics;
}

interface ApprovalEvent {
    type: string;
    timestamp: Date;
    details: any;
}

interface ApprovalMetrics {
    decisionTime: number;
    automatic: boolean;
    escalated: boolean;
    modified: boolean;
}

interface EscalationTrigger {
    type: "timeout" | "sensitivity" | "conflict";
    value: any;
}

interface EscalationResult {
    requestId: string;
    escalated: boolean;
    escalationLevel: number;
    finalApprover: string;
    timeline: ApprovalEvent[];
}

interface ApprovalCondition {
    type: string;
    check: (context: any) => boolean;
    message: string;
}

interface ConditionalApprovalResult {
    requestId: string;
    approved: boolean;
    conditionsMet: Map<string, boolean>;
    violations: string[];
    remediation?: string[];
}

interface ApprovalFeedback {
    correct: boolean;
    reason?: string;
    suggestedAction?: string;
}

interface LearningResult {
    totalRequests: number;
    approvalRate: number;
    learningMetrics: LearningMetrics;
    policyEvolution: PolicyEvolution[];
}

interface LearningMetrics {
    patterns: any[];
    trends: any[];
    adaptationRate: number;
    accuracyImprovement: number;
}

interface PolicyEvolution {
    timestamp: Date;
    policy: string;
    change: string;
    trigger: string;
}

/**
 * Create standard approval scenarios for testing
 */
export const STANDARD_APPROVAL_SCENARIOS: ApprovalScenario[] = [
    {
        name: "automatic-low-risk",
        description: "Automatic approval for low-risk operations",
        toolRequest: {
            toolId: "monitor_tool",
            action: "read",
            parameters: { target: "system_metrics" },
            context: {
                agentId: "monitoring-agent",
                sensitivity: "low",
                reason: "Routine monitoring",
            },
        },
        expectedOutcome: {
            approved: true,
        },
    },
    {
        name: "user-approval-required",
        description: "High-risk operation requires user approval",
        toolRequest: {
            toolId: "executor_tool",
            action: "execute",
            parameters: { command: "system_modify" },
            context: {
                agentId: "maintenance-agent",
                sensitivity: "high",
                reason: "System configuration change",
            },
        },
        expectedOutcome: {
            approved: true,
            conditions: ["user_approved"],
        },
        userResponses: [
            {
                timing: "immediate",
                action: "approve",
                reason: "Maintenance window",
            },
        ],
    },
    {
        name: "conditional-approval",
        description: "Approval with runtime conditions",
        toolRequest: {
            toolId: "data_tool",
            action: "delete",
            parameters: { scope: "user_data" },
            context: {
                agentId: "cleanup-agent",
                sensitivity: "medium",
                reason: "Data retention policy",
            },
        },
        expectedOutcome: {
            approved: true,
            conditions: ["backup_verified", "user_consent"],
        },
    },
    {
        name: "denied-critical",
        description: "Critical operation denied by policy",
        toolRequest: {
            toolId: "security_tool",
            action: "disable",
            parameters: { feature: "authentication" },
            context: {
                agentId: "test-agent",
                sensitivity: "critical",
                reason: "Testing",
            },
        },
        expectedOutcome: {
            approved: false,
            reason: "Critical security feature cannot be disabled",
        },
    },
];

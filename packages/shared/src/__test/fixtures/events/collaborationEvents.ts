/**
 * Enhanced collaboration event fixtures using the factory pattern for testing
 * real-time collaborative features, task coordination, and decision workflows.
 */

import { RunStatus } from "../../../api/types.js";
import { type RunSocketEventPayloads } from "../../../consts/socketEvents.js";
import { BranchStatus } from "../../../run/enums.js";
import { type DeferredDecisionData, RunStatusChangeReason, type RunTaskInfo } from "../../../run/types.js";
import { testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { BaseEventFactory } from "./BaseEventFactory.js";
import { type BaseEvent } from "./types.js";

// ========================================
// Core Event Types
// ========================================

interface CollaborationEvent<T> extends BaseEvent {
    event: keyof RunSocketEventPayloads;
    data: T;
}

// ========================================
// RunTaskEventFactory for task lifecycle events
// ========================================

export class RunTaskEventFactory extends BaseEventFactory<CollaborationEvent<RunTaskInfo>, RunTaskInfo> {
    protected taskCounter = 0;
    protected runCounter = 0;

    constructor() {
        super("runTask", {
            defaults: {
                runId: testValues.snowflakeId(),
                runStatus: RunStatus.InProgress,
                percentComplete: 0,
                activeBranches: [],
                subcontextUpdates: {},
            },
            validation: (data: RunTaskInfo) => {
                if (!data.runId) return "runId is required";
                if (typeof data.percentComplete !== "number" || data.percentComplete < 0 || data.percentComplete > 100) {
                    return "percentComplete must be a number between 0 and 100";
                }
                if (!Array.isArray(data.activeBranches)) return "activeBranches must be an array";
                return true;
            },
        });
    }

    get single(): CollaborationEvent<RunTaskInfo> {
        return this.create();
    }

    get sequence(): CollaborationEvent<RunTaskInfo>[] {
        const runId = testValues.snowflakeId();
        return [
            this.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 0 }),
            this.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 25 }),
            this.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 50 }),
            this.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 75 }),
            this.createTask(runId, { runStatus: RunStatus.Completed, percentComplete: 100 }),
        ];
    }

    get variants(): Record<string, CollaborationEvent<RunTaskInfo> | CollaborationEvent<RunTaskInfo>[]> {
        return {
            taskStarted: this.createTask(testValues.snowflakeId(), {
                runStatus: RunStatus.InProgress,
                percentComplete: 0,
            }),
            taskInProgress: this.createTask(testValues.snowflakeId(), {
                runStatus: RunStatus.InProgress,
                percentComplete: 50,
            }),
            taskCompleted: this.createTask(testValues.snowflakeId(), {
                runStatus: RunStatus.Completed,
                percentComplete: 100,
            }),
            taskFailed: this.createTask(testValues.snowflakeId(), {
                runStatus: RunStatus.Failed,
                runStatusChangeReason: RunStatusChangeReason.Error,
            }),
            taskPaused: this.createTask(testValues.snowflakeId(), {
                runStatus: RunStatus.Paused,
                runStatusChangeReason: RunStatusChangeReason.UserPaused,
            }),
            parallelTasks: this.createParallelTasks(),
            sequentialTasks: this.createSequentialTasks(),
            errorRecovery: this.createErrorRecoverySequence(),
        };
    }

    /**
     * Create a task event with specific state
     */
    createTask(runId: string, overrides: Partial<RunTaskInfo> = {}): CollaborationEvent<RunTaskInfo> {
        const processId = testValues.snowflakeId();
        const subroutineInstanceId = `${testValues.snowflakeId()}.${Date.now()}`;

        return this.create({
            runId,
            activeBranches: [{
                locationStack: [{
                    locationId: testValues.snowflakeId(),
                    objectId: testValues.snowflakeId(),
                    objectType: "Routine",
                    subroutineId: testValues.snowflakeId(),
                }],
                nodeStartTimeMs: Date.now(),
                processId,
                status: BranchStatus.Active,
                subroutineInstanceId,
            }],
            subcontextUpdates: {
                [subroutineInstanceId]: {
                    allInputsMap: { input1: "test_value" },
                    allOutputsMap: { output1: "result_value" },
                },
            },
            ...overrides,
        });
    }

    /**
     * Create multiple parallel tasks
     */
    createParallelTasks(): CollaborationEvent<RunTaskInfo>[] {
        const runId = testValues.snowflakeId();
        const processId1 = testValues.snowflakeId();
        const processId2 = testValues.snowflakeId();

        return [
            this.createTask(runId, {
                activeBranches: [{
                    locationStack: [{
                        locationId: testValues.snowflakeId(),
                        objectId: testValues.snowflakeId(),
                        objectType: "Routine",
                        subroutineId: testValues.snowflakeId(),
                    }],
                    nodeStartTimeMs: Date.now(),
                    processId: processId1,
                    status: BranchStatus.Active,
                    subroutineInstanceId: `${testValues.snowflakeId()}.${Date.now()}`,
                }],
                percentComplete: 25,
            }),
            this.createTask(runId, {
                activeBranches: [{
                    locationStack: [{
                        locationId: testValues.snowflakeId(),
                        objectId: testValues.snowflakeId(),
                        objectType: "Routine",
                        subroutineId: testValues.snowflakeId(),
                    }],
                    nodeStartTimeMs: Date.now(),
                    processId: processId2,
                    status: BranchStatus.Active,
                    subroutineInstanceId: `${testValues.snowflakeId()}.${Date.now()}`,
                }],
                percentComplete: 25,
            }),
        ];
    }

    /**
     * Create sequential task execution
     */
    createSequentialTasks(): CollaborationEvent<RunTaskInfo>[] {
        const runId = testValues.snowflakeId();
        const baseTask = this.createTask(runId);

        return [
            { ...baseTask, data: { ...baseTask.data, percentComplete: 0 } },
            { ...baseTask, data: { ...baseTask.data, percentComplete: 33 } },
            { ...baseTask, data: { ...baseTask.data, percentComplete: 66 } },
            { ...baseTask, data: { ...baseTask.data, percentComplete: 100, runStatus: RunStatus.Completed } },
        ];
    }

    /**
     * Create error recovery sequence
     */
    createErrorRecoverySequence(): CollaborationEvent<RunTaskInfo>[] {
        const runId = testValues.snowflakeId();

        return [
            this.createTask(runId, { percentComplete: 50 }),
            this.createTask(runId, { runStatus: RunStatus.Failed, runStatusChangeReason: RunStatusChangeReason.Error }),
            this.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 50 }),
            this.createTask(runId, { runStatus: RunStatus.Completed, percentComplete: 100 }),
        ];
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: CollaborationEvent<RunTaskInfo>,
    ): Record<string, unknown> {
        return {
            ...state,
            runStatus: event.data.runStatus,
            percentComplete: event.data.percentComplete,
            lastTaskUpdate: Date.now(),
        };
    }
}

// ========================================
// DecisionRequestEventFactory for decision workflows
// ========================================

export class DecisionRequestEventFactory extends BaseEventFactory<CollaborationEvent<DeferredDecisionData>, DeferredDecisionData> {
    protected decisionCounter = 0;

    constructor() {
        super("runTaskDecisionRequest", {
            defaults: {
                __type: "Waiting" as const,
                key: "",
                options: [],
                decisionType: "chooseOne" as const,
            },
            validation: (data: DeferredDecisionData) => {
                if (data.__type !== "Waiting") return "__type must be 'Waiting'";
                if (!data.key) return "key is required";
                if (!Array.isArray(data.options) || data.options.length === 0) return "options must be a non-empty array";
                if (!["chooseOne", "chooseMultiple"].includes(data.decisionType)) {
                    return "decisionType must be 'chooseOne' or 'chooseMultiple'";
                }
                return true;
            },
        });
    }

    get single(): CollaborationEvent<DeferredDecisionData> {
        return this.createBooleanDecision();
    }

    get sequence(): CollaborationEvent<DeferredDecisionData>[] {
        return [
            this.createBooleanDecision(),
            this.createChoiceDecision(),
            this.createInputDecision(),
            this.createApprovalDecision(),
        ];
    }

    get variants(): Record<string, CollaborationEvent<DeferredDecisionData> | CollaborationEvent<DeferredDecisionData>[]> {
        return {
            boolean: this.createBooleanDecision(),
            choice: this.createChoiceDecision(),
            multipleChoice: this.createMultipleChoiceDecision(),
            input: this.createInputDecision(),
            approval: this.createApprovalDecision(),
            timeout: this.createTimeoutDecision(),
            urgent: this.createUrgentDecision(),
            workflow: this.createDecisionWorkflow(),
        };
    }

    /**
     * Create a boolean decision (yes/no)
     */
    createBooleanDecision(message?: string): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `decision_${++this.decisionCounter}`,
            message: message || "Do you want to proceed?",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Yes", nodeData: { value: true } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "No", nodeData: { value: false } },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create a choice decision
     */
    createChoiceDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `choice_${++this.decisionCounter}`,
            message: "Select processing method",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Fast processing", nodeData: { method: "fast" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Accurate processing", nodeData: { method: "accurate" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Balanced processing", nodeData: { method: "balanced" } },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create a multiple choice decision
     */
    createMultipleChoiceDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `multi_${++this.decisionCounter}`,
            message: "Select features to enable",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Feature A", nodeData: { feature: "a" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Feature B", nodeData: { feature: "b" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Feature C", nodeData: { feature: "c" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Feature D", nodeData: { feature: "d" } },
            ],
            decisionType: "chooseMultiple",
        });
    }

    /**
     * Create an input decision
     */
    createInputDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `input_${++this.decisionCounter}`,
            message: "Enter configuration value",
            options: [
                {
                    nodeId: testValues.snowflakeId(),
                    nodeLabel: "Configuration input",
                    nodeData: {
                        type: "input",
                        inputType: "text",
                        placeholder: "Enter value...",
                        validation: { required: true, minLength: 3 },
                    },
                },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create an approval decision
     */
    createApprovalDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `approval_${++this.decisionCounter}`,
            message: "Approve deployment to production",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Approve", nodeData: { action: "approve", requiresComment: false } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Reject", nodeData: { action: "reject", requiresComment: true } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Request changes", nodeData: { action: "changes", requiresComment: true } },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create a decision with timeout
     */
    createTimeoutDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `timeout_${++this.decisionCounter}`,
            message: "Auto-timeout decision (5 minutes)",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Continue", nodeData: { default: true } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Stop", nodeData: { default: false } },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create an urgent decision
     */
    createUrgentDecision(): CollaborationEvent<DeferredDecisionData> {
        return this.create({
            key: `urgent_${++this.decisionCounter}`,
            message: "URGENT: System overload detected. Immediate action required.",
            options: [
                { nodeId: testValues.snowflakeId(), nodeLabel: "Scale up immediately", nodeData: { action: "scale", priority: "high" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Graceful shutdown", nodeData: { action: "shutdown", priority: "high" } },
                { nodeId: testValues.snowflakeId(), nodeLabel: "Monitor and wait", nodeData: { action: "monitor", priority: "medium" } },
            ],
            decisionType: "chooseOne",
        });
    }

    /**
     * Create a complete decision workflow
     */
    createDecisionWorkflow(): CollaborationEvent<DeferredDecisionData>[] {
        return [
            this.createBooleanDecision("Initialize process?"),
            this.createChoiceDecision(),
            this.createApprovalDecision(),
        ];
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: CollaborationEvent<DeferredDecisionData>,
    ): Record<string, unknown> {
        return {
            ...state,
            pendingDecision: event.data.key,
            decisionType: event.data.decisionType,
            lastDecisionRequest: Date.now(),
        };
    }
}

// ========================================
// CollaborationFlowEventFactory for multi-user workflows
// ========================================

export class CollaborationFlowEventFactory extends BaseEventFactory<CollaborationEvent<RunTaskInfo | DeferredDecisionData>, RunTaskInfo | DeferredDecisionData> {
    private taskFactory: RunTaskEventFactory;
    private decisionFactory: DecisionRequestEventFactory;

    constructor() {
        super("collaborationFlow", {
            validation: (data) => {
                // Validation is handled by individual factories
                return true;
            },
        });
        this.taskFactory = new RunTaskEventFactory();
        this.decisionFactory = new DecisionRequestEventFactory();
    }

    get single(): CollaborationEvent<RunTaskInfo | DeferredDecisionData> {
        return this.taskFactory.single;
    }

    get sequence(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        return this.createCompleteWorkflow();
    }

    get variants(): Record<string, CollaborationEvent<RunTaskInfo | DeferredDecisionData> | CollaborationEvent<RunTaskInfo | DeferredDecisionData>[]> {
        return {
            simpleWorkflow: this.createSimpleWorkflow(),
            parallelWorkflow: this.createParallelWorkflow(),
            decisionWorkflow: this.createDecisionWorkflow(),
            errorHandlingWorkflow: this.createErrorHandlingWorkflow(),
            approvalWorkflow: this.createApprovalWorkflow(),
            timeoutWorkflow: this.createTimeoutWorkflow(),
            multiUserWorkflow: this.createMultiUserWorkflow(),
            complexWorkflow: this.createComplexWorkflow(),
        };
    }

    /**
     * Create a simple sequential workflow
     */
    createSimpleWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            this.taskFactory.createTask(runId, { percentComplete: 0 }),
            this.taskFactory.createTask(runId, { percentComplete: 50 }),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a parallel execution workflow
     */
    createParallelWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            ...this.taskFactory.createParallelTasks(),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a workflow with decision points
     */
    createDecisionWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            this.taskFactory.createTask(runId, { percentComplete: 25 }),
            this.decisionFactory.createBooleanDecision("Continue processing?"),
            this.taskFactory.createTask(runId, { percentComplete: 75 }),
            this.decisionFactory.createChoiceDecision(),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create an error handling workflow
     */
    createErrorHandlingWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            this.taskFactory.createTask(runId, { percentComplete: 30 }),
            this.taskFactory.createTask(runId, { runStatus: RunStatus.Failed, runStatusChangeReason: RunStatusChangeReason.Error }),
            this.decisionFactory.createBooleanDecision("Retry the failed task?"),
            this.taskFactory.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 30 }),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create an approval workflow
     */
    createApprovalWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            this.taskFactory.createTask(runId, { percentComplete: 80 }),
            this.decisionFactory.createApprovalDecision(),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a workflow with timeout scenarios
     */
    createTimeoutWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            this.taskFactory.createTask(runId, { percentComplete: 50 }),
            this.decisionFactory.createTimeoutDecision(),
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a multi-user collaboration workflow
     */
    createMultiUserWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            // User 1 starts task
            this.taskFactory.createTask(runId, { percentComplete: 25 }),
            // User 2 makes decision
            this.decisionFactory.createChoiceDecision(),
            // User 1 continues
            this.taskFactory.createTask(runId, { percentComplete: 50 }),
            // Manager approves
            this.decisionFactory.createApprovalDecision(),
            // User 1 completes
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a complete workflow with all patterns
     */
    createCompleteWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            // Initial task
            this.taskFactory.createTask(runId, { percentComplete: 10 }),

            // First decision
            this.decisionFactory.createBooleanDecision("Initialize system?"),

            // Parallel processing
            ...this.taskFactory.createParallelTasks(),

            // Choice decision
            this.decisionFactory.createChoiceDecision(),

            // More progress
            this.taskFactory.createTask(runId, { percentComplete: 70 }),

            // Approval required
            this.decisionFactory.createApprovalDecision(),

            // Final completion
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    /**
     * Create a complex multi-stage workflow
     */
    createComplexWorkflow(): CollaborationEvent<RunTaskInfo | DeferredDecisionData>[] {
        const runId = testValues.snowflakeId();
        return [
            // Stage 1: Initialization
            this.taskFactory.createTask(runId, { percentComplete: 5 }),
            this.decisionFactory.createChoiceDecision(),

            // Stage 2: Processing (parallel)
            ...this.taskFactory.createParallelTasks(),

            // Stage 3: Validation
            this.taskFactory.createTask(runId, { percentComplete: 60 }),
            this.decisionFactory.createBooleanDecision("Validation passed?"),

            // Stage 4: Error handling (conditional)
            this.taskFactory.createTask(runId, { runStatus: RunStatus.Failed, runStatusChangeReason: RunStatusChangeReason.Error }),
            this.decisionFactory.createBooleanDecision("Retry with different parameters?"),

            // Stage 5: Recovery
            this.taskFactory.createTask(runId, { runStatus: RunStatus.InProgress, percentComplete: 60 }),

            // Stage 6: Final approval
            this.decisionFactory.createApprovalDecision(),

            // Stage 7: Deployment
            this.taskFactory.createTask(runId, { percentComplete: 90 }),
            this.decisionFactory.createUrgentDecision(),

            // Stage 8: Completion
            this.taskFactory.createTask(runId, { percentComplete: 100, runStatus: RunStatus.Completed }),
        ];
    }

    protected applyEventToState(
        state: Record<string, unknown>,
        event: CollaborationEvent<RunTaskInfo | DeferredDecisionData>,
    ): Record<string, unknown> {
        if (event.event === "runTask") {
            return this.taskFactory["applyEventToState"](state, event as CollaborationEvent<RunTaskInfo>);
        } else if (event.event === "runTaskDecisionRequest") {
            return this.decisionFactory["applyEventToState"](state, event as CollaborationEvent<DeferredDecisionData>);
        }
        return state;
    }
}

// ========================================
// Factory Instances and Main Export
// ========================================

export const runTaskEventFactory = new RunTaskEventFactory();
export const decisionRequestEventFactory = new DecisionRequestEventFactory();
export const collaborationFlowEventFactory = new CollaborationFlowEventFactory();

// ========================================
// Backward Compatibility Layer
// ========================================

// Maintain the existing collaborationEventFixtures structure for backward compatibility
export const collaborationEventFixtures = {
    runTask: {
        taskCreated: runTaskEventFactory.variants.taskStarted,
        taskInProgress: runTaskEventFactory.variants.taskInProgress,
        taskCompleted: runTaskEventFactory.variants.taskCompleted,
        taskFailed: runTaskEventFactory.variants.taskFailed,
        parallelTasks: runTaskEventFactory.variants.parallelTasks,
    },

    decisionRequest: {
        simpleDecision: decisionRequestEventFactory.variants.boolean,
        multipleChoice: decisionRequestEventFactory.variants.choice,
        inputDecision: decisionRequestEventFactory.variants.input,
        approvalDecision: decisionRequestEventFactory.variants.approval,
    },

    sequences: {
        sequentialTasks: runTaskEventFactory.variants.sequentialTasks,
        parallelExecution: runTaskEventFactory.variants.parallelTasks,
        decisionFlow: collaborationFlowEventFactory.variants.decisionWorkflow,
        multiUserFlow: collaborationFlowEventFactory.variants.multiUserWorkflow,
        errorRecoveryFlow: collaborationFlowEventFactory.variants.errorHandlingWorkflow,
        approvalWorkflow: collaborationFlowEventFactory.variants.approvalWorkflow,
    },

    // Legacy factory functions
    factories: {
        createRunTask: (runId: string, task: Partial<RunTaskInfo>) =>
            runTaskEventFactory.createTask(runId, task),

        createDecisionRequest: (runId: string, taskId: string, decision: Partial<DeferredDecisionData>) =>
            decisionRequestEventFactory.create({
                key: `${runId}_${taskId}_decision`,
                ...decision,
            }),

        createTaskUpdate: (taskId: string, updates: Partial<RunTaskInfo>) =>
            runTaskEventFactory.create({ runId: taskId, ...updates }),
    },
};

// ========================================
// Modern Factory Pattern Exports
// ========================================

/**
 * Enhanced collaboration patterns with comprehensive testing scenarios
 */
export const collaborationPatterns = {
    // Task lifecycle patterns
    taskLifecycle: {
        simple: runTaskEventFactory.createSequentialTasks(),
        parallel: runTaskEventFactory.createParallelTasks(),
        errorRecovery: runTaskEventFactory.createErrorRecoverySequence(),
    },

    // Decision patterns
    decisionPatterns: {
        booleanDecisions: decisionRequestEventFactory.createDecisionWorkflow(),
        multipleChoice: [decisionRequestEventFactory.variants.choice],
        approvalFlow: [decisionRequestEventFactory.variants.approval],
        timeoutScenarios: [decisionRequestEventFactory.variants.timeout],
    },

    // Collaboration patterns
    collaboration: {
        simple: collaborationFlowEventFactory.variants.simpleWorkflow,
        parallel: collaborationFlowEventFactory.variants.parallelWorkflow,
        multiUser: collaborationFlowEventFactory.variants.multiUserWorkflow,
        complex: collaborationFlowEventFactory.variants.complexWorkflow,
    },

    // Real-time scenarios
    realTimeScenarios: {
        synchronization: collaborationFlowEventFactory.createParallelWorkflow(),
        handoffs: collaborationFlowEventFactory.createMultiUserWorkflow(),
        errorHandling: collaborationFlowEventFactory.createErrorHandlingWorkflow(),
        timeouts: collaborationFlowEventFactory.createTimeoutWorkflow(),
    },
};

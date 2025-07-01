import { type IEventBus } from "../../../events/types.js";
import { EventTypes } from "../../../events/index.js";
import { type ISwarmContextManager } from "../../shared/SwarmContextManager.js";
import { type StepResult, type RoutineCompletionData } from "@vrooli/shared";

/**
 * RoutineEventCoordinator - Thin wrapper around EventBus for routine-specific events
 * 
 * This class provides a clean interface for routine-specific event patterns while
 * delegating all actual event handling to the existing EventBus infrastructure.
 * 
 * ## Design Principles:
 * 1. **Thin Wrapper**: No custom event logic, just routine-specific patterns
 * 2. **Type Safety**: Strongly typed event payloads for routine events
 * 3. **Context Updates**: Coordinates with SwarmContextManager for state updates
 * 4. **No Duplication**: Uses existing EventBus patterns and delivery guarantees
 * 
 * ## Key Benefits:
 * - Centralizes routine event patterns in one place
 * - Provides type-safe methods for common routine events
 * - Automatically updates context when events are emitted
 * - Maintains consistency with existing event infrastructure
 */
export class RoutineEventCoordinator {
    private readonly contextId: string;

    constructor(
        private readonly eventBus: IEventBus,
        private readonly swarmContextManager?: ISwarmContextManager,
        contextId: string,
    ) {
        this.contextId = contextId;
    }

    /**
     * Emit routine started event
     */
    async emitRoutineStarted(
        executionId: string,
        routineId: string,
        userId: string,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.ROUTINE_STARTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                routineId,
                userId,
                startTime: new Date().toISOString(),
            },
        });

        // Update context if available
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.startTime": new Date().toISOString(),
                "execution.status": "started",
            });
        }
    }

    /**
     * Emit step started event
     */
    async emitStepStarted(
        executionId: string,
        stepId: string,
        stepType: string,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.STEP_STARTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                stepId,
                stepType,
                startTime: new Date().toISOString(),
            },
        });

        // Update context with current step
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.currentStep": stepId,
                "execution.currentStepType": stepType,
                "execution.lastStepStartTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit step completed event
     */
    async emitStepCompleted(
        executionId: string,
        stepId: string,
        result: StepResult,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.STEP_COMPLETED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                stepId,
                result,
                completionTime: new Date().toISOString(),
            },
        });

        // Update context with step results
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.lastCompletedStep": stepId,
                [`execution.stepResults.${stepId}`]: result,
                "execution.lastStepCompletionTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit step failed event
     */
    async emitStepFailed(
        executionId: string,
        stepId: string,
        error: string,
        details?: Record<string, unknown>,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.STEP_FAILED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                stepId,
                error,
                details,
                failureTime: new Date().toISOString(),
            },
        });

        // Update context with failure information
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.lastFailedStep": stepId,
                [`execution.stepErrors.${stepId}`]: { error, details },
                "execution.lastFailureTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit routine completed event
     */
    async emitRoutineCompleted(
        executionId: string,
        result: RoutineCompletionData,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.ROUTINE_COMPLETED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                result,
                completionTime: new Date().toISOString(),
            },
        });

        // Update context with completion data
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.status": "completed",
                "execution.result": result,
                "execution.completionTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit routine failed event
     */
    async emitRoutineFailed(
        executionId: string,
        error: string,
        failedAtStep?: string,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.ROUTINE_FAILED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                error,
                failedAtStep,
                failureTime: new Date().toISOString(),
            },
        });

        // Update context with failure data
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.status": "failed",
                "execution.error": error,
                "execution.failedAtStep": failedAtStep,
                "execution.failureTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit variable updated event
     */
    async emitVariableUpdated(
        executionId: string,
        variableName: string,
        value: unknown,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.VARIABLE_UPDATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                variableName,
                value,
                updateTime: new Date().toISOString(),
            },
        });

        // Variables are managed directly in context
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                [`variables.${variableName}`]: value,
                "execution.lastVariableUpdate": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit resource allocated event
     */
    async emitResourceAllocated(
        executionId: string,
        resourceType: string,
        amount: number,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.RESOURCE_ALLOCATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                resourceType,
                amount,
                allocationTime: new Date().toISOString(),
            },
        });

        // Track resource allocation in context
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                [`resources.allocated.${resourceType}`]: amount,
                "resources.lastAllocationTime": new Date().toISOString(),
            });
        }
    }

    /**
     * Emit checkpoint created event
     */
    async emitCheckpointCreated(
        executionId: string,
        checkpointId: string,
        stepId: string,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.CHECKPOINT_CREATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                checkpointId,
                stepId,
                creationTime: new Date().toISOString(),
            },
        });

        // Track checkpoint in context
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                [`execution.checkpoints.${checkpointId}`]: {
                    stepId,
                    createdAt: new Date().toISOString(),
                },
                "execution.lastCheckpoint": checkpointId,
            });
        }
    }

    /**
     * Emit approval requested event
     */
    async emitApprovalRequested(
        executionId: string,
        stepId: string,
        approvalType: string,
        requestData: Record<string, unknown>,
    ): Promise<void> {
        await this.eventBus.publish({
            type: EventTypes.APPROVAL_REQUESTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                executionId,
                stepId,
                approvalType,
                requestData,
                requestTime: new Date().toISOString(),
            },
        });

        // Track approval request in context
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.pendingApproval": {
                    stepId,
                    type: approvalType,
                    requestedAt: new Date().toISOString(),
                },
                "execution.status": "awaiting_approval",
            });
        }
    }
}
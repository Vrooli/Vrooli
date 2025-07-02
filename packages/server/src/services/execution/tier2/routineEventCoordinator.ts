import { EventTypes, type RoutineCompletionData, type StepResult } from "@vrooli/shared";
import { getEventBus } from "../../events/eventBus.js";
import { type ISwarmContextManager } from "../../shared/SwarmContextManager.js";

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
        private readonly swarmContextManager?: ISwarmContextManager,
        contextId: string,
    ) {
        this.contextId = contextId;
    }

    /**
     * Emit routine started event
     */
    async emitRoutineStarted(
        swarmId: string,
        routineId: string,
        userId: string,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.STARTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        stepId: string,
        stepType: string,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.STEP_STARTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        stepId: string,
        result: StepResult,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.STEP_COMPLETED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        stepId: string,
        error: string,
        details?: Record<string, unknown>,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.STEP_FAILED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        result: RoutineCompletionData,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.COMPLETED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        error: string,
        failedAtStep?: string,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.FAILED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        variableName: string,
        value: unknown,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.ROUTINE.VARIABLE_UPDATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        resourceType: string,
        amount: number,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.RESOURCE_ALLOCATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        checkpointId: string,
        stepId: string,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.CHECKPOINT_CREATED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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
        swarmId: string,
        stepId: string,
        approvalType: string,
        requestData: Record<string, unknown>,
    ): Promise<void> {
        await getEventBus().publish({
            type: EventTypes.CHAT.TOOL_APPROVAL_REQUESTED,
            source: { tier: 2, component: "RoutineExecution" },
            data: {
                swarmId,
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

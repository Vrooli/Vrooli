import { describe, it, expect, beforeEach, vi } from "vitest";
import { RoutineEventCoordinator } from "./RoutineEventCoordinator.js";
import { type IEventBus } from "../../../events/types.js";
import { type ISwarmContextManager } from "../../shared/SwarmContextManager.js";
import { EventTypes } from "../../../events/index.js";
import { type StepResult, type RoutineCompletionData } from "@vrooli/shared";

describe("RoutineEventCoordinator", () => {
    let mockEventBus: IEventBus;
    let mockSwarmContextManager: ISwarmContextManager;
    let coordinator: RoutineEventCoordinator;
    const contextId = "test-context-456";
    const executionId = "exec-789";

    beforeEach(() => {
        // Mock event bus
        mockEventBus = {
            publish: vi.fn().mockResolvedValue(undefined),
            on: vi.fn(),
            off: vi.fn(),
        } as any;

        // Mock SwarmContextManager
        mockSwarmContextManager = {
            updateContext: vi.fn().mockResolvedValue(undefined),
            getContext: vi.fn(),
        } as any;

        // Create coordinator
        coordinator = new RoutineEventCoordinator(
            mockEventBus,
            mockSwarmContextManager,
            contextId,
        );
    });

    describe("Routine Events", () => {
        it("should emit routine started event with context update", async () => {
            const routineId = "routine-123";
            const userId = "user-456";

            await coordinator.emitRoutineStarted(executionId, routineId, userId);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.ROUTINE_STARTED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    routineId,
                    userId,
                    startTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.startTime": expect.any(String),
                    "execution.status": "started",
                },
            );
        });

        it("should emit routine completed event with result", async () => {
            const result: RoutineCompletionData = {
                outputs: { result: "success" },
                metrics: {
                    totalSteps: 5,
                    stepsCompleted: 5,
                    duration: 1000,
                },
            };

            await coordinator.emitRoutineCompleted(executionId, result);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.ROUTINE_COMPLETED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    result,
                    completionTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.status": "completed",
                    "execution.result": result,
                    "execution.completionTime": expect.any(String),
                },
            );
        });

        it("should emit routine failed event with error details", async () => {
            const error = "Execution failed";
            const failedAtStep = "step-123";

            await coordinator.emitRoutineFailed(executionId, error, failedAtStep);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.ROUTINE_FAILED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    error,
                    failedAtStep,
                    failureTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.status": "failed",
                    "execution.error": error,
                    "execution.failedAtStep": failedAtStep,
                    "execution.failureTime": expect.any(String),
                },
            );
        });
    });

    describe("Step Events", () => {
        it("should emit step started event", async () => {
            const stepId = "step-001";
            const stepType = "action";

            await coordinator.emitStepStarted(executionId, stepId, stepType);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.STEP_STARTED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    stepId,
                    stepType,
                    startTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.currentStep": stepId,
                    "execution.currentStepType": stepType,
                    "execution.lastStepStartTime": expect.any(String),
                },
            );
        });

        it("should emit step completed event with results", async () => {
            const stepId = "step-002";
            const result: StepResult = {
                success: true,
                outputs: { data: "processed" },
            };

            await coordinator.emitStepCompleted(executionId, stepId, result);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.STEP_COMPLETED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    stepId,
                    result,
                    completionTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.lastCompletedStep": stepId,
                    [`execution.stepResults.${stepId}`]: result,
                    "execution.lastStepCompletionTime": expect.any(String),
                },
            );
        });

        it("should emit step failed event with error details", async () => {
            const stepId = "step-003";
            const error = "Step execution failed";
            const details = { code: "TIMEOUT", retries: 3 };

            await coordinator.emitStepFailed(executionId, stepId, error, details);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.STEP_FAILED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    stepId,
                    error,
                    details,
                    failureTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.lastFailedStep": stepId,
                    [`execution.stepErrors.${stepId}`]: { error, details },
                    "execution.lastFailureTime": expect.any(String),
                },
            );
        });
    });

    describe("Variable and Resource Events", () => {
        it("should emit variable updated event", async () => {
            const variableName = "userName";
            const value = "John Doe";

            await coordinator.emitVariableUpdated(executionId, variableName, value);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.VARIABLE_UPDATED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    variableName,
                    value,
                    updateTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    [`variables.${variableName}`]: value,
                    "execution.lastVariableUpdate": expect.any(String),
                },
            );
        });

        it("should emit resource allocated event", async () => {
            const resourceType = "cpu";
            const amount = 2;

            await coordinator.emitResourceAllocated(executionId, resourceType, amount);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.RESOURCE_ALLOCATED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    resourceType,
                    amount,
                    allocationTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    [`resources.allocated.${resourceType}`]: amount,
                    "resources.lastAllocationTime": expect.any(String),
                },
            );
        });
    });

    describe("Checkpoint and Approval Events", () => {
        it("should emit checkpoint created event", async () => {
            const checkpointId = "checkpoint-001";
            const stepId = "step-004";

            await coordinator.emitCheckpointCreated(executionId, checkpointId, stepId);

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.CHECKPOINT_CREATED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    checkpointId,
                    stepId,
                    creationTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    [`execution.checkpoints.${checkpointId}`]: {
                        stepId,
                        createdAt: expect.any(String),
                    },
                    "execution.lastCheckpoint": checkpointId,
                },
            );
        });

        it("should emit approval requested event", async () => {
            const stepId = "step-005";
            const approvalType = "manual";
            const requestData = { action: "delete", resource: "user-data" };

            await coordinator.emitApprovalRequested(
                executionId,
                stepId,
                approvalType,
                requestData,
            );

            // Verify event was published
            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: EventTypes.APPROVAL_REQUESTED,
                source: { tier: 2, component: "RoutineExecution" },
                data: {
                    executionId,
                    stepId,
                    approvalType,
                    requestData,
                    requestTime: expect.any(String),
                },
            });

            // Verify context was updated
            expect(mockSwarmContextManager.updateContext).toHaveBeenCalledWith(
                contextId,
                {
                    "execution.pendingApproval": {
                        stepId,
                        type: approvalType,
                        requestedAt: expect.any(String),
                    },
                    "execution.status": "awaiting_approval",
                },
            );
        });
    });

    describe("Without SwarmContextManager", () => {
        it("should work without context manager", async () => {
            // Create coordinator without context manager
            const coordinatorNoContext = new RoutineEventCoordinator(
                mockEventBus,
                undefined,
                contextId,
            );

            // Should emit event without error
            await expect(
                coordinatorNoContext.emitRoutineStarted(executionId, "routine-123", "user-456"),
            ).resolves.not.toThrow();

            // Event should still be published
            expect(mockEventBus.publish).toHaveBeenCalled();
        });
    });
});
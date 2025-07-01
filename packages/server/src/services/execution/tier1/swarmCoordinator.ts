/**
 * SwarmCoordinator - Direct Tier 1 Coordination Implementation
 * 
 * This extends SwarmStateMachine to implement TierCommunicationInterface,
 * eliminating the need for the deprecated TierOneCoordinator wrapper.
 * 
 * ## Architecture Benefits:
 * - Removes unnecessary abstraction layer (TierOneCoordinator)
 * - Direct integration with SwarmContextManager for live updates
 * - Eliminates in-memory locking anti-patterns
 * - Proper distributed coordination through Redis
 * 
 * ## Key Features:
 * - Full TierCommunicationInterface implementation
 * - SwarmContextManager integration for emergent capabilities
 * - Direct Tier 1→2 delegation through TierCommunicationInterface
 * - Event-driven coordination without complex state machines
 */

import {
    type ExecutionResult,
    type ExecutionStatus,
    type RoutineExecutionInput,
    type SessionUser,
    type SwarmCoordinationInput,
    type SwarmId,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    generatePK,
    isRoutineExecutionInput,
    isSwarmCoordinationInput
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { SwarmContextManager } from "../shared/SwarmContextManager.js";
import { RoutineOrchestrator } from "../tier2/routineOrchestrator.js";
import { SwarmStateMachine } from "./swarmStateMachine.js";

/**
 * SwarmCoordinator - Direct implementation of Tier 1 coordination
 * 
 * This class combines the proven SwarmStateMachine with TierCommunicationInterface
 * to provide complete Tier 1 coordination without unnecessary wrapper layers.
 */
export class SwarmCoordinator extends SwarmStateMachine implements TierCommunicationInterface {
    private readonly routineOrchestrator: RoutineOrchestrator;

    // Track active swarms for status queries
    private readonly activeSwarms: Map<string, {
        startTime: Date;
        userId: string;
        goal: string;
    }> = new Map();

    constructor() {
        const contextManager = new SwarmContextManager();
        super(contextManager);
        this.routineOrchestrator = new RoutineOrchestrator(contextManager);

        logger.info("[SwarmCoordinator] Initialized with direct Tier 1 coordination", {
            hasContextManager: true,
            hasTier2Connection: true,
        });
    }

    /**
     * Execute a tier execution request with type-safe input validation
     * 
     * Handles both SwarmCoordinationInput (creating swarms) and 
     * RoutineExecutionInput (delegating to Tier 2)
     */
    async execute(
        request: SwarmCoordinationInput,
    ): Promise<ExecutionResult<null>> {
        logger.info("[SwarmCoordinator] Executing tier request", {
            swarmId: request.context.swarmId,
            inputType: request.input?.constructor?.name || typeof request.input,
        });

        const startTime = Date.now();

        try {
            // Type-safe routing with proper discrimination
            if (isSwarmCoordinationInput(request.input)) {
                return await this.handleSwarmCoordination(
                    request as TierExecutionRequest<SwarmCoordinationInput>,
                    startTime,
                ) as ExecutionResult<null>;

            } else if (isRoutineExecutionInput(request.input)) {
                return await this.handleRoutineExecution(
                    request as TierExecutionRequest<RoutineExecutionInput>,
                    startTime,
                ) as ExecutionResult<null>;

            } else {
                // Input type not supported by Tier 1
                const inputTypeName = request.input?.constructor?.name || typeof request.input;
                throw new Error(
                    `Unsupported input type for Tier 1: ${inputTypeName}. ` +
                    "Tier 1 supports SwarmCoordinationInput and RoutineExecutionInput only.",
                );
            }
        } catch (error) {
            logger.error("[SwarmCoordinator] Execution failed", {
                swarmId: request.context.swarmId,
                error: error instanceof Error ? error.message : String(error),
                stack: error instanceof Error ? error.stack : undefined,
            });

            return {
                success: false,
                error: {
                    code: "TIER1_EXECUTION_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    tier: "tier1",
                    type: error instanceof Error ? error.constructor.name : "Error",
                    context: {
                        inputType: request.input?.constructor?.name || typeof request.input,
                        userId: request.context.userData.id,
                        swarmId: request.context.swarmId,
                    },
                },
                result: null,
                resourcesUsed: {
                    creditsUsed: "0", // BigInt as string
                    durationMs: Date.now() - startTime,
                    stepsExecuted: 0,
                    memoryUsedMB: 0,
                },
                duration: Date.now() - startTime,
                context: request.context,
            };
        }
    }

    /**
     * Handle swarm coordination requests
     */
    private async handleSwarmCoordination(
        request: SwarmCoordinationInput,
        startTime: number,
    ): Promise<ExecutionResult<null>> {
        const swarmInput = request.input as SwarmCoordinationInput;
        const swarmId = request.context.swarmId || generatePK().toString();

        // Extract resource allocation
        const maxCredits = this.parseCreditsFromString(request.allocation.maxCredits);
        const maxTime = request.allocation.maxDurationMs;

        // Generate conversation ID for the swarm
        const conversationId = generatePK().toString();

        // Create initial swarm context through SwarmContextManager
        await this.contextManager.createContext(swarmId, {
            updatedBy: request.context.userData.id,
            resources: {
                total: {
                    credits: request.allocation.maxCredits,
                    durationMs: request.allocation.maxDurationMs,
                    memoryMB: request.allocation.maxMemoryMB,
                    concurrentExecutions: request.allocation.maxConcurrentSteps,
                },
                available: {
                    credits: request.allocation.maxCredits,
                    durationMs: request.allocation.maxDurationMs,
                    memoryMB: request.allocation.maxMemoryMB,
                    concurrentExecutions: request.allocation.maxConcurrentSteps,
                },
                allocated: [],
                usageHistory: [],
            },
            blackboard: new Map([
                ["conversationId", conversationId],
                ["goal", swarmInput.goal],
                ["userId", request.context.userData.id],
            ]),
        });

        // Start the swarm through parent class
        const initiatingUser: SessionUser = {
            id: request.context.userData.id,
            name: "User",
            hasPremium: false,
        } as SessionUser;

        await this.start(conversationId, swarmInput.goal, initiatingUser, swarmId);

        // Track active swarm
        this.activeSwarms.set(swarmId, {
            startTime: new Date(),
            userId: request.context.userData.id,
            goal: swarmInput.goal,
        });

        return {
            success: true,
            result: null,
            outputs: {
                swarmId,
                swarmName: swarmConfig.name,
                agentCount: swarmInput.availableTools?.length || 0,
                conversationId,
            },
            resourcesUsed: {
                creditsUsed: "0", // BigInt as string
                durationMs: Date.now() - startTime,
                stepsExecuted: 0,
                memoryUsedMB: 0,
                apiCalls: 0,
            },
            duration: Date.now() - startTime,
            context: request.context,
        };
    }

    /**
     * Handle routine execution requests by delegating to Tier 2
     */
    private async handleRoutineExecution(
        request: TierExecutionRequest<RoutineExecutionInput>,
        startTime: number,
    ): Promise<ExecutionResult<null>> {
        const routineInput = request.input;

        // Create properly formatted Tier 2 request - direct delegation without protocol wrapper
        const tier2Request: TierExecutionRequest<RoutineExecutionInput> = {
            context: {
                ...request.context,
                routineId: routineInput.routineId,
            },
            input: routineInput,
            allocation: request.allocation,
            options: request.options,
        };

        // Delegate to Tier 2
        const result = await this.routineOrchestrator.execute(tier2Request);

        logger.info("[SwarmCoordinator] Routine execution delegated to Tier 2", {
            routineId: routineInput.routineId,
            executionResult: result.success ? "completed" : "failed",
        });

        // Update swarm resource consumption if this is part of a swarm
        if (request.context.swarmId && result.resourcesUsed) {
            try {
                await this.contextManager.allocateResources(request.context.swarmId, {
                    entityId: request.context.swarmId,
                    entityType: "run",
                    allocated: {
                        credits: BigInt(result.resourcesUsed.creditsUsed),
                        timeoutMs: result.resourcesUsed.durationMs,
                        memoryMB: result.resourcesUsed.memoryUsedMB || 0,
                        concurrentExecutions: 0,
                    },
                    purpose: `Routine execution: ${routineInput.routineId}`,
                    priority: "medium",
                });
            } catch (error) {
                logger.warn("[SwarmCoordinator] Failed to update swarm resources", {
                    swarmId: request.context.swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        return result;
    }

    /**
     * Get execution status for a swarm
     * 
     * Retrieves current execution status including progress, metadata, and error information.
     * Uses SwarmContextManager to get unified context and execution state.
     * 
     * @param swarmId - The unique identifier for the swarm execution
     * @returns Promise resolving to detailed execution status information
     * @throws Error if context retrieval fails
     */
    async getExecutionStatus(swarmId: SwarmId): Promise<ExecutionStatus> {
        try {
            // Try to get context from SwarmContextManager
            const context = await this.contextManager.getContext(swarmId);

            if (!context) {
                return {
                    swarmId,
                    status: "failed",
                    error: {
                        code: "SWARM_NOT_FOUND",
                        message: `Swarm ${swarmId} not found`,
                    },
                };
            }

            // Map internal state to execution status
            // Note: Using string literals since BaseStates values are different from ExecutionStates enum
            const statusMap: Record<string, ExecutionStatus["status"]> = {
                "UNINITIALIZED": "pending",
                "STARTING": "running",
                "RUNNING": "running",
                "IDLE": "running",
                "PAUSED": "paused",
                "STOPPED": "completed",
                "FAILED": "failed",
                "TERMINATED": "cancelled",
            };

            const executionState = context.executionState?.currentStatus || "UNINITIALIZED";
            const activeRuns = context.executionState?.activeRuns?.length || 0;
            const metrics = context.executionState?.metrics;

            return {
                swarmId,
                status: statusMap[executionState] || "failed",
                progress: this.calculateProgressFromContext(context),
                metadata: {
                    currentPhase: this.getCurrentSagaStatus(),
                    activeRuns,
                    completedRuns: metrics?.successfulExecutions || 0,
                    swarmGoal: context.blackboard.get("goal") as string,
                },
            };

        } catch (error) {
            logger.error("[SwarmCoordinator] Failed to get execution status", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                swarmId,
                status: "failed",
                error: {
                    code: "STATUS_ERROR",
                    message: error instanceof Error ? error.message : "Unknown error",
                },
            };
        }
    }

    /**
     * Cancel an execution (swarm)
     * 
     * Gracefully stops a swarm execution, cleans up resources, and updates context.
     * Removes the swarm from active tracking and notifies the SwarmContextManager.
     * 
     * @param swarmId - The unique identifier for the swarm execution to cancel
     * @returns Promise that resolves when cancellation is complete
     * @throws Error if the swarm is not found or cancellation fails
     */
    async cancelExecution(swarmId: SwarmId): Promise<void> {
        try {
            // Stop the swarm through parent class
            await this.stop(swarmId);

            // Remove from active swarms
            this.activeSwarms.delete(swarmId);

            // Clean up context
            const context = await this.contextManager.getContext(swarmId);
            if (context) {
                // Release any child swarm resources
                const childSwarmIds = context.blackboard.get("childSwarmIds") as string[] || [];
                for (const childId of childSwarmIds) {
                    try {
                        await this.cancelExecution(childId);
                    } catch (error) {
                        logger.warn("[SwarmCoordinator] Failed to cancel child swarm", {
                            parentSwarmId: swarmId,
                            childSwarmId: childId,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

        } catch (error) {
            logger.error("[SwarmCoordinator] Failed to cancel execution", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get tier capabilities and service information
     * 
     * Returns comprehensive information about this tier's capabilities, including supported
     * input types, performance characteristics, and resource limits. Used by other tiers
     * for request routing and resource planning.
     * 
     * @returns Promise resolving to tier capabilities configuration
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: "tier1",
            supportedInputTypes: ["SwarmCoordinationInput", "RoutineExecutionInput"],
            maxConcurrency: 10,
            resourceLimits: {
                maxCredits: "unlimited",
                maxDurationMs: 3600000, // 1 hour
                maxMemoryMB: 1024,
            },
        };
    }

    /**
     * Helper to safely parse credits from string format
     */
    private parseCreditsFromString(creditsStr: string): number {
        if (creditsStr === "unlimited") {
            return Number.MAX_SAFE_INTEGER;
        }

        const parsed = parseInt(creditsStr, 10);
        if (isNaN(parsed) || parsed < 0) {
            throw new Error(`Invalid credits allocation: ${creditsStr}`);
        }

        return parsed;
    }

    /**
     * Calculate progress from context
     */
    private calculateProgressFromContext(context: any): number {
        const totalCredits = parseInt(context.resources?.total?.credits || "0");
        const availableCredits = parseInt(context.resources?.available?.credits || "0");
        const consumedCredits = totalCredits - availableCredits;

        const resourceProgress = totalCredits > 0 ? consumedCredits / totalCredits : 0;
        const metrics = context.executionState?.metrics;
        const taskProgress = metrics?.totalExecutions > 0
            ? metrics.successfulExecutions / metrics.totalExecutions
            : 0;

        return Math.min(Math.max(resourceProgress, taskProgress) * 100, 100);
    }
}

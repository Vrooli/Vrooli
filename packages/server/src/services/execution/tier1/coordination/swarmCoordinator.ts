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
 * - ResourceFlowProtocol for proper Tier 1→2 delegation
 * - Event-driven coordination without complex state machines
 */

import {
    type ExecutionId,
    type ExecutionResult,
    ExecutionStates,
    type ExecutionStatus,
    type RoutineExecutionInput,
    type SessionUser,
    type SwarmCoordinationInput,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    generatePK,
    isRoutineExecutionInput,
    isSwarmCoordinationInput
} from "@vrooli/shared";
import {
    type Logger,
} from "winston";
import { ResourceFlowProtocol } from "../../shared/ResourceFlowProtocol.js";
import { type ISwarmContextManager } from "../../shared/SwarmContextManager.js";
import { type ConversationBridge } from "../intelligence/conversationBridge.js";
import { SwarmStateMachine } from "./swarmStateMachine.js";

/**
 * SwarmCoordinator - Direct implementation of Tier 1 coordination
 * 
 * This class combines the proven SwarmStateMachine with TierCommunicationInterface
 * to provide complete Tier 1 coordination without unnecessary wrapper layers.
 */
export class SwarmCoordinator extends SwarmStateMachine implements TierCommunicationInterface {
    private readonly tier2Orchestrator: TierCommunicationInterface;
    private readonly resourceFlowProtocol: ResourceFlowProtocol;

    // Track active swarms for status queries
    private readonly activeSwarms: Map<string, {
        startTime: Date;
        userId: string;
        goal: string;
    }> = new Map();

    constructor(
        logger: Logger,
        contextManager: ISwarmContextManager,
        conversationBridge: ConversationBridge,
        tier2Orchestrator: TierCommunicationInterface,
    ) {
        super(logger, contextManager, conversationBridge);
        this.tier2Orchestrator = tier2Orchestrator;
        this.resourceFlowProtocol = new ResourceFlowProtocol(logger);

        this.logger.info("[SwarmCoordinator] Initialized with direct Tier 1 coordination", {
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
    async execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        this.logger.info("[SwarmCoordinator] Executing tier request", {
            executionId: request.context.executionId,
            inputType: request.input?.constructor?.name || typeof request.input,
        });

        const startTime = Date.now();

        try {
            // Type-safe routing with proper discrimination
            if (isSwarmCoordinationInput(request.input)) {
                return await this.handleSwarmCoordination(
                    request as TierExecutionRequest<SwarmCoordinationInput>,
                    startTime,
                ) as ExecutionResult<TOutput>;

            } else if (isRoutineExecutionInput(request.input)) {
                return await this.handleRoutineExecution(
                    request as TierExecutionRequest<RoutineExecutionInput>,
                    startTime,
                ) as ExecutionResult<TOutput>;

            } else {
                // Input type not supported by Tier 1
                const inputTypeName = request.input?.constructor?.name || typeof request.input;
                throw new Error(
                    `Unsupported input type for Tier 1: ${inputTypeName}. ` +
                    "Tier 1 supports SwarmCoordinationInput and RoutineExecutionInput only.",
                );
            }
        } catch (error) {
            this.logger.error("[SwarmCoordinator] Execution failed", {
                executionId: request.context.executionId,
                error: error instanceof Error ? error.message : String(error),
                stack: error instanceof Error ? error.stack : undefined,
            });

            return {
                executionId: request.context.executionId,
                status: "failed",
                error: {
                    code: "TIER1_EXECUTION_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    tier: "tier1",
                    type: error instanceof Error ? error.constructor.name : "Error",
                    context: {
                        inputType: request.input?.constructor?.name || typeof request.input,
                        userId: request.context.userId,
                        swarmId: request.context.swarmId,
                    },
                },
                resourceUsage: {
                    credits: 0,
                    tokens: 0,
                    duration: Date.now() - startTime,
                },
                duration: Date.now() - startTime,
            };
        }
    }

    /**
     * Handle swarm coordination requests
     */
    private async handleSwarmCoordination(
        request: TierExecutionRequest<SwarmCoordinationInput>,
        startTime: number,
    ): Promise<ExecutionResult<any>> {
        const swarmInput = request.input;
        const swarmId = request.context.executionId;

        // Extract resource allocation
        const maxCredits = this.parseCreditsFromString(request.allocation.maxCredits);
        const maxTime = request.allocation.maxDurationMs;

        // Create swarm configuration
        const swarmConfig = {
            swarmId,
            name: `Swarm: ${swarmInput.goal.substring(0, 50)}...`,
            description: swarmInput.goal,
            goal: swarmInput.goal,
            resources: {
                maxCredits,
                maxTokens: Math.floor(maxTime / 1000), // Estimate tokens from duration
                maxTime,
                tools: swarmInput.availableAgents.map(agent => ({
                    name: agent.name,
                    description: agent.capabilities.join(", "),
                })),
            },
            config: {
                model: "gpt-4",
                temperature: 0.7,
                autoApproveTools: false,
                parallelExecutionLimit: Math.min(swarmInput.availableAgents.length, 5),
            },
            userId: request.context.userId,
            organizationId: request.context.organizationId,
        };

        // Generate conversation ID for the swarm
        const conversationId = generatePK().toString();

        // Create initial swarm context through SwarmContextManager
        await this.contextManager.createContext(swarmId, {
            updatedBy: request.context.userId,
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
                ["userId", request.context.userId],
            ]),
        });

        // Start the swarm through parent class
        const initiatingUser: SessionUser = {
            id: request.context.userId,
            name: "User",
            hasPremium: false,
        } as SessionUser;

        await this.start(conversationId, swarmInput.goal, initiatingUser, swarmId);

        // Track active swarm
        this.activeSwarms.set(swarmId, {
            startTime: new Date(),
            userId: request.context.userId,
            goal: swarmInput.goal,
        });

        return {
            executionId: request.context.executionId,
            status: "completed",
            result: {
                swarmId,
                swarmName: swarmConfig.name,
                agentCount: swarmInput.availableAgents.length,
                conversationId,
            },
            resourceUsage: {
                credits: 0, // Swarm creation is free, execution will consume credits
                tokens: 0,
                duration: Date.now() - startTime,
            },
            duration: Date.now() - startTime,
        };
    }

    /**
     * Handle routine execution requests by delegating to Tier 2
     */
    private async handleRoutineExecution(
        request: TierExecutionRequest<RoutineExecutionInput>,
        startTime: number,
    ): Promise<ExecutionResult<any>> {
        const routineInput = request.input;

        // Create properly formatted Tier 2 request using ResourceFlowProtocol
        const tier2Request: TierExecutionRequest<RoutineExecutionInput> = {
            context: {
                ...request.context,
                parentExecutionId: request.context.executionId,
                routineId: routineInput.routineId,
            },
            input: routineInput,
            allocation: request.allocation,
            options: request.options,
        };

        // Delegate to Tier 2
        const result = await this.tier2Orchestrator.execute(tier2Request);

        this.logger.info("[SwarmCoordinator] Routine execution delegated to Tier 2", {
            routineId: routineInput.routineId,
            executionResult: result.status,
        });

        // Update swarm resource consumption if this is part of a swarm
        if (request.context.swarmId && result.resourceUsage) {
            try {
                await this.contextManager.allocateResources(request.context.swarmId, {
                    entityId: request.context.executionId,
                    entityType: "run",
                    allocated: {
                        credits: BigInt(result.resourceUsage.credits),
                        timeoutMs: result.resourceUsage.duration,
                        memoryMB: 0,
                        concurrentExecutions: 0,
                    },
                    purpose: `Routine execution: ${routineInput.routineId}`,
                    priority: "medium",
                });
            } catch (error) {
                this.logger.warn("[SwarmCoordinator] Failed to update swarm resources", {
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
     * @param executionId - The unique identifier for the swarm execution
     * @returns Promise resolving to detailed execution status information
     * @throws Error if context retrieval fails
     */
    async getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus> {
        try {
            // Try to get context from SwarmContextManager
            const context = await this.contextManager.getContext(executionId);

            if (!context) {
                return {
                    executionId,
                    status: "failed",
                    error: {
                        code: "SWARM_NOT_FOUND",
                        message: `Swarm ${executionId} not found`,
                    },
                };
            }

            // Map internal state to execution status
            const statusMap: Record<string, ExecutionStatus["status"]> = {
                [ExecutionStates.UNINITIALIZED]: "pending",
                [ExecutionStates.STARTING]: "running",
                [ExecutionStates.RUNNING]: "running",
                [ExecutionStates.IDLE]: "running",
                [ExecutionStates.PAUSED]: "paused",
                [ExecutionStates.STOPPED]: "completed",
                [ExecutionStates.FAILED]: "failed",
                [ExecutionStates.TERMINATED]: "cancelled",
            };

            const executionState = context.executionState?.currentStatus || ExecutionStates.UNINITIALIZED;
            const activeRuns = context.executionState?.activeRuns?.length || 0;
            const metrics = context.executionState?.metrics;

            return {
                executionId,
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
            this.logger.error("[SwarmCoordinator] Failed to get execution status", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                executionId,
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
     * @param executionId - The unique identifier for the swarm execution to cancel
     * @returns Promise that resolves when cancellation is complete
     * @throws Error if the swarm is not found or cancellation fails
     */
    async cancelExecution(executionId: ExecutionId): Promise<void> {
        try {
            // Stop the swarm through parent class
            await this.stop(executionId);

            // Remove from active swarms
            this.activeSwarms.delete(executionId);

            // Clean up context
            const context = await this.contextManager.getContext(executionId);
            if (context) {
                // Release any child swarm resources
                const childSwarmIds = context.blackboard.get("childSwarmIds") as string[] || [];
                for (const childId of childSwarmIds) {
                    try {
                        await this.cancelExecution(childId);
                    } catch (error) {
                        this.logger.warn("[SwarmCoordinator] Failed to cancel child swarm", {
                            parentSwarmId: executionId,
                            childSwarmId: childId,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

        } catch (error) {
            this.logger.error("[SwarmCoordinator] Failed to cancel execution", {
                executionId,
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
            estimatedLatency: {
                p50: 5000,  // 5 seconds
                p95: 15000, // 15 seconds  
                p99: 30000, // 30 seconds
            },
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

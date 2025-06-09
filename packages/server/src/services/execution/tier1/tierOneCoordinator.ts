import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { type TierCommunicationInterface } from "@vrooli/shared";
import { SwarmStateMachine } from "./coordination/swarmStateMachine.js";
import { ConversationBridge } from "./intelligence/conversationBridge.js";
import { TeamManager } from "./organization/teamManager.js";
import { ResourceManager } from "./organization/resourceManager.js";
import { StrategyEngine } from "./intelligence/strategyEngine.js";
import { MetacognitiveMonitor } from "./intelligence/metacognitiveMonitor.js";
import { SwarmStateStoreFactory } from "./state/swarmStateStoreFactory.js";
import { type ISwarmStateStore } from "./state/swarmStateStore.js";
import {
    type SwarmStatus,
    type Swarm,
    SwarmState,
    type SessionUser,
    nanoid,
} from "@vrooli/shared";

/**
 * Tier One Coordinator
 * 
 * Main entry point for Tier 1 coordination intelligence.
 * Manages swarm lifecycle, strategic planning, and metacognitive operations.
 */
export class TierOneCoordinator {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly tier2Orchestrator: TierCommunicationInterface;
    private readonly stateStore: ISwarmStateStore;
    private readonly swarmMachines: Map<string, SwarmStateMachine> = new Map();
    private readonly teamManager: TeamManager;
    private readonly resourceManager: ResourceManager;
    private readonly strategyEngine: StrategyEngine;
    private readonly metacognitiveMonitor: MetacognitiveMonitor;
    private readonly conversationBridge: ConversationBridge;

    constructor(logger: Logger, eventBus: EventBus, tier2Orchestrator: TierCommunicationInterface) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.tier2Orchestrator = tier2Orchestrator;
        
        // Initialize state store
        this.stateStore = SwarmStateStoreFactory.getInstance(logger);
        
        // Initialize components
        this.teamManager = new TeamManager(logger);
        this.resourceManager = new ResourceManager(logger);
        this.strategyEngine = new StrategyEngine(logger);
        this.metacognitiveMonitor = new MetacognitiveMonitor(logger, eventBus);
        this.conversationBridge = new ConversationBridge(logger);
        
        // Setup event handlers
        this.setupEventHandlers();
        
        this.logger.info("[TierOneCoordinator] Initialized");
    }

    /**
     * Starts a new swarm
     */
    async startSwarm(config: {
        swarmId: string;
        name: string;
        description: string;
        goal: string;
        resources: {
            maxCredits: number;
            maxTokens: number;
            maxTime: number;
            tools: Array<{ name: string; description: string }>;
        };
        config: {
            model: string;
            temperature: number;
            autoApproveTools: boolean;
            parallelExecutionLimit: number;
        };
        userId: string;
        organizationId?: string;
        parentSwarmId?: string; // NEW: For child swarms
    }): Promise<void> {
        this.logger.info("[TierOneCoordinator] Starting swarm", {
            swarmId: config.swarmId,
            name: config.name,
        });

        try {
            // If this is a child swarm, verify parent exists and reserve resources
            if (config.parentSwarmId) {
                const reservationResult = await this.reserveResourcesForChild(
                    config.parentSwarmId,
                    config.swarmId,
                    {
                        credits: config.resources.maxCredits,
                        tokens: config.resources.maxTokens,
                        time: config.resources.maxTime,
                    }
                );

                if (!reservationResult.success) {
                    throw new Error(`Failed to reserve resources from parent swarm: ${reservationResult.message}`);
                }
            }

            // Create swarm object
            const swarm: Swarm = {
                id: config.swarmId,
                name: config.name,
                description: config.description,
                state: SwarmState.UNINITIALIZED,
                config: {
                    maxAgents: 10,
                    minAgents: 1,
                    consensusThreshold: 0.7,
                    decisionTimeout: 300000, // 5 minutes
                    adaptationInterval: 60000, // 1 minute
                    resourceOptimization: true,
                    learningEnabled: true,
                    maxBudget: config.resources.maxCredits,
                    maxDuration: config.resources.maxTime,
                    ...config.config,
                },
                parentSwarmId: config.parentSwarmId, // NEW: Set parent relationship
                childSwarmIds: [],
                resources: {
                    allocated: {
                        credits: config.resources.maxCredits,
                        tokens: config.resources.maxTokens,
                        time: config.resources.maxTime,
                    },
                    consumed: {
                        credits: 0,
                        tokens: 0,
                        time: 0,
                    },
                    remaining: {
                        credits: config.resources.maxCredits,
                        tokens: config.resources.maxTokens,
                        time: config.resources.maxTime,
                    },
                    reservedByChildren: {
                        credits: 0,
                        tokens: 0,
                        time: 0,
                    },
                    childReservations: [],
                },
                metrics: {
                    tasksCompleted: 0,
                    tasksFailed: 0,
                    avgTaskDuration: 0,
                    resourceEfficiency: 0,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
                metadata: {
                    userId: config.userId,
                    organizationId: config.organizationId,
                    version: "2.0.0",
                    parentSwarmId: config.parentSwarmId, // Also store in metadata for easy access
                },
            };

            // Store swarm state
            await this.stateStore.createSwarm(config.swarmId, swarm);

            // Create state machine
            const stateMachine = new SwarmStateMachine(
                this.logger,
                this.eventBus,
                this.stateStore,
                this.conversationBridge,
            );

            this.swarmMachines.set(config.swarmId, stateMachine);

            // Start the swarm
            const initiatingUser = { 
                id: config.userId, 
                name: "User", 
                hasPremium: false 
            } as SessionUser;
            await stateMachine.start(config.swarmId, config.goal, initiatingUser);

            // Emit swarm started event
            await this.eventBus.publish("swarm.started", {
                swarmId: config.swarmId,
                name: config.name,
                userId: config.userId,
            });

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to start swarm", {
                swarmId: config.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Requests run execution from a swarm
     */
    async requestRunExecution(request: {
        swarmId: string;
        runId: string;
        routineVersionId: string;
        inputs: Record<string, unknown>;
        config: {
            strategy?: string;
            model: string;
            maxSteps: number;
            timeout: number;
        };
    }): Promise<void> {
        const stateMachine = this.swarmMachines.get(request.swarmId);
        if (!stateMachine) {
            throw new Error(`Swarm ${request.swarmId} not found`);
        }

        // Create run execution event for swarm to handle
        await stateMachine.handleEvent({
            type: "internal_task_assignment",
            conversationId: request.swarmId,
            sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
            payload: {
                runId: request.runId,
                routineVersionId: request.routineVersionId,
                inputs: request.inputs,
                config: request.config
            }
        });
    }

    /**
     * Gets swarm status
     */
    async getSwarmStatus(swarmId: string): Promise<{
        status: SwarmStatus;
        progress?: number;
        currentPhase?: string;
        activeRuns?: number;
        completedRuns?: number;
        errors?: string[];
    }> {
        try {
            const swarm = await this.stateStore.getSwarm(swarmId);
            if (!swarm) {
                return {
                    status: SwarmStatus.Unknown,
                    errors: ["Swarm not found"],
                };
            }

            const stateMachine = this.swarmMachines.get(swarmId);
            const currentPhase = stateMachine?.getCurrentSagaStatus();

            // Map internal state to external status
            const statusMap: Record<SwarmState, SwarmStatus> = {
                [SwarmState.UNINITIALIZED]: SwarmStatus.Pending,
                [SwarmState.STARTING]: SwarmStatus.Running,
                [SwarmState.RUNNING]: SwarmStatus.Running,
                [SwarmState.IDLE]: SwarmStatus.Running,
                [SwarmState.PAUSED]: SwarmStatus.Paused,
                [SwarmState.STOPPED]: SwarmStatus.Completed,
                [SwarmState.FAILED]: SwarmStatus.Failed,
                [SwarmState.TERMINATED]: SwarmStatus.Cancelled,
            };

            return {
                status: statusMap[swarm.state] || SwarmStatus.Unknown,
                progress: this.calculateProgress(swarm),
                currentPhase,
                activeRuns: swarm.metrics?.tasksCompleted || 0,
                completedRuns: swarm.metrics?.tasksCompleted || 0,
                errors: swarm.errors,
            };

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to get swarm status", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                status: SwarmStatus.Unknown,
                errors: [error instanceof Error ? error.message : "Unknown error"],
            };
        }
    }

    /**
     * Cancels a swarm
     */
    async cancelSwarm(swarmId: string, userId: string, reason?: string): Promise<void> {
        const stateMachine = this.swarmMachines.get(swarmId);
        if (!stateMachine) {
            throw new Error(`Swarm ${swarmId} not found`);
        }

        // Get swarm info before stopping for cleanup
        const swarm = await this.stateStore.getSwarm(swarmId);

        await stateMachine.stop(swarmId);
        
        // If this was a child swarm, release resources back to parent
        if (swarm?.parentSwarmId) {
            await this.releaseResourcesFromChild(swarm.parentSwarmId, swarmId);
        }

        // Cancel any child swarms
        if (swarm?.childSwarmIds && swarm.childSwarmIds.length > 0) {
            for (const childId of swarm.childSwarmIds) {
                try {
                    await this.cancelSwarm(childId, userId, `Parent swarm ${swarmId} cancelled`);
                } catch (error) {
                    this.logger.warn(`[TierOneCoordinator] Failed to cancel child swarm ${childId}`, {
                        parentSwarmId: swarmId,
                        childSwarmId: childId,
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            }
        }
        
        // Remove from active machines
        this.swarmMachines.delete(swarmId);
        
        // Emit cancellation event
        await this.eventBus.publish("swarm.cancelled", {
            swarmId,
            userId,
            reason,
        });
    }

    /**
     * Reserves resources for a child swarm
     */
    async reserveResourcesForChild(
        parentSwarmId: string,
        childSwarmId: string,
        reservation: { credits: number; tokens: number; time: number }
    ): Promise<{ success: boolean; message?: string }> {
        try {
            const swarm = await this.stateStore.getSwarm(parentSwarmId);
            if (!swarm) {
                return { success: false, message: `Parent swarm ${parentSwarmId} not found` };
            }

            // Check if parent has enough remaining resources
            const available = {
                credits: swarm.resources.remaining.credits - swarm.resources.reservedByChildren.credits,
                tokens: swarm.resources.remaining.tokens - swarm.resources.reservedByChildren.tokens,
                time: swarm.resources.remaining.time - swarm.resources.reservedByChildren.time,
            };

            if (reservation.credits > available.credits ||
                reservation.tokens > available.tokens ||
                reservation.time > available.time) {
                return {
                    success: false,
                    message: `Insufficient resources. Available: ${JSON.stringify(available)}, Requested: ${JSON.stringify(reservation)}`
                };
            }

            // Add reservation
            swarm.resources.reservedByChildren.credits += reservation.credits;
            swarm.resources.reservedByChildren.tokens += reservation.tokens;
            swarm.resources.reservedByChildren.time += reservation.time;

            swarm.resources.childReservations.push({
                childSwarmId,
                reserved: reservation,
                createdAt: new Date(),
            });

            swarm.childSwarmIds.push(childSwarmId);
            swarm.updatedAt = new Date();

            // Update state store
            await this.stateStore.updateSwarm(parentSwarmId, swarm);

            // Emit reservation event
            await this.eventBus.publish("swarm.resource.reserved", {
                parentSwarmId,
                childSwarmId,
                reservation,
            });

            this.logger.info(`[TierOneCoordinator] Reserved resources for child swarm`, {
                parentSwarmId,
                childSwarmId,
                reservation,
            });

            return { success: true };

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to reserve resources", {
                parentSwarmId,
                childSwarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return { success: false, message: "Internal error during resource reservation" };
        }
    }

    /**
     * Releases resources from a child swarm back to parent
     */
    async releaseResourcesFromChild(
        parentSwarmId: string,
        childSwarmId: string
    ): Promise<{ success: boolean; message?: string }> {
        try {
            const swarm = await this.stateStore.getSwarm(parentSwarmId);
            if (!swarm) {
                return { success: false, message: `Parent swarm ${parentSwarmId} not found` };
            }

            // Find and remove the child reservation
            const reservationIndex = swarm.resources.childReservations.findIndex(
                r => r.childSwarmId === childSwarmId
            );

            if (reservationIndex === -1) {
                return { success: false, message: `No reservation found for child swarm ${childSwarmId}` };
            }

            const reservation = swarm.resources.childReservations[reservationIndex];

            // Release the reserved resources
            swarm.resources.reservedByChildren.credits -= reservation.reserved.credits;
            swarm.resources.reservedByChildren.tokens -= reservation.reserved.tokens;
            swarm.resources.reservedByChildren.time -= reservation.reserved.time;

            // Remove reservation record
            swarm.resources.childReservations.splice(reservationIndex, 1);

            // Remove from child list
            const childIndex = swarm.childSwarmIds.indexOf(childSwarmId);
            if (childIndex > -1) {
                swarm.childSwarmIds.splice(childIndex, 1);
            }

            swarm.updatedAt = new Date();

            // Update state store
            await this.stateStore.updateSwarm(parentSwarmId, swarm);

            // Emit release event
            await this.eventBus.publish("swarm.resource.released", {
                parentSwarmId,
                childSwarmId,
                released: reservation.reserved,
            });

            this.logger.info(`[TierOneCoordinator] Released resources from child swarm`, {
                parentSwarmId,
                childSwarmId,
                released: reservation.reserved,
            });

            return { success: true };

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to release resources", {
                parentSwarmId,
                childSwarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return { success: false, message: "Internal error during resource release" };
        }
    }

    /**
     * Shuts down the coordinator
     */
    async shutdown(): Promise<void> {
        this.logger.info("[TierOneCoordinator] Shutting down");
        
        // Stop all active swarms
        for (const [swarmId, stateMachine] of this.swarmMachines) {
            try {
                await stateMachine.requestStop("shutdown");
            } catch (error) {
                this.logger.error("[TierOneCoordinator] Error stopping swarm during shutdown", {
                    swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        
        this.swarmMachines.clear();
    }

    /**
     * Private helper methods
     */
    private setupEventHandlers(): void {
        // Handle run completion events from Tier 2
        this.eventBus.on("run.completed", async (event) => {
            const { swarmId, runId } = event.data;
            const stateMachine = this.swarmMachines.get(swarmId);
            if (stateMachine) {
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId: swarmId,
                    sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                    payload: { type: "run_completed", runId }
                });
            }
        });

        // Handle resource alerts
        this.eventBus.on("resources.low", async (event) => {
            const { swarmId } = event.data;
            const stateMachine = this.swarmMachines.get(swarmId);
            if (stateMachine) {
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId: swarmId,
                    sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                    payload: { type: "resource_alert", ...event.data }
                });
            }
        });

        // Handle metacognitive insights
        this.eventBus.on("metacognitive.insight", async (event) => {
            const { swarmId } = event.data;
            const stateMachine = this.swarmMachines.get(swarmId);
            if (stateMachine) {
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId: swarmId,
                    sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                    payload: { type: "metacognitive_insight", ...event.data }
                });
            }
        });

        // Handle child swarm completion events
        this.eventBus.on("swarm.completed", async (event) => {
            const { swarmId } = event.data;
            const swarm = await this.stateStore.getSwarm(swarmId);
            
            // If this completed swarm has a parent, notify parent and release resources
            if (swarm?.parentSwarmId) {
                const parentStateMachine = this.swarmMachines.get(swarm.parentSwarmId);
                if (parentStateMachine) {
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: swarm.parentSwarmId,
                        sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                        payload: { 
                            type: "child_swarm_completed", 
                            childSwarmId: swarmId,
                            completedAt: new Date().toISOString()
                        }
                    });
                }
                
                // Release resources back to parent
                await this.releaseResourcesFromChild(swarm.parentSwarmId, swarmId);
            }
        });

        // Handle child swarm failure events
        this.eventBus.on("swarm.failed", async (event) => {
            const { swarmId, error } = event.data;
            const swarm = await this.stateStore.getSwarm(swarmId);
            
            // If this failed swarm has a parent, notify parent and release resources
            if (swarm?.parentSwarmId) {
                const parentStateMachine = this.swarmMachines.get(swarm.parentSwarmId);
                if (parentStateMachine) {
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: swarm.parentSwarmId,
                        sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                        payload: { 
                            type: "child_swarm_failed", 
                            childSwarmId: swarmId,
                            error,
                            failedAt: new Date().toISOString()
                        }
                    });
                }
                
                // Release resources back to parent
                await this.releaseResourcesFromChild(swarm.parentSwarmId, swarmId);
            }
        });

        // Handle resource reservation events
        this.eventBus.on("swarm.resource.reserved", async (event) => {
            const { parentSwarmId, childSwarmId, reservation } = event.data;
            this.logger.info("[TierOneCoordinator] Child swarm resources reserved", {
                parentSwarmId,
                childSwarmId,
                reservation,
            });
        });

        // Handle resource release events  
        this.eventBus.on("swarm.resource.released", async (event) => {
            const { parentSwarmId, childSwarmId, released } = event.data;
            this.logger.info("[TierOneCoordinator] Child swarm resources released", {
                parentSwarmId,
                childSwarmId,
                released,
            });
        });
    }

    private calculateProgress(swarm: Swarm): number {
        // Calculate progress based on resource consumption and task completion
        const resourceProgress = swarm.resources.consumed.credits / swarm.resources.allocated.maxCredits;
        const timeProgress = swarm.resources.consumed.time / swarm.resources.allocated.maxTime;
        
        return Math.min(Math.max(resourceProgress, timeProgress) * 100, 100);
    }
}
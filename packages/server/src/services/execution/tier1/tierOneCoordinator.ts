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
    }): Promise<void> {
        this.logger.info("[TierOneCoordinator] Starting swarm", {
            swarmId: config.swarmId,
            name: config.name,
        });

        try {
            // Create swarm object
            const swarm: Swarm = {
                id: config.swarmId,
                state: SwarmState.UNINITIALIZED,
                config: {
                    name: config.name,
                    description: config.description,
                    goal: config.goal,
                    ...config.config,
                },
                team: {
                    agents: [],
                    capabilities: [],
                    activeMembers: 0,
                },
                resources: {
                    allocated: config.resources,
                    consumed: {
                        credits: 0,
                        tokens: 0,
                        time: 0,
                    },
                    remaining: { ...config.resources },
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

        await stateMachine.stop(swarmId);
        
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
    }

    private calculateProgress(swarm: Swarm): number {
        // Calculate progress based on resource consumption and task completion
        const resourceProgress = swarm.resources.consumed.credits / swarm.resources.allocated.maxCredits;
        const timeProgress = swarm.resources.consumed.time / swarm.resources.allocated.maxTime;
        
        return Math.min(Math.max(resourceProgress, timeProgress) * 100, 100);
    }
}
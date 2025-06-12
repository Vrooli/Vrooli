import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import { 
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ExecutionResult,
    type ExecutionId,
    type ExecutionStatus,
    type TierCapabilities,
    type RoutineExecutionInput,
} from "@vrooli/shared";
import { SwarmStateMachine } from "./coordination/swarmStateMachine.js";
import { ConversationBridge } from "./intelligence/conversationBridge.js";
import { TeamManager } from "./organization/teamManager.js";
import { ResourceManager } from "./organization/resourceManager.js";
import { StrategyEngine } from "./intelligence/strategyEngine.js";
import { MetacognitiveMonitorAdapter as MetacognitiveMonitor } from "../monitoring/adapters/MetacognitiveMonitorAdapter.js";
import { SwarmStateStoreFactory } from "./state/swarmStateStoreFactory.js";
import { type ISwarmStateStore } from "./state/swarmStateStore.js";
import {
    type SwarmStatus,
    type Swarm,
    SwarmState,
    type SessionUser,
    generatePK,
    generatePublicId,
    BotConfig,
    ChatConfig,
} from "@vrooli/shared";
import { DbProvider } from "../../../db/provider.js";
import { PrismaChatStore } from "../../../services/conversation/chatStore.js";
import { type BotParticipant } from "../../../services/conversation/types.js";

/**
 * Tier One Coordinator
 * 
 * Main entry point for Tier 1 coordination intelligence.
 * Manages swarm lifecycle, strategic planning, and metacognitive operations.
 */
export class TierOneCoordinator implements TierCommunicationInterface {
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
    private readonly chatStore: PrismaChatStore;
    private readonly creationLocks: Map<string, Promise<void>> = new Map(); // Simple in-memory lock

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
        this.chatStore = new PrismaChatStore();
        
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
        leaderBotId?: string; // Optional bot ID, defaults to Valyxa
    }): Promise<void> {
        this.logger.info("[TierOneCoordinator] Starting swarm", {
            swarmId: config.swarmId,
            name: config.name,
        });

        // Simple distributed lock to prevent concurrent creation
        const existingLock = this.creationLocks.get(config.swarmId);
        if (existingLock) {
            this.logger.warn("[TierOneCoordinator] Swarm creation already in progress, waiting", {
                swarmId: config.swarmId,
            });
            await existingLock;
            // Check if swarm was created by the other process
            const swarm = await this.stateStore.getSwarm(config.swarmId);
            if (swarm) {
                this.logger.info("[TierOneCoordinator] Swarm created by concurrent process", {
                    swarmId: config.swarmId,
                });
                return;
            }
        }

        // Create a lock for this swarm creation
        const creationPromise = this.performSwarmCreation(config);
        this.creationLocks.set(config.swarmId, creationPromise);
        
        try {
            await creationPromise;
        } finally {
            // Always clean up the lock
            this.creationLocks.delete(config.swarmId);
        }
    }

    /**
     * Performs the actual swarm creation logic
     */
    private async performSwarmCreation(config: {
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
        parentSwarmId?: string;
        leaderBotId?: string;
    }): Promise<void> {
        let resourcesReserved = false;
        let conversationId: string | null = null;
        
        try {
            // Validate resource configuration
            if (config.resources.maxCredits <= 0 || 
                config.resources.maxTokens <= 0 || 
                config.resources.maxTime <= 0) {
                throw new Error("Invalid resource configuration: all limits must be positive");
            }

            // Validate swarmId format
            if (!config.swarmId || typeof config.swarmId !== 'string') {
                throw new Error("Invalid swarmId: must be a non-empty string");
            }

            // Check if swarm already exists (idempotency)
            const existingSwarm = await this.stateStore.getSwarm(config.swarmId);
            if (existingSwarm) {
                // Check if chat exists and return existing swarm
                const existingConversationId = existingSwarm.metadata?.conversationId;
                if (existingConversationId) {
                    const prisma = DbProvider.get();
                    const existingChat = await prisma.chat.findUnique({
                        where: { id: BigInt(existingConversationId) }
                    });
                    if (existingChat) {
                        // Swarm already initialized, return success
                        this.logger.info("[TierOneCoordinator] Swarm already exists, returning success", {
                            swarmId: config.swarmId,
                            conversationId: existingConversationId,
                        });
                        return;
                    }
                }
                // If we get here, swarm exists but chat doesn't - continue with initialization
                this.logger.warn("[TierOneCoordinator] Swarm exists but chat missing, reinitializing", {
                    swarmId: config.swarmId,
                });
            }
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
                resourcesReserved = true;
            }

            // Generate numeric ID for conversation/chat
            conversationId = generatePK().toString();
            
            // Fetch the leader bot user (use provided ID or default to Valyxa)
            const leaderBotId = config.leaderBotId || SEEDED_PUBLIC_IDS.Valyxa;
            const prisma = DbProvider.get();
            
            // Use transaction for atomic chat creation
            const { chat, botUser, leaderBot } = await prisma.$transaction(async (tx) => {
                // Fetch bot user
                const botUser = await tx.user.findUnique({
                    where: { id: leaderBotId },
                    select: {
                        id: true,
                        name: true,
                        handle: true,
                        isBot: true,
                        botSettings: true,
                    },
                });

                if (!botUser || !botUser.isBot) {
                    throw new Error(`Bot user not found or is not a bot: ${leaderBotId}`);
                }
                
                // Validate bot has settings
                if (!botUser.botSettings) {
                    throw new Error(`Bot user ${leaderBotId} has no bot settings configured`);
                }
                
                // Validate bot has required capabilities
                const botConfig = BotConfig.parse(botUser, this.logger);
                if (!botConfig.model) {
                    throw new Error(`Bot ${leaderBotId} has no model configured`);
                }

                // Create the chat in the database
                const chat = await tx.chat.create({
                    data: {
                        id: BigInt(conversationId),
                        publicId: generatePublicId(),
                        creators: {
                            create: {
                                user: { connect: { id: config.userId } }
                            }
                        },
                        config: ChatConfig.default().export() as any,
                        labels: ["swarm", config.name],
                    },
                });

                // Add the bot as a participant
                await tx.chat_participants.create({
                    data: {
                        chat: { connect: { id: chat.id } },
                        user: { connect: { id: botUser.id } },
                    },
                });

                // Create BotParticipant from the bot user
                const leaderBot: BotParticipant = {
                    id: botUser.id,
                    name: botUser.name,
                    config: botConfig, // Use already parsed config
                    meta: { role: "leader" },
                };

                return { chat, botUser, leaderBot };
            });

            // Initialize conversation state in the chat store
            const conversationState = {
                id: conversationId,
                config: ChatConfig.default().export(),
                participants: [leaderBot],
                availableTools: [],
                initialLeaderSystemMessage: "",
                teamConfig: undefined,
            };
            
            await this.chatStore.saveState(conversationId, conversationState);
            
            // Force immediate write to ensure state is persisted with timeout
            const saveSuccess = await this.chatStore.finalizeSave(5000);
            if (!saveSuccess) {
                throw new Error("Failed to persist conversation state within timeout");
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
                    conversationId: conversationId, // Store conversation ID mapping
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

            // Start the swarm with the conversationId
            const initiatingUser = { 
                id: config.userId, 
                name: "User", 
                hasPremium: false 
            } as SessionUser;
            await stateMachine.start(conversationId, config.goal, initiatingUser);

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
            
            // Cleanup: Release resources if they were reserved
            if (resourcesReserved && config.parentSwarmId) {
                try {
                    await this.releaseResourcesFromChild(config.parentSwarmId, config.swarmId);
                    this.logger.info("[TierOneCoordinator] Released reserved resources after failure", {
                        parentSwarmId: config.parentSwarmId,
                        swarmId: config.swarmId,
                    });
                } catch (cleanupError) {
                    this.logger.error("[TierOneCoordinator] Failed to release resources during cleanup", {
                        parentSwarmId: config.parentSwarmId,
                        swarmId: config.swarmId,
                        error: cleanupError instanceof Error ? cleanupError.message : String(cleanupError),
                    });
                }
            }
            
            // Try to cleanup orphaned chat if conversationId was generated
            if (conversationId) {
                try {
                    const prisma = DbProvider.get();
                    await prisma.chat.delete({
                        where: { id: BigInt(conversationId) }
                    });
                    this.logger.info("[TierOneCoordinator] Cleaned up orphaned chat", {
                        conversationId,
                        swarmId: config.swarmId,
                    });
                } catch (cleanupError) {
                    this.logger.error("[TierOneCoordinator] Failed to cleanup orphaned chat", {
                        conversationId,
                        swarmId: config.swarmId,
                        error: cleanupError instanceof Error ? cleanupError.message : String(cleanupError),
                    });
                }
                
                // Note: Chat store cleanup would require access to the cached store instance
                // For now, the cache will expire naturally
            }
            
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

        // Get the swarm to find the conversationId and resource allocation
        const swarm = await this.stateStore.getSwarm(request.swarmId);
        if (!swarm) {
            throw new Error(`Swarm state not found for ${request.swarmId}`);
        }
        
        const conversationId = swarm.metadata?.conversationId || request.swarmId;

        try {
            // Create run execution event for swarm coordination
            await stateMachine.handleEvent({
                type: "internal_task_assignment",
                conversationId,
                sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                payload: {
                    runId: request.runId,
                    routineVersionId: request.routineVersionId,
                    inputs: request.inputs,
                    config: request.config
                }
            });

            // NEW: Also delegate to TierTwo through the standardized interface
            const tier2Request: TierExecutionRequest<RoutineExecutionInput> = {
                context: {
                    executionId: request.runId,
                    swarmId: request.swarmId,
                    userId: swarm.metadata?.userId || "system",
                    organizationId: swarm.metadata?.organizationId,
                },
                input: {
                    routineId: request.routineVersionId,
                    parameters: request.inputs,
                    workflow: {
                        steps: [], // Will be loaded by TierTwo from routineVersionId
                        dependencies: [],
                        parallelBranches: []
                    }
                },
                allocation: {
                    maxCredits: swarm.resources.remaining.credits,
                    maxTokens: swarm.resources.remaining.tokens,
                    maxTime: swarm.resources.remaining.time,
                },
                options: {
                    strategy: request.config.strategy || "conversational",
                    model: request.config.model,
                    maxSteps: request.config.maxSteps,
                    timeout: request.config.timeout,
                }
            };

            // Delegate to Tier 2 for actual run execution
            const result = await this.tier2Orchestrator.execute(tier2Request);
            
            this.logger.info("[TierOneCoordinator] Run execution delegated to Tier 2", {
                runId: request.runId,
                executionResult: result.status,
            });

            // Update swarm resource consumption based on execution result
            if (result.resourceUsage) {
                swarm.resources.consumed.credits += result.resourceUsage.credits;
                swarm.resources.consumed.tokens += result.resourceUsage.tokens;
                swarm.resources.consumed.time += result.resourceUsage.duration;
                
                swarm.resources.remaining.credits -= result.resourceUsage.credits;
                swarm.resources.remaining.tokens -= result.resourceUsage.tokens;
                swarm.resources.remaining.time -= result.resourceUsage.duration;
                
                await this.stateStore.updateSwarm(request.swarmId, swarm);
            }

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to execute run", {
                runId: request.runId,
                swarmId: request.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            
            // Notify swarm of execution failure
            await stateMachine.handleEvent({
                type: "internal_status_update",
                conversationId,
                sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                payload: { 
                    type: "run_failed", 
                    runId: request.runId,
                    error: error instanceof Error ? error.message : String(error)
                }
            });
            
            throw error;
        }
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

    // TierCommunicationInterface implementation

    /**
     * Execute a tier execution request
     */
    async execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>> {
        this.logger.info("[TierOneCoordinator] Executing tier request", {
            contextType: typeof request.input,
        });

        try {
            // Route based on input type/context
            if (this.isSwarmStartRequest(request.input)) {
                const swarmInput = request.input as any;
                await this.startSwarm({
                    swarmId: request.context.executionId,
                    name: swarmInput.goal || "Swarm Execution",
                    description: swarmInput.goal || "AI Swarm Task",
                    goal: swarmInput.goal,
                    resources: {
                        maxCredits: request.allocation.maxCredits,
                        maxTokens: request.allocation.maxTokens,
                        maxTime: request.allocation.maxTime,
                        tools: swarmInput.availableAgents?.map((agent: any) => ({
                            name: agent.name,
                            description: agent.capabilities?.join(", ") || ""
                        })) || []
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.7,
                        autoApproveTools: false,
                        parallelExecutionLimit: 3
                    },
                    userId: request.context.userId || "system",
                    organizationId: request.context.organizationId,
                });

                return {
                    executionId: request.context.executionId,
                    status: "completed",
                    result: { swarmId: request.context.executionId } as TOutput,
                    resourceUsage: {
                        credits: 0,
                        tokens: 0,
                        duration: 0
                    },
                    duration: 0
                };
            } else {
                throw new Error(`Unsupported input type for Tier 1: ${typeof request.input}`);
            }
        } catch (error) {
            this.logger.error("[TierOneCoordinator] Execution failed", {
                error: error instanceof Error ? error.message : String(error),
            });
            
            return {
                executionId: request.context.executionId,
                status: "failed",
                error: {
                    code: "EXECUTION_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error"
                },
                resourceUsage: {
                    credits: 0,
                    tokens: 0,
                    duration: 0
                },
                duration: 0
            };
        }
    }

    /**
     * Get execution status for a swarm
     */
    async getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus> {
        try {
            const swarmStatus = await this.getSwarmStatus(executionId);
            
            // Map SwarmStatus to ExecutionStatus
            const statusMap: Record<string, ExecutionStatus["status"]> = {
                "Pending": "pending",
                "Running": "running", 
                "Paused": "paused",
                "Completed": "completed",
                "Failed": "failed",
                "Cancelled": "cancelled",
                "Unknown": "failed"
            };

            return {
                executionId,
                status: statusMap[swarmStatus.status] || "failed",
                progress: swarmStatus.progress,
                metadata: {
                    currentPhase: swarmStatus.currentPhase,
                    activeRuns: swarmStatus.activeRuns,
                    completedRuns: swarmStatus.completedRuns,
                    errors: swarmStatus.errors
                }
            };
        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to get execution status", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });
            
            return {
                executionId,
                status: "failed",
                error: {
                    code: "STATUS_ERROR",
                    message: error instanceof Error ? error.message : "Unknown error"
                }
            };
        }
    }

    /**
     * Cancel an execution (swarm)
     */
    async cancelExecution(executionId: ExecutionId): Promise<void> {
        try {
            await this.cancelSwarm(executionId, "system", "Cancelled via tier interface");
        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to cancel execution", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get tier capabilities
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: "tier1",
            supportedInputTypes: ["SwarmCoordinationInput"],
            maxConcurrency: 10,
            estimatedLatency: {
                p50: 5000,  // 5 seconds
                p95: 15000, // 15 seconds  
                p99: 30000  // 30 seconds
            },
            resourceLimits: {
                maxCredits: "unlimited",
                maxDurationMs: 3600000, // 1 hour
                maxMemoryMB: 1024
            }
        };
    }

    /**
     * Helper to determine if input is a swarm start request
     */
    private isSwarmStartRequest(input: unknown): boolean {
        return typeof input === "object" && 
               input !== null && 
               "goal" in input;
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
    private async getConversationIdForSwarm(swarmId: string): Promise<string> {
        const swarm = await this.stateStore.getSwarm(swarmId);
        if (!swarm) {
            this.logger.warn(`[TierOneCoordinator] Swarm not found: ${swarmId}, using swarmId as conversationId`);
            return swarmId;
        }
        return swarm.metadata?.conversationId || swarmId;
    }
    
    private setupEventHandlers(): void {
        // Handle run completion events from Tier 2
        this.eventBus.on("run.completed", async (event) => {
            const { swarmId, runId } = event.data;
            const stateMachine = this.swarmMachines.get(swarmId);
            if (stateMachine) {
                const conversationId = await this.getConversationIdForSwarm(swarmId);
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId,
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
                const conversationId = await this.getConversationIdForSwarm(swarmId);
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId,
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
                const conversationId = await this.getConversationIdForSwarm(swarmId);
                await stateMachine.handleEvent({
                    type: "internal_status_update",
                    conversationId,
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
                    const parentConversationId = await this.getConversationIdForSwarm(swarm.parentSwarmId);
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: parentConversationId,
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
                    const parentConversationId = await this.getConversationIdForSwarm(swarm.parentSwarmId);
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: parentConversationId,
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
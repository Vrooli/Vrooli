import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { BaseComponent } from "../shared/BaseComponent.js";
// Socket events now handled through unified event system
import {
    type ExecutionId, type ExecutionResult, type ExecutionStatus, type RoutineExecutionInput,
    type SwarmCoordinationInput, type TierCapabilities, type TierCommunicationInterface,
    type TierExecutionRequest, isRoutineExecutionInput, isSwarmCoordinationInput, SEEDED_PUBLIC_IDS,
} from "@vrooli/shared";
import { EventTypes } from "../../events/index.js";
import { SwarmStateMachine } from "./coordination/swarmStateMachine.js";
import { ResourceManager } from "./organization/resourceManager.js";
import { TeamManager } from "./organization/teamManager.js";
// All intelligence functionality now provided by emergent agents - see docs/architecture/execution/emergent-capabilities/
import {
    type ExecutionState,
    type SessionUser,
    type Swarm,
    type SwarmStatus,
    type TeamFormation,
    BotConfig,
    ChatConfig,
    ExecutionStates,
    generatePK,
    generatePublicId,
} from "@vrooli/shared";
import { DbProvider } from "../../../db/provider.js";
import { PrismaChatStore } from "../../../services/conversation/chatStore.js";
import { type BotParticipant } from "../../../services/conversation/types.js";
import { type ISwarmContextManager, SwarmContextManager } from "../shared/SwarmContextManager.js";
import { type UnifiedSwarmContext } from "../shared/UnifiedSwarmContext.js";
import { type ConversationBridge, createConversationBridge } from "./intelligence/conversationBridge.js";

/**
 * Tier One Coordinator
 * 
 * Main entry point for Tier 1 coordination intelligence.
 * Manages swarm lifecycle, strategic planning, and metacognitive operations.
 */
export class TierOneCoordinator extends BaseComponent implements TierCommunicationInterface {
    private readonly tier2Orchestrator: TierCommunicationInterface;
    private readonly swarmMachines: Map<string, SwarmStateMachine> = new Map();
    private readonly teamManager: TeamManager;
    private readonly resourceManager: ResourceManager;
    // Intelligence components removed - functionality provided by emergent agents
    private readonly chatStore: PrismaChatStore;
    private readonly conversationBridge: ConversationBridge;
    private readonly creationLocks: Map<string, Promise<void>> = new Map(); // Simple in-memory lock
    private readonly contextManager: ISwarmContextManager; // Primary state management via SwarmContextManager

    constructor(
        logger: Logger,
        eventBus: EventBus,
        tier2Orchestrator: TierCommunicationInterface,
        contextManager?: ISwarmContextManager, // Optional for backward compatibility during migration
    ) {
        super(logger, eventBus, "TierOneCoordinator");
        this.tier2Orchestrator = tier2Orchestrator;

        // Initialize modern SwarmContextManager if provided, otherwise create default instance
        if (contextManager) {
            this.contextManager = contextManager;
            this.logger.info("[TierOneCoordinator] Using provided SwarmContextManager");
        } else {
            // Create default SwarmContextManager for backward compatibility
            this.contextManager = new SwarmContextManager();
            this.logger.info("[TierOneCoordinator] Created default SwarmContextManager");
        }

        // Initialize minimal components
        this.teamManager = new TeamManager(logger);
        this.resourceManager = new ResourceManager(logger);
        this.chatStore = new PrismaChatStore();
        this.conversationBridge = createConversationBridge(logger);

        // Setup event handlers
        this.setupEventHandlers();

        this.logger.info("[TierOneCoordinator] Initialized with modern architecture", {
            hasContextManager: true,
            contextManagerType: contextManager ? "provided" : "default",
            modernArchitecture: true,
        });
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
            const context = await this.contextManager.getContext(config.swarmId);
            if (context) {
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
            if (!config.swarmId || typeof config.swarmId !== "string") {
                throw new Error("Invalid swarmId: must be a non-empty string");
            }

            // Check if swarm already exists (idempotency)
            const existingContext = await this.contextManager.getContext(config.swarmId);
            if (existingContext) {
                // Check if chat exists and return existing swarm
                const existingConversationId = existingContext.blackboard.get("conversationId") as string;
                if (existingConversationId) {
                    const prisma = DbProvider.get();
                    const existingChat = await prisma.chat.findUnique({
                        where: { id: BigInt(existingConversationId) },
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
                    },
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
                                user: { connect: { id: config.userId } },
                            },
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

            // Initialize unified swarm context for emergent capabilities
            // This replaces the old swarm creation - everything is now in the context
            await this.initializeSwarmContext(config.swarmId, config, conversationId);

            // Emit initial swarm state through unified event system
            await this.publishUnifiedEvent(EventTypes.STATE_SWARM_UPDATED, {
                entityType: "swarm",
                entityId: config.swarmId,
                newState: ExecutionStates.UNINITIALIZED,
                message: "Swarm initialization in progress",
            }, {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
                conversationId,
            });

            // Emit initial resource allocation through unified event system
            await this.publishUnifiedEvent(EventTypes.RESOURCE_SWARM_UPDATED, {
                entityType: "swarm",
                entityId: config.swarmId,
                resourceType: "credits",
                allocated: config.resources.maxCredits.toString(),
                consumed: "0",
                remaining: config.resources.maxCredits.toString(),
            }, {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
                conversationId,
            });

            // Create state machine with SwarmContextManager
            const stateMachine = new SwarmStateMachine(
                this.logger,
                this.contextManager,
                this.conversationBridge,
            );

            this.swarmMachines.set(config.swarmId, stateMachine);

            // Start the swarm with the conversationId and swarmId
            const initiatingUser = {
                id: config.userId,
                name: "User",
                hasPremium: false,
            } as SessionUser;
            await stateMachine.start(conversationId, config.goal, initiatingUser, config.swarmId);

            // Emit swarm started event using unified event system
            await this.publishUnifiedEvent(EventTypes.TEAM_FORMED, {
                swarmId: config.swarmId,
                name: config.name,
                userId: config.userId,
                goal: config.goal,
                resources: config.resources,
            }, {
                deliveryGuarantee: "reliable",
                priority: "high",
                tags: ["swarm", "coordination"],
                userId: config.userId,
                conversationId: config.swarmId,
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
                        where: { id: BigInt(conversationId) },
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

        // Get the context to find the conversationId
        const context = await this.contextManager.getContext(request.swarmId);
        if (!context) {
            throw new Error(`Swarm context not found for ${request.swarmId}`);
        }

        const conversationId = context.blackboard.get("conversationId") as string || request.swarmId;

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
                    config: request.config,
                },
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
                        parallelBranches: [],
                    },
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
                },
            };

            // Delegate to Tier 2 for actual run execution
            const result = await this.tier2Orchestrator.execute(tier2Request);

            this.logger.info("[TierOneCoordinator] Run execution delegated to Tier 2", {
                runId: request.runId,
                executionResult: result.status,
            });

            // Update swarm resource consumption based on execution result
            if (result.resourceUsage) {
                // Allocate resources used by the run
                await this.contextManager.allocateResources(request.swarmId, {
                    entityId: request.runId,
                    entityType: "run",
                    allocated: {
                        credits: BigInt(result.resourceUsage.credits),
                        timeoutMs: result.resourceUsage.duration,
                        memoryMB: 0, // Not tracked at this level
                        concurrentExecutions: 0,
                    },
                    purpose: `Run execution: ${request.runId}`,
                    priority: "medium",
                });
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
                    error: error instanceof Error ? error.message : String(error),
                },
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
            const context = await this.contextManager.getContext(swarmId);
            if (!context) {
                return {
                    status: SwarmStatus.Unknown,
                    errors: ["Swarm context not found"],
                };
            }

            const stateMachine = this.swarmMachines.get(swarmId);
            const currentPhase = stateMachine?.getCurrentSagaStatus();

            // Map internal state to external status
            const statusMap: Record<ExecutionState, SwarmStatus> = {
                [ExecutionStates.UNINITIALIZED]: SwarmStatus.Pending,
                [ExecutionStates.STARTING]: SwarmStatus.Running,
                [ExecutionStates.RUNNING]: SwarmStatus.Running,
                [ExecutionStates.IDLE]: SwarmStatus.Running,
                [ExecutionStates.PAUSED]: SwarmStatus.Paused,
                [ExecutionStates.STOPPED]: SwarmStatus.Completed,
                [ExecutionStates.FAILED]: SwarmStatus.Failed,
                [ExecutionStates.TERMINATED]: SwarmStatus.Cancelled,
            };

            const executionState = context.executionState.currentStatus as ExecutionState;
            const activeRuns = context.executionState.activeRuns?.length || 0;
            const metrics = context.executionState.metrics;
            
            return {
                status: statusMap[executionState] || SwarmStatus.Unknown,
                progress: this.calculateProgressFromContext(context),
                currentPhase,
                activeRuns,
                completedRuns: metrics?.successfulExecutions || 0,
                errors: [], // Errors would be in context events/blackboard
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

        // Get context info before stopping for cleanup
        const context = await this.contextManager.getContext(swarmId);

        await stateMachine.stop(swarmId);

        // If this was a child swarm, release resources back to parent
        const parentSwarmId = context?.blackboard.get("parentSwarmId") as string;
        if (parentSwarmId) {
            await this.releaseResourcesFromChild(parentSwarmId, swarmId);
        }

        // Cancel any child swarms
        const childSwarmIds = context?.blackboard.get("childSwarmIds") as string[] || [];
        if (childSwarmIds.length > 0) {
            for (const childId of childSwarmIds) {
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
        await this.publishUnifiedEvent(
            "swarm.cancelled",
            {
                swarmId,
                userId,
                reason,
            },
            {
                userId,
                priority: "high",
                deliveryGuarantee: "reliable",
            },
        );
    }

    /**
     * Reserves resources for a child swarm
     */
    async reserveResourcesForChild(
        parentSwarmId: string,
        childSwarmId: string,
        reservation: { credits: number; tokens: number; time: number },
    ): Promise<{ success: boolean; message?: string }> {
        try {
            // Allocate resources from parent to child using contextManager
            const allocation = await this.contextManager.allocateResources(parentSwarmId, {
                entityId: childSwarmId,
                entityType: "swarm",
                allocated: {
                    credits: BigInt(reservation.credits),
                    timeoutMs: reservation.time,
                    memoryMB: 0, // Not tracked at this level
                    concurrentExecutions: 0,
                },
                purpose: `Child swarm allocation: ${childSwarmId}`,
                priority: "high",
            });

            // Track child swarm relationship
            const context = await this.contextManager.getContext(parentSwarmId);
            if (context) {
                const childSwarmIds = context.blackboard.get("childSwarmIds") as string[] || [];
                if (!childSwarmIds.includes(childSwarmId)) {
                    childSwarmIds.push(childSwarmId);
                    await this.contextManager.updateContext(parentSwarmId, {
                        blackboard: new Map([...context.blackboard, ["childSwarmIds", childSwarmIds]]),
                    }, "Add child swarm");
                }
            }

            // Emit reservation event
            await this.publishUnifiedEvent(
                "swarm.resource.reserved",
                {
                    parentSwarmId,
                    childSwarmId,
                    reservation,
                },
                {
                    conversationId: parentSwarmId,
                    priority: "medium",
                    deliveryGuarantee: "reliable",
                },
            );

            this.logger.info("[TierOneCoordinator] Reserved resources for child swarm", {
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
        childSwarmId: string,
    ): Promise<{ success: boolean; message?: string }> {
        try {
            // Find and release the allocation for the child swarm
            const context = await this.contextManager.getContext(parentSwarmId);
            if (!context) {
                return { success: false, message: `Parent swarm context ${parentSwarmId} not found` };
            }

            // Find the allocation for this child swarm
            const childAllocation = context.resources.allocated.find(
                alloc => alloc.entityId === childSwarmId && alloc.entityType === "swarm",
            );

            if (childAllocation) {
                // Release the allocation
                await this.contextManager.releaseResources(parentSwarmId, childAllocation.id);
            }

            // Remove from child list in blackboard
            const childSwarmIds = context.blackboard.get("childSwarmIds") as string[] || [];
            const childIndex = childSwarmIds.indexOf(childSwarmId);
            if (childIndex > -1) {
                childSwarmIds.splice(childIndex, 1);
                await this.contextManager.updateContext(parentSwarmId, {
                    blackboard: new Map([...context.blackboard, ["childSwarmIds", childSwarmIds]]),
                }, "Remove child swarm");
            }

            // Emit release event
            await this.publishUnifiedEvent(
                "swarm.resource.released",
                {
                    parentSwarmId,
                    childSwarmId,
                    released: childAllocation?.allocated || { credits: 0, tokens: 0, time: 0 },
                },
                {
                    conversationId: parentSwarmId,
                    priority: "medium",
                    deliveryGuarantee: "reliable",
                },
            );

            this.logger.info("[TierOneCoordinator] Released resources from child swarm", {
                parentSwarmId,
                childSwarmId,
                released: childAllocation?.allocated,
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
     * Execute a tier execution request with type-safe input validation
     */
    async execute<TInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        this.logger.info("[TierOneCoordinator] Executing tier request", {
            executionId: request.context.executionId,
            inputType: request.input?.constructor?.name || typeof request.input,
        });

        const startTime = Date.now();

        try {
            // Type-safe routing with proper discrimination
            if (isSwarmCoordinationInput(request.input)) {
                const swarmInput = request.input as SwarmCoordinationInput;

                // Extract resource allocation with type safety
                const maxCredits = this.parseCreditsFromString(request.allocation.maxCredits);
                const maxTokens = Math.floor(request.allocation.maxDurationMs / 1000); // Estimate tokens from duration
                const maxTime = request.allocation.maxDurationMs;

                await this.startSwarm({
                    swarmId: request.context.executionId,
                    name: `Swarm: ${swarmInput.goal.substring(0, 50)}...`,
                    description: swarmInput.goal,
                    goal: swarmInput.goal,
                    resources: {
                        maxCredits,
                        maxTokens,
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
                });

                return {
                    executionId: request.context.executionId,
                    status: "completed",
                    result: {
                        swarmId: request.context.executionId,
                        swarmName: `Swarm: ${swarmInput.goal.substring(0, 50)}...`,
                        agentCount: swarmInput.availableAgents.length,
                    } as TOutput,
                    resourceUsage: {
                        credits: 0, // Swarm creation is free, execution will consume credits
                        tokens: 0,
                        duration: Date.now() - startTime,
                    },
                    duration: Date.now() - startTime,
                };

            } else if (isRoutineExecutionInput(request.input)) {
                // Delegate routine execution to Tier 2
                const routineInput = request.input as RoutineExecutionInput;

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

                return await this.tier2Orchestrator.execute(tier2Request) as ExecutionResult<TOutput>;

            } else {
                // Input type not supported by Tier 1
                const inputTypeName = request.input?.constructor?.name || typeof request.input;
                throw new Error(
                    `Unsupported input type for Tier 1: ${inputTypeName}. ` +
                    "Tier 1 supports SwarmCoordinationInput and RoutineExecutionInput only.",
                );
            }
        } catch (error) {
            this.logger.error("[TierOneCoordinator] Execution failed", {
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
                "Unknown": "failed",
            };

            return {
                executionId,
                status: statusMap[swarmStatus.status] || "failed",
                progress: swarmStatus.progress,
                metadata: {
                    currentPhase: swarmStatus.currentPhase,
                    activeRuns: swarmStatus.activeRuns,
                    completedRuns: swarmStatus.completedRuns,
                    errors: swarmStatus.errors,
                },
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
                    message: error instanceof Error ? error.message : "Unknown error",
                },
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
                p99: 30000,  // 30 seconds
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
        const context = await this.contextManager.getContext(swarmId);
        if (!context) {
            this.logger.warn(`[TierOneCoordinator] Context not found: ${swarmId}, using swarmId as conversationId`);
            return swarmId;
        }
        return context.blackboard.get("conversationId") as string || swarmId;
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
                    payload: { type: "run_completed", runId },
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
                    payload: { type: "resource_alert", ...event.data },
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
                    payload: { type: "metacognitive_insight", ...event.data },
                });
            }
        });

        // Handle child swarm completion events
        this.eventBus.on("swarm.completed", async (event) => {
            const { swarmId } = event.data;
            const context = await this.contextManager.getContext(swarmId);
            const parentSwarmId = context?.blackboard.get("parentSwarmId") as string;

            // If this completed swarm has a parent, notify parent and release resources
            if (parentSwarmId) {
                const parentStateMachine = this.swarmMachines.get(parentSwarmId);
                if (parentStateMachine) {
                    const parentConversationId = await this.getConversationIdForSwarm(parentSwarmId);
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: parentConversationId,
                        sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                        payload: {
                            type: "child_swarm_completed",
                            childSwarmId: swarmId,
                            completedAt: new Date().toISOString(),
                        },
                    });
                }

                // Release resources back to parent
                await this.releaseResourcesFromChild(parentSwarmId, swarmId);
            }
        });

        // Handle child swarm failure events
        this.eventBus.on("swarm.failed", async (event) => {
            const { swarmId, error } = event.data;
            const context = await this.contextManager.getContext(swarmId);
            const parentSwarmId = context?.blackboard.get("parentSwarmId") as string;

            // If this failed swarm has a parent, notify parent and release resources
            if (parentSwarmId) {
                const parentStateMachine = this.swarmMachines.get(parentSwarmId);
                if (parentStateMachine) {
                    const parentConversationId = await this.getConversationIdForSwarm(parentSwarmId);
                    await parentStateMachine.handleEvent({
                        type: "internal_status_update",
                        conversationId: parentConversationId,
                        sessionUser: { id: "system", name: "System", hasPremium: false } as SessionUser,
                        payload: {
                            type: "child_swarm_failed",
                            childSwarmId: swarmId,
                            error,
                            failedAt: new Date().toISOString(),
                        },
                    });
                }

                // Release resources back to parent
                await this.releaseResourcesFromChild(parentSwarmId, swarmId);
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

    private calculateProgressFromContext(context: UnifiedSwarmContext): number {
        // Calculate progress based on resource consumption from context
        const totalCredits = parseInt(context.resources.total.credits || "0");
        const availableCredits = parseInt(context.resources.available.credits || "0");
        const consumedCredits = totalCredits - availableCredits;
        
        const resourceProgress = totalCredits > 0 ? consumedCredits / totalCredits : 0;
        const metrics = context.executionState.metrics;
        const taskProgress = metrics?.totalExecutions > 0 
            ? metrics.successfulExecutions / metrics.totalExecutions 
            : 0;

        return Math.min(Math.max(resourceProgress, taskProgress) * 100, 100);
    }

    /**
     * Initialize unified swarm context for emergent capabilities
     * 
     * This creates the data-driven context that enables agents to modify swarm behavior
     * through configuration changes rather than code changes.
     */
    private async initializeSwarmContext(
        swarmId: string,
        config: {
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
        },
        conversationId: string,
    ): Promise<void> {

        try {
            this.logger.debug("[TierOneCoordinator] Initializing swarm context for emergent capabilities", {
                swarmId,
                name: config.name,
            });

            // Create initial context configuration that enables emergent capabilities
            const initialContext = {
                // Identity and basic metadata
                updatedBy: config.userId,

                // Resource management based on swarm allocation
                resources: {
                    total: {
                        credits: config.resources.maxCredits.toString(),
                        durationMs: config.resources.maxTime,
                        memoryMB: 1024, // Default memory allocation
                        concurrentExecutions: config.config.parallelExecutionLimit,
                        models: {
                            [config.config.model]: {
                                tokensPerMinute: config.resources.maxTokens,
                                maxConcurrentRequests: config.config.parallelExecutionLimit,
                                costPerToken: "0.01", // Default cost estimation
                            },
                        },
                        tools: config.resources.tools.reduce((acc, tool) => {
                            acc[tool.name] = {
                                creditsPerUse: "10", // Default cost per tool use
                                avgExecutionTimeMs: 5000, // Default execution time
                                memoryRequirementMB: 64, // Default memory requirement
                            };
                            return acc;
                        }, {} as any),
                    },
                    allocated: [],
                    available: {
                        credits: config.resources.maxCredits.toString(),
                        durationMs: config.resources.maxTime,
                        memoryMB: 1024,
                        concurrentExecutions: config.config.parallelExecutionLimit,
                        models: {
                            [config.config.model]: {
                                tokensPerMinute: config.resources.maxTokens,
                                maxConcurrentRequests: config.config.parallelExecutionLimit,
                                costPerToken: "0.01",
                            },
                        },
                        tools: config.resources.tools.reduce((acc, tool) => {
                            acc[tool.name] = {
                                creditsPerUse: "10",
                                avgExecutionTimeMs: 5000,
                                memoryRequirementMB: 64,
                            };
                            return acc;
                        }, {} as any),
                    },
                    usageHistory: [],
                },

                // Policies that agents can modify for emergent capabilities
                policy: {
                    security: {
                        permissions: {
                            "swarm_leader": {
                                canExecuteTools: config.resources.tools.map(t => t.name),
                                canAccessResources: ["credits", "tokens", "memory", "time"],
                                canModifyContext: ["blackboard.*", "execution.agents.*"],
                                maxResourceAllocation: {
                                    maxCredits: Math.floor(config.resources.maxCredits * 0.8).toString(),
                                    maxDurationMs: Math.floor(config.resources.maxTime * 0.8),
                                    maxMemoryMB: 800,
                                    maxConcurrentSteps: config.config.parallelExecutionLimit,
                                },
                            },
                            "swarm_agent": {
                                canExecuteTools: config.resources.tools.filter(t =>
                                    !t.name.includes("admin") && !t.name.includes("system"),
                                ).map(t => t.name),
                                canAccessResources: ["credits", "memory"],
                                canModifyContext: ["blackboard.items.*"],
                                maxResourceAllocation: {
                                    maxCredits: Math.floor(config.resources.maxCredits * 0.3).toString(),
                                    maxDurationMs: Math.floor(config.resources.maxTime * 0.3),
                                    maxMemoryMB: 256,
                                    maxConcurrentSteps: 1,
                                },
                            },
                        },
                        scanning: {
                            enabledScanners: ["pii", "malware", "injection"],
                            blockOnViolation: true,
                            alertingThresholds: {
                                "pii": 3,
                                "malware": 1,
                                "injection": 1,
                            },
                        },
                        toolApproval: {
                            requireApprovalForTools: config.config.autoApproveTools ? [] : config.resources.tools.map(t => t.name),
                            autoApproveForRoles: config.config.autoApproveTools ? ["swarm_leader", "swarm_agent"] : ["swarm_leader"],
                            approvalTimeoutMs: 300000, // 5 minutes
                        },
                    },
                    resource: {
                        allocation: {
                            strategy: "elastic" as const,
                            tierAllocationRatios: {
                                tier1ToTier2: 0.8, // 80% of swarm resources to routine execution
                                tier2ToTier3: 0.6, // 60% of routine resources to step execution
                            },
                            bufferPercentages: {
                                emergency: 10,
                                optimization: 5,
                                parallel: 15,
                            },
                            contention: {
                                strategy: "priority_based" as const,
                                preemptionEnabled: true,
                                priorityWeights: {
                                    "critical": 10,
                                    "high": 5,
                                    "medium": 2,
                                    "low": 1,
                                },
                            },
                        },
                        thresholds: {
                            resourceUtilization: {
                                warning: 70,
                                critical: 90,
                                optimization: 50,
                            },
                            latency: {
                                targetMs: 5000,
                                warningMs: 10000,
                                criticalMs: 30000,
                            },
                            failureRate: {
                                warningPercent: 10,
                                criticalPercent: 25,
                            },
                        },
                        history: {
                            recentAllocations: [],
                            performanceMetrics: {
                                avgUtilization: 0,
                                peakUtilization: 0,
                                bottleneckFrequency: {},
                            },
                            optimizationHistory: [],
                        },
                    },
                    organizational: {
                        structure: {
                            hierarchy: [
                                {
                                    level: 1,
                                    roles: ["swarm_leader"],
                                    authority: ["approve_tools", "allocate_resources", "create_teams"],
                                },
                                {
                                    level: 2,
                                    roles: ["swarm_agent"],
                                    authority: ["execute_tasks", "update_blackboard"],
                                },
                            ],
                            groups: [
                                {
                                    id: "primary_team",
                                    name: "Primary Execution Team",
                                    members: [], // Will be populated as agents join
                                    responsibilities: ["goal_achievement", "task_execution"],
                                },
                            ],
                            dependencies: [],
                        },
                        functional: {
                            missions: [
                                {
                                    id: "primary_mission",
                                    name: config.name,
                                    objectives: [config.goal],
                                    assignedRoles: ["swarm_leader", "swarm_agent"],
                                    expectedDuration: config.resources.maxTime,
                                    priority: "high" as const,
                                },
                            ],
                            goals: [],
                        },
                        normative: {
                            norms: [
                                {
                                    id: "resource_conservation",
                                    name: "Conserve Resources",
                                    type: "obligation" as const,
                                    condition: "resourceUtilization > 80%",
                                    action: "reduce_concurrent_tasks",
                                    priority: 8,
                                },
                                {
                                    id: "collaboration_requirement",
                                    name: "Collaborate on Complex Tasks",
                                    type: "obligation" as const,
                                    condition: "taskComplexity > 7",
                                    action: "request_team_assistance",
                                    priority: 6,
                                },
                            ],
                            sanctions: [
                                {
                                    normId: "resource_conservation",
                                    violationSeverity: "moderate" as const,
                                    action: "resource_reduction" as const,
                                },
                            ],
                        },
                    },
                },

                // Configuration that enables emergent behavior
                configuration: {
                    timeouts: {
                        routineExecutionMs: config.resources.maxTime,
                        stepExecutionMs: 30000,
                        approvalTimeoutMs: 300000,
                        idleTimeoutMs: 600000,
                    },
                    retries: {
                        maxRetries: 3,
                        backoffStrategy: "exponential" as const,
                        baseDelayMs: 1000,
                        maxDelayMs: 30000,
                    },
                    features: {
                        emergentGoalGeneration: true,    // Allow agents to create new goals
                        adaptiveResourceAllocation: true, // Allow resource strategy changes
                        crossSwarmCommunication: false,   // Disabled initially for security
                        autonomousToolApproval: config.config.autoApproveTools,
                        contextualLearning: true,         // Enable context-based learning
                    },
                    coordination: {
                        maxParallelAgents: config.config.parallelExecutionLimit,
                        communicationProtocol: "event_driven" as const,
                        consensusThreshold: 0.7,
                        leadershipElection: "manual" as const, // Leader specified during creation
                    },
                },

                // Shared blackboard for inter-agent communication
                blackboard: new Map([
                    ["conversationId", conversationId],
                    ["parentSwarmId", config.parentSwarmId || null],
                    ["userId", config.userId],
                    ["organizationId", config.organizationId || null],
                    ["name", config.name],
                    ["description", config.description],
                    ["goal", config.goal],
                ]),

                // Current execution state
                execution: {
                    status: "initializing" as const,
                    teams: [
                        {
                            id: "primary_team",
                            name: "Primary Execution Team",
                            leader: config.leaderBotId || "system",
                            members: [],
                            status: "forming" as const,
                        },
                    ],
                    agents: [],
                    activeRuns: [],
                },

                // Context metadata
                metadata: {
                    createdBy: config.userId,
                    createdFrom: config.parentSwarmId,
                    subscribers: [],
                    emergencyContacts: [config.userId],
                    retentionPolicy: {
                        keepHistoryDays: 30,
                        archiveAfterDays: 90,
                        deleteAfterDays: 365,
                    },
                    diagnostics: {
                        contextSize: 0, // Will be calculated by SwarmContextManager
                        updateFrequency: 0,
                        subscriptionCount: 0,
                        lastOptimization: new Date(),
                    },
                },
            };

            // Create the context
            await this.contextManager.createContext(swarmId, initialContext);

            this.logger.info("[TierOneCoordinator] Initialized swarm context with emergent capabilities", {
                swarmId,
                name: config.name,
                emergentFeaturesEnabled: Object.values(initialContext.configuration.features).filter(Boolean).length,
                securityRoles: Object.keys(initialContext.policy.security.permissions).length,
                resourcePoolsConfigured: Object.keys(initialContext.resources.total.models).length + Object.keys(initialContext.resources.total.tools).length,
            });

        } catch (error) {
            this.logger.error("[TierOneCoordinator] Failed to initialize swarm context", {
                swarmId,
                name: config.name,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't throw - swarm can still function without unified context
        }
    }

}

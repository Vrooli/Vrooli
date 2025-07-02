/**
 * SwarmStateMachine - Autonomous Swarm Coordination
 * 
 * This is the battle-tested implementation from conversation/responseEngine.ts,
 * adapted for the tier1 execution architecture. It provides elegant, event-driven
 * swarm coordination without overly complex state transitions.
 * 
 * Key features:
 * - Simple, proven state model (UNINITIALIZED → STARTING → RUNNING/IDLE → STOPPED/FAILED)
 * - Event queue with autonomous draining
 * - Tool approval/rejection flows
 * - Graceful shutdown with statistics
 * 
 * The beauty of this design is that complex behaviors (goal setting, team formation,
 * task decomposition) emerge from AI agent decisions rather than being hard-coded
 * as states. Agents use tools like update_swarm_shared_state, resource_manage, and
 * spawn_swarm to accomplish these tasks when they determine it's necessary.
 */

import {
    EventTypes,
    generatePK,
    toBotId,
    toSwarmId,
    type BotParticipant,
    type ChatConfigObject,
    type ChatMessage,
    type ConversationContext,
    type ConversationTrigger,
    type SessionUser,
    type SocketEventPayloads,
    type SwarmSubTask,
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import type { ConversationEngine } from "../../conversation/conversationEngine.js";
import type { EventBus } from "../../events/eventBus.js";
import type { BaseServiceEvent } from "../../events/types.js";
import type { ResponseService } from "../../response/responseService.js";
import { BaseStateMachine, BaseStates, type BaseState } from "../shared/BaseStateMachine.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { ContextSubscription, ContextUpdateEvent, UnifiedSwarmContext } from "../shared/UnifiedSwarmContext.js";

export const SwarmState = BaseStates;
export type State = BaseState;

/**
 * SwarmStateMachine
 * 
 * Manages the lifecycle of an autonomous agent swarm. Instead of prescriptive states
 * for goal setting, team formation, etc., this implementation lets those behaviors
 * emerge from agent decisions. The state machine focuses on operational states only.
 * 
 * States:
 * - UNINITIALIZED: Not yet started
 * - STARTING: Initializing swarm with goal and leader
 * - RUNNING: Actively processing events
 * - IDLE: Waiting for events (but monitoring for work)
 * - PAUSED: Temporarily suspended
 * - STOPPED: Gracefully ended
 * - FAILED: Error occurred
 * - TERMINATED: Force shutdown
 * 
 * Agents handle complex coordination through tools:
 * - update_swarm_shared_state: Manage subtasks, team, resources
 * - resource_manage: Find/create teams, routines, etc.
 * - spawn_swarm: Create child swarms for complex subtasks
 * - run_routine: Execute discovered routines
 */
export class SwarmStateMachine extends BaseStateMachine<State, BaseServiceEvent> {
    private conversationId: string | null = null;
    private initiatingUser: SessionUser | null = null;
    private swarmId: string | null = null;
    private contextSubscription: ContextSubscription | null = null;
    private swarmContext: UnifiedSwarmContext | null = null;

    constructor(
        private readonly contextManager: ISwarmContextManager, // REQUIRED: SwarmContextManager for unified state management
        private readonly conversationEngine: ConversationEngine, // NEW: For conversation orchestration
        private readonly responseService: ResponseService, // NEW: For individual bot responses
        private readonly eventBus: EventBus, // NEW: For event communication
    ) {
        super(SwarmState.UNINITIALIZED, "SwarmStateMachine");
    }

    // Implementation of ManagedTaskStateMachine methods
    public getTaskId(): string {
        if (!this.conversationId) {
            logger.error("SwarmStateMachine: getTaskId called before conversationId was set.");
            return "undefined_swarm_task_id";
        }
        return this.conversationId;
    }


    public getAssociatedUserId(): string | undefined {
        return this.initiatingUser?.id;
    }

    /**
     * Starts the swarm with a goal and initial configuration
     */
    async start(convoId: string, goal: string, initiatingUser: SessionUser, swarmId?: string): Promise<void> {
        if (this.state !== SwarmState.UNINITIALIZED) {
            logger.warn(`SwarmStateMachine for ${convoId} already started. Current state: ${this.state}`);
            return;
        }

        this.conversationId = convoId;
        this.initiatingUser = initiatingUser;
        this.swarmId = swarmId || convoId;
        this.state = SwarmState.STARTING;
        logger.info(`Starting SwarmStateMachine for ${convoId} with goal: "${goal}", swarmId: ${this.swarmId}`);

        try {
            // Initialize context with SwarmContextManager
            const initialContext: Partial<UnifiedSwarmContext> = {
                execution: {
                    goal,
                    status: "initializing",
                    priority: "medium",
                    createdAt: new Date(),
                    updatedAt: new Date(),
                },
                participants: {
                    bots: {},
                    users: {
                        [initiatingUser.id]: {
                            id: initiatingUser.id,
                            role: "initiator",
                            joinedAt: new Date(),
                            active: true,
                        },
                    },
                },
                blackboard: {
                    items: {},
                    subscriptions: {},
                },
            };

            await this.contextManager.createContext(this.swarmId, initialContext);

            // Subscribe to context changes
            this.contextSubscription = await this.contextManager.subscribe(
                this.swarmId,
                async (context) => {
                    this.swarmContext = context;
                },
            );

            // Use ConversationEngine to initiate the swarm with an AI leader
            const trigger: ConversationTrigger = {
                type: "swarm_event",
                event: {
                    id: generatePK().toString(),
                    type: EventTypes.SWARM.STARTED,
                    timestamp: new Date(),
                    data: {
                        chatId: convoId,
                        goal,
                        initiatingUser: initiatingUser.id,
                    },
                },
            };

            // Get context and let ConversationEngine select the appropriate bot
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "conversation",
            });

            // Transition to appropriate state based on result
            this.state = result.success ? SwarmState.IDLE : SwarmState.FAILED;

            // Emit state change event
            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.SWARM.STATE_CHANGED,
                timestamp: new Date(),
                source: "swarm_state_machine",
                data: {
                    entityType: "swarm",
                    entityId: this.conversationId,
                    oldState: "UNINITIALIZED",
                    newState: this.state,
                    message: result.success ? "Swarm initialized successfully" : "Swarm initialization failed",
                },
            });

            // Process any queued events
            if (this.eventQueue.length > 0) {
                this.drain().catch(err =>
                    logger.error("Error draining initial event queue", { error: err, conversationId: convoId }),
                );
            }

        } catch (error) {
            this.state = SwarmState.FAILED;

            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.SWARM.STATE_CHANGED,
                timestamp: new Date(),
                source: "swarm_state_machine",
                data: {
                    entityType: "swarm",
                    entityId: this.conversationId,
                    oldState: "STARTING",
                    newState: "FAILED",
                    message: `Swarm initialization failed: ${error instanceof Error ? error.message : String(error)}`,
                    error: error instanceof Error ? {
                        name: error.name,
                        message: error.message,
                        stack: error.stack,
                    } : { message: String(error) },
                },
            });

            throw error;
        }
    }

    /**
     * Handles incoming events by queuing them
     * Override to ensure proper validation
     */
    async handleEvent(ev: BaseServiceEvent): Promise<void> {
        // Validate based on event type
        if (ev.type === EventTypes.SWARM.STARTED) {
            const data = ev.data as SocketEventPayloads[typeof EventTypes.SWARM.STARTED];
            if (!data.chatId) {
                logger.warn(`[SwarmStateMachine] Missing chatId in ${ev.type} event`);
                return;
            }
        }

        // Delegate to parent class which handles queuing and draining
        await super.handleEvent(ev);
    }

    /**
     * Process a single event (implements abstract method from BaseStateMachine)
     */
    protected async processEvent(event: BaseServiceEvent): Promise<void> {
        logger.debug(`[SwarmStateMachine] Handling event: ${event.type}`);

        switch (event.type) {

            case EventTypes.CHAT.MESSAGE_ADDED:
                await this.handleExternalMessage(event as BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_ADDED]>);
                break;

            case EventTypes.CHAT.TOOL_APPROVAL_GRANTED:
                await this.handleApprovedTool(event as BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_GRANTED]>);
                break;

            case EventTypes.CHAT.TOOL_APPROVAL_REJECTED:
                await this.handleRejectedTool(event as BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_REJECTED]>);
                break;

            case EventTypes.RUN.TASK_READY:
                await this.handleInternalTaskAssignment(event as BaseServiceEvent<SocketEventPayloads[typeof EventTypes.RUN.TASK_READY]>);
                break;

            case EventTypes.RUN.COMPLETED:
            case EventTypes.RUN.FAILED:
                await this.handleInternalStatusUpdate(event as BaseServiceEvent<SocketEventPayloads[typeof EventTypes.RUN.COMPLETED | typeof EventTypes.RUN.FAILED]>);
                break;

            default:
                logger.warn(`Unknown event type: ${event.type}`);
        }
    }



    /**
     * Override stop to add requesting user parameter
     */
    async stop(
        mode: "graceful" | "force" = "graceful",
        reason?: string,
        _requestingUser?: SessionUser,
    ): Promise<{
        success: boolean;
        message?: string;
        finalState?: any;
        error?: string;
    }> {
        // Delegate to parent class
        return super.stop(mode, reason);
    }

    /**
     * Called when stopping - return final state data
     * Implements abstract method from BaseStateMachine
     */
    protected async onStop(mode: "graceful" | "force", reason?: string): Promise<any> {
        if (!this.conversationId) {
            throw new Error("SwarmStateMachine: onStop() called before conversationId was set.");
        }

        // Cleanup context subscription
        if (this.contextSubscription) {
            try {
                await this.contextManager.unsubscribe(this.contextSubscription.id);
                this.contextSubscription = null;
                logger.debug("[SwarmStateMachine] Cleaned up context subscription", {
                    conversationId: this.conversationId,
                    swarmId: this.swarmId,
                });
            } catch (error) {
                logger.warn("[SwarmStateMachine] Failed to cleanup context subscription", {
                    conversationId: this.conversationId,
                    swarmId: this.swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        // Get final statistics from context
        let finalState;
        if (this.swarmId) {
            try {
                const context = await this.contextManager.getContext(this.swarmId);
                const subtasksItem = context.blackboard?.items?.swarm_subtasks;
                const statsItem = context.blackboard?.items?.swarm_stats;
                const subtasks = subtasksItem?.content as SwarmSubTask[] || [];
                const stats = statsItem?.content as any || {};

                const totalSubTasks = subtasks.length;
                const completedSubTasks = subtasks.filter((task: SwarmSubTask) =>
                    task.status === "done",
                ).length;

                finalState = {
                    endedAt: new Date().toISOString(),
                    reason: reason || "Swarm stopped",
                    mode,
                    totalSubTasks,
                    completedSubTasks,
                    totalCreditsUsed: stats.totalCredits || "0",
                    totalToolCalls: stats.totalToolCalls || 0,
                };
            } catch (error) {
                logger.warn("[SwarmStateMachine] Could not get final statistics", {
                    error: error instanceof Error ? error.message : String(error),
                });
                finalState = {
                    endedAt: new Date().toISOString(),
                    reason: reason || "Swarm stopped",
                    mode,
                    totalSubTasks: 0,
                    completedSubTasks: 0,
                    totalCreditsUsed: "0",
                    totalToolCalls: 0,
                };
            }
        } else {
            finalState = {
                endedAt: new Date().toISOString(),
                reason: reason || "Swarm stopped",
                mode,
                totalSubTasks: 0,
                completedSubTasks: 0,
                totalCreditsUsed: "0",
                totalToolCalls: 0,
            };
        }

        // Emit swarm stopped event
        await this.eventBus.publish({
            id: generatePK().toString(),
            type: EventTypes.SWARM.STATE_CHANGED,
            timestamp: new Date(),
            data: {
                entityType: "swarm",
                entityId: this.conversationId,
                oldState: this.state,
                newState: mode === "force" ? "TERMINATED" : "STOPPED",
                message: reason || "Swarm stopped",
                metadata: finalState,
            },
        });

        return finalState;
    }

    /**
     * Called when entering idle state
     * Implements abstract method from BaseStateMachine
     */
    protected async onIdle(): Promise<void> {
        // Could implement autonomous monitoring here
        // For now, just log
        logger.debug("[SwarmStateMachine] Entered IDLE state", {
            conversationId: this.conversationId,
        });
    }

    /**
     * Transform UnifiedSwarmContext to ConversationContext for ConversationEngine
     */
    private async transformToConversationContext(context: UnifiedSwarmContext): Promise<ConversationContext> {
        if (!this.swarmId || !this.initiatingUser) {
            throw new Error("Missing swarmId or initiatingUser");
        }

        // Get participants from context or use empty array
        const participants: BotParticipant[] = Object.entries(context.participants?.bots || {}).map(([id, bot]) => ({
            id: toBotId(id),
            config: bot.config || { id, name: bot.name || "Unknown Bot" },
            state: {
                isProcessing: bot.status === "processing",
                isWaiting: bot.status === "waiting",
                hasResponded: bot.status === "completed",
            },
            isAvailable: bot.status !== "error",
            name: bot.name || "Unknown Bot",
        }));

        return {
            swarmId: toSwarmId(this.swarmId),
            userData: this.initiatingUser,
            timestamp: new Date(),
            participants,
            conversationHistory: [],
            availableTools: [],
            teamConfig: undefined,
            sharedState: context.blackboard?.items || {},
        };
    }

    /**
     * Called when pausing
     * Implements abstract method from BaseStateMachine
     */
    protected async onPause(): Promise<void> {
        this.logLifecycleEvent("Paused", {
            conversationId: this.conversationId,
        });
    }

    /**
     * Called when resuming
     * Implements abstract method from BaseStateMachine
     */
    protected async onResume(): Promise<void> {
        this.logLifecycleEvent("Resumed", {
            conversationId: this.conversationId,
        });
    }

    /**
     * Determine if an error is fatal
     * Implements abstract method from BaseStateMachine
     */
    protected async isErrorFatal(error: unknown, _event: BaseServiceEvent): Promise<boolean> {
        // For now, only certain errors are considered fatal
        if (error instanceof Error) {
            // Network errors are recoverable
            if (error.message.includes("ECONNREFUSED") ||
                error.message.includes("ETIMEDOUT") ||
                error.message.includes("ENOTFOUND")) {
                return false;
            }

            // Configuration errors are fatal
            if (error.message.includes("No leader bot") ||
                error.message.includes("Conversation state not found") ||
                error.message.includes("Invalid configuration")) {
                return true;
            }
        }

        // Default to non-fatal to allow recovery
        return false;
    }

    private async updateConversationConfig(
        conversationId: string,
        updates: Partial<ChatConfigObject>,
    ): Promise<void> {
        if (!this.swarmId) {
            logger.error("[SwarmStateMachine] Cannot update config without swarmId", { conversationId });
            return;
        }

        try {
            // Get current context to merge updates
            const currentContext = await this.contextManager.getContext(this.swarmId);

            // Transform conversation config updates to unified context format
            const contextUpdates: Partial<UnifiedSwarmContext> = {};

            // Only update fields that are actually provided
            if (updates.goal !== undefined) {
                contextUpdates.execution = {
                    ...currentContext.execution,
                    goal: updates.goal,
                };
            }

            if (updates.blackboard) {
                // Transform blackboard items to unified format
                const blackboardItems = (updates.blackboard || []).reduce((acc, item) => {
                    const itemId = item.id || generatePK();
                    acc[itemId] = {
                        id: itemId,
                        type: item.type || "data",
                        content: item.content,
                        metadata: item.metadata || {},
                        createdAt: new Date(),
                        updatedAt: new Date(),
                    };
                    return acc;
                }, {} as Record<string, any>);

                contextUpdates.blackboard = {
                    subscriptions: {},
                    items: {
                        ...currentContext.blackboard.items,
                        ...blackboardItems,
                    },
                };
            }

            if (updates.subtasks) {
                // Store subtasks in blackboard for now
                contextUpdates.blackboard = {
                    subscriptions: {},
                    items: {
                        ...currentContext.blackboard?.items,
                        swarm_subtasks: {
                            id: "swarm_subtasks",
                            type: "data" as const,
                            content: updates.subtasks,
                            metadata: { systemGenerated: true },
                            createdAt: new Date(),
                            updatedAt: new Date(),
                        },
                    },
                };
            }

            if (updates.stats) {
                // Store stats in blackboard for now
                contextUpdates.blackboard = {
                    subscriptions: {},
                    items: {
                        ...currentContext.blackboard?.items,
                        swarm_stats: {
                            id: "swarm_stats",
                            type: "data" as const,
                            content: updates.stats,
                            metadata: { systemGenerated: true },
                            createdAt: new Date(),
                            updatedAt: new Date(),
                        },
                    },
                };
            }

            // Update context with merged changes
            await this.contextManager.updateContext(
                this.swarmId,
                contextUpdates,
                `Swarm configuration updated: ${Object.keys(updates).join(", ")}`,
            );

            logger.info("[SwarmStateMachine] Updated swarm context via SwarmContextManager", {
                swarmId: this.swarmId,
                conversationId,
                updatedFields: Object.keys(updates),
                emergentCapabilitiesEnabled: true,
            });

        } catch (error) {
            logger.error("[SwarmStateMachine] Failed to update context via SwarmContextManager", {
                swarmId: this.swarmId,
                conversationId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    private async handleExternalMessage(event: BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_ADDED]>): Promise<void> {
        logger.info("[SwarmStateMachine] Handling external message", {
            conversationId: this.conversationId,
            messageId: event.data.message?.id,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger from the external message
            const trigger: ConversationTrigger = {
                type: "user_message",
                message: event.data.message as ChatMessage,
            };

            // Orchestrate conversation - let ConversationEngine handle bot selection
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "conversation",
            });

            logger.info("[SwarmStateMachine] Generated response to external message", {
                conversationId: this.conversationId,
                messagesGenerated: result.messages.length,
                success: result.success,
            });

            // Update state if needed
            if (!result.success && this.state === SwarmState.RUNNING) {
                this.state = SwarmState.IDLE;
                logger.info("[SwarmStateMachine] Transitioned to IDLE due to message handling failure");
            }

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling external message", {
                conversationId: this.conversationId,
                error: error instanceof Error ? error.message : String(error),
            });

            if (await this.isErrorFatal(error, event)) {
                this.state = SwarmState.FAILED;
                logger.error("[SwarmStateMachine] Transitioned to FAILED due to fatal error", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    private async handleApprovedTool(event: BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_GRANTED]>): Promise<void> {
        logger.info("[SwarmStateMachine] Handling approved tool", {
            conversationId: this.conversationId,
            tool: event.data.pendingToolCall?.toolName,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            const pendingToolCall = event.data.pendingToolCall;
            if (!pendingToolCall) {
                logger.error("[SwarmStateMachine] No pending tool call in approved tool event");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for tool approval
            const trigger: ConversationTrigger = {
                type: "tool_response",
                toolResult: {
                    success: true,
                    output: { approved: true },
                    toolCall: {
                        id: pendingToolCall.id || generatePK().toString(),
                        function: {
                            name: pendingToolCall.toolName,
                            arguments: pendingToolCall.params,
                        },
                    },
                    executionTime: 0,
                    creditsUsed: "0",
                },
                requester: toBotId("system"), // Let ConversationEngine determine the bot
            };

            // Orchestrate conversation for tool execution
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "conversation",
            });

            logger.info("[SwarmStateMachine] Executed approved tool", {
                conversationId: this.conversationId,
                toolName: pendingToolCall.toolName,
                success: result.success,
            });

            // Emit tool execution event
            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.TOOL_COMPLETED,
                timestamp: new Date(),
                data: {
                    toolName: pendingToolCall.toolName,
                    toolCallId: pendingToolCall.id || generatePK().toString(),
                    parameters: pendingToolCall.params,
                    result: { approved: true, executed: result.success },
                    duration: result.duration,
                    creditsUsed: result.resourcesUsed.creditsUsed,
                },
            });

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling approved tool", {
                conversationId: this.conversationId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async handleRejectedTool(event: BaseServiceEvent<SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_REJECTED]>): Promise<void> {
        logger.info("[SwarmStateMachine] Handling rejected tool", {
            conversationId: this.conversationId,
            tool: event.data.pendingToolCall?.toolName,
            reason: event.data.reason,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            const pendingToolCall = event.data.pendingToolCall;
            const rejectionReason = event.data.reason || "Tool use was rejected";

            if (!pendingToolCall) {
                logger.error("[SwarmStateMachine] No pending tool call in rejected tool event");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for tool rejection
            const trigger: ConversationTrigger = {
                type: "tool_response",
                toolResult: {
                    success: false,
                    error: {
                        code: "TOOL_REJECTED",
                        message: rejectionReason,
                        tier: "tier1",
                        type: "ToolRejectionError",
                    },
                    toolCall: {
                        id: pendingToolCall.id || generatePK().toString(),
                        function: {
                            name: pendingToolCall.toolName,
                            arguments: pendingToolCall.params,
                        },
                    },
                    executionTime: 0,
                    creditsUsed: "0",
                },
                requester: toBotId("system"), // Let ConversationEngine determine the bot
            };

            // Orchestrate conversation for fallback strategy
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "reasoning", // Use reasoning to find alternative approach
            });

            logger.info("[SwarmStateMachine] Generated fallback strategy for rejected tool", {
                conversationId: this.conversationId,
                toolName: pendingToolCall.toolName,
                success: result.success,
            });

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling rejected tool", {
                conversationId: this.conversationId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async handleInternalTaskAssignment(event: BaseServiceEvent<SocketEventPayloads[typeof EventTypes.RUN.TASK_READY]>): Promise<void> {
        logger.info("[SwarmStateMachine] Handling internal task assignment", {
            conversationId: this.conversationId,
            runId: event.data.runId,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for task assignment
            const trigger: ConversationTrigger = {
                type: "swarm_event",
                event: event as BaseServiceEvent,
            };

            // Orchestrate conversation for task coordination
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "reasoning", // Use reasoning for task coordination
            });

            logger.info("[SwarmStateMachine] Coordinated task assignment", {
                conversationId: this.conversationId,
                runId: event.data.runId,
                success: result.success,
            });

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling task assignment", {
                conversationId: this.conversationId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async handleInternalStatusUpdate(event: BaseServiceEvent<SocketEventPayloads[typeof EventTypes.RUN.COMPLETED | typeof EventTypes.RUN.FAILED]>): Promise<void> {
        logger.info("[SwarmStateMachine] Handling internal status update", {
            conversationId: this.conversationId,
            eventType: event.type,
            runId: event.data.runId,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for status update
            const trigger: ConversationTrigger = {
                type: "swarm_event",
                event: event as BaseServiceEvent,
            };

            // Orchestrate conversation for status processing
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: event.type === EventTypes.RUN.FAILED ? "reasoning" : "conversation",
            });

            logger.info("[SwarmStateMachine] Processed status update", {
                conversationId: this.conversationId,
                eventType: event.type,
                success: result.success,
            });

            // Update state based on event type
            if (event.type === EventTypes.RUN.FAILED && this.state === SwarmState.RUNNING) {
                this.state = SwarmState.IDLE;
                logger.info("[SwarmStateMachine] Run failed, entering idle state");
            }

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling status update", {
                conversationId: this.conversationId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Process individual context changes and trigger appropriate state machine reactions
     */
    private async processContextChange(path: string, event: ContextUpdateEvent): Promise<void> {
        // React to execution status changes
        if (path.startsWith("execution.status")) {
            const newStatus = this.swarmContext?.execution?.status;
            logger.info("[SwarmStateMachine] Swarm execution status changed via context", {
                newStatus,
                conversationId: this.conversationId,
                swarmId: event.swarmId,
            });

            // Could trigger state transitions here if needed
            // For now, just log the change
        }

        // React to security policy changes
        if (path.startsWith("policy.security")) {
            logger.info("[SwarmStateMachine] Security policy updated via context", {
                path,
                conversationId: this.conversationId,
            });

            // Emit event for security policy changes
            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.SWARM.POLICY_UPDATED,
                timestamp: new Date(),
                data: {
                    policyType: "security",
                    path,
                    change: event.changes?.[path],
                    emergent: event.emergentCapability,
                },
            });
        }

        // React to resource policy changes
        if (path.startsWith("policy.resource")) {
            logger.info("[SwarmStateMachine] Resource policy updated via context", {
                path,
                conversationId: this.conversationId,
            });

            // Emit event for resource policy changes
            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.SWARM.POLICY_UPDATED,
                timestamp: new Date(),
                data: {
                    policyType: "resource",
                    path,
                    change: event.changes?.[path],
                    emergent: event.emergentCapability,
                },
            });
        }

        // React to organizational changes
        if (path.startsWith("policy.organizational")) {
            logger.info("[SwarmStateMachine] Organizational structure updated via context", {
                path,
                conversationId: this.conversationId,
            });

            // Emit event for organizational changes
            await this.eventBus.publish({
                id: generatePK().toString(),
                type: EventTypes.SWARM.POLICY_UPDATED,
                timestamp: new Date(),
                data: {
                    policyType: "organizational",
                    path,
                    change: event.changes?.[path],
                    emergent: event.emergentCapability,
                },
            });
        }

        // React to blackboard changes (shared state updates)
        if (path.startsWith("blackboard.items")) {
            logger.debug("[SwarmStateMachine] Blackboard state updated via context", {
                path,
                conversationId: this.conversationId,
            });

            // Blackboard changes are frequent, so only log at debug level
            // Agents can react to these through their own subscriptions
        }
    }

}

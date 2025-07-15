/**
 * SwarmStateMachine - Autonomous Swarm Coordination
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

import { ChatConfig, EventTypes, generatePK, RunState, toSwarmId, type BotParticipant, type ConversationContext, type ConversationTrigger, type RunEventData, type SessionUser, type SocketEventPayloads, type StateMachineState, type SwarmState, type SwarmSubTask } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import type { SwarmExecutionTask } from "../../../tasks/taskTypes.js";
import type { ConversationEngine } from "../../conversation/conversationEngine.js";
import { EventInterceptor } from "../../events/EventInterceptor.js";
import { InMemoryLockService } from "../../events/LockService.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ServiceEvent } from "../../events/types.js";
import type { CachedConversationStateStore } from "../../response/chatStore.js";
import { BaseStateMachine } from "../shared/BaseStateMachine.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";

/**
 * SwarmStateMachine
 * 
 * Manages the lifecycle of an autonomous agent swarm. Instead of prescriptive states
 * for goal setting, team formation, etc., this implementation lets those behaviors
 * emerge from agent decisions. The state machine focuses on operational states only.
 * 
 * Agents handle complex coordination through tools:
 * - update_swarm_shared_state: Manage subtasks, team, resources
 * - resource_manage: Find/create teams, routines, etc.
 * - spawn_swarm: Create child swarms for complex subtasks
 * - run_routine: Execute discovered routines
 */
export class SwarmStateMachine extends BaseStateMachine<ServiceEvent> {
    private initiatingUser: SessionUser | null = null;
    private swarmId: string | null = null;

    // Event handling services
    private eventInterceptor: EventInterceptor;

    constructor(
        private readonly contextManager: ISwarmContextManager, // REQUIRED: SwarmContextManager for unified state management
        private readonly conversationEngine: ConversationEngine, // NEW: For conversation orchestration
        private readonly chatStore: CachedConversationStateStore, // For loading chat configuration
    ) {
        // Pass empty coordination config for now - will be updated when swarmId is set
        super(RunState.UNINITIALIZED, "SwarmStateMachine", {});

        // Initialize event handling services
        this.eventInterceptor = new EventInterceptor(
            new InMemoryLockService(),
            this.contextManager,
        );
    }

    // Implementation of ManagedTaskStateMachine methods
    public getTaskId(): string {
        if (!this.swarmId) {
            logger.error("SwarmStateMachine: getTaskId called before swarmId was set.");
            return "undefined_swarm_task_id";
        }
        return this.swarmId;
    }


    public getAssociatedUserId(): string | undefined {
        return this.initiatingUser?.id;
    }

    /**
     * Starts the swarm with a SwarmExecutionTask
     */
    async start(request: SwarmExecutionTask): Promise<{ success: boolean; error?: any }> {
        if (this.state !== RunState.UNINITIALIZED) {
            logger.warn(`SwarmStateMachine already started. Current state: ${this.state}`);
            return { success: false, error: `Already started. Current state: ${this.state}` };
        }

        // Extract data from unified request structure
        const { input } = request;
        const swarmId = input.swarmId || generatePK().toString();
        const goal = input.goal;
        const userData = input.userData;

        this.initiatingUser = userData;
        this.swarmId = swarmId;

        // Update coordination config now that we have swarmId
        this.coordinationConfig.contextId = swarmId;
        this.coordinationConfig.swarmId = swarmId;
        if (input.chatId) {
            this.coordinationConfig.chatId = input.chatId;
        }

        this.state = RunState.LOADING;
        logger.info(`Starting SwarmStateMachine for swarmId: ${swarmId} with goal: "${goal}"`);

        try {
            // Load chat configuration if chatId is provided
            let chatConfig = ChatConfig.default().export();
            if (input.chatId) {
                const conversationState = await this.chatStore.get(input.chatId);
                if (conversationState?.config) {
                    chatConfig = conversationState.config;
                    logger.info(`Loaded chat config from chatId: ${input.chatId}`, {
                        hasActiveBotId: !!chatConfig.activeBotId,
                        goal: chatConfig.goal,
                    });
                }
            }

            // Initialize context with SwarmContextManager using proper SwarmState structure
            const initialContext: Partial<SwarmState> = {
                swarmId: toSwarmId(swarmId),
                version: 1,
                chatConfig: {
                    ...chatConfig,
                    goal, // Move goal into chatConfig where it belongs
                    swarmLeader: chatConfig.swarmLeader || userData.id, // Set initiating user as default leader
                },
                execution: {
                    status: RunState.LOADING,
                    agents: [], // Will be populated when bots join
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: {
                        credits: 0,
                        tokens: 0,
                        time: 0,
                    },
                    remaining: {
                        credits: 1000, // Default allocation
                        tokens: 10000,
                        time: 3600,
                    },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: userData.id,
                    subscribers: new Set<string>(),
                },
            };

            if (!this.swarmId) {
                logger.error("SwarmStateMachine: Attempted to create context without swarmId");
                throw new Error("Missing swarmId");
            }
            await this.contextManager.createContext(this.swarmId, initialContext);

            // Use ConversationEngine to initiate the swarm with an AI leader
            const trigger: ConversationTrigger = {
                type: "start",
            };

            // Get context and let ConversationEngine select the appropriate bot
            const context = await this.contextManager.getContext(this.swarmId!);
            if (!context) {
                throw new Error("Failed to get swarm context");
            }
            const conversationContext = await this.transformToConversationContext(context);

            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "conversation",
            });

            // Transition to appropriate state based on result
            this.state = result.success ? RunState.READY : RunState.FAILED;

            // Setup event subscriptions now that we have all required IDs
            if (result.success) {
                await this.setupEventSubscriptions();
            }

            // Emit state change event
            const { proceed, reason } = await EventPublisher.emit(EventTypes.SWARM.STATE_CHANGED, {
                chatId: this.swarmId || "unknown",
                oldState: "UNINITIALIZIALIZED" as StateMachineState,
                newState: this.state as StateMachineState,
                message: result.success ? "Swarm initialized successfully" : "Swarm initialization failed",
            });

            if (!proceed) {
                logger.warn("[SwarmStateMachine] Swarm state change blocked", {
                    swarmId: this.swarmId,
                    reason,
                    newState: this.state,
                });
                // Continue anyway for state transitions - this is informational
            }

            // Process any queued events
            if (this.eventQueue.length > 0) {
                this.drain().catch(err =>
                    logger.error("Error draining initial event queue", { error: err, swarmId }),
                );
            }

            // Return success result
            return { success: result.success };

        } catch (error) {
            this.state = RunState.FAILED;

            const { proceed: errorProceed, reason: errorReason } = await EventPublisher.emit(EventTypes.SWARM.STATE_CHANGED, {
                chatId: this.swarmId || "unknown",
                oldState: "STARTING" as StateMachineState,
                newState: this.state as StateMachineState,
                message: `Swarm initialization failed: ${error instanceof Error ? error.message : String(error)}`,
            });

            if (!errorProceed) {
                logger.error("[SwarmStateMachine] Critical swarm error state change blocked", {
                    swarmId: this.swarmId,
                    reason: errorReason,
                    error: error instanceof Error ? error.message : String(error),
                });
                // Continue anyway - error states must be recorded
            }

            return {
                success: false,
                error: {
                    message: error instanceof Error ? error.message : String(error),
                    name: error instanceof Error ? error.name : "UnknownError",
                    stack: error instanceof Error ? error.stack : undefined,
                },
            };
        }
    }

    /**
     * Handles incoming events by queuing them
     * Override to ensure proper validation
     */
    async handleEvent(ev: ServiceEvent): Promise<void> {
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
    protected async processEvent(event: ServiceEvent): Promise<void> {
        logger.debug(`[SwarmStateMachine] Processing event: ${event.type}`);

        // Handle events based on exact type matches
        switch (event.type) {
            // Chat events
            case EventTypes.CHAT.MESSAGE_ADDED:
                await this.handleExternalMessage(event);
                break;

            // Tool approval events
            case EventTypes.TOOL.APPROVAL_GRANTED:
                await this.handleApprovedTool(event);
                break;
            case EventTypes.TOOL.APPROVAL_REJECTED:
                await this.handleRejectedTool(event);
                break;

            // Cancellation events
            case EventTypes.CHAT.CANCELLATION_REQUESTED:
                await this.handleCancellationRequest(event);
                break;

            // Run status events
            case EventTypes.RUN.COMPLETED:
            case EventTypes.RUN.FAILED:
                await this.handleInternalStatusUpdate(event);
                break;

            default:
                // Handle pattern-based events
                if (event.type.startsWith("swarm/")) {
                    await this.handleSwarmEvent(event);
                } else if (event.type.startsWith("safety/") || event.type.startsWith("security/")) {
                    await this.handleSafetyEvent(event);
                } else {
                    logger.debug(`[SwarmStateMachine] Unhandled event type: ${event.type}`);
                }
                break;
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
        if (!this.swarmId) {
            throw new Error("SwarmStateMachine: onStop() called before swarmId was set.");
        }

        // Context is managed by SwarmContextManager - no local cleanup needed

        // Get final statistics from context
        let finalState;
        if (this.swarmId) {
            try {
                const context = await this.contextManager.getContext(this.swarmId);

                if (context) {
                    // Access subtasks and stats from chatConfig
                    const subtasks = context.chatConfig.subtasks || [];
                    const stats = context.chatConfig.stats;

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
                        totalCreditsUsed: stats.totalCredits,
                        totalToolCalls: stats.totalToolCalls,
                    };
                } else {
                    // Context not found, use defaults
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
        const { proceed: stopProceed, reason: stopReason } = await EventPublisher.emit(EventTypes.SWARM.STATE_CHANGED, {
            chatId: this.swarmId || "unknown",
            oldState: this.state as StateMachineState,
            newState: (mode === "force" ? "TERMINATED" : "COMPLETED") as StateMachineState,
            message: reason || "Swarm stopped",
        });

        if (!stopProceed) {
            logger.warn("[SwarmStateMachine] Swarm stop event blocked", {
                swarmId: this.swarmId,
                reason: stopReason,
                mode,
            });
            // Continue anyway - swarm must be stopped
        }

        return finalState;
    }

    /**
     * Called when entering ready state
     * Implements abstract method from BaseStateMachine
     */
    protected async onIdle(): Promise<void> {
        // Could implement autonomous monitoring here
        // For now, just log
        logger.debug("[SwarmStateMachine] Entered ready state", {
            swarmId: this.swarmId,
        });
    }

    /**
     * Get event patterns this state machine should subscribe to
     * @returns Array of event patterns
     */
    protected getEventPatterns(): Array<{ pattern: string }> {
        // Return patterns for all event types we handle
        // The shouldHandleEvent method will filter by specific IDs
        return [
            // Chat events
            { pattern: "chat/*" },                    // All chat events
            { pattern: "chat/message/*" },            // Message events
            { pattern: "chat/cancellation/*" },       // Cancellation events

            // Tool events
            { pattern: "tool/*" },                    // All tool events
            { pattern: "tool/approval/*" },           // Tool approval events

            // Swarm events
            { pattern: "swarm/*" },                   // All swarm events
            { pattern: "swarm/state/*" },             // State changes
            { pattern: "swarm/goal/*" },              // Goal events

            // Run events
            { pattern: "run/*" },                     // All run events

            // User events
            { pattern: "user/*" },                    // All user events
            { pattern: "user/credits/*" },            // Credit updates
            { pattern: "user/permissions/*" },        // Permission changes

            // Safety/security events
            { pattern: "safety/*" },                  // All safety events (legacy)
            { pattern: "security/*" },                // All security events (new naming)
        ];
    }

    /**
     * Determine if this state machine instance should handle a specific event
     * @param event - The event to check
     * @returns true if this instance should process the event
     */
    protected shouldHandleEvent(event: ServiceEvent): boolean {
        // Check if event is relevant to this swarm instance
        const eventData = event.data as any;

        // Check various ID fields
        if (eventData.swarmId && eventData.swarmId !== this.swarmId) {
            return false;
        }

        // For user events, check if they're for our initiating user
        if (event.type.startsWith("user/") && this.initiatingUser) {
            const userIdMatch = event.type.match(/^user\/([^/]+)\//);
            if (userIdMatch && userIdMatch[1] !== this.initiatingUser.id) {
                return false;
            }
        }

        return true;
    }

    /**
     * Transform SwarmState to ConversationContext for ConversationEngine
     */
    private async transformToConversationContext(context: SwarmState): Promise<ConversationContext> {
        if (!this.swarmId || !this.initiatingUser) {
            throw new Error("Missing swarmId or initiatingUser");
        }

        // Get participants from context or use empty array
        const participants: BotParticipant[] = context.execution?.agents || [];

        // Convert blackboard array to object for shared state
        const blackboardItems = context.chatConfig.blackboard || [];
        const sharedState: Record<string, any> = {};
        for (const item of blackboardItems) {
            // Use item.id as the key (canonical BlackboardItem structure)
            sharedState[item.id] = item.value;
        }

        return {
            swarmId: toSwarmId(this.swarmId),
            userData: this.initiatingUser,
            timestamp: new Date(),
            participants,
            conversationHistory: [],
            availableTools: [],
            teamConfig: undefined,
            sharedState,
        };
    }

    /**
     * Called when pausing
     * Implements abstract method from BaseStateMachine
     */
    protected async onPause(): Promise<void> {
        this.logLifecycleEvent("Paused", {
            swarmId: this.swarmId,
        });
    }

    /**
     * Called when resuming
     * Implements abstract method from BaseStateMachine
     */
    protected async onResume(): Promise<void> {
        this.logLifecycleEvent("Resumed", {
            swarmId: this.swarmId,
        });
    }

    /**
     * Determine if an error is fatal
     * Implements abstract method from BaseStateMachine
     */
    protected async isErrorFatal(error: unknown, _event: ServiceEvent): Promise<boolean> {
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


    private async handleExternalMessage(event: ServiceEvent): Promise<void> {
        if (event.type !== EventTypes.CHAT.MESSAGE_ADDED) {
            logger.error("[SwarmStateMachine] Unexpected event type", {
                eventType: event.type,
            });
            return;
        }

        const messageEvent = event.data as SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_ADDED];
        const message = messageEvent.messages[0];

        logger.info("[SwarmStateMachine] Handling external message", {
            swarmId: this.swarmId,
            messageId: message.id,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            if (!context) {
                throw new Error("Context not found");
            }
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger from the external message
            const trigger: ConversationTrigger = {
                type: "user_message",
                message,
            };

            // Orchestrate conversation - let ConversationEngine handle bot selection
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "conversation",
            });

            logger.info("[SwarmStateMachine] Generated response to external message", {
                swarmId: this.swarmId,
                messagesGenerated: result.messages.length,
                success: result.success,
            });

            // Update state if needed
            if (!result.success && this.state === RunState.RUNNING) {
                this.state = RunState.READY;
                logger.info("[SwarmStateMachine] Transitioned to READY due to message handling failure");
            }

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling external message", {
                swarmId: this.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });

            if (await this.isErrorFatal(error, event)) {
                this.state = RunState.FAILED;
                logger.error("[SwarmStateMachine] Transitioned to FAILED due to fatal error", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    private async handleApprovedTool(event: ServiceEvent): Promise<void> {
        // Type assertion based on the event type check in processEvent
        const data = event.data as SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_GRANTED];
        const { toolName, toolCallId, pendingId, callerBotId } = data;

        logger.info("[SwarmStateMachine] Handling approved tool with three-step flow", {
            swarmId: this.swarmId,
            toolName,
            toolCallId,
            pendingId,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            if (!toolName || !toolCallId) {
                logger.error("[SwarmStateMachine] Missing required tool data in approved tool event", {
                    toolName,
                    toolCallId,
                    pendingId,
                });
                return;
            }

            // Recover original tool arguments from execution context
            const originalToolCall = event.execution?.originalToolCall;
            if (!originalToolCall) {
                logger.error("Tool approval missing original tool call context", {
                    toolName,
                    toolCallId,
                    eventId: event.id,
                });
                throw new Error("Cannot execute tool without original arguments");
            }

            // Step 1: Check for bot interception with full swarm context
            const swarmContext = await this.contextManager.getContext(this.swarmId!);
            if (!swarmContext) {
                throw new Error("Cannot get swarm context for event interception");
            }
            const interceptionResult = await this.eventInterceptor.checkInterception(event, swarmContext);
            if (interceptionResult.intercepted && interceptionResult.progression !== "continue") {
                logger.info("Tool approval blocked by bot interception", {
                    toolName,
                    responses: interceptionResult.responses,
                    progression: interceptionResult.progression,
                });

                // Emit tool failed event to notify caller
                const { proceed: failProceed, reason: failReason } = await EventPublisher.emit(EventTypes.TOOL.FAILED, {
                    chatId: this.swarmId || "unknown",
                    toolCallId,
                    toolName,
                    error: `Tool execution blocked by security system: ${interceptionResult.progression}`,
                    duration: 0,
                    callerBotId: callerBotId || "system",
                });

                if (!failProceed) {
                    logger.warn("[SwarmStateMachine] Tool failure event blocked", {
                        swarmId: this.swarmId,
                        toolName,
                        reason: failReason,
                    });
                    // Continue anyway - failure must be recorded
                }

                return;
            }

            // Step 2: Route to conversation engine for tool execution
            // Now that tools handle their own approval inline, we can always use the conversation engine
            await this.routeToConversationEngine(event);

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling approved tool", {
                swarmId: this.swarmId,
                toolName,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }


    private async routeToConversationEngine(event: ServiceEvent): Promise<void> {
        // Type assertion - this method is called for tool approval events
        const data = event.data as SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_GRANTED];

        logger.info("[SwarmStateMachine] Routing to conversation engine for tool execution", {
            swarmId: this.swarmId,
            toolName: data.toolName,
        });

        const { toolName } = data;

        // Note: Tool approval has already been granted at this point.
        // The tool will be executed by the conversation engine, which will
        // use the ToolRunner that now has inline approval checks.

        // Get context from contextManager
        const context = await this.contextManager.getContext(this.swarmId!);
        if (!context) {
            throw new Error("Context not found");
        }
        const conversationContext = await this.transformToConversationContext(context);

        // Create trigger for continuing after tool approval
        const trigger: ConversationTrigger = {
            type: "continue",
            lastEvent: event,
        };

        // Orchestrate conversation to handle the approved tool
        const result = await this.conversationEngine.orchestrateConversation({
            context: conversationContext,
            trigger,
            strategy: "conversation",
        });

        logger.info("[SwarmStateMachine] Tool execution handled by conversation engine", {
            swarmId: this.swarmId,
            toolName,
            success: result.success,
        });
    }

    private async handleRejectedTool(event: ServiceEvent): Promise<void> {
        // Type assertion based on the event type check in processEvent
        const data = event.data as SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_REJECTED];

        logger.info("[SwarmStateMachine] Handling rejected tool", {
            swarmId: this.swarmId,
            toolName: data.toolName,
            pendingId: data.pendingId,
            reason: data.reason,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            const { toolName, toolCallId, pendingId, reason } = data;
            const rejectionReason = reason || "Tool use was rejected";

            if (!toolName || !toolCallId) {
                logger.error("[SwarmStateMachine] Missing required tool data in rejected tool event", {
                    toolName,
                    toolCallId,
                    pendingId,
                });
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            if (!context) {
                throw new Error("Context not found");
            }
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for tool rejection
            const trigger: ConversationTrigger = {
                type: "continue",
                lastEvent: event,
            };

            // Orchestrate conversation for fallback strategy
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: "reasoning", // Use reasoning to find alternative approach
            });

            logger.info("[SwarmStateMachine] Generated fallback strategy for rejected tool", {
                swarmId: this.swarmId,
                toolName,
                rejectionReason,
                success: result.success,
            });

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling rejected tool", {
                swarmId: this.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async handleCancellationRequest(event: ServiceEvent): Promise<void> {
        // Type assertion based on the event type check in processEvent
        const data = event.data as SocketEventPayloads[typeof EventTypes.CHAT.CANCELLATION_REQUESTED];

        logger.info("[SwarmStateMachine] Handling cancellation request", {
            swarmId: this.swarmId,
            userId: event.metadata?.userId,
            chatId: data.chatId,
        });

        // Stop the swarm gracefully
        await this.stop("graceful", "User requested cancellation");
    }

    private async handleSwarmEvent(event: ServiceEvent): Promise<void> {
        logger.debug("[SwarmStateMachine] Handling swarm event", {
            swarmId: this.swarmId,
            eventType: event.type,
        });

        // Handle specific swarm events using EventTypes constants
        switch (event.type) {
            case EventTypes.SWARM.STATE_CHANGED: {
                // Type assertion for state changed event
                const data = event.data as SocketEventPayloads[typeof EventTypes.SWARM.STATE_CHANGED];
                // Handle state changes from child swarms
                logger.info("[SwarmStateMachine] Child swarm state changed", {
                    parentSwarm: this.swarmId,
                    childSwarm: data.chatId,
                    oldState: data.oldState,
                    newState: data.newState,
                });
                break;
            }

            case EventTypes.SWARM.GOAL_CREATED:
            case EventTypes.SWARM.GOAL_UPDATED:
            case EventTypes.SWARM.GOAL_COMPLETED:
            case EventTypes.SWARM.GOAL_FAILED:
                // Handle goal-related events
                logger.info("[SwarmStateMachine] Swarm goal event", {
                    swarmId: this.swarmId,
                    eventType: event.type,
                });
                break;

            default:
                logger.debug("[SwarmStateMachine] Unhandled swarm event type", {
                    swarmId: this.swarmId,
                    eventType: event.type,
                });
                break;
        }
    }

    private async handleSafetyEvent(event: ServiceEvent): Promise<void> {
        logger.warn("[SwarmStateMachine] Handling safety event", {
            swarmId: this.swarmId,
            eventType: event.type,
            priority: "HIGH",
        });

        // Safety events should be handled with high priority using EventTypes constants
        switch (event.type) {
            case EventTypes.SECURITY.EMERGENCY_STOP:
                await this.stop("force", "Emergency stop requested");
                break;

            case EventTypes.SECURITY.THREAT_DETECTED:
            case EventTypes.SECURITY.PERMISSION_CHECK:
            case EventTypes.SECURITY.ACCESS_BLOCKED:
            case EventTypes.SECURITY.POLICY_VIOLATED:
                // These would be handled by safety agents through the event bus
                logger.info("[SwarmStateMachine] Security event, delegating to security agents", {
                    swarmId: this.swarmId,
                    eventType: event.type,
                });
                break;

            default:
                logger.debug("[SwarmStateMachine] Unhandled safety/security event type", {
                    swarmId: this.swarmId,
                    eventType: event.type,
                });
                break;
        }
    }

    private async handleInternalStatusUpdate(event: ServiceEvent): Promise<void> {
        // Type assertion - this method handles RUN.COMPLETED and RUN.FAILED events
        const data = event.data as RunEventData;

        logger.info("[SwarmStateMachine] Handling internal status update", {
            swarmId: this.swarmId,
            eventType: event.type,
            runId: data.runId,
        });

        try {
            if (!this.swarmId) {
                logger.error("[SwarmStateMachine] No swarmId available");
                return;
            }

            // Get context from contextManager
            const context = await this.contextManager.getContext(this.swarmId);
            if (!context) {
                throw new Error("Context not found");
            }
            const conversationContext = await this.transformToConversationContext(context);

            // Create trigger for status update
            const trigger: ConversationTrigger = {
                type: "continue",
                lastEvent: event,
            };

            // Orchestrate conversation for status processing
            const result = await this.conversationEngine.orchestrateConversation({
                context: conversationContext,
                trigger,
                strategy: event.type === EventTypes.RUN.FAILED ? "reasoning" : "conversation",
            });

            logger.info("[SwarmStateMachine] Processed status update", {
                swarmId: this.swarmId,
                eventType: event.type,
                success: result.success,
            });

            // Update state based on event type
            if (event.type === EventTypes.RUN.FAILED && this.state === RunState.RUNNING) {
                this.state = RunState.READY;
                logger.info("[SwarmStateMachine] Run failed, entering ready state");
            }

        } catch (error) {
            logger.error("[SwarmStateMachine] Error handling status update", {
                swarmId: this.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }


}

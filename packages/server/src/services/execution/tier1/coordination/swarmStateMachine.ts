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
    type SessionUser, 
    type PendingToolCallEntry,
    type ChatToolCallRecord,
    type ChatConfigObject,
    type BotParticipant,
    type BotConfigObject,
    type SwarmSubTask,
    generatePK,
    PendingToolCallStatus,
    ChatConfig,
    ExecutionStates,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type ISwarmStateStore } from "../state/swarmStateStore.js";
import { type ConversationBridge } from "../intelligence/conversationBridge.js";
import { BaseStateMachine, BaseStates, type BaseEvent } from "../../shared/BaseStateMachine.js";
import { EventTypes } from "../../../events/index.js";

/**
 * Swarm event types
 */
export type SwarmEventType = 
    | "swarm_started"
    | "external_message_created"
    | "tool_approval_response"
    | "ApprovedToolExecutionRequest"
    | "RejectedToolExecutionRequest"
    | "internal_task_assignment"
    | "internal_status_update";

/**
 * Base swarm event interface
 */
export interface SwarmEvent extends BaseEvent {
    type: SwarmEventType;
    conversationId: string;
    sessionUser: SessionUser;
    goal?: string;
    payload?: any;
}

/**
 * Swarm started event
 */
export interface SwarmStartedEvent extends SwarmEvent {
    type: "swarm_started";
    goal: string;
}

// Use the shared state definitions
export const SwarmState = BaseStates;
export type State = keyof typeof SwarmState;

/**
 * Conversation state interface (temporary until ConversationBridge exports it)
 */
interface ConversationState {
    id: string;
    config: ChatConfigObject;
    participants: BotParticipant[];
    availableTools: any[];
    initialLeaderSystemMessage: string;
    teamConfig?: any;
}

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
export class SwarmStateMachine extends BaseStateMachine<State, SwarmEvent> {
    private conversationId: string | null = null;
    private initiatingUser: SessionUser | null = null;

    constructor(
        logger: Logger,
        private readonly stateStore: ISwarmStateStore,
        private readonly conversationBridge?: ConversationBridge, // Optional for backward compatibility
    ) {
        super(logger, SwarmState.UNINITIALIZED, "SwarmStateMachine");
    }

    // Implementation of ManagedTaskStateMachine methods
    public getTaskId(): string {
        if (!this.conversationId) {
            this.logger.error("SwarmStateMachine: getTaskId called before conversationId was set.");
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
    async start(convoId: string, goal: string, initiatingUser: SessionUser): Promise<void> {
        if (this.state !== SwarmState.UNINITIALIZED) {
            this.logger.warn(`SwarmStateMachine for ${convoId} already started. Current state: ${this.state}`);
            return;
        }
        
        this.conversationId = convoId;
        this.initiatingUser = initiatingUser;
        this.state = SwarmState.STARTING;
        this.logger.info(`Starting SwarmStateMachine for ${convoId} with goal: "${goal}"`);

        try {
            await this.errorHandler.execute(async () => {
                // Get or create conversation state
                let convoState = await this.getConversationState(convoId);
                if (!convoState) {
                    // Create minimal conversation state
                    convoState = await this.createConversationState(convoId, goal, initiatingUser);
                }

                // Find leader bot
                const leaderBot = this.findLeaderBot(convoState);
                if (!leaderBot) {
                    this.logger.error(`No leader bot found for ${convoId}, cannot start swarm`);
                    throw new Error(`No leader bot found for swarm ${convoId}`);
                }

                // Generate system message for the leader
                const systemMessage = await this.conversationBridge.generateAgentResponse(
                    leaderBot,
                    { state: "STARTING" },
                    { goal },
                    "You are starting a new swarm. Set up the team and plan as needed.",
                    convoId,
                );

                // Update conversation config
                await this.updateConversationConfig(convoId, {
                    goal,
                    subtasks: [],
                    blackboard: [],
                    resources: [],
                    stats: ChatConfig.defaultStats(),
                    swarmLeader: leaderBot.id,
                });

                // Queue the started event
                const startEvent: SwarmStartedEvent = {
                    type: "swarm_started",
                    conversationId: convoId,
                    goal,
                    sessionUser: initiatingUser,
                };
                
                this.state = SwarmState.IDLE;
                
                // Emit state change to UI via unified events
                await this.publishUnifiedEvent(
                    EventTypes.STATE_SWARM_UPDATED,
                    {
                        entityType: "swarm",
                        entityId: this.conversationId!,
                        oldState: "UNINITIALIZED",
                        newState: ExecutionStates.STARTING,
                        message: "Swarm initialization complete, entering idle state",
                    },
                    {
                        conversationId: this.conversationId!,
                        priority: "medium",
                        deliveryGuarantee: "fire-and-forget",
                    },
                );
                
                // Emit initial config via unified events
                await this.publishUnifiedEvent(
                    EventTypes.CONFIG_SWARM_UPDATED,
                    {
                        entityType: "swarm",
                        entityId: this.conversationId!,
                        config: {
                            goal,
                            subtasks: [],
                            swarmLeader: leaderBot.id,
                            stats: {
                                startedAt: Date.now(),
                                totalToolCalls: 0,
                                totalCredits: "0",
                                lastProcessingCycleEndedAt: null,
                            },
                        },
                    },
                    {
                        conversationId: this.conversationId!,
                        priority: "medium",
                        deliveryGuarantee: "fire-and-forget",
                    },
                );
                
                await this.handleEvent(startEvent);
                
                // Process any queued events
                if (this.eventQueue.length > 0) {
                    this.drain().catch(err => 
                        this.logger.error("Error draining initial event queue", { error: err, conversationId: convoId }),
                    );
                }
            }, "startSwarm", { conversationId: convoId, goal });
        } catch (error) {
            this.state = SwarmState.FAILED;
            
            // Emit failure state via unified events
            await this.publishUnifiedEvent(
                EventTypes.STATE_SWARM_UPDATED,
                {
                    entityType: "swarm",
                    entityId: this.conversationId!,
                    oldState: "STARTING",
                    newState: ExecutionStates.FAILED,
                    message: `Swarm initialization failed: ${error instanceof Error ? error.message : String(error)}`,
                    error: error instanceof Error ? {
                        name: error.name,
                        message: error.message,
                        stack: error.stack,
                    } : { message: String(error) },
                },
                {
                    conversationId: this.conversationId!,
                    priority: "high",
                    deliveryGuarantee: "reliable",
                },
            );
            
            throw error;
        }
    }

    /**
     * Handles incoming events by queuing them
     * Override to ensure conversationId is provided
     */
    async handleEvent(ev: SwarmEvent): Promise<void> {
        if (!this.validateEvent(ev, ["conversationId"])) {
            return;
        }
        
        // Delegate to parent class which handles queuing and draining
        await super.handleEvent(ev);
    }

    /**
     * Process a single event (implements abstract method from BaseStateMachine)
     */
    protected async processEvent(event: SwarmEvent): Promise<void> {
        this.logger.debug(`[SwarmStateMachine] Handling event: ${event.type}`, {
            conversationId: event.conversationId,
        });

        switch (event.type) {
            case "swarm_started":
                await this.handleSwarmStarted(event as SwarmStartedEvent);
                break;
            
            case "external_message_created":
                await this.handleExternalMessage(event);
                break;
            
            case "ApprovedToolExecutionRequest":
                await this.handleApprovedTool(event);
                break;
            
            case "RejectedToolExecutionRequest":
                await this.handleRejectedTool(event);
                break;
            
            case "internal_task_assignment":
                await this.handleInternalTaskAssignment(event);
                break;
            
            case "internal_status_update":
                await this.handleInternalStatusUpdate(event);
                break;
            
            default:
                this.logger.warn(`Unknown event type: ${event.type}`);
        }
    }

    /**
     * Handles swarm started event
     */
    private async handleSwarmStarted(event: SwarmStartedEvent): Promise<void> {
        const convoState = await this.getConversationState(event.conversationId);
        if (!convoState) {
            this.logger.error(`Conversation state not found for ${event.conversationId}`);
            return;
        }

        // Get the leader bot
        const leaderBot = this.findLeaderBot(convoState);
        if (!leaderBot) {
            this.logger.error(`No leader bot found for ${event.conversationId}`);
            return;
        }

        // Let the leader bot initialize the swarm
        const response = await this.conversationBridge.generateAgentResponse(
            leaderBot,
            { state: "STARTED", goal: event.goal },
            convoState.config,
            `The swarm has started with goal: "${event.goal}". Initialize the team and create a plan.`,
            event.conversationId,
        );

        this.logger.info("[SwarmStateMachine] Leader response to swarm start", {
            conversationId: event.conversationId,
            responseLength: response.length,
        });

        // Emit event for monitoring
        await this.publishUnifiedEvent(
            EventTypes.STATE_SWARM_UPDATED,
            {
                entityType: "swarm",
                entityId: event.conversationId,
                oldState: "STARTING",
                newState: "INITIALIZED",
                message: `Swarm initialized with goal: "${event.goal}"`,
                metadata: {
                    goal: event.goal,
                    leaderId: leaderBot.id,
                },
            },
            {
                conversationId: event.conversationId,
                priority: "medium",
                deliveryGuarantee: "reliable",
            },
        );
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

        const convoState = await this.getConversationState(this.conversationId);
        if (!convoState) {
            throw new Error("Conversation state not found");
        }

        // Calculate final statistics
        const subtasks = convoState.config.subtasks || [];
        const totalSubTasks = subtasks.length;
        const completedSubTasks = subtasks.filter((task: SwarmSubTask) => 
            task.status === "done",
        ).length;

        const finalState = {
            endedAt: new Date().toISOString(),
            reason: reason || "Swarm stopped",
            mode,
            totalSubTasks,
            completedSubTasks,
            totalCreditsUsed: convoState.config.stats?.totalCredits || "0",
            totalToolCalls: convoState.config.stats?.totalToolCalls || 0,
        };

        // Emit swarm stopped event
        await this.publishUnifiedEvent(
            EventTypes.STATE_SWARM_UPDATED,
            {
                entityType: "swarm",
                entityId: this.conversationId,
                oldState: this.state,
                newState: mode === "force" ? "TERMINATED" : "STOPPED",
                message: reason || "Swarm stopped",
                metadata: finalState,
            },
            {
                conversationId: this.conversationId,
                priority: "high",
                deliveryGuarantee: "reliable",
            },
        );

        return finalState;
    }

    /**
     * Called when entering idle state
     * Implements abstract method from BaseStateMachine
     */
    protected async onIdle(): Promise<void> {
        // Could implement autonomous monitoring here
        // For now, just log
        this.logger.debug("[SwarmStateMachine] Entered IDLE state", {
            conversationId: this.conversationId,
        });
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
    protected async isErrorFatal(error: unknown, event: SwarmEvent): Promise<boolean> {
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

    private async getConversationState(conversationId: string): Promise<ConversationState | null> {
        return this.conversationBridge.getConversationState(conversationId);
    }

    private async createConversationState(
        conversationId: string, 
        goal: string, 
        user: SessionUser,
    ): Promise<ConversationState> {
        // Check if conversation already exists
        const state = await this.conversationBridge.getConversationState(conversationId);
        
        if (!state) {
            // This should not happen anymore as TierOneCoordinator creates the conversation
            // But log a warning and return a minimal state to avoid crashes
            this.logger.warn(`[SwarmStateMachine] Conversation ${conversationId} not found, this shouldn't happen`);
            return {
                id: conversationId,
                config: ChatConfig.default().export(),
                participants: [],
                availableTools: [],
                initialLeaderSystemMessage: "",
                teamConfig: undefined,
            };
        }
        
        // Update the goal in the existing conversation if needed
        if (goal && goal !== state.config.goal) {
            const updatedConfig = { ...state.config, goal };
            // Update through proper channels
            await this.conversationBridge.updateConversationConfig(conversationId, updatedConfig);
            state.config = updatedConfig;
        }
        
        return state;
    }

    private findLeaderBot(convoState: ConversationState): BotParticipant | null {
        const leaderId = convoState.config.swarmLeader;
        if (leaderId) {
            return convoState.participants.find(p => p.id === leaderId) || null;
        }
        // Return first participant (which should be the leader bot added by TierOneCoordinator)
        return convoState.participants[0] || null;
    }

    private async updateConversationConfig(
        conversationId: string, 
        updates: Partial<ChatConfigObject>,
    ): Promise<void> {
        // Update config in state store
        this.logger.debug(`Updating conversation config for ${conversationId}`, updates);
    }

    private async getDrainDelay(conversationId: string): Promise<number> {
        // Could get from config, for now return 0 for immediate processing
        return 0;
    }

    private async handleExternalMessage(event: SwarmEvent): Promise<void> {
        // Handle external messages by routing to appropriate agents
        this.logger.info("[SwarmStateMachine] Handling external message", {
            conversationId: event.conversationId,
        });

        await this.errorHandler.execute(async () => {
            const convoState = await this.getConversationState(event.conversationId);
            if (!convoState) {
                this.logger.error(`Conversation state not found for ${event.conversationId}`);
                return;
            }

            // Get the leader bot to handle the message
            const leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                this.logger.error(`No leader bot found for ${event.conversationId}`);
                return;
            }

            // Generate response through ConversationBridge
            const messageText = event.payload?.message?.text || event.payload?.text || "External message received";
            const response = await this.conversationBridge.generateAgentResponse(
                leaderBot,
                { state: this.state, messageReceived: true },
                convoState.config,
                `Process this message: "${messageText}"`,
                event.conversationId,
            );

            this.logger.info("[SwarmStateMachine] Generated response to external message", {
                conversationId: event.conversationId,
                responseLength: response.length,
            });

            // Emit response event for monitoring
            await this.publishUnifiedEvent(
                EventTypes.ROUTINE_STEP_COMPLETED,
                {
                    runId: event.conversationId,
                    stepId: generatePK(),
                    stepType: "message_processing",
                    duration: 0,
                    creditsUsed: "0",
                    success: true,
                    outputs: {
                        messageProcessed: messageText,
                        responseGenerated: true,
                    },
                },
                {
                    conversationId: event.conversationId,
                    priority: "medium",
                    deliveryGuarantee: "fire-and-forget",
                },
            );
        }, "handleExternalMessage", { conversationId: event.conversationId });
    }

    private async handleApprovedTool(event: SwarmEvent): Promise<void> {
        // Handle approved tool execution
        this.logger.info("[SwarmStateMachine] Handling approved tool", {
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName,
        });

        await this.errorHandler.execute(async () => {
            const pendingToolCall = event.payload?.pendingToolCall;
            if (!pendingToolCall) {
                this.logger.error("[SwarmStateMachine] No pending tool call in approved tool event");
                return;
            }

            const convoState = await this.getConversationState(event.conversationId);
            if (!convoState) {
                this.logger.error(`Conversation state not found for ${event.conversationId}`);
                return;
            }

            // Get the leader bot to execute the tool
            const leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                this.logger.error(`No leader bot found for ${event.conversationId}`);
                return;
            }

            // Execute the approved tool through ConversationBridge
            const toolPrompt = `Execute the approved tool: ${pendingToolCall.toolName} with parameters: ${JSON.stringify(pendingToolCall.params)}`;
            const response = await this.conversationBridge.generateAgentResponse(
                leaderBot,
                { 
                    state: this.state, 
                    toolExecution: true,
                    approvedTool: pendingToolCall, 
                },
                convoState.config,
                toolPrompt,
                event.conversationId,
            );

            this.logger.info("[SwarmStateMachine] Executed approved tool", {
                conversationId: event.conversationId,
                toolName: pendingToolCall.toolName,
                responseLength: response.length,
            });

            // Emit tool execution event
            await this.publishUnifiedEvent(
                EventTypes.TOOL_COMPLETED,
                {
                    toolName: pendingToolCall.toolName,
                    toolCallId: pendingToolCall.id || generatePK(),
                    parameters: pendingToolCall.params,
                    result: { approved: true, executed: true },
                    duration: 0,
                    creditsUsed: "0",
                },
                {
                    conversationId: event.conversationId,
                    priority: "medium",
                    deliveryGuarantee: "reliable",
                },
            );
        }, "handleApprovedTool", { 
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName, 
        });
    }

    private async handleRejectedTool(event: SwarmEvent): Promise<void> {
        // Handle rejected tool
        this.logger.info("[SwarmStateMachine] Handling rejected tool", {
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName,
            reason: event.payload?.reason,
        });

        await this.errorHandler.execute(async () => {
            const pendingToolCall = event.payload?.pendingToolCall;
            const rejectionReason = event.payload?.reason || "Tool use was rejected";

            if (!pendingToolCall) {
                this.logger.error("[SwarmStateMachine] No pending tool call in rejected tool event");
                return;
            }

            const convoState = await this.getConversationState(event.conversationId);
            if (!convoState) {
                this.logger.error(`Conversation state not found for ${event.conversationId}`);
                return;
            }

            // Get the leader bot to handle the rejection
            const leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                this.logger.error(`No leader bot found for ${event.conversationId}`);
                return;
            }

            // Generate alternative approach through ConversationBridge
            const fallbackPrompt = `The tool "${pendingToolCall.toolName}" was rejected. Reason: ${rejectionReason}. Please find an alternative approach to accomplish the goal without using this tool.`;
            const response = await this.conversationBridge.generateAgentResponse(
                leaderBot,
                { 
                    state: this.state, 
                    toolRejected: true,
                    rejectedTool: pendingToolCall,
                    rejectionReason, 
                },
                convoState.config,
                fallbackPrompt,
                event.conversationId,
            );

            this.logger.info("[SwarmStateMachine] Generated fallback strategy for rejected tool", {
                conversationId: event.conversationId,
                toolName: pendingToolCall.toolName,
                responseLength: response.length,
            });

            // Emit tool rejection handled event
            await this.publishUnifiedEvent(
                EventTypes.TOOL_APPROVAL_REJECTED,
                {
                    toolName: pendingToolCall.toolName,
                    toolCallId: pendingToolCall.id || generatePK(),
                    pendingId: pendingToolCall.pendingId,
                    rejectedBy: event.sessionUser?.id,
                    reason: rejectionReason,
                },
                {
                    conversationId: event.conversationId,
                    priority: "medium",
                    deliveryGuarantee: "reliable",
                },
            );
        }, "handleRejectedTool", { 
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName, 
        });
    }

    private async handleInternalTaskAssignment(event: SwarmEvent): Promise<void> {
        // Handle internal task assignment (e.g., run execution requests)
        this.logger.info("[SwarmStateMachine] Handling internal task assignment", {
            conversationId: event.conversationId,
            payload: event.payload,
        });

        await this.errorHandler.execute(async () => {
            const convoState = await this.getConversationState(event.conversationId);
            if (!convoState) {
                this.logger.error(`Conversation state not found for ${event.conversationId}`);
                return;
            }

            // Get the leader bot to handle the task assignment
            const leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                this.logger.error(`No leader bot found for ${event.conversationId}`);
                return;
            }

            // Generate task coordination response
            const taskPrompt = `A new task has been assigned: ${JSON.stringify(event.payload)}. Coordinate the team to handle this task.`;
            const response = await this.conversationBridge.generateAgentResponse(
                leaderBot,
                { 
                    state: this.state, 
                    taskAssignment: true,
                    assignedTask: event.payload, 
                },
                convoState.config,
                taskPrompt,
                event.conversationId,
            );

            this.logger.info("[SwarmStateMachine] Coordinated task assignment", {
                conversationId: event.conversationId,
                taskType: event.payload?.runId ? "run_execution" : "general_task",
                responseLength: response.length,
            });

            // Emit task coordination event
            await this.publishUnifiedEvent(
                EventTypes.STATE_TASK_UPDATED,
                {
                    entityType: "task",
                    entityId: event.payload?.taskId || generatePK(),
                    newState: "assigned",
                    message: "Task assigned to swarm for coordination",
                    metadata: {
                        swarmId: event.conversationId,
                        taskPayload: event.payload,
                        taskType: event.payload?.runId ? "run_execution" : "general_task",
                    },
                },
                {
                    conversationId: event.conversationId,
                    priority: "high",
                    deliveryGuarantee: "reliable",
                },
            );
        }, "handleInternalTaskAssignment", { conversationId: event.conversationId });
    }

    private async handleInternalStatusUpdate(event: SwarmEvent): Promise<void> {
        // Handle internal status updates (e.g., run completion, resource alerts)
        this.logger.info("[SwarmStateMachine] Handling internal status update", {
            conversationId: event.conversationId,
            updateType: event.payload?.type,
        });

        await this.errorHandler.execute(async () => {
            const convoState = await this.getConversationState(event.conversationId);
            if (!convoState) {
                this.logger.error(`Conversation state not found for ${event.conversationId}`);
                return;
            }

            // Get the leader bot to process the status update
            const leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                this.logger.error(`No leader bot found for ${event.conversationId}`);
                return;
            }

            // Generate status processing response based on update type
            let statusPrompt = `Status update received: ${JSON.stringify(event.payload)}`;
            
            switch (event.payload?.type) {
                case "run_completed":
                    statusPrompt = `Run ${event.payload.runId} has completed successfully. Update the team and plan next actions.`;
                    break;
                case "run_failed":
                    statusPrompt = `Run ${event.payload.runId} has failed with error: ${event.payload.error}. Analyze the failure and determine recovery actions.`;
                    break;
                case "resource_alert":
                    statusPrompt = `Resource alert: ${JSON.stringify(event.payload)}. Review resource usage and optimize if needed.`;
                    break;
                case "metacognitive_insight":
                    statusPrompt = `Metacognitive insight received: ${JSON.stringify(event.payload)}. Incorporate this insight into swarm operations.`;
                    break;
                case "child_swarm_completed":
                    statusPrompt = `Child swarm ${event.payload.childSwarmId} has completed. Integrate results and continue with parent swarm goals.`;
                    break;
                case "child_swarm_failed":
                    statusPrompt = `Child swarm ${event.payload.childSwarmId} has failed: ${event.payload.error}. Assess impact and adjust strategy.`;
                    break;
                default:
                    statusPrompt = `Status update received: ${JSON.stringify(event.payload)}. Process this information and take appropriate action.`;
            }

            const response = await this.conversationBridge.generateAgentResponse(
                leaderBot,
                { 
                    state: this.state, 
                    statusUpdate: true,
                    updateData: event.payload, 
                },
                convoState.config,
                statusPrompt,
                event.conversationId,
            );

            this.logger.info("[SwarmStateMachine] Processed status update", {
                conversationId: event.conversationId,
                updateType: event.payload?.type,
                responseLength: response.length,
            });

            // Emit status processing event based on update type
            const eventType = event.payload?.type === "run_completed" ? EventTypes.ROUTINE_COMPLETED :
                             event.payload?.type === "run_failed" ? EventTypes.ROUTINE_FAILED :
                             event.payload?.type === "resource_alert" ? EventTypes.RESOURCE_EXHAUSTED :
                             EventTypes.STATE_SWARM_UPDATED;
                             
            const eventData = event.payload?.type === "run_completed" || event.payload?.type === "run_failed" ? {
                runId: event.payload.runId,
                totalDuration: event.payload.duration || 0,
                creditsUsed: event.payload.creditsUsed || "0",
                error: event.payload?.error,
            } : event.payload?.type === "resource_alert" ? {
                entityType: "swarm",
                entityId: event.conversationId,
                resourceType: event.payload.resourceType || "credits",
                threshold: event.payload.threshold,
                current: event.payload.current,
            } : {
                entityType: "swarm",
                entityId: event.conversationId,
                newState: this.state,
                message: `Status update processed: ${event.payload?.type}`,
                metadata: event.payload,
            };
            
            await this.publishUnifiedEvent(
                eventType,
                eventData,
                {
                    conversationId: event.conversationId,
                    priority: "medium",
                    deliveryGuarantee: "reliable",
                },
            );
        }, "handleInternalStatusUpdate", { 
            conversationId: event.conversationId,
            updateType: event.payload?.type, 
        });
    }
}

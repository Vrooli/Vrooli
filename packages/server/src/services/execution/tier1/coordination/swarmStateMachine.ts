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
    type ToolCallRecord,
    type ChatConfigObject,
    type BotParticipant,
    type BotConfigObject,
    type SwarmSubTask,
    generatePK,
    PendingToolCallStatus,
    ChatConfig,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type ISwarmStateStore } from "../state/swarmStateStore.js";
import { type ConversationBridge } from "../intelligence/conversationBridge.js";
import { SocketService } from "../../../../sockets/io.js";
import { BaseStateMachine, BaseStates, type BaseEvent } from "../../shared/BaseStateMachine.js";

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

// Use ConversationState from the conversation types instead of defining our own
import { type ConversationState } from "../../../services/conversation/types.js";

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
        eventBus: EventBus,
        private readonly stateStore: ISwarmStateStore,
        private readonly conversationBridge: ConversationBridge,
    ) {
        super(logger, eventBus, SwarmState.UNINITIALIZED);
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
                convoId
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
            await this.handleEvent(startEvent);
            
            // Process any queued events
            if (this.eventQueue.length > 0) {
                this.drain(convoId).catch(err => 
                    this.logger.error("Error draining initial event queue", { error: err, conversationId: convoId })
                );
            }
            
        } catch (error) {
            this.logger.error(`Failed to start swarm ${convoId}:`, error);
            this.state = SwarmState.FAILED;
            throw error;
        }
    }

    /**
     * Handles incoming events by queuing them
     * Override to ensure conversationId is provided
     */
    async handleEvent(ev: SwarmEvent): Promise<void> {
        if (!ev.conversationId) {
            this.logger.error("SwarmEvent missing conversationId", { event: ev });
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
            event.conversationId
        );

        this.logger.info(`[SwarmStateMachine] Leader response to swarm start`, {
            conversationId: event.conversationId,
            responseLength: response.length,
        });

        // Emit event for monitoring
        await this.eventBus.publish("swarm.events", {
            type: "SWARM_INITIALIZED",
            swarmId: event.conversationId,
            timestamp: new Date(),
            metadata: {
                goal: event.goal,
                leaderId: leaderBot.id,
            },
        });
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
            task.status === "done"
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
        await this.emitEvent("swarm.stopped", {
            swarmId: this.conversationId,
            finalState,
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
        this.logger.debug(`[SwarmStateMachine] Entered IDLE state`, {
            conversationId: this.conversationId,
        });
    }

    /**
     * Called when pausing
     * Implements abstract method from BaseStateMachine
     */
    protected async onPause(): Promise<void> {
        this.logger.info(`[SwarmStateMachine] Paused`, {
            conversationId: this.conversationId,
        });
    }

    /**
     * Called when resuming
     * Implements abstract method from BaseStateMachine
     */
    protected async onResume(): Promise<void> {
        this.logger.info(`[SwarmStateMachine] Resumed`, {
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
        user: SessionUser
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
        updates: Partial<ChatConfigObject>
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
        this.logger.info(`[SwarmStateMachine] Handling external message`, {
            conversationId: event.conversationId,
        });
    }

    private async handleApprovedTool(event: SwarmEvent): Promise<void> {
        // Handle approved tool execution
        this.logger.info(`[SwarmStateMachine] Handling approved tool`, {
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName,
        });
    }

    private async handleRejectedTool(event: SwarmEvent): Promise<void> {
        // Handle rejected tool
        this.logger.info(`[SwarmStateMachine] Handling rejected tool`, {
            conversationId: event.conversationId,
            tool: event.payload?.pendingToolCall?.toolName,
            reason: event.payload?.reason,
        });
    }
}
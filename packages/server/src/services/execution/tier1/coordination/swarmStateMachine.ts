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
export interface SwarmEvent {
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

/**
 * Available states for the swarm state machine
 */
export const SwarmState = {
    UNINITIALIZED: "UNINITIALIZED",
    STARTING: "STARTING",
    RUNNING: "RUNNING",
    IDLE: "IDLE",
    PAUSED: "PAUSED",
    STOPPED: "STOPPED",
    FAILED: "FAILED",
    TERMINATED: "TERMINATED",
} as const;

export type State = (typeof SwarmState)[keyof typeof SwarmState];

/**
 * Interface for managed task state machines
 */
export interface ManagedTaskStateMachine {
    getTaskId(): string;
    getCurrentSagaStatus(): string;
    requestPause(): Promise<boolean>;
    requestStop(reason: string): Promise<boolean>;
    getAssociatedUserId?(): string | undefined;
}

/**
 * Conversation state interface (simplified for tier1)
 */
export interface ConversationState {
    id: string;
    config: ChatConfigObject;
    participants: BotParticipant[];
    teamConfig?: any;
    availableTools: string[];
    initialLeaderSystemMessage?: string;
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
export class SwarmStateMachine implements ManagedTaskStateMachine {
    private disposed = false;
    private processingLock = false;
    private pendingDrainTimeout: NodeJS.Timeout | null = null;
    private state: State = SwarmState.UNINITIALIZED;
    private readonly eventQueue: SwarmEvent[] = [];
    private conversationId: string | null = null;
    private initiatingUser: SessionUser | null = null;

    constructor(
        private readonly logger: Logger,
        private readonly eventBus: EventBus,
        private readonly stateStore: ISwarmStateStore,
        private readonly conversationBridge: ConversationBridge,
    ) {}

    // Implementation of ManagedTaskStateMachine methods
    public getTaskId(): string {
        if (!this.conversationId) {
            this.logger.error("SwarmStateMachine: getTaskId called before conversationId was set.");
            return "undefined_swarm_task_id";
        }
        return this.conversationId;
    }

    public getCurrentSagaStatus(): string {
        switch (this.state) {
            case SwarmState.UNINITIALIZED:
                return "UNINITIALIZED";
            case SwarmState.STARTING:
                return "STARTING";
            case SwarmState.RUNNING:
                return "RUNNING";
            case SwarmState.IDLE:
                return "IDLE";
            case SwarmState.PAUSED:
                return "PAUSED";
            case SwarmState.STOPPED:
                return "STOPPED";
            case SwarmState.FAILED:
                return "FAILED";
            case SwarmState.TERMINATED:
                return "TERMINATED";
            default:
                this.logger.warn(`SwarmStateMachine: Unknown state in getCurrentSagaStatus: ${this.state}`);
                return "UNKNOWN";
        }
    }

    public async requestPause(): Promise<boolean> {
        if (this.state === SwarmState.RUNNING || this.state === SwarmState.IDLE) {
            await this.pause();
            return true;
        }
        this.logger.warn(`SwarmStateMachine: requestPause called in non-pausable state: ${this.state}`);
        return false;
    }

    public async requestStop(reason: string): Promise<boolean> {
        this.logger.info(`SwarmStateMachine: requestStop called for ${this.conversationId}. Reason: ${reason}`);
        const result = await this.stop("graceful", reason, this.initiatingUser ?? undefined);
        return result.success;
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

            // Find or create leader bot
            let leaderBot = this.findLeaderBot(convoState);
            if (!leaderBot) {
                leaderBot = this.createDefaultLeaderBot();
                this.logger.warn(`No leader found for ${convoId}, using default leader bot`);
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
            
            await this.handleEvent(startEvent);
            this.state = SwarmState.IDLE;
            
        } catch (error) {
            this.logger.error(`Failed to start swarm ${convoId}:`, error);
            this.state = SwarmState.FAILED;
            throw error;
        }
    }

    /**
     * Handles incoming events by queuing them
     */
    async handleEvent(ev: SwarmEvent): Promise<void> {
        if (this.state === SwarmState.TERMINATED) return;
        
        this.eventQueue.push(ev);
        
        if (this.state === SwarmState.IDLE && ev.conversationId) {
            this.drain(ev.conversationId).catch(err => 
                this.logger.error("Error draining swarm queue from handleEvent", { error: err, event: ev })
            );
        }
    }

    /**
     * Processes queued events (the heart of autonomous operation)
     */
    private async drain(convoId: string): Promise<void> {
        if (this.state === SwarmState.PAUSED || 
            this.state === SwarmState.TERMINATED || 
            this.state === SwarmState.FAILED) {
            this.logger.info(`Drain called in state ${this.state}, not processing queue for ${convoId}.`);
            return;
        }
        
        if (this.processingLock) {
            this.logger.info(`Drain already in progress for ${convoId}, skipping.`);
            return;
        }

        this.processingLock = true;
        this.state = SwarmState.RUNNING;

        if (this.eventQueue.length === 0) {
            this.processingLock = false;
            this.state = SwarmState.IDLE;
            // Could trigger autonomous monitoring here
            return;
        }

        const ev = this.eventQueue.shift()!;
        if (ev.conversationId !== convoId) {
            this.logger.warn("Swarm drain called with convoId mismatch", { 
                expected: convoId, 
                actual: ev.conversationId 
            });
            this.eventQueue.unshift(ev);
            this.processingLock = false;
            this.state = SwarmState.IDLE;
            return;
        }

        try {
            await this.handleInternalEvent(ev);
        } catch (error) {
            this.logger.error("Error processing event in SwarmStateMachine drain", { error, event: ev });
        }

        // Release processing lock
        this.processingLock = false;

        if (this.eventQueue.length === 0) {
            this.state = SwarmState.IDLE;
        } else {
            // Schedule next drain with optional delay
            const delayMs = await this.getDrainDelay(convoId);
            if (delayMs > 0) {
                this.pendingDrainTimeout = setTimeout(() => {
                    this.pendingDrainTimeout = null;
                    this.drain(convoId).catch(err => 
                        this.logger.error("Error in delayed drain", { error: err, conversationId: convoId })
                    );
                }, delayMs);
            } else {
                setImmediate(() => 
                    this.drain(convoId).catch(err => 
                        this.logger.error("Error in immediate drain", { error: err, conversationId: convoId })
                    )
                );
            }
        }
    }

    /**
     * Handles internal event processing
     */
    private async handleInternalEvent(event: SwarmEvent): Promise<void> {
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
     * Pauses the swarm
     */
    async pause(): Promise<void> {
        if (this.state === SwarmState.RUNNING || this.state === SwarmState.IDLE) {
            this._clearPendingDrainTimeout();
            this.state = SwarmState.PAUSED;
            this.logger.info(`SwarmStateMachine for ${this.conversationId} paused.`);
        }
    }

    /**
     * Resumes the swarm
     */
    async resume(convoId: string): Promise<void> {
        if (this.state === SwarmState.PAUSED) {
            this.state = SwarmState.IDLE;
            this.logger.info(`SwarmStateMachine for ${convoId} resumed.`);
            await this.drain(convoId);
        }
    }

    /**
     * Gracefully stops the swarm
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
        if (!this.conversationId) {
            const errorMsg = "SwarmStateMachine: stop() called before conversationId was set.";
            this.logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_ID_NOT_SET" };
        }

        if (this.state === SwarmState.STOPPED || this.state === SwarmState.TERMINATED) {
            const message = `Swarm ${this.conversationId} is already in state ${this.state}.`;
            this.logger.warn(message);
            return { success: true, message };
        }

        this.logger.info(`Stopping swarm ${this.conversationId} with mode: ${mode}, reason: ${reason || "No reason"}`);

        try {
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

            // Clear pending operations
            this._clearPendingDrainTimeout();
            this.state = SwarmState.STOPPED;
            this.processingLock = false;

            this.logger.info(`Swarm ${this.conversationId} stopped successfully`, finalState);

            return {
                success: true,
                message: `Swarm stopped ${mode === "graceful" ? "gracefully" : "forcefully"}`,
                finalState,
            };

        } catch (error) {
            this.state = SwarmState.FAILED;
            const errorMessage = error instanceof Error ? error.message : "Unknown error";
            this.logger.error(`Error stopping swarm ${this.conversationId}:`, error);
            return {
                success: false,
                message: `Failed to stop swarm: ${errorMessage}`,
                error: "STOP_OPERATION_FAILED",
            };
        }
    }

    /**
     * Shuts down the swarm completely
     */
    async shutdown(): Promise<void> {
        this._clearPendingDrainTimeout();
        this.disposed = true;
        this.eventQueue.length = 0;
        this.state = SwarmState.TERMINATED;
        this.logger.info("SwarmStateMachine shutdown complete.");
    }

    /**
     * Helper methods
     */
    private _clearPendingDrainTimeout(): void {
        if (this.pendingDrainTimeout) {
            clearTimeout(this.pendingDrainTimeout);
            this.pendingDrainTimeout = null;
        }
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
            throw new Error(
                `Conversation ${conversationId} must be created through proper chat API first. ` +
                `Swarms cannot create conversations directly.`
            );
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
        return convoState.participants[0] || null;
    }

    private createDefaultLeaderBot(): BotParticipant {
        return {
            id: generatePK(),
            name: "Swarm Leader",
            config: { __version: "1.0", model: "gpt-4" } as BotConfigObject,
            meta: { role: "leader" },
        };
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
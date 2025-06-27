/**
 * Socket Event Adapter
 * 
 * Bridges the unified event system with socket.io emissions.
 * Subscribes to relevant events on the event bus and transforms them
 * into socket events for real-time client updates.
 * 
 * This adapter enables:
 * - Agents to monitor and modify socket communications
 * - Better separation of concerns (events vs socket transport)
 * - Easier testing and debugging
 * - Support for emergent capabilities through agent-driven socket events
 */

import { type Logger } from "winston";
import { type IEventBus } from "../eventBus.js";
import { type BaseEvent, type EventMetadata } from "../types.js";
import { EventTypes, EventPatterns } from "../index.js";
import { SocketService } from "../../../sockets/SocketService.js";
import {
    type SwarmSocketEventPayloads,
    type RunSocketEventPayloads,
    type ExecutionState,
    type Swarm,
    type SwarmConfiguration,
    ExecutionStates,
} from "@vrooli/shared";
import { SwarmSocketEmitter } from "../../swarmSocketEmitter.js";

/**
 * State update event for socket broadcast
 */
export interface StateUpdateEvent extends BaseEvent {
    type: `state/${string}`;
    data: {
        entityType: "swarm" | "run" | "task";
        entityId: string;
        oldState?: string;
        newState: string;
        message?: string;
        metadata?: Record<string, unknown>;
    };
    metadata: EventMetadata & {
        socketRooms?: string[];
        broadcast?: boolean;
    };
}

/**
 * Resource update event for socket broadcast
 */
export interface ResourceUpdateEvent extends BaseEvent {
    type: `resource/${string}`;
    data: {
        entityType: "swarm" | "user";
        entityId: string;
        resourceType: "credits" | "tokens" | "time";
        allocated: string;
        consumed: string;
        remaining: string;
        metadata?: Record<string, unknown>;
    };
    metadata: EventMetadata & {
        socketRooms?: string[];
    };
}

/**
 * Team update event for socket broadcast
 */
export interface TeamUpdateEvent extends BaseEvent {
    type: `team/${string}`;
    data: {
        swarmId: string;
        teamId?: string;
        swarmLeader?: string;
        subtaskLeaders?: Record<string, string>;
        metadata?: Record<string, unknown>;
    };
    metadata: EventMetadata & {
        socketRooms?: string[];
    };
}

/**
 * Configuration update event for socket broadcast
 */
export interface ConfigUpdateEvent extends BaseEvent {
    type: `config/${string}`;
    data: {
        entityType: "swarm" | "routine";
        entityId: string;
        config: Record<string, unknown>;
        metadata?: Record<string, unknown>;
    };
    metadata: EventMetadata & {
        socketRooms?: string[];
    };
}

/**
 * Socket event adapter for bridging event bus to socket.io
 */
export class SocketEventAdapter {
    private readonly logger: Logger;
    private readonly eventBus: IEventBus;
    private readonly socketService: SocketService;
    private readonly swarmEmitter: SwarmSocketEmitter;
    private subscriptionIds: string[] = [];
    private isStarted = false;

    constructor(
        eventBus: IEventBus,
        logger: Logger,
        socketService?: SocketService,
        swarmEmitter?: SwarmSocketEmitter,
    ) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.socketService = socketService || SocketService.get();
        this.swarmEmitter = swarmEmitter || SwarmSocketEmitter.get();
    }

    /**
     * Start the adapter and setup event subscriptions
     */
    async start(): Promise<void> {
        if (this.isStarted) {
            this.logger.warn("[SocketEventAdapter] Already started");
            return;
        }

        try {
            await this.setupSubscriptions();
            this.isStarted = true;
            this.logger.info("[SocketEventAdapter] Started successfully");
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to start", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Stop the adapter and cleanup subscriptions
     */
    async stop(): Promise<void> {
        if (!this.isStarted) {
            return;
        }

        try {
            // Unsubscribe from all events
            for (const subscriptionId of this.subscriptionIds) {
                await this.eventBus.unsubscribe(subscriptionId);
            }
            this.subscriptionIds = [];
            this.isStarted = false;
            this.logger.info("[SocketEventAdapter] Stopped successfully");
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to stop cleanly", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Setup all event subscriptions
     */
    private async setupSubscriptions(): Promise<void> {
        // Subscribe to state updates
        const stateId = await this.eventBus.subscribe(
            "state/#",
            this.handleStateUpdate.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(stateId);

        // Subscribe to resource updates
        const resourceId = await this.eventBus.subscribe(
            "resource/#",
            this.handleResourceUpdate.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(resourceId);

        // Subscribe to team updates
        const teamId = await this.eventBus.subscribe(
            "team/#",
            this.handleTeamUpdate.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(teamId);

        // Subscribe to config updates
        const configId = await this.eventBus.subscribe(
            "config/#",
            this.handleConfigUpdate.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(configId);

        // Subscribe to tool approval events
        const toolApprovalId = await this.eventBus.subscribe(
            EventTypes.TOOL_APPROVAL_REQUIRED,
            this.handleToolApproval.bind(this),
            { deliveryGuarantee: "reliable" },
        );
        this.subscriptionIds.push(toolApprovalId);

        // Subscribe to routine lifecycle events
        const routineId = await this.eventBus.subscribe(
            EventPatterns.ROUTINE_EVENTS,
            this.handleRoutineEvent.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(routineId);

        // Subscribe to goal events
        const goalId = await this.eventBus.subscribe(
            EventPatterns.GOAL_EVENTS,
            this.handleGoalEvent.bind(this),
            { deliveryGuarantee: "fire-and-forget" },
        );
        this.subscriptionIds.push(goalId);

        this.logger.info("[SocketEventAdapter] Subscribed to event patterns", {
            subscriptionCount: this.subscriptionIds.length,
        });
    }

    /**
     * Handle state update events
     */
    private async handleStateUpdate(event: StateUpdateEvent): Promise<void> {
        try {
            const { entityType, entityId, newState, message } = event.data;
            const { conversationId, socketRooms } = event.metadata || {};

            if (entityType === "swarm" && conversationId) {
                // Map state to ExecutionState enum
                const executionState = this.mapToExecutionState(newState);
                
                // Emit swarm state update
                const payload: SwarmSocketEventPayloads["swarmStateUpdate"] = {
                    chatId: conversationId,
                    swarmId: entityId,
                    state: executionState,
                    message,
                };

                this.swarmEmitter.emitSwarmStateUpdate(
                    conversationId,
                    entityId,
                    executionState,
                    message,
                );

                this.logger.debug("[SocketEventAdapter] Emitted swarm state update", {
                    swarmId: entityId,
                    state: executionState,
                });
            } else if (entityType === "run" && socketRooms?.length) {
                // Emit run state update to specified rooms
                for (const room of socketRooms) {
                    this.socketService.emitSocketEvent(
                        "runTask",
                        room,
                        {
                            runId: entityId,
                            status: newState,
                            message,
                        },
                    );
                }
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle state update", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle resource update events
     */
    private async handleResourceUpdate(event: ResourceUpdateEvent): Promise<void> {
        try {
            const { entityType, entityId, resourceType, allocated, consumed, remaining } = event.data;
            const { conversationId, userId } = event.metadata || {};

            if (entityType === "swarm" && conversationId) {
                // Emit swarm resource update
                const resources = {
                    allocated: parseInt(allocated, 10),
                    consumed: parseInt(consumed, 10),
                    remaining: parseInt(remaining, 10),
                };

                this.swarmEmitter.emitSwarmResourceUpdate(
                    conversationId,
                    entityId,
                    { resources } as any,
                );

                this.logger.debug("[SocketEventAdapter] Emitted swarm resource update", {
                    swarmId: entityId,
                    resourceType,
                });
            } else if (entityType === "user" && userId) {
                // Emit user credit update
                if (resourceType === "credits") {
                    this.socketService.emitSocketEvent(
                        "apiCredits",
                        userId,
                        {
                            credits: remaining,
                            timestamp: new Date().toISOString(),
                        },
                    );
                }
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle resource update", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle team update events
     */
    private async handleTeamUpdate(event: TeamUpdateEvent): Promise<void> {
        try {
            const { swarmId, teamId, swarmLeader, subtaskLeaders } = event.data;
            const { conversationId } = event.metadata || {};

            if (conversationId) {
                this.swarmEmitter.emitSwarmTeamUpdate(
                    conversationId,
                    swarmId,
                    teamId,
                    swarmLeader,
                    subtaskLeaders,
                );

                this.logger.debug("[SocketEventAdapter] Emitted swarm team update", {
                    swarmId,
                    teamId,
                });
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle team update", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle config update events
     */
    private async handleConfigUpdate(event: ConfigUpdateEvent): Promise<void> {
        try {
            const { entityType, entityId, config } = event.data;
            const { conversationId } = event.metadata || {};

            if (entityType === "swarm" && conversationId) {
                this.swarmEmitter.emitSwarmConfigUpdate(conversationId, config);

                this.logger.debug("[SocketEventAdapter] Emitted swarm config update", {
                    swarmId: entityId,
                });
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle config update", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle tool approval events
     */
    private async handleToolApproval(event: BaseEvent): Promise<void> {
        try {
            const data = event.data as any;
            const { conversationId, userId } = event.metadata || {};

            if (conversationId || userId) {
                const room = conversationId || userId;
                this.socketService.emitSocketEvent(
                    "tool_approval_required",
                    room!,
                    {
                        pendingId: data.pendingId,
                        toolCallId: data.toolCallId,
                        toolName: data.toolName,
                        parameters: data.parameters,
                        callerBotId: data.callerBotId,
                        approvalTimeoutAt: data.approvalTimeoutAt,
                    },
                );

                this.logger.debug("[SocketEventAdapter] Emitted tool approval request", {
                    toolName: data.toolName,
                    room,
                });
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle tool approval", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle routine lifecycle events
     */
    private async handleRoutineEvent(event: BaseEvent): Promise<void> {
        try {
            const { conversationId, userId } = event.metadata || {};
            const room = conversationId || userId;

            if (!room) return;

            // Emit routine status updates
            if (event.type === EventTypes.ROUTINE_STARTED) {
                this.socketService.emitSocketEvent("routineStarted", room, event.data);
            } else if (event.type === EventTypes.ROUTINE_COMPLETED) {
                this.socketService.emitSocketEvent("routineCompleted", room, event.data);
            } else if (event.type === EventTypes.ROUTINE_FAILED) {
                this.socketService.emitSocketEvent("routineFailed", room, event.data);
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle routine event", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Handle goal events
     */
    private async handleGoalEvent(event: BaseEvent): Promise<void> {
        try {
            const { conversationId } = event.metadata || {};
            
            if (!conversationId) return;

            // Emit goal updates
            if (event.type === EventTypes.SWARM_GOAL_CREATED) {
                this.socketService.emitSocketEvent("goalCreated", conversationId, event.data);
            } else if (event.type === EventTypes.SWARM_GOAL_COMPLETED) {
                this.socketService.emitSocketEvent("goalCompleted", conversationId, event.data);
            }
        } catch (error) {
            this.logger.error("[SocketEventAdapter] Failed to handle goal event", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Map string state to ExecutionState enum
     */
    private mapToExecutionState(state: string): ExecutionState {
        const normalizedState = state.toUpperCase().replace(/-/g, "_");
        
        // Check if it's a valid ExecutionState
        if (Object.values(ExecutionStates).includes(normalizedState as ExecutionState)) {
            return normalizedState as ExecutionState;
        }

        // Map common state names
        const stateMap: Record<string, ExecutionState> = {
            "INITIALIZING": ExecutionStates.Initializing,
            "RUNNING": ExecutionStates.Running,
            "EXECUTING": ExecutionStates.Running,
            "PAUSED": ExecutionStates.Paused,
            "COMPLETED": ExecutionStates.Completed,
            "SUCCESS": ExecutionStates.Completed,
            "FAILED": ExecutionStates.Failed,
            "ERROR": ExecutionStates.Failed,
            "CANCELLED": ExecutionStates.Cancelled,
            "ABORTED": ExecutionStates.Cancelled,
            "WAITING": ExecutionStates.Paused,
            "PLANNING": ExecutionStates.Initializing,
        };

        return stateMap[normalizedState] || ExecutionStates.Running;
    }

    /**
     * Get adapter statistics
     */
    getStats(): {
        isStarted: boolean;
        subscriptionCount: number;
    } {
        return {
            isStarted: this.isStarted,
            subscriptionCount: this.subscriptionIds.length,
        };
    }
}

/**
 * Create a socket event adapter instance
 */
export function createSocketEventAdapter(
    eventBus: IEventBus,
    logger: Logger,
    socketService?: SocketService,
    swarmEmitter?: SwarmSocketEmitter,
): SocketEventAdapter {
    return new SocketEventAdapter(eventBus, logger, socketService, swarmEmitter);
}
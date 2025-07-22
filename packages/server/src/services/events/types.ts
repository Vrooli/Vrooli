/**
 * Event System Types
 * 
 * Core event types for Vrooli's unified event architecture.
 * Provides essential metadata for routing, correlation, and tracking.
 * 
 * This replaces the fragmented event system with a unified approach that enables
 * true emergent capabilities through agent-extensible event types.
 */

import type { BotParticipant, ChatEventData, RunEventData, SecurityEventData, SocketEvent, SocketEventPayloads, SwarmEventData } from "@vrooli/shared";

/**
 * Event progression control - determines what happens after an event is processed
 */
export type ProgressionControl = "continue" | "block" | "defer" | "retry";

/**
 * Bot response to an event, including progression decision
 */
export interface BotEventResponse {
    /** What should happen next in the event flow */
    progression: ProgressionControl;
    /** Human-readable reason for the decision */
    reason?: string;
    /** Additional data to pass along */
    data?: any;
    /** Should other bots be prevented from processing this event? */
    exclusive?: boolean;
    /** For defer/retry, how long to wait before retrying (ms) */
    retryAfter?: number;
}

/**
 * Event execution modes for the unified model
 */
export enum EventMode {
    /** Emit and continue (old fire-and-forget) */
    PASSIVE = "passive",
    /** Allow interception, but don't wait if no subscribers */
    INTERCEPTABLE = "interceptable",
    /** Always wait for explicit approval */
    APPROVAL = "approval",
    /** Wait for all safety agents */
    CONSENSUS = "consensus",
}

/**
 * Unified event behavior configuration
 */
export interface EventBehavior {
    /** Event execution mode */
    mode: EventMode;
    /** Can bots intercept and handle this event? */
    interceptable: boolean;
    /** Barrier configuration for APPROVAL/CONSENSUS modes */
    barrierConfig?: BarrierSyncConfig;
    /** Default priority level */
    defaultPriority?: "low" | "medium" | "high" | "critical";
}

/**
 * Base event interface that all Vrooli events must implement.
 * Provides essential metadata for routing, correlation, and tracking.
 */
export interface ServiceEvent<
    T extends SocketEventPayloads[SocketEvent] = SocketEventPayloads[SocketEvent]
> {
    /** Unique event identifier */
    id: string;

    /** Event type using hierarchical MQTT-style naming (e.g., "swarm/goal/created") */
    type: string;

    /** When the event occurred */
    timestamp: Date;

    /** Event-specific payload data */
    data: T;

    /** Additional metadata for routing and processing */
    metadata?: EventMetadata;

    /** Progression control for event processing */
    progression?: {
        /** Current progression state */
        state: ProgressionControl;
        /** Bots that have processed this event */
        processedBy: Array<{
            botId: string;
            response: BotEventResponse;
            timestamp: Date;
        }>;
        /** Final progression decision (after all bot processing) */
        finalDecision?: ProgressionControl;
        /** Reason for final decision */
        finalReason?: string;
    };

    /** Execution context for tool and routine events */
    execution?: {
        /** For tool events */
        toolCallId?: string;
        /** For routine events */
        runId?: string;
        /** Originating swarm */
        parentSwarmId?: string;
        /** Preserve full tool context */
        originalToolCall?: {
            function: {
                name: string;
                arguments: any;
            };
            id: string;
        };
    };
}

/**
 * Event metadata for routing and processing
 */
export interface EventMetadata {
    /** Priority for queue processing */
    priority: "low" | "medium" | "high" | "critical";

    /** Delivery guarantee for event processing */
    deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";

    /** User ID for permission-based routing */
    userId?: string;
}

/**
 * Configuration for barrier synchronization (blocking events)
 */
export interface BarrierSyncConfig {
    /** Minimum number of responses required (can be number or "all") */
    quorum: number | "all";

    /** Maximum wait time for responses (ms) */
    timeoutMs: number;

    /** What progression to use on timeout */
    timeoutAction: ProgressionControl;

    /** Which agent types must respond */
    requiredResponders?: string[];

    /** Minimum number of "continue" responses to proceed */
    continueThreshold?: number;

    /** If any bot returns "block", immediately block? */
    blockOnFirst?: boolean;
}

/**
 * Safety events that require consensus from safety agents.
 * These events BLOCK execution until safety agents respond.
 * 
 * @example
 * // Pre-action safety check
 * const safetyEvent: SafetyEvent = {
 *   type: "safety/pre_action",
 *   data: {
 *     action: "execute_financial_trade",
 *     riskLevel: "high",
 *     requiredApprovals: ["fraud_detection", "compliance_check"]
 *   },
 *   metadata: {
 *     priority: "critical"
 *   }
 * }
 */
export interface SafetyEvent extends ServiceEvent {
    type: `safety/${string}` | `emergency/${string}` | `threat/${string}`;
}

/**
 * Tier 1 coordination events for strategic planning and resource allocation.
 * Typically fire-and-forget unless involving resource constraints.
 * 
 * @example
 * // Goal creation
 * const goalEvent: CoordinationEvent = {
 *   type: "swarm/goal/created",
 *   data: {
 *     swarmId: "swarm_123",
 *     goalDescription: "Generate market analysis",
 *     estimatedCredits: 50000,
 *     priority: "high"
 *   }
 * }
 */
export interface CoordinationEvent extends ServiceEvent {
    type: `swarm/${string}` | `goal/${string}` | `team/${string}` | `resource/${string}`;
}

/**
 * Tier 2 process events for routine execution and state management.
 * Completion events use reliable delivery for audit trails.
 * 
 * @example
 * // Routine completion
 * const processEvent: ProcessEvent = {
 *   type: "routine/completed",
 *   data: {
 *     routineId: "routine_456",
 *     runId: "run_789", 
 *     totalDuration: 120000,
 *     creditsUsed: "2500",
 *     outputs: { reportUrl: "https://..." }
 *   },
 *   metadata: { deliveryGuarantee: "reliable" }
 * }
 */
export interface ProcessEvent extends ServiceEvent {
    type: `routine/${string}` | `state/${string}` | `context/${string}`;
}

/**
 * Tier 3 execution events for individual step and tool operations.
 * Most are fire-and-forget for performance, failures use reliable delivery.
 * 
 * @example
 * // Step completion
 * const executionEvent: ExecutionEvent = {
 *   type: "step/completed",
 *   data: {
 *     stepId: "step_101112",
 *     runId: "run_789",
 *     duration: 2500,
 *     creditsUsed: "150",
 *     outputs: { result: "analysis complete" }
 *   }
 * }
 */
export interface ExecutionEvent extends ServiceEvent {
    type: `step/${string}` | `tool/${string}` | `strategy/${string}`;
}

/**
 * Event handler function type
 */
export type EventHandler<T extends ServiceEvent = ServiceEvent> = (event: T) => Promise<void>;

/**
 * Event subscription options
 */
export interface SubscriptionOptions {
    /** Filter events before calling handler */
    filter?: (event: ServiceEvent) => boolean;
    /** Maximum number of events to process per batch */
    batchSize?: number;
    /** Maximum number of retries on handler failure */
    maxRetries?: number;
}

/**
 * Unique subscription identifier
 */
export type EventSubscriptionId = string;

/**
 * Result of publishing an event (unified for all modes)
 */
export interface EventPublishResult {
    /** Whether the event was successfully published */
    success: boolean;
    /** Any error that occurred during publishing */
    error?: Error;
    /** Time taken to publish (ms) */
    duration: number;
    /** The ID of the published event */
    eventId?: string;
    /** For blocking events - the progression result */
    progression?: ProgressionControl;
    /** For blocking events - bot responses */
    responses?: Array<{
        responderId: string;
        response: BotEventResponse;
        timestamp: Date;
    }>;
    /** Whether this was a blocking operation */
    wasBlocking?: boolean;
}


/**
 * Event schema definition for validation
 */
export interface EventSchema<T = unknown> {
    /** Event type pattern this schema applies to */
    eventType: string;
    /** Human-readable description */
    description: string;
    /** JSON schema for validation */
    schema: Record<string, unknown>;
    /** Example event data */
    examples: T[];
}

/**
 * Event type information for agent discovery
 */
export interface EventTypeInfo {
    type: string;
    description: string;
    deliveryGuarantee: "fire-and-forget" | "reliable" | "barrier-sync";
    schema?: Record<string, unknown>;
    examples?: unknown[];
}

/**
 * Event validation result
 */
export interface ValidationResult {
    valid: boolean;
    errors?: string[];
}

/**
 * Unified event bus interface with intelligent barrier handling
 */
export interface IEventBus {
    /**
     * Publish an event with automatic mode detection
     */
    publish(event: Omit<ServiceEvent, "id" | "timestamp">): Promise<EventPublishResult>;

    /**
     * Subscribe to event patterns with optional filtering
     */
    subscribe(
        pattern: string | string[],
        handler: EventHandler,
        options?: SubscriptionOptions
    ): Promise<EventSubscriptionId>;

    /**
     * Unsubscribe from events
     */
    unsubscribe(subscriptionId: EventSubscriptionId): Promise<void>;

    /**
     * Start the event bus
     */
    start(): Promise<void>;

    /**
     * Stop the event bus
     */
    stop(): Promise<void>;

    /**
     * Get subscriber count for a pattern (for optimization)
     */
    getSubscriberCount(pattern: string): number;
}

/**
 * Event subscription information
 */
export interface EventSubscription {
    id: EventSubscriptionId;
    patterns: string[];
    handler: EventHandler;
    options: SubscriptionOptions;
    createdAt: Date;
}

/**
 * Bot decision interfaces for event interception
 */
export interface BotDecisionContext {
    event: ServiceEvent;
    bot: BotParticipant;
    /** Current swarm state for JEXL evaluation */
    swarmState?: any;
}

export interface BotDecision {
    /** Should this bot handle the event? */
    shouldHandle: boolean;
    /** Bot's response if it handles the event */
    response?: BotEventResponse;
    /** Priority for handling order */
    priority?: number;
}

/**
 * Bot decision maker interface - bots implement this
 */
export interface IDecisionMaker {
    decide(context: BotDecisionContext): Promise<BotDecision>;
}

/**
 * Interception result
 */
export interface InterceptionResult {
    /** Was the event intercepted by any bot? */
    intercepted: boolean;
    /** Final progression decision after all bot processing */
    progression: ProgressionControl;
    /** Bot responses */
    responses: Array<{
        botId: string;
        response: BotEventResponse;
    }>;
    /** Aggregated data from all responses */
    aggregatedData?: any;
}


/**
 * Lock service interface for distributed locking
 */
export interface ILockService {
    acquire(key: string, options: LockOptions): Promise<Lock>;
}

export interface Lock {
    release(): Promise<void>;
}

export interface LockOptions {
    ttl: number;       // Time to live in ms
    retries?: number;  // Number of retry attempts
}

/**
 * Type guards for event data types
 */

/**
 * Check if event data is ChatEventData
 */
export function isChatEventData(data: unknown): data is ChatEventData {
    return typeof data === "object" && data !== null && "chatId" in data;
}

/**
 * Check if event data is RunEventData
 */
export function isRunEventData(data: unknown): data is RunEventData {
    return typeof data === "object" && data !== null && "runId" in data;
}

/**
 * Check if event data is SwarmEventData (alias for ChatEventData)
 */
export function isSwarmEventData(data: unknown): data is SwarmEventData {
    return isChatEventData(data);
}

/**
 * Check if event data is SecurityEventData
 */
export function isSecurityEventData(data: unknown): data is SecurityEventData {
    return (
        isChatEventData(data) &&
        "triggeredBy" in data &&
        "severity" in data &&
        typeof (data as any).triggeredBy === "object" &&
        ["info", "warning", "critical"].includes((data as any).severity)
    );
}

/**
 * Extract chat ID from event data if present
 */
export function extractChatId(event: ServiceEvent): string | null {
    if (isChatEventData(event.data)) {
        return event.data.chatId;
    }
    return null;
}

/**
 * Extract run ID from event data if present
 */
export function extractRunId(event: ServiceEvent): string | null {
    if (isRunEventData(event.data)) {
        return event.data.runId;
    }
    return null;
}



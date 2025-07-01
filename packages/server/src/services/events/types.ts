/**
 * Event System Types
 * 
 * Core event types for Vrooli's unified event architecture.
 * Provides essential metadata for routing, correlation, and tracking.
 * 
 * This replaces the fragmented event system with a unified approach that enables
 * true emergent capabilities through agent-extensible event types.
 */

/**
 * Interface for SocketService to avoid circular dependencies
 */
export interface SocketServiceInterface {
    emitSocketEvent<T extends string>(event: T, roomId: string, payload: any): void;
}

/**
 * Base event interface that all Vrooli events must implement.
 * Provides essential metadata for routing, correlation, and tracking.
 */
export interface BaseEvent {
    /** Unique event identifier */
    id: string;

    /** Event type using hierarchical MQTT-style naming (e.g., "swarm/goal/created") */
    type: string;

    /** When the event occurred */
    timestamp: Date;

    /** Where the event originated */
    source: EventSource;

    /** Optional correlation ID for tracking related events */
    correlationId?: string;

    /** Event-specific payload data */
    data: unknown;

    /** Additional metadata for routing and processing */
    metadata?: EventMetadata;
}

/**
 * Event source information for tracing and debugging
 */
export interface EventSource {
    /** Which execution tier (1=coordination, 2=process, 3=execution, "cross-cutting") */
    tier: 1 | 2 | 3 | "cross-cutting" | "safety";

    /** Component name within the tier */
    component: string;

    /** Specific instance ID for distributed systems */
    instanceId?: string;
}

/**
 * Event metadata for delivery and processing
 */
export interface EventMetadata {
    /** Delivery guarantee level */
    deliveryGuarantee: "fire-and-forget" | "reliable" | "barrier-sync";

    /** Priority for queue processing */
    priority: "low" | "medium" | "high" | "critical";

    /** Barrier sync configuration (for approval events) */
    barrierConfig?: BarrierSyncConfig;

    /** Additional tags for filtering */
    tags?: string[];

    /** User ID for permission-based routing */
    userId?: string;

    /** Conversation/swarm ID for context */
    conversationId?: string;
}

/**
 * Configuration for barrier synchronization (blocking events)
 */
export interface BarrierSyncConfig {
    /** Minimum number of OK responses required */
    quorum: number;

    /** Maximum wait time for responses (ms) */
    timeoutMs: number;

    /** Action to take on timeout */
    timeoutAction: "auto-approve" | "auto-reject" | "keep-pending";

    /** Which agent types must respond */
    requiredResponders?: string[];
}

/**
 * Safety events that require barrier synchronization.
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
 *     deliveryGuarantee: "barrier-sync",
 *     barrierConfig: { quorum: 2, timeoutMs: 5000, timeoutAction: "auto-reject" }
 *   }
 * }
 */
export interface SafetyEvent extends BaseEvent {
    source: EventSource & { tier: "safety" };
    type: `safety/${string}` | `emergency/${string}` | `threat/${string}`;
    metadata: EventMetadata & {
        deliveryGuarantee: "barrier-sync";
        barrierConfig: BarrierSyncConfig;
    };
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
export interface CoordinationEvent extends BaseEvent {
    source: EventSource & { tier: 1 };
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
export interface ProcessEvent extends BaseEvent {
    source: EventSource & { tier: 2 };
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
export interface ExecutionEvent extends BaseEvent {
    source: EventSource & { tier: 3 };
    type: `step/${string}` | `tool/${string}` | `strategy/${string}`;
}

/**
 * Goal-related events for tracking objectives and outcomes.
 * Triggered when swarm goals are created, updated, completed, or failed.
 */
export interface GoalEvent extends CoordinationEvent {
    type: "swarm/goal/created" | "swarm/goal/updated" | "swarm/goal/completed" | "swarm/goal/failed";
    data: GoalEventData;
}

export interface GoalEventData {
    swarmId: string;
    goalId?: string;
    goalDescription: string;
    priority: "low" | "medium" | "high" | "critical";
    estimatedCredits?: string;
    deadline?: Date;
    teamId?: string;
    requiredCapabilities?: string[];
    // For updates/completion
    previousGoal?: string;
    completionTime?: Date;
    actualCredits?: string;
    failureReason?: string;
}

/**
 * Tool execution events for external system integration.
 * Includes approval workflow, execution tracking, and cost monitoring.
 */
export interface ToolEvent extends ExecutionEvent {
    type: "tool/called" | "tool/completed" | "tool/failed" |
    "tool/approval_required" | "tool/approval_granted" | "tool/approval_rejected" |
    "tool/approval_timeout" | "tool/approval_cancelled" |
    "tool/scheduled_execution" | "tool/rate_limited";
    data: ToolEventData;
}

export interface ToolEventData {
    toolName: string;
    toolCallId: string;
    parameters?: Record<string, unknown>;
    result?: unknown;
    error?: string;
    duration?: number;
    creditsUsed?: string;
    // Approval-specific
    pendingId?: string;
    callerBotId?: string;
    approvalTimeoutAt?: number;
    approvedBy?: string;
    rejectedBy?: string;
    reason?: string;
    approvalDuration?: number;
    cancellationReason?: string;
    timeoutDuration?: number;
    autoRejected?: boolean;
}

/**
 * Pre-action events that require safety clearance before proceeding.
 * Always use barrier-sync delivery to block execution pending approval.
 */
export interface PreActionEvent extends SafetyEvent {
    type: "safety/pre_action";
    data: PreActionEventData;
    metadata: EventMetadata & {
        deliveryGuarantee: "barrier-sync";
        barrierConfig: BarrierSyncConfig;
    };
}

export interface PreActionEventData {
    action: string;
    context: Record<string, unknown>;
    riskLevel: "low" | "medium" | "high" | "critical";
    requiredApprovals: string[];
    estimatedCost?: string;
    timeoutMs?: number;
    fallbackAction?: "emergency_stop" | "safe_fallback" | "user_prompt";
}

/**
 * Resource rate limiting events for monitoring and control.
 * Emitted when rate limits are exceeded or quotas are approached.
 */
export interface ResourceRateLimitEvent extends BaseEvent {
    type: "resource/rate_limited" | "resource/quota_warning" | "resource/quota_exhausted";
    source: EventSource & { tier: "cross-cutting" };
    data: ResourceRateLimitEventData;
}

export interface ResourceRateLimitEventData {
    originalEventId: string;
    originalEventType: string;
    limitType: "user" | "event_type" | "global" | "cost";
    retryAfterMs?: number;
    userId?: string;
    conversationId?: string;
    quotaRemaining?: number;
    quotaLimit?: number;
    resetTime?: Date;
    costIncurred?: number;
}

/**
 * Event handler function type
 */
export type EventHandler<T extends BaseEvent = BaseEvent> = (event: T) => Promise<void>;

/**
 * Event subscription options
 */
export interface SubscriptionOptions {
    /** Filter events before calling handler */
    filter?: (event: BaseEvent) => boolean;
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
 * Result of publishing an event
 */
export interface EventPublishResult {
    /** Whether the event was successfully published */
    success: boolean;
    /** Any error that occurred during publishing */
    error?: Error;
    /** Time taken to publish (ms) */
    duration: number;
}

/**
 * Result of barrier synchronization
 */
export interface EventBarrierSyncResult {
    success: boolean;
    responses: Array<{ responderId: string; response: "OK" | "ALARM"; reason?: string }>;
    timedOut: boolean;
    duration: number;
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
 * Enhanced event bus interface supporting delivery guarantees and barrier synchronization
 */
export interface IEventBus {
    /**
     * Publish an event with specified delivery guarantee
     */
    publish<T extends BaseEvent>(event: T): Promise<EventPublishResult>;

    /**
     * Subscribe to event patterns with optional filtering
     */
    subscribe<T extends BaseEvent>(
        pattern: string | string[],
        handler: EventHandler<T>,
        options?: SubscriptionOptions
    ): Promise<EventSubscriptionId>;

    /**
     * Unsubscribe from events
     */
    unsubscribe(subscriptionId: EventSubscriptionId): Promise<void>;

    /**
     * Handle barrier sync events (blocking until responses received)
     */
    publishBarrierSync<T extends SafetyEvent>(
        event: T
    ): Promise<EventBarrierSyncResult>;

    /**
     * Start the event bus
     */
    start(): Promise<void>;

    /**
     * Stop the event bus
     */
    stop(): Promise<void>;
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

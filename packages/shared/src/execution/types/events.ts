/**
 * Core type definitions for the event-driven architecture
 * These types define the communication protocol between tiers
 * 
 * IMPORTANT: These types reflect the actual implementation where complex behaviors
 * emerge from AI decisions rather than being hardcoded. The architecture uses
 * simple event types with flexible payloads instead of prescriptive enums.
 */


/**
 * Base event interface matching actual implementation
 * The implementation uses a simpler structure with optional fields
 */
export interface ExecutionEvent {
    id: string;
    type: string;
    timestamp: Date;
    source?: {  // Optional in implementation
        tier: string;  // Actually uses strings like "tier1.swarm"
        component: string;
        instanceId: string;
    };
    correlationId?: string;  // Optional in implementation
    causationId?: string;
    data?: unknown;  // The implementation uses 'data' field for payloads
    metadata?: Record<string, unknown>;  // Simple object, not structured
}

/**
 * Event channel patterns actually used in the implementation
 * These follow hierarchical dot notation or slash notation
 */
export type EventChannel =
    | `execution.metrics.${string}`
    | `execution.events.type.${string}`
    | "strategy/performance/completed"
    | `state.machine.${string}`
    | "metric.recorded"
    | "error.occurred"
    | `swarm.${string}`
    | `run.${string}`
    | `step.${string}`;

/**
 * Swarm event constants for runtime usage
 * These provide the actual string values that can be used at runtime
 */
export const SwarmEventType = {
    // Swarm lifecycle events
    SWARM_STARTED: "swarm_started",
    SWARM_DISSOLVED: "swarm.dissolved",

    // Team management events
    TEAM_FORMED: "team.formed",
    TEAM_UPDATED: "team.updated",
    TEAM_DISSOLVED: "team.dissolved",
    TEAM_MANAGER_SHUTDOWN: "team.manager.shutdown",

    // External interactions
    EXTERNAL_MESSAGE_CREATED: "external_message_created",
    TOOL_APPROVAL_RESPONSE: "tool_approval_response",
    APPROVED_TOOL_EXECUTION_REQUEST: "ApprovedToolExecutionRequest",
    REJECTED_TOOL_EXECUTION_REQUEST: "RejectedToolExecutionRequest",

    // Internal coordination
    INTERNAL_TASK_ASSIGNMENT: "internal_task_assignment",
    INTERNAL_STATUS_UPDATE: "internal_status_update",

    // State changes
    SWARM_STATE_CHANGED: "swarm.state.changed",
} as const;

/**
 * Tier 1 event types as actually implemented
 * Derived from the runtime constants for type safety
 */
export type SwarmEventTypeValues = typeof SwarmEventType[keyof typeof SwarmEventType];

/**
 * Tier 2 event types as actually implemented
 */
export type RunEventType =
    | "START_EXECUTION"
    | "EXECUTION_RESULT"
    | "EXECUTION_ERROR"
    | "run.started"
    | "run.completed"
    | "run.failed"
    | "run.step_ready"
    | "step.navigation";

/**
 * Tier 3 event types as actually implemented
 */
export type StepEventType =
    | "step.started"
    | "step.completed"
    | "step.failed"
    | "strategy.selected"
    | "tool.executed"
    | "EXECUTE_STEP";

/**
 * Cross-cutting event types
 */
export type SystemEventType =
    | "metric.recorded"
    | "error.occurred"
    | "resource.tracked"
    | "security.alert";

/**
 * Event subscription as implemented
 * The actual implementation uses simpler patterns
 */
export interface EventSubscription {
    pattern: string;  // e.g., "swarm.*", "run.completed"
    handler: (event: ExecutionEvent) => void | Promise<void>;
    filter?: (event: ExecutionEvent) => boolean;
}

/**
 * Simple event bus interface matching actual implementation
 */
export interface EventBus {
    emit(event: ExecutionEvent): void;
    on(pattern: string, handler: (event: ExecutionEvent) => void): void;
    off(pattern: string, handler: (event: ExecutionEvent) => void): void;
    subscribe(subscription: EventSubscription): void;
    shutdown(): Promise<void>;
}

/**
 * Event payloads are kept simple and flexible
 * Complex behaviors emerge from AI interpretation of these events
 */

/**
 * Common event data patterns used across tiers
 */
export interface SwarmEventData {
    swarmId: string;
    [key: string]: unknown;
}

export interface RunEventData {
    runId: string;
    [key: string]: unknown;
}

export interface StepEventData {
    stepId: string;
    runId?: string;
    [key: string]: unknown;
}

// ResourceUsage is now imported from core.ts to avoid duplication

/**
 * Event patterns for emergent agent subscriptions
 * Agents can subscribe to these patterns to provide capabilities
 */
export const EMERGENT_EVENT_PATTERNS = {
    // Monitoring agents subscribe to these
    PERFORMANCE: ["*.completed", "*.failed", "metric.recorded"],

    // Security agents subscribe to these
    SECURITY: ["*.failed", "security.*", "tool.executed"],

    // Optimization agents subscribe to these
    OPTIMIZATION: ["step.completed", "resource.tracked", "strategy/performance/*"],

    // Learning agents subscribe to these
    LEARNING: ["*.completed", "adaptation.*", "pattern.discovered"],
} as const;

/**
 * Type guard for execution events
 */
export function isExecutionEvent(event: unknown): event is ExecutionEvent {
    return (
        typeof event === "object" &&
        event !== null &&
        "type" in event &&
        typeof (event as any).type === "string"
    );
}

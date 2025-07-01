/**
 * Unified Event System
 * 
 * This module provides the complete unified event system for Vrooli that replaces
 * the fragmented event architecture with a consistent, emergent-capable system.
 * 
 * Key Features:
 * - MQTT-style hierarchical topics (swarm/*, execution/*, tool/*, etc.)
 * - Consistent BaseEvent format across all components
 * - Three delivery guarantees: fire-and-forget, reliable, barrier-sync
 * - Barrier synchronization for blocking events (tool approval system)
 * - Agent extensibility for emergent capabilities
 * - Type safety with TypeScript definitions
 * 
 * This system enables true emergent intelligence by providing minimal infrastructure
 * with maximum flexibility for specialized agent swarms to implement advanced capabilities.
 */

import { nanoid } from "nanoid";
import { type Logger } from "winston";
import { type ApprovalSystem } from "./approval.js";
import { type EventBus } from "./eventBus.js";
import type { BarrierSyncConfig, BaseEvent, EventMetadata, EventSource, SocketServiceInterface } from "./types.js";

/**
 * Unified Event System Configuration
 */
export interface UnifiedEventSystemConfig {
    /** Backend type */
    backend: "redis" | "memory" | "hybrid";

    /** Redis configuration (if using Redis backend) */
    redis?: {
        url?: string;
        streamName?: string;
        maxHistorySize?: number;
        eventTtl?: number;
    };

    /** Performance settings */
    batchSize?: number;
    maxEventHistory?: number;

    /** Integration settings */
    logger?: Logger;
    enableMetrics?: boolean;
    enableReplay?: boolean;

    /** Socket service for automatic socket event emission */
    socketService?: SocketServiceInterface;
}


/**
 * Unified Event System Factory
 * 
 * Creates a complete event system with all components configured and ready to use.
 */
export interface UnifiedEventSystem {
    eventBus: EventBus;
    approvalSystem: ApprovalSystem;
}

/**
 * Event type constants for easy reference
 */
export const EventTypes = {
    // Goal management
    SWARM_GOAL_CREATED: "swarm/goal/created",
    SWARM_GOAL_UPDATED: "swarm/goal/updated",
    SWARM_GOAL_COMPLETED: "swarm/goal/completed",
    SWARM_GOAL_FAILED: "swarm/goal/failed",

    // Tool execution
    TOOL_CALLED: "tool/called",
    TOOL_COMPLETED: "tool/completed",
    TOOL_FAILED: "tool/failed",
    TOOL_APPROVAL_REQUIRED: "tool/approval_required",
    TOOL_APPROVAL_GRANTED: "tool/approval_granted",
    TOOL_APPROVAL_REJECTED: "tool/approval_rejected",
    TOOL_APPROVAL_TIMEOUT: "tool/approval_timeout",
    TOOL_APPROVAL_CANCELLED: "tool/approval_cancelled",
    TOOL_SCHEDULED_EXECUTION: "tool/scheduled_execution",
    TOOL_RATE_LIMITED: "tool/rate_limited",

    // Routine lifecycle
    ROUTINE_STARTED: "routine/started",
    ROUTINE_COMPLETED: "routine/completed",
    ROUTINE_FAILED: "routine/failed",
    ROUTINE_STEP_COMPLETED: "routine/step_completed",

    // Step execution
    STEP_STARTED: "step/started",
    STEP_COMPLETED: "step/completed",
    STEP_FAILED: "step/failed",
    STEP_RETRIED: "step/retried",

    // Safety events
    SAFETY_PRE_ACTION: "safety/pre_action",
    SAFETY_POST_ACTION: "safety/post_action",
    EMERGENCY_STOP: "emergency/stop",
    THREAT_DETECTED: "threat/detected",

    // Access control events (for emergent security)
    ACCESS_ATTEMPT_ROUTINE: "access/attempt/routine",
    ACCESS_ATTEMPT_RESOURCE: "access/attempt/resource",
    ACCESS_ATTEMPT_API: "access/attempt/api",
    ACCESS_GRANTED: "access/granted",
    ACCESS_DENIED: "access/denied",
    ACCESS_ERROR: "access/error",

    // Execution errors
    EXECUTION_ERROR_OCCURRED: "execution/error/occurred",
    EXECUTION_ERROR_REPORTING_FAILED: "execution/error/reporting_failed",

    // Strategy events
    STRATEGY_PERFORMANCE_MEASURED: "strategy/performance/measured",
    STRATEGY_PERFORMANCE_FAILED: "strategy/performance/failed",
    STRATEGY_FEEDBACK_RECEIVED: "strategy/feedback/received",
    STRATEGY_THRESHOLD_CROSSED: "strategy/threshold/crossed",

    // Recovery events
    RECOVERY_ATTEMPT: "recovery/attempt",
    RECOVERY_OUTCOME: "recovery/outcome",
    FALLBACK_EXECUTED: "fallback/executed",
    CIRCUIT_BREAKER_STATE_CHANGE: "circuit_breaker/state_change",

    // Resource events
    RESOURCE_ALLOCATED: "resource/allocated",
    RESOURCE_EXHAUSTED: "resource/exhausted",
    RESOURCE_OPTIMIZED: "resource/optimized",
    RESOURCE_RELEASED: "resource/released",
    RESOURCE_USAGE: "resource/usage",

    // Team events
    TEAM_FORMED: "team/formed",
    TEAM_MEMBER_ADDED: "team/member_added",
    TEAM_MEMBER_REMOVED: "team/member_removed",
    TEAM_DISBANDED: "team/disbanded",

    // State update events (for socket broadcasts)
    STATE_SWARM_UPDATED: "state/swarm/updated",
    STATE_RUN_UPDATED: "state/run/updated",
    STATE_TASK_UPDATED: "state/task/updated",

    // Config update events
    CONFIG_SWARM_UPDATED: "config/swarm/updated",
    CONFIG_ROUTINE_UPDATED: "config/routine/updated",

    // Team update events  
    TEAM_SWARM_UPDATED: "team/swarm/updated",

    // Resource update events for swarms
    RESOURCE_SWARM_UPDATED: "resource/swarm/updated",
    RESOURCE_USER_UPDATED: "resource/user/updated",

    // Bot status events (for socket broadcasts)
    BOT_STATUS_UPDATED: "bot/status/updated",
    BOT_RESPONSE_STREAM: "bot/response/stream",
    BOT_MODEL_REASONING_STREAM: "bot/reasoning/stream",
    BOT_TYPING_UPDATED: "bot/typing/updated",
} as const;

/**
 * Event pattern constants for subscription
 */
export const EventPatterns = {
    // All events
    ALL: "#",

    // Goal events
    GOAL_EVENTS: "swarm/goal/*",

    // Tool events
    TOOL_EVENTS: "tool/*",
    TOOL_APPROVAL_EVENTS: "tool/approval_*",

    // Routine events
    ROUTINE_EVENTS: "routine/*",

    // Step events
    STEP_EVENTS: "step/*",

    // Safety events
    SAFETY_EVENTS: "safety/*",
    EMERGENCY_EVENTS: "emergency/*",

    // Error events
    ERROR_EVENTS: "execution/error/*",

    // Strategy events
    STRATEGY_EVENTS: "strategy/*",

    // Recovery events
    RECOVERY_EVENTS: "recovery/*",
    FALLBACK_EVENTS: "fallback/*",

    // Resource events
    RESOURCE_EVENTS: "resource/*",

    // Team events
    TEAM_EVENTS: "team/*",

    // Tier-specific events
    TIER1_EVENTS: "swarm/*",
    TIER2_EVENTS: "routine/*",
    TIER3_EVENTS: "step/*",

    // Bot events
    BOT_EVENTS: "bot/*",
    BOT_STATUS_EVENTS: "bot/status/*",
    BOT_RESPONSE_EVENTS: "bot/response/*",
} as const;

/**
 * Utility functions for creating common event structures
 */
// Re-export types for easy access
export type { BaseEvent, EventMetadata, EventSource, IEventBus, SocketServiceInterface } from "./types.js";

export const EventUtils = {
    /**
     * Create a basic event source
     */
    createEventSource(
        tier: 1 | 2 | 3 | "cross-cutting" | "safety",
        component: string,
        instanceId?: string,
    ): EventSource {
        return {
            tier,
            component,
            instanceId: instanceId || nanoid(),
        } as EventSource;
    },

    /**
     * Create basic event metadata
     */
    createEventMetadata(
        deliveryGuarantee: "fire-and-forget" | "reliable" | "barrier-sync" = "fire-and-forget",
        priority: "low" | "medium" | "high" | "critical" = "medium",
        options?: {
            tags?: string[];
            userId?: string;
            conversationId?: string;
            barrierConfig?: BarrierSyncConfig;
        },
    ): EventMetadata {
        return {
            deliveryGuarantee,
            priority,
            tags: options?.tags,
            userId: options?.userId,
            conversationId: options?.conversationId,
            barrierConfig: options?.barrierConfig,
        };
    },

    /**
     * Create a complete base event
     */
    createBaseEvent<T = unknown>(
        type: string,
        data: T,
        source: EventSource,
        metadata?: EventMetadata,
    ): BaseEvent {
        return {
            id: nanoid(),
            type,
            timestamp: new Date(),
            source,
            correlationId: nanoid(),
            data,
            metadata,
        };
    },
};


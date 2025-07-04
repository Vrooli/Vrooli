/**
 * Conversation-Specific Types for ConversationEngine
 * 
 * This module contains only NEW conversation-specific types that don't exist
 * elsewhere. It imports existing types to maintain clean architecture and
 * avoid duplication.
 */

// Import existing base types
import type { ChatMessage } from "../api/types.js";
import type { SocketEvent } from "../consts/socketEvents.js";
import type { BotParticipant, ConversationConstraints, ConversationContext } from "./context.js";
import type { ExecutionError, ExecutionResourceUsage, ExecutionStrategy } from "./core.js";
import type { BotId, TurnId } from "./ids.js";
import type { ResponseResult } from "./results.js";

/**
 * Tool call interface for AI function calls
 */
export interface ToolCall {
    id: string;
    function: {
        name: string;
        arguments: string | Record<string, unknown>;
    };
}

/**
 * Tool definition interface
 */
export interface Tool {
    name: string;
    description: string;
    inputSchema: Record<string, unknown>;
    outputSchema?: Record<string, unknown>;
    category?: string;
    permissions?: string[];
}

/**
 * Trigger types for conversation initiation (NEW - conversation-specific)
 */
export type ConversationTrigger =
    | {
        type: "user_message";
        message: ChatMessage;
    }
    | {
        type: "event";
        event: SocketEvent;
    }
    | {
        type: "scheduled_turn";
        participants: BotId[];
        reason: string;
        scheduledAt: Date;
    }
    | {
        type: "continuation";
        previousTurnId: TurnId;
        reason: string;
        participants?: BotId[];
    };

/**
 * Tool execution result (NEW - conversation-specific)
 */
export interface ToolResult {
    readonly success: boolean;
    readonly output?: unknown;
    readonly error?: ExecutionError;
    readonly toolCall: ToolCall;
    readonly executionTime: number;
    readonly creditsUsed: string;
}

/**
 * Parameters for conversation orchestration (NEW - conversation-specific)
 */
export interface ConversationParams {
    readonly context: ConversationContext;
    readonly trigger: ConversationTrigger;
    readonly strategy: ExecutionStrategy;
    readonly constraints?: ConversationConstraints;
    readonly metadata?: Record<string, unknown>;
}

/**
 * Parameters for turn execution (NEW - conversation-specific)
 */
export interface TurnExecutionParams {
    readonly participants: BotParticipant[];
    readonly context: ConversationContext;
    readonly strategy: ExecutionStrategy;
    readonly turnId: TurnId;
    readonly executionMode?: "sequential" | "parallel" | "adaptive";
}

/**
 * Result from executing a conversation turn (NEW - conversation-specific)
 */
export interface TurnExecutionResult {
    readonly messages: ChatMessage[];
    readonly resourcesUsed: ExecutionResourceUsage;
    readonly participantResults: Map<BotId, ResponseResult>;
    readonly turnMetrics?: TurnMetrics;
}

/**
 * Metrics for a single turn (NEW - conversation-specific)
 */
export interface TurnMetrics {
    readonly totalDuration: number;
    readonly participantCount: number;
    readonly messageCount: number;
    readonly toolCallCount: number;
    readonly averageConfidence?: number;
    readonly executionMode: "sequential" | "parallel" | "adaptive";
}

/**
 * Configuration for conversation engine (NEW - conversation-specific)
 */
export interface ConversationEngineConfig {
    readonly maxConcurrentTurns: number;
    readonly defaultTimeoutMs: number;
    readonly enableBotSelection: boolean;
    readonly enableMetrics: boolean;
    readonly fallbackBehavior: "first_available" | "random" | "most_capable";
}

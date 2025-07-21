/**
 * Conversation-Specific Types for ConversationEngine
 * 
 * This module contains only NEW conversation-specific types that don't exist
 * elsewhere. It imports existing types to maintain clean architecture and
 * avoid duplication.
 */

// Import existing base types
import type { ChatMessage } from "../api/types.js";
import type { MessageConfigObject } from "../shape/configs/message.js";
import type { UnifiedEvent } from "../consts/socketEvents.js";
import type { BotParticipant, ConversationConstraints, ConversationContext } from "./context.js";
import type { ExecutionError, ExecutionResourceUsage, ExecutionStrategy } from "./core.js";
import type { BotId, TurnId } from "./ids.js";
import type { ResponseResult } from "./results.js";

/**
 * MessageState - Lightweight Message Representation for Conversation Processing
 * 
 * This is the core message type used by the conversation engine and AI services.
 * It represents a simplified, performance-optimized version of ChatMessage that
 * contains only the essential fields needed for conversation processing.
 * 
 * ## Architectural Principle: Unidirectional Data Flow
 * 
 * MessageState follows the principle of unidirectional data flow:
 * ```
 * ChatMessage (database) → MessageState (conversation) → LLMMessage (AI service)
 * ```
 * 
 * ## Why MessageState Exists:
 * 
 * 1. **Performance**: ChatMessage includes heavy relational data (full User objects,
 *    ReactionSummary arrays, etc.) that slows down conversation processing
 * 
 * 2. **Type Safety**: Prevents the need for unsafe type assertions between
 *    incompatible message representations
 * 
 * 3. **Clear Boundaries**: Establishes a clean interface between the database
 *    layer and conversation processing layer
 * 
 * 4. **Memory Efficiency**: Reduces memory usage in conversation loops that
 *    process many messages
 * 
 * ## Usage Guidelines:
 * 
 * - ✅ Use MessageState for conversation processing and AI service interfaces
 * - ✅ Convert ChatMessage → MessageState when moving from database to conversation layer
 * - ✅ Convert MessageState → LLMMessage when interfacing with AI services
 * - ❌ Never attempt MessageState → ChatMessage conversion (data loss inevitable)
 * - ❌ Never use unsafe type assertions ("as ChatMessage", "as MessageState")
 * 
 * @see ChatMessage - Full database representation with all relations
 * @see LLMMessage - AI service format for LLM APIs
 */
export type MessageState = Pick<ChatMessage, "id" | "createdAt" | "config" | "language" | "text"> & {
    /** Parent message reference (simplified from full relation) */
    parent?: {
        id: string;
    } | null;
    
    /** User reference (simplified from full User object) */
    user?: {
        id: string;
    } | null;
};

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
 * Trigger types for conversation initiation
 */
export type ConversationTrigger =
    | {
        type: "user_message";
        message: Pick<ChatMessage, "config">;
    }
    | {
        type: "start";
    }
    | {
        type: "continue";
        /** Not currently used, but may be useful in the future. */
        lastEvent: UnifiedEvent;
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
    readonly messages: MessageState[];
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
    readonly fallbackBehavior: "first_available" | "random" | "most_capable";
}

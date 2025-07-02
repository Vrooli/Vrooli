/**
 * Conversation-Specific Types for ConversationEngine
 * 
 * This module contains only NEW conversation-specific types that don't exist
 * elsewhere. It imports existing types to maintain clean architecture and
 * avoid duplication.
 */

// Import existing base types
import type { ChatMessage } from "../../api/types.js";
import type { BotParticipant, ConversationConstraints, ConversationContext } from "./context.js";
import type { ExecutionError, ExecutionResourceUsage, ExecutionStrategy } from "./core.js";
import type { BotId, TurnId } from "./ids.js";
import type { ResponseResult } from "./results.js";
import type { SwarmEvent } from "./swarm.js";

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
        expectedResponseCount?: number;
        urgency?: "low" | "medium" | "high";
    }
    | {
        type: "tool_response";
        toolResult: ToolResult;
        requester: BotId;
        followUpNeeded?: boolean;
    }
    | {
        type: "swarm_event";
        event: SwarmEvent;
        affectedBots?: BotId[];
        eventPriority?: "low" | "medium" | "high";
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
 * Context for bot selection intelligence (NEW - conversation-specific)
 */
export interface BotSelectionContext {
    readonly trigger: ConversationTrigger;
    readonly availableParticipants: BotParticipant[];
    readonly conversationHistory: ChatMessage[];
    readonly sharedState: Record<string, unknown>;
    readonly constraints: ConversationConstraints;
    readonly selectionCriteria?: BotSelectionCriteria;
}

/**
 * Criteria for intelligent bot selection (NEW - conversation-specific)
 */
export interface BotSelectionCriteria {
    readonly requiredCapabilities?: string[];
    readonly preferredSpecializations?: string[];
    readonly excludeParticipants?: BotId[];
    readonly maxParticipants?: number;
    readonly diversityWeight?: number; // 0-1, higher = more diverse selection
    readonly expertiseWeight?: number; // 0-1, higher = prefer specialists
}

/**
 * Result from bot selection process (NEW - conversation-specific)
 */
export interface BotSelectionResult {
    readonly selectedBots: BotParticipant[];
    readonly selectionReason: string;
    readonly confidence: number; // 0-1
    readonly fallbackUsed: boolean;
    readonly alternativeOptions?: BotParticipant[];
    readonly selectionMetadata?: Record<string, unknown>;
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
    readonly defaultStrategy: ExecutionStrategy;
    readonly maxConcurrentTurns: number;
    readonly defaultTimeoutMs: number;
    readonly enableBotSelection: boolean;
    readonly enableMetrics: boolean;
    readonly fallbackBehavior: "first_available" | "random" | "most_capable";
}

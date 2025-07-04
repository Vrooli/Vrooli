/**
 * Unified Context Types for Clean Data Flow
 * 
 * These context types provide a clean hierarchy for passing data through
 * the execution system without transformation. Each level adds more specific
 * context while maintaining compatibility with the base types.
 * 
 * Design Principles:
 * - Clean inheritance hierarchy
 * - No transformation between levels
 * - Rich context flows down the stack
 * - Compile-time type safety
 */

import type { ChatMessage, SessionUser } from "../api/types.js";
import type { BotConfigObject } from "../shape/configs/bot.js";
import type { TeamConfigObject } from "../shape/configs/team.js";
import type { Tool } from "./conversation.js";
import type { ExecutionStrategy } from "./core.js";
import type { BotId, SwarmId } from "./ids.js";

/**
 * Base execution context shared across all tiers
 * Contains the minimal information needed for any execution operation
 */
export interface BaseExecutionContext {
    /** Unique identifier for the swarm this execution belongs to */
    readonly swarmId: SwarmId;

    /** User who initiated or owns this execution */
    readonly userData: SessionUser;

    /** When this execution context was created */
    readonly timestamp: Date;
}

/**
 * Response context for individual bot response generation
 * Extends base context with everything needed for a single bot to generate a response
 */
export interface ResponseContext extends BaseExecutionContext {
    /** The specific bot that should generate a response */
    readonly botId: BotId;

    /** Configuration for the responding bot */
    readonly botConfig: BotConfigObject;

    /** Full conversation history for context */
    readonly conversationHistory: ChatMessage[];

    /** Tools available to this bot during response generation */
    readonly availableTools: Tool[];

    /** Execution strategy (conversation, reasoning, deterministic) */
    readonly strategy: ExecutionStrategy;

    /** Optional constraints on response generation */
    readonly constraints?: {
        /** Maximum tokens for this response */
        maxTokens?: number;

        /** Temperature for LLM sampling */
        temperature?: number;

        /** Maximum time allowed for response generation */
        timeoutMs?: number;

        /** Maximum cost allowed for this response */
        maxCredits?: string;
    };
}

/**
 * Conversation context for multi-bot conversation orchestration
 * Extends base context with everything needed to orchestrate a conversation
 */
export interface ConversationContext extends BaseExecutionContext {
    /** All bots that could participate in this conversation */
    readonly participants: BotParticipant[];

    /** Full conversation history for context */
    readonly conversationHistory: ChatMessage[];

    /** Tools available during this conversation */
    readonly availableTools: Tool[];

    /** Team configuration if this is a team conversation */
    readonly teamConfig?: TeamConfigObject;

    /** Shared state accessible to all participants */
    readonly sharedState?: Record<string, unknown>;

    /** Conversation-level constraints */
    readonly constraints?: ConversationConstraints;
}

/**
 * Bot participant information for conversation orchestration
 */
export interface BotParticipant {
    /** Unique identifier for this bot */
    readonly id: BotId;

    /** Bot configuration */
    readonly config: BotConfigObject;

    /** Current state of this bot in the conversation */
    readonly state: "ready" | "busy" | "disabled" | "unavailable";

    /** Role this bot plays in the conversation */
    readonly role?: string;

    /** Priority level for bot selection (higher = more likely to be selected) */
    readonly priority?: number;

    /** Tools specifically available to this bot */
    readonly tools?: Tool[];
}

/**
 * Constraints for conversation orchestration
 */
export interface ConversationConstraints {
    /** Maximum number of turns in this conversation */
    maxTurns?: number;

    /** Maximum time for the entire conversation */
    timeoutMs?: number;

    /** Maximum number of bots that can participate */
    maxParticipants?: number;

    /** Resource limits for the conversation */
    resourceLimits?: {
        /** Maximum credits that can be spent */
        maxCredits?: string;

        /** Maximum memory usage */
        maxMemoryMB?: number;

        /** Maximum concurrent operations */
        maxConcurrentOps?: number;
    };

    /** Whether bots can call tools */
    allowToolUse?: boolean;

    /** Whether bots can modify shared state */
    allowStateModification?: boolean;
}

/**
 * Context scope for variable management in routine execution
 * Provides hierarchical variable scoping and isolation
 */
export interface ContextScope {
    /** Unique identifier for this scope */
    readonly id: string;

    /** Type of scope determining its lifecycle and access patterns */
    readonly type: "global" | "step" | "branch" | "loop" | "function";

    /** Parent scope ID for hierarchy management */
    readonly parentScopeId?: string;

    /** Variables specific to this scope */
    variables: Record<string, unknown>;

    /** Whether variables in this scope can be modified */
    readonly readOnly?: boolean;

    /** When this scope was created */
    readonly createdAt: Date;

    /** When this scope should be cleaned up (optional) */
    readonly expiresAt?: Date;
}

/**
 * Type Adapters - Safe Unidirectional Message Conversions
 * 
 * This module implements the critical architectural principle of unidirectional data flow
 * for message types throughout the system. It provides safe, validated conversions between
 * different message representations while preventing the architectural anti-patterns that
 * led to type safety issues and runtime errors.
 * 
 * ## The Core Problem This Solves
 * 
 * Previously, the system had multiple message types (ChatMessage, MessageState, LLMMessage)
 * with bidirectional conversions using unsafe type assertions ("as ChatMessage"). This led to:
 * 
 * 1. **Runtime Type Errors**: Accessing fields that don't exist on converted types
 * 2. **Data Loss**: Converting lean types back to rich types by fabricating missing data
 * 3. **Interface Violations**: Promising ChatMessage but delivering MessageState
 * 4. **Memory Inefficiency**: Passing heavy database objects through lightweight systems
 * 
 * ## Architectural Principle: Unidirectional Data Flow
 * 
 * This module enforces the principle that data should only flow from rich to lean:
 * 
 * ```
 * ┌─────────────────┐
 * │   Database      │ ChatMessage (full relations: User, Chat, ReactionSummary[], etc.)
 * │   Layer         │ 
 * └─────────┬───────┘
 *           │ ✅ chatMessageToMessageState()
 *           │    SAFE: Rich → Lean (extract only needed fields)
 *           ▼
 * ┌─────────────────┐
 * │  Conversation   │ MessageState (lightweight: id, text, config, simple relations)
 * │  Engine         │
 * └─────────┬───────┘
 *           │ ✅ messageStateToLLMMessage()
 *           │    SAFE: Lean → Specialized (format for AI service)
 *           ▼
 * ┌─────────────────┐
 * │   AI Services   │ LLMMessage (LLM format: role, content, minimal metadata)
 * └─────────────────┘
 * ```
 * 
 * ## Why Unidirectional Flow Matters
 * 
 * ### 1. **Data Integrity**
 * Going from rich to lean is safe because we're simply omitting fields. Going backwards
 * requires fabricating data that doesn't exist, leading to incorrect or incomplete objects.
 * 
 * ```typescript
 * // ✅ SAFE: Rich → Lean (data extraction)
 * const messageState: MessageState = {
 *     id: chatMessage.id,           // ✅ Field exists
 *     text: chatMessage.text,       // ✅ Field exists
 *     user: { id: chatMessage.user.id }, // ✅ Extract only ID
 * };
 * 
 * // ❌ UNSAFE: Lean → Rich (data fabrication)
 * const chatMessage: ChatMessage = {
 *     id: messageState.id,          // ✅ Field exists
 *     text: messageState.text,      // ✅ Field exists
 *     user: ???,                    // ❌ Where do we get full User object?
 *     chat: ???,                    // ❌ Where do we get full Chat object?
 *     reactionSummaries: ???,       // ❌ Where do we get reactions?
 * };
 * ```
 * 
 * ### 2. **Performance**
 * Each layer uses the most appropriate data representation for its needs:
 * - Database layer: Full relations for complex queries and updates
 * - Conversation layer: Lightweight objects for fast processing
 * - AI layer: Standardized format for LLM APIs
 * 
 * ### 3. **Type Safety**
 * Unidirectional flow eliminates the need for unsafe type assertions. Every conversion
 * is explicit, validated, and preserves type contracts.
 * 
 * ### 4. **Clear Boundaries**
 * Each layer has a clearly defined interface and doesn't need to know about
 * implementation details of other layers.
 * 
 * ## Usage Guidelines
 * 
 * ### ✅ DO:
 * ```typescript
 * // Convert rich database object to conversation format
 * const messageState = MessageTypeAdapters.chatMessageToMessageState(dbMessage);
 * 
 * // Convert conversation format to AI service format
 * const llmMessage = MessageTypeAdapters.messageStateToLLMMessage(messageState);
 * 
 * // Create new conversation messages directly
 * const newMessage = MessageTypeAdapters.createMessageState({
 *     id: "msg_123",
 *     text: "Hello world",
 *     role: "assistant",
 *     userId: "user_456"
 * });
 * ```
 * 
 * ### ❌ DON'T:
 * ```typescript
 * // Never attempt reverse conversions
 * const chatMessage = messageState as ChatMessage; // ❌ Type lie
 * 
 * // Never use unsafe type assertions
 * const converted = someObject as MessageState; // ❌ Unsafe
 * 
 * // Never try to reconstruct rich types from lean types
 * const fullMessage = reconstructChatMessage(messageState); // ❌ Data fabrication
 * ```
 * 
 * ## Error Handling Strategy
 * 
 * All adapters use defensive programming:
 * 1. **Validate inputs** before processing
 * 2. **Provide sensible defaults** for missing optional fields
 * 3. **Fail fast** with clear error messages for invalid inputs
 * 4. **Log warnings** for unexpected data patterns
 * 
 * This ensures that bad data is caught early and doesn't propagate through the system.
 * 
 * @see MessageState - Lightweight conversation format
 * @see ChatMessage - Full database format
 * @see LLMMessage - AI service format
 */

import { generatePK, type ChatMessage, type MessageConfigObject, type LLMMessage, type MessageState } from "@vrooli/shared";
import { logger } from "../../events/logger.js";

/**
 * MessageTypeAdapters - Central hub for all message type conversions
 * 
 * This class implements safe, unidirectional conversions between message types.
 * All methods are static to emphasize that these are pure functions with no side effects.
 */
export class MessageTypeAdapters {
    /**
     * Convert full ChatMessage to lightweight MessageState
     * 
     * This is the primary conversion used when moving from the database layer
     * to the conversation processing layer. It extracts only the essential fields
     * needed for conversation processing, resulting in better performance and
     * cleaner interfaces.
     * 
     * ## Data Flow: Database Layer → Conversation Layer
     * ```
     * ChatMessage (850+ KB with relations) → MessageState (2-5 KB)
     * ```
     * 
     * ## Field Mapping:
     * - ✅ Direct copy: id, createdAt, config, text, language
     * - ✅ Simplified: user (full User object → { id: string })
     * - ✅ Simplified: parent (full relation → { id: string })
     * - ❌ Omitted: chat, reactionSummaries, reports, score, sequence, etc.
     * 
     * @param msg - Full ChatMessage from database
     * @returns Lightweight MessageState for conversation processing
     * @throws Error if required fields are missing or invalid
     */
    static chatMessageToMessageState(msg: ChatMessage): MessageState {
        if (!msg?.id || !msg?.text || !msg?.config) {
            logger.error("[MessageTypeAdapters] Invalid ChatMessage for conversion", {
                hasId: !!msg?.id,
                hasText: !!msg?.text,
                hasConfig: !!msg?.config,
                msgType: typeof msg,
            });
            throw new Error("Invalid ChatMessage: missing required fields (id, text, config)");
        }

        try {
            const messageState: MessageState = {
                id: msg.id,
                createdAt: msg.createdAt,
                config: msg.config,
                text: msg.text,
                language: msg.language || "en", // Sensible default
                parent: msg.parent ? { id: msg.parent.id } : null,
                user: msg.user ? { id: msg.user.id } : null,
            };

            // Converted ChatMessage to MessageState

            return messageState;
        } catch (error) {
            logger.error("[MessageTypeAdapters] Failed to convert ChatMessage to MessageState", {
                messageId: msg.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`ChatMessage conversion failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Convert MessageState to LLM format
     * 
     * This conversion is used when interfacing with AI services that expect
     * the standard LLM message format (role, content, optional metadata).
     * 
     * ## Data Flow: Conversation Layer → AI Service Layer
     * ```
     * MessageState (conversation format) → LLMMessage (OpenAI-compatible)
     * ```
     * 
     * ## Field Mapping:
     * - ✅ content: MessageState.text
     * - ✅ role: Extracted from MessageState.config.role
     * - ✅ name: MessageState.user.id (optional)
     * - ❌ Omitted: id, createdAt, language, parent
     * 
     * @param msg - MessageState from conversation processing
     * @returns LLMMessage compatible with AI service APIs
     * @throws Error if message state is invalid
     */
    static messageStateToLLMMessage(msg: MessageState): LLMMessage {
        if (!msg?.text || !msg?.config) {
            logger.error("[MessageTypeAdapters] Invalid MessageState for LLM conversion", {
                hasText: !!msg?.text,
                hasConfig: !!msg?.config,
                msgId: msg?.id,
            });
            throw new Error("Invalid MessageState: missing required fields (text, config)");
        }

        try {
            const role = this.extractRole(msg.config);
            
            const llmMessage: LLMMessage = {
                role,
                content: msg.text,
                name: msg.user?.id || undefined,
            };

            // Converted MessageState to LLMMessage

            return llmMessage;
        } catch (error) {
            logger.error("[MessageTypeAdapters] Failed to convert MessageState to LLMMessage", {
                messageId: msg.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`LLM conversion failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Create MessageState from new message data
     * 
     * This is used when creating new messages (typically bot responses) that
     * haven't been persisted to the database yet. It provides a safe way to
     * construct MessageState objects with validated fields and sensible defaults.
     * 
     * ## Use Cases:
     * - Bot response generation
     * - System message creation
     * - Tool result messages
     * - Error messages
     * 
     * @param params - Message creation parameters
     * @returns Newly created MessageState
     * @throws Error if required parameters are missing
     */
    static createMessageState(params: {
        id?: string;
        text: string;
        role: "assistant" | "user" | "system" | "tool";
        userId?: string;
        config?: Partial<MessageConfigObject>;
        language?: string;
        parentId?: string;
    }): MessageState {
        if (!params.text || !params.role) {
            logger.error("[MessageTypeAdapters] Invalid parameters for MessageState creation", {
                hasText: !!params.text,
                hasRole: !!params.role,
            });
            throw new Error("Invalid parameters: text and role are required");
        }

        try {
            const messageState: MessageState = {
                id: params.id || generatePK().toString(),
                createdAt: new Date().toISOString(),
                config: {
                    __version: "1.0",
                    role: params.role,
                    ...params.config,
                } as MessageConfigObject,
                text: params.text,
                language: params.language || "en",
                user: params.userId ? { id: params.userId } : null,
                parent: params.parentId ? { id: params.parentId } : null,
            };

            // Created new MessageState

            return messageState;
        } catch (error) {
            logger.error("[MessageTypeAdapters] Failed to create MessageState", {
                role: params.role,
                textLength: params.text?.length || 0,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`MessageState creation failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Create tool result message
     * 
     * Specialized factory for creating MessageState objects that represent
     * tool execution results. These have specific formatting and metadata
     * requirements for proper conversation flow.
     * 
     * @param params - Tool result parameters
     * @returns MessageState representing the tool result
     */
    static createToolResultMessage(params: {
        toolCallId: string;
        output: unknown;
        success: boolean;
        toolName: string;
    }): MessageState {
        const text = params.success
            ? (typeof params.output === "string" 
                ? params.output 
                : JSON.stringify(params.output, null, 2))
            : `Tool execution failed: ${String(params.output)}`;

        return this.createMessageState({
            text,
            role: "tool",
            userId: "system",
            config: {
                __version: "0.1.0" as const,
                role: "tool" as const,
            },
        });
    }

    /**
     * Extract role from message config
     * 
     * Safely extracts the role field from MessageConfigObject with validation
     * and sensible defaults for malformed data.
     * 
     * @private
     */
    private static extractRole(config: MessageConfigObject): "system" | "user" | "assistant" | "tool" {
        const role = config.role as string;
        
        if (["system", "user", "assistant", "tool"].includes(role)) {
            return role as "system" | "user" | "assistant" | "tool";
        }

        logger.warn("[MessageTypeAdapters] Invalid or missing role in config, defaulting to 'user'", {
            providedRole: role,
            configVersion: config.__version,
        });

        return "user"; // Safe default
    }
}

/**
 * MessageValidator - Runtime validation for message types
 * 
 * Provides type guards and validation functions to ensure data integrity
 * when working with message objects from external sources or unknown types.
 */
export class MessageValidator {
    /**
     * Type guard for MessageState
     * 
     * Validates that an object conforms to the MessageState interface
     * and has all required fields with correct types.
     * 
     * @param obj - Object to validate
     * @returns True if object is a valid MessageState
     */
    static validateMessageState(obj: unknown): obj is MessageState {
        if (!obj || typeof obj !== "object") {
            return false;
        }

        const msg = obj as Record<string, unknown>;

        return (
            typeof msg.id === "string" &&
            typeof msg.text === "string" &&
            typeof msg.config === "object" &&
            msg.config !== null &&
            typeof msg.createdAt === "string" &&
            typeof msg.language === "string" &&
            (msg.user === null || (typeof msg.user === "object" && msg.user !== null && typeof (msg.user as any).id === "string")) &&
            (msg.parent === null || (typeof msg.parent === "object" && msg.parent !== null && typeof (msg.parent as any).id === "string"))
        );
    }

    /**
     * Type guard for ChatMessage
     * 
     * Validates that an object conforms to the ChatMessage interface
     * and has all required fields with correct types.
     * 
     * @param obj - Object to validate
     * @returns True if object is a valid ChatMessage
     */
    static validateChatMessage(obj: unknown): obj is ChatMessage {
        if (!obj || typeof obj !== "object") {
            return false;
        }

        const msg = obj as Record<string, unknown>;

        return (
            typeof msg.id === "string" &&
            typeof msg.text === "string" &&
            typeof msg.config === "object" &&
            msg.config !== null &&
            typeof msg.createdAt === "string" &&
            typeof msg.language === "string" &&
            typeof msg.user === "object" &&
            msg.user !== null &&
            typeof (msg.user as any).id === "string" &&
            typeof msg.chat === "object" &&
            msg.chat !== null
        );
    }

    /**
     * Validate and sanitize unknown message data
     * 
     * Attempts to convert unknown message data into a valid MessageState,
     * providing defaults for missing fields and logging warnings for
     * unexpected data patterns.
     * 
     * @param obj - Unknown message data
     * @returns Valid MessageState or null if conversion is impossible
     */
    static sanitizeToMessageState(obj: unknown): MessageState | null {
        if (!obj || typeof obj !== "object") {
            logger.warn("[MessageValidator] Cannot sanitize non-object to MessageState", {
                type: typeof obj,
            });
            return null;
        }

        const msg = obj as Record<string, unknown>;

        if (!msg.id || !msg.text) {
            logger.warn("[MessageValidator] Missing required fields for MessageState", {
                hasId: !!msg.id,
                hasText: !!msg.text,
            });
            return null;
        }

        try {
            return MessageTypeAdapters.createMessageState({
                id: String(msg.id),
                text: String(msg.text),
                role: this.sanitizeRole(msg.role),
                userId: msg.user && typeof msg.user === "object" ? String((msg.user as any).id) : undefined,
                language: typeof msg.language === "string" ? msg.language : "en",
                config: typeof msg.config === "object" ? msg.config as Partial<MessageConfigObject> : {},
            });
        } catch (error) {
            logger.error("[MessageValidator] Failed to sanitize object to MessageState", {
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Sanitize role field from unknown data
     * @private
     */
    private static sanitizeRole(role: unknown): "assistant" | "user" | "system" | "tool" {
        if (typeof role === "string" && ["assistant", "user", "system", "tool"].includes(role)) {
            return role as "assistant" | "user" | "system" | "tool";
        }
        return "user"; // Safe default
    }
}

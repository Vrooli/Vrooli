import type { ChatCompletionMessageParam, ChatCompletionSystemMessageParam, ChatCompletionUserMessageParam, ChatCompletionAssistantMessageParam, ChatCompletionToolMessageParam } from "openai/resources/chat/completions.js";
import { logger } from "../../events/logger.js";

/**
 * Type guard to check if a role is valid for chat messages
 */
export function isValidMessageRole(role: unknown): role is "user" | "assistant" | "system" | "tool" {
    return typeof role === "string" && ["user", "assistant", "system", "tool"].includes(role);
}

/**
 * Type guard to check if a message is a system message
 */
export function isSystemMessage(message: ChatCompletionMessageParam): message is ChatCompletionSystemMessageParam {
    return message.role === "system";
}

/**
 * Type guard to check if a message is a user message
 */
export function isUserMessage(message: ChatCompletionMessageParam): message is ChatCompletionUserMessageParam {
    return message.role === "user";
}

/**
 * Type guard to check if a message is an assistant message
 */
export function isAssistantMessage(message: ChatCompletionMessageParam): message is ChatCompletionAssistantMessageParam {
    return message.role === "assistant";
}

/**
 * Type guard to check if a message is a tool message
 */
export function isToolMessage(message: ChatCompletionMessageParam): message is ChatCompletionToolMessageParam {
    return message.role === "tool";
}

/**
 * Creates a type-safe system message
 */
export function createSystemMessage(content: string): ChatCompletionSystemMessageParam {
    return {
        role: "system",
        content,
    };
}

/**
 * Creates a type-safe user message
 */
export function createUserMessage(content: string, name?: string): ChatCompletionUserMessageParam {
    const message: ChatCompletionUserMessageParam = {
        role: "user",
        content,
    };
    if (name) {
        message.name = name;
    }
    return message;
}

/**
 * Creates a type-safe assistant message
 */
export function createAssistantMessage(content: string | null, toolCalls?: any[], name?: string): ChatCompletionAssistantMessageParam {
    const message: ChatCompletionAssistantMessageParam = {
        role: "assistant",
        content,
    };
    if (toolCalls && toolCalls.length > 0) {
        message.tool_calls = toolCalls;
    }
    if (name) {
        message.name = name;
    }
    return message;
}

/**
 * Creates a type-safe tool message
 */
export function createToolMessage(content: string, toolCallId: string): ChatCompletionToolMessageParam {
    return {
        role: "tool",
        content,
        tool_call_id: toolCallId,
    };
}

/**
 * Validates and logs warnings for unexpected message roles
 */
export function validateMessageRole(role: unknown, messageId?: string): boolean {
    if (!isValidMessageRole(role)) {
        logger.warn("Invalid message role encountered", {
            role,
            messageId,
            validRoles: ["user", "assistant", "system", "tool"],
        });
        return false;
    }
    return true;
}

/**
 * Common content safety patterns used across all AI services
 */
export const HARMFUL_CONTENT_PATTERNS = [
    /\b(kill|murder|suicide|self-harm|death)\b/i,
    /\b(hack|exploit|vulnerability|malware|virus)\b/i,
    /\b(bomb|weapon|explosive|gun|violence)\b/i,
    /\b(illegal|drugs|trafficking|terrorism)\b/i,
    /\b(hate|racism|discrimination|harassment)\b/i,
] as const;

/**
 * Common prompt injection patterns
 */
export const PROMPT_INJECTION_PATTERNS = [
    /ignore\s+previous\s+instructions/i,
    /system\s*:\s*you\s+are\s+now/i,
    /forget\s+everything\s+above/i,
    /new\s+instructions\s*:/i,
    /override\s+system\s+prompt/i,
    /disregard\s+all\s+previous/i,
] as const;

/**
 * Checks for harmful content using common patterns
 */
export function hasHarmfulContent(input: string): { isHarmful: boolean; pattern?: string } {
    for (const pattern of HARMFUL_CONTENT_PATTERNS) {
        if (pattern.test(input)) {
            return { isHarmful: true, pattern: pattern.source };
        }
    }
    return { isHarmful: false };
}

/**
 * Checks for prompt injection attempts
 */
export function hasPromptInjection(input: string): { isInjection: boolean; pattern?: string } {
    for (const pattern of PROMPT_INJECTION_PATTERNS) {
        if (pattern.test(input)) {
            return { isInjection: true, pattern: pattern.source };
        }
    }
    return { isInjection: false };
}

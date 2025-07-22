import type { ChatCompletionMessageParam } from "openai/resources/chat/completions.js";
import type { MessageState } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import {
    createSystemMessage,
    createUserMessage,
    createAssistantMessage,
    createToolMessage,
    validateMessageRole,
} from "./messageValidation.js";

/**
 * Shared implementation of generateContext for all AI services.
 * Converts MessageState[] to ChatCompletionMessageParam[] with proper type safety.
 * 
 * @param messages - Array of MessageState objects from the conversation
 * @param systemMessage - Optional system message to prepend
 * @param serviceName - Name of the service (for logging)
 * @returns Array of properly typed chat completion messages
 */
export function generateContextFromMessages(
    messages: MessageState[],
    systemMessage?: string,
    serviceName = "AIService",
): ChatCompletionMessageParam[] {
    const context: ChatCompletionMessageParam[] = [];

    // Add system message if provided
    if (systemMessage) {
        context.push(createSystemMessage(systemMessage));
    }

    // Convert each message to the appropriate format
    for (const message of messages) {
        const role = message.config?.role;

        // Skip messages without a role
        if (!role) {
            logger.warn(`[${serviceName}] Message without role encountered`, {
                messageId: message.id,
            });
            continue;
        }

        // Validate the role
        if (!validateMessageRole(role, message.id)) {
            continue;
        }

        // Convert based on role
        switch (role) {
            case "user":
                context.push(createUserMessage(message.text || ""));
                break;

            case "assistant":
                // Check if this assistant message has tool calls
                const toolCalls = message.config?.toolCalls;
                if (toolCalls && toolCalls.length > 0) {
                    // Convert tool calls to OpenAI format
                    const formattedToolCalls = toolCalls.map(tc => ({
                        id: tc.id,
                        type: "function" as const,
                        function: {
                            name: tc.function.name,
                            arguments: typeof tc.function.arguments === "string" 
                                ? tc.function.arguments 
                                : JSON.stringify(tc.function.arguments),
                        },
                    }));
                    context.push(createAssistantMessage(message.text || null, formattedToolCalls));
                    
                    // Add tool response messages if results are present
                    for (const toolCall of toolCalls) {
                        if (toolCall.result) {
                            let resultContent: string;
                            if (toolCall.result.success) {
                                resultContent = JSON.stringify(toolCall.result.output);
                            } else {
                                resultContent = JSON.stringify({ 
                                    error: {
                                        code: (toolCall.result as any).error?.code || "unknown",
                                        message: (toolCall.result as any).error?.message || "Unknown error",
                                    },
                                });
                            }
                            context.push(createToolMessage(resultContent, toolCall.id));
                        }
                    }
                } else {
                    context.push(createAssistantMessage(message.text || ""));
                }
                break;

            case "system":
                // Additional system messages in the conversation
                context.push(createSystemMessage(message.text || ""));
                break;

            case "tool":
                // Standalone tool messages are not typically used in the OpenAI format
                // Tool responses are usually handled as part of assistant messages with tool_calls
                logger.warn(`[${serviceName}] Standalone tool message encountered - skipping`, {
                    messageId: message.id,
                    text: message.text?.substring(0, 100), // Log first 100 chars for debugging
                });
                continue;

            default:
                // This should not happen due to validation, but TypeScript needs it
                logger.warn(`[${serviceName}] Unexpected role after validation`, {
                    role,
                    messageId: message.id,
                });
        }
    }

    return context;
}

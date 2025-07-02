/**
 * Context Builder for Response Service
 * 
 * This adapter bridges between the unified ResponseContext and the existing
 * conversation ContextBuilder. It provides a clean interface while reusing
 * the existing context building logic.
 */

import type { ResponseContext } from "@vrooli/shared";
import type { ChatMessage } from "@vrooli/shared";
import type { ContextBuilder as ConversationContextBuilder } from "../conversation/contextBuilder.js";

/**
 * Adapter that converts unified ResponseContext to existing ContextBuilder format
 * This allows us to reuse existing context building logic while providing a clean interface
 */
export class ContextBuilder {
    constructor(
        private readonly conversationContextBuilder: ConversationContextBuilder,
    ) { }

    /**
     * Build conversation context messages for LLM from ResponseContext
     * This is a zero-transformation adapter that delegates to existing logic
     */
    async buildContext(context: ResponseContext): Promise<ChatMessage[]> {
        // Convert ResponseContext to ContextBuilder.build() parameters
        const botParticipant = {
            id: context.botId,
            config: context.botConfig,
            name: context.botConfig.name || "Assistant",
        };

        // Convert tools to OpenAI format (simplified for now)
        const openAITools: unknown[] = context.availableTools.map(tool => ({
            type: "function",
            name: tool.name,
            description: tool.description,
            parameters: tool.inputSchema,
        }));

        // Build system message
        const systemMessage = await this.buildSystemMessage(context);

        // Delegate to existing context builder
        const buildResult = await this.conversationContextBuilder.build(
            context.swarmId, // chatId
            botParticipant, // bot
            context.botConfig.model || "gpt-4", // aiModel
            undefined, // startMessageId (use most recent)
            {
                tools: openAITools,
                systemMessage,
            },
        );

        // Convert MessageState[] to ChatMessage[]
        return buildResult.messages.map(messageState => ({
            id: messageState.id,
            createdAt: messageState.createdAt,
            config: messageState.config,
            user: messageState.user || { id: "unknown" },
            text: messageState.text || "",
            language: messageState.language || "en",
        }));
    }

    /**
     * Build system message for the bot
     * This delegates to any existing system message building logic
     */
    async buildSystemMessage(
        context: ResponseContext,
        goal?: string,
        _teamConfig?: unknown,
    ): Promise<string> {
        // For now, create a basic system message
        // This can be enhanced to delegate to existing system message building logic
        const botName = context.botConfig.name || "Assistant";
        const botDescription = context.botConfig.description || "A helpful AI assistant";
        const strategy = context.strategy || "conversation";

        let systemMessage = `You are ${botName}. ${botDescription}\n\n`;

        if (goal) {
            systemMessage += `Your current goal: ${goal}\n\n`;
        }

        systemMessage += `Execution strategy: ${strategy}\n\n`;

        if (context.availableTools && context.availableTools.length > 0) {
            systemMessage += "You have access to the following tools:\n";
            context.availableTools.forEach(tool => {
                systemMessage += `- ${tool.name}: ${tool.description}\n`;
            });
            systemMessage += "\n";
        }

        systemMessage += "Please provide helpful, accurate, and relevant responses to the user's messages.";

        return systemMessage;
    }

    /**
     * Validate that context has required fields for building
     */
    validateContext(context: ResponseContext): { valid: boolean; errors: string[] } {
        const errors: string[] = [];

        if (!context.swarmId) {
            errors.push("Missing swarmId");
        }

        if (!context.botId) {
            errors.push("Missing botId");
        }

        if (!context.botConfig) {
            errors.push("Missing botConfig");
        }

        if (!context.userData) {
            errors.push("Missing userData");
        }

        if (!Array.isArray(context.conversationHistory)) {
            errors.push("Invalid conversationHistory");
        }

        if (!Array.isArray(context.availableTools)) {
            errors.push("Invalid availableTools");
        }

        return {
            valid: errors.length === 0,
            errors,
        };
    }

    /**
     * Get effective constraints from context and defaults
     */
    getEffectiveConstraints(context: ResponseContext): Required<NonNullable<ResponseContext["constraints"]>> {
        const defaults = {
            maxTokens: 4000,
            temperature: 0.7,
            timeoutMs: 60000, // 60 seconds
            maxCredits: "1000", // Default credit limit
        };

        return {
            maxTokens: context.constraints?.maxTokens ?? defaults.maxTokens,
            temperature: context.constraints?.temperature ?? defaults.temperature,
            timeoutMs: context.constraints?.timeoutMs ?? defaults.timeoutMs,
            maxCredits: context.constraints?.maxCredits ?? defaults.maxCredits,
        };
    }
}

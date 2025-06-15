/**
 * ConversationBridge - Bridge between SwarmStateMachine and conversation/responseEngine
 * 
 * This provides a simple interface for the SwarmStateMachine to generate AI responses
 * and execute tools through the existing conversation infrastructure.
 */

import { type Logger } from "winston";
import { type SessionUser, type MessageState } from "@vrooli/shared";
import { type ConversationState } from "../../../conversation/types.js";

export interface ConversationBridgeConfig {
    conversationId: string;
    sessionUser: SessionUser;
}

export class ConversationBridge {
    constructor(
        private readonly logger: Logger,
    ) {}

    /**
     * Generate an AI response for a given prompt
     */
    async generateResponse(
        config: ConversationBridgeConfig,
        prompt: string,
        systemMessage?: string,
    ): Promise<MessageState[]> {
        this.logger.info("[ConversationBridge] Generating response", {
            conversationId: config.conversationId,
            promptLength: prompt.length,
        });

        try {
            // Lazy load to avoid initialization issues in tests
            const { completionService } = await import("../../../conversation/responseEngine.js");
            
            // Get the conversation state
            const conversationState = await completionService.getConversationState(config.conversationId);
            if (!conversationState) {
                throw new Error(`Conversation not found: ${config.conversationId}`);
            }

            // Use the completion service's reasoning engine to generate response
            const reasoningEngine = completionService.getReasoningEngine();
            
            // For now, we'll create a simple message and let the reasoning engine handle it
            // In a full implementation, this would properly integrate with the message flow
            this.logger.warn("[ConversationBridge] Response generation not fully implemented yet");
            
            return [];
        } catch (error) {
            this.logger.error("[ConversationBridge] Failed to generate response", {
                error: error instanceof Error ? error.message : String(error),
                conversationId: config.conversationId,
            });
            throw error;
        }
    }

    /**
     * Execute a tool call
     */
    async executeTool(
        config: ConversationBridgeConfig,
        toolName: string,
        args: Record<string, unknown>,
    ): Promise<unknown> {
        this.logger.info("[ConversationBridge] Executing tool", {
            conversationId: config.conversationId,
            toolName,
        });

        try {
            // Lazy load to avoid initialization issues in tests
            const { completionService } = await import("../../../conversation/responseEngine.js");
            
            // Get the tool from registry
            const toolRegistry = completionService.getToolRegistry();
            const tool = await toolRegistry.getToolByName(toolName);
            
            if (!tool) {
                throw new Error(`Tool not found: ${toolName}`);
            }

            // Execute through the reasoning engine's tool runner
            const reasoningEngine = completionService.getReasoningEngine();
            const result = await reasoningEngine.toolRunner.execute(
                tool,
                args,
                config.sessionUser,
            );

            return result;
        } catch (error) {
            this.logger.error("[ConversationBridge] Failed to execute tool", {
                error: error instanceof Error ? error.message : String(error),
                conversationId: config.conversationId,
                toolName,
            });
            throw error;
        }
    }

    /**
     * Handle tool approval response
     */
    async handleToolApproval(
        config: ConversationBridgeConfig,
        toolCallId: string,
        approved: boolean,
        reason?: string,
    ): Promise<void> {
        this.logger.info("[ConversationBridge] Handling tool approval", {
            conversationId: config.conversationId,
            toolCallId,
            approved,
        });

        // This would integrate with the pending tool call system
        // For now, just log it
        this.logger.warn("[ConversationBridge] Tool approval handling not fully implemented yet");
    }
}

// Export a factory function for easy creation
export function createConversationBridge(logger: Logger): ConversationBridge {
    return new ConversationBridge(logger);
}
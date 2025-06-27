/**
 * ConversationBridge - Bridge between SwarmStateMachine and conversation/responseEngine
 * 
 * This provides a simple interface for the SwarmStateMachine to generate AI responses
 * and execute tools through the existing conversation infrastructure.
 */

import { type Logger } from "winston";
import { type SessionUser, type MessageState } from "@vrooli/shared";
import { type BotParticipant } from "../../../conversation/types.js";

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
            const toolRegistry = completionService.getToolRegistry();
            
            // Get available tools for this conversation
            const availableTools = await toolRegistry.getToolsForConversation(config.conversationId);
            
            // Create a bot participant for the current conversation
            // In a real implementation, this would come from the conversation state
            const bot: BotParticipant = {
                id: conversationState.config.swarmLeader || "tier1-coordinator",
                name: "Tier1 Coordinator",
                config: conversationState.bots?.[0]?.config || {},
                meta: {
                    role: "coordinator",
                    systemPrompt: systemMessage,
                },
            };
            
            // Use the existing system message generation if no custom one provided
            let finalSystemMessage = systemMessage;
            if (!finalSystemMessage) {
                const goal = conversationState.config.swarmTask || "Process current request";
                finalSystemMessage = await completionService.generateSystemMessageForBot(
                    goal,
                    bot,
                    conversationState.config,
                    conversationState.teamConfig,
                );
            }
            
            // Default limits for tool calls and credits
            const DEFAULT_TOOL_CALLS_PER_RESPONSE = 10;
            const DEFAULT_CREDITS_PER_RESPONSE = 1000;
            
            // Run the reasoning loop with the prompt as a standalone text message
            const result = await reasoningEngine.runLoop(
                { text: prompt }, // Use text-based input for the prompt
                finalSystemMessage,
                availableTools,
                bot,
                config.sessionUser.id, // Use user ID as credit account
                conversationState.config,
                conversationState.config.limits?.toolCallsPerResponse || DEFAULT_TOOL_CALLS_PER_RESPONSE,
                BigInt(conversationState.config.limits?.creditsPerResponse || DEFAULT_CREDITS_PER_RESPONSE),
                config.sessionUser,
                config.conversationId,
                conversationState.config.model || "gpt-4", // Default model
            );
            
            this.logger.info("[ConversationBridge] Successfully generated response", {
                conversationId: config.conversationId,
                messageLength: result.finalMessage.text?.length || 0,
                toolCallsCount: result.responseStats.toolCallsExecuted,
                creditsUsed: result.responseStats.creditsUsed.toString(),
            });
            
            // Return the final message in an array as expected
            return [result.finalMessage];
            
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
            
            // Get the reasoning engine's tool runner
            const reasoningEngine = completionService.getReasoningEngine();
            const toolRunner = reasoningEngine.toolRunner;
            
            // Execute the tool using the tool runner's run method
            const result = await toolRunner.run(toolName, args, {
                conversationId: config.conversationId,
                callerBotId: "tier1-coordinator",
                sessionUser: config.sessionUser,
            });

            if (result.ok) {
                this.logger.info("[ConversationBridge] Tool executed successfully", {
                    conversationId: config.conversationId,
                    toolName,
                    creditsUsed: result.data.creditsUsed,
                });
                return result.data.output;
            } else {
                this.logger.error("[ConversationBridge] Tool execution failed", {
                    conversationId: config.conversationId,
                    toolName,
                    error: result.error.message,
                });
                throw new Error(`Tool execution failed: ${result.error.message}`);
            }
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

        try {
            // Lazy load to avoid initialization issues in tests
            const { completionService } = await import("../../../conversation/responseEngine.js");
            
            // Get the conversation state to access pending tool calls
            const conversationState = await completionService.getConversationState(config.conversationId);
            if (!conversationState) {
                throw new Error(`Conversation not found: ${config.conversationId}`);
            }

            // Find the pending tool call
            const pendingToolCalls = conversationState.config.pendingToolCalls || [];
            const toolCallIndex = pendingToolCalls.findIndex(call => call.id === toolCallId);
            
            if (toolCallIndex === -1) {
                this.logger.warn("[ConversationBridge] Tool call not found in pending list", {
                    conversationId: config.conversationId,
                    toolCallId,
                });
                return;
            }

            const toolCall = pendingToolCalls[toolCallIndex];
            
            // Update the tool call status based on approval
            if (approved) {
                toolCall.status = "approved";
                this.logger.info("[ConversationBridge] Tool call approved", {
                    conversationId: config.conversationId,
                    toolCallId,
                    toolName: toolCall.toolName,
                });
            } else {
                toolCall.status = "rejected";
                toolCall.rejectionReason = reason;
                this.logger.info("[ConversationBridge] Tool call rejected", {
                    conversationId: config.conversationId,
                    toolCallId,
                    toolName: toolCall.toolName,
                    reason,
                });
            }

            // Update the conversation state with the modified pending tool calls
            const updatedConfig = {
                ...conversationState.config,
                pendingToolCalls,
            };
            
            completionService.updateConversationConfig(config.conversationId, updatedConfig);
            
        } catch (error) {
            this.logger.error("[ConversationBridge] Failed to handle tool approval", {
                error: error instanceof Error ? error.message : String(error),
                conversationId: config.conversationId,
                toolCallId,
            });
            throw error;
        }
    }
}

// Export a factory function for easy creation
export function createConversationBridge(logger: Logger): ConversationBridge {
    return new ConversationBridge(logger);
}

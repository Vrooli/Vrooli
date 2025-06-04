import { ChatConfig, MINUTES_1_MS, MessageConfig, ResourceSubType, generatePK, getTranslation, type ChatConfigObject } from "@local/shared";
import { type MessageState } from "../../conversation/types.js";
import { type SubroutineExecutionResult } from "../unifiedExecutionEngine.js";
import { type ExecutionStrategy, type ExecutionStrategyDependencies } from "./executionStrategy.js";

// Constants
const MAX_CONVERSATION_TURNS = 10;
const TURN_TIMEOUT_MS = MINUTES_1_MS;
const CONVERSATION_CONTEXT_WINDOW = 5; // Number of previous messages to include

/**
 * Execution strategy for conversational, multi-turn subroutines.
 * 
 * This strategy is used for:
 * - Customer service interactions
 * - Iterative content creation
 * - Collaborative problem-solving
 * - Tutorial and guidance routines
 * - Complex negotiations or discussions
 * 
 * Key features:
 * - Multi-turn conversation support
 * - Context preservation across turns
 * - User interaction handling
 * - Flexible termination conditions
 * - Rich message formatting
 */
export class ConversationalStrategy implements ExecutionStrategy {
    readonly name = "ConversationalStrategy";

    /**
     * Checks if this strategy can handle the given routine type.
     */
    canHandle(routineSubType: string, config?: Record<string, unknown>): boolean {
        // Handle web routines (often used for searches and discussions)
        if (routineSubType === ResourceSubType.RoutineWeb) {
            return true;
        }

        // Check if config explicitly requests conversational strategy
        if (config?.executionStrategy === "conversational") {
            return true;
        }

        // Check for conversational keywords
        const conversationalKeywords = [
            "chat", "converse", "discuss", "talk", "dialogue",
            "interview", "consult", "advise", "guide", "tutorial",
            "negotiate", "collaborate", "brainstorm", "iterate",
            "customer", "service", "support", "help",
        ];

        const routineName = config?.name as string || "";
        const routineDescription = config?.description as string || "";
        const combined = `${routineName} ${routineDescription}`.toLowerCase();

        return conversationalKeywords.some(keyword => combined.includes(keyword));
    }

    /**
     * Executes a conversational subroutine.
     */
    async execute(deps: ExecutionStrategyDependencies): Promise<SubroutineExecutionResult> {
        const { context, reasoningEngine, messageStore, logger } = deps;
        const startTime = Date.now();
        const messages: MessageState[] = [];
        let totalCreditsUsed = BigInt(0);
        let totalToolCalls = 0;

        logger.info(`Executing conversational strategy for subroutine ${context.subroutineInstanceId}`);

        try {
            // Build initial system message
            const systemMessage = await this.buildSystemMessage(deps);

            // Create a temporary conversation ID for this execution
            const tempConversationId = `conv_${context.subroutineInstanceId}`;

            // Initialize conversation with user message
            const initialMessage = this.buildInitialMessage(context);
            if (initialMessage) {
                messages.push(initialMessage);
                await messageStore.addMessage(tempConversationId, initialMessage);
            }

            // Conversation loop
            let turnCount = 0;
            let conversationComplete = false;

            while (!conversationComplete && turnCount < MAX_CONVERSATION_TURNS) {
                turnCount++;

                // Check for timeout
                if (Date.now() - startTime > context.limits.maxTimeMs) {
                    throw new Error("Conversation timed out");
                }

                // Check credit limits
                if (totalCreditsUsed >= context.limits.maxCredits) {
                    throw new Error("Credit limit exceeded during conversation");
                }

                // Execute one turn of conversation
                const turnResult = await this.executeConversationTurn(
                    deps,
                    tempConversationId,
                    systemMessage,
                    messages,
                    turnCount,
                );

                // Update totals
                messages.push(turnResult.message);
                totalCreditsUsed += turnResult.creditsUsed;
                totalToolCalls += turnResult.toolCalls;

                // Check if conversation is complete
                conversationComplete = this.checkConversationComplete(
                    turnResult.message,
                    context,
                    turnCount,
                );

                // Store message for context
                await messageStore.addMessage(tempConversationId, turnResult.message);
            }

            // Extract outputs from conversation
            const outputs = this.extractOutputsFromConversation(messages, context);

            // Update IO mapping with outputs
            for (const [key, value] of Object.entries(outputs)) {
                if (context.ioMapping.outputs[key]) {
                    context.ioMapping.outputs[key].value = value;
                }
            }

            logger.info(`Conversational strategy completed successfully for ${context.subroutineInstanceId}`, {
                turns: turnCount,
                creditsUsed: totalCreditsUsed.toString(),
                toolCalls: totalToolCalls,
            });

            return {
                success: true,
                ioMapping: context.ioMapping,
                creditsUsed: totalCreditsUsed,
                timeElapsed: Date.now() - startTime,
                toolCallsCount: totalToolCalls,
                messages,
                metadata: {
                    strategy: this.name,
                    metrics: {
                        conversationTurns: turnCount,
                        averageResponseTime: (Date.now() - startTime) / turnCount,
                    },
                },
            };

        } catch (error) {
            logger.error(`Conversational strategy failed for ${context.subroutineInstanceId}`, error);

            return {
                success: false,
                ioMapping: context.ioMapping,
                creditsUsed: totalCreditsUsed,
                timeElapsed: Date.now() - startTime,
                toolCallsCount: totalToolCalls,
                messages,
                error: {
                    code: "CONVERSATION_ERROR",
                    message: error instanceof Error ? error.message : "Unknown conversation error",
                    details: error,
                },
                metadata: {
                    strategy: this.name,
                },
            };
        }
    }

    /**
     * Builds the system message for the conversation.
     */
    private async buildSystemMessage(deps: ExecutionStrategyDependencies): Promise<string> {
        const { context } = deps;
        const { name, description, instructions } = getTranslation(
            context.routine,
            context.userData.languages,
            true,
        );

        let systemMessage = `You are engaging in a conversational task: "${name || "Unnamed Conversation"}"\n\n`;

        if (description) {
            systemMessage += `Description: ${description}\n\n`;
        }

        if (instructions) {
            systemMessage += `Instructions:\n${instructions}\n\n`;
        }

        // Add conversation guidelines
        systemMessage += "Conversation Guidelines:\n";
        systemMessage += "- Engage naturally and helpfully with the user\n";
        systemMessage += "- Stay focused on the task objectives\n";
        systemMessage += "- Ask clarifying questions when needed\n";
        systemMessage += "- Signal when the conversation objectives are met\n";
        systemMessage += "- Be concise but thorough in responses\n\n";

        // Add expected outputs
        if (Object.keys(context.ioMapping.outputs).length > 0) {
            systemMessage += "Expected conversation outcomes:\n";
            for (const [key, output] of Object.entries(context.ioMapping.outputs)) {
                systemMessage += `- ${output.name || key}`;
                if (output.description) {
                    systemMessage += `: ${output.description}`;
                }
                systemMessage += "\n";
            }
        }

        return systemMessage;
    }

    /**
     * Builds the initial user message from inputs.
     */
    private buildInitialMessage(context: ExecutionStrategyDependencies["context"]): MessageState | null {
        const inputs = context.ioMapping.inputs;

        // Look for a message or prompt input
        const messageText = inputs.message?.value || inputs.prompt?.value || inputs.question?.value;

        if (!messageText) {
            return null;
        }

        return {
            id: generatePK().toString(),
            createdAt: new Date(),
            config: MessageConfig.default().export(),
            text: String(messageText),
            language: context.userData.languages?.[0] || "en",
            parent: null,
            user: { id: context.userData.id },
        } as MessageState;
    }

    /**
     * Executes one turn of the conversation.
     */
    private async executeConversationTurn(
        deps: ExecutionStrategyDependencies,
        conversationId: string,
        systemMessage: string,
        previousMessages: MessageState[],
        turnNumber: number,
    ): Promise<{
        message: MessageState;
        creditsUsed: bigint;
        toolCalls: number;
    }> {
        const { context, reasoningEngine } = deps;

        // Get the last message ID for context
        const lastMessageId = previousMessages[previousMessages.length - 1]?.id;
        if (!lastMessageId) {
            throw new Error("No previous message for conversation turn");
        }

        // Build turn-specific config
        const turnConfig = this.buildTurnConfig(context, turnNumber);

        // Execute reasoning for this turn
        const { finalMessage, responseStats } = await reasoningEngine.runLoop(
            { id: lastMessageId },
            systemMessage,
            deps.availableTools,
            deps.botParticipant,
            context.userData.id,
            turnConfig,
            this.calculateTurnToolAllocation(context, turnNumber),
            this.calculateTurnCreditAllocation(context, turnNumber),
            context.userData,
            conversationId,
            deps.botParticipant.config.model,
            deps.abortSignal,
        );

        return {
            message: finalMessage,
            creditsUsed: responseStats.creditsUsed,
            toolCalls: responseStats.toolCalls,
        };
    }

    /**
     * Builds configuration for a conversation turn.
     */
    private buildTurnConfig(
        context: ExecutionStrategyDependencies["context"],
        turnNumber: number,
    ): ChatConfigObject {
        const baseConfig = context.parentSwarmContext?.chatConfig || ChatConfig.default().export();

        return {
            ...baseConfig,
            // Adjust limits for each turn
            limits: {
                ...baseConfig.limits,
                maxToolCallsPerBotResponse: Math.max(1, Math.floor(context.limits.maxToolCalls / MAX_CONVERSATION_TURNS)),
                maxCreditsPerBotResponse: (context.limits.maxCredits / BigInt(MAX_CONVERSATION_TURNS)).toString(),
                maxDurationMs: TURN_TIMEOUT_MS,
            },
        };
    }

    /**
     * Calculates tool allocation for a conversation turn.
     */
    private calculateTurnToolAllocation(
        context: ExecutionStrategyDependencies["context"],
        _turnNumber: number,
    ): number {
        // Distribute tool calls across expected turns
        return Math.max(1, Math.floor(context.limits.maxToolCalls / MAX_CONVERSATION_TURNS));
    }

    /**
     * Calculates credit allocation for a conversation turn.
     */
    private calculateTurnCreditAllocation(
        context: ExecutionStrategyDependencies["context"],
        _turnNumber: number,
    ): bigint {
        // Distribute credits across expected turns
        return context.limits.maxCredits / BigInt(MAX_CONVERSATION_TURNS);
    }

    /**
     * Checks if the conversation is complete.
     */
    private checkConversationComplete(
        lastMessage: MessageState,
        context: ExecutionStrategyDependencies["context"],
        turnCount: number,
    ): boolean {
        // Check if we've reached the turn limit
        if (turnCount >= MAX_CONVERSATION_TURNS) {
            return true;
        }

        // Check for completion signals in the message
        const completionPhrases = [
            "conversation complete",
            "task completed",
            "objectives met",
            "finished",
            "all done",
            "nothing else",
            "that's all",
        ];

        const messageText = lastMessage.text.toLowerCase();
        const hasCompletionPhrase = completionPhrases.some(phrase => messageText.includes(phrase));

        // Check if all required outputs have been mentioned
        const outputKeys = Object.keys(context.ioMapping.outputs);
        const allOutputsMentioned = outputKeys.every(key =>
            messageText.includes(key.toLowerCase()) ||
            messageText.includes((context.ioMapping.outputs[key].name || "").toLowerCase()),
        );

        return hasCompletionPhrase || allOutputsMentioned;
    }

    /**
     * Extracts outputs from the conversation history.
     */
    private extractOutputsFromConversation(
        messages: MessageState[],
        context: ExecutionStrategyDependencies["context"],
    ): Record<string, unknown> {
        const outputs: Record<string, unknown> = {};
        const outputKeys = Object.keys(context.ioMapping.outputs);

        // For single output, use the entire conversation
        if (outputKeys.length === 1) {
            outputs[outputKeys[0]] = messages
                .filter(m => m.config?.role === "assistant")
                .map(m => m.text)
                .join("\n\n");
            return outputs;
        }

        // For multiple outputs, try to extract from the last few assistant messages
        const assistantMessages = messages
            .filter(m => m.config?.role === "assistant")
            .slice(-3); // Last 3 assistant messages

        // Concatenate for analysis
        const combinedText = assistantMessages.map(m => m.text).join("\n\n");

        // Try structured extraction
        for (const key of outputKeys) {
            const outputName = context.ioMapping.outputs[key].name || key;

            // Look for labeled sections
            const sectionRegex = new RegExp(`${outputName}:?\\s*([^\\n]+(?:\\n(?!\\w+:)[^\\n]+)*)`, "i");
            const match = combinedText.match(sectionRegex);

            if (match) {
                outputs[key] = match[1].trim();
            }
        }

        // If no structured outputs found, use the full conversation
        if (Object.keys(outputs).length === 0 && outputKeys.length > 0) {
            outputs[outputKeys[0]] = combinedText;
        }

        return outputs;
    }
} 

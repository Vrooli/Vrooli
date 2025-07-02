/**
 * ConversationalStrategy - Minimal Clean Architecture Implementation
 * 
 * This strategy demonstrates true clean architecture delegation:
 * Strategy coordinates → ConversationEngine orchestrates → ResponseService generates
 * 
 * The strategy is now minimal (~100 lines) because it properly delegates
 * all conversation logic to ConversationEngine instead of duplicating it.
 */

import {
    type ChatMessage,
    type ConversationContext,
    type ConversationParams,
    type ConversationResult,
    type ConversationTrigger,
    type IExecutionStrategy,
    MINUTES_10_MS,
    type ResourceUsage,
    type SessionUser,
    type ExecutionContext as StrategyExecutionContext,
    type StrategyExecutionResult,
    generatePK,
    toSwarmId,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";

/**
 * ConversationalStrategy - Minimal Implementation
 * 
 * Focuses ONLY on:
 * 1. Strategy selection logic (canHandle)
 * 2. Resource estimation 
 * 3. Minimal context conversion
 * 4. Delegation to ConversationEngine
 * 
 * Everything else is handled by ConversationEngine/ResponseService.
 */
export class ConversationalStrategy implements IExecutionStrategy {
    readonly type = "conversational";
    readonly name = "ConversationalStrategy";
    readonly version = "3.0.0-minimal";

    /**
     * 🎯 Main Strategy Execution - Pure Delegation
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        const startTime = Date.now();

        try {
            // Minimal validation
            if (!context.stepId || !context.stepType) {
                throw new Error("Missing required stepId or stepType");
            }

            // Minimal context conversion - let ConversationEngine handle the details
            const conversationParams = this.buildMinimalConversationParams(context);

            // Delegate to ConversationEngine (this does all the work)
            const result = await this.conversationEngine.orchestrateConversation(conversationParams);

            // Simple result conversion
            return this.convertResult(result, context, startTime);

        } catch (error) {
            const executionTime = Date.now() - startTime;
            logger.error("[ConversationalStrategy] Execution failed", {
                stepId: context.stepId,
                error: error instanceof Error ? error.message : String(error),
                executionTime,
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : String(error),
                metadata: {
                    strategy: this.name,
                    executionTime,
                    stepId: context.stepId,
                },
            };
        }
    }

    /**
     * 🔄 Minimal Context Conversion
     * 
     * Build basic ConversationParams and let ConversationEngine handle
     * bot selection, conversation building, tool execution, etc.
     */
    private buildMinimalConversationParams(context: StrategyExecutionContext): ConversationParams {
        // Create minimal conversation context - ConversationEngine will enrich it
        const conversationContext: ConversationContext = {
            swarmId: toSwarmId(context.metadata?.conversationId || context.stepId),
            userData: {
                id: context.metadata?.userId || "user",
                name: context.config?.userName as string || "User",
            } as SessionUser,
            timestamp: new Date(),
            participants: [], // Let ConversationEngine handle bot selection
            conversationHistory: this.buildBasicHistory(context),
            availableTools: (context.resources?.tools as any[]) || [],
            sharedState: context.metadata || {},
        };

        // Create basic trigger - ConversationEngine will handle the details
        const trigger: ConversationTrigger = {
            type: "user_message",
            message: this.buildBasicMessage(context),
            expectedResponseCount: 1,
        };

        return {
            context: conversationContext,
            trigger,
            strategy: "conversation", // Use conversation strategy in ConversationEngine
            constraints: {
                maxTurns: 5,
                timeoutMs: context.constraints?.maxTime || MINUTES_10_MS,
                resourceLimits: {
                    maxCredits: (context.resources?.credits || 10000).toString(),
                },
            },
        };
    }

    /**
     * 📨 Build Basic Message from Context
     */
    private buildBasicMessage(context: StrategyExecutionContext): ChatMessage {
        // Look for user input
        const userInput = context.inputs.message || context.inputs.prompt || context.inputs.question;

        let text: string;
        if (userInput) {
            text = String(userInput);
        } else {
            // Create from task description and inputs
            const taskName = context.config?.name || "Conversational Task";
            const inputText = Object.entries(context.inputs)
                .map(([key, value]) => `${key}: ${value}`)
                .join(", ");
            text = `Help with ${taskName}. Inputs: ${inputText}`;
        }

        return {
            id: generatePK().toString(),
            createdAt: new Date(),
            config: {
                role: "user",
                text,
            },
            user: { id: context.metadata?.userId || "user" },
        };
    }

    /**
     * 💬 Build Basic History (Minimal)
     */
    private buildBasicHistory(context: StrategyExecutionContext): ChatMessage[] {
        // Just include basic context - ConversationEngine will handle complex history
        const messages: ChatMessage[] = [];

        if (context.config?.description) {
            messages.push({
                id: generatePK().toString(),
                createdAt: new Date(),
                config: {
                    role: "system",
                    text: `Task: ${context.config.description}`,
                },
                user: { id: "system" },
            });
        }

        return messages;
    }

    /**
     * 🔄 Simple Result Conversion
     */
    private convertResult(
        result: ConversationResult,
        context: StrategyExecutionContext,
        startTime: number,
    ): StrategyExecutionResult {
        if (!result.success) {
            return {
                success: false,
                error: result.error?.message || "Conversation failed",
                metadata: {
                    strategy: this.name,
                    executionTime: Date.now() - startTime,
                },
            };
        }

        // Simple output extraction - just use the last assistant message
        const assistantMessages = result.messages.filter(msg => msg.config?.role === "assistant");
        const lastMessage = assistantMessages[assistantMessages.length - 1];
        const outputs: Record<string, unknown> = {
            result: lastMessage?.config?.text || lastMessage?.text || "No response generated",
        };

        return {
            success: true,
            outputs,
            resourceUsage: {
                credits: parseInt(result.resourcesUsed.creditsUsed || "0", 10),
                tokens: result.resourcesUsed.toolCalls || 0,
                time: result.resourcesUsed.durationMs || 0,
                memory: result.resourcesUsed.memoryUsedMB || 0,
            },
            confidence: this.calculateSimpleConfidence(result),
            metadata: {
                strategy: this.name,
                executionTime: Date.now() - startTime,
                messageCount: result.messages.length,
                conversationComplete: result.conversationComplete,
            },
        };
    }

    /**
     * 🔢 Simple Confidence Calculation
     */
    private calculateSimpleConfidence(result: ConversationResult): number {
        // Use ConversationEngine's confidence if available, otherwise default
        if (result.metadata?.turnMetrics?.averageConfidence) {
            return result.metadata.turnMetrics.averageConfidence;
        }
        return result.conversationComplete ? 0.8 : 0.6;
    }

    // ==================== STRATEGY-SPECIFIC LOGIC (PRESERVED) ====================

    /**
     * ✅ Strategy Selection Logic - This is the core strategy responsibility
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        // Explicit strategy request
        if (config?.strategy === "conversational" || config?.executionStrategy === "conversational") {
            return true;
        }

        // Web routines (often conversational)
        if (stepType === "RoutineWeb" || stepType === "web") {
            return true;
        }

        // Conversational keywords
        const conversationalKeywords = [
            "chat", "converse", "discuss", "talk", "dialogue",
            "interview", "consult", "advise", "guide", "tutorial",
            "customer", "service", "support", "help",
            "creative", "explore", "natural", "language",
        ];

        const combined = `${stepType} ${config?.name || ""} ${config?.description || ""}`.toLowerCase();
        return conversationalKeywords.some(keyword => combined.includes(keyword));
    }

    /**
     * 📊 Resource Estimation - Strategy-specific performance characteristics
     */
    estimateResources(context: StrategyExecutionContext): ResourceUsage {
        const inputComplexity = Object.keys(context.inputs).length;
        const historyLength = context.history?.recentSteps?.length || 0;

        // Base estimates for conversational tasks
        const baseTokens = 1000; // Slightly higher than deterministic due to conversation
        const complexity = Math.min(1 + (inputComplexity * 0.1) + (historyLength * 0.05), 2.0);

        return {
            tokens: Math.ceil(baseTokens * complexity),
            apiCalls: 2, // ConversationEngine adds bot selection call
            computeTime: Math.ceil(8000 * complexity), // 8-16 seconds
            cost: baseTokens * complexity * 0.00002, // GPT-4 pricing
        };
    }
}

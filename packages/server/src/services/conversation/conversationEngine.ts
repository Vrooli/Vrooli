/**
 * ConversationEngine - Intelligent Conversation Orchestration
 * 
 * This service provides intelligent multi-bot conversation orchestration,
 * replacing the missing functionality from SwarmStateMachine while maintaining
 * clean separation from response generation.
 * 
 * Key Responsibilities:
 * - Intelligent bot selection (replaces missing findLeaderBot)
 * - Conversation turn orchestration
 * - Multi-bot coordination
 * - Clean delegation to ResponseService
 * 
 * Design Principles:
 * - Clean input/output types (ConversationParams → ConversationResult)
 * - Zero transformation logic (direct delegation to ResponseService)
 * - AI-powered bot selection with rule-based fallbacks
 * - Strategy-agnostic conversation handling
 */

import type {
    BotId,
    BotParticipant,
    ChatMessage,
    ConversationContext,
    ConversationEngineConfig,
    ConversationParams,
    ConversationResult,
    ConversationTrigger,
    ExecutionResourceUsage,
    ExecutionStrategy,
    ResponseContext,
    ResponseResult,
    SwarmId,
    TurnExecutionParams,
    TurnExecutionResult,
    TurnId,
} from "@vrooli/shared";
import { toTurnId } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import type { EventBus } from "../events/eventBus.js";
import type { SwarmContextManager } from "../execution/shared/SwarmContextManager.js";
import type { UnifiedSwarmContext } from "../execution/shared/UnifiedSwarmContext.js";
import type { ResponseService } from "../response/responseService.js";
import { CompositeGraph as UnifiedCompositeGraph, type AgentSelectionResult } from "./agentGraph.js";

/**
 * Default configuration for ConversationEngine
 */
const DEFAULT_CONFIG: ConversationEngineConfig = {
    maxConcurrentTurns: 5,
    defaultTimeoutMs: 300000, // 5 minutes
    enableBotSelection: true,
    enableMetrics: true,
    fallbackBehavior: "most_capable",
};

/**
 * ConversationEngine - Orchestrates multi-bot conversations with intelligence
 * 
 * This is the core conversation orchestration service that replaces the fragmented
 * conversation logic from SwarmStateMachine and provides clean interfaces for
 * both conversation and reasoning strategies.
 */
export class ConversationEngine {
    private readonly config: ConversationEngineConfig;
    private readonly agentGraph: UnifiedCompositeGraph;

    constructor(
        private readonly responseService: ResponseService,
        private readonly contextManager: SwarmContextManager,
        config?: Partial<ConversationEngineConfig>,
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };

        // Initialize the unified composite agent graph
        this.agentGraph = new UnifiedCompositeGraph();

        logger.info("[ConversationEngine] Initialized with intelligent conversation orchestration", {
            config: this.config,
        });
    }

    /**
     * 🎯 Main Entry Point: Orchestrate Multi-Bot Conversation
     * 
     * INPUT: ConversationParams (unified conversation parameters)
     * OUTPUT: ConversationResult (unified conversation result)
     * 
     * This method replaces the fragmented conversation logic from SwarmStateMachine
     * and provides intelligent bot selection, turn coordination, and result aggregation.
     */
    async orchestrateConversation(params: ConversationParams): Promise<ConversationResult> {
        const startTime = Date.now();
        const turnId = toTurnId(`turn_${params.context.swarmId}_${Date.now()}`);

        logger.info("[ConversationEngine] Starting conversation orchestration", {
            swarmId: params.context.swarmId,
            strategy: params.strategy,
            triggerType: params.trigger.type,
            participantCount: params.context.participants.length,
            turnId,
        });

        try {
            // 1. Validate conversation context
            this.validateConversationContext(params.context);

            // 2. Select responding bots (replaces missing findLeaderBot)
            const selectionResult = await this.selectRespondingBots(params.trigger, params.context);

            logger.info("[ConversationEngine] Bot selection completed", {
                selectedCount: selectionResult.responders.length,
                strategy: selectionResult.strategy,
            });

            // 3. Execute conversation turn with selected agents
            const turnResult = await this.executeTurn({
                participants: selectionResult.responders,
                context: params.context,
                strategy: params.strategy,
                turnId,
                executionMode: this.determineExecutionMode(params.strategy, selectionResult.responders),
            });

            // 4. Update participant states
            const updatedParticipants = await this.updateParticipantStates(
                params.context.participants,
                turnResult.participantResults,
            );

            // 5. Determine next conversation action
            const nextAction = this.determineNextAction(
                params.trigger,
                turnResult,
                params.context,
            );

            // 6. Build final conversation result
            const result: ConversationResult = {
                success: true,
                messages: turnResult.messages,
                updatedParticipants,
                nextAction,
                conversationComplete: this.isConversationComplete(nextAction),
                turnId,
                swarmId: params.context.swarmId,
                resourcesUsed: turnResult.resourcesUsed,
                duration: Date.now() - startTime,
                metadata: {
                    strategy: params.strategy,
                    participantCount: selectionResult.responders.length,
                    executionMode: this.determineExecutionMode(params.strategy, selectionResult.responders),
                    botSelection: {
                        requested: params.context.participants.map(p => p.id),
                        selected: selectionResult.responders.map(p => p.id),
                        selectionStrategy: selectionResult.strategy,
                    },
                    turnMetrics: turnResult.turnMetrics,
                },
            };

            logger.info("[ConversationEngine] Conversation orchestration completed", {
                success: true,
                messageCount: result.messages.length,
                duration: result.duration,
                conversationComplete: result.conversationComplete,
                nextAction: result.nextAction?.type,
            });

            return result;

        } catch (error) {
            logger.error("[ConversationEngine] Conversation orchestration failed", {
                error: error instanceof Error ? error.message : String(error),
                swarmId: params.context.swarmId,
                strategy: params.strategy,
                duration: Date.now() - startTime,
            });

            return this.createErrorResult(
                params.context.swarmId,
                turnId,
                startTime,
                error instanceof Error ? error : new Error(String(error)),
                "CONVERSATION_ORCHESTRATION_FAILED",
            );
        }
    }

    /**
     * 🤖 Rule-Based Bot Selection using AgentGraph
     * 
     * Selection priority:
     * 1. Direct mentions (respondingBots in trigger config)
     * 2. Event mapping (event-to-bot role mapping)
     * 3. Topic subscriptions (blackboard pattern subscriptions)
     * 4. Active bot (swarm baton pattern via blackboard)
     * 5. Fallback to leader
     * 
     * Public method to allow SwarmStateMachine to use bot selection logic.
     */
    public async selectRespondingBots(event: ConversationTrigger, context: UnifiedSwarmContext): Promise<AgentSelectionResult> {
        try {
            // Get UnifiedSwarmContext from contextManager
            const swarmId = (context as unknown as ConversationContext).swarmId;
            const unifiedContext = await this.contextManager.getContext(swarmId);

            if (!unifiedContext) {
                logger.warn("[ConversationEngine] No unified context found, returning empty selection");
                return {
                    responders: [],
                    strategy: "fallback",
                };
            }

            // Use UnifiedAgentGraph for selection
            const selectionResult = await this.agentGraph.selectResponders(
                unifiedContext,
                event,
            );

            return selectionResult;

        } catch (error) {
            logger.error("[ConversationEngine] Bot selection error, returning empty selection", {
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                responders: [],
                strategy: "fallback",
            };
        }
    }

    /**
     * 🔄 Execute Conversation Turn with Selected Bots
     * 
     * INPUT: TurnExecutionParams (selected bots and context)
     * OUTPUT: TurnExecutionResult (messages and resource usage)
     * 
     * This method executes a conversation turn by delegating to ResponseService
     * for each selected bot, handling parallel/sequential execution, and
     * aggregating the results with zero transformation.
     */
    private async executeTurn(params: TurnExecutionParams): Promise<TurnExecutionResult> {
        const startTime = Date.now();
        const messages: ChatMessage[] = [];
        const participantResults = new Map<BotId, ResponseResult>();
        const totalResourcesUsed: ExecutionResourceUsage = {
            creditsUsed: "0",
            durationMs: 0,
            toolCalls: 0,
            memoryUsedMB: 0,
            stepsExecuted: 0,
        };

        logger.debug("[ConversationEngine] Executing conversation turn", {
            participantCount: params.participants.length,
            strategy: params.strategy,
            executionMode: params.executionMode,
            turnId: params.turnId,
        });

        // Execute responses based on execution mode
        if (params.executionMode === "parallel") {
            await this.executeParallelTurn(params, messages, participantResults, totalResourcesUsed);
        } else {
            await this.executeSequentialTurn(params, messages, participantResults, totalResourcesUsed);
        }

        const duration = Date.now() - startTime;
        totalResourcesUsed.durationMs = duration;

        return {
            messages,
            resourcesUsed: totalResourcesUsed,
            participantResults,
            turnMetrics: {
                totalDuration: duration,
                participantCount: params.participants.length,
                messageCount: messages.length,
                toolCallCount: totalResourcesUsed.toolCalls,
                averageConfidence: this.calculateAverageConfidence(participantResults),
                executionMode: params.executionMode || "sequential",
            },
        };
    }

    /**
     * Execute turn with bots responding in parallel
     */
    private async executeParallelTurn(
        params: TurnExecutionParams,
        messages: ChatMessage[],
        participantResults: Map<BotId, ResponseResult>,
        totalResourcesUsed: ExecutionResourceUsage,
    ): Promise<void> {
        const responsePromises = params.participants.map(async (participant) => {
            const responseContext = this.createResponseContext(participant, params);
            const result = await this.responseService.generateResponse({
                context: responseContext,
                abortSignal: undefined,
            });

            return { participant, result };
        });

        const responses = await Promise.allSettled(responsePromises);

        for (const responsePromise of responses) {
            if (responsePromise.status === "fulfilled") {
                const { participant, result } = responsePromise.value;

                participantResults.set(participant.id, result);

                if (result.success) {
                    messages.push(result.message);
                    this.aggregateResourceUsage(totalResourcesUsed, result.resourcesUsed);
                }
            } else {
                logger.warn("[ConversationEngine] Parallel bot response failed", {
                    error: responsePromise.reason,
                });
            }
        }
    }

    /**
     * Execute turn with bots responding sequentially
     */
    private async executeSequentialTurn(
        params: TurnExecutionParams,
        messages: ChatMessage[],
        participantResults: Map<BotId, ResponseResult>,
        totalResourcesUsed: ExecutionResourceUsage,
    ): Promise<void> {
        for (const participant of params.participants) {
            const responseContext = this.createResponseContext(participant, params, messages);
            const result = await this.responseService.generateResponse({
                context: responseContext,
                abortSignal: undefined,
            });

            participantResults.set(participant.id, result);

            if (result.success) {
                messages.push(result.message);
                this.aggregateResourceUsage(totalResourcesUsed, result.resourcesUsed);
            } else {
                logger.warn("[ConversationEngine] Sequential bot response failed", {
                    botId: participant.id,
                    error: result.error?.message,
                });
                // Continue with other bots even if one fails
            }
        }
    }

    /**
     * Create ResponseContext for a specific bot participant
     * This is the zero-transformation delegation to ResponseService
     */
    private createResponseContext(
        participant: BotParticipant,
        params: TurnExecutionParams,
        additionalMessages: ChatMessage[] = [],
    ): ResponseContext {
        // Build complete conversation history including new messages from this turn
        const fullConversationHistory = [
            ...params.context.conversationHistory,
            ...additionalMessages,
        ];

        return {
            swarmId: params.context.swarmId,
            userData: params.context.userData,
            timestamp: new Date(),
            botId: participant.id,
            botConfig: participant.config,
            conversationHistory: fullConversationHistory,
            availableTools: params.context.availableTools,
            strategy: params.strategy,
            constraints: {
                maxTokens: 1000, // Default max tokens
                temperature: 0.7, // Default temperature
                timeoutMs: this.config.defaultTimeoutMs,
                maxCredits: "2000", // Default credit limit per bot response
            },
        };
    }

    /**
     * Utility Methods for Conversation Orchestration
     */

    private validateConversationContext(context: ConversationContext): void {
        if (!context.swarmId) {
            throw new Error("Missing swarmId in conversation context");
        }
        if (!context.userData) {
            throw new Error("Missing userData in conversation context");
        }
        if (!Array.isArray(context.participants)) {
            throw new Error("Invalid participants in conversation context");
        }
        if (!Array.isArray(context.conversationHistory)) {
            throw new Error("Invalid conversationHistory in conversation context");
        }
        if (!Array.isArray(context.availableTools)) {
            throw new Error("Invalid availableTools in conversation context");
        }
    }

    private determineExecutionMode(
        strategy: ExecutionStrategy,
        selectedParticipants: BotParticipant[],
    ): "sequential" | "parallel" | "adaptive" {
        // Simple heuristic: reasoning and deterministic prefer sequential, conversation allows parallel
        if (strategy === "reasoning" || strategy === "deterministic") {
            return "sequential";
        }
        if (selectedParticipants.length === 1) {
            return "sequential";
        }
        return "parallel";
    }

    private determineNextAction(
        trigger: ConversationTrigger,
        turnResult: TurnExecutionResult,
        _context: ConversationContext,
    ): ConversationResult["nextAction"] {
        const hasToolCalls = Array.from(turnResult.participantResults.values())
            .some(result => result.toolCalls && result.toolCalls.length > 0);

        const shouldContinue = Array.from(turnResult.participantResults.values())
            .some(result => result.continueConversation);

        if (hasToolCalls) {
            return { type: "continue", reason: "Tool calls require follow-up responses" };
        }

        if (shouldContinue) {
            return { type: "continue", reason: "Bots indicated conversation should continue" };
        }

        if (trigger.type === "user_message") {
            return { type: "wait_for_user" };
        }

        return { type: "complete", result: "Conversation completed", reason: "No further action required" };
    }

    private isConversationComplete(nextAction?: ConversationResult["nextAction"]): boolean {
        return nextAction?.type === "complete" || nextAction?.type === "error";
    }

    private aggregateResourceUsage(total: ExecutionResourceUsage, additional: ExecutionResourceUsage): void {
        const totalCredits = BigInt(total.creditsUsed) + BigInt(additional.creditsUsed);
        total.creditsUsed = totalCredits.toString();
        total.toolCalls += additional.toolCalls;
        total.memoryUsedMB += additional.memoryUsedMB;
        total.stepsExecuted += additional.stepsExecuted;
    }

    private calculateAverageConfidence(participantResults: Map<BotId, ResponseResult>): number {
        const confidenceValues = Array.from(participantResults.values())
            .map(result => result.confidence)
            .filter((confidence): confidence is number => confidence !== undefined);

        if (confidenceValues.length === 0) {
            return 0.5; // Default confidence
        }

        return confidenceValues.reduce((sum, confidence) => sum + confidence, 0) / confidenceValues.length;
    }

    private async updateParticipantStates(
        originalParticipants: BotParticipant[],
        participantResults: Map<BotId, ResponseResult>,
    ): Promise<ConversationResult["updatedParticipants"]> {
        return originalParticipants.map(participant => {
            const result = participantResults.get(participant.id);

            return {
                botId: participant.id,
                success: result?.success ?? false,
                message: result?.success ? result.message : undefined,
                updatedState: {
                    isProcessing: false,
                    isWaiting: false,
                    hasResponded: result?.success ?? false,
                    error: result?.error?.message,
                    lastActive: new Date(),
                },
                resourcesUsed: result?.resourcesUsed ?? {
                    creditsUsed: "0",
                    durationMs: 0,
                    toolCalls: 0,
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
                error: result?.error,
            };
        });
    }

    private createErrorResult(
        swarmId: SwarmId,
        turnId: TurnId,
        startTime: number,
        error: Error,
        errorCode: string,
    ): ConversationResult {
        const duration = Date.now() - startTime;

        return {
            success: false,
            messages: [],
            updatedParticipants: [],
            conversationComplete: true,
            turnId,
            swarmId,
            error: {
                code: errorCode,
                message: error.message,
                tier: "tier2", // ConversationEngine is part of tier2
                type: error.constructor.name,
                context: {
                    timestamp: new Date().toISOString(),
                    service: "ConversationEngine",
                    swarmId,
                    turnId,
                },
            },
            resourcesUsed: {
                creditsUsed: "0",
                durationMs: duration,
                toolCalls: 0,
                memoryUsedMB: 0,
                stepsExecuted: 0,
            },
            duration,
            metadata: {
                errorCode,
                errorMessage: error.message,
            },
        };
    }

    /**
     * Factory method to create ConversationEngine with proper dependencies
     */
    static create(
        responseService: ResponseService,
        contextManager: SwarmContextManager,
        _eventBus: EventBus, // Kept for interface compatibility, not used
        config?: Partial<ConversationEngineConfig>,
    ): ConversationEngine {
        return new ConversationEngine(responseService, contextManager, config);
    }
}

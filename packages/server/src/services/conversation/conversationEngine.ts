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
    BotSelectionContext,
    BotSelectionResult,
    ChatMessage,
    ConversationContext,
    ConversationEngineConfig,
    ConversationParams,
    ConversationResult,
    ConversationTrigger,
    ExecutionResourceUsage,
    ResponseContext,
    ResponseResult,
    SessionUser,
    SwarmId,
    Tool,
    TurnExecutionParams,
    TurnExecutionResult,
    TurnId,
} from "@vrooli/shared";
import { generatePK, toBotId, toSwarmId, toTurnId } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import type { EventBus } from "../events/eventBus.js";
import type { SwarmContextManager } from "../execution/shared/SwarmContextManager.js";
import type { ResponseService } from "../response/responseService.js";
import { PromptService, type PromptContext } from "./promptService.js";

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

    constructor(
        private readonly responseService: ResponseService,
        private readonly contextManager: SwarmContextManager,
        config?: Partial<ConversationEngineConfig>,
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };

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
            const selectionResult = await this.selectRespondingBots({
                trigger: params.trigger,
                availableParticipants: params.context.participants,
                conversationHistory: params.context.conversationHistory,
                sharedState: params.context.sharedState ?? {},
                constraints: params.constraints ?? {},
            });

            logger.info("[ConversationEngine] Bot selection completed", {
                selectedCount: selectionResult.selectedBots.length,
                selectionReason: selectionResult.selectionReason,
                confidence: selectionResult.confidence,
                fallbackUsed: selectionResult.fallbackUsed,
            });

            // 3. Execute conversation turn with selected bots
            const turnResult = await this.executeTurn({
                participants: selectionResult.selectedBots,
                context: params.context,
                strategy: params.strategy,
                turnId,
                executionMode: this.determineExecutionMode(params.strategy, selectionResult.selectedBots),
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
                    participantCount: selectionResult.selectedBots.length,
                    executionMode: this.determineExecutionMode(params.strategy, selectionResult.selectedBots),
                    botSelection: {
                        requested: params.context.participants.map(p => p.id),
                        selected: selectionResult.selectedBots.map(p => p.id),
                        selectionReason: selectionResult.selectionReason,
                        fallbackUsed: selectionResult.fallbackUsed,
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
     * 🤖 Intelligent Bot Selection - Replaces Missing findLeaderBot
     * 
     * INPUT: BotSelectionContext (selection criteria and context)
     * OUTPUT: BotSelectionResult (selected bots with reasoning)
     * 
     * This method uses AI-powered bot selection to intelligently choose which bots
     * should participate in the conversation, replacing the missing findLeaderBot
     * method from SwarmStateMachine with superior AI-driven selection.
     * 
     * Public method to allow SwarmStateMachine to use bot selection logic.
     */
    public async selectRespondingBots(context: BotSelectionContext): Promise<BotSelectionResult> {
        if (!this.config.enableBotSelection || context.availableParticipants.length === 0) {
            return this.fallbackBotSelection(context);
        }

        try {
            logger.debug("[ConversationEngine] Starting AI-powered bot selection", {
                availableCount: context.availableParticipants.length,
                triggerType: context.trigger.type,
                historyLength: context.conversationHistory.length,
            });

            // Create AI-powered bot selection using ResponseService
            const botSelectionPrompt = await this.buildBotSelectionPrompt(context);
            const selectionContext: ResponseContext = {
                swarmId: toSwarmId("bot_selector"),
                userData: { id: "system", name: "Bot Selector" } as SessionUser,
                timestamp: new Date(),
                botId: toBotId("bot_selector"),
                botConfig: this.createBotSelectorConfig(),
                conversationHistory: botSelectionPrompt,
                availableTools: [this.createBotSelectionTool()],
                strategy: "reasoning",
                constraints: {
                    maxTokens: 1000,
                    temperature: 0.3,
                    timeoutMs: 30000,
                    maxCredits: "500", // Small credit limit for bot selection
                },
            };

            const response = await this.responseService.generateResponse({
                context: selectionContext,
                abortSignal: undefined,
            });

            if (response.success && response.toolCalls) {
                const selectionTool = response.toolCalls.find(tc => tc.function.name === "select_bots");
                if (selectionTool) {
                    try {
                        const selection = JSON.parse(selectionTool.function.arguments as string);
                        const selectedBots = context.availableParticipants.filter(p =>
                            selection.selectedBotIds.includes(p.id),
                        );

                        if (selectedBots.length > 0) {
                            logger.info("[ConversationEngine] AI bot selection successful", {
                                selectedCount: selectedBots.length,
                                confidence: response.confidence ?? 0.7, // eslint-disable-line no-magic-numbers
                                reason: selection.reason,
                            });

                            return {
                                selectedBots,
                                selectionReason: selection.reason || "AI-powered selection",
                                confidence: response.confidence ?? 0.7, // eslint-disable-line no-magic-numbers
                                fallbackUsed: false,
                                alternativeOptions: context.availableParticipants.filter(p =>
                                    !selection.selectedBotIds.includes(p.id),
                                ),
                                selectionMetadata: {
                                    aiModel: selectionContext.botConfig.model,
                                    selectionTime: response.duration,
                                    creditsUsed: response.resourcesUsed.creditsUsed,
                                },
                            };
                        }
                    } catch (parseError) {
                        logger.warn("[ConversationEngine] Failed to parse AI bot selection", {
                            error: parseError instanceof Error ? parseError.message : String(parseError),
                        });
                    }
                }
            }

            // AI selection failed, use fallback
            logger.warn("[ConversationEngine] AI bot selection failed, using fallback", {
                responseSuccess: response.success,
                toolCallCount: response.toolCalls?.length ?? 0,
            });

            return this.fallbackBotSelection(context);

        } catch (error) {
            logger.error("[ConversationEngine] Bot selection error, using fallback", {
                error: error instanceof Error ? error.message : String(error),
            });

            return this.fallbackBotSelection(context);
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
                const botId = toBotId(participant.id);

                participantResults.set(botId, result);

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

            const botId = toBotId(participant.id);
            participantResults.set(botId, result);

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
            botId: toBotId(participant.id),
            botConfig: participant.config,
            conversationHistory: fullConversationHistory,
            availableTools: params.context.availableTools,
            strategy: params.strategy,
            constraints: {
                maxTokens: participant.config.maxTokens,
                temperature: participant.config.temperature,
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
        strategy: "conversation" | "reasoning",
        selectedBots: BotParticipant[],
    ): "sequential" | "parallel" | "adaptive" {
        // Simple heuristic: reasoning strategy prefers sequential, conversation allows parallel
        if (strategy === "reasoning") {
            return "sequential";
        }
        if (selectedBots.length === 1) {
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

    /**
     * Rule-based fallback bot selection when AI selection fails
     */
    private fallbackBotSelection(context: BotSelectionContext): BotSelectionResult {
        logger.info("[ConversationEngine] Using fallback bot selection", {
            fallbackBehavior: this.config.fallbackBehavior,
            availableCount: context.availableParticipants.length,
        });

        let selectedBots: BotParticipant[];
        let selectionReason: string;

        switch (this.config.fallbackBehavior) {
            case "most_capable":
                // Select bots with the most capabilities/specializations
                selectedBots = context.availableParticipants
                    .sort((a, b) => {
                        const aCapabilities = (a.capabilities?.length || 0) + (a.specializations?.length || 0);
                        const bCapabilities = (b.capabilities?.length || 0) + (b.specializations?.length || 0);
                        return bCapabilities - aCapabilities;
                    })
                    .slice(0, Math.min(2, context.availableParticipants.length));
                selectionReason = "Fallback: Selected most capable bots based on capabilities count";
                break;

            case "random": {
                // Random selection
                const shuffled = [...context.availableParticipants].sort(() => Math.random() - 0.5);
                selectedBots = shuffled.slice(0, Math.min(1, context.availableParticipants.length));
                selectionReason = "Fallback: Random bot selection";
                break;
            }

            case "first_available":
            default:
                // Default: select first available bot
                selectedBots = context.availableParticipants.slice(0, 1);
                selectionReason = "Fallback: Selected first available bot";
                break;
        }

        return {
            selectedBots,
            selectionReason,
            confidence: 0.5, // Medium confidence for fallback
            fallbackUsed: true,
            alternativeOptions: context.availableParticipants.filter(p =>
                !selectedBots.some(selected => selected.id === p.id),
            ),
        };
    }

    /**
     * Bot selection AI helper methods
     */
    private createBotSelectorConfig() {
        return {
            id: "bot_selector",
            name: "Bot Selector",
            description: "AI agent specialized in selecting appropriate bots for conversation tasks",
            model: "gpt-4",
            maxTokens: 1000,
            temperature: 0.3,
            systemPrompt: "You are an expert at analyzing conversation contexts and selecting the most appropriate bots to handle specific tasks.",
        };
    }

    /**
     * Enhanced bot selection prompt using PromptService with proper bot configuration
     * This uses the enhanced PromptService to properly handle bot configuration and swarm context
     */
    private async buildBotSelectionPrompt(context: BotSelectionContext): Promise<ChatMessage[]> {
        try {
            // Create a bot selector bot configuration
            const botSelectorBot = {
                id: "bot_selector",
                name: "Bot Selector",
                config: {
                    ...this.createBotSelectorConfig(),
                    agentSpec: {
                        goal: "Select the most appropriate bots for conversation tasks",
                        role: "coordinator",
                        prompt: {
                            mode: "direct" as const,
                            source: "direct" as const,
                            content: this.createBotSelectionDirectPrompt(context),
                            variables: {
                                "trigger.type": "context.trigger.type",
                                "available.bots": "context.availableParticipants",
                                "conversation.history": "context.conversationHistory",
                            },
                        },
                    },
                },
                meta: {
                    role: "coordinator",
                },
                capabilities: ["bot_selection", "context_analysis"],
                specializations: ["conversation_orchestration"],
            };

            // Create PromptContext for bot selection
            const promptContext: PromptContext = {
                goal: "Select appropriate bots for the current conversation task",
                bot: botSelectorBot,
                convoConfig: {
                    teamId: "bot_selection_team",
                    swarmLeader: "bot_selector",
                    subtasks: [],
                    subtaskLeaders: {},
                    eventSubscriptions: {},
                    blackboard: [],
                    resources: [],
                    records: [],
                    stats: {},
                    limits: {},
                    pendingToolCalls: [],
                },
                swarmId: "bot_selection_swarm",
                userId: "system",
                // Pass context data for variable resolution
                input: {
                    trigger: context.trigger,
                    availableParticipants: context.availableParticipants,
                    conversationHistory: context.conversationHistory,
                    sharedState: context.sharedState,
                    constraints: context.constraints,
                },
            };

            // Use PromptService to build the system message
            const systemPrompt = await PromptService.buildSystemMessage(promptContext);

            return [{
                id: generatePK().toString(),
                createdAt: new Date(),
                config: {
                    role: "user",
                    text: systemPrompt,
                },
                user: { id: "system" },
            }];
        } catch (error) {
            logger.warn("[ConversationEngine] Failed to use PromptService for bot selection, falling back to basic prompt", { error });
            return this.createFallbackBotSelectionPrompt(context);
        }
    }

    /**
     * Create direct prompt content for bot selection
     */
    private createBotSelectionDirectPrompt(context: BotSelectionContext): string {
        return `You are an expert bot selector. Analyze the conversation context and select the most appropriate bots.

CONTEXT:
Trigger Type: {{input.trigger.type}}
Available Participants: {{input.availableParticipants.length}} bots
Conversation History: {{input.conversationHistory.length}} messages

Your task is to select the best bot(s) for this situation. Consider:
- Bot capabilities and specializations  
- Context requirements and complexity
- Conversation flow and user needs
- Resource efficiency and response quality

Use the select_bots tool to make your selection with reasoning.`;
    }

    /**
     * Create fallback bot selection prompt when PromptService fails
     */
    private createFallbackBotSelectionPrompt(context: BotSelectionContext): ChatMessage[] {
        const contextSummary = this.buildContextSummary(context);
        const availableBotsSummary = this.buildAvailableBotsSummary(context.availableParticipants);
        
        const fallbackPrompt = `Select appropriate bots for conversation task.

Context: ${contextSummary}
Available Bots: ${availableBotsSummary}
Conversation: ${context.conversationHistory.length} messages

Use select_bots tool to choose the best bot(s) with reasoning.`;

        return [{
            id: generatePK().toString(),
            createdAt: new Date(),
            config: {
                role: "user",
                text: fallbackPrompt,
            },
            user: { id: "system" },
        }];
    }

    /**
     * Build context summary for bot selection template
     */
    private buildContextSummary(context: BotSelectionContext): string {
        const MAX_MESSAGE_PREVIEW_LENGTH = 200;
        let summary = `Trigger: ${context.trigger.type}`;

        if (context.trigger.type === "user_message") {
            const message = (context.trigger as { type: "user_message"; message: ChatMessage }).message;
            if (message?.text) {
                summary += `\nUser Message: "${message.text.substring(0, MAX_MESSAGE_PREVIEW_LENGTH)}${message.text.length > MAX_MESSAGE_PREVIEW_LENGTH ? "..." : ""}"`;
            }
        }

        if (Object.keys(context.sharedState || {}).length > 0) {
            summary += `\nShared State: Active with ${Object.keys(context.sharedState || {}).length} entries`;
        }

        if (Object.keys(context.constraints || {}).length > 0) {
            summary += `\nConstraints: ${Object.keys(context.constraints || {}).length} active constraints`;
        }

        return summary;
    }

    /**
     * Build available bots summary for bot selection template
     */
    private buildAvailableBotsSummary(participants: BotParticipant[]): string {
        return participants.map(bot => {
            const capabilities = bot.capabilities?.join(", ") || "None specified";
            const specializations = bot.specializations?.join(", ") || "None specified";
            const description = bot.config.description || "No description";

            return `- ${bot.name} (${bot.id}): ${description}
  Capabilities: ${capabilities}
  Specializations: ${specializations}`;
        }).join("\n");
    }

    private createBotSelectionTool(): Tool {
        return {
            name: "select_bots",
            description: "Select appropriate bots for the conversation task",
            inputSchema: {
                type: "object",
                properties: {
                    selectedBotIds: {
                        type: "array",
                        items: { type: "string" },
                        description: "Array of bot IDs to select for the conversation",
                    },
                    reason: {
                        type: "string",
                        description: "Explanation for why these bots were selected",
                    },
                    confidence: {
                        type: "number",
                        minimum: 0,
                        maximum: 1,
                        description: "Confidence in this selection (0-1)",
                    },
                },
                required: ["selectedBotIds", "reason"],
            },
        };
    }

    private async updateParticipantStates(
        originalParticipants: BotParticipant[],
        participantResults: Map<BotId, ResponseResult>,
    ): Promise<ConversationResult["updatedParticipants"]> {
        return originalParticipants.map(participant => {
            const botId = toBotId(participant.id);
            const result = participantResults.get(botId);

            return {
                botId,
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
                tier: "conversation-engine",
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
        eventBus: EventBus,
        config?: Partial<ConversationEngineConfig>,
    ): ConversationEngine {
        return new ConversationEngine(responseService, contextManager, eventBus, config);
    }
}

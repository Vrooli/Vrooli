/**
 * Simple Strategy Provider
 * 
 * Provides basic manifest-based strategy selection that emits events for
 * strategy evolution agents to analyze and optimize.
 * 
 * This replaces the deprecated StrategySelector which contained embedded
 * intelligence that should emerge from agent routines.
 */

import {
    type ExecutionContext,
    type ExecutionStrategy,
    type IExecutionStrategy,
    type StrategyFactoryConfig,
    nanoid,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import type { ConversationEngine } from "../../../conversation/conversationEngine.js";
import { getEventBus } from "../../../events/eventBus.js";
import type { ResponseService } from "../../../response/responseService.js";
import { ConversationalStrategy } from "../strategies/conversationalStrategy.js";
import { DeterministicStrategy } from "../strategies/deterministicStrategy.js";
import { ReasoningStrategy } from "../strategies/reasoningStrategy.js";
import { type ToolOrchestrator } from "./toolOrchestrator.js";
import { type ValidationEngine } from "./validationEngine.js";

/**
 * Usage hints provided by Tier 2 for agent analysis
 */
export interface UsageHints {
    historicalSuccessRate?: number;
    executionFrequency?: number;
    averageComplexity?: number;
    userPreference?: ExecutionStrategy;
    domainRestrictions?: string[];
}

/**
 * Simple Strategy Provider - Enables agent-driven strategy optimization
 * 
 * This component provides ONLY:
 * 1. Manifest-based strategy selection (data-driven)
 * 2. Event emission for strategy evolution agents
 * 3. Basic strategy instantiation
 * 
 * All sophisticated strategy selection, graduation logic, and optimization
 * emerges from agents analyzing strategy execution events.
 */
export class SimpleStrategyProvider {
    private readonly strategies: Map<ExecutionStrategy, IExecutionStrategy>;
    private readonly config: StrategyFactoryConfig;
    private readonly toolOrchestrator?: ToolOrchestrator;
    private readonly validationEngine?: ValidationEngine;
    private readonly conversationEngine?: ConversationEngine;
    private readonly responseService?: ResponseService;

    constructor(
        config: StrategyFactoryConfig,
        toolOrchestrator?: ToolOrchestrator,
        validationEngine?: ValidationEngine,
        conversationEngine?: ConversationEngine,
        responseService?: ResponseService,
    ) {
        this.config = config;
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;
        this.conversationEngine = conversationEngine;
        this.responseService = responseService;
        this.strategies = new Map();

        // Initialize available strategies
        this.initializeStrategies();
    }

    /**
     * Helper method for publishing events using unified event system
     */
    private async publishEvent(
        eventType: string,
        data: any,
        options?: {
            deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
            priority?: "low" | "medium" | "high" | "critical";
            tags?: string[];
        },
    ): Promise<void> {
        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                data,
                EventUtils.createEventSource(3, "SimpleStrategyProvider", nanoid()),
                EventUtils.createEventMetadata(
                    options?.deliveryGuarantee || "fire-and-forget",
                    options?.priority || "medium",
                    {
                        tags: options?.tags || ["strategy", "tier3"],
                    },
                ),
            );

            getEventBus().publish(event);

        } catch (eventError) {
            logger.error("[SimpleStrategyProvider] Failed to publish unified event", {
                eventType,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });
        }
    }

    /**
     * Gets strategy based on manifest declaration and emits events for agent optimization
     */
    async getStrategy(
        context: ExecutionContext,
        usageHints?: UsageHints,
    ): Promise<IExecutionStrategy> {
        const stepId = context.stepId;

        // 1. Get declared strategy from manifest (data-driven)
        const declaredStrategy = this.getDeclaredStrategy(context);

        logger.debug(`[SimpleStrategyProvider] Manifest strategy: ${declaredStrategy}`, { stepId });

        // 2. Basic validation - only essential safety checks
        const validatedStrategy = await this.validateBasicSafety(declaredStrategy, context);

        // 3. Emit strategy selection event for evolution agents to analyze using unified event system
        await this.publishEvent(EventTypes.STRATEGY_PERFORMANCE_MEASURED, {
            stepId,
            routineId: context.routineId,
            declaredStrategy,
            selectedStrategy: validatedStrategy,
            usageHints,
            context: {
                stepType: context.stepType,
                tier: "tier3",
                userId: context.userData.id,
                swarmId: context.swarmId,
            },
            selectionReason: validatedStrategy === declaredStrategy ? "manifest_declared" : "safety_override",
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            tags: ["strategy", "selection", "optimization"],
        });

        // 4. Return strategy instance
        return this.getOrCreateStrategy(validatedStrategy);
    }

    /**
     * Gets declared strategy from context (purely data-driven)
     */
    private getDeclaredStrategy(context: ExecutionContext): ExecutionStrategy {
        // Check explicit strategy in config
        const configStrategy = context.config?.strategy as string;
        if (configStrategy && this.isValidStrategy(configStrategy)) {
            return configStrategy as ExecutionStrategy;
        }

        // Check step type hints (basic pattern matching)
        const stepType = context.stepType?.toLowerCase() || "";
        if (stepType.includes("deterministic") || stepType.includes("automated")) {
            return "deterministic";
        }
        if (stepType.includes("reasoning") || stepType.includes("analysis")) {
            return "reasoning";
        }

        // Default to configured default
        return this.config.defaultStrategy;
    }

    /**
     * Basic safety validation only - complex policies handled by security agents
     */
    private async validateBasicSafety(strategy: ExecutionStrategy, context: ExecutionContext): Promise<ExecutionStrategy> {
        // Only essential safety override: no conversational for explicitly sensitive data
        const hasSensitiveData = context.config?.sensitiveData === true;

        if (strategy === "conversational" && hasSensitiveData) {
            logger.warn("[SimpleStrategyProvider] Safety override: conversational not allowed for sensitive data", {
                stepId: context.stepId,
                originalStrategy: strategy,
            });

            // Emit safety override event for security agents using unified event system
            await this.publishEvent(EventTypes.THREAT_DETECTED, {
                stepId: context.stepId,
                originalStrategy: strategy,
                overrideStrategy: "deterministic",
                reason: "sensitive_data_protection",
                threatType: "strategy_safety_override",
                timestamp: new Date(),
            }, {
                deliveryGuarantee: "reliable",
                priority: "high",
                tags: ["security", "strategy", "threat-detected"],
            });

            return "deterministic";
        }

        return strategy;
    }

    /**
     * Gets or creates a strategy instance
     */
    private getOrCreateStrategy(type: ExecutionStrategy): IExecutionStrategy {
        let strategy = this.strategies.get(type);

        if (!strategy) {
            strategy = this.createStrategy(type);
            this.strategies.set(type, strategy);
        }

        return strategy;
    }

    /**
     * Creates a strategy instance
     */
    private createStrategy(type: ExecutionStrategy): IExecutionStrategy {
        switch (type) {
            case "conversational":
                if (!this.conversationEngine) {
                    throw new Error("ConversationEngine is required for ConversationalStrategy");
                }
                return new ConversationalStrategy(this.conversationEngine);

            case "reasoning":
                // ReasoningStrategy uses ResponseService for single-turn reasoning
                if (!this.responseService) {
                    throw new Error("ResponseService is required for ReasoningStrategy");
                }
                return new ReasoningStrategy(this.responseService, this.toolOrchestrator, this.validationEngine);

            case "deterministic":
                return new DeterministicStrategy(this.toolOrchestrator, this.validationEngine);

            default:
                logger.warn(`[SimpleStrategyProvider] Unknown strategy type: ${type}, falling back to conversational`);
                if (!this.conversationEngine) {
                    throw new Error("ConversationEngine is required for ConversationalStrategy fallback");
                }
                return new ConversationalStrategy(this.conversationEngine);
        }
    }

    /**
     * Initializes available strategies
     */
    private initializeStrategies(): void {
        // Strategies are created on-demand to reduce memory usage
        logger.debug("[SimpleStrategyProvider] Strategy provider initialized");
    }

    /**
     * Validates strategy type
     */
    private isValidStrategy(strategy: string): boolean {
        const validStrategies: ExecutionStrategy[] = ["conversational", "reasoning", "deterministic"];
        return validStrategies.includes(strategy);
    }
}

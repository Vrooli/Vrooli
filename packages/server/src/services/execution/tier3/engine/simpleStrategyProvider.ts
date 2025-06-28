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
    type StrategyFactoryConfig,
    type StrategyType,
    StrategyType as StrategyTypeEnum,
    nanoid,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../../../events/types.js";
import { type IEventBus, EventTypes, EventUtils } from "../../../events/index.js";
import { getUnifiedEventSystem } from "../../../events/initialization/eventSystemService.js";
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
    userPreference?: StrategyType;
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
    private readonly strategies: Map<StrategyType, ExecutionStrategy>;
    private readonly config: StrategyFactoryConfig;
    private readonly logger: Logger;
    private readonly toolOrchestrator?: ToolOrchestrator;
    private readonly validationEngine?: ValidationEngine;
    private readonly eventBus: IEventBus;
    private readonly unifiedEventBus: IEventBus | null;

    constructor(
        config: StrategyFactoryConfig,
        logger: Logger,
        eventBus: IEventBus,
        toolOrchestrator?: ToolOrchestrator,
        validationEngine?: ValidationEngine,
    ) {
        this.config = config;
        this.logger = logger;
        this.eventBus = eventBus;
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;
        this.strategies = new Map();

        // Get unified event system for modern event publishing
        this.unifiedEventBus = getUnifiedEventSystem();

        // Initialize available strategies
        this.initializeStrategies();
    }

    /**
     * Helper method for publishing events using unified event system
     */
    private async publishUnifiedEvent(
        eventType: string,
        data: any,
        options?: {
            deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
            priority?: "low" | "medium" | "high" | "critical";
            tags?: string[];
        },
    ): Promise<void> {
        if (!this.unifiedEventBus) {
            this.logger.debug("[SimpleStrategyProvider] Unified event bus not available, falling back to legacy publisher");
            await this.eventBus.publish(eventType, data);
            return;
        }

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

            await this.unifiedEventBus.publish(event);

            this.logger.debug("[SimpleStrategyProvider] Published unified event", {
                eventType,
                deliveryGuarantee: options?.deliveryGuarantee,
                priority: options?.priority,
            });

        } catch (eventError) {
            this.logger.error("[SimpleStrategyProvider] Failed to publish unified event", {
                eventType,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });

            // Fallback to legacy publisher
            await this.eventBus.publish(eventType, data);
        }
    }

    /**
     * Gets strategy based on manifest declaration and emits events for agent optimization
     */
    async getStrategy(
        context: ExecutionContext,
        usageHints?: UsageHints,
    ): Promise<ExecutionStrategy> {
        const stepId = context.stepId;

        // 1. Get declared strategy from manifest (data-driven)
        const declaredStrategy = this.getDeclaredStrategy(context);

        this.logger.debug(`[SimpleStrategyProvider] Manifest strategy: ${declaredStrategy}`, { stepId });

        // 2. Basic validation - only essential safety checks
        const validatedStrategy = await this.validateBasicSafety(declaredStrategy, context);

        // 3. Emit strategy selection event for evolution agents to analyze using unified event system
        await this.publishUnifiedEvent(EventTypes.STRATEGY_PERFORMANCE_MEASURED, {
            stepId,
            routineId: context.routineId,
            declaredStrategy,
            selectedStrategy: validatedStrategy,
            usageHints,
            context: {
                stepType: context.stepType,
                tier: "tier3",
                userId: context.userId,
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
    private getDeclaredStrategy(context: ExecutionContext): StrategyType {
        // Check explicit strategy in config
        const configStrategy = context.config?.strategy as string;
        if (configStrategy && this.isValidStrategy(configStrategy)) {
            return configStrategy as StrategyType;
        }

        // Check step type hints (basic pattern matching)
        const stepType = context.stepType?.toLowerCase() || "";
        if (stepType.includes("deterministic") || stepType.includes("automated")) {
            return StrategyTypeEnum.DETERMINISTIC;
        }
        if (stepType.includes("reasoning") || stepType.includes("analysis")) {
            return StrategyTypeEnum.REASONING;
        }

        // Default to configured default
        return this.config.defaultStrategy;
    }

    /**
     * Basic safety validation only - complex policies handled by security agents
     */
    private async validateBasicSafety(strategy: StrategyType, context: ExecutionContext): Promise<StrategyType> {
        // Only essential safety override: no conversational for explicitly sensitive data
        const hasSensitiveData = context.config?.sensitiveData === true;

        if (strategy === StrategyTypeEnum.CONVERSATIONAL && hasSensitiveData) {
            this.logger.warn("[SimpleStrategyProvider] Safety override: conversational not allowed for sensitive data", {
                stepId: context.stepId,
                originalStrategy: strategy,
            });

            // Emit safety override event for security agents using unified event system
            await this.publishUnifiedEvent(EventTypes.THREAT_DETECTED, {
                stepId: context.stepId,
                originalStrategy: strategy,
                overrideStrategy: StrategyTypeEnum.DETERMINISTIC,
                reason: "sensitive_data_protection",
                threatType: "strategy_safety_override",
                timestamp: new Date(),
            }, {
                deliveryGuarantee: "reliable",
                priority: "high",
                tags: ["security", "strategy", "threat-detected"],
            });

            return StrategyTypeEnum.DETERMINISTIC;
        }

        return strategy;
    }

    /**
     * Gets or creates a strategy instance
     */
    private getOrCreateStrategy(type: StrategyType): ExecutionStrategy {
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
    private createStrategy(type: StrategyType): ExecutionStrategy {
        switch (type) {
            case StrategyTypeEnum.CONVERSATIONAL:
                return new ConversationalStrategy(this.logger, this.toolOrchestrator, this.validationEngine);

            case StrategyTypeEnum.REASONING:
                return new ReasoningStrategy(this.logger);

            case StrategyTypeEnum.DETERMINISTIC:
                return new DeterministicStrategy(this.logger, this.toolOrchestrator, this.validationEngine);

            default:
                this.logger.warn(`[SimpleStrategyProvider] Unknown strategy type: ${type}, falling back to conversational`);
                return new ConversationalStrategy(this.logger, this.toolOrchestrator, this.validationEngine);
        }
    }

    /**
     * Initializes available strategies
     */
    private initializeStrategies(): void {
        // Strategies are created on-demand to reduce memory usage
        this.logger.debug("[SimpleStrategyProvider] Strategy provider initialized");
    }

    /**
     * Validates strategy type
     */
    private isValidStrategy(strategy: string): boolean {
        return Object.values(StrategyTypeEnum).includes(strategy as StrategyType);
    }
}

// Migration helper removed - use SimpleStrategyProvider.getStrategy() directly

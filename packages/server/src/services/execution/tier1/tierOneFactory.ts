/**
 * Tier One Factory - Creates and configures Tier 1 components with LLM integration
 * 
 * Sets up the complete Tier 1 coordination intelligence with real LLM reasoning.
 */

import { type Logger } from "winston";
import { EventBus } from "../cross-cutting/eventBus.js";
import { LLMService } from "./llm/llmService.js";
import { MockLLMProvider } from "./llm/providers/mockProvider.js";
import { StrategyEngine } from "./intelligence/strategyEngine.js";
import { MetacognitiveMonitor } from "./intelligence/metacognitiveMonitor.js";

/**
 * Tier 1 configuration
 */
export interface TierOneConfig {
    llm: {
        defaultProvider?: string;
        mockMode?: boolean; // Use mock provider for development
    };
}

/**
 * Tier 1 components
 */
export interface TierOneComponents {
    llmService: LLMService;
    strategyEngine: StrategyEngine;
    metacognitiveMonitor: MetacognitiveMonitor;
}

/**
 * Factory for creating Tier 1 coordination intelligence
 */
export class TierOneFactory {
    /**
     * Creates and configures all Tier 1 components
     */
    static async create(
        logger: Logger,
        eventBus: EventBus,
        config: TierOneConfig = { llm: { mockMode: true } },
    ): Promise<TierOneComponents> {
        logger.info("[TierOneFactory] Creating Tier 1 components", {
            mockMode: config.llm.mockMode,
        });

        // Create LLM Service
        const llmService = new LLMService(logger, eventBus);

        // Register providers
        if (config.llm.mockMode) {
            const mockProvider = new MockLLMProvider();
            llmService.registerProvider(mockProvider);
            logger.info("[TierOneFactory] Registered mock LLM provider");
        }

        // TODO: Add real LLM providers here
        // if (config.llm.openai) {
        //     const openaiProvider = new OpenAIProvider(config.llm.openai);
        //     llmService.registerProvider(openaiProvider);
        // }

        // Set default provider
        if (config.llm.defaultProvider) {
            llmService.setDefaultProvider(config.llm.defaultProvider);
        }

        // Create Strategy Engine with LLM integration
        const strategyEngine = new StrategyEngine(logger, eventBus, llmService);

        // Create Metacognitive Monitor (simplified)
        const metacognitiveMonitor = new MetacognitiveMonitor(eventBus, logger);

        logger.info("[TierOneFactory] Tier 1 components created successfully");

        return {
            llmService,
            strategyEngine,
            metacognitiveMonitor,
        };
    }

    /**
     * Validates LLM integration
     */
    static async validateLLMIntegration(components: TierOneComponents): Promise<boolean> {
        try {
            const providers = components.llmService.getProviders();
            if (providers.length === 0) {
                return false;
            }

            const status = await components.llmService.getProviderStatus();
            const hasAvailableProvider = Object.values(status).some(s => s.available);

            return hasAvailableProvider;
        } catch (error) {
            return false;
        }
    }

    /**
     * Tests LLM reasoning capability
     */
    static async testReasoning(components: TierOneComponents): Promise<{
        success: boolean;
        analysis?: string;
        decision?: string;
        error?: string;
    }> {
        try {
            // Test strategic analysis
            const analysis = await components.strategyEngine.analyzeSituation({
                goal: "Test strategic analysis capability",
                observations: { testMode: true },
                knowledge: {
                    swarmId: "test-swarm",
                    facts: new Map(),
                    insights: [],
                    decisions: [],
                },
                progress: {
                    tasksCompleted: 0,
                    tasksTotal: 1,
                    currentPhase: "testing",
                    milestones: [],
                },
            });

            // Test decision generation
            const decisions = await components.strategyEngine.generateDecisions({
                goal: "Test decision generation",
                orientation: { testMode: true },
                constraints: { budget: 100 },
            });

            return {
                success: true,
                analysis,
                decision: decisions[0]?.decision || "No decisions generated",
            };

        } catch (error) {
            return {
                success: false,
                error: error instanceof Error ? error.message : String(error),
            };
        }
    }
}
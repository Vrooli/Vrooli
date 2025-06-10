/**
 * Tier One Factory - Creates and configures Tier 1 components
 * 
 * Sets up the complete Tier 1 coordination intelligence using the ConversationBridge
 * for real LLM reasoning through the existing conversation service infrastructure.
 */

import { type Logger } from "winston";
import { EventBus } from "../cross-cutting/events/eventBus.js";
import { StrategyEngine } from "./intelligence/strategyEngine.js";
import { MetacognitiveMonitor } from "./intelligence/metacognitiveMonitor.js";

/**
 * Tier 1 configuration
 */
export interface TierOneConfig {
    // Configuration can be extended as needed
    enableMetacognition?: boolean;
}

/**
 * Tier 1 components
 */
export interface TierOneComponents {
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
        config: TierOneConfig = { enableMetacognition: true },
    ): Promise<TierOneComponents> {
        logger.info("[TierOneFactory] Creating Tier 1 components", {
            enableMetacognition: config.enableMetacognition,
        });

        // Create Strategy Engine (uses ConversationBridge for real LLM reasoning)
        const strategyEngine = new StrategyEngine(logger, eventBus);

        // Create Metacognitive Monitor (event-driven intelligence)
        const metacognitiveMonitor = new MetacognitiveMonitor(eventBus, logger);

        logger.info("[TierOneFactory] Tier 1 components created successfully");

        return {
            strategyEngine,
            metacognitiveMonitor,
        };
    }

    /**
     * Validates LLM integration through ConversationBridge
     */
    static async validateLLMIntegration(components: TierOneComponents): Promise<boolean> {
        try {
            // Test if the conversation bridge can perform basic reasoning
            const testResult = await components.strategyEngine.analyzeSituation({
                goal: "Test LLM integration",
                observations: { testMode: true },
                knowledge: {
                    swarmId: "validation-test",
                    facts: new Map(),
                    insights: [],
                    decisions: [],
                },
                progress: {
                    tasksCompleted: 0,
                    tasksTotal: 1,
                    currentPhase: "validation",
                    milestones: [],
                },
            });

            return testResult && testResult.length > 0;
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
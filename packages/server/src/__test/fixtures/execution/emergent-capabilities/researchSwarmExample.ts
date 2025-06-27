/**
 * Research Swarm Example
 * 
 * Demonstrates how a research capability emerges from agent collaboration.
 * The swarm starts with basic information gathering but evolves to develop:
 * - Cross-domain synthesis
 * - Novel insight generation
 * - Predictive analysis
 * 
 * This is NOT hard-coded - it emerges from agent interactions!
 */

import type { SwarmFixture } from "../types.js";
import { swarmFactory } from "../executionFactories.js";
import {
    EMERGENCE_PATTERNS,
    EVOLUTION_TRIGGERS,
    createMeasurableEmergence,
    createCollaborativeCapability,
    createCompoundGrowthCapability,
} from "../emergentCapabilityHelpers.js";

/**
 * Academic Research Swarm
 * 
 * Configuration: Simple research goal
 * Emergence: Develops interdisciplinary synthesis and novel hypothesis generation
 */
export const academicResearchSwarm: SwarmFixture = swarmFactory.createComplete({
    config: {
        goal: "Research emerging trends in quantum computing applications",
        // Note: We do NOT specify HOW to research - that emerges!
        preferredModel: "gpt-4",
        swarmTask: "Collaborative research investigation",
        swarmSubTasks: [
            {
                id: "literature-review",
                description: "Review recent publications", // Simple task
                assignedAgents: [], // Agents self-assign based on expertise
                status: "pending",
            },
            {
                id: "pattern-analysis", 
                description: "Identify patterns", // Vague - agents figure it out
                assignedAgents: [],
                status: "pending",
            },
            {
                id: "synthesis",
                description: "Synthesize findings", // No methodology specified
                assignedAgents: [],
                status: "pending",
            },
        ],
        // Minimal configuration - behaviors emerge from agent interaction
        chatLabel: "Quantum Research Team",
        maxTokens: 4000, // Allow rich discussions
    },
    emergence: createMeasurableEmergence(EMERGENCE_PATTERNS.collaborativeLearning, {
        capabilities: [
            "interdisciplinary_synthesis", // Emerges from diverse agent perspectives
            "novel_hypothesis_generation", // Emerges from knowledge combination
            "trend_prediction", // Emerges from pattern analysis
            "research_methodology_optimization", // Emerges from repeated research
        ],
        eventPatterns: [
            "research.paper.analyzed",
            "research.pattern.discovered", 
            "research.hypothesis.proposed",
            "research.validation.completed",
            "agent.knowledge.shared",
            "agent.insight.generated",
        ],
        evolutionPath: "information_gathering → pattern_recognition → synthesis → prediction",
        emergenceConditions: {
            minAgents: 3, // Need diverse perspectives
            requiredResources: [
                "research_database",
                "shared_knowledge_graph", 
                "hypothesis_registry",
                "validation_framework",
            ],
            environmentalFactors: [
                "research_domain_complexity",
                "available_literature_volume",
                "interdisciplinary_connections",
            ],
            minEvents: 20, // Takes time for patterns to emerge
            timeframe: 7200000, // 2 hours for deep research
        },
        learningMetrics: {
            performanceImprovement: "60% reduction in time to novel insights through collaborative pattern discovery",
            adaptationTime: "5 research cycles to develop efficient methodology",
            innovationRate: "1 novel research direction per 15 papers analyzed",
        },
    }),
    integration: {
        tier: "tier1",
        producedEvents: [
            "tier1.research.insight.discovered",
            "tier1.research.hypothesis.generated",
            "tier1.research.validation.requested",
            "tier1.research.methodology.improved",
        ],
        consumedEvents: [
            "tier2.research.execution.completed",
            "tier3.research.tools.result",
            "system.research.data.available",
        ],
        sharedResources: [
            "research_blackboard", // Agents share findings
            "knowledge_graph", // Collective understanding
            "hypothesis_space", // Explored possibilities
            "methodology_registry", // Learned approaches
        ],
        crossTierDependencies: {
            dependsOn: ["tier2_research_routines", "tier3_research_tools"],
            provides: ["research_coordination", "insight_generation", "hypothesis_validation"],
        },
    },
    swarmMetadata: {
        formation: "dynamic", // Agents reorganize based on discoveries
        coordinationPattern: "emergence", // No central coordinator
        expectedAgentCount: 5,
        minViableAgents: 3,
        roles: [
            { role: "literature_analyst", count: 2 },
            { role: "pattern_detector", count: 1 },
            { role: "synthesis_expert", count: 1 },
            { role: "hypothesis_generator", count: 1 },
        ],
    },
    validation: {
        emergenceTests: [
            "interdisciplinary_connection_discovery",
            "novel_hypothesis_quality",
            "prediction_accuracy_over_time",
        ],
        integrationTests: [
            "knowledge_propagation_efficiency", 
            "cross_agent_learning",
            "collective_insight_generation",
        ],
        evolutionTests: [
            "research_speed_improvement",
            "insight_quality_growth",
            "methodology_refinement",
        ],
    },
    metadata: {
        domain: "research",
        complexity: "complex",
        maintainer: "research-team",
        lastUpdated: new Date().toISOString(),
    },
});

/**
 * Market Research Swarm
 * 
 * Shows how the same basic configuration can lead to different emergent capabilities
 * based on the domain and agent interactions
 */
export const marketResearchSwarm: SwarmFixture = swarmFactory.createComplete({
    config: {
        goal: "Analyze emerging market trends in sustainable technology",
        swarmTask: "Market trend analysis",
        swarmSubTasks: [
            {
                id: "data-collection",
                description: "Gather market data",
                assignedAgents: [],
                status: "pending",
            },
            {
                id: "trend-identification",
                description: "Identify trends", 
                assignedAgents: [],
                status: "pending",
            },
            {
                id: "prediction",
                description: "Make predictions",
                assignedAgents: [],
                status: "pending",
            },
        ],
    },
    emergence: {
        capabilities: [
            "market_signal_detection", // Emerges from noise filtering
            "competitive_intelligence", // Emerges from cross-market analysis
            "demand_prediction", // Emerges from pattern correlation
            "investment_opportunity_identification", // Emerges from trend synthesis
        ],
        eventPatterns: [
            "market.data.processed",
            "market.signal.detected",
            "market.correlation.found",
            "prediction.generated",
        ],
        evolutionPath: "data_aggregation → signal_detection → predictive_modeling → strategic_insights",
        emergenceConditions: {
            minAgents: 4,
            requiredResources: ["market_feeds", "historical_data", "correlation_engine"],
            minEvents: 50, // Markets are noisy
        },
        learningMetrics: {
            performanceImprovement: "45% better signal-to-noise ratio through collaborative filtering",
            adaptationTime: "10 market cycles to stable prediction model",
            innovationRate: "1 new market indicator discovered per 20 analyses",
        },
    },
    integration: {
        tier: "tier1",
        producedEvents: [
            "tier1.market.trend.identified",
            "tier1.market.prediction.generated",
            "tier1.market.opportunity.discovered",
        ],
        consumedEvents: [
            "tier2.market.analysis.completed",
            "tier3.market.data.fetched",
        ],
        sharedResources: ["market_intelligence", "trend_database", "prediction_models"],
    },
    swarmMetadata: {
        formation: "matrix", // Cross-functional analysis
        coordinationPattern: "negotiation", // Agents debate interpretations
        expectedAgentCount: 6,
        minViableAgents: 4,
    },
});

/**
 * Demonstration of measurable capabilities
 */
export const researchCapabilities = {
    interdisciplinarySynthesis: createCollaborativeCapability(
        "interdisciplinary_synthesis",
        3, // Requires 3+ agents with different expertise
    ),
    
    insightGeneration: createCompoundGrowthCapability(
        "insight_generation_rate",
        0.1, // Start with 0.1 insights per cycle
        0.15, // 15% growth per cycle
        "insights/cycle",
    ),
    
    predictionAccuracy: {
        name: "trend_prediction_accuracy",
        metric: "prediction_correctness",
        baseline: 0.4, // Random guessing
        target: 0.85, // Expert level
        unit: "accuracy",
        description: "Improves through pattern learning and validation feedback",
        evolutionFormula: "0.4 + 0.45 * (1 - exp(-validated_predictions/20))",
        minEventsForMeasurement: 10,
    },
};

/**
 * Evolution triggers for research swarms
 */
export const researchEvolutionTriggers = {
    // When swarm gets good at pattern recognition, evolve to prediction
    patternMastery: EVOLUTION_TRIGGERS.performanceThreshold(
        "pattern_recognition_accuracy",
        0.8,
    ),
    
    // When predictions are consistently accurate, evolve to strategic planning
    predictionConfidence: {
        type: "performance" as const,
        threshold: {
            metric: "prediction_accuracy",
            operator: ">" as const,
            value: 0.75,
        },
        action: {
            targetStrategy: "routing" as const, // Route to specialized prediction models
            preserveState: true,
            migrationSteps: [
                "Identify successful prediction patterns",
                "Create specialized prediction routes",
                "Maintain general research capability",
                "Deploy hybrid approach",
            ],
        },
    },
};

/**
 * Example of how emergence is validated in tests
 */
export function validateResearchEmergence(swarm: SwarmFixture): boolean {
    // Check that capabilities aren't in the base config
    const config = swarm.config as any;
    if (config.capabilities || config.predictionModel || config.synthesisMethod) {
        console.error("Capabilities should emerge, not be configured!");
        return false;
    }
    
    // Verify emergence conditions are met
    const conditions = swarm.emergence.emergenceConditions;
    if (!conditions || conditions.minAgents! < 3) {
        console.error("Research requires multiple perspectives to emerge");
        return false;
    }
    
    // Ensure evolution path is defined
    if (!swarm.emergence.evolutionPath?.includes("→")) {
        console.error("Must define evolution stages");
        return false;
    }
    
    return true;
}
/**
 * Customer Service Evolution Example
 * 
 * Demonstrates how a routine evolves from conversational → reasoning → deterministic
 * through actual usage patterns. This is NOT predetermined - the evolution happens
 * because the routine learns optimal response patterns from customer interactions.
 * 
 * Key principle: Evolution is driven by performance metrics and usage patterns,
 * not manual optimization.
 */

import type { RoutineFixture } from "../../types.js";
import { routineFactory } from "../../executionFactories.js";
import { createMeasurableCapability } from "../../executionFactories.js";

/**
 * Stage 1: Conversational Customer Service (Initial deployment)
 * 
 * The routine starts with simple conversational interaction.
 * It doesn't know optimal patterns yet.
 */
export const customerServiceV1: RoutineFixture = routineFactory.createComplete({
    config: {
        name: "Customer Inquiry Handler",
        description: "Handle customer inquiries with natural conversation",
        // Basic configuration - doesn't specify exact steps
        steps: [
            {
                stepType: "conversational",
                description: "Understand customer need", // Vague - AI figures out how
            },
            {
                stepType: "conversational", 
                description: "Research solution", // No specific method given
            },
            {
                stepType: "conversational",
                description: "Provide response", // Format emerges from practice
            },
        ],
    },
    emergence: {
        capabilities: [
            "natural_language_understanding", // Basic capability
            "empathy_expression", // Emerges from customer feedback
            "solution_retrieval", // Basic knowledge lookup
        ],
        eventPatterns: [
            "customer.inquiry.received",
            "routine.solution.attempted",
            "customer.feedback.received",
        ],
        evolutionPath: "conversational → reasoning → deterministic → routing",
        emergenceConditions: {
            minEvents: 50, // Needs customer interactions to evolve
            requiredResources: ["knowledge_base", "customer_feedback"],
            timeframe: 86400000, // 24 hours of operation
        },
        learningMetrics: {
            performanceImprovement: "Track resolution time and satisfaction",
            adaptationTime: "10 interactions to identify common patterns",
            innovationRate: "New response template per 25 interactions",
        },
        measurableCapabilities: [
            createMeasurableCapability(
                "customer_satisfaction",
                "satisfaction_score",
                6.5, // baseline
                8.5, // target
                "out_of_10",
                "Customer satisfaction rating",
            ),
            createMeasurableCapability(
                "resolution_time",
                "average_resolution_minutes",
                15, // baseline
                8, // target
                "minutes",
                "Time to resolve customer inquiry",
            ),
        ],
    },
    integration: {
        tier: "tier2",
        producedEvents: [
            "tier2.customer.inquiry.processed",
            "tier2.solution.provided",
            "tier2.pattern.learned",
        ],
        consumedEvents: [
            "tier1.customer.classified",
            "tier3.knowledge.retrieved",
        ],
        sharedResources: ["customer_context", "solution_templates"],
        crossTierDependencies: {
            dependsOn: ["tier1_customer_classification", "tier3_knowledge_tools"],
            provides: ["inquiry_processing", "customer_assistance"],
        },
    },
    evolutionStage: {
        strategy: "conversational",
        version: "1.0.0",
        metrics: {
            avgDuration: 15000, // 15 minutes initially
            avgCredits: 150, // High cost due to inefficiency
            successRate: 0.75, // Learning from mistakes
            errorRate: 0.25,
            satisfaction: 0.65, // Baseline satisfaction
        },
    },
    domain: "customer-service",
    navigator: "native",
});

/**
 * Stage 2: Reasoning Customer Service (After learning common patterns)
 * 
 * The routine has identified patterns and can reason about different inquiry types.
 * This evolution happened automatically based on performance metrics.
 */
export const customerServiceV2: RoutineFixture = routineFactory.createComplete({
    config: {
        ...customerServiceV1.config,
        // Configuration evolves to include learned patterns
        steps: [
            {
                stepType: "reasoning",
                description: "Classify inquiry type using learned patterns",
                // Now includes classification logic that emerged
            },
            {
                stepType: "reasoning",
                description: "Apply domain-specific reasoning for solution",
                // Reasoning patterns learned from successful resolutions  
            },
            {
                stepType: "conversational",
                description: "Deliver personalized response",
                // Still conversational for human touch
            },
        ],
    },
    emergence: {
        capabilities: [
            ...customerServiceV1.emergence.capabilities,
            // New capabilities that emerged:
            "inquiry_classification", // Emerged from pattern recognition
            "contextual_reasoning", // Emerged from complex case handling
            "multi_step_problem_solving", // Emerged from difficult cases
            "emotional_intelligence", // Emerged from feedback patterns
        ],
        measurableCapabilities: [
            createMeasurableCapability(
                "customer_satisfaction",
                "satisfaction_score", 
                8.5, // improved from V1
                9.2, // new target
                "out_of_10",
                "Customer satisfaction with reasoning approach",
            ),
            createMeasurableCapability(
                "resolution_time",
                "average_resolution_minutes",
                8, // improved from V1
                5, // new target
                "minutes", 
                "Faster through pattern recognition",
            ),
            createMeasurableCapability(
                "first_contact_resolution",
                "resolved_without_escalation",
                60, // baseline
                85, // target
                "%",
                "Percentage resolved on first contact",
            ),
        ],
    },
    evolutionStage: {
        strategy: "reasoning",
        version: "2.0.0",
        metrics: {
            avgDuration: 8000, // 47% faster than V1
            avgCredits: 100, // 33% cheaper
            successRate: 0.88, // Much more reliable
            errorRate: 0.12,
            satisfaction: 0.85, // Higher satisfaction
        },
        previousVersion: "1.0.0",
        improvements: [
            "Added inquiry pattern classification",
            "Implemented contextual reasoning chains",
            "Developed emotional response templates",
        ],
    },
});

/**
 * Stage 3: Deterministic Customer Service (Fully optimized patterns)
 * 
 * The routine has learned optimal decision trees for most common cases.
 * It can handle 80% of inquiries deterministically.
 */
export const customerServiceV3: RoutineFixture = routineFactory.createComplete({
    config: {
        ...customerServiceV2.config,
        steps: [
            {
                stepType: "deterministic",
                description: "Instant classification using decision tree",
                // Optimal patterns crystallized into fast logic
            },
            {
                stepType: "deterministic", 
                description: "Execute optimal solution path",
                // Most efficient solution sequences determined
            },
            {
                stepType: "deterministic",
                description: "Generate response from proven templates",
                // Best response formats identified
            },
        ],
    },
    emergence: {
        capabilities: [
            ...customerServiceV2.emergence.capabilities,
            // Advanced capabilities that emerged:
            "instant_pattern_matching", // Optimized classification
            "solution_path_optimization", // Best sequence selection
            "predictive_customer_needs", // Anticipates follow-up questions
            "automated_quality_assurance", // Self-validates responses
        ],
        measurableCapabilities: [
            createMeasurableCapability(
                "customer_satisfaction",
                "satisfaction_score",
                9.2, // maintained high quality
                9.5, // slight improvement
                "out_of_10",
                "Consistent quality through deterministic patterns",
            ),
            createMeasurableCapability(
                "resolution_time", 
                "average_resolution_minutes",
                5, // from V2
                2, // very fast
                "minutes",
                "Near-instant for common patterns",
            ),
            createMeasurableCapability(
                "cost_efficiency",
                "credits_per_resolution",
                100, // from V2
                25, // 75% cost reduction
                "credits",
                "Highly optimized resource usage",
            ),
        ],
    },
    evolutionStage: {
        strategy: "deterministic",
        version: "3.0.0", 
        metrics: {
            avgDuration: 2000, // 87% faster than V1
            avgCredits: 25, // 83% cheaper than V1
            successRate: 0.96, // Highly reliable
            errorRate: 0.04,
            satisfaction: 0.95, // Excellent satisfaction
        },
        previousVersion: "2.0.0",
        improvements: [
            "Crystallized optimal decision trees",
            "Implemented instant pattern matching",
            "Added predictive response generation",
        ],
    },
});

/**
 * Stage 4: Routing Customer Service (Handles complex cases intelligently)
 * 
 * The routine can handle 95% of cases deterministically, but intelligently
 * routes complex cases to appropriate specialists or reasoning modes.
 */
export const customerServiceV4: RoutineFixture = routineFactory.createComplete({
    config: {
        ...customerServiceV3.config,
        steps: [
            {
                stepType: "routing",
                description: "Intelligent triage and routing",
                // Routes based on complexity analysis
            },
            {
                stepType: "deterministic",
                description: "Handle common cases instantly",
                // Fast path for known patterns
            },
            {
                stepType: "reasoning",
                description: "Deep analysis for complex cases", 
                // Escalation path for novel situations
            },
        ],
    },
    emergence: {
        capabilities: [
            ...customerServiceV3.emergence.capabilities,
            // Ultimate capabilities:
            "complexity_assessment", // Judges case difficulty
            "specialist_routing", // Routes to best handler
            "hybrid_strategy_selection", // Chooses optimal approach
            "continuous_learning", // Updates patterns from new cases
        ],
        measurableCapabilities: [
            createMeasurableCapability(
                "routing_accuracy",
                "correct_escalation_rate",
                0, // new capability
                95, // high accuracy
                "%",
                "Percentage of cases routed to optimal handler",
            ),
            createMeasurableCapability(
                "system_efficiency",
                "total_cost_per_customer",
                25, // from V3
                15, // optimal resource allocation
                "credits",
                "Overall cost including routing overhead",
            ),
        ],
    },
    evolutionStage: {
        strategy: "routing",
        version: "4.0.0",
        metrics: {
            avgDuration: 1500, // Slight improvement through routing
            avgCredits: 15, // Better resource allocation
            successRate: 0.99, // Near-perfect through routing
            errorRate: 0.01,
            satisfaction: 0.98, // Excellent through specialization
        },
        previousVersion: "3.0.0",
        improvements: [
            "Added intelligent case routing",
            "Implemented hybrid strategy selection",
            "Added continuous learning from edge cases",
        ],
    },
});

/**
 * Evolution pathway showing the complete journey
 * This demonstrates how emergent capabilities compound over time
 */
export const customerServiceEvolution = routineFactory.createEvolutionPath(4);

/**
 * Evolution trigger examples showing what causes transitions
 */
export const customerServiceEvolutionTriggers = [
    {
        fromVersion: "1.0.0",
        toVersion: "2.0.0",
        trigger: {
            type: "performance_threshold" as const,
            threshold: 100, // 100 customer interactions
            metric: "interaction_count",
            condition: "satisfaction_score > 7.0 AND pattern_recognition_accuracy > 0.8",
        },
        description: "Evolves to reasoning when patterns are identified",
    },
    {
        fromVersion: "2.0.0", 
        toVersion: "3.0.0",
        trigger: {
            type: "performance_threshold" as const,
            threshold: 0.9, // 90% success rate
            metric: "success_rate",
            condition: "common_case_coverage > 0.8 AND response_time_stability > 0.9",
        },
        description: "Evolves to deterministic when patterns are stable",
    },
    {
        fromVersion: "3.0.0",
        toVersion: "4.0.0", 
        trigger: {
            type: "usage_count" as const,
            threshold: 1000, // 1000 cases handled
            metric: "total_cases",
            condition: "edge_case_frequency > 0.1 AND specialist_availability = true",
        },
        description: "Evolves to routing when edge cases need specialists",
    },
];

/**
 * Performance comparison showing compound improvement
 */
export const evolutionMetricsComparison = {
    v1: customerServiceV1.evolutionStage?.metrics,
    v2: customerServiceV2.evolutionStage?.metrics,
    v3: customerServiceV3.evolutionStage?.metrics,
    v4: customerServiceV4.evolutionStage?.metrics,
    
    // Compound improvements
    totalSpeedupV4: (customerServiceV1.evolutionStage?.metrics.avgDuration || 0) / 
                    (customerServiceV4.evolutionStage?.metrics.avgDuration || 1), // 10x faster
    totalCostReductionV4: 1 - ((customerServiceV4.evolutionStage?.metrics.avgCredits || 0) / 
                              (customerServiceV1.evolutionStage?.metrics.avgCredits || 1)), // 90% cheaper
    totalQualityImprovement: (customerServiceV4.evolutionStage?.metrics.satisfaction || 0) - 
                            (customerServiceV1.evolutionStage?.metrics.satisfaction || 0), // +50% satisfaction
};
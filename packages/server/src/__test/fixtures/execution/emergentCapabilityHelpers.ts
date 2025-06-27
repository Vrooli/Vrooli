/**
 * Enhanced Emergent Capability Helpers
 * 
 * Provides helper functions for creating fixtures that demonstrate emergent capabilities.
 * These helpers ensure that capabilities are defined as emerging from configuration,
 * not as hard-coded behavior.
 */

import type {
    EmergenceDefinition,
    EnhancedEmergenceDefinition,
    MeasurableCapability,
    EmergencePattern,
    EvolutionTrigger,
} from "./types.js";

/**
 * Common emergence patterns that can be reused across fixtures
 */
export const EMERGENCE_PATTERNS = {
    /**
     * Collaborative Learning: Capabilities emerge from agent collaboration
     */
    collaborativeLearning: {
        name: "collaborative_learning",
        requires: {
            minAgents: 2,
            minEvents: 10,
            eventTypes: ["agent.knowledge.shared", "agent.insight.generated"],
            resources: ["shared_memory", "communication_channel"],
            timeWindow: 3600000, // 1 hour
        },
        produces: {
            capabilities: ["knowledge_synthesis", "collective_intelligence", "pattern_recognition"],
            behaviors: ["self_organization", "role_specialization", "knowledge_sharing"],
            optimizations: ["communication_efficiency", "task_distribution", "learning_acceleration"],
        },
        metrics: [
            {
                name: "collective_intelligence_score",
                metric: "group_performance_vs_individual",
                baseline: 1.0,
                target: 2.5,
                unit: "performance_ratio",
                description: "How much better the group performs vs individuals",
            },
            {
                name: "knowledge_synthesis_rate",
                metric: "new_insights_per_interaction",
                baseline: 0.1,
                target: 0.8,
                unit: "insights/interaction",
                description: "Rate of novel insight generation",
            },
        ],
    },

    /**
     * Performance Optimization: Capabilities emerge from usage patterns
     */
    performanceOptimization: {
        name: "performance_optimization",
        requires: {
            minEvents: 50,
            eventTypes: ["execution.completed", "performance.measured"],
            resources: ["execution_history", "performance_metrics"],
            timeWindow: 86400000, // 24 hours
        },
        produces: {
            capabilities: ["pattern_optimization", "resource_efficiency", "predictive_scaling"],
            behaviors: ["adaptive_execution", "resource_management", "load_balancing"],
            optimizations: ["execution_speed", "resource_usage", "cost_efficiency"],
        },
        metrics: [
            {
                name: "execution_efficiency",
                metric: "time_to_completion",
                baseline: 100,
                target: 30,
                unit: "seconds",
                description: "Time to complete standard tasks",
            },
            {
                name: "resource_utilization",
                metric: "compute_efficiency",
                baseline: 60,
                target: 90,
                unit: "%",
                description: "Percentage of allocated resources actually used",
            },
        ],
    },

    /**
     * Security Awareness: Security capabilities emerge from threat exposure
     */
    securityAwareness: {
        name: "security_awareness",
        requires: {
            minEvents: 20,
            eventTypes: ["security.scan.completed", "threat.detected", "security.pattern.learned"],
            resources: ["vulnerability_database", "threat_intelligence", "security_patterns"],
            timeWindow: 7200000, // 2 hours
        },
        produces: {
            capabilities: ["threat_detection", "vulnerability_assessment", "security_pattern_recognition"],
            behaviors: ["proactive_scanning", "risk_assessment", "security_hardening"],
            optimizations: ["false_positive_reduction", "detection_speed", "coverage_improvement"],
        },
        metrics: [
            {
                name: "threat_detection_accuracy",
                metric: "true_positive_rate",
                baseline: 70,
                target: 95,
                unit: "%",
                description: "Percentage of actual threats correctly identified",
            },
            {
                name: "false_positive_rate",
                metric: "false_alarm_percentage",
                baseline: 30,
                target: 5,
                unit: "%",
                description: "Percentage of false security alerts",
            },
        ],
    },

    /**
     * Quality Assurance: Quality capabilities emerge from feedback loops
     */
    qualityAssurance: {
        name: "quality_assurance",
        requires: {
            minEvents: 30,
            eventTypes: ["quality.check.completed", "feedback.received", "defect.detected"],
            resources: ["quality_standards", "feedback_history", "defect_patterns"],
            timeWindow: 14400000, // 4 hours
        },
        produces: {
            capabilities: ["defect_prediction", "quality_assessment", "process_improvement"],
            behaviors: ["continuous_monitoring", "feedback_integration", "standard_enforcement"],
            optimizations: ["defect_reduction", "quality_consistency", "review_efficiency"],
        },
        metrics: [
            {
                name: "defect_prediction_accuracy",
                metric: "predicted_defects_found",
                baseline: 40,
                target: 85,
                unit: "%",
                description: "Percentage of predicted defects actually found",
            },
            {
                name: "quality_improvement_rate",
                metric: "quality_score_trend",
                baseline: 0,
                target: 15,
                unit: "score_increase_per_cycle",
                description: "Rate of quality score improvement",
            },
        ],
    },
} as const;

/**
 * Common evolution triggers that drive capability development
 */
export const EVOLUTION_TRIGGERS = {
    usageThreshold: (threshold: number, metric: string): EvolutionTrigger => ({
        type: "usage_count",
        threshold,
        metric,
        conditions: [`${metric} >= ${threshold}`, "success_rate > 0.8"],
    }),

    performanceThreshold: (threshold: number, metric: string): EvolutionTrigger => ({
        type: "performance_threshold",
        threshold,
        metric,
        conditions: [`${metric} >= ${threshold}`, "error_rate < 0.1"],
    }),

    errorRateThreshold: (threshold: number): EvolutionTrigger => ({
        type: "error_rate",
        threshold,
        metric: "error_rate",
        conditions: [`error_rate <= ${threshold}`, "minimum_usage_met"],
    }),

    timeBasedEvolution: (hours: number): EvolutionTrigger => ({
        type: "time_based",
        threshold: hours * 3600000, // Convert to milliseconds
        metric: "time_elapsed",
        conditions: ["sufficient_data_collected", "stable_performance"],
    }),
} as const;

/**
 * Create measurable emergence definition with concrete metrics
 */
export function createMeasurableEmergence(
    pattern: typeof EMERGENCE_PATTERNS[keyof typeof EMERGENCE_PATTERNS],
    overrides: Partial<EmergenceDefinition> = {},
): EnhancedEmergenceDefinition {
    return {
        capabilities: pattern.produces.capabilities,
        eventPatterns: pattern.requires.eventTypes,
        evolutionPath: "basic → adaptive → optimized → intelligent",
        emergenceConditions: {
            minAgents: pattern.requires.minAgents,
            minEvents: pattern.requires.minEvents,
            requiredResources: pattern.requires.resources,
            timeframe: pattern.requires.timeWindow,
            environmentalFactors: ["system_stability", "resource_availability"],
        },
        learningMetrics: {
            performanceImprovement: "Compound improvement through pattern learning",
            adaptationTime: "Progressive reduction in time to optimal performance",
            innovationRate: "Accelerating discovery of new patterns and optimizations",
        },
        ...overrides,
        measurableCapabilities: pattern.metrics,
        emergenceTests: [
            {
                setup: `Configure system with ${pattern.name} requirements`,
                trigger: `Generate ${pattern.requires.minEvents} events of required types`,
                expectedOutcome: `Observe emergence of ${pattern.produces.capabilities.join(", ")}`,
                measurementMethod: `Validate metrics show improvement from baseline to target`,
            },
        ],
    };
}

/**
 * Create collaborative capability that emerges from agent interaction
 */
export function createCollaborativeCapability(
    name: string,
    requiredAgents: number,
    baseCapabilities: string[],
    emergentCapabilities: string[],
): MeasurableCapability {
    return {
        name,
        metric: "collaborative_performance_multiplier",
        baseline: 1.0, // Individual performance
        target: 1.5 + (requiredAgents * 0.3), // Performance multiplier based on collaboration
        unit: "performance_ratio",
        description: `Performance improvement through ${requiredAgents}-agent collaboration`,
        evolutionFormula: `baseline * (1 + collaboration_coefficient * agent_count * experience_factor)`,
        minEventsForMeasurement: requiredAgents * 5, // Need multiple interactions per agent
    };
}

/**
 * Create compound growth capability that improves exponentially
 */
export function createCompoundGrowthCapability(
    name: string,
    initialValue: number,
    growthRate: number,
    targetCycles: number,
): MeasurableCapability {
    const finalValue = initialValue * Math.pow(1 + growthRate, targetCycles);
    
    return {
        name,
        metric: "compound_improvement_value",
        baseline: initialValue,
        target: finalValue,
        unit: "improvement_units",
        description: `Compound growth at ${growthRate * 100}% per cycle over ${targetCycles} cycles`,
        evolutionFormula: `baseline * (1 + ${growthRate}) ^ cycle_count`,
        minEventsForMeasurement: targetCycles * 2,
    };
}

/**
 * Create evolution pathway showing capability development stages
 */
export function createEvolutionPathway(
    stages: string[],
    improvementFactors: number[],
    timePerStage: number,
): {
    stages: Array<{
        name: string;
        improvementFactor: number;
        timeToReach: number;
        description: string;
    }>;
    totalImprovement: number;
    totalTime: number;
} {
    if (stages.length !== improvementFactors.length) {
        throw new Error("Stages and improvement factors must have same length");
    }

    const evolutionStages = stages.map((stage, index) => ({
        name: stage,
        improvementFactor: improvementFactors[index],
        timeToReach: (index + 1) * timePerStage,
        description: `Stage ${index + 1}: ${improvementFactors[index]}x improvement`,
    }));

    const totalImprovement = improvementFactors.reduce((total, factor) => total * factor, 1);
    const totalTime = stages.length * timePerStage;

    return {
        stages: evolutionStages,
        totalImprovement,
        totalTime,
    };
}

/**
 * Create domain-specific emergence patterns
 */
export function createDomainEmergence(
    domain: string,
    domainCapabilities: string[],
    domainResources: string[],
    domainEvents: string[],
): EmergenceDefinition {
    return {
        capabilities: domainCapabilities,
        eventPatterns: domainEvents,
        evolutionPath: `${domain}_basic → ${domain}_intermediate → ${domain}_advanced → ${domain}_expert`,
        emergenceConditions: {
            minEvents: 25,
            requiredResources: domainResources,
            environmentalFactors: [`${domain}_complexity`, `${domain}_data_availability`],
            timeframe: 7200000, // 2 hours for domain specialization
        },
        learningMetrics: {
            performanceImprovement: `${domain}-specific optimization through specialized learning`,
            adaptationTime: `Rapid adaptation to ${domain} patterns and requirements`,
            innovationRate: `Discovery of novel ${domain} approaches and optimizations`,
        },
    };
}

/**
 * Validate that emergence definition follows emergent architecture principles
 */
export function validateEmergentPrinciples(emergence: EmergenceDefinition): {
    isValid: boolean;
    violations: string[];
    warnings: string[];
} {
    const violations: string[] = [];
    const warnings: string[] = [];

    // Principle 1: Capabilities must be specific and measurable
    emergence.capabilities.forEach(capability => {
        if (capability.length < 5) {
            violations.push(`Capability '${capability}' is too vague`);
        }
        if (!capability.includes("_")) {
            warnings.push(`Capability '${capability}' should use snake_case convention`);
        }
        if (capability.includes("basic") || capability.includes("simple")) {
            warnings.push(`Capability '${capability}' sounds like hard-coded basic functionality`);
        }
    });

    // Principle 2: Must define emergence conditions
    if (!emergence.emergenceConditions) {
        violations.push("Must define emergence conditions");
    } else {
        if (!emergence.emergenceConditions.minEvents && !emergence.emergenceConditions.minAgents) {
            violations.push("Must specify minimum events or agents for emergence");
        }
        if (!emergence.emergenceConditions.requiredResources || emergence.emergenceConditions.requiredResources.length === 0) {
            warnings.push("Should specify required resources for emergence");
        }
    }

    // Principle 3: Evolution path should show progression
    if (!emergence.evolutionPath) {
        warnings.push("Should define evolution pathway");
    } else if (!emergence.evolutionPath.includes("→") && !emergence.evolutionPath.includes("->")) {
        warnings.push("Evolution path should show clear progression with arrows");
    }

    // Principle 4: Learning metrics should be concrete
    if (!emergence.learningMetrics) {
        warnings.push("Should define learning metrics");
    } else {
        const metrics = emergence.learningMetrics;
        if (!metrics.performanceImprovement.match(/\d+%/) && !metrics.performanceImprovement.includes("reduction")) {
            warnings.push("Performance improvement should include quantifiable benefits");
        }
    }

    return {
        isValid: violations.length === 0,
        violations,
        warnings,
    };
}

/**
 * Create emergence validation test
 */
export function createEmergenceValidationTest(
    capability: string,
    testScenario: {
        inputConditions: string[];
        expectedBehavior: string;
        measurementCriteria: string[];
    },
): {
    setup: string;
    trigger: string;
    expectedOutcome: string;
    measurementMethod: string;
} {
    return {
        setup: `Configure environment with: ${testScenario.inputConditions.join(", ")}`,
        trigger: `Execute scenario to test ${capability} emergence`,
        expectedOutcome: testScenario.expectedBehavior,
        measurementMethod: `Validate: ${testScenario.measurementCriteria.join(" AND ")}`,
    };
}

/**
 * Create performance benchmark that validates emergent improvement
 */
export function createEmergencePerformanceBenchmark(
    capability: string,
    baselineMetric: { value: number; unit: string },
    targetMetric: { value: number; unit: string },
    measurementMethod: string,
): MeasurableCapability {
    const improvementRatio = baselineMetric.value / targetMetric.value;
    
    return {
        name: `${capability}_performance`,
        metric: measurementMethod,
        baseline: baselineMetric.value,
        target: targetMetric.value,
        unit: baselineMetric.unit,
        description: `${capability} performance improvement: ${improvementRatio.toFixed(1)}x better`,
        evolutionFormula: `exponential_improvement(baseline, learning_rate, experience_cycles)`,
        minEventsForMeasurement: Math.max(10, Math.ceil(improvementRatio)),
    };
}

/**
 * Helper to create error recovery emergence pattern
 */
export function createErrorRecoveryEmergence(
    errorTypes: string[],
    recoveryStrategies: string[],
): EmergenceDefinition {
    return {
        capabilities: [
            "error_pattern_recognition",
            "recovery_strategy_selection",
            "failure_prediction",
            "resilience_optimization",
        ],
        eventPatterns: [
            ...errorTypes.map(type => `error.${type}.occurred`),
            ...recoveryStrategies.map(strategy => `recovery.${strategy}.attempted`),
            "recovery.success",
            "recovery.failure",
        ],
        evolutionPath: "reactive_recovery → proactive_prevention → predictive_resilience",
        emergenceConditions: {
            minEvents: errorTypes.length * 3, // Need examples of each error type
            requiredResources: ["error_history", "recovery_patterns", "system_health_metrics"],
            environmentalFactors: ["system_stability", "error_frequency", "recovery_success_rate"],
            timeframe: 21600000, // 6 hours to learn patterns
        },
        learningMetrics: {
            performanceImprovement: "90% reduction in system downtime through predictive recovery",
            adaptationTime: "5 error cycles to identify optimal recovery strategies",
            innovationRate: "1 new recovery pattern per 20 error occurrences",
        },
    };
}

/**
 * Helper to create resource optimization emergence pattern
 */
export function createResourceOptimizationEmergence(
    resourceTypes: string[],
): EmergenceDefinition {
    return {
        capabilities: [
            "resource_usage_prediction",
            "allocation_optimization",
            "waste_detection",
            "capacity_planning",
        ],
        eventPatterns: [
            ...resourceTypes.map(type => `resource.${type}.allocated`),
            ...resourceTypes.map(type => `resource.${type}.released`),
            "resource.shortage.detected",
            "resource.waste.identified",
        ],
        evolutionPath: "reactive_allocation → predictive_allocation → optimal_utilization",
        emergenceConditions: {
            minEvents: resourceTypes.length * 10, // Need usage patterns
            requiredResources: ["usage_history", "capacity_metrics", "cost_data"],
            environmentalFactors: ["demand_variability", "resource_availability", "cost_constraints"],
            timeframe: 43200000, // 12 hours to learn usage patterns
        },
        learningMetrics: {
            performanceImprovement: "60% cost reduction through optimal resource allocation",
            adaptationTime: "2 hours to adapt to new demand patterns",
            innovationRate: "1 optimization discovery per 100 allocation decisions",
        },
    };
}

export default {
    EMERGENCE_PATTERNS,
    EVOLUTION_TRIGGERS,
    createMeasurableEmergence,
    createCollaborativeCapability,
    createCompoundGrowthCapability,
    createEvolutionPathway,
    createDomainEmergence,
    validateEmergentPrinciples,
    createEmergenceValidationTest,
    createEmergencePerformanceBenchmark,
    createErrorRecoveryEmergence,
    createResourceOptimizationEmergence,
};
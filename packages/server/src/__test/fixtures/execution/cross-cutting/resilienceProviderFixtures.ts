/**
 * Resilience Provider Test Fixtures
 * 
 * Comprehensive test fixtures for the ResilienceProvider component, providing
 * validated configurations for testing fault tolerance, error recovery, and
 * resilience patterns across all three tiers.
 */

import type {
    BaseConfigObject,
} from "@vrooli/shared";
import {
    chatConfigFixtures,
} from "@vrooli/shared";
import type {
    ExecutionFixture,
} from "../types.js";
import { BaseExecutionFixtureFactory } from "../executionFactories.js";

/**
 * Resilience-specific fixture interface
 */
export interface ResilienceFixture<TConfig extends BaseConfigObject = BaseConfigObject> extends ExecutionFixture<TConfig> {
    resilience: {
        /** Failure scenarios this fixture handles */
        failureScenarios: Array<{
            type: "network" | "resource" | "timeout" | "validation" | "permission" | "cascade";
            probability: number;
            impact: "low" | "medium" | "high" | "critical";
            recoveryStrategy: "retry" | "fallback" | "circuit-break" | "escalate";
        }>;
        /** Recovery patterns implemented */
        recoveryPatterns: string[];
        /** Fault tolerance mechanisms */
        faultTolerance: {
            redundancy: number;
            failoverTime: number;
            dataConsistency: "eventual" | "strong" | "weak";
        };
        /** Resilience metrics */
        metrics: {
            mtbf: number; // Mean time between failures
            mttr: number; // Mean time to recovery
            availability: number; // Target availability percentage
            rpo: number; // Recovery point objective (seconds)
            rto: number; // Recovery time objective (seconds)
        };
    };
}

/**
 * Factory for creating resilience test fixtures
 */
export class ResilienceFixtureFactory extends BaseExecutionFixtureFactory<BaseConfigObject> {
    protected ConfigClass = chatConfigFixtures.ChatConfig;
    protected tier = "cross-tier" as const;

    createMinimal(overrides?: Partial<ResilienceFixture>): ResilienceFixture {
        return {
            config: chatConfigFixtures.minimal,
            emergence: {
                capabilities: ["basic_recovery", "error_handling"],
                evolutionPath: "reactive → proactive → self-healing",
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: ["resilience.error.detected", "resilience.recovery.started"],
                consumedEvents: ["tier*.error", "system.health"],
                ...overrides?.integration,
            },
            resilience: {
                failureScenarios: [{
                    type: "timeout",
                    probability: 0.1,
                    impact: "low",
                    recoveryStrategy: "retry",
                }],
                recoveryPatterns: ["exponential_backoff"],
                faultTolerance: {
                    redundancy: 1,
                    failoverTime: 5000,
                    dataConsistency: "eventual",
                },
                metrics: {
                    mtbf: 86400, // 24 hours
                    mttr: 300, // 5 minutes
                    availability: 99.0,
                    rpo: 60,
                    rto: 300,
                },
                ...overrides?.resilience,
            },
        };
    }

    createComplete(overrides?: Partial<ResilienceFixture>): ResilienceFixture {
        return {
            config: chatConfigFixtures.complete,
            emergence: {
                capabilities: [
                    "predictive_failure_detection",
                    "autonomous_recovery",
                    "cascade_prevention",
                    "self_healing",
                    "performance_degradation_management",
                ],
                eventPatterns: ["*.error", "*.failure", "*.timeout", "*.degraded"],
                evolutionPath: "reactive → proactive → predictive → self-healing",
                emergenceConditions: {
                    minEvents: 100,
                    requiredResources: ["monitoring", "metrics", "recovery_engine"],
                    timeframe: 3600000, // 1 hour
                },
                learningMetrics: {
                    performanceImprovement: "50% reduction in MTTR",
                    adaptationTime: "10 failure cycles to optimal recovery",
                    innovationRate: "2 new recovery patterns per 100 failures",
                },
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: [
                    "resilience.failure.predicted",
                    "resilience.recovery.initiated",
                    "resilience.recovery.completed",
                    "resilience.circuit.opened",
                    "resilience.circuit.closed",
                    "resilience.failover.triggered",
                ],
                consumedEvents: [
                    "tier1.swarm.unhealthy",
                    "tier2.routine.failed",
                    "tier3.execution.error",
                    "system.resource.exhausted",
                    "network.connection.lost",
                ],
                sharedResources: ["health_metrics", "recovery_state", "circuit_breakers"],
                crossTierDependencies: {
                    dependsOn: ["monitoring", "metrics", "event_bus"],
                    provides: ["fault_tolerance", "recovery_management", "health_monitoring"],
                },
                mcpTools: ["health_check", "recovery_coordinator", "circuit_breaker"],
                ...overrides?.integration,
            },
            resilience: {
                failureScenarios: [
                    {
                        type: "network",
                        probability: 0.05,
                        impact: "high",
                        recoveryStrategy: "circuit-break",
                    },
                    {
                        type: "resource",
                        probability: 0.02,
                        impact: "critical",
                        recoveryStrategy: "fallback",
                    },
                    {
                        type: "timeout",
                        probability: 0.1,
                        impact: "medium",
                        recoveryStrategy: "retry",
                    },
                    {
                        type: "cascade",
                        probability: 0.01,
                        impact: "critical",
                        recoveryStrategy: "escalate",
                    },
                ],
                recoveryPatterns: [
                    "exponential_backoff",
                    "circuit_breaker",
                    "bulkhead_isolation",
                    "timeout_with_fallback",
                    "retry_with_jitter",
                    "adaptive_concurrency",
                ],
                faultTolerance: {
                    redundancy: 3,
                    failoverTime: 1000,
                    dataConsistency: "strong",
                },
                metrics: {
                    mtbf: 604800, // 7 days
                    mttr: 60, // 1 minute
                    availability: 99.99,
                    rpo: 5,
                    rto: 60,
                },
                ...overrides?.resilience,
            },
            validation: {
                emergenceTests: [
                    "predictive_failure_detection",
                    "cascade_prevention",
                    "self_healing_verification",
                ],
                integrationTests: [
                    "cross_tier_failure_propagation",
                    "recovery_coordination",
                    "circuit_breaker_behavior",
                ],
                evolutionTests: [
                    "mttr_improvement",
                    "failure_prediction_accuracy",
                    "recovery_pattern_optimization",
                ],
                ...overrides?.validation,
            },
            metadata: {
                domain: "infrastructure",
                complexity: "complex",
                maintainer: "platform-team",
                lastUpdated: new Date().toISOString(),
                ...overrides?.metadata,
            },
        };
    }

    createWithDefaults(overrides?: Partial<ResilienceFixture>): ResilienceFixture {
        return this.createMinimal({
            emergence: {
                capabilities: ["standard_recovery", "metric_tracking"],
                evolutionPath: "manual → automated → intelligent",
            },
            resilience: {
                failureScenarios: [
                    {
                        type: "timeout",
                        probability: 0.05,
                        impact: "medium",
                        recoveryStrategy: "retry",
                    },
                    {
                        type: "validation",
                        probability: 0.1,
                        impact: "low",
                        recoveryStrategy: "fallback",
                    },
                ],
                recoveryPatterns: ["simple_retry", "fixed_delay_backoff"],
                faultTolerance: {
                    redundancy: 2,
                    failoverTime: 3000,
                    dataConsistency: "eventual",
                },
                metrics: {
                    mtbf: 172800, // 48 hours
                    mttr: 180, // 3 minutes
                    availability: 99.5,
                    rpo: 30,
                    rto: 180,
                },
            },
            ...overrides,
        });
    }

    createVariant(variant: string, overrides?: Partial<ResilienceFixture>): ResilienceFixture {
        const variants: Record<string, () => ResilienceFixture> = {
            highAvailability: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "zero_downtime_recovery",
                        "predictive_scaling",
                        "autonomous_failover",
                    ],
                },
                resilience: {
                    failureScenarios: [
                        {
                            type: "network",
                            probability: 0.001,
                            impact: "critical",
                            recoveryStrategy: "fallback",
                        },
                    ],
                    recoveryPatterns: [
                        "active_active_failover",
                        "geographic_redundancy",
                        "data_replication",
                    ],
                    faultTolerance: {
                        redundancy: 5,
                        failoverTime: 100,
                        dataConsistency: "strong",
                    },
                    metrics: {
                        mtbf: 2592000, // 30 days
                        mttr: 10, // 10 seconds
                        availability: 99.999,
                        rpo: 1,
                        rto: 10,
                    },
                },
                ...overrides,
            }),

            chaosEngineering: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "chaos_injection",
                        "weakness_detection",
                        "resilience_verification",
                    ],
                },
                resilience: {
                    failureScenarios: [
                        {
                            type: "network",
                            probability: 0.3,
                            impact: "high",
                            recoveryStrategy: "retry",
                        },
                        {
                            type: "resource",
                            probability: 0.2,
                            impact: "critical",
                            recoveryStrategy: "circuit-break",
                        },
                        {
                            type: "cascade",
                            probability: 0.1,
                            impact: "critical",
                            recoveryStrategy: "escalate",
                        },
                    ],
                    recoveryPatterns: [
                        "chaos_monkey",
                        "latency_injection",
                        "resource_exhaustion_simulation",
                    ],
                    faultTolerance: {
                        redundancy: 2,
                        failoverTime: 5000,
                        dataConsistency: "eventual",
                    },
                    metrics: {
                        mtbf: 3600, // 1 hour (intentionally low for testing)
                        mttr: 300, // 5 minutes
                        availability: 95.0, // Lower target for chaos testing
                        rpo: 60,
                        rto: 300,
                    },
                },
                ...overrides,
            }),

            costOptimized: () => this.createMinimal({
                emergence: {
                    capabilities: [
                        "cost_aware_recovery",
                        "resource_optimization",
                        "degraded_operation",
                    ],
                },
                resilience: {
                    failureScenarios: [
                        {
                            type: "resource",
                            probability: 0.1,
                            impact: "medium",
                            recoveryStrategy: "fallback",
                        },
                    ],
                    recoveryPatterns: [
                        "graceful_degradation",
                        "priority_shedding",
                        "resource_pooling",
                    ],
                    faultTolerance: {
                        redundancy: 1,
                        failoverTime: 10000,
                        dataConsistency: "weak",
                    },
                    metrics: {
                        mtbf: 43200, // 12 hours
                        mttr: 600, // 10 minutes
                        availability: 98.0,
                        rpo: 300,
                        rto: 600,
                    },
                },
                ...overrides,
            }),
        };

        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown resilience variant: ${variant}`);
        }

        return factory();
    }

    /**
     * Create fixtures for different failure scenarios
     */
    createFailureScenario(
        scenario: "network_partition" | "resource_exhaustion" | "cascade_failure" | "slow_degradation",
        overrides?: Partial<ResilienceFixture>,
    ): ResilienceFixture {
        const scenarios = {
            network_partition: () => this.createComplete({
                resilience: {
                    failureScenarios: [{
                        type: "network",
                        probability: 1.0, // Guaranteed failure for testing
                        impact: "critical",
                        recoveryStrategy: "circuit-break",
                    }],
                    recoveryPatterns: ["network_partition_detection", "split_brain_prevention"],
                    faultTolerance: {
                        redundancy: 3,
                        failoverTime: 1000,
                        dataConsistency: "eventual",
                    },
                    metrics: {
                        mtbf: 0, // Immediate failure
                        mttr: 60,
                        availability: 0,
                        rpo: 60,
                        rto: 120,
                    },
                },
                ...overrides,
            }),

            resource_exhaustion: () => this.createComplete({
                resilience: {
                    failureScenarios: [{
                        type: "resource",
                        probability: 1.0,
                        impact: "critical",
                        recoveryStrategy: "fallback",
                    }],
                    recoveryPatterns: ["resource_limiting", "request_throttling"],
                    faultTolerance: {
                        redundancy: 1,
                        failoverTime: 5000,
                        dataConsistency: "weak",
                    },
                    metrics: {
                        mtbf: 0,
                        mttr: 300,
                        availability: 0,
                        rpo: 300,
                        rto: 600,
                    },
                },
                ...overrides,
            }),

            cascade_failure: () => this.createComplete({
                resilience: {
                    failureScenarios: [{
                        type: "cascade",
                        probability: 1.0,
                        impact: "critical",
                        recoveryStrategy: "escalate",
                    }],
                    recoveryPatterns: ["cascade_detection", "dependency_isolation"],
                    faultTolerance: {
                        redundancy: 3,
                        failoverTime: 500,
                        dataConsistency: "strong",
                    },
                    metrics: {
                        mtbf: 0,
                        mttr: 120,
                        availability: 0,
                        rpo: 30,
                        rto: 180,
                    },
                },
                ...overrides,
            }),

            slow_degradation: () => this.createComplete({
                resilience: {
                    failureScenarios: [{
                        type: "timeout",
                        probability: 0.5,
                        impact: "medium",
                        recoveryStrategy: "retry",
                    }],
                    recoveryPatterns: ["adaptive_timeout", "performance_monitoring"],
                    faultTolerance: {
                        redundancy: 2,
                        failoverTime: 15000,
                        dataConsistency: "eventual",
                    },
                    metrics: {
                        mtbf: 3600,
                        mttr: 900,
                        availability: 75.0,
                        rpo: 600,
                        rto: 1200,
                    },
                },
                ...overrides,
            }),
        };

        const factory = scenarios[scenario];
        if (!factory) {
            throw new Error(`Unknown failure scenario: ${scenario}`);
        }

        return factory();
    }

    /**
     * Create evolution path showing resilience improvement
     */
    createEvolutionPath(stages = 4): ResilienceFixture[] {
        const evolutionStages = ["reactive", "proactive", "predictive", "self-healing"];
        const baseMtbf = 86400; // 24 hours in seconds
        const baseMttr = 300; // 5 minutes in seconds
        const baseAvailability = 99.0; // 99%
        const baseRpo = 60; // 1 minute in seconds
        const baseRto = 300; // 5 minutes in seconds
        const improvementFactor = 0.9; // 0.9% availability improvement per stage
        
        return Array.from({ length: Math.min(stages, evolutionStages.length) }, (_, i) => {
            const baseMetrics = {
                mtbf: baseMtbf * Math.pow(2, i), // Double MTBF each stage
                mttr: baseMttr / Math.pow(2, i), // Halve MTTR each stage
                availability: baseAvailability + (improvementFactor * i), // Improve availability
                rpo: baseRpo / Math.pow(2, i),
                rto: baseRto / Math.pow(2, i),
            };

            return this.createComplete({
                emergence: {
                    capabilities: [
                        "basic_recovery",
                        ...(i >= 1 ? ["failure_prediction"] : []),
                        ...(i >= 2 ? ["pattern_learning"] : []),
                        ...(i >= 3 ? ["autonomous_healing"] : []),
                    ],
                    evolutionPath: evolutionStages.slice(0, i + 1).join(" → "),
                },
                resilience: {
                    failureScenarios: [
                        {
                            type: "timeout",
                            probability: 0.1 / (i + 1), // Reduce failure probability
                            impact: i < 2 ? "high" : "medium",
                            recoveryStrategy: i < 2 ? "retry" : "circuit-break",
                        },
                    ],
                    recoveryPatterns: [
                        "basic_retry",
                        ...(i >= 1 ? ["exponential_backoff"] : []),
                        ...(i >= 2 ? ["circuit_breaker", "bulkhead"] : []),
                        ...(i >= 3 ? ["self_healing", "predictive_scaling"] : []),
                    ],
                    faultTolerance: {
                        redundancy: 1 + i,
                        failoverTime: 5000 / (i + 1), // 5 seconds base, faster each stage
                        dataConsistency: i < 2 ? "eventual" : "strong",
                    },
                    metrics: baseMetrics,
                },
            });
        });
    }
}

// Export factory instance
export const resilienceFixtureFactory = new ResilienceFixtureFactory();

// Export pre-built fixtures
export const resilienceFixtures = {
    minimal: resilienceFixtureFactory.createMinimal(),
    complete: resilienceFixtureFactory.createComplete(),
    withDefaults: resilienceFixtureFactory.createWithDefaults(),
    variants: {
        highAvailability: resilienceFixtureFactory.createVariant("highAvailability"),
        chaosEngineering: resilienceFixtureFactory.createVariant("chaosEngineering"),
        costOptimized: resilienceFixtureFactory.createVariant("costOptimized"),
    },
    scenarios: {
        networkPartition: resilienceFixtureFactory.createFailureScenario("network_partition"),
        resourceExhaustion: resilienceFixtureFactory.createFailureScenario("resource_exhaustion"),
        cascadeFailure: resilienceFixtureFactory.createFailureScenario("cascade_failure"),
        slowDegradation: resilienceFixtureFactory.createFailureScenario("slow_degradation"),
    },
    evolution: {
        reactive: resilienceFixtureFactory.createEvolutionPath(1)[0],
        proactive: resilienceFixtureFactory.createEvolutionPath(2)[1],
        predictive: resilienceFixtureFactory.createEvolutionPath(3)[2],
        selfHealing: resilienceFixtureFactory.createEvolutionPath(4)[3],
    },
};

/**
 * Resource Provider Test Fixtures
 * 
 * Comprehensive test fixtures for the ResourceProvider component, providing
 * validated configurations for testing resource allocation, usage tracking,
 * and optimization patterns across all three tiers.
 */

import type {
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject,
} from "@vrooli/shared";
import {
    chatConfigFixtures,
    routineConfigFixtures,
    runConfigFixtures,
} from "@vrooli/shared";
import type {
    ExecutionFixture,
    EmergenceDefinition,
    IntegrationDefinition,
} from "../types.js";
import { BaseExecutionFixtureFactory } from "../executionFactories.js";

/**
 * Resource types available in the system
 */
export type ResourceType = "cpu" | "memory" | "storage" | "network" | "api_calls" | "credits" | "gpu";

/**
 * Resource allocation strategy
 */
export type AllocationStrategy = "fair_share" | "priority_based" | "elastic" | "reserved" | "spot";

/**
 * Resource-specific fixture interface
 */
export interface ResourceFixture<TConfig extends BaseConfigObject = BaseConfigObject> extends ExecutionFixture<TConfig> {
    resources: {
        /** Available resource pools */
        pools: Array<{
            name: string;
            type: ResourceType;
            capacity: number;
            unit: string;
            tier: "tier1" | "tier2" | "tier3" | "shared";
            allocation: AllocationStrategy;
            reserved?: number;
            burstable?: boolean;
            cost?: number;
        }>;
        /** Resource usage patterns */
        usagePatterns: Array<{
            pattern: string;
            resources: ResourceType[];
            typical: Record<ResourceType, number>;
            peak: Record<ResourceType, number>;
            duration: number;
        }>;
        /** Resource limits and quotas */
        limits: {
            perUser?: Record<ResourceType, number>;
            perTeam?: Record<ResourceType, number>;
            perRequest?: Record<ResourceType, number>;
            global?: Record<ResourceType, number>;
        };
        /** Optimization strategies */
        optimization: {
            strategies: string[];
            targetUtilization: number;
            scalingPolicy: "vertical" | "horizontal" | "hybrid";
            costOptimization: boolean;
        };
        /** Resource monitoring */
        monitoring: {
            metrics: string[];
            alertThresholds: Record<string, number>;
            samplingInterval: number;
            retentionDays: number;
        };
    };
}

/**
 * Factory for creating resource test fixtures
 */
export class ResourceFixtureFactory extends BaseExecutionFixtureFactory<BaseConfigObject> {
    protected ConfigClass = runConfigFixtures.RunConfig;
    protected tier = "cross-tier" as const;

    createMinimal(overrides?: Partial<ResourceFixture>): ResourceFixture {
        return {
            config: runConfigFixtures.minimal,
            emergence: {
                capabilities: ["basic_allocation", "usage_tracking"],
                evolutionPath: "static → dynamic → intelligent",
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: ["resource.allocated", "resource.released"],
                consumedEvents: ["resource.request", "resource.release"],
                sharedResources: ["resource_pool", "usage_metrics"],
                ...overrides?.integration,
            },
            resources: {
                pools: [{
                    name: "default",
                    type: "cpu",
                    capacity: 100,
                    unit: "cores",
                    tier: "shared",
                    allocation: "fair_share",
                }],
                usagePatterns: [{
                    pattern: "standard",
                    resources: ["cpu"],
                    typical: { cpu: 10 },
                    peak: { cpu: 20 },
                    duration: 60000,
                }],
                limits: {
                    perRequest: { cpu: 50 },
                },
                optimization: {
                    strategies: ["basic_scheduling"],
                    targetUtilization: 70,
                    scalingPolicy: "vertical",
                    costOptimization: false,
                },
                monitoring: {
                    metrics: ["cpu_usage"],
                    alertThresholds: { cpu_usage: 90 },
                    samplingInterval: 60000,
                    retentionDays: 7,
                },
                ...overrides?.resources,
            },
        };
    }

    createComplete(overrides?: Partial<ResourceFixture>): ResourceFixture {
        return {
            config: runConfigFixtures.complete,
            emergence: {
                capabilities: [
                    "intelligent_allocation",
                    "predictive_scaling",
                    "cost_optimization",
                    "multi_tenant_isolation",
                    "resource_forecasting",
                    "auto_scaling",
                    "waste_detection",
                ],
                eventPatterns: [
                    "resource.*",
                    "scaling.*",
                    "allocation.*",
                    "optimization.*",
                ],
                evolutionPath: "static → dynamic → predictive → autonomous",
                emergenceConditions: {
                    minEvents: 1000,
                    requiredResources: ["monitoring", "analytics", "ml_models"],
                    timeframe: 86400000, // 24 hours
                },
                learningMetrics: {
                    performanceImprovement: "30% better resource utilization",
                    adaptationTime: "24 hours to optimal allocation patterns",
                    innovationRate: "5 new optimization patterns per week",
                },
                ...overrides?.emergence,
            },
            integration: {
                tier: "cross-tier",
                producedEvents: [
                    "resource.pool.created",
                    "resource.allocated",
                    "resource.released",
                    "resource.scaled",
                    "resource.optimized",
                    "resource.alert",
                    "resource.forecast",
                ],
                consumedEvents: [
                    "tier1.resource.request",
                    "tier2.resource.demand",
                    "tier3.resource.usage",
                    "system.performance.metrics",
                    "billing.cost.update",
                ],
                sharedResources: [
                    "resource_pools",
                    "allocation_table",
                    "usage_history",
                    "cost_metrics",
                ],
                crossTierDependencies: {
                    dependsOn: ["monitoring", "metrics_collector", "scheduler"],
                    provides: ["resource_management", "capacity_planning", "cost_control"],
                },
                mcpTools: ["resource_allocator", "usage_analyzer", "cost_optimizer"],
                ...overrides?.integration,
            },
            resources: {
                pools: [
                    {
                        name: "tier1_dedicated",
                        type: "cpu",
                        capacity: 1000,
                        unit: "cores",
                        tier: "tier1",
                        allocation: "reserved",
                        reserved: 200,
                        burstable: true,
                        cost: 0.1,
                    },
                    {
                        name: "tier1_memory",
                        type: "memory",
                        capacity: 4096,
                        unit: "GB",
                        tier: "tier1",
                        allocation: "elastic",
                        burstable: true,
                        cost: 0.05,
                    },
                    {
                        name: "tier2_compute",
                        type: "cpu",
                        capacity: 2000,
                        unit: "cores",
                        tier: "tier2",
                        allocation: "priority_based",
                        cost: 0.08,
                    },
                    {
                        name: "tier3_gpu",
                        type: "gpu",
                        capacity: 100,
                        unit: "units",
                        tier: "tier3",
                        allocation: "spot",
                        cost: 1.0,
                    },
                    {
                        name: "shared_api",
                        type: "api_calls",
                        capacity: 1000000,
                        unit: "calls/day",
                        tier: "shared",
                        allocation: "fair_share",
                        cost: 0.0001,
                    },
                    {
                        name: "global_credits",
                        type: "credits",
                        capacity: 1000000,
                        unit: "credits",
                        tier: "shared",
                        allocation: "priority_based",
                        cost: 0.01,
                    },
                ],
                usagePatterns: [
                    {
                        pattern: "ml_training",
                        resources: ["gpu", "memory", "storage"],
                        typical: { gpu: 50, memory: 2048, storage: 500 },
                        peak: { gpu: 100, memory: 4096, storage: 1000 },
                        duration: 3600000, // 1 hour
                    },
                    {
                        pattern: "api_serving",
                        resources: ["cpu", "memory", "api_calls"],
                        typical: { cpu: 100, memory: 512, api_calls: 10000 },
                        peak: { cpu: 500, memory: 2048, api_calls: 50000 },
                        duration: 86400000, // 24 hours
                    },
                    {
                        pattern: "batch_processing",
                        resources: ["cpu", "memory", "storage"],
                        typical: { cpu: 200, memory: 1024, storage: 200 },
                        peak: { cpu: 1000, memory: 4096, storage: 1000 },
                        duration: 7200000, // 2 hours
                    },
                ],
                limits: {
                    perUser: {
                        cpu: 100,
                        memory: 1024,
                        gpu: 10,
                        api_calls: 100000,
                        credits: 10000,
                    },
                    perTeam: {
                        cpu: 1000,
                        memory: 8192,
                        gpu: 50,
                        api_calls: 1000000,
                        credits: 100000,
                    },
                    perRequest: {
                        cpu: 50,
                        memory: 512,
                        gpu: 5,
                        api_calls: 1000,
                        credits: 100,
                    },
                    global: {
                        cpu: 5000,
                        memory: 20480,
                        gpu: 100,
                        api_calls: 10000000,
                        credits: 1000000,
                    },
                },
                optimization: {
                    strategies: [
                        "bin_packing",
                        "predictive_scaling",
                        "spot_instance_management",
                        "resource_recycling",
                        "workload_consolidation",
                        "auto_shutdown",
                    ],
                    targetUtilization: 85,
                    scalingPolicy: "hybrid",
                    costOptimization: true,
                },
                monitoring: {
                    metrics: [
                        "cpu_usage",
                        "memory_usage",
                        "gpu_utilization",
                        "api_call_rate",
                        "credit_burn_rate",
                        "queue_depth",
                        "allocation_efficiency",
                        "waste_percentage",
                    ],
                    alertThresholds: {
                        cpu_usage: 90,
                        memory_usage: 85,
                        gpu_utilization: 95,
                        api_call_rate: 90,
                        credit_burn_rate: 1000,
                        queue_depth: 100,
                        waste_percentage: 20,
                    },
                    samplingInterval: 10000, // 10 seconds
                    retentionDays: 30,
                },
                ...overrides?.resources,
            },
            validation: {
                emergenceTests: [
                    "intelligent_allocation_behavior",
                    "predictive_scaling_accuracy",
                    "cost_optimization_effectiveness",
                ],
                integrationTests: [
                    "multi_tier_resource_sharing",
                    "allocation_fairness",
                    "limit_enforcement",
                ],
                evolutionTests: [
                    "utilization_improvement",
                    "cost_reduction",
                    "allocation_speed",
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

    createWithDefaults(overrides?: Partial<ResourceFixture>): ResourceFixture {
        return this.createMinimal({
            resources: {
                pools: [
                    {
                        name: "standard_cpu",
                        type: "cpu",
                        capacity: 500,
                        unit: "cores",
                        tier: "shared",
                        allocation: "fair_share",
                        cost: 0.05,
                    },
                    {
                        name: "standard_memory",
                        type: "memory",
                        capacity: 2048,
                        unit: "GB",
                        tier: "shared",
                        allocation: "fair_share",
                        cost: 0.03,
                    },
                ],
                usagePatterns: [
                    {
                        pattern: "default",
                        resources: ["cpu", "memory"],
                        typical: { cpu: 50, memory: 256 },
                        peak: { cpu: 100, memory: 512 },
                        duration: 300000, // 5 minutes
                    },
                ],
                limits: {
                    perUser: { cpu: 200, memory: 1024 },
                    perRequest: { cpu: 100, memory: 512 },
                },
                optimization: {
                    strategies: ["basic_scheduling", "simple_scaling"],
                    targetUtilization: 75,
                    scalingPolicy: "vertical",
                    costOptimization: true,
                },
                monitoring: {
                    metrics: ["cpu_usage", "memory_usage", "allocation_count"],
                    alertThresholds: {
                        cpu_usage: 85,
                        memory_usage: 80,
                    },
                    samplingInterval: 30000, // 30 seconds
                    retentionDays: 14,
                },
            },
            ...overrides,
        });
    }

    createVariant(variant: string, overrides?: Partial<ResourceFixture>): ResourceFixture {
        const variants: Record<string, () => ResourceFixture> = {
            highPerformance: () => this.createComplete({
                resources: {
                    pools: [
                        {
                            name: "premium_cpu",
                            type: "cpu",
                            capacity: 5000,
                            unit: "cores",
                            tier: "tier3",
                            allocation: "reserved",
                            reserved: 1000,
                            cost: 0.2,
                        },
                        {
                            name: "premium_gpu",
                            type: "gpu",
                            capacity: 500,
                            unit: "units",
                            tier: "tier3",
                            allocation: "reserved",
                            reserved: 100,
                            cost: 2.0,
                        },
                    ],
                    usagePatterns: [
                        {
                            pattern: "high_performance",
                            resources: ["cpu", "gpu", "memory"],
                            typical: { cpu: 1000, gpu: 100, memory: 8192 },
                            peak: { cpu: 5000, gpu: 500, memory: 32768 },
                            duration: 3600000,
                        },
                    ],
                    limits: {
                        perUser: { cpu: 5000, gpu: 500, memory: 32768 },
                    },
                    optimization: {
                        strategies: ["performance_first", "latency_optimization"],
                        targetUtilization: 95,
                        scalingPolicy: "horizontal",
                        costOptimization: false,
                    },
                    monitoring: {
                        metrics: ["latency", "throughput", "queue_time"],
                        alertThresholds: { latency: 10, queue_time: 1000 },
                        samplingInterval: 1000, // 1 second
                        retentionDays: 90,
                    },
                },
                ...overrides,
            }),

            costOptimized: () => this.createComplete({
                resources: {
                    pools: [
                        {
                            name: "spot_cpu",
                            type: "cpu",
                            capacity: 2000,
                            unit: "cores",
                            tier: "shared",
                            allocation: "spot",
                            cost: 0.02,
                        },
                        {
                            name: "preemptible_memory",
                            type: "memory",
                            capacity: 8192,
                            unit: "GB",
                            tier: "shared",
                            allocation: "spot",
                            cost: 0.01,
                        },
                    ],
                    usagePatterns: [
                        {
                            pattern: "cost_sensitive",
                            resources: ["cpu", "memory"],
                            typical: { cpu: 100, memory: 512 },
                            peak: { cpu: 500, memory: 2048 },
                            duration: 600000,
                        },
                    ],
                    limits: {
                        perUser: { cpu: 500, memory: 2048 },
                        perRequest: { cpu: 100, memory: 512 },
                    },
                    optimization: {
                        strategies: [
                            "spot_bidding",
                            "workload_migration",
                            "off_peak_scheduling",
                            "resource_packing",
                        ],
                        targetUtilization: 90,
                        scalingPolicy: "hybrid",
                        costOptimization: true,
                    },
                    monitoring: {
                        metrics: ["cost_per_request", "spot_interruption_rate"],
                        alertThresholds: {
                            cost_per_request: 0.1,
                            spot_interruption_rate: 0.1,
                        },
                        samplingInterval: 60000,
                        retentionDays: 30,
                    },
                },
                ...overrides,
            }),

            multiTenant: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "tenant_isolation",
                        "fair_resource_sharing",
                        "noisy_neighbor_detection",
                    ],
                },
                resources: {
                    pools: [
                        {
                            name: "tenant_isolated_cpu",
                            type: "cpu",
                            capacity: 10000,
                            unit: "cores",
                            tier: "shared",
                            allocation: "fair_share",
                            cost: 0.1,
                        },
                    ],
                    usagePatterns: [
                        {
                            pattern: "multi_tenant",
                            resources: ["cpu", "memory", "storage"],
                            typical: { cpu: 50, memory: 256, storage: 100 },
                            peak: { cpu: 200, memory: 1024, storage: 500 },
                            duration: 3600000,
                        },
                    ],
                    limits: {
                        perUser: { cpu: 100, memory: 512, storage: 1000 },
                        perTeam: { cpu: 1000, memory: 5120, storage: 10000 },
                    },
                    optimization: {
                        strategies: [
                            "tenant_isolation",
                            "fair_queuing",
                            "resource_quotas",
                            "priority_classes",
                        ],
                        targetUtilization: 80,
                        scalingPolicy: "hybrid",
                        costOptimization: true,
                    },
                    monitoring: {
                        metrics: [
                            "tenant_usage",
                            "fairness_index",
                            "isolation_violations",
                        ],
                        alertThresholds: {
                            fairness_index: 0.8,
                            isolation_violations: 0,
                        },
                        samplingInterval: 5000,
                        retentionDays: 60,
                    },
                },
                ...overrides,
            }),
        };

        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown resource variant: ${variant}`);
        }

        return factory();
    }

    /**
     * Create resource shortage scenarios for testing
     */
    createShortageScenario(
        resource: ResourceType,
        severity: "mild" | "moderate" | "severe" | "critical",
    ): ResourceFixture {
        const severityMultipliers = {
            mild: 0.8,
            moderate: 0.5,
            severe: 0.2,
            critical: 0.05,
        };

        const multiplier = severityMultipliers[severity];

        return this.createComplete({
            resources: {
                pools: [{
                    name: `${resource}_constrained`,
                    type: resource,
                    capacity: 100 * multiplier,
                    unit: resource === "memory" ? "GB" : "units",
                    tier: "shared",
                    allocation: "priority_based",
                    cost: 0.5 / multiplier, // Higher cost when scarce
                }],
                usagePatterns: [{
                    pattern: "shortage_scenario",
                    resources: [resource],
                    typical: { [resource]: 80 },
                    peak: { [resource]: 150 }, // Exceeds available capacity
                    duration: 300000,
                }],
                limits: {
                    perRequest: { [resource]: 10 * multiplier },
                },
                optimization: {
                    strategies: ["aggressive_scheduling", "preemption", "queuing"],
                    targetUtilization: 95,
                    scalingPolicy: "vertical",
                    costOptimization: false,
                },
                monitoring: {
                    metrics: ["queue_depth", "rejection_rate", "wait_time"],
                    alertThresholds: {
                        queue_depth: 50,
                        rejection_rate: 0.1,
                        wait_time: 30000,
                    },
                    samplingInterval: 1000,
                    retentionDays: 7,
                },
            },
        });
    }

    /**
     * Create evolution path showing resource management improvement
     */
    createEvolutionPath(stages = 4): ResourceFixture[] {
        const evolutionStages = ["static", "dynamic", "predictive", "autonomous"];
        
        return Array.from({ length: Math.min(stages, evolutionStages.length) }, (_, i) => {
            return this.createComplete({
                emergence: {
                    capabilities: [
                        "basic_allocation",
                        ...(i >= 1 ? ["dynamic_scaling"] : []),
                        ...(i >= 2 ? ["usage_prediction", "cost_optimization"] : []),
                        ...(i >= 3 ? ["self_optimization", "waste_elimination"] : []),
                    ],
                    evolutionPath: evolutionStages.slice(0, i + 1).join(" → "),
                },
                resources: {
                    pools: [{
                        name: "evolving_pool",
                        type: "cpu",
                        capacity: 1000 * (i + 1), // Increase capacity
                        unit: "cores",
                        tier: "shared",
                        allocation: i < 2 ? "fair_share" : "priority_based",
                        burstable: i >= 1,
                        cost: 0.1 / (i + 1), // Reduce cost through efficiency
                    }],
                    usagePatterns: [{
                        pattern: "evolving",
                        resources: ["cpu"],
                        typical: { cpu: 100 },
                        peak: { cpu: 200 },
                        duration: 300000,
                    }],
                    limits: {
                        perRequest: { cpu: 50 * (i + 1) },
                    },
                    optimization: {
                        strategies: [
                            "basic_scheduling",
                            ...(i >= 1 ? ["auto_scaling"] : []),
                            ...(i >= 2 ? ["predictive_provisioning"] : []),
                            ...(i >= 3 ? ["ml_optimization"] : []),
                        ],
                        targetUtilization: 70 + (5 * i), // Improve utilization
                        scalingPolicy: i < 2 ? "vertical" : "hybrid",
                        costOptimization: i >= 2,
                    },
                    monitoring: {
                        metrics: [
                            "usage",
                            ...(i >= 1 ? ["efficiency"] : []),
                            ...(i >= 2 ? ["cost", "waste"] : []),
                            ...(i >= 3 ? ["prediction_accuracy"] : []),
                        ],
                        alertThresholds: {
                            usage: 90 - (5 * i), // More efficient thresholds
                        },
                        samplingInterval: 60000 / (i + 1), // Faster sampling
                        retentionDays: 7 * (i + 1), // Longer retention
                    },
                },
            });
        });
    }
}

// Export factory instance
export const resourceFixtureFactory = new ResourceFixtureFactory();

// Export pre-built fixtures
export const resourceFixtures = {
    minimal: resourceFixtureFactory.createMinimal(),
    complete: resourceFixtureFactory.createComplete(),
    withDefaults: resourceFixtureFactory.createWithDefaults(),
    variants: {
        highPerformance: resourceFixtureFactory.createVariant("highPerformance"),
        costOptimized: resourceFixtureFactory.createVariant("costOptimized"),
        multiTenant: resourceFixtureFactory.createVariant("multiTenant"),
    },
    shortages: {
        mildCpuShortage: resourceFixtureFactory.createShortageScenario("cpu", "mild"),
        moderateMemoryShortage: resourceFixtureFactory.createShortageScenario("memory", "moderate"),
        severeGpuShortage: resourceFixtureFactory.createShortageScenario("gpu", "severe"),
        criticalApiShortage: resourceFixtureFactory.createShortageScenario("api_calls", "critical"),
    },
    evolution: {
        static: resourceFixtureFactory.createEvolutionPath(1)[0],
        dynamic: resourceFixtureFactory.createEvolutionPath(2)[1],
        predictive: resourceFixtureFactory.createEvolutionPath(3)[2],
        autonomous: resourceFixtureFactory.createEvolutionPath(4)[3],
    },
};

/**
 * Resource Monitor - Monitoring and analytics for resource usage across all tiers
 * 
 * This component provides:
 * - Real-time usage monitoring across all tiers
 * - Cross-tier resource analytics and reporting
 * - Optimization suggestions based on usage patterns
 * - Performance tracking and efficiency analysis
 * 
 * Note: This is a monitoring-only component that observes resource events
 * but does NOT perform resource allocation (which violates tier hierarchy)
 */

import { type Logger } from "winston";
import {
    type ResourceType,
    type ExecutionResourceUsage,
    type ResourceAllocation,
    type OptimizationSuggestion,
    type EfficiencyMetrics,
    type ResourceAccounting,
    type ResourceUsageSummary,
    type ResourceCostSummary,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import {
    UsageTracker,
    ResourceMetrics,
    type UsageTrackerConfig,
} from "../../cross-cutting/resources/index.js";

/**
 * Cross-tier resource monitoring event
 */
interface CrossTierResourceEvent {
    tier: "tier1" | "tier2" | "tier3";
    type: "allocation" | "usage" | "release" | "optimization";
    timestamp: Date;
    data: Record<string, unknown>;
}

/**
 * Resource efficiency analysis
 */
interface EfficiencyAnalysis {
    overallEfficiency: number;
    tierEfficiency: Record<string, number>;
    bottlenecks: Array<{
        tier: string;
        resource: ResourceType;
        severity: "low" | "medium" | "high" | "critical";
        description: string;
    }>;
    trends: Array<{
        metric: string;
        direction: "improving" | "declining" | "stable";
        changePercent: number;
    }>;
}

/**
 * Resource utilization summary
 */
interface UtilizationSummary {
    period: {
        start: Date;
        end: Date;
    };
    totalAllocations: number;
    successRate: number;
    averageDuration: number;
    costEfficiency: number;
    resourceBreakdown: Map<ResourceType, {
        total: number;
        peak: number;
        average: number;
        cost: number;
    }>;
}

/**
 * Resource Monitor - Observes and analyzes resource usage across all tiers
 * 
 * Features:
 * - Event-driven monitoring (does not control allocation)
 * - Cross-tier efficiency analysis
 * - Performance trend identification
 * - Optimization recommendations
 * - Cost analysis and reporting
 */
export class ResourceMonitor {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    
    // Monitoring components
    private readonly tier1Tracker: UsageTracker;
    private readonly tier2Tracker: UsageTracker;
    private readonly tier3Tracker: UsageTracker;
    private readonly overallMetrics: ResourceMetrics;
    
    // Event storage
    private readonly recentEvents: CrossTierResourceEvent[] = [];
    private readonly maxEvents = 10000;
    
    // Analysis cache
    private lastAnalysis?: EfficiencyAnalysis;
    private lastAnalysisTime = 0;
    private readonly analysisIntervalMs = 300000; // 5 minutes
    
    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Initialize tracking for each tier
        this.tier1Tracker = new UsageTracker(
            {
                trackerId: "tier1-monitor",
                scope: "swarm",
                scopeId: "global",
                windowSize: 3600000, // 1 hour
                aggregationInterval: 300000, // 5 minutes
            },
            logger,
            eventBus,
        );
        
        this.tier2Tracker = new UsageTracker(
            {
                trackerId: "tier2-monitor",
                scope: "run",
                scopeId: "global",
                windowSize: 3600000,
                aggregationInterval: 300000,
            },
            logger,
            eventBus,
        );
        
        this.tier3Tracker = new UsageTracker(
            {
                trackerId: "tier3-monitor",
                scope: "step",
                scopeId: "global",
                windowSize: 3600000,
                aggregationInterval: 300000,
            },
            logger,
            eventBus,
        );
        
        this.overallMetrics = new ResourceMetrics("cross-tier-monitor", logger, eventBus);
        
        this.startEventListening();
        
        this.logger.info("[ResourceMonitor] Initialized cross-tier monitoring");
    }
    
    /**
     * Get comprehensive usage report across all tiers
     */
    async getUsageReport(
        startTime?: Date,
        endTime?: Date,
    ): Promise<ResourceAccounting> {
        const period = {
            start: startTime || new Date(Date.now() - 3600000), // Last hour
            end: endTime || new Date(),
        };
        
        // Get usage summaries from each tier
        const tier1Summary = this.tier1Tracker.getSummary();
        const tier2Summary = this.tier2Tracker.getSummary();
        const tier3Summary = this.tier3Tracker.getSummary();
        
        // Aggregate usage across tiers
        const aggregatedUsage = this.aggregateUsageAcrossTiers(
            tier1Summary,
            tier2Summary,
            tier3Summary,
        );
        
        // Calculate costs
        const costs = this.calculateCrossTierCosts(aggregatedUsage);
        
        // Calculate efficiency metrics
        const efficiency = this.calculateEfficiencyMetrics();
        
        return {
            period,
            usage: aggregatedUsage,
            costs,
            efficiency,
        };
    }
    
    /**
     * Get optimization suggestions based on cross-tier analysis
     */
    async getOptimizationSuggestions(): Promise<OptimizationSuggestion[]> {
        const analysis = await this.performEfficiencyAnalysis();
        const suggestions: OptimizationSuggestion[] = [];
        
        // Generate suggestions based on bottlenecks
        for (const bottleneck of analysis.bottlenecks) {
            if (bottleneck.severity === "high" || bottleneck.severity === "critical") {
                suggestions.push({
                    id: generatePk(),
                    type: "reduce",
                    targetResource: bottleneck.resource,
                    currentUsage: 0, // Would be calculated from actual data
                    projectedSavings: 0, // Would be calculated
                    implementation: `Optimize ${bottleneck.resource} usage in ${bottleneck.tier}`,
                    risk: bottleneck.severity === "critical" ? "low" : "medium",
                });
            }
        }
        
        // Generate suggestions based on trends
        for (const trend of analysis.trends) {
            if (trend.direction === "declining" && Math.abs(trend.changePercent) > 10) {
                suggestions.push({
                    id: generatePk(),
                    type: "substitute",
                    targetResource: trend.metric,
                    currentUsage: 0,
                    projectedSavings: Math.abs(trend.changePercent),
                    implementation: `Address declining trend in ${trend.metric}`,
                    risk: "medium",
                });
            }
        }
        
        return suggestions;
    }
    
    /**
     * Get utilization summary for a specific time period
     */
    async getUtilizationSummary(
        startTime?: Date,
        endTime?: Date,
    ): Promise<UtilizationSummary> {
        const period = {
            start: startTime || new Date(Date.now() - 3600000),
            end: endTime || new Date(),
        };
        
        // Analyze events in the time period
        const periodEvents = this.recentEvents.filter(
            event => event.timestamp >= period.start && event.timestamp <= period.end,
        );
        
        const totalAllocations = periodEvents.filter(e => e.type === "allocation").length;
        const usageEvents = periodEvents.filter(e => e.type === "usage");
        
        // Calculate success rate
        const successfulUsage = usageEvents.filter(
            e => e.data.success === true,
        ).length;
        const successRate = usageEvents.length > 0 ? successfulUsage / usageEvents.length : 0;
        
        // Calculate average duration
        const durations = usageEvents
            .map(e => e.data.duration as number)
            .filter(d => typeof d === "number");
        const averageDuration = durations.length > 0
            ? durations.reduce((sum, d) => sum + d, 0) / durations.length
            : 0;
        
        // Calculate cost efficiency (simplified)
        const totalCost = usageEvents
            .map(e => e.data.cost as number)
            .filter(c => typeof c === "number")
            .reduce((sum, c) => sum + c, 0);
        const costEfficiency = totalCost > 0 ? successfulUsage / totalCost : 0;
        
        // Resource breakdown (simplified)
        const resourceBreakdown = new Map();
        // Would populate with actual resource usage data
        
        return {
            period,
            totalAllocations,
            successRate,
            averageDuration,
            costEfficiency,
            resourceBreakdown,
        };
    }
    
    /**
     * Start listening to resource events from all tiers
     */
    private startEventListening(): void {
        // Listen to Tier 1 events
        this.eventBus.subscribe("swarm.*", async (event) => {
            this.recordEvent("tier1", event.type, event.timestamp, event.data);
            
            if (event.type.includes("resource")) {
                await this.tier1Tracker.recordExecution(
                    event.data.usage as ExecutionResourceUsage,
                );
            }
        });
        
        // Listen to Tier 2 events
        this.eventBus.subscribe("run.*", async (event) => {
            this.recordEvent("tier2", event.type, event.timestamp, event.data);
            
            if (event.type.includes("resource") || event.type.includes("usage")) {
                await this.tier2Tracker.recordExecution(
                    event.data.usage as ExecutionResourceUsage,
                );
            }
        });
        
        // Listen to Tier 3 events
        this.eventBus.subscribe("resource.*", async (event) => {
            this.recordEvent("tier3", event.type, event.timestamp, event.data);
            
            if (event.data.usage) {
                await this.tier3Tracker.recordExecution(
                    event.data.usage as ExecutionResourceUsage,
                );
            }
        });
        
        // Listen to metrics events from shared components
        this.eventBus.subscribe("metrics.*", async (event) => {
            await this.overallMetrics.recordMetric(
                event.type,
                event.data.value as number,
                event.data.unit as string,
                event.data,
            );
        });
        
        this.logger.debug("[ResourceMonitor] Started event listening");
    }
    
    /**
     * Record a cross-tier resource event
     */
    private recordEvent(
        tier: "tier1" | "tier2" | "tier3",
        type: string,
        timestamp: Date,
        data: Record<string, unknown>,
    ): void {
        const event: CrossTierResourceEvent = {
            tier,
            type: type.includes("allocation") ? "allocation" :
                  type.includes("usage") ? "usage" :
                  type.includes("release") ? "release" : "optimization",
            timestamp,
            data,
        };
        
        this.recentEvents.push(event);
        
        // Keep only recent events
        if (this.recentEvents.length > this.maxEvents) {
            this.recentEvents.shift();
        }
    }
    
    /**
     * Perform comprehensive efficiency analysis
     */
    private async performEfficiencyAnalysis(): Promise<EfficiencyAnalysis> {
        const now = Date.now();
        
        // Use cached analysis if recent
        if (this.lastAnalysis && (now - this.lastAnalysisTime) < this.analysisIntervalMs) {
            return this.lastAnalysis;
        }
        
        // Calculate overall efficiency
        const tier1Summary = this.tier1Tracker.getSummary();
        const tier2Summary = this.tier2Tracker.getSummary();
        const tier3Summary = this.tier3Tracker.getSummary();
        
        const overallEfficiency = (
            (tier1Summary.efficiency || 0) +
            (tier2Summary.efficiency || 0) +
            (tier3Summary.efficiency || 0)
        ) / 3;
        
        // Calculate tier-specific efficiency
        const tierEfficiency = {
            tier1: tier1Summary.efficiency || 0,
            tier2: tier2Summary.efficiency || 0,
            tier3: tier3Summary.efficiency || 0,
        };
        
        // Identify bottlenecks (simplified)
        const bottlenecks = [];
        if (tierEfficiency.tier1 < 0.7) {
            bottlenecks.push({
                tier: "tier1",
                resource: ResourceType.CREDITS,
                severity: "medium" as const,
                description: "Low efficiency in swarm coordination",
            });
        }
        
        // Calculate trends (simplified)
        const trends = [
            {
                metric: "overall_efficiency",
                direction: "stable" as const,
                changePercent: 0,
            },
        ];
        
        this.lastAnalysis = {
            overallEfficiency,
            tierEfficiency,
            bottlenecks,
            trends,
        };
        this.lastAnalysisTime = now;
        
        return this.lastAnalysis;
    }
    
    /**
     * Aggregate usage summaries across tiers
     */
    private aggregateUsageAcrossTiers(
        tier1: any,
        tier2: any,
        tier3: any,
    ): ResourceUsageSummary[] {
        // Simplified aggregation - would implement proper cross-tier aggregation
        return [];
    }
    
    /**
     * Calculate costs across all tiers
     */
    private calculateCrossTierCosts(usage: ResourceUsageSummary[]): ResourceCostSummary[] {
        // Simplified cost calculation
        return [];
    }
    
    /**
     * Calculate efficiency metrics
     */
    private calculateEfficiencyMetrics(): EfficiencyMetrics {
        // Simplified efficiency calculation
        return {
            overallEfficiency: 0.8,
            wastedResources: 15,
            optimizationPotential: 20,
            recommendations: [],
        };
    }
    
    /**
     * Shutdown monitoring
     */
    shutdown(): void {
        this.tier1Tracker.shutdown();
        this.tier2Tracker.shutdown();
        this.tier3Tracker.shutdown();
        this.overallMetrics.shutdown();
        
        this.logger.info("[ResourceMonitor] Shutdown complete");
    }
}
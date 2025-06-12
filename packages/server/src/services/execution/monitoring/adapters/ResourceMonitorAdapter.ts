/**
 * ResourceMonitorAdapter - Adapter for integration layer ResourceMonitor
 * 
 * Maintains backward compatibility with the original ResourceMonitor
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { ResourceMetricsAdapter } from "./ResourceMetricsAdapter.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";
import { 
    type ResourceAccounting,
    type ResourceOptimizationSuggestion as OptimizationSuggestion,
    type ResourceUsageSummary,
    type ResourceCostSummary,
    type EfficiencyMetrics as SharedEfficiencyMetrics,
} from "@vrooli/shared";

/**
 * Utilization summary structure
 */
export interface UtilizationSummary {
    period: {
        start: Date;
        end: Date;
    };
    resources: {
        credits: {
            allocated: number;
            used: number;
            utilization: number;
        };
        tokens: {
            allocated: number;
            used: number;
            utilization: number;
        };
        tools: {
            allocated: number;
            used: number;
            utilization: number;
        };
    };
    efficiency: SharedEfficiencyMetrics;
    trends: {
        utilizationTrend: 'increasing' | 'stable' | 'decreasing';
        efficiencyTrend: 'improving' | 'stable' | 'degrading';
    };
}

/**
 * ResourceMonitorAdapter - Cross-tier resource monitoring through unified service
 */
export class ResourceMonitorAdapter extends BaseMonitoringAdapter {
    private readonly resourceMetrics: ResourceMetricsAdapter;
    private readonly defaultLimits = {
        credits: 10000,
        tokens: 1000000,
        tools: 100,
    };

    constructor(
        eventBus: EventBus,
        logger: Logger,
        resourceMetrics?: ResourceMetricsAdapter
    ) {
        super(
            {
                componentName: "ResourceMonitor",
                tier: "cross-cutting",
                eventChannels: [
                    "resource.allocated",
                    "resource.released",
                    "resource.exhausted",
                    "tier1.resource",
                    "tier2.resource",
                    "tier3.resource",
                ],
                enableEventBus: true,
            },
            logger,
            eventBus
        );

        // Use provided resourceMetrics or create new instance
        this.resourceMetrics = resourceMetrics || new ResourceMetricsAdapter(eventBus, logger);
    }

    /**
     * Get usage report for a time period
     */
    async getUsageReport(startTime?: Date, endTime?: Date): Promise<ResourceAccounting> {
        const start = startTime || new Date(Date.now() - 86400000); // Default: 24 hours ago
        const end = endTime || new Date();

        // Query metrics from unified service
        const metrics = await this.queryMetrics({
            startTime: start,
            endTime: end,
        });

        // Aggregate usage data
        const usage = this.aggregateUsage(metrics);
        const cost = this.calculateCost(usage);
        const efficiency = this.calculateEfficiency(usage, metrics);

        const report: ResourceAccounting = {
            period: {
                start,
                end,
            },
            usage,
            cost,
            efficiency,
            allocations: [], // Would need to track allocations separately
        };

        // Record the report generation
        await this.recordMetric(
            'usage_report_generated',
            1,
            'business',
            {
                periodStart: start,
                periodEnd: end,
                totalCost: cost.reduce((sum, c) => sum + c.totalCost, 0),
            }
        );

        return report;
    }

    /**
     * Get optimization suggestions
     */
    async getOptimizationSuggestions(): Promise<OptimizationSuggestion[]> {
        const suggestions: OptimizationSuggestion[] = [];

        // Get recent metrics and analysis
        const costAnalysis = this.resourceMetrics.getCostAnalysis();
        const bottlenecks = this.resourceMetrics.detectBottlenecks();
        const utilization = this.resourceMetrics.getUtilizationSummary();

        // Generate suggestions based on analysis
        if (costAnalysis.optimizationPotential > 100) {
            suggestions.push({
                type: 'reduce',
                description: 'High optimization potential detected in resource usage',
                estimatedSavings: costAnalysis.optimizationPotential,
                impact: 'high',
                implementation: 'Review failed executions and implement retry strategies',
            });
        }

        if (utilization.utilizationRate < 0.3) {
            suggestions.push({
                type: 'batch',
                description: 'Low resource utilization detected',
                estimatedSavings: 0,
                impact: 'medium',
                implementation: 'Consider batching operations or increasing concurrency',
            });
        }

        // Add suggestions from bottleneck analysis
        for (const bottleneck of bottlenecks) {
            suggestions.push({
                type: 'remove_bottleneck',
                description: `Bottleneck detected in ${bottleneck.resource}`,
                estimatedSavings: 0,
                impact: bottleneck.severity as 'low' | 'medium' | 'high',
                implementation: bottleneck.recommendations.join('; '),
            });
        }

        // Check for cost trends
        if (costAnalysis.trends.daily.length >= 3) {
            const recentTrend = costAnalysis.trends.daily.slice(-3);
            const avgIncrease = recentTrend[2] - recentTrend[0];
            if (avgIncrease > recentTrend[0] * 0.2) {
                suggestions.push({
                    type: 'control_costs',
                    description: 'Rapid cost increase detected',
                    estimatedSavings: avgIncrease * 30, // Monthly projection
                    impact: 'high',
                    implementation: 'Implement cost controls and budget alerts',
                });
            }
        }

        // Record suggestions generated
        await this.recordMetric(
            'optimization_suggestions_generated',
            suggestions.length,
            'business',
            {
                highImpact: suggestions.filter(s => s.impact === 'high').length,
                totalSavings: suggestions.reduce((sum, s) => sum + s.estimatedSavings, 0),
            }
        );

        return suggestions;
    }

    /**
     * Get utilization summary
     */
    async getUtilizationSummary(startTime?: Date, endTime?: Date): Promise<UtilizationSummary> {
        const start = startTime || new Date(Date.now() - 3600000); // Default: 1 hour ago
        const end = endTime || new Date();

        // Query metrics
        const metrics = await this.queryMetrics({
            startTime: start,
            endTime: end,
        });

        // Calculate utilization for each resource type
        const creditMetrics = metrics.filter(m => m.name.includes('credits'));
        const tokenMetrics = metrics.filter(m => m.name.includes('tokens'));
        const toolMetrics = metrics.filter(m => m.name.includes('tool'));

        const creditsUsed = this.sumMetricValues(creditMetrics);
        const tokensUsed = this.sumMetricValues(tokenMetrics);
        const toolsUsed = this.sumMetricValues(toolMetrics);

        const creditsAllocated = this.defaultLimits.credits;
        const tokensAllocated = this.defaultLimits.tokens;
        const toolsAllocated = this.defaultLimits.tools;

        // Calculate efficiency
        const usage = { credits: creditsUsed, tokens: tokensUsed, toolCalls: toolsUsed };
        const efficiency = this.calculateEfficiency(usage as any, metrics);

        // Determine trends
        const utilizationTrend = this.determineUtilizationTrend(metrics);
        const efficiencyTrend = this.determineEfficiencyTrend(metrics);

        const summary: UtilizationSummary = {
            period: { start, end },
            resources: {
                credits: {
                    allocated: creditsAllocated,
                    used: creditsUsed,
                    utilization: creditsUsed / creditsAllocated,
                },
                tokens: {
                    allocated: tokensAllocated,
                    used: tokensUsed,
                    utilization: tokensUsed / tokensAllocated,
                },
                tools: {
                    allocated: toolsAllocated,
                    used: toolsUsed,
                    utilization: toolsUsed / toolsAllocated,
                },
            },
            efficiency,
            trends: {
                utilizationTrend,
                efficiencyTrend,
            },
        };

        // Record utilization metrics
        await this.recordMetric(
            'overall_utilization',
            (creditsUsed / creditsAllocated + tokensUsed / tokensAllocated + toolsUsed / toolsAllocated) / 3,
            'performance',
            { period: `${start.toISOString()}_${end.toISOString()}` }
        );

        return summary;
    }

    /**
     * Handle incoming events
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        // Forward relevant events to ResourceMetrics
        if (channel.startsWith('resource.') || channel.includes('.resource')) {
            await this.resourceMetrics.recordExecution(
                {
                    credits: event.data?.credits || 0,
                    tokens: event.data?.tokens || 0,
                    toolCalls: event.data?.toolCalls || 0,
                },
                event.data?.duration || 0,
                event.data?.success !== false,
                event.data?.operation
            );
        }

        // Track specific resource events
        switch (event.type) {
            case 'resource_allocated':
                await this.recordMetric(
                    'resource_allocated',
                    event.data.amount,
                    'business',
                    {
                        resourceType: event.data.type,
                        tier: event.source?.tier,
                        requestor: event.data.requestor,
                    }
                );
                break;

            case 'resource_released':
                await this.recordMetric(
                    'resource_released',
                    event.data.amount,
                    'business',
                    {
                        resourceType: event.data.type,
                        tier: event.source?.tier,
                        used: event.data.used,
                        unused: event.data.amount - event.data.used,
                    }
                );
                break;

            case 'resource_exhausted':
                await this.recordMetric(
                    'resource_exhausted',
                    1,
                    'health',
                    {
                        resourceType: event.data.type,
                        tier: event.source?.tier,
                        requested: event.data.requested,
                        available: event.data.available,
                    }
                );
                break;
        }
    }

    /**
     * Private helper methods
     */

    private aggregateUsage(metrics: any[]): ResourceUsageSummary[] {
        const resourceTypes = [
            { type: ResourceType.CREDITS, filter: 'credits' },
            { type: ResourceType.TOKENS, filter: 'tokens' },
            { type: ResourceType.API_CALLS, filter: 'tool' },
        ];

        return resourceTypes.map(({ type, filter }) => {
            const relevantMetrics = metrics.filter(m => m.name.includes(filter));
            const total = this.sumMetricValues(relevantMetrics);
            const values = relevantMetrics.map(m => Number(m.value));
            
            return {
                resourceType: type,
                totalConsumed: total,
                peakUsage: Math.max(...values, 0),
                averageUsage: values.length > 0 ? total / values.length : 0,
                utilizationRate: 0.8, // Would calculate based on limits
            };
        });
    }

    private calculateCost(usageSummaries: ResourceUsageSummary[]): ResourceCostSummary[] {
        const costModels = {
            [ResourceType.CREDITS]: 0.01,
            [ResourceType.TOKENS]: 0.00002, // per token
            [ResourceType.API_CALLS]: 0.05,
        };

        return usageSummaries.map(usage => {
            const unitCost = costModels[usage.resourceType] || 0;
            const totalCost = usage.totalConsumed * unitCost;

            return {
                resourceType: usage.resourceType,
                totalCost,
                breakdown: [
                    {
                        category: 'base',
                        amount: totalCost,
                        percentage: 100,
                    }
                ],
            };
        });
    }

    private calculateEfficiency(usageSummaries: ResourceUsageSummary[], metrics: any[]): SharedEfficiencyMetrics {
        const successMetrics = metrics.filter(m => m.name.includes('success'));
        const failureMetrics = metrics.filter(m => m.name.includes('failure'));
        
        const totalOperations = successMetrics.length + failureMetrics.length;
        const successRate = totalOperations > 0 ? successMetrics.length / totalOperations : 0;
        
        const totalCredits = usageSummaries.find(u => u.resourceType === ResourceType.CREDITS)?.totalConsumed || 0;
        const avgCreditsPerOp = totalOperations > 0 ? totalCredits / totalOperations : 0;
        const utilizationRate = avgCreditsPerOp > 0 ? Math.min(1, 100 / avgCreditsPerOp) : 0;
        
        const wasteRate = 1 - successRate;
        const optimizationScore = (successRate * 0.6) + (utilizationRate * 0.4);
        const totalCost = usageSummaries.reduce((sum, usage) => {
            const costModels = {
                [ResourceType.CREDITS]: 0.01,
                [ResourceType.TOKENS]: 0.00002,
                [ResourceType.API_CALLS]: 0.05,
            };
            return sum + (usage.totalConsumed * (costModels[usage.resourceType] || 0));
        }, 0);
        const costPerOutcome = successMetrics.length > 0 ? totalCost / successMetrics.length : 0;

        return {
            utilizationRate,
            wasteRate,
            optimizationScore,
            costPerOutcome,
        };
    }

    private sumMetricValues(metrics: any[]): number {
        return metrics.reduce((sum, metric) => {
            const value = typeof metric.value === 'number' ? metric.value : 0;
            return sum + value;
        }, 0);
    }

    private determineUtilizationTrend(metrics: any[]): 'increasing' | 'stable' | 'decreasing' {
        if (metrics.length < 10) return 'stable';
        
        const firstHalf = metrics.slice(0, Math.floor(metrics.length / 2));
        const secondHalf = metrics.slice(Math.floor(metrics.length / 2));
        
        const firstHalfSum = this.sumMetricValues(firstHalf);
        const secondHalfSum = this.sumMetricValues(secondHalf);
        
        const change = (secondHalfSum - firstHalfSum) / firstHalfSum;
        
        if (change > 0.1) return 'increasing';
        if (change < -0.1) return 'decreasing';
        return 'stable';
    }

    private determineEfficiencyTrend(metrics: any[]): 'improving' | 'stable' | 'degrading' {
        const successMetrics = metrics.filter(m => m.name.includes('success'));
        const failureMetrics = metrics.filter(m => m.name.includes('failure'));
        
        if (successMetrics.length + failureMetrics.length < 10) return 'stable';
        
        // Compare first half vs second half success rates
        const midpoint = Math.floor(metrics.length / 2);
        const firstHalfSuccess = successMetrics.filter(m => metrics.indexOf(m) < midpoint).length;
        const firstHalfTotal = metrics.slice(0, midpoint).filter(m => 
            m.name.includes('success') || m.name.includes('failure')
        ).length;
        
        const secondHalfSuccess = successMetrics.filter(m => metrics.indexOf(m) >= midpoint).length;
        const secondHalfTotal = metrics.slice(midpoint).filter(m => 
            m.name.includes('success') || m.name.includes('failure')
        ).length;
        
        const firstRate = firstHalfTotal > 0 ? firstHalfSuccess / firstHalfTotal : 0;
        const secondRate = secondHalfTotal > 0 ? secondHalfSuccess / secondHalfTotal : 0;
        
        const change = secondRate - firstRate;
        
        if (change > 0.1) return 'improving';
        if (change < -0.1) return 'degrading';
        return 'stable';
    }
}
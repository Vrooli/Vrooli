/**
 * MetacognitiveMonitorAdapter - Adapter for Tier 1 MetacognitiveMonitor
 * 
 * Maintains backward compatibility with the original MetacognitiveMonitor
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";

/**
 * Performance analysis input (from original MetacognitiveMonitor)
 */
export interface PerformanceAnalysisInput {
    swarmId: string;
    executionTime: number;
    resourcesUsed: {
        agents: number;
        credits: number;
        decisions: number;
    };
    outcomes: {
        decisionsApproved: number;
        decisionsRejected: number;
        goalsAchieved: number;
        goalsFailed: number;
    };
    errors: Array<{
        timestamp: Date;
        type: string;
        severity: 'low' | 'medium' | 'high';
        message: string;
    }>;
}

/**
 * Performance snapshot structure
 */
export interface PerformanceSnapshot {
    timestamp: Date;
    swarmId: string;
    metrics: {
        executionTime: number;
        resourceEfficiency: number;
        successRate: number;
        errorRate: number;
        decisionQuality: number;
    };
    trends: {
        performance: 'improving' | 'stable' | 'degrading';
        resourceUsage: 'increasing' | 'stable' | 'decreasing';
        reliability: 'improving' | 'stable' | 'degrading';
    };
}

/**
 * Decision outcome structure
 */
export interface DecisionOutcome {
    decisionId: string;
    swarmId: string;
    timestamp: Date;
    decision: string;
    outcome: 'successful' | 'failed' | 'partial';
    impact: {
        goalsAdvanced: number;
        resourcesSaved: number;
        timeReduced: number;
    };
    feedback?: string;
}

/**
 * MetacognitiveMonitorAdapter - Routes Tier 1 metacognitive monitoring through unified service
 */
export class MetacognitiveMonitorAdapter extends BaseMonitoringAdapter {
    private performanceHistory: Map<string, PerformanceSnapshot[]> = new Map();
    private decisionHistory: Map<string, DecisionOutcome[]> = new Map();

    constructor(
        eventBus: EventBus,
        logger: Logger
    ) {
        super(
            {
                componentName: "MetacognitiveMonitor",
                tier: 1,
                eventChannels: ["swarm.performance", "swarm.decisions"],
                enableEventBus: true,
            },
            logger,
            eventBus
        );
    }

    /**
     * Collect performance data from a swarm execution
     */
    async collectPerformanceData(input: PerformanceAnalysisInput): Promise<void> {
        // Calculate metrics
        const totalDecisions = input.outcomes.decisionsApproved + input.outcomes.decisionsRejected;
        const decisionApprovalRate = totalDecisions > 0 ? 
            input.outcomes.decisionsApproved / totalDecisions : 0;
        
        const totalGoals = input.outcomes.goalsAchieved + input.outcomes.goalsFailed;
        const goalSuccessRate = totalGoals > 0 ? 
            input.outcomes.goalsAchieved / totalGoals : 0;
        
        const resourceEfficiency = input.resourcesUsed.credits > 0 ?
            (input.outcomes.goalsAchieved * 100) / input.resourcesUsed.credits : 0;
        
        const errorRate = input.errors.length / Math.max(1, totalDecisions);
        
        // Record metrics to unified service
        await this.recordMetric(
            'swarm.execution_time',
            input.executionTime,
            'performance',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.resource_efficiency',
            resourceEfficiency,
            'performance',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.decision_approval_rate',
            decisionApprovalRate,
            'business',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.goal_success_rate',
            goalSuccessRate,
            'business',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.error_rate',
            errorRate,
            'health',
            { 
                swarmId: input.swarmId,
                errorCount: input.errors.length,
                highSeverityCount: input.errors.filter(e => e.severity === 'high').length,
            }
        );

        // Record resource usage
        await this.recordMetric(
            'swarm.agents_used',
            input.resourcesUsed.agents,
            'performance',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.credits_used',
            input.resourcesUsed.credits,
            'business',
            { swarmId: input.swarmId }
        );

        await this.recordMetric(
            'swarm.decisions_made',
            input.resourcesUsed.decisions,
            'business',
            { swarmId: input.swarmId }
        );

        // Create performance snapshot
        const snapshot: PerformanceSnapshot = {
            timestamp: new Date(),
            swarmId: input.swarmId,
            metrics: {
                executionTime: input.executionTime,
                resourceEfficiency,
                successRate: goalSuccessRate,
                errorRate,
                decisionQuality: decisionApprovalRate,
            },
            trends: this.analyzeTrends(input.swarmId, {
                performance: goalSuccessRate,
                resourceUsage: input.resourcesUsed.credits,
                reliability: 1 - errorRate,
            }),
        };

        // Store snapshot
        if (!this.performanceHistory.has(input.swarmId)) {
            this.performanceHistory.set(input.swarmId, []);
        }
        this.performanceHistory.get(input.swarmId)!.push(snapshot);

        // Emit event for metacognitive agents
        await this.emitEvent('swarm.performance.analyzed', {
            swarmId: input.swarmId,
            snapshot,
            timestamp: new Date(),
        });

        // Log errors if any
        for (const error of input.errors) {
            await this.recordMetric(
                `swarm.error.${error.type}`,
                1,
                'health',
                {
                    swarmId: input.swarmId,
                    severity: error.severity,
                    message: error.message,
                    timestamp: error.timestamp,
                }
            );
        }
    }

    /**
     * Get performance snapshots for a swarm
     */
    getPerformanceSnapshots(swarmId: string): PerformanceSnapshot[] {
        return this.performanceHistory.get(swarmId) || [];
    }

    /**
     * Record a decision outcome
     */
    async recordDecisionOutcome(
        swarmId: string,
        decisionId: string,
        outcome: DecisionOutcome
    ): Promise<void> {
        // Store decision outcome
        if (!this.decisionHistory.has(swarmId)) {
            this.decisionHistory.set(swarmId, []);
        }
        this.decisionHistory.get(swarmId)!.push(outcome);

        // Record to unified service
        await this.recordMetric(
            `swarm.decision.${outcome.outcome}`,
            1,
            'business',
            {
                swarmId,
                decisionId,
                decision: outcome.decision,
                impact: outcome.impact,
                feedback: outcome.feedback,
            }
        );

        // Record impact metrics
        await this.recordMetric(
            'swarm.goals_advanced',
            outcome.impact.goalsAdvanced,
            'business',
            { swarmId, decisionId }
        );

        await this.recordMetric(
            'swarm.resources_saved',
            outcome.impact.resourcesSaved,
            'business',
            { swarmId, decisionId }
        );

        await this.recordMetric(
            'swarm.time_reduced',
            outcome.impact.timeReduced,
            'performance',
            { swarmId, decisionId }
        );

        // Emit event for learning
        await this.emitEvent('swarm.decision.outcome', {
            swarmId,
            decisionId,
            outcome,
            timestamp: new Date(),
        });
    }

    /**
     * Request performance analysis (triggers metacognitive review)
     */
    async requestPerformanceAnalysis(swarmId: string): Promise<void> {
        const snapshots = this.getPerformanceSnapshots(swarmId);
        
        if (snapshots.length === 0) {
            this.logger?.warn(`[MetacognitiveMonitor] No performance data for swarm ${swarmId}`);
            return;
        }

        // Emit event to trigger metacognitive analysis
        await this.emitEvent('swarm.performance.review_requested', {
            swarmId,
            snapshots,
            decisionHistory: this.decisionHistory.get(swarmId) || [],
            timestamp: new Date(),
        });
    }

    /**
     * Handle incoming events
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        switch (channel) {
            case 'swarm.performance':
                // Handle performance events from swarms
                if (event.type === 'execution_completed') {
                    await this.collectPerformanceData(event.data);
                }
                break;
                
            case 'swarm.decisions':
                // Handle decision events from swarms
                if (event.type === 'decision_outcome') {
                    await this.recordDecisionOutcome(
                        event.swarmId,
                        event.decisionId,
                        event.outcome
                    );
                }
                break;
        }
    }

    /**
     * Analyze trends for a swarm
     */
    private analyzeTrends(
        swarmId: string,
        currentMetrics: { performance: number; resourceUsage: number; reliability: number }
    ): PerformanceSnapshot['trends'] {
        const history = this.performanceHistory.get(swarmId) || [];
        
        if (history.length < 3) {
            return {
                performance: 'stable',
                resourceUsage: 'stable',
                reliability: 'stable',
            };
        }

        // Get recent history
        const recent = history.slice(-3);
        
        // Analyze performance trend
        const perfValues = recent.map(s => s.metrics.successRate);
        perfValues.push(currentMetrics.performance);
        const perfTrend = this.calculateTrend(perfValues);
        
        // Analyze resource usage trend
        const resourceValues = recent.map(s => s.metrics.resourceEfficiency);
        resourceValues.push(currentMetrics.resourceUsage);
        const resourceTrend = this.calculateTrend(resourceValues);
        
        // Analyze reliability trend
        const reliabilityValues = recent.map(s => 1 - s.metrics.errorRate);
        reliabilityValues.push(currentMetrics.reliability);
        const reliabilityTrend = this.calculateTrend(reliabilityValues);

        return {
            performance: perfTrend > 0.1 ? 'improving' : perfTrend < -0.1 ? 'degrading' : 'stable',
            resourceUsage: resourceTrend > 0.1 ? 'increasing' : resourceTrend < -0.1 ? 'decreasing' : 'stable',
            reliability: reliabilityTrend > 0.1 ? 'improving' : reliabilityTrend < -0.1 ? 'degrading' : 'stable',
        };
    }

    /**
     * Calculate trend from values (positive = increasing, negative = decreasing)
     */
    private calculateTrend(values: number[]): number {
        if (values.length < 2) return 0;
        
        let sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        const n = values.length;
        
        for (let i = 0; i < n; i++) {
            sumX += i;
            sumY += values[i];
            sumXY += i * values[i];
            sumX2 += i * i;
        }
        
        // Calculate slope of linear regression
        const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);
        return slope;
    }
}
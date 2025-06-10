/**
 * Strategy Performance Event Schemas
 * 
 * Defines the standardized event types for strategy performance monitoring and learning.
 * These events enable event-driven learning through specialized agent swarms rather than
 * direct strategy modification.
 */

import { type BaseEvent } from "../types/events.js";
import { type StrategyType } from "../types/strategies.js";

/**
 * Base strategy event interface
 */
export interface BaseStrategyEvent extends BaseEvent {
    source: {
        tier: 3;
        component: "strategy-execution";
        instanceId: string; // Strategy name
    };
    data: {
        strategyType: StrategyType;
        strategyName: string;
        strategyVersion: string;
    };
}

/**
 * Strategy execution completed event
 * Emitted after every successful strategy execution
 */
export interface StrategyPerformanceEvent extends BaseStrategyEvent {
    type: "strategy/performance/completed";
    data: BaseStrategyEvent["data"] & {
        execution: {
            stepId: string;
            stepType: string;
            success: true;
            executionTime: number;
            resourceUsage: {
                credits: number;
                tokens: number;
                apiCalls: number;
                cost: number;
            };
            confidence: number;
            outputs: unknown;
        };
        context: {
            stepType: string;
            inputComplexity: number;
            constraintCount: number;
            historyDepth: number;
        };
    };
}

/**
 * Strategy execution failed event
 * Emitted after every failed strategy execution
 */
export interface StrategyFailureEvent extends BaseStrategyEvent {
    type: "strategy/performance/failed";
    data: BaseStrategyEvent["data"] & {
        execution: {
            stepId: string;
            stepType: string;
            success: false;
            executionTime: number;
            resourceUsage: {
                credits: number;
                tokens: number;
                apiCalls: number;
                cost: number;
            };
            error: string;
        };
        context: {
            stepType: string;
            inputComplexity: number;
            constraintCount: number;
            historyDepth: number;
        };
    };
}

/**
 * Strategy feedback event
 * Emitted when external feedback is provided about strategy performance
 */
export interface StrategyFeedbackEvent extends BaseStrategyEvent {
    type: "strategy/feedback/received";
    data: BaseStrategyEvent["data"] & {
        feedback: {
            outcome: "success" | "partial" | "failure";
            userSatisfaction?: number; // 0-1
            performanceScore: number; // 0-1
            issues?: string[];
            improvements?: string[];
        };
        source: "user" | "system" | "agent";
    };
}

/**
 * Strategy threshold crossed event
 * Emitted when strategy performance crosses significant thresholds
 */
export interface StrategyThresholdEvent extends BaseStrategyEvent {
    type: "strategy/threshold/crossed";
    data: BaseStrategyEvent["data"] & {
        threshold: {
            type: "success_rate" | "execution_time" | "confidence" | "cost_efficiency";
            direction: "above" | "below";
            value: number;
            previous: number;
            significance: "low" | "medium" | "high";
        };
        context: {
            totalExecutions: number;
            recentPerformance: {
                successRate: number;
                averageExecutionTime: number;
                averageConfidence: number;
                averageCost: number;
            };
        };
    };
}

/**
 * Strategy evolution event
 * Emitted when a strategy shows significant improvement or degradation over time
 */
export interface StrategyEvolutionEvent extends BaseStrategyEvent {
    type: "strategy/evolution/detected";
    data: BaseStrategyEvent["data"] & {
        evolution: {
            direction: "improvement" | "degradation" | "stable";
            metric: "overall" | "success_rate" | "efficiency" | "confidence";
            magnitude: number; // Percentage change
            timeframe: {
                start: Date;
                end: Date;
                executionCount: number;
            };
        };
        comparison: {
            before: {
                successRate: number;
                averageExecutionTime: number;
                averageConfidence: number;
                averageCost: number;
            };
            after: {
                successRate: number;
                averageExecutionTime: number;
                averageConfidence: number;
                averageCost: number;
            };
        };
    };
}

/**
 * Strategy pattern discovered event
 * Emitted when patterns are detected in strategy performance data
 */
export interface StrategyPatternEvent extends BaseStrategyEvent {
    type: "strategy/pattern/discovered";
    data: BaseStrategyEvent["data"] & {
        pattern: {
            type: "step_type_affinity" | "context_sensitivity" | "resource_correlation" | "failure_clustering";
            description: string;
            confidence: number; // 0-1
            significance: "low" | "medium" | "high";
        };
        details: {
            stepTypes?: string[]; // For step_type_affinity
            contextFactors?: Array<{
                factor: string;
                correlation: number;
            }>; // For context_sensitivity
            resourcePatterns?: Array<{
                resource: string;
                optimalRange: { min: number; max: number };
            }>; // For resource_correlation
            failureClusters?: Array<{
                errorType: string;
                frequency: number;
                commonContext: Record<string, unknown>;
            }>; // For failure_clustering
        };
        actionableInsights: string[];
    };
}

/**
 * Strategy learning recommendation event
 * Emitted by learning agents with specific recommendations for strategy improvement
 */
export interface StrategyLearningEvent extends BaseStrategyEvent {
    type: "strategy/learning/recommendation";
    data: BaseStrategyEvent["data"] & {
        recommendation: {
            type: "parameter_adjustment" | "framework_selection" | "resource_optimization" | "context_adaptation";
            priority: "low" | "medium" | "high" | "critical";
            confidence: number; // 0-1
            expectedImpact: {
                metric: "success_rate" | "execution_time" | "confidence" | "cost";
                improvement: number; // Expected percentage improvement
            };
        };
        details: {
            currentState: Record<string, unknown>;
            recommendedChanges: Record<string, unknown>;
            reasoning: string;
            supportingEvidence: Array<{
                type: "execution_data" | "pattern_analysis" | "comparative_study";
                description: string;
                confidence: number;
            }>;
        };
        implementation: {
            mechanism: "configuration_update" | "prompt_adjustment" | "framework_selection" | "resource_allocation";
            complexity: "simple" | "moderate" | "complex";
            testingRequired: boolean;
        };
    };
}

/**
 * Union type for all strategy events
 */
export type StrategyEvent =
    | StrategyPerformanceEvent
    | StrategyFailureEvent  
    | StrategyFeedbackEvent
    | StrategyThresholdEvent
    | StrategyEvolutionEvent
    | StrategyPatternEvent
    | StrategyLearningEvent;

/**
 * Event type constants for subscription patterns
 */
export const StrategyEventTypes = {
    PERFORMANCE_COMPLETED: "strategy/performance/completed",
    PERFORMANCE_FAILED: "strategy/performance/failed",
    FEEDBACK_RECEIVED: "strategy/feedback/received",
    THRESHOLD_CROSSED: "strategy/threshold/crossed",
    EVOLUTION_DETECTED: "strategy/evolution/detected",
    PATTERN_DISCOVERED: "strategy/pattern/discovered",
    LEARNING_RECOMMENDATION: "strategy/learning/recommendation",
} as const;

/**
 * Event subscription patterns for learning agents
 */
export const StrategyEventPatterns = {
    // All strategy events
    ALL_STRATEGY_EVENTS: "strategy/*",
    
    // Performance events only
    PERFORMANCE_EVENTS: "strategy/performance/*",
    
    // Learning and improvement events
    LEARNING_EVENTS: "strategy/{feedback,threshold,evolution,pattern,learning}/*",
    
    // Failure analysis events
    FAILURE_EVENTS: "strategy/{performance/failed,pattern/discovered}",
    
    // Real-time performance monitoring
    REAL_TIME_MONITORING: "strategy/{performance/*,threshold/crossed}",
    
    // Strategy-specific events
    strategySpecific: (strategyType: StrategyType) => `strategy/*/${strategyType.toLowerCase()}`,
    
    // Context-specific events  
    contextSpecific: (stepType: string) => `strategy/*/stepType:${stepType}`,
} as const;

/**
 * Helper functions for event filtering and analysis
 */
export class StrategyEventUtils {
    /**
     * Check if an event is a strategy performance event
     */
    static isPerformanceEvent(event: BaseEvent): event is StrategyPerformanceEvent | StrategyFailureEvent {
        return event.type.startsWith("strategy/performance/");
    }

    /**
     * Check if an event is a strategy learning event
     */
    static isLearningEvent(event: BaseEvent): event is StrategyFeedbackEvent | StrategyLearningEvent {
        return event.type === "strategy/feedback/received" || event.type === "strategy/learning/recommendation";
    }

    /**
     * Extract strategy type from event
     */
    static getStrategyType(event: StrategyEvent): StrategyType {
        return event.data.strategyType;
    }

    /**
     * Extract execution context from performance events
     */
    static getExecutionContext(event: StrategyPerformanceEvent | StrategyFailureEvent): {
        stepType: string;
        complexity: number;
        success: boolean;
        resourceUsage: number;
    } {
        return {
            stepType: event.data.context.stepType,
            complexity: event.data.context.inputComplexity + event.data.context.constraintCount,
            success: event.data.execution.success,
            resourceUsage: event.data.execution.resourceUsage.cost,
        };
    }

    /**
     * Calculate performance score from event data
     */
    static calculatePerformanceScore(event: StrategyPerformanceEvent): number {
        const execution = event.data.execution;
        const baseScore = execution.success ? 0.6 : 0;
        const confidenceScore = execution.confidence * 0.3;
        const efficiencyScore = Math.max(0, 1 - (execution.executionTime / 30000)) * 0.1; // 30s baseline
        
        return Math.min(1, baseScore + confidenceScore + efficiencyScore);
    }

    /**
     * Extract error patterns from failure events
     */
    static extractErrorPattern(events: StrategyFailureEvent[]): {
        commonErrors: Map<string, number>;
        contextPatterns: Map<string, number>;
        timeDistribution: Map<string, number>;
    } {
        const commonErrors = new Map<string, number>();
        const contextPatterns = new Map<string, number>();
        const timeDistribution = new Map<string, number>();

        for (const event of events) {
            // Error type frequency
            const errorType = event.data.execution.error.split(' ')[0] || 'Unknown';
            commonErrors.set(errorType, (commonErrors.get(errorType) || 0) + 1);

            // Context pattern frequency
            const contextKey = `${event.data.context.stepType}-${event.data.context.inputComplexity}`;
            contextPatterns.set(contextKey, (contextPatterns.get(contextKey) || 0) + 1);

            // Time distribution
            const hour = event.timestamp.getHours();
            const timeKey = `${Math.floor(hour / 6) * 6}:00`; // 6-hour buckets
            timeDistribution.set(timeKey, (timeDistribution.get(timeKey) || 0) + 1);
        }

        return { commonErrors, contextPatterns, timeDistribution };
    }
}
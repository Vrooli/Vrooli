/**
 * Example: Strategy Metrics and Event-Driven Learning
 * 
 * This example demonstrates how the ReasoningStrategy now emits performance events
 * instead of implementing direct learning, enabling event-driven optimization through
 * specialized learning agent swarms.
 */

import { type Logger } from "winston";
import { StrategyType } from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/events/eventBus.js";
import { StrategyEventEmitter } from "../events/strategyEventEmitter.js";
import { StrategyMetricsStore } from "../metrics/strategyMetricsStore.js";

/**
 * Example learning agent that subscribes to strategy performance events
 */
class LearningAgent {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private patterns = new Map<string, number>();

    constructor(logger: Logger, eventBus: EventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Subscribe to strategy performance events
        this.setupSubscriptions();
    }

    private async setupSubscriptions(): Promise<void> {
        // Subscribe to all strategy performance events
        await this.eventBus.subscribe({
            id: "learning-agent-strategy-performance",
            pattern: "strategy/performance/*",
            handler: async (event) => {
                await this.analyzePerformanceEvent(event);
            },
        });

        // Subscribe to threshold events for significant changes
        await this.eventBus.subscribe({
            id: "learning-agent-strategy-thresholds", 
            pattern: "strategy/threshold/*",
            handler: async (event) => {
                await this.handleThresholdEvent(event);
            },
        });

        this.logger.info("[LearningAgent] Subscribed to strategy events");
    }

    private async analyzePerformanceEvent(event: any): Promise<void> {
        const { data } = event;
        
        // Extract patterns from successful executions
        if (data.execution.success) {
            const pattern = `${data.context.stepType}-${data.execution.confidence > 0.8 ? 'high' : 'low'}-confidence`;
            const count = this.patterns.get(pattern) || 0;
            this.patterns.set(pattern, count + 1);
            
            this.logger.debug("[LearningAgent] Analyzed performance pattern", {
                pattern,
                count: count + 1,
                executionTime: data.execution.executionTime,
                confidence: data.execution.confidence,
            });
        }
        
        // Detect patterns that indicate optimization opportunities
        if (this.patterns.size > 10) {
            await this.generateOptimizationRecommendations(data.strategyType);
        }
    }

    private async handleThresholdEvent(event: any): Promise<void> {
        const { data } = event;
        
        this.logger.info("[LearningAgent] Strategy threshold crossed", {
            strategyType: data.strategyType,
            threshold: data.threshold,
            significance: data.threshold.significance,
        });
        
        // Generate learning recommendations for significant performance changes
        if (data.threshold.significance === "high") {
            await this.generateUrgentRecommendation(data);
        }
    }

    private async generateOptimizationRecommendations(strategyType: StrategyType): Promise<void> {
        // Analyze patterns and generate recommendations
        const topPatterns = Array.from(this.patterns.entries())
            .sort((a, b) => b[1] - a[1])
            .slice(0, 3);

        const recommendation = {
            id: `learning-rec-${Date.now()}`,
            type: "strategy/learning/recommendation",
            timestamp: new Date(),
            source: {
                tier: 3 as const,
                component: "learning-agent",
                instanceId: "pattern-analyzer",
            },
            correlationId: `learning-${strategyType}`,
            data: {
                strategyType,
                strategyName: "ReasoningStrategy",
                strategyVersion: "2.0.0",
                recommendation: {
                    type: "framework_selection" as const,
                    priority: "medium" as const,
                    confidence: 0.75,
                    expectedImpact: {
                        metric: "success_rate" as const,
                        improvement: 0.15, // 15% improvement expected
                    },
                },
                details: {
                    currentState: { patterns: Object.fromEntries(this.patterns) },
                    recommendedChanges: {
                        optimizeFor: topPatterns.map(([pattern]) => pattern),
                        adjustParameters: {
                            confidence_threshold: 0.8,
                            framework_selection: "evidence_based",
                        },
                    },
                    reasoning: `Analysis of ${this.patterns.size} execution patterns suggests optimization opportunities`,
                    supportingEvidence: [
                        {
                            type: "pattern_analysis" as const,
                            description: `Top performing patterns: ${topPatterns.map(([p, c]) => `${p} (${c}x)`).join(", ")}`,
                            confidence: 0.8,
                        },
                    ],
                },
                implementation: {
                    mechanism: "configuration_update" as const,
                    complexity: "moderate" as const,
                    testingRequired: true,
                },
            },
        };

        // Emit learning recommendation
        await this.eventBus.publish(recommendation);
        
        this.logger.info("[LearningAgent] Generated optimization recommendation", {
            strategyType,
            topPatterns: topPatterns.length,
            expectedImprovement: "15%",
        });

        // Reset pattern tracking
        this.patterns.clear();
    }

    private async generateUrgentRecommendation(thresholdData: any): Promise<void> {
        const recommendation = {
            id: `urgent-rec-${Date.now()}`,
            type: "strategy/learning/recommendation",
            timestamp: new Date(),
            source: {
                tier: 3 as const,
                component: "learning-agent",
                instanceId: "threshold-monitor",
            },
            correlationId: `urgent-${thresholdData.strategyType}`,
            data: {
                strategyType: thresholdData.strategyType,
                strategyName: thresholdData.strategyName,
                strategyVersion: thresholdData.strategyVersion,
                recommendation: {
                    type: "parameter_adjustment" as const,
                    priority: "high" as const,
                    confidence: 0.9,
                    expectedImpact: {
                        metric: thresholdData.threshold.type,
                        improvement: 0.25, // 25% improvement expected
                    },
                },
                details: {
                    currentState: { threshold: thresholdData.threshold },
                    recommendedChanges: {
                        immediate: true,
                        adjustments: this.getThresholdAdjustments(thresholdData.threshold),
                    },
                    reasoning: `Significant ${thresholdData.threshold.direction} change in ${thresholdData.threshold.type} detected`,
                    supportingEvidence: [
                        {
                            type: "execution_data" as const,
                            description: `${thresholdData.threshold.type} changed from ${thresholdData.threshold.previous} to ${thresholdData.threshold.value}`,
                            confidence: 1.0,
                        },
                    ],
                },
                implementation: {
                    mechanism: "parameter_adjustment" as const,
                    complexity: "simple" as const,
                    testingRequired: false,
                },
            },
        };

        await this.eventBus.publish(recommendation);
        
        this.logger.warn("[LearningAgent] Generated urgent recommendation", {
            threshold: thresholdData.threshold,
        });
    }

    private getThresholdAdjustments(threshold: any): Record<string, any> {
        switch (threshold.type) {
            case "success_rate":
                return threshold.direction === "below" ? 
                    { retry_attempts: 2, fallback_strategy: "conversational" } :
                    { confidence_threshold: 0.9 };
            case "execution_time":
                return threshold.direction === "above" ?
                    { timeout_reduction: 0.8, parallel_processing: true } :
                    { resource_allocation: "increase" };
            case "confidence":
                return threshold.direction === "below" ?
                    { validation_strictness: "high", evidence_requirement: "increased" } :
                    { confidence_threshold: threshold.value + 0.1 };
            default:
                return { monitoring: "increased" };
        }
    }
}

/**
 * Example usage demonstrating the complete event-driven learning flow
 */
export async function demonstrateEventDrivenLearning(logger: Logger): Promise<void> {
    logger.info("[StrategyMetricsExample] Starting event-driven learning demonstration");

    // 1. Set up event bus
    const eventBus = new EventBus(logger);
    await eventBus.start();

    // 2. Create learning agent that will optimize strategies
    const learningAgent = new LearningAgent(logger, eventBus);

    // 3. Create strategy metrics infrastructure
    const metricsStore = new StrategyMetricsStore(logger);
    const eventEmitter = new StrategyEventEmitter(logger, eventBus);

    // 4. Simulate strategy executions with performance events
    logger.info("[StrategyMetricsExample] Simulating strategy executions...");
    
    for (let i = 0; i < 15; i++) {
        const success = Math.random() > 0.2; // 80% success rate
        const executionTime = 1000 + Math.random() * 5000; // 1-6 seconds
        const confidence = success ? 0.7 + Math.random() * 0.3 : 0; // 0.7-1.0 if successful
        
        // Record execution in metrics store
        metricsStore.recordExecution({
            stepId: `step-${i}`,
            stepType: ["analyze", "evaluate", "decide", "synthesize"][Math.floor(Math.random() * 4)],
            success,
            executionTime,
            tokensUsed: Math.floor(500 + Math.random() * 1000),
            apiCalls: Math.floor(1 + Math.random() * 3),
            cost: (500 + Math.random() * 1000) * 0.001,
            confidence,
            timestamp: new Date(),
            context: {
                inputSize: Math.floor(1 + Math.random() * 5),
                constraintsCount: Math.floor(0 + Math.random() * 3),
                historySize: Math.floor(0 + Math.random() * 10),
            },
        });

        // Emit performance event
        if (success) {
            await eventEmitter.emitStrategyPerformance({
                strategyType: StrategyType.REASONING,
                strategyName: "ReasoningStrategy",
                strategyVersion: "2.0.0",
                execution: {
                    stepId: `step-${i}`,
                    stepType: ["analyze", "evaluate", "decide", "synthesize"][Math.floor(Math.random() * 4)],
                    success: true,
                    executionTime,
                    resourceUsage: {
                        credits: Math.floor(500 + Math.random() * 1000),
                        tokens: Math.floor(500 + Math.random() * 1000),
                        apiCalls: Math.floor(1 + Math.random() * 3),
                        cost: (500 + Math.random() * 1000) * 0.001,
                    },
                    confidence,
                    outputs: { result: `Analysis result ${i}` },
                },
                context: {
                    stepType: ["analyze", "evaluate", "decide", "synthesize"][Math.floor(Math.random() * 4)],
                    inputComplexity: Math.floor(1 + Math.random() * 5),
                    constraintCount: Math.floor(0 + Math.random() * 3),
                    historyDepth: Math.floor(0 + Math.random() * 10),
                },
                timestamp: new Date(),
            });
        } else {
            await eventEmitter.emitStrategyFailure({
                strategyType: StrategyType.REASONING,
                strategyName: "ReasoningStrategy",
                strategyVersion: "2.0.0",
                execution: {
                    stepId: `step-${i}`,
                    stepType: ["analyze", "evaluate", "decide", "synthesize"][Math.floor(Math.random() * 4)],
                    success: false,
                    executionTime,
                    resourceUsage: {
                        credits: Math.floor(500 + Math.random() * 1000),
                        tokens: Math.floor(500 + Math.random() * 1000),
                        apiCalls: Math.floor(1 + Math.random() * 3),
                        cost: (500 + Math.random() * 1000) * 0.001,
                    },
                    error: ["Timeout", "ValidationError", "InsufficientData", "ModelError"][Math.floor(Math.random() * 4)],
                },
                context: {
                    stepType: ["analyze", "evaluate", "decide", "synthesize"][Math.floor(Math.random() * 4)],
                    inputComplexity: Math.floor(1 + Math.random() * 5),
                    constraintCount: Math.floor(0 + Math.random() * 3),
                    historyDepth: Math.floor(0 + Math.random() * 10),
                },
                timestamp: new Date(),
            });
        }

        // Small delay between executions
        await new Promise(resolve => setTimeout(resolve, 10));
    }

    // 5. Check final metrics
    const finalMetrics = metricsStore.getAggregatedMetrics();
    logger.info("[StrategyMetricsExample] Final metrics", {
        totalExecutions: finalMetrics.totalExecutions,
        successRate: finalMetrics.successCount / finalMetrics.totalExecutions,
        averageExecutionTime: finalMetrics.averageExecutionTime,
        averageConfidence: finalMetrics.averageConfidence,
    });

    // 6. Check threshold events  
    await eventEmitter.checkThresholds({
        successRate: finalMetrics.successCount / finalMetrics.totalExecutions,
        averageExecutionTime: finalMetrics.averageExecutionTime,
        averageConfidence: finalMetrics.averageConfidence,
        averageCost: finalMetrics.averageCost,
        totalExecutions: finalMetrics.totalExecutions,
    }, {
        type: StrategyType.REASONING,
        name: "ReasoningStrategy",
        version: "2.0.0",
    });

    // 7. Show event bus metrics
    const eventMetrics = eventBus.getMetrics();
    logger.info("[StrategyMetricsExample] Event bus metrics", {
        totalEvents: eventMetrics.totalEvents,
        activeSubscriptions: eventMetrics.activeSubscriptions,
        eventTypes: Array.from(eventMetrics.eventTypes.entries()),
    });

    // 8. Cleanup
    await eventBus.stop();
    
    logger.info("[StrategyMetricsExample] Demonstration complete - learning agents processed events and generated recommendations");
}

/**
 * Key benefits of this event-driven approach:
 * 
 * 1. **Separation of Concerns**: Strategies focus on execution, learning agents focus on optimization
 * 2. **Emergent Intelligence**: Multiple learning agents can compete and collaborate
 * 3. **Flexible Learning**: Different teams can deploy different learning strategies
 * 4. **Real-time Adaptation**: Changes happen through configuration updates, not code changes
 * 5. **Auditable Learning**: All learning decisions are captured as events
 * 6. **Scalable Architecture**: Learning agents can be distributed and specialized
 */
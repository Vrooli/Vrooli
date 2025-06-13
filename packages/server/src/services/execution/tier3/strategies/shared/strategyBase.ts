/**
 * Minimal Strategy Base - Event-driven execution strategies
 * 
 * This base class provides ONLY core strategy execution functionality.
 * All performance tracking, optimization, and learning capabilities
 * emerge from strategy optimization agents.
 * 
 * IMPORTANT: This component does NOT:
 * - Track performance metrics
 * - Optimize execution paths
 * - Learn from past executions
 * - Make improvement decisions
 * - Store execution history
 * 
 * It ONLY executes strategies and emits events.
 */

import { type Logger } from "winston";
import {
    type ExecutionContext as StrategyExecutionContext,
    type ExecutionStrategy,
    type StrategyExecutionResult,
    type StrategyFeedback,
    type StrategyPerformance,
    type ResourceUsage,
    StrategyType,
} from "@vrooli/shared";
import { type EventBus } from "../../../cross-cutting/events/eventBus.js";
import { ExecutionEventEmitter, ComponentEventEmitter } from "../../../cross-cutting/monitoring/ExecutionEventEmitter.js";
import { ErrorHandler, ComponentErrorHandler } from "../../../shared/ErrorHandler.js";

/**
 * Minimal strategy configuration
 */
export interface MinimalStrategyConfig {
    maxRetries?: number;
    timeoutMs?: number;
}

/**
 * Minimal strategy execution metadata
 */
export interface MinimalExecutionMetadata {
    startTime: Date;
    endTime?: Date;
    retryCount: number;
    errors: string[];
}

/**
 * Minimal Strategy Base
 * 
 * Provides basic strategy execution with event emission.
 * Strategy optimization agents subscribe to events to provide
 * performance tracking, pattern detection, and learning.
 */
export abstract class MinimalStrategyBase implements ExecutionStrategy {
    abstract readonly type: StrategyType;
    abstract readonly name: string;
    abstract readonly version: string;

    protected readonly eventEmitter: ComponentEventEmitter;
    protected readonly errorHandler: ComponentErrorHandler;
    protected readonly config: MinimalStrategyConfig;

    constructor(
        protected readonly logger: Logger,
        eventBus: EventBus,
        config: MinimalStrategyConfig = {}
    ) {
        this.config = {
            maxRetries: config.maxRetries || 3,
            timeoutMs: config.timeoutMs || 300000, // 5 minutes
        };
        
        const executionEmitter = new ExecutionEventEmitter(logger, eventBus);
        this.eventEmitter = executionEmitter.createComponentEmitter(3, `strategy:${this.name}`);
        
        const errorHandler = new ErrorHandler(logger, executionEmitter.eventPublisher);
        this.errorHandler = errorHandler.createComponentHandler(`Strategy:${this.name}`);
    }

    /**
     * Execute strategy with basic retry logic and event emission
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        const executionId = `${this.name}:${context.stepId}:${Date.now()}`;
        const metadata: MinimalExecutionMetadata = {
            startTime: new Date(),
            retryCount: 0,
            errors: [],
        };

        // Emit execution start
        await this.emitExecutionEvent(executionId, "started", context, metadata);

        let lastError: Error | undefined;
        let result: StrategyExecutionResult | undefined;

        // Basic retry loop
        for (let attempt = 0; attempt < this.config.maxRetries!; attempt++) {
            metadata.retryCount = attempt;

            try {
                // Strategy-specific execution
                result = await this.executeStrategy(context, metadata);
                
                // Success - emit and return
                metadata.endTime = new Date();
                await this.emitExecutionEvent(executionId, "completed", context, metadata, result);
                return result;

            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));
                metadata.errors.push(lastError.message);
                
                // Emit retry event
                if (attempt < this.config.maxRetries! - 1) {
                    await this.emitExecutionEvent(executionId, "retrying", context, metadata, undefined, lastError);
                    
                    // Basic exponential backoff
                    await this.delay(Math.pow(2, attempt) * 1000);
                }
            }
        }

        // All retries failed
        metadata.endTime = new Date();
        await this.emitExecutionEvent(executionId, "failed", context, metadata, undefined, lastError);
        
        throw lastError || new Error("Strategy execution failed");
    }

    /**
     * Process feedback - just emit for agents to analyze
     */
    async processFeedback(feedback: StrategyFeedback): Promise<void> {
        await this.eventEmitter.emitMetric(
            "quality",
            "strategy.feedback",
            feedback.rating,
            "rating",
            {
                strategyType: this.type,
                strategyName: this.name,
                executionId: feedback.executionId,
                issues: JSON.stringify(feedback.issues || []),
                suggestions: JSON.stringify(feedback.suggestions || []),
            }
        );
        
        // Agents will analyze feedback and evolve strategies
    }

    /**
     * Get performance metrics - returns minimal data
     * Agents track detailed performance
     */
    async getPerformanceMetrics(): Promise<StrategyPerformance> {
        // Return minimal metrics
        // Real performance tracking happens in agents
        return {
            averageExecutionTime: 0,
            successRate: 0,
            averageResourceUsage: {
                cpu: 0,
                memory: 0,
                credits: 0,
            },
            totalExecutions: 0,
            lastExecutionTime: new Date(),
        };
    }

    /**
     * Abstract method for strategy-specific execution
     */
    protected abstract executeStrategy(
        context: StrategyExecutionContext,
        metadata: MinimalExecutionMetadata
    ): Promise<StrategyExecutionResult>;

    /**
     * Helper to calculate resource usage
     * Just estimates, agents will track real usage
     */
    protected calculateResourceUsage(
        startTime: Date,
        endTime: Date,
        baseCredits: number
    ): ResourceUsage {
        const duration = endTime.getTime() - startTime.getTime();
        return {
            cpu: duration * 0.001, // Simple estimate
            memory: 100, // Fixed estimate
            credits: baseCredits,
        };
    }

    /**
     * Helper for delay
     */
    protected delay(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Emit execution event
     */
    private async emitExecutionEvent(
        executionId: string,
        event: "started" | "completed" | "failed" | "retrying",
        context: StrategyExecutionContext,
        metadata: MinimalExecutionMetadata,
        result?: StrategyExecutionResult,
        error?: Error
    ): Promise<void> {
        const duration = metadata.endTime 
            ? metadata.endTime.getTime() - metadata.startTime.getTime()
            : Date.now() - metadata.startTime.getTime();

        await this.eventEmitter.emitExecutionEvent(executionId, event as any, {
            strategyType: this.type,
            strategyName: this.name,
            strategyVersion: this.version,
            stepId: context.stepId,
            routineId: context.routine?.id,
            duration,
            retryCount: metadata.retryCount,
            errors: metadata.errors,
            result: result ? {
                success: result.success,
                outputCount: result.outputs ? Object.keys(result.outputs).length : 0,
                resourcesUsed: result.resourcesUsed,
            } : undefined,
            error: error ? {
                message: error.message,
                type: error.constructor.name,
            } : undefined,
        });
    }
}

/**
 * Example strategy optimization agent:
 * 
 * ```typescript
 * // This would be a routine deployed by teams, NOT hardcoded
 * class StrategyOptimizationAgent {
 *     private executionHistory = new Map<string, ExecutionRecord[]>();
 *     
 *     async onExecutionCompleted(event: ExecutionEvent) {
 *         const { strategyName, duration, resourcesUsed } = event.data;
 *         
 *         // Track execution patterns
 *         this.trackExecution(strategyName, { duration, resourcesUsed });
 *         
 *         // Analyze for optimization opportunities
 *         const patterns = this.analyzePatterns(strategyName);
 *         
 *         if (patterns.shouldOptimize) {
 *             await this.proposeOptimization(strategyName, patterns);
 *         }
 *     }
 *     
 *     async onStrategyFeedback(event: MetricEvent) {
 *         // Learn from user feedback
 *         // Evolve strategy selection logic
 *     }
 * }
 * ```
 */
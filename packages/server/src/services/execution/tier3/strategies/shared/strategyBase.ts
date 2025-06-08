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
import { PerformanceTracker, type PerformanceEntry } from "./performanceTracker.js";

/**
 * Strategy configuration options
 */
export interface StrategyConfig {
    enablePerformanceTracking?: boolean;
    maxRetries?: number;
    timeoutMs?: number;
    learningEnabled?: boolean;
}

/**
 * Strategy execution metadata
 */
export interface ExecutionMetadata {
    startTime: Date;
    endTime?: Date;
    retryCount: number;
    errors: string[];
    warnings: string[];
}

/**
 * Abstract base class for execution strategies
 * Extracts common patterns from ConversationalStrategy and DeterministicStrategy
 */
export abstract class StrategyBase implements ExecutionStrategy {
    abstract readonly type: StrategyType;
    abstract readonly name: string;
    abstract readonly version: string;

    protected readonly logger: Logger;
    protected readonly config: StrategyConfig;
    protected readonly performanceTracker: PerformanceTracker;

    // Common configuration
    protected static readonly DEFAULT_MAX_RETRIES = 3;
    protected static readonly DEFAULT_TIMEOUT_MS = 300000; // 5 minutes
    protected static readonly RETRY_DELAY_MS = 1000;

    constructor(
        logger: Logger,
        config: StrategyConfig = {}
    ) {
        this.logger = logger;
        this.config = {
            enablePerformanceTracking: true,
            maxRetries: StrategyBase.DEFAULT_MAX_RETRIES,
            timeoutMs: StrategyBase.DEFAULT_TIMEOUT_MS,
            learningEnabled: true,
            ...config,
        };
        
        this.performanceTracker = new PerformanceTracker();
    }

    /**
     * Main execution method - implements common execution lifecycle
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        const metadata: ExecutionMetadata = {
            startTime: new Date(),
            retryCount: 0,
            errors: [],
            warnings: [],
        };

        let lastError: Error | undefined;
        let result: StrategyExecutionResult | undefined;

        // Retry loop with exponential backoff
        for (let attempt = 0; attempt < this.config.maxRetries!; attempt++) {
            metadata.retryCount = attempt;

            try {
                this.logger.debug(`[${this.name}] Execution attempt ${attempt + 1}`, {
                    stepId: context.stepId,
                    maxRetries: this.config.maxRetries,
                });

                // Strategy-specific execution
                result = await this.executeStrategy(context, metadata);
                
                // Success - break retry loop
                break;

            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));
                metadata.errors.push(lastError.message);

                this.logger.warn(`[${this.name}] Execution attempt ${attempt + 1} failed`, {
                    stepId: context.stepId,
                    error: lastError.message,
                    attempt: attempt + 1,
                    maxRetries: this.config.maxRetries,
                });

                // If not the last attempt, wait before retrying
                if (attempt < this.config.maxRetries! - 1) {
                    const delay = StrategyBase.RETRY_DELAY_MS * Math.pow(2, attempt);
                    await this.sleep(delay);
                }
            }
        }

        metadata.endTime = new Date();
        const executionTime = metadata.endTime.getTime() - metadata.startTime.getTime();

        // If we exhausted retries without success, return failure
        if (!result) {
            result = this.createFailureResult(context, lastError, metadata);
        }

        // Record performance if enabled
        if (this.config.enablePerformanceTracking) {
            this.recordPerformance(result, executionTime, metadata);
        }

        return result;
    }

    /**
     * Strategy-specific execution logic - to be implemented by subclasses
     */
    protected abstract executeStrategy(
        context: StrategyExecutionContext,
        metadata: ExecutionMetadata
    ): Promise<StrategyExecutionResult>;

    /**
     * Get performance feedback for learning and adaptation
     */
    getPerformance(): StrategyPerformance {
        const metrics = this.performanceTracker.getMetrics();
        const feedback = this.performanceTracker.generateFeedback();

        return {
            successRate: metrics.successRate,
            averageExecutionTime: metrics.averageExecutionTime,
            averageConfidence: metrics.averageConfidence,
            totalExecutions: metrics.totalExecutions,
            adaptationNeeded: feedback.shouldAdapt,
            optimizationPotential: feedback.optimizationPotential,
            recommendations: feedback.recommendations,
        };
    }

    /**
     * Apply feedback to improve strategy performance
     */
    async applyFeedback(feedback: StrategyFeedback): Promise<void> {
        this.logger.info(`[${this.name}] Applying performance feedback`, {
            successRate: feedback.successRate,
            recommendation: feedback.recommendation,
        });

        // Apply strategy-specific optimizations
        await this.applyStrategySpecificFeedback(feedback);
    }

    /**
     * Strategy-specific feedback application - to be implemented by subclasses
     */
    protected async applyStrategySpecificFeedback(feedback: StrategyFeedback): Promise<void> {
        // Default implementation - subclasses can override
        this.logger.debug(`[${this.name}] No strategy-specific feedback application implemented`);
    }

    /**
     * Record performance metrics
     */
    protected recordPerformance(
        result: StrategyExecutionResult,
        executionTime: number,
        metadata: ExecutionMetadata
    ): void {
        const entry: PerformanceEntry = {
            timestamp: metadata.startTime,
            executionTime,
            success: result.success,
            confidence: result.confidence || 0,
            metadata: {
                retryCount: metadata.retryCount,
                errorCount: metadata.errors.length,
                warningCount: metadata.warnings.length,
                strategy: this.name,
                version: this.version,
            },
        };

        this.performanceTracker.recordPerformance(entry);
    }

    /**
     * Create a failure result with common error handling
     */
    protected createFailureResult(
        context: StrategyExecutionContext,
        error: Error | undefined,
        metadata: ExecutionMetadata
    ): StrategyExecutionResult {
        const executionTime = metadata.endTime!.getTime() - metadata.startTime.getTime();

        return {
            success: false,
            outputs: {},
            resourceUsage: this.calculateResourceUsage(context, executionTime, false),
            confidence: 0,
            metadata: {
                strategy: this.name,
                version: this.version,
                executionTime,
                retryCount: metadata.retryCount,
                errors: metadata.errors,
                failureReason: error?.message || 'Unknown error',
            },
        };
    }

    /**
     * Calculate resource usage - can be overridden by strategies
     */
    protected calculateResourceUsage(
        context: StrategyExecutionContext,
        executionTime: number,
        success: boolean
    ): ResourceUsage {
        // Basic resource calculation - strategies can override for more accurate tracking
        const baseCredits = success ? 10 : 5; // Success costs more due to completion
        const timeMultiplier = executionTime / 1000; // Per second
        
        return {
            credits: Math.ceil(baseCredits * Math.max(1, timeMultiplier * 0.1)),
            tokens: Math.ceil(timeMultiplier * 2), // Estimate token usage
            time: executionTime,
            memory: 50, // MB estimate
        };
    }

    /**
     * Sleep utility for retry delays
     */
    protected sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Common validation helper
     */
    protected validateContext(context: StrategyExecutionContext): void {
        if (!context.stepId) {
            throw new Error("Context missing required stepId");
        }
        if (!context.stepType) {
            throw new Error("Context missing required stepType");
        }
    }

    /**
     * Get recent performance trend for quick decision making
     */
    protected getRecentPerformanceTrend() {
        return this.performanceTracker.getRecentTrend();
    }

    /**
     * Clear performance history (useful for major strategy changes)
     */
    protected clearPerformanceHistory(): void {
        this.performanceTracker.clearHistory();
    }
}
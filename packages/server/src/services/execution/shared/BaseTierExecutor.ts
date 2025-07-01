/**
 * Base Tier Executor with Standardized Error Handling
 * 
 * This abstract base class provides common functionality for all tier executors,
 * including standardized error handling, resource tracking, and logging patterns.
 * It ensures consistent behavior across the three-tier architecture.
 * 
 * ## Key Features
 * 
 * ### Standardized Error Handling
 * - Consistent error structure across all tiers with tier identification
 * - Automatic error context collection and logging
 * - Event emission for error tracking and monitoring
 * - Graceful handling of both synchronous and asynchronous errors
 * 
 * ### Resource Tracking
 * - Automatic resource usage tracking with ResourceTracker integration
 * - Real-time execution metrics collection
 * - Resource validation against allocation limits
 * - Hierarchical resource accounting support
 * 
 * ### Execution Lifecycle Management
 * - Automatic execution phase tracking (validation → execution → cleanup)
 * - Execution metrics with start/end times and success/failure status
 * - Temporary execution tracking with automatic cleanup
 * - Event emission for execution lifecycle events
 * 
 * ## Usage Pattern
 * 
 * ```typescript
 * export class TierThreeExecutor extends BaseTierExecutor {
 *     constructor(eventBus: EventBus) {
 *         super(logger, eventBus, "TierThreeExecutor", "tier3");
 *     }
 * 
 *     async execute<TInput, TOutput>(request: TierExecutionRequest<TInput>): Promise<ExecutionResult<TOutput>> {
 *         return this.executeWithErrorHandling(request, async (req) => {
 *             return this.executeImpl<TInput, TOutput>(req);
 *         });
 *     }
 * 
 *     protected async executeImpl<TInput, TOutput>(request: TierExecutionRequest<TInput>): Promise<ExecutionResult<TOutput>> {
 *         // Tier-specific implementation
 *         // Errors are automatically caught and standardized by base class
 *     }
 * }
 * ```
 * 
 * ## Error Context Enhancement
 * 
 * Subclasses can override these methods to provide tier-specific context:
 * - `getInputTypeName()` - Provide detailed input type information
 * - `getAdditionalErrorContext()` - Add tier-specific error context
 * - `getStrategyFromRequest()` - Extract execution strategy information
 * 
 * ## Event Emission
 * 
 * The base class automatically emits these events:
 * - `tier.execution.completed` - On successful execution
 * - `tier.execution.failed` - On execution failure
 * 
 * Events include tier identification, execution metrics, and error context.
 */

import {
    type CoreResourceAllocation,
    type ExecutionResult,
    ResourceAggregator,
    ResourceTracker,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { BaseComponent } from "./BaseComponent.js";

/**
 * Tier-specific error context
 */
export interface TierErrorContext {
    tier: "tier1" | "tier2" | "tier3";
    swarmId: string;
    inputType?: string;
    phase: "validation" | "execution" | "cleanup" | "delegation";
    additionalContext?: Record<string, unknown>;
}

/**
 * Standard execution metrics collected by all tiers
 */
export interface ExecutionMetrics {
    startTime: number;
    endTime?: number;
    duration?: number;
    resourceTracker: ResourceTracker;
    phase: string;
    success?: boolean;
    errorType?: string;
}

/**
 * Abstract base class for all tier executors
 */
export abstract class BaseTierExecutor extends BaseComponent {
    private readonly tierName: "tier1" | "tier2" | "tier3";
    private readonly componentName: string;
    protected readonly activeExecutions = new Map<string, ExecutionMetrics>();

    constructor(
        componentName: string,
        tierName: "tier1" | "tier2" | "tier3",
    ) {
        super(componentName);
        this.tierName = tierName;
        this.componentName = componentName;
    }

    /**
     * Execute with standardized error handling and resource tracking
     */
    protected async executeWithErrorHandling<TInput, TOutput>(
        request: TierExecutionRequest<TInput>,
        executeImpl: (request: TierExecutionRequest<TInput>) => Promise<ExecutionResult<TOutput>>,
    ): Promise<ExecutionResult<TOutput>> {
        const startTime = Date.now();
        const swarmId = request.context.swarmId;
        const resourceTracker = new ResourceTracker(request.allocation);

        // Track execution start
        this.activeExecutions.set(swarmId, {
            startTime,
            resourceTracker,
            phase: "validation",
        });

        try {
            // Phase 1: Input validation
            this.updateExecutionPhase(swarmId, "validation");
            this.validateRequest(request);

            // Phase 2: Execution
            this.updateExecutionPhase(swarmId, "execution");
            const result = await executeImpl(request);

            // Phase 3: Result validation and cleanup
            this.updateExecutionPhase(swarmId, "cleanup");
            this.validateResult(result, request);

            // Mark execution as successful
            const metrics = this.activeExecutions.get(swarmId)!;
            metrics.endTime = Date.now();
            metrics.duration = metrics.endTime - startTime;
            metrics.success = true;

            // Emit success event (don't let event publishing failure affect result)
            this.publishEvent("tier.execution.completed", {
                tier: this.tierName,
                swarmId,
                duration: metrics.duration,
                resourceUsage: result.resourcesUsed || resourceTracker.getCurrentUsage(),
            }, {
                deliveryGuarantee: "reliable",
                priority: "medium",
                tags: ["execution", "completion", this.tierName],
            }).catch(eventError => {
                logger.error(`[${this.componentName}] Failed to emit success event`, {
                    eventError: eventError instanceof Error ? eventError.message : String(eventError),
                });
            });

            return result;

        } catch (error) {
            return this.createStandardizedErrorResult<TOutput>(error, request, startTime);
        } finally {
            // Always clean up tracking
            this.cleanupExecution(swarmId);
        }
    }

    /**
     * Create standardized error result with tier identification and context
     */
    protected createStandardizedErrorResult<TOutput>(
        error: unknown,
        request: TierExecutionRequest<any>,
        startTime: number,
    ): ExecutionResult<TOutput> {
        const swarmId = request.context.swarmId;
        const duration = Date.now() - startTime;

        // Get current execution metrics if available
        const metrics = this.activeExecutions.get(swarmId);
        const resourceUsage = metrics?.resourceTracker.getCurrentUsage() || {
            creditsUsed: "0",
            durationMs: duration,
            memoryUsedMB: 0,
            stepsExecuted: 0,
        };

        // Update metrics
        if (metrics) {
            metrics.endTime = Date.now();
            metrics.duration = duration;
            metrics.success = false;
            metrics.errorType = error instanceof Error ? error.constructor.name : "UnknownError";
        }

        // Create error context
        const errorContext: TierErrorContext = {
            tier: this.tierName,
            swarmId,
            inputType: this.getInputTypeName(request.input),
            phase: metrics?.phase || "unknown",
            additionalContext: this.getAdditionalErrorContext(request, error),
        };

        // Log error with full context
        logger.error(`[${this.componentName}] Execution failed`, {
            ...errorContext,
            error: error instanceof Error ? error.message : String(error),
            stack: error instanceof Error ? error.stack : undefined,
            duration,
        });

        // Emit error event
        this.publishEvent("tier.execution.failed", {
            ...errorContext,
            error: error instanceof Error ? error.message : String(error),
            duration,
            resourceUsage,
        }, {
            deliveryGuarantee: "reliable",
            priority: "high",
            tags: ["execution", "error", this.tierName],
        }).catch(eventError => {
            logger.error(`[${this.componentName}] Failed to emit error event`, {
                eventError: eventError instanceof Error ? eventError.message : String(eventError),
            });
        });

        // Create standardized error result
        return {
            success: false,
            error: {
                code: `${this.tierName.toUpperCase()}_EXECUTION_FAILED`,
                message: error instanceof Error ? error.message : "Unknown error",
                tier: this.tierName,
                type: error instanceof Error ? error.constructor.name : "Error",
                phase: errorContext.phase,
                context: errorContext.additionalContext,
            },
            resourcesUsed: resourceUsage,
            duration,
            context: request.context,
            metadata: {
                tier: this.tierName,
                strategy: this.getStrategyFromRequest(request),
                version: "1.0.0",
                timestamp: new Date().toISOString(),
                errorContext,
            },
            confidence: 0.0,
            performanceScore: 0.0,
        };
    }

    /**
     * Validate request structure and content (override in subclasses)
     */
    protected validateRequest(request: TierExecutionRequest<any>): void {
        if (!request.context?.swarmId) {
            throw new Error("Missing execution ID in request context");
        }

        if (!request.allocation) {
            throw new Error("Missing resource allocation in request");
        }

        if (request.input === null || request.input === undefined) {
            throw new Error("Missing input in request");
        }
    }

    /**
     * Validate execution result (override in subclasses)
     */
    protected validateResult(result: ExecutionResult<any>, request: TierExecutionRequest<any>): void {
        if (result.success === undefined) {
            throw new Error("Execution result missing success flag");
        }

        if (!result.success && !result.error) {
            throw new Error("Failed execution result missing error details");
        }

        if (!result.resourcesUsed) {
            logger.warn(`[${this.componentName}] Execution result missing resource usage`, {
                swarmId: request.context.swarmId,
            });
        }
    }

    /**
     * Update execution phase for tracking
     */
    private updateExecutionPhase(swarmId: string, phase: string): void {
        const metrics = this.activeExecutions.get(swarmId);
        if (metrics) {
            metrics.phase = phase;
        }
    }

    /**
     * Clean up execution tracking
     */
    private cleanupExecution(swarmId: string): void {
        // Keep metrics for a short time for debugging, then remove
        setTimeout(() => {
            this.activeExecutions.delete(swarmId);
        }, 60_000); // 1 minute retention
    }

    /**
     * Get input type name for error context (override in subclasses)
     */
    protected getInputTypeName(input: unknown): string {
        return input?.constructor?.name || typeof input;
    }

    /**
     * Get additional error context specific to tier (override in subclasses)
     */
    protected getAdditionalErrorContext(
        request: TierExecutionRequest<any>,
        error: unknown,
    ): Record<string, unknown> {
        return {
            hasAllocation: !!request.allocation,
            hasOptions: !!request.options,
            inputKeys: typeof request.input === "object" && request.input !== null
                ? Object.keys(request.input)
                : [],
        };
    }

    /**
     * Extract strategy from request for error context (override in subclasses)
     */
    protected getStrategyFromRequest(request: TierExecutionRequest<any>): string {
        // Try common patterns to extract strategy
        if (typeof request.input === "object" && request.input !== null) {
            const input = request.input as any;
            if (input.strategy) return input.strategy;
            if (input.config?.strategy) return input.config.strategy;
        }
        if (request.options?.strategy) return request.options.strategy;
        return "unknown";
    }

    /**
     * Get current execution metrics for monitoring
     */
    public getExecutionMetrics(swarmId: string): ExecutionMetrics | undefined {
        return this.activeExecutions.get(swarmId);
    }

    /**
     * Get all active executions for tier monitoring
     */
    public getActiveExecutions(): Map<string, ExecutionMetrics> {
        return new Map(this.activeExecutions);
    }

    /**
     * Abstract method that subclasses must implement for tier-specific execution
     */
    protected abstract executeImpl<TInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>>;
}

/**
 * Utility functions for resource management across tiers
 */
export class TierResourceUtils {
    /**
     * Create child allocation from parent with validation
     */
    static createChildAllocation(
        parentAllocation: CoreResourceAllocation,
        childRatio: number, // 0.0 to 1.0
        strategy: "strict" | "elastic" | "best_effort" = "strict",
    ): CoreResourceAllocation {
        if (childRatio <= 0 || childRatio > 1) {
            throw new Error(`Invalid child allocation ratio: ${childRatio}. Must be between 0 and 1.`);
        }

        const parseCredits = (credits: string): bigint => {
            return credits === "unlimited" ? BigInt(Number.MAX_SAFE_INTEGER) : BigInt(credits);
        };

        const formatCredits = (credits: bigint): string => {
            return credits === BigInt(Number.MAX_SAFE_INTEGER) ? "unlimited" : credits.toString();
        };

        return {
            maxCredits: formatCredits(parseCredits(parentAllocation.maxCredits) * BigInt(Math.floor(childRatio * 100)) / 100n),
            maxDurationMs: Math.floor(parentAllocation.maxDurationMs * childRatio),
            maxMemoryMB: Math.floor(parentAllocation.maxMemoryMB * childRatio),
            maxConcurrentSteps: Math.floor(parentAllocation.maxConcurrentSteps * childRatio),
        };
    }

    /**
     * Validate allocation hierarchy
     */
    static validateAllocationHierarchy(
        childAllocation: CoreResourceAllocation,
        parentAllocation: CoreResourceAllocation,
    ): { isValid: boolean; violations: string[] } {
        return ResourceAggregator.validateAllocationHierarchy(childAllocation, parentAllocation);
    }
}

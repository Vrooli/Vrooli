/**
 * Resource Aggregation and Tracking Utilities
 * 
 * This module provides utilities for tracking and aggregating resource usage
 * across the three-tier execution hierarchy. It ensures proper resource
 * accounting and enables hierarchical resource management.
 * 
 * ## Purpose
 * 
 * The three-tier execution architecture requires sophisticated resource management
 * to ensure fair allocation, prevent resource exhaustion, and enable proper billing.
 * This module provides:
 * 
 * - **Hierarchical Resource Tracking**: Parent-child execution resource accounting
 * - **BigInt Credit Calculations**: Precise credit handling for micropayments
 * - **Resource Aggregation**: Combine usage across parallel and sequential executions
 * - **Limit Validation**: Prevent executions from exceeding allocated resources
 * - **Performance Monitoring**: Track resource efficiency and optimization opportunities
 * 
 * ## Architecture Integration
 * 
 * ### Tier 1 (Coordination Intelligence)
 * ```
 * Swarm Execution (Parent)
 * ├── Agent 1: Routine A (Child)
 * ├── Agent 2: Routine B (Child)  
 * └── Agent 3: Routine C (Child)
 * ```
 * 
 * ### Tier 2 (Process Intelligence)
 * ```
 * Routine Execution (Parent)
 * ├── Step 1: Data Collection (Child)
 * ├── Step 2: Analysis (Child)
 * └── Step 3: Report Generation (Child)
 * ```
 * 
 * ### Tier 3 (Execution Intelligence)
 * ```
 * Step Execution (Parent)
 * ├── Tool Call 1: API Request (Child)
 * ├── Tool Call 2: Data Transform (Child)
 * └── Tool Call 3: Output Validation (Child)
 * ```
 * 
 * ## Resource Types
 * 
 * ### Credits (BigInt String)
 * - **Purpose**: Micropayment tracking for LLM inference, API calls, compute time
 * - **Precision**: Arbitrary precision using BigInt for sub-cent calculations
 * - **Aggregation**: Always summed (cumulative cost across all operations)
 * - **Example**: "1000000" (1 million micro-credits = $1.00)
 * 
 * ### Duration (Milliseconds)
 * - **Purpose**: Execution time tracking and timeout enforcement
 * - **Parallel**: Maximum duration (concurrent execution)
 * - **Sequential**: Sum of durations (serial execution)
 * - **Real-time**: Continuously updated during execution
 * 
 * ### Memory (Megabytes)
 * - **Purpose**: Peak memory usage tracking
 * - **Aggregation**: Maximum across all child executions
 * - **Monitoring**: Prevents memory exhaustion in containerized environments
 * - **Allocation**: Reserved memory pools for child executions
 * 
 * ### Steps/Operations
 * - **Purpose**: Complexity and progress tracking
 * - **Aggregation**: Always summed across all executions
 * - **Metrics**: Used for performance analysis and optimization
 * 
 * ## Usage Examples
 * 
 * ### Basic Resource Tracking
 * ```typescript
 * const allocation = {
 *     maxCredits: "1000000", // 1M micro-credits
 *     maxDurationMs: 60000,   // 1 minute
 *     maxMemoryMB: 512,       // 512MB
 *     maxConcurrentSteps: 5   // 5 parallel operations
 * };
 * 
 * const tracker = new ResourceTracker(allocation);
 * 
 * // Add usage incrementally
 * const result = tracker.addUsage({
 *     creditsUsed: "50000",   // 50k micro-credits
 *     durationMs: 1000,       // 1 second
 *     memoryUsedMB: 128,      // 128MB peak
 *     stepsExecuted: 1
 * });
 * 
 * if (!result.success) {
 *     throw new Error(result.error);
 * }
 * ```
 * 
 * ### Hierarchical Resource Aggregation
 * ```typescript
 * const childUsages = [
 *     { creditsUsed: "100000", durationMs: 2000, memoryUsedMB: 256, stepsExecuted: 3 },
 *     { creditsUsed: "150000", durationMs: 1500, memoryUsedMB: 128, stepsExecuted: 2 },
 *     { creditsUsed: "75000",  durationMs: 3000, memoryUsedMB: 64,  stepsExecuted: 1 }
 * ];
 * 
 * // For parallel execution (max duration, peak memory)
 * const parallelAggregate = ResourceAggregator.aggregateUsage(childUsages, "parallel");
 * // Result: { creditsUsed: "325000", durationMs: 3000, memoryUsedMB: 256, stepsExecuted: 6 }
 * 
 * // For sequential execution (sum duration, peak memory)
 * const sequentialAggregate = ResourceAggregator.aggregateUsage(childUsages, "sequential");
 * // Result: { creditsUsed: "325000", durationMs: 6500, memoryUsedMB: 256, stepsExecuted: 6 }
 * ```
 * 
 * ### Resource Limit Validation
 * ```typescript
 * const validation = ResourceAggregator.wouldExceedAllocation(
 *     currentUsage,
 *     proposedAdditionalUsage,
 *     allocation
 * );
 * 
 * if (validation.wouldExceed) {
 *     console.log(`Would exceed: ${validation.exceededResources.join(", ")}`);
 *     console.log(`Projected usage:`, validation.projectedUsage);
 *     return; // Cancel execution
 * }
 * ```
 * 
 * ### Hierarchical Allocation Creation
 * ```typescript
 * const parentAllocation = { maxCredits: "1000000", maxDurationMs: 60000, maxMemoryMB: 2048, maxConcurrentSteps: 10 };
 * 
 * const childAllocation = ResourceAggregator.createHierarchicalAllocation(
 *     TierResourceUtils.createChildAllocation(parentAllocation, 0.3), // 30% of parent resources
 *     "parent-execution-123",
 *     parentAllocation,
 *     "elastic" // Allow some flexibility in resource usage
 * );
 * ```
 * 
 * ## Performance Considerations
 * 
 * ### BigInt Arithmetic
 * - All credit calculations use BigInt for precision
 * - String representation for JSON serialization
 * - Efficient arithmetic operations for large numbers
 * 
 * ### Memory Efficiency
 * - Minimal object creation during aggregation
 * - Early termination for limit validation
 * - Reuse of calculation results
 * 
 * ### Real-time Updates
 * - ResourceTracker provides live duration updates
 * - Incremental usage addition with validation
 * - Efficient remaining resource calculation
 */

import {
    type CoreResourceAllocation,
    type ExecutionResourceUsage,
} from "./core.js";

/**
 * Detailed resource usage breakdown by category
 */
export interface DetailedResourceUsage extends ExecutionResourceUsage {
    /** Breakdown by execution phase */
    phaseBreakdown?: {
        initialization: Partial<ExecutionResourceUsage>;
        execution: Partial<ExecutionResourceUsage>;
        finalization: Partial<ExecutionResourceUsage>;
    };

    /** Breakdown by resource type */
    typeBreakdown?: {
        computation: Partial<ExecutionResourceUsage>;
        storage: Partial<ExecutionResourceUsage>;
        network: Partial<ExecutionResourceUsage>;
        llmInference: Partial<ExecutionResourceUsage>;
    };

    /** Child execution resource usage (for aggregation) */
    childUsages?: Array<{
        executionId: string;
        usage: ExecutionResourceUsage;
    }>;
}

/**
 * Resource allocation with hierarchy tracking
 */
export interface HierarchicalResourceAllocation extends CoreResourceAllocation {
    /** Parent allocation context */
    parentAllocation?: {
        executionId: string;
        allocatedToChild: CoreResourceAllocation;
        remainingInParent: CoreResourceAllocation;
    };

    /** Reserved for child executions */
    reservedForChildren?: CoreResourceAllocation;

    /** Allocation strategy */
    strategy: "strict" | "elastic" | "best_effort";

    /** Allocation timestamp */
    allocatedAt: Date;
}

/**
 * Resource aggregator for hierarchical resource tracking
 */
export class ResourceAggregator {
    /**
     * Aggregate resource usage from multiple child executions
     * 
     * This handles different aggregation strategies for parallel vs sequential execution:
     * - Credits: Always sum (cumulative cost)
     * - Duration: Max for parallel, sum for sequential
     * - Memory: Max (peak usage)
     * - Steps: Always sum
     */
    static aggregateUsage(
        childUsages: ExecutionResourceUsage[],
        executionType: "parallel" | "sequential" = "parallel",
    ): ExecutionResourceUsage {
        if (childUsages.length === 0) {
            return this.createEmptyUsage();
        }

        const aggregated: ExecutionResourceUsage = {
            creditsUsed: this.sumBigIntStrings(childUsages.map(u => u.creditsUsed)),
            durationMs: executionType === "parallel"
                ? Math.max(...childUsages.map(u => u.durationMs))
                : childUsages.reduce((sum, u) => sum + u.durationMs, 0),
            memoryUsedMB: Math.max(...childUsages.map(u => u.memoryUsedMB || 0)),
            stepsExecuted: childUsages.reduce((sum, u) => sum + u.stepsExecuted, 0),
            toolCalls: childUsages.reduce((sum, u) => sum + u.toolCalls, 0),
        };

        return aggregated;
    }

    /**
     * Create detailed usage breakdown from child executions
     */
    static createDetailedUsage(
        ownUsage: ExecutionResourceUsage,
        childUsages: Array<{ executionId: string; usage: ExecutionResourceUsage }>,
    ): DetailedResourceUsage {
        const aggregatedChildUsage = this.aggregateUsage(
            childUsages.map(c => c.usage),
        );

        const totalUsage = this.addUsages(ownUsage, aggregatedChildUsage);

        return {
            ...totalUsage,
            childUsages,
            phaseBreakdown: {
                initialization: { durationMs: 0, creditsUsed: "0", memoryUsedMB: 0, stepsExecuted: 0 },
                execution: totalUsage,
                finalization: { durationMs: 0, creditsUsed: "0", memoryUsedMB: 0, stepsExecuted: 0 },
            },
        };
    }

    /**
     * Check if resource usage would exceed allocation
     */
    static wouldExceedAllocation(
        currentUsage: ExecutionResourceUsage,
        additionalUsage: ExecutionResourceUsage,
        allocation: CoreResourceAllocation,
    ): {
        wouldExceed: boolean;
        exceededResources: string[];
        projectedUsage: ExecutionResourceUsage;
    } {
        const projectedUsage = this.addUsages(currentUsage, additionalUsage);
        const exceededResources: string[] = [];

        // Check credits
        if (this.compareBigIntStrings(projectedUsage.creditsUsed, allocation.maxCredits) > 0) {
            exceededResources.push("credits");
        }

        // Check duration
        if (projectedUsage.durationMs > allocation.maxDurationMs) {
            exceededResources.push("duration");
        }

        // Check memory
        if ((projectedUsage.memoryUsedMB || 0) > allocation.maxMemoryMB) {
            exceededResources.push("memory");
        }

        return {
            wouldExceed: exceededResources.length > 0,
            exceededResources,
            projectedUsage,
        };
    }

    /**
     * Calculate remaining resources from allocation
     */
    static calculateRemaining(
        allocation: CoreResourceAllocation,
        usage: ExecutionResourceUsage,
    ): CoreResourceAllocation {
        return {
            maxCredits: this.subtractBigIntStrings(allocation.maxCredits, usage.creditsUsed),
            maxDurationMs: Math.max(0, allocation.maxDurationMs - usage.durationMs),
            maxMemoryMB: Math.max(0, allocation.maxMemoryMB - usage.memoryUsedMB),
            maxConcurrentSteps: allocation.maxConcurrentSteps, // Concurrency limit doesn't decrease
        };
    }

    /**
     * Create hierarchical allocation with parent tracking
     */
    static createHierarchicalAllocation(
        allocation: CoreResourceAllocation,
        parentExecutionId?: string,
        parentAllocation?: CoreResourceAllocation,
        strategy: "strict" | "elastic" | "best_effort" = "strict",
    ): HierarchicalResourceAllocation {
        const hierarchical: HierarchicalResourceAllocation = {
            ...allocation,
            strategy,
            allocatedAt: new Date(),
        };

        if (parentExecutionId && parentAllocation) {
            hierarchical.parentAllocation = {
                executionId: parentExecutionId,
                allocatedToChild: allocation,
                remainingInParent: this.subtractAllocations(parentAllocation, allocation),
            };
        }

        return hierarchical;
    }

    /**
     * Validate allocation hierarchy constraints
     */
    static validateAllocationHierarchy(
        childAllocation: CoreResourceAllocation,
        parentAllocation: CoreResourceAllocation,
    ): {
        isValid: boolean;
        violations: string[];
    } {
        const violations: string[] = [];

        // Check credits
        if (this.compareBigIntStrings(childAllocation.maxCredits, parentAllocation.maxCredits) > 0) {
            violations.push(`Child credits (${childAllocation.maxCredits}) exceed parent credits (${parentAllocation.maxCredits})`);
        }

        // Check duration
        if (childAllocation.maxDurationMs > parentAllocation.maxDurationMs) {
            violations.push(`Child duration (${childAllocation.maxDurationMs}ms) exceeds parent duration (${parentAllocation.maxDurationMs}ms)`);
        }

        // Check memory
        if (childAllocation.maxMemoryMB > parentAllocation.maxMemoryMB) {
            violations.push(`Child memory (${childAllocation.maxMemoryMB}MB) exceeds parent memory (${parentAllocation.maxMemoryMB}MB)`);
        }

        // Check concurrency
        if (childAllocation.maxConcurrentSteps > parentAllocation.maxConcurrentSteps) {
            violations.push(`Child concurrency (${childAllocation.maxConcurrentSteps}) exceeds parent concurrency (${parentAllocation.maxConcurrentSteps})`);
        }

        return {
            isValid: violations.length === 0,
            violations,
        };
    }

    // Utility methods for BigInt string arithmetic

    private static sumBigIntStrings(values: string[]): string {
        return values.reduce((sum, value) => {
            const a = BigInt(sum);
            const b = BigInt(value);
            return (a + b).toString();
        }, "0");
    }

    private static subtractBigIntStrings(a: string, b: string): string {
        const result = BigInt(a) - BigInt(b);
        return result < BigInt(0) ? "0" : result.toString();
    }

    private static compareBigIntStrings(a: string, b: string): number {
        const bigA = BigInt(a);
        const bigB = BigInt(b);
        if (bigA < bigB) return -1;
        if (bigA > bigB) return 1;
        return 0;
    }

    private static addUsages(a: ExecutionResourceUsage, b: ExecutionResourceUsage): ExecutionResourceUsage {
        return {
            creditsUsed: this.sumBigIntStrings([a.creditsUsed, b.creditsUsed]),
            durationMs: a.durationMs + b.durationMs,
            memoryUsedMB: Math.max(a.memoryUsedMB || 0, b.memoryUsedMB || 0), // Peak memory usage
            stepsExecuted: a.stepsExecuted + b.stepsExecuted,
            toolCalls: a.toolCalls + b.toolCalls,
        };
    }

    private static subtractAllocations(a: CoreResourceAllocation, b: CoreResourceAllocation): CoreResourceAllocation {
        return {
            maxCredits: this.subtractBigIntStrings(a.maxCredits, b.maxCredits),
            maxDurationMs: Math.max(0, a.maxDurationMs - b.maxDurationMs),
            maxMemoryMB: Math.max(0, a.maxMemoryMB - b.maxMemoryMB),
            maxConcurrentSteps: a.maxConcurrentSteps, // Concurrency doesn't subtract
        };
    }

    private static createEmptyUsage(): ExecutionResourceUsage {
        return {
            creditsUsed: "0",
            durationMs: 0,
            memoryUsedMB: 0,
            stepsExecuted: 0,
            toolCalls: 0,
        };
    }
}

/**
 * Resource tracker for individual executions
 */
export class ResourceTracker {
    private startTime: number;
    private usage: ExecutionResourceUsage;
    private allocation: CoreResourceAllocation;

    constructor(allocation: CoreResourceAllocation) {
        this.startTime = Date.now();
        this.allocation = allocation;
        this.usage = {
            creditsUsed: "0",
            durationMs: 0,
            memoryUsedMB: 0,
            stepsExecuted: 0,
            toolCalls: 0,
        };
    }

    /**
     * Add resource consumption and validate against limits
     */
    addUsage(additionalUsage: Partial<ExecutionResourceUsage>): {
        success: boolean;
        error?: string;
    } {
        const newUsage: ExecutionResourceUsage = {
            creditsUsed: ResourceAggregator["sumBigIntStrings"]([
                this.usage.creditsUsed,
                additionalUsage.creditsUsed || "0",
            ]),
            durationMs: this.usage.durationMs + (additionalUsage.durationMs || 0),
            memoryUsedMB: Math.max(this.usage.memoryUsedMB, additionalUsage.memoryUsedMB || 0),
            stepsExecuted: this.usage.stepsExecuted + (additionalUsage.stepsExecuted || 0),
            toolCalls: (this.usage.toolCalls || 0) + (additionalUsage.toolCalls || 0),
        };

        // Check against allocation limits
        const validation = ResourceAggregator.wouldExceedAllocation(
            { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0, toolCalls: 0 },
            newUsage,
            this.allocation,
        );

        if (validation.wouldExceed) {
            return {
                success: false,
                error: `Resource limit exceeded: ${validation.exceededResources.join(", ")}`,
            };
        }

        this.usage = newUsage;
        return { success: true };
    }

    /**
     * Get current usage with real-time duration
     */
    getCurrentUsage(): ExecutionResourceUsage {
        return {
            ...this.usage,
            durationMs: Date.now() - this.startTime,
        };
    }

    /**
     * Get remaining allocation
     */
    getRemainingAllocation(): CoreResourceAllocation {
        return ResourceAggregator.calculateRemaining(this.allocation, this.getCurrentUsage());
    }

    /**
     * Check if execution can continue within limits
     */
    canContinue(estimatedAdditionalUsage?: Partial<ExecutionResourceUsage>): boolean {
        if (!estimatedAdditionalUsage) {
            return true;
        }

        const validation = ResourceAggregator.wouldExceedAllocation(
            this.getCurrentUsage(),
            estimatedAdditionalUsage as ExecutionResourceUsage,
            this.allocation,
        );

        return !validation.wouldExceed;
    }
}

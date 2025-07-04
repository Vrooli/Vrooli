/**
 * Core execution types for the three-tier architecture
 * 
 * This module defines the fundamental types and interfaces used across
 * all three tiers of Vrooli's execution architecture.
 */
import type { SessionUser } from "../api/types.js";
import type { SwarmId } from "./ids.js";

/**
 * Execution strategy types for conversation and reasoning
 * Simple string union for strategy selection
 */
export type ExecutionStrategy = "conversation" | "reasoning" | "deterministic";

/**
 * Execution context shared across all tiers
 * This provides the common execution environment and metadata
 */
export interface ExecutionContext {
    readonly swarmId: SwarmId;
    readonly userData: SessionUser;
    readonly timestamp: Date;
    readonly resources?: AvailableResources;
    readonly history?: ExecutionHistory;
}

/**
 * Available resources for execution
 */
export interface AvailableResources {
    models: ModelResource[];
    tools: ToolResource[];
    apis: ApiResource[];
    credits: number;
    timeLimit?: number;
}

/**
 * Model resource definition
 */
export interface ModelResource {
    provider: string;
    model: string;
    capabilities: string[];
    cost: number;
    available: boolean;
}

/**
 * Tool resource definition
 */
export interface ToolResource {
    name: string;
    type: string;
    description: string;
    parameters: Record<string, unknown>;
}

/**
 * API resource definition
 */
export interface ApiResource {
    name: string;
    endpoint: string;
    authentication?: Record<string, unknown>;
}

/**
 * Execution history for context
 */
export interface ExecutionHistory {
    recentSteps: StepExecution[];
    totalExecutions: number;
    successRate: number;
}

/**
 * Step execution record
 */
export interface StepExecution {
    stepId: string;
    strategy: string;
    result: "success" | "failure" | "partial";
    duration: number;
}

/**
 * Resource allocation for execution
 */
export interface CoreResourceAllocation {
    /** Maximum number of credits that can be used by the swarm, as a stringified BigInt */
    maxCredits: string;
    /** Maximum duration of the swarm in milliseconds */
    maxDurationMs: number;
    /** Maximum memory allocation in MB */
    maxMemoryMB: number;
    /** Maximum number of concurrent steps */
    maxConcurrentSteps: number;
}

/**
 * Execution resource usage tracking
 * Used for tracking resource consumption during execution
 */
export interface ExecutionResourceUsage {
    /** Credits consumed (as string to match existing BigInt handling) */
    creditsUsed: string;
    /** Execution duration in milliseconds */
    durationMs: number;
    /** Number of steps executed (optional) */
    stepsExecuted: number;
    /** Memory used in megabytes (optional) */
    memoryUsedMB: number;
    /** Number of tool calls made */
    toolCalls: number;
}

/**
 * Execution result returned by each tier
 */
export interface ExecutionResult<T = unknown> {
    success: boolean;
    result?: T;
    outputs?: Record<string, unknown>;
    error?: ExecutionError;
    resourcesUsed: ExecutionResourceUsage;
    duration: number;
    context: ExecutionContext;
    metadata?: ExecutionMetadata;
    confidence?: number;
    performanceScore?: number;
}

/**
 * Execution metadata
 */
export interface ExecutionMetadata {
    strategy?: string;
    version?: string;
    executionPhases?: Record<string, unknown>;
    performance?: Record<string, unknown>;
    timestamp?: string;
    [key: string]: unknown;
}

/**
 * Execution error details
 */
export interface ExecutionError {
    code: string;
    message: string;
    tier: "tier1" | "tier2" | "tier3";
    type?: string;
    strategy?: string;
    phase?: string;
    cause?: Error;
    context?: Record<string, unknown>;
}

/**
 * Execution status enumeration
 */
export enum ExecutionStates {
    PENDING = "pending",
    RUNNING = "running",
    COMPLETED = "completed",
    FAILED = "failed",
    CANCELLED = "cancelled",
    PAUSED = "paused"
}

/**
 * Execution status response for monitoring
 */
export interface ExecutionStatus {
    swarmId: SwarmId;
    status: ExecutionStates;
    progress?: number;
    metadata?: {
        currentPhase?: string;
        activeRuns?: number;
        completedRuns?: number;
        [key: string]: unknown;
    };
    error?: {
        code: string;
        message: string;
        [key: string]: unknown;
    };
}

/**
 * Retry policy configuration
 */
export interface RetryPolicy {
    maxRetries: number;
    backoffMs: number;
    backoffMultiplier: number;
    maxBackoffMs: number;
}

/**
 * Execution options
 */
export interface ExecutionOptions {
    timeout?: number;
    priority?: "low" | "medium" | "high";
    retryPolicy?: RetryPolicy;
    debugMode?: boolean;
}

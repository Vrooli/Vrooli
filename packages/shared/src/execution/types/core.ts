/**
 * Core execution types for the three-tier architecture
 * 
 * This module defines the fundamental types and interfaces used across
 * all three tiers of Vrooli's execution architecture.
 */

/**
 * Unique identifier types for execution tracking
 */
export type ExecutionId = string;
export type StepId = string;
export type RoutineId = string;
export type SwarmId = string;

/**
 * Execution context shared across all tiers
 * This provides the common execution environment and metadata
 */
export interface ExecutionContext {
    readonly executionId: ExecutionId;
    readonly parentExecutionId?: ExecutionId;
    readonly swarmId: SwarmId;
    readonly userId: string;
    readonly timestamp: Date;
    readonly correlationId: string;
    readonly stepId?: StepId;
    readonly routineId?: RoutineId;
    readonly stepType?: string;
    readonly inputs?: Record<string, unknown>;
    readonly config?: Record<string, unknown>;
    readonly resources?: AvailableResources;
    readonly history?: ExecutionHistory;
    readonly constraints?: ExecutionConstraints;
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
 * Execution constraints
 */
export interface ExecutionConstraints {
    maxTokens?: number;
    maxTime?: number;
    maxCost?: number;
    requiredConfidence?: number;
    maxExecutionTime?: number;
}

/**
 * Resource allocation for execution
 */
export interface ResourceAllocation {
    maxCredits: string; // BigInt as string
    maxDurationMs: number;
    maxMemoryMB: number;
    maxConcurrentSteps: number;
}

/**
 * Resource usage tracking
 */
export interface ResourceUsage {
    creditsUsed: string; // BigInt as string
    durationMs: number;
    memoryUsedMB: number;
    stepsExecuted: number;
    tokens?: number;
    apiCalls?: number;
    computeTime?: number;
    cost?: number;
}

/**
 * Execution result returned by each tier
 */
export interface ExecutionResult<T = unknown> {
    success: boolean;
    result?: T;
    outputs?: Record<string, unknown>;
    error?: ExecutionError;
    resourcesUsed: ResourceUsage;
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
export enum ExecutionStatus {
    PENDING = "pending",
    RUNNING = "running",
    COMPLETED = "completed",
    FAILED = "failed",
    CANCELLED = "cancelled"
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
    strategy?: string;
    debugMode?: boolean;
}

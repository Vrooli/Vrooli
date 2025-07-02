/**
 * Core type definitions for Tier 3: Execution Intelligence
 * These types define the strategy-based execution capabilities
 */

import type { ExecutionResourceUsage, ExecutionStrategy } from "./core.js";

/**
 * Strategy execution result
 */
export interface StrategyExecutionResult {
    success: boolean;
    result?: unknown;
    error?: string;
    metadata: StrategyMetadata;
    feedback: StrategyFeedback;
}

/**
 * Strategy metadata
 */
export interface StrategyMetadata {
    strategyType: ExecutionStrategy;
    executionTime: number;
    resourceUsage: ExecutionResourceUsage;
    confidence: number;
    fallbackUsed: boolean;
}

/**
 * Strategy feedback for learning
 */
export interface StrategyFeedback {
    outcome: "success" | "partial" | "failure";
    userSatisfaction?: number; // 0-1
    performanceScore: number; // 0-1
    issues?: string[];
    improvements?: string[];
}

/**
 * Base execution strategy interface
 * Compatible with existing ExecutionStrategy
 */
export interface IExecutionStrategy {
    type: ExecutionStrategy;
    name: string;
    version: string;

    // Core execution method
    execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult>;

    // Strategy capabilities
    canHandle(stepType: string, config?: Record<string, unknown>): boolean;
    estimateResources(context: StrategyExecutionContext): ExecutionResourceUsage;

    // Performance metrics
    getPerformanceMetrics(): StrategyPerformance;
}

/**
 * Execution context for strategies
 */
export interface StrategyExecutionContext {
    stepId: string;
    stepType: string;
    inputs: Record<string, unknown>;
    config: Record<string, unknown>;
    resources: StrategyAvailableResources;
    history: StrategyExecutionHistory;
    constraints: StrategyExecutionConstraints;
}

/**
 * Available resources for execution
 */
export interface StrategyAvailableResources {
    models: StrategyModelResource[];
    tools: StrategyToolResource[];
    apis: StrategyApiResource[];
    credits: number;
    timeLimit?: number;
}

/**
 * Model resource definition
 */
export interface StrategyModelResource {
    provider: string;
    model: string;
    capabilities: string[];
    cost: number; // per token
    available: boolean;
}

/**
 * Tool resource definition
 */
export interface StrategyToolResource {
    name: string;
    type: string;
    description: string;
    parameters: Record<string, unknown>;
    cost?: number;
    rateLimit?: number;
}

/**
 * API resource definition
 */
export interface StrategyApiResource {
    name: string;
    baseUrl: string;
    authType: "none" | "apiKey" | "oauth" | "custom";
    rateLimit?: number;
    cost?: number;
}

/**
 * Execution history for context
 */
export interface StrategyExecutionHistory {
    recentSteps: Array<{
        stepId: string;
        strategy: ExecutionStrategy;
        result: "success" | "failure";
        duration: number;
    }>;
    totalExecutions: number;
    successRate: number;
}

/**
 * Execution constraints
 */
export interface StrategyExecutionConstraints {
    maxTokens?: number;
    maxTime?: number;
    maxCost?: number;
    maxRetries?: number;
    requiredConfidence?: number;
    allowedStrategies?: ExecutionStrategy[];
}

/**
 * Strategy performance metrics
 */
export interface StrategyPerformance {
    totalExecutions: number;
    successCount: number;
    failureCount: number;
    averageExecutionTime: number;
    averageResourceUsage: ExecutionResourceUsage;
    averageConfidence: number;
    evolutionScore: number; // 0-1, how much the strategy has improved
}

/**
 * Strategy evolution tracking
 */
export interface StrategyEvolution {
    strategyType: ExecutionStrategy;
    version: number;
    improvements: Array<{
        timestamp: Date;
        description: string;
        impact: number; // -1 to 1
    }>;
    currentPerformance: StrategyPerformance;
    targetPerformance: Partial<StrategyPerformance>;
}

/**
 * Tool execution request
 */
export interface ToolExecutionRequest {
    toolName: string;
    parameters: Record<string, unknown>;
    timeout?: number;
    retryPolicy?: StrategyRetryPolicy;
}

/**
 * Retry policy for tool execution
 */
export interface StrategyRetryPolicy {
    maxRetries: number;
    backoffStrategy: "linear" | "exponential";
    initialDelay: number;
    maxDelay: number;
}

/**
 * Tool execution result
 */
export interface ToolExecutionResult {
    success: boolean;
    output?: unknown;
    error?: string;
    duration: number;
    retries: number;
}

/**
 * Strategy factory configuration
 */
export interface StrategyFactoryConfig {
    defaultStrategy: ExecutionStrategy;
    fallbackChain: ExecutionStrategy[];
    adaptationEnabled: boolean;
    learningRate: number;
}

/**
 * Unified executor configuration
 */
export interface UnifiedExecutorConfig {
    strategyFactory: StrategyFactoryConfig;
    resourceLimits: ExecutionResourceUsage;
    sandboxEnabled: boolean;
    telemetryEnabled: boolean;
}

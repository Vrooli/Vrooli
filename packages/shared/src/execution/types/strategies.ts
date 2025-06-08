/**
 * Core type definitions for Tier 3: Execution Intelligence
 * These types define the strategy-based execution capabilities
 */

/**
 * Execution strategy types
 * Maintains compatibility with existing strategy system
 */
export enum StrategyType {
    CONVERSATIONAL = "CONVERSATIONAL",
    REASONING = "REASONING",
    DETERMINISTIC = "DETERMINISTIC",
    HYBRID = "HYBRID",
    CUSTOM = "CUSTOM"
}

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
    strategyType: StrategyType;
    executionTime: number;
    resourceUsage: ResourceUsage;
    confidence: number;
    fallbackUsed: boolean;
}

/**
 * Strategy-specific resource usage
 * Used by execution strategies for lightweight resource tracking
 */
export interface StrategyResourceUsage {
    tokens?: number;
    apiCalls?: number;
    computeTime?: number;
    memory?: number;
    cost?: number;
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
export interface ExecutionStrategy {
    type: StrategyType;
    name: string;
    version: string;
    
    // Core execution method
    execute(context: ExecutionContext): Promise<StrategyExecutionResult>;
    
    // Strategy capabilities
    canHandle(stepType: string, config?: Record<string, unknown>): boolean;
    estimateResources(context: ExecutionContext): ResourceUsage;
    
    // Learning and adaptation
    learn(feedback: StrategyFeedback): void;
    getPerformanceMetrics(): StrategyPerformance;
}

/**
 * Execution context for strategies
 */
export interface ExecutionContext {
    stepId: string;
    stepType: string;
    inputs: Record<string, unknown>;
    config: Record<string, unknown>;
    resources: AvailableResources;
    history: ExecutionHistory;
    constraints: ExecutionConstraints;
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
    cost: number; // per token
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
    cost?: number;
    rateLimit?: number;
}

/**
 * API resource definition
 */
export interface ApiResource {
    name: string;
    baseUrl: string;
    authType: "none" | "apiKey" | "oauth" | "custom";
    rateLimit?: number;
    cost?: number;
}

/**
 * Execution history for context
 */
export interface ExecutionHistory {
    recentSteps: Array<{
        stepId: string;
        strategy: StrategyType;
        result: "success" | "failure";
        duration: number;
    }>;
    totalExecutions: number;
    successRate: number;
}

/**
 * Execution constraints
 */
export interface ExecutionConstraints {
    maxTokens?: number;
    maxTime?: number;
    maxCost?: number;
    maxRetries?: number;
    requiredConfidence?: number;
    allowedStrategies?: StrategyType[];
}

/**
 * Strategy performance metrics
 */
export interface StrategyPerformance {
    totalExecutions: number;
    successCount: number;
    failureCount: number;
    averageExecutionTime: number;
    averageResourceUsage: ResourceUsage;
    averageConfidence: number;
    evolutionScore: number; // 0-1, how much the strategy has improved
}

/**
 * Strategy evolution tracking
 */
export interface StrategyEvolution {
    strategyType: StrategyType;
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
    retryPolicy?: RetryPolicy;
}

/**
 * Retry policy for tool execution
 */
export interface RetryPolicy {
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
    defaultStrategy: StrategyType;
    fallbackChain: StrategyType[];
    adaptationEnabled: boolean;
    learningRate: number;
}

/**
 * Unified executor configuration
 */
export interface UnifiedExecutorConfig {
    strategyFactory: StrategyFactoryConfig;
    resourceLimits: ResourceUsage;
    sandboxEnabled: boolean;
    telemetryEnabled: boolean;
}

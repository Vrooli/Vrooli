import { generatePK } from "@vrooli/shared";
import { type Logger } from "winston";

/**
 * User data available during execution
 */
export interface UserData {
    id: string;
    email?: string;
    name?: string;
    languages?: string[];
    preferences?: Record<string, unknown>;
}

/**
 * Step configuration from routine definition
 */
export interface StepConfig {
    requiredInputs?: string[];
    outputTransformations?: Record<string, unknown>;
    timeoutMs?: number;
    retryPolicy?: {
        maxRetries: number;
        backoffMs: number;
    };
}

/**
 * Usage hints for strategy selection
 */
export interface UsageHints {
    historicalSuccessRate?: number;
    executionFrequency?: number;
    averageComplexity?: number;
    userPreference?: string;
    domainRestrictions?: string[];
}

/**
 * ExecutionRunContext - Runtime environment for Tier 3 step execution
 * 
 * This class maintains the execution context passed from Tier 2 (Process Intelligence)
 * to Tier 3 (Execution Intelligence). It provides the minimal, focused context needed
 * for executing individual routine steps.
 * 
 * ## Purpose
 * ExecutionRunContext is designed for:
 * - Individual step execution with user context
 * - Environment and configuration management
 * - Metadata storage for step-specific data
 * - Immutable context during execution for consistency
 * 
 * ## Key Differences from Other Context Types
 * 
 * ### Different from RunExecutionContext (Tier 2)
 * RunExecutionContext tracks the entire routine execution:
 * - Navigation state (current location, visited locations)
 * - Resource tracking and limits
 * - Parallel branch management
 * - Variables, blackboard, and scopes
 * 
 * ExecutionRunContext focuses only on step execution:
 * - User data and preferences
 * - Environment variables
 * - Step-specific configuration
 * - Metadata storage (replaces variables)
 * 
 * ### Different from Tier 2 RunExecutionContext
 * RunExecutionContext (Tier 2) provides comprehensive tracking:
 * - variables: Record<string, unknown>
 * - blackboard: Record<string, unknown>
 * - scopes: ContextScope[]
 * 
 * ExecutionRunContext uses a class-based approach with:
 * - Immutable fields for consistency
 * - Metadata Map for flexible storage
 * - User and environment focus
 * 
 * ## Context Transformation
 * When Tier 2 delegates to Tier 3:
 * ```
 * RunExecutionContext → ExecutionRunContext
 * - runId, routineId preserved
 * - variables → metadata
 * - swarmId carried forward
 * - User data extracted from swarm context
 * - Step config added based on current location
 * ```
 * 
 * ## Design Principles
 * 1. **Immutability**: All fields are readonly to prevent side effects
 * 2. **Focused Scope**: Only data needed for step execution
 * 3. **Type Safety**: Class-based with strong typing
 * 4. **Extensibility**: Metadata allows flexible data storage
 * 
 * The ExecutionRunContext is intentionally minimal, containing only what's needed
 * for Tier 3 to execute individual steps, while Tier 2 maintains the comprehensive
 * execution state.
 * 
 * @see RunExecutionContext in tier2/orchestration/unifiedRunStateMachine.ts - The Tier 2 context
 */
export class ExecutionRunContext {
    readonly runId: string;
    readonly routineId: string;
    readonly routineName: string;
    readonly currentStepId?: string;
    readonly parentRunId?: string;
    readonly swarmId?: string;

    readonly userData: UserData;
    readonly environment: Record<string, string>;
    readonly stepConfig?: StepConfig;
    readonly usageHints?: UsageHints;

    private readonly metadata: Map<string, unknown>;
    private readonly startTime: number;
    private readonly logger?: Logger;

    constructor(config: ExecutionRunContextConfig) {
        this.runId = config.runId || generatePK();
        this.routineId = config.routineId;
        this.routineName = config.routineName;
        this.currentStepId = config.currentStepId;
        this.parentRunId = config.parentRunId;
        this.swarmId = config.swarmId;

        this.userData = config.userData;
        this.environment = config.environment || {};
        this.stepConfig = config.stepConfig;
        this.usageHints = config.usageHints;

        this.metadata = new Map(Object.entries(config.metadata || {}));
        this.startTime = Date.now();
        this.logger = config.logger;
    }

    /**
     * Creates a child context for nested execution
     */
    createChildContext(overrides: Partial<ExecutionRunContextConfig>): ExecutionRunContext {
        return new ExecutionRunContext({
            ...this.toConfig(),
            ...overrides,
            parentRunId: this.runId,
            runId: undefined, // Generate new ID
        });
    }

    /**
     * Gets metadata value
     */
    getMetadata<T = unknown>(key: string): T | undefined {
        return this.metadata.get(key) as T;
    }

    /**
     * Gets all metadata as object
     */
    getAllMetadata(): Record<string, unknown> {
        return Object.fromEntries(this.metadata);
    }

    /**
     * Gets elapsed time since context creation
     */
    getElapsedTime(): number {
        return Date.now() - this.startTime;
    }

    /**
     * Gets environment variable
     */
    getEnvVar(key: string, defaultValue?: string): string | undefined {
        return this.environment[key] || defaultValue;
    }

    /**
     * Checks if running in development mode
     */
    isDevelopment(): boolean {
        return this.environment.NODE_ENV === "development";
    }

    /**
     * Checks if running in production mode
     */
    isProduction(): boolean {
        return this.environment.NODE_ENV === "production";
    }

    /**
     * Gets user's preferred language
     */
    getUserLanguage(): string {
        return this.userData.languages?.[0] || "en";
    }

    /**
     * Checks if user has a specific preference
     */
    hasUserPreference(key: string): boolean {
        return this.userData.preferences?.[key] !== undefined;
    }

    /**
     * Gets user preference value
     */
    getUserPreference<T = unknown>(key: string, defaultValue?: T): T | undefined {
        return (this.userData.preferences?.[key] as T) || defaultValue;
    }

    /**
     * Logs context information
     */
    logContext(level: "debug" | "info" | "warn" | "error", message: string, extra?: Record<string, unknown>): void {
        if (!this.logger) return;

        const contextData = {
            runId: this.runId,
            routineId: this.routineId,
            stepId: this.currentStepId,
            userId: this.userData.id,
            elapsed: this.getElapsedTime(),
            ...extra,
        };

        this.logger[level](message, contextData);
    }

    /**
     * Converts context to configuration object
     */
    toConfig(): ExecutionRunContextConfig {
        return {
            runId: this.runId,
            routineId: this.routineId,
            routineName: this.routineName,
            currentStepId: this.currentStepId,
            parentRunId: this.parentRunId,
            swarmId: this.swarmId,
            userData: this.userData,
            environment: this.environment,
            stepConfig: this.stepConfig,
            usageHints: this.usageHints,
            metadata: this.getAllMetadata(),
            logger: this.logger,
        };
    }

    /**
     * Creates a summary for logging
     */
    toSummary(): Record<string, unknown> {
        return {
            runId: this.runId,
            routineId: this.routineId,
            routineName: this.routineName,
            currentStepId: this.currentStepId,
            userId: this.userData.id,
            elapsed: this.getElapsedTime(),
            hasParent: !!this.parentRunId,
            hasSwarm: !!this.swarmId,
        };
    }
}

/**
 * Configuration for creating ExecutionRunContext
 */
export interface ExecutionRunContextConfig {
    runId?: string;
    routineId: string;
    routineName: string;
    currentStepId?: string;
    parentRunId?: string;
    swarmId?: string;

    userData: UserData;
    environment?: Record<string, string>;
    stepConfig?: StepConfig;
    usageHints?: UsageHints;
    metadata?: Record<string, unknown>;

    logger?: Logger;
}

/**
 * Factory for creating ExecutionRunContext instances
 */
export class ExecutionRunContextFactory {
    private readonly defaultEnvironment: Record<string, string>;
    private readonly logger?: Logger;

    constructor(defaultEnvironment?: Record<string, string>, logger?: Logger) {
        this.defaultEnvironment = defaultEnvironment || {};
        this.logger = logger;
    }

    /**
     * Creates a new ExecutionRunContext
     */
    create(config: ExecutionRunContextConfig): ExecutionRunContext {
        return new ExecutionRunContext({
            ...config,
            environment: {
                ...this.defaultEnvironment,
                ...config.environment,
            },
            logger: config.logger || this.logger,
        });
    }

    /**
     * Creates an ExecutionRunContext from serialized data
     */
    fromJSON(data: Record<string, unknown>): ExecutionRunContext {
        return new ExecutionRunContext({
            runId: data.runId as string,
            routineId: data.routineId as string,
            routineName: data.routineName as string,
            currentStepId: data.currentStepId as string | undefined,
            parentRunId: data.parentRunId as string | undefined,
            swarmId: data.swarmId as string | undefined,
            userData: data.userData as UserData,
            environment: data.environment as Record<string, string> || {},
            stepConfig: data.stepConfig as StepConfig | undefined,
            usageHints: data.usageHints as UsageHints | undefined,
            metadata: data.metadata as Record<string, unknown> || {},
            logger: this.logger,
        });
    }
}

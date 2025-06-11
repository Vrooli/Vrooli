/**
 * Unified Context Model
 * 
 * This module consolidates the various context models used across all three tiers
 * into a single flexible context system. Each tier gets appropriate views of the
 * unified context without duplicating data structures.
 * 
 * Benefits:
 * - Eliminates duplication between context types
 * - Provides type-safe views for each tier
 * - Centralized context validation and transformation
 * - Simplified context flow between tiers
 * - Consistent data access patterns
 */

import { type Logger } from "winston";
import {
    type BaseContext,
    type ContextScope,
    type CoordinationContext,
    type ProcessContext,
    type ExecutionContext,
    type ContextMetadata,
} from "@vrooli/shared/dist/execution/types/context.js";

/**
 * Core context data structure that contains all context information
 * This is the single source of truth for context data
 */
export interface UnifiedContextData {
    // Core identification
    id: string;
    timestamp: Date;
    metadata: ContextMetadata;
    
    // Hierarchical structure
    swarmId?: string;
    runId?: string;
    stepId?: string;
    
    // Memory systems
    variables: Map<string, unknown>;
    scopes: ContextScope[];
    blackboard: Record<string, unknown>;
    sharedMemory: Record<string, unknown>;
    
    // State tracking
    states: {
        coordination?: CoordinationState;
        navigation?: NavigationState;
        orchestration?: OrchestrationState;
        adaptation?: AdaptationState;
    };
    
    // Execution data
    inputs: Record<string, unknown>;
    outputs: Record<string, unknown>;
    toolCalls: ToolCallRecord[];
    
    // Performance and learning
    performance: PerformanceMetrics;
    learningData: LearningRecord[];
    
    // Configuration
    config: ContextConfig;
}

/**
 * Context configuration
 */
export interface ContextConfig {
    tier: 1 | 2 | 3;
    isolationLevel: "none" | "variable" | "scope" | "full";
    persistenceLevel: "memory" | "checkpoint" | "permanent";
    securityLevel: "public" | "private" | "sensitive";
    maxVariableSize: number;
    maxScopeDepth: number;
}

/**
 * Performance metrics
 */
export interface PerformanceMetrics {
    stepDurations: Record<string, number>;
    resourceUsage: Record<string, number>;
    bottlenecks: string[];
    efficiencyScore: number;
    memoryUsage: number;
    contextAccessCount: number;
}

/**
 * Learning record
 */
export interface LearningRecord {
    id: string;
    timestamp: Date;
    pattern: string;
    confidence: number;
    frequency: number;
    applicability: string[];
}

/**
 * Tool call record
 */
export interface ToolCallRecord {
    id: string;
    timestamp: Date;
    toolName: string;
    parameters: Record<string, unknown>;
    result?: unknown;
    duration?: number;
    success: boolean;
}

/**
 * State interfaces (imported from existing types)
 */
export interface CoordinationState {
    phase: "planning" | "executing" | "monitoring" | "adapting";
    activeGoals: string[];
    completedGoals: string[];
    blockedGoals: string[];
}

export interface NavigationState {
    currentLocation: string;
    locationStack: string[];
    visitedLocations: Set<string>;
    branchStates: Record<string, BranchState>;
}

export interface BranchState {
    id: string;
    status: "pending" | "active" | "completed" | "failed";
    parallel: boolean;
    startedAt?: Date;
    completedAt?: Date;
}

export interface OrchestrationState {
    phase: "initializing" | "running" | "optimizing" | "completing";
    activeSteps: string[];
    pendingSteps: string[];
    completedSteps: string[];
    failedSteps: string[];
}

export interface AdaptationState {
    strategies: string[];
    optimizations: string[];
    learnings: string[];
    adaptations: string[];
}

/**
 * Tier-specific views of the unified context
 * These provide type-safe, read-only access to appropriate data for each tier
 */

/**
 * Tier 1 (Coordination) view - focuses on swarm coordination
 */
export interface CoordinationContextView {
    readonly id: string;
    readonly swarmId: string;
    readonly timestamp: Date;
    readonly metadata: ContextMetadata;
    readonly sharedMemory: Record<string, unknown>;
    readonly coordinationState: CoordinationState;
    readonly blackboard: Record<string, unknown>;
    readonly performance: Pick<PerformanceMetrics, "efficiencyScore" | "memoryUsage">;
}

/**
 * Tier 2 (Process) view - focuses on routine execution
 */
export interface ProcessContextView {
    readonly id: string;
    readonly runId: string;
    readonly routineId?: string;
    readonly timestamp: Date;
    readonly metadata: ContextMetadata;
    readonly variables: ReadonlyMap<string, unknown>;
    readonly scopes: readonly ContextScope[];
    readonly navigationState: NavigationState;
    readonly orchestrationState: OrchestrationState;
    readonly performance: PerformanceMetrics;
    readonly config: ContextConfig;
}

/**
 * Tier 3 (Execution) view - focuses on step execution
 */
export interface ExecutionContextView {
    readonly id: string;
    readonly stepId: string;
    readonly runId?: string;
    readonly timestamp: Date;
    readonly metadata: ContextMetadata;
    readonly inputs: Record<string, unknown>;
    readonly outputs: Record<string, unknown>;
    readonly toolCalls: readonly ToolCallRecord[];
    readonly adaptationState: AdaptationState;
    readonly learningData: readonly LearningRecord[];
    readonly config: ContextConfig;
}

/**
 * Unified Context Manager
 * 
 * Manages the unified context and provides tier-specific views
 */
export class UnifiedContextManager {
    private readonly logger: Logger;
    private readonly contextData: UnifiedContextData;

    constructor(logger: Logger, initialData?: Partial<UnifiedContextData>) {
        this.logger = logger;
        this.contextData = this.initializeContext(initialData);
    }

    /**
     * Initialize context with default values
     */
    private initializeContext(initialData?: Partial<UnifiedContextData>): UnifiedContextData {
        return {
            id: initialData?.id || `ctx-${Date.now()}`,
            timestamp: new Date(),
            metadata: initialData?.metadata || {
                userId: "system",
                sessionId: "default",
                requestId: "default",
                tags: [],
            },
            swarmId: initialData?.swarmId,
            runId: initialData?.runId,
            stepId: initialData?.stepId,
            variables: initialData?.variables || new Map(),
            scopes: initialData?.scopes || [],
            blackboard: initialData?.blackboard || {},
            sharedMemory: initialData?.sharedMemory || {},
            states: initialData?.states || {},
            inputs: initialData?.inputs || {},
            outputs: initialData?.outputs || {},
            toolCalls: initialData?.toolCalls || [],
            performance: initialData?.performance || {
                stepDurations: {},
                resourceUsage: {},
                bottlenecks: [],
                efficiencyScore: 1.0,
                memoryUsage: 0,
                contextAccessCount: 0,
            },
            learningData: initialData?.learningData || [],
            config: initialData?.config || {
                tier: 1,
                isolationLevel: "scope",
                persistenceLevel: "checkpoint",
                securityLevel: "private",
                maxVariableSize: 1024 * 1024, // 1MB
                maxScopeDepth: 10,
            },
            ...initialData,
        };
    }

    /**
     * Get Tier 1 (Coordination) view of the context
     */
    getCoordinationView(): CoordinationContextView {
        if (!this.contextData.swarmId) {
            throw new Error("SwarmId required for coordination view");
        }

        return {
            id: this.contextData.id,
            swarmId: this.contextData.swarmId,
            timestamp: this.contextData.timestamp,
            metadata: this.contextData.metadata,
            sharedMemory: this.contextData.sharedMemory,
            coordinationState: this.contextData.states.coordination || {
                phase: "planning",
                activeGoals: [],
                completedGoals: [],
                blockedGoals: [],
            },
            blackboard: this.contextData.blackboard,
            performance: {
                efficiencyScore: this.contextData.performance.efficiencyScore,
                memoryUsage: this.contextData.performance.memoryUsage,
            },
        };
    }

    /**
     * Get Tier 2 (Process) view of the context
     */
    getProcessView(): ProcessContextView {
        if (!this.contextData.runId) {
            throw new Error("RunId required for process view");
        }

        return {
            id: this.contextData.id,
            runId: this.contextData.runId,
            routineId: this.contextData.metadata.tags.find(tag => tag.startsWith("routine:"))?.split(":")[1],
            timestamp: this.contextData.timestamp,
            metadata: this.contextData.metadata,
            variables: this.contextData.variables,
            scopes: this.contextData.scopes,
            navigationState: this.contextData.states.navigation || {
                currentLocation: "start",
                locationStack: [],
                visitedLocations: new Set(),
                branchStates: {},
            },
            orchestrationState: this.contextData.states.orchestration || {
                phase: "initializing",
                activeSteps: [],
                pendingSteps: [],
                completedSteps: [],
                failedSteps: [],
            },
            performance: this.contextData.performance,
            config: this.contextData.config,
        };
    }

    /**
     * Get Tier 3 (Execution) view of the context
     */
    getExecutionView(): ExecutionContextView {
        if (!this.contextData.stepId) {
            throw new Error("StepId required for execution view");
        }

        return {
            id: this.contextData.id,
            stepId: this.contextData.stepId,
            runId: this.contextData.runId,
            timestamp: this.contextData.timestamp,
            metadata: this.contextData.metadata,
            inputs: this.contextData.inputs,
            outputs: this.contextData.outputs,
            toolCalls: this.contextData.toolCalls,
            adaptationState: this.contextData.states.adaptation || {
                strategies: [],
                optimizations: [],
                learnings: [],
                adaptations: [],
            },
            learningData: this.contextData.learningData,
            config: this.contextData.config,
        };
    }

    /**
     * Update context data
     */
    updateContext(updates: Partial<UnifiedContextData>): void {
        Object.assign(this.contextData, updates);
        this.contextData.timestamp = new Date();
        this.contextData.performance.contextAccessCount++;
    }

    /**
     * Set variable in appropriate scope
     */
    setVariable(key: string, value: unknown, scopeId?: string): void {
        if (scopeId) {
            const scope = this.contextData.scopes.find(s => s.id === scopeId);
            if (scope) {
                scope.variables[key] = value;
            } else {
                throw new Error(`Scope ${scopeId} not found`);
            }
        } else {
            this.contextData.variables.set(key, value);
        }
        this.updateTimestamp();
    }

    /**
     * Get variable from appropriate scope
     */
    getVariable(key: string, scopeId?: string): unknown {
        this.contextData.performance.contextAccessCount++;
        
        if (scopeId) {
            const scope = this.contextData.scopes.find(s => s.id === scopeId);
            return scope?.variables[key];
        }
        
        return this.contextData.variables.get(key);
    }

    /**
     * Clone context for isolation
     */
    clone(newId?: string): UnifiedContextManager {
        const clonedData: UnifiedContextData = {
            ...this.contextData,
            id: newId || `${this.contextData.id}-clone-${Date.now()}`,
            timestamp: new Date(),
            variables: new Map(this.contextData.variables),
            scopes: this.contextData.scopes.map(scope => ({ ...scope, variables: { ...scope.variables } })),
            blackboard: { ...this.contextData.blackboard },
            sharedMemory: { ...this.contextData.sharedMemory },
            states: { ...this.contextData.states },
            inputs: { ...this.contextData.inputs },
            outputs: { ...this.contextData.outputs },
            toolCalls: [...this.contextData.toolCalls],
            performance: { ...this.contextData.performance },
            learningData: [...this.contextData.learningData],
            config: { ...this.contextData.config },
        };

        return new UnifiedContextManager(this.logger, clonedData);
    }

    /**
     * Serialize context for persistence
     */
    serialize(): string {
        const serializable = {
            ...this.contextData,
            variables: Object.fromEntries(this.contextData.variables),
            states: {
                ...this.contextData.states,
                navigation: this.contextData.states.navigation ? {
                    ...this.contextData.states.navigation,
                    visitedLocations: Array.from(this.contextData.states.navigation.visitedLocations),
                } : undefined,
            },
        };

        return JSON.stringify(serializable);
    }

    /**
     * Deserialize context from persistence
     */
    static deserialize(logger: Logger, serialized: string): UnifiedContextManager {
        const data = JSON.parse(serialized);
        
        // Convert variables back to Map
        data.variables = new Map(Object.entries(data.variables || {}));
        
        // Convert visitedLocations back to Set
        if (data.states?.navigation?.visitedLocations) {
            data.states.navigation.visitedLocations = new Set(data.states.navigation.visitedLocations);
        }

        return new UnifiedContextManager(logger, data);
    }

    /**
     * Get context statistics
     */
    getStatistics(): {
        variableCount: number;
        scopeCount: number;
        toolCallCount: number;
        memoryUsage: number;
        accessCount: number;
    } {
        return {
            variableCount: this.contextData.variables.size,
            scopeCount: this.contextData.scopes.length,
            toolCallCount: this.contextData.toolCalls.length,
            memoryUsage: this.contextData.performance.memoryUsage,
            accessCount: this.contextData.performance.contextAccessCount,
        };
    }

    private updateTimestamp(): void {
        this.contextData.timestamp = new Date();
    }
}

/**
 * Context factory for creating tier-specific contexts
 */
export class UnifiedContextFactory {
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Create coordination context for Tier 1
     */
    createCoordinationContext(swarmId: string, metadata: ContextMetadata): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            swarmId,
            metadata,
            config: {
                tier: 1,
                isolationLevel: "scope",
                persistenceLevel: "permanent",
                securityLevel: "private",
                maxVariableSize: 10 * 1024 * 1024, // 10MB for swarm coordination
                maxScopeDepth: 5,
            },
        });
    }

    /**
     * Create process context for Tier 2
     */
    createProcessContext(runId: string, metadata: ContextMetadata, parentContext?: UnifiedContextManager): UnifiedContextManager {
        const contextManager = new UnifiedContextManager(this.logger, {
            runId,
            metadata,
            swarmId: parentContext?.getCoordinationView().swarmId,
            config: {
                tier: 2,
                isolationLevel: "variable",
                persistenceLevel: "checkpoint",
                securityLevel: "private",
                maxVariableSize: 1024 * 1024, // 1MB for routine execution
                maxScopeDepth: 10,
            },
        });

        // Inherit shared memory from parent if available
        if (parentContext) {
            try {
                const coordinationView = parentContext.getCoordinationView();
                contextManager.updateContext({
                    sharedMemory: coordinationView.sharedMemory,
                });
            } catch {
                // Parent might not be a coordination context
            }
        }

        return contextManager;
    }

    /**
     * Create execution context for Tier 3
     */
    createExecutionContext(stepId: string, metadata: ContextMetadata, parentContext?: UnifiedContextManager): UnifiedContextManager {
        const contextManager = new UnifiedContextManager(this.logger, {
            stepId,
            metadata,
            runId: parentContext?.getProcessView().runId,
            config: {
                tier: 3,
                isolationLevel: "full",
                persistenceLevel: "memory",
                securityLevel: "public",
                maxVariableSize: 512 * 1024, // 512KB for step execution
                maxScopeDepth: 3,
            },
        });

        // Inherit variables from parent if available
        if (parentContext) {
            try {
                const processView = parentContext.getProcessView();
                contextManager.updateContext({
                    variables: new Map(processView.variables),
                    scopes: [...processView.scopes],
                });
            } catch {
                // Parent might not be a process context
            }
        }

        return contextManager;
    }
}
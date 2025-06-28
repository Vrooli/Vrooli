/**
 * Run state store
 * Manages persistent state for routine execution
 */

import {
    DAYS_1_S,
    RunState,
    type BranchExecution,
    type Checkpoint,
    type Location,
    type StepExecution,
} from "@vrooli/shared";
import { type Redis } from "ioredis";
import { logger } from "../../../../events/logger.js";
import { CacheService } from "../../../../redisConn.js";
import { RedisIndexManager } from "../../shared/RedisIndexManager.js";
import { type RunExecutionContext } from "../orchestration/unifiedRunStateMachine.js";

// Constants
const DEFAULT_TTL_S = DAYS_1_S; // 24 hours

/**
 * Run configuration
 */
export interface RunConfig {
    id: string;
    routineId: string;
    userId: string;
    inputs: Record<string, unknown>;
    metadata: Record<string, unknown>;
    createdAt: Date;
    updatedAt: Date;
}

/**
 * Run state store interface - Phase 2B Complete
 * 
 * This interface has been updated to remove all deprecated ProcessRunContext methods.
 * All context management now flows through the modern IModernRunStateStore interface.
 * 
 * ## Migration Complete
 * The following methods have been removed:
 * - getContext() - replaced by IModernRunStateStore.getRunContext()
 * - updateContext() - replaced by IModernRunStateStore.updateRunContext()
 * - setVariable() - variables now managed through RunExecutionContext
 * - getVariable() - variables now accessed through RunExecutionContext
 * 
 * ## Interface Evolution
 * This interface now focuses purely on:
 * - Run lifecycle management
 * - State tracking
 * - Location and navigation tracking
 * - Branch and step execution tracking
 * - Checkpoint management
 * - Query operations
 * 
 * For context management, use IModernRunStateStore interface methods.
 * 
 * @see IModernRunStateStore for context management methods
 * @see RunExecutionContext for the modern context structure
 */
export interface IRunStateStore {
    // Initialization
    initialize(): Promise<void>;

    // Run lifecycle
    createRun(runId: string, config: RunConfig): Promise<void>;
    getRun(runId: string): Promise<RunConfig | null>;
    updateRun(runId: string, updates: Partial<RunConfig>): Promise<void>;
    deleteRun(runId: string): Promise<void>;

    // State management
    getRunState(runId: string): Promise<RunState>;
    updateRunState(runId: string, state: RunState): Promise<void>;

    // Location tracking
    getCurrentLocation(runId: string): Promise<Location | null>;
    updateLocation(runId: string, location: Location): Promise<void>;
    getLocationHistory(runId: string): Promise<Location[]>;

    // Branch management
    createBranch(runId: string, branch: BranchExecution): Promise<void>;
    getBranch(runId: string, branchId: string): Promise<BranchExecution | null>;
    updateBranch(runId: string, branchId: string, updates: Partial<BranchExecution>): Promise<void>;
    listBranches(runId: string): Promise<BranchExecution[]>;

    // Step tracking
    recordStepExecution(runId: string, step: StepExecution): Promise<void>;
    getStepExecution(runId: string, stepId: string): Promise<StepExecution | null>;
    listStepExecutions(runId: string): Promise<StepExecution[]>;

    // Checkpoint management
    createCheckpoint(runId: string, checkpoint: Checkpoint): Promise<void>;
    getCheckpoint(runId: string, checkpointId: string): Promise<Checkpoint | null>;
    listCheckpoints(runId: string): Promise<Checkpoint[]>;
    deleteCheckpoint(runId: string, checkpointId: string): Promise<void>;
    restoreFromCheckpoint(runId: string, checkpointId: string): Promise<void>;

    // Query operations
    listActiveRuns(): Promise<string[]>;
    getRunsByState(state: RunState): Promise<string[]>;
    getRunsByUser(userId: string): Promise<string[]>;
}

/**
 * Modern run state store interface - Phase 2A of context system migration
 * 
 * This interface represents the target architecture for Tier 2 state management,
 * replacing the deprecated ProcessRunContext methods with modern RunExecutionContext-based
 * methods that align with the three-tier execution architecture.
 * 
 * ## Key Changes from IRunStateStore:
 * 1. **Removed deprecated context methods**: No more getContext(), updateContext(), 
 *    setVariable(), getVariable() that depend on ProcessRunContext
 * 2. **Added modern context methods**: getRunContext() and updateRunContext() that
 *    work with RunExecutionContext from UnifiedRunStateMachine
 * 3. **Maintains all other operations**: Location tracking, branch management,
 *    step tracking, checkpoint management, and query operations remain unchanged
 * 
 * ## Context Architecture Alignment:
 * This interface supports the documented three-tier context flow:
 * ```
 * Tier 1: SwarmContext
 *    ↓ (inheritance via parentContext)
 * Tier 2: RunExecutionContext (this interface manages)
 *    ↓ (transformation for step execution)
 * Tier 3: ExecutionRunContext
 * ```
 * 
 * ## Migration Path:
 * - **Phase 2A** (Current): Create this interface, update UnifiedRunStateMachine to use it
 * - **Phase 2B** (Next): Remove deprecated methods from IRunStateStore completely
 * - **Phase 2C** (Final): All components migrated to modern context management
 * 
 * ## Implementation Guidelines:
 * Implementations of this interface should:
 * 1. Store RunExecutionContext as serialized JSON in persistent storage
 * 2. Handle context inheritance from swarm configurations 
 * 3. Support the complete RunExecutionContext structure including navigation,
 *    resource tracking, and progress management
 * 4. Maintain backward compatibility during transition via adapter patterns
 * 
 * @see RunExecutionContext in UnifiedRunStateMachine for the context structure
 * @see UnifiedRunStateMachine for the primary consumer of this interface
 * @see docs/plans/context-system-final-migration.md for the complete migration plan
 */
export interface IModernRunStateStore {
    // Initialization
    initialize(): Promise<void>;

    // Run lifecycle
    createRun(runId: string, config: RunConfig): Promise<void>;
    getRun(runId: string): Promise<RunConfig | null>;
    updateRun(runId: string, updates: Partial<RunConfig>): Promise<void>;
    deleteRun(runId: string): Promise<void>;

    // State management
    getRunState(runId: string): Promise<RunState>;
    updateRunState(runId: string, state: RunState): Promise<void>;

    // Modern context management - replaces deprecated ProcessRunContext methods
    /**
     * Retrieves the complete RunExecutionContext for a run
     * 
     * This method replaces the deprecated getContext() method, providing access to
     * the comprehensive execution state including navigation, resources, and progress.
     * 
     * @param runId The unique identifier for the run
     * @returns Promise resolving to the complete RunExecutionContext
     * @throws Error if the run does not exist
     * 
     * ## Context Structure:
     * The returned RunExecutionContext includes:
     * - Core identification (runId, routineId, swarmId)
     * - Navigation state (navigator, currentLocation, visitedLocations)
     * - Execution state (variables, outputs, completedSteps, parallelBranches)
     * - Swarm inheritance (parentContext, availableResources, sharedKnowledge)
     * - Resource tracking (resourceLimits, resourceUsage)
     * - Progress tracking (progress, retryCount, lastError)
     * - Compatibility fields (blackboard, scopes for ProcessRunContext migration)
     * 
     * ## Usage Example:
     * ```typescript
     * const context = await stateStore.getRunContext(runId);
     * console.log(`Run ${context.runId} is at location ${context.currentLocation.id}`);
     * console.log(`Variables: ${JSON.stringify(context.variables)}`);
     * console.log(`Progress: ${context.progress.percentComplete}%`);
     * ```
     */
    getRunContext(runId: string): Promise<RunExecutionContext>;

    /**
     * Updates the complete RunExecutionContext for a run
     * 
     * This method replaces the deprecated updateContext() method, storing the
     * comprehensive execution state including all navigation, resource, and progress data.
     * 
     * @param runId The unique identifier for the run
     * @param context The complete RunExecutionContext to store
     * @throws Error if the run does not exist
     * 
     * ## Implementation Notes:
     * - Should serialize the entire context structure to persistent storage
     * - Must handle nested objects (navigator, parentContext, progress, etc.)
     * - Should maintain atomic updates to prevent context corruption
     * - May implement optimized storage for frequently updated fields
     * 
     * ## Context Persistence:
     * The implementation should persist:
     * - All navigation state for resume capability
     * - Resource usage for limit enforcement
     * - Progress tracking for monitoring
     * - Variable state for step continuation
     * - Swarm context for coordination
     * 
     * ## Usage Example:
     * ```typescript
     * const context = await stateStore.getRunContext(runId);
     * context.variables.newOutput = result;
     * context.progress.percentComplete = 75;
     * context.resourceUsage.creditsUsed = "150";
     * await stateStore.updateRunContext(runId, context);
     * ```
     */
    updateRunContext(runId: string, context: RunExecutionContext): Promise<void>;

    // Location tracking (unchanged from IRunStateStore)
    getCurrentLocation(runId: string): Promise<Location | null>;
    updateLocation(runId: string, location: Location): Promise<void>;
    getLocationHistory(runId: string): Promise<Location[]>;

    // Branch management (unchanged from IRunStateStore)
    createBranch(runId: string, branch: BranchExecution): Promise<void>;
    getBranch(runId: string, branchId: string): Promise<BranchExecution | null>;
    updateBranch(runId: string, branchId: string, updates: Partial<BranchExecution>): Promise<void>;
    listBranches(runId: string): Promise<BranchExecution[]>;

    // Step tracking (unchanged from IRunStateStore)
    recordStepExecution(runId: string, step: StepExecution): Promise<void>;
    getStepExecution(runId: string, stepId: string): Promise<StepExecution | null>;
    listStepExecutions(runId: string): Promise<StepExecution[]>;

    // Checkpoint management (unchanged from IRunStateStore)
    createCheckpoint(runId: string, checkpoint: Checkpoint): Promise<void>;
    getCheckpoint(runId: string, checkpointId: string): Promise<Checkpoint | null>;
    listCheckpoints(runId: string): Promise<Checkpoint[]>;
    deleteCheckpoint(runId: string, checkpointId: string): Promise<void>;
    restoreFromCheckpoint(runId: string, checkpointId: string): Promise<void>;

    // Query operations (unchanged from IRunStateStore)
    listActiveRuns(): Promise<string[]>;
    getRunsByState(state: RunState): Promise<string[]>;
    getRunsByUser(userId: string): Promise<string[]>;
}

/**
 * Redis-based run state store - implements both legacy and modern interfaces
 * 
 * This implementation supports both the legacy IRunStateStore interface (with deprecated
 * ProcessRunContext methods) and the modern IModernRunStateStore interface (with 
 * RunExecutionContext methods) to enable seamless migration.
 * 
 * ## Migration Strategy:
 * - Phase 2A: Implement both interfaces side by side
 * - Phase 2B: Remove deprecated methods from IRunStateStore
 * - Phase 2C: Only implement IModernRunStateStore
 */
export class RedisRunStateStore implements IRunStateStore, IModernRunStateStore {
    private readonly keyPrefix = "run:";
    private readonly indexPrefix = "run_index:";
    private readonly subsidiaryTtl = DEFAULT_TTL_S;
    private readonly indexManager: RedisIndexManager;
    private redis: Redis;

    // In-memory cache for frequently accessed contexts (Phase 2C optimization)
    private readonly contextCache = new Map<string, {
        context: RunExecutionContext;
        timestamp: number;
        accessCount: number;
    }>();
    private readonly CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes
    private readonly MAX_CACHE_SIZE = 100;

    constructor(redis: Redis) {
        this.redis = redis;
        this.indexManager = new RedisIndexManager(redis, logger, DEFAULT_TTL_S);
    }

    async initialize(): Promise<void> {
        // Base store doesn't need explicit initialization
    }

    async createRun(runId: string, config: RunConfig): Promise<void> {
        // Store the run config directly in Redis
        const key = this.getRunKey(runId);
        await this.redis.setex(key, this.subsidiaryTtl, JSON.stringify(config));

        // Initialize state
        await this.updateRunState(runId, RunState.PENDING);

        // Initialize empty modern context
        const initialContext: RunExecutionContext = {
            runId,
            routineId: config.routineId,
            navigator: null as any, // Will be set by UnifiedRunStateMachine
            currentLocation: { id: "start", nodeId: "start", stepId: "start" },
            visitedLocations: [],
            variables: config.inputs,
            outputs: {},
            completedSteps: [],
            parallelBranches: [],
            blackboard: {},
            scopes: [{
                id: "global",
                name: "Global Scope",
                variables: config.inputs,
            }],
            resourceLimits: {
                maxCredits: "1000",
                maxDurationMs: 300000,
                maxMemoryMB: 512,
                maxSteps: 50,
            },
            resourceUsage: {
                creditsUsed: "0",
                durationMs: 0,
                memoryUsedMB: 0,
                stepsExecuted: 0,
                startTime: new Date(),
            },
            progress: {
                currentStepId: null,
                completedSteps: [],
                totalSteps: 0,
                percentComplete: 0,
            },
            retryCount: 0,
        };
        await this.updateRunContext(runId, initialContext);

        // Update indexes
        await this.updateIndexes(runId, config);

        logger.info("Run created", { runId });
    }

    async getRun(runId: string): Promise<RunConfig | null> {
        const key = this.getRunKey(runId);
        const data = await this.redis.get(key);

        if (!data) {
            return null;
        }

        try {
            return JSON.parse(data) as RunConfig;
        } catch (error) {
            logger.error("[RedisRunStateStore] Failed to parse run config", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    async updateRun(runId: string, updates: Partial<RunConfig>): Promise<void> {
        const current = await this.getRun(runId);
        if (!current) {
            throw new Error(`Run ${runId} not found`);
        }

        const updated = {
            ...current,
            ...updates,
            updatedAt: new Date(),
        };

        const key = this.getRunKey(runId);
        await this.redis.setex(key, this.subsidiaryTtl, JSON.stringify(updated));

        // Update indexes if userId changed
        if (updates.userId && updates.userId !== current.userId) {
            await this.updateIndexes(runId, updated);
        }
    }

    async deleteRun(runId: string): Promise<void> {
        const config = await this.getRun(runId);
        if (!config) return;

        // Remove from indexes first
        await this.removeFromIndexes(runId, config);

        // Clean up subsidiary data
        await this.cleanupRunData(runId);

        // Remove from context cache
        this.removeCachedContext(runId);

        // Delete main run
        const key = this.getRunKey(runId);
        await this.redis.del(key);

        logger.info("Run deleted", { runId });
    }

    async getRunState(runId: string): Promise<RunState> {
        const key = this.getStateKey(runId);
        const state = await this.redis.get(key);

        return (state as RunState) || RunState.PENDING;
    }

    async updateRunState(runId: string, state: RunState): Promise<void> {
        const oldState = await this.getRunState(runId);
        const key = this.getStateKey(runId);
        await this.redis.setex(key, this.subsidiaryTtl, state);

        // Update state index using IndexManager
        await this.indexManager.updateStateIndex(
            runId,
            oldState === RunState.PENDING ? null : oldState,
            state,
            (s) => this.getStateIndexKey(s),
            Object.values(RunState),
        );

        // Remove from active if terminal state
        const terminalStates = [RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED];
        if (terminalStates.includes(state)) {
            await this.indexManager.removeFromSet(this.getActiveIndexKey(), runId);
        }
    }

    // Legacy ProcessRunContext methods have been removed - use modern context methods instead

    /**
     * Modern context management methods - Phase 2A implementation
     * 
     * These methods implement the IModernRunStateStore interface, providing
     * RunExecutionContext-based context management that replaces the deprecated
     * ProcessRunContext methods.
     */

    /**
     * Retrieves the complete RunExecutionContext for a run
     * 
     * This method implements the modern context retrieval pattern, returning the
     * comprehensive execution state that includes navigation, resources, and progress.
     * 
     * ## Implementation Strategy:
     * 1. First checks for modern RunExecutionContext in storage
     * 2. Falls back to legacy ProcessRunContext and transforms it if needed
     * 3. Creates a default context if none exists
     * 
     * ## Storage Key:
     * Uses `${runId}:run_execution_context` for modern context storage,
     * separate from the legacy `${runId}:context` key.
     */
    async getRunContext(runId: string): Promise<RunExecutionContext> {
        // Check in-memory cache first
        const cached = this.getCachedContext(runId);
        if (cached) {
            return cached;
        }

        const key = this.getRunExecutionContextKey(runId);
        const data = await this.redis.get(key);

        if (data) {
            try {
                const rawContext = JSON.parse(data);

                // Reconstruct proper RunExecutionContext with correct types
                const context: RunExecutionContext = {
                    ...rawContext,
                    // Ensure Date objects are properly reconstructed
                    resourceUsage: {
                        ...rawContext.resourceUsage,
                        startTime: new Date(rawContext.resourceUsage.startTime),
                    },
                };

                // Validate that this is a proper RunExecutionContext
                if (context.runId && context.routineId) {
                    // Cache the retrieved context
                    this.setCachedContext(runId, context);
                    return context;
                }
            } catch (error) {
                logger.warn("[RedisRunStateStore] Failed to parse modern context, falling back", {
                    runId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        // Fallback: Try to migrate from legacy ProcessRunContext
        try {
            // Try to load from legacy context storage for migration
            const legacyKey = this.getContextKey(runId);
            const legacyData = await this.redis.get(legacyKey);
            const legacyContext = legacyData ? JSON.parse(legacyData) : null;
            const runConfig = await this.get(runId);

            if (runConfig) {
                logger.info("[RedisRunStateStore] Migrating legacy context to RunExecutionContext", { runId });

                // Create a basic RunExecutionContext from legacy data
                const migratedContext: RunExecutionContext = {
                    runId,
                    routineId: runConfig.routineId,

                    // Navigation state - will need to be set by UnifiedRunStateMachine
                    navigator: null as any, // This will be set when the context is first used
                    currentLocation: { id: "start", nodeId: "start", stepId: "start" }, // Default location
                    visitedLocations: [],

                    // Execution state from legacy context
                    variables: legacyContext.variables || {},
                    outputs: {},
                    completedSteps: [],
                    parallelBranches: [],

                    // Context management (ProcessRunContext compatibility)
                    blackboard: legacyContext.blackboard || {},
                    scopes: legacyContext.scopes || [{
                        id: "global",
                        name: "Global Scope",
                        variables: legacyContext.variables || {},
                    }],

                    // Resource tracking - initialize with defaults
                    resourceLimits: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 512,
                        maxSteps: 50,
                    },
                    resourceUsage: {
                        creditsUsed: "0",
                        durationMs: 0,
                        memoryUsedMB: 0,
                        stepsExecuted: 0,
                        startTime: new Date(),
                    },

                    // Progress tracking - initialize
                    progress: {
                        currentStepId: null,
                        completedSteps: [],
                        totalSteps: 0,
                        percentComplete: 0,
                    },

                    // Error handling
                    retryCount: 0,
                };

                // Store the migrated context for future use
                await this.updateRunContext(runId, migratedContext);

                // Cache the migrated context
                this.setCachedContext(runId, migratedContext);

                return migratedContext;
            }
        } catch (error) {
            logger.debug("[RedisRunStateStore] Could not migrate legacy context", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
        }

        // Final fallback: Create minimal default context
        logger.warn("[RedisRunStateStore] Creating default RunExecutionContext", { runId });

        const defaultContext: RunExecutionContext = {
            runId,
            routineId: runId, // Fallback to runId if no routine info available

            // Navigation state - minimal defaults
            navigator: null as any, // Will be set by UnifiedRunStateMachine
            currentLocation: { id: "start", nodeId: "start", stepId: "start" },
            visitedLocations: [],

            // Execution state - empty defaults
            variables: {},
            outputs: {},
            completedSteps: [],
            parallelBranches: [],

            // Context management - empty defaults
            blackboard: {},
            scopes: [{
                id: "global",
                name: "Global Scope",
                variables: {},
            }],

            // Resource tracking - defaults
            resourceLimits: {
                maxCredits: "1000",
                maxDurationMs: 300000,
                maxMemoryMB: 512,
                maxSteps: 50,
            },
            resourceUsage: {
                creditsUsed: "0",
                durationMs: 0,
                memoryUsedMB: 0,
                stepsExecuted: 0,
                startTime: new Date(),
            },

            // Progress tracking - defaults
            progress: {
                currentStepId: null,
                completedSteps: [],
                totalSteps: 0,
                percentComplete: 0,
            },

            // Error handling
            retryCount: 0,
        };

        // Cache the default context
        this.setCachedContext(runId, defaultContext);

        return defaultContext;
    }

    /**
     * Updates the complete RunExecutionContext for a run
     * 
     * This method implements the modern context storage pattern, persisting the
     * comprehensive execution state in a structured format optimized for the
     * three-tier architecture.
     * 
     * ## Implementation Details:
     * - Stores context as JSON in Redis with proper TTL
     * - Uses a separate key from legacy ProcessRunContext
     * - Handles serialization of complex nested objects
     * - Maintains atomic updates to prevent corruption
     * 
     * ## Storage Optimization:
     * The implementation serializes the entire context but could be optimized
     * in the future to store frequently updated fields separately.
     */
    async updateRunContext(runId: string, context: RunExecutionContext): Promise<void> {
        try {
            const key = this.getRunExecutionContextKey(runId);

            // Prepare context for serialization
            const serializableContext = {
                ...context,
                // Ensure dates are properly serialized
                resourceUsage: {
                    ...context.resourceUsage,
                    startTime: context.resourceUsage.startTime.toISOString(),
                },
            };

            // Store with TTL
            await this.redis.setex(key, this.subsidiaryTtl, JSON.stringify(serializableContext));

            // Update cache with new context
            this.setCachedContext(runId, context);

            logger.debug("[RedisRunStateStore] Updated RunExecutionContext", {
                runId,
                contextSize: JSON.stringify(serializableContext).length,
                currentStepId: context.progress.currentStepId,
                percentComplete: context.progress.percentComplete,
            });

        } catch (error) {
            logger.error("[RedisRunStateStore] Failed to update RunExecutionContext", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getCurrentLocation(runId: string): Promise<Location | null> {
        const key = this.getCurrentLocationKey(runId);
        const data = await this.redis.get(key);

        return data ? JSON.parse(data) : null;
    }

    async updateLocation(runId: string, location: Location): Promise<void> {
        // Update current location
        const key = this.getCurrentLocationKey(runId);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(location));

        // Add to location history
        const historyKey = this.getLocationHistoryKey(runId);
        await redis.rpush(historyKey, JSON.stringify(location));
        await redis.expire(historyKey, this.subsidiaryTtl);
    }

    async getLocationHistory(runId: string): Promise<Location[]> {
        const key = this.getLocationHistoryKey(runId);
        const history = await redis.lrange(key, 0, -1);

        return history.map(item => JSON.parse(item));
    }

    async createBranch(runId: string, branch: BranchExecution): Promise<void> {
        const key = this.getBranchKey(runId, branch.id);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(branch));

        // Add to branches set using IndexManager
        await this.indexManager.addToSet(this.getBranchIndexKey(runId), branch.id, this.subsidiaryTtl);
    }

    async getBranch(runId: string, branchId: string): Promise<BranchExecution | null> {
        const key = this.getBranchKey(runId, branchId);
        const data = await redis.get(key);

        return data ? JSON.parse(data) : null;
    }

    async updateBranch(runId: string, branchId: string, updates: Partial<BranchExecution>): Promise<void> {
        const current = await this.getBranch(runId, branchId);
        if (!current) {
            throw new Error(`Branch ${branchId} not found`);
        }

        const updated = { ...current, ...updates };
        const key = this.getBranchKey(runId, branchId);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(updated));
    }

    async listBranches(runId: string): Promise<BranchExecution[]> {
        const branchIds = await this.indexManager.getSetMembers(this.getBranchIndexKey(runId));
        const branches: BranchExecution[] = [];

        for (const branchId of branchIds) {
            const branch = await this.getBranch(runId, branchId);
            if (branch) {
                branches.push(branch);
            }
        }

        return branches;
    }

    async recordStepExecution(runId: string, step: StepExecution): Promise<void> {
        const key = this.getStepKey(runId, step.stepId);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(step));

        // Add to steps list using IndexManager
        await this.indexManager.addToList(
            this.getStepListKey(runId),
            step.stepId,
            "tail",
            this.subsidiaryTtl,
        );
    }

    async getStepExecution(runId: string, stepId: string): Promise<StepExecution | null> {
        const key = this.getStepKey(runId, stepId);
        const data = await redis.get(key);

        return data ? JSON.parse(data) : null;
    }

    async listStepExecutions(runId: string): Promise<StepExecution[]> {
        const stepIds = await this.indexManager.getListMembers(this.getStepListKey(runId));
        const steps: StepExecution[] = [];

        for (const stepId of stepIds) {
            const step = await this.getStepExecution(runId, stepId);
            if (step) {
                steps.push(step);
            }
        }

        return steps;
    }

    async createCheckpoint(runId: string, checkpoint: Checkpoint): Promise<void> {
        const key = this.getCheckpointKey(runId, checkpoint.id);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(checkpoint));

        // Add to checkpoints list using IndexManager
        await this.indexManager.addToList(
            this.getCheckpointListKey(runId),
            checkpoint.id,
            "tail",
            this.subsidiaryTtl,
        );
    }

    async getCheckpoint(runId: string, checkpointId: string): Promise<Checkpoint | null> {
        const key = this.getCheckpointKey(runId, checkpointId);
        const data = await redis.get(key);

        return data ? JSON.parse(data) : null;
    }

    async listCheckpoints(runId: string): Promise<Checkpoint[]> {
        const checkpointIds = await this.indexManager.getListMembers(this.getCheckpointListKey(runId));
        const checkpoints: Checkpoint[] = [];

        for (const checkpointId of checkpointIds) {
            const checkpoint = await this.getCheckpoint(runId, checkpointId);
            if (checkpoint) {
                checkpoints.push(checkpoint);
            }
        }

        return checkpoints;
    }

    async deleteCheckpoint(runId: string, checkpointId: string): Promise<void> {
        const checkpoint = await this.getCheckpoint(runId, checkpointId);
        if (!checkpoint) {
            logger.debug("Checkpoint not found for deletion", { runId, checkpointId });
            return; // Silently succeed if checkpoint doesn't exist
        }

        try {
            // Remove checkpoint data
            const checkpointKey = this.getCheckpointKey(runId, checkpointId);
            await this.redis.del(checkpointKey);

            // Remove from checkpoint list index
            await this.indexManager.removeFromList(
                this.getCheckpointListKey(runId),
                checkpointId,
            );

            logger.info("Checkpoint deleted", { runId, checkpointId });
        } catch (error) {
            logger.error("Failed to delete checkpoint", {
                runId,
                checkpointId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async restoreFromCheckpoint(runId: string, checkpointId: string): Promise<void> {
        const checkpoint = await this.getCheckpoint(runId, checkpointId);
        if (!checkpoint) {
            throw new Error(`Checkpoint ${checkpointId} not found`);
        }

        // Restore context - checkpoint may have legacy ProcessRunContext format
        const currentContext = await this.getRunContext(runId);

        // The checkpoint.context is of type RunContext from shared, which might be ProcessRunContext format
        if (checkpoint.context && typeof checkpoint.context === "object" && "variables" in checkpoint.context) {
            // Merge checkpoint data into the modern context
            currentContext.variables = checkpoint.context.variables || {};
            if ("blackboard" in checkpoint.context) {
                currentContext.blackboard = checkpoint.context.blackboard || {};
            }
            if ("scopes" in checkpoint.context && Array.isArray(checkpoint.context.scopes)) {
                currentContext.scopes = checkpoint.context.scopes;
            }
        }

        await this.updateRunContext(runId, currentContext);

        // Restore location
        await this.updateLocation(runId, checkpoint.location);

        // Update state to RUNNING
        await this.updateRunState(runId, RunState.RUNNING);

        logger.info("Restored from checkpoint", { runId, checkpointId });
    }

    async listActiveRuns(): Promise<string[]> {
        return await this.indexManager.getSetMembers(this.getActiveIndexKey());
    }

    async getRunsByState(state: RunState): Promise<string[]> {
        return await this.indexManager.getSetMembers(this.getStateIndexKey(state));
    }

    async getRunsByUser(userId: string): Promise<string[]> {
        return await this.indexManager.getSetMembers(this.getUserIndexKey(userId));
    }

    /**
     * Clean up all subsidiary data for a run
     */
    private async cleanupRunData(runId: string): Promise<void> {
        try {
            // Clean up all indexes for this run using patterns
            const indexPatterns = [
                `${this.keyPrefix}${runId}:*`,
                `${this.indexPrefix}*`,
            ];

            await this.indexManager.cleanupItemFromAllIndexes(runId, indexPatterns);

            // Clean up individual data keys using pattern
            await this.indexManager.cleanupIndexesByPattern(`${this.keyPrefix}${runId}:*`);

            logger.debug("[RedisRunStateStore] Cleaned up run data", { runId });
        } catch (error) {
            logger.error("[RedisRunStateStore] Failed to cleanup run data", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async updateIndexes(runId: string, config: RunConfig): Promise<void> {
        // Add to active runs set using IndexManager
        await this.indexManager.addToSet(this.getActiveIndexKey(), runId);

        // Add to user's runs
        await this.indexManager.addToSet(this.getUserIndexKey(config.userId), runId);
    }

    private async removeFromIndexes(runId: string, config: RunConfig): Promise<void> {
        // Remove from active runs using IndexManager
        await this.indexManager.removeFromSet(this.getActiveIndexKey(), runId);

        // Remove from user's runs
        await this.indexManager.removeFromSet(this.getUserIndexKey(config.userId), runId);

        // Remove from all state indexes
        for (const state of Object.values(RunState)) {
            await this.indexManager.removeFromSet(this.getStateIndexKey(state as RunState), runId);
        }
    }

    // ========================================
    // CONTEXT CACHING METHODS (Phase 2C)
    // ========================================

    /**
     * Get context from in-memory cache if available and not expired
     */
    private getCachedContext(runId: string): RunExecutionContext | null {
        const cached = this.contextCache.get(runId);

        if (!cached) {
            return null;
        }

        const now = Date.now();
        const isExpired = (now - cached.timestamp) > this.CACHE_TTL_MS;

        if (isExpired) {
            this.contextCache.delete(runId);
            return null;
        }

        // Update access count and timestamp for LRU tracking
        cached.accessCount++;
        cached.timestamp = now;

        logger.debug("[RedisRunStateStore] Context cache hit", {
            runId,
            accessCount: cached.accessCount,
            cacheSize: this.contextCache.size,
        });

        return cached.context;
    }

    /**
     * Store context in in-memory cache with LRU eviction
     */
    private setCachedContext(runId: string, context: RunExecutionContext): void {
        const now = Date.now();

        // If cache is at capacity, evict least recently used item
        if (this.contextCache.size >= this.MAX_CACHE_SIZE) {
            this.evictLeastRecentlyUsed();
        }

        this.contextCache.set(runId, {
            context: { ...context }, // Shallow copy to prevent mutations
            timestamp: now,
            accessCount: 1,
        });

        logger.debug("[RedisRunStateStore] Context cached", {
            runId,
            cacheSize: this.contextCache.size,
        });
    }

    /**
     * Remove context from cache (called when run is deleted)
     */
    private removeCachedContext(runId: string): void {
        this.contextCache.delete(runId);
    }

    /**
     * Evict the least recently used context from cache
     */
    private evictLeastRecentlyUsed(): void {
        let oldestKey: string | null = null;
        let oldestTimestamp = Date.now();

        for (const [runId, cached] of this.contextCache.entries()) {
            if (cached.timestamp < oldestTimestamp) {
                oldestTimestamp = cached.timestamp;
                oldestKey = runId;
            }
        }

        if (oldestKey) {
            this.contextCache.delete(oldestKey);
            logger.debug("[RedisRunStateStore] Evicted context from cache", {
                evictedRunId: oldestKey,
                cacheSize: this.contextCache.size,
            });
        }
    }

    /**
     * Clear entire context cache (useful for testing or memory management)
     */
    private clearContextCache(): void {
        const previousSize = this.contextCache.size;
        this.contextCache.clear();

        logger.info("[RedisRunStateStore] Cleared context cache", {
            previousSize,
        });
    }

    // Key generation helpers
    private getRunKey(runId: string): string {
        return `${this.keyPrefix}${runId}`;
    }

    private getStateKey(runId: string): string {
        return `${this.keyPrefix}${runId}:state`;
    }

    /**
     * @deprecated Only used for legacy context migration
     */
    private getContextKey(runId: string): string {
        return `${this.keyPrefix}${runId}:context`;
    }

    /**
     * Gets the Redis key for modern RunExecutionContext storage
     * 
     * This uses a separate key from the legacy ProcessRunContext to enable
     * side-by-side storage during the migration period.
     */
    private getRunExecutionContextKey(runId: string): string {
        return `${this.keyPrefix}${runId}:run_execution_context`;
    }

    private getCurrentLocationKey(runId: string): string {
        return `${this.keyPrefix}${runId}:current_location`;
    }

    private getLocationHistoryKey(runId: string): string {
        return `${this.keyPrefix}${runId}:location_history`;
    }

    private getBranchKey(runId: string, branchId: string): string {
        return `${this.keyPrefix}${runId}:branch:${branchId}`;
    }

    private getBranchIndexKey(runId: string): string {
        return `${this.keyPrefix}${runId}:branches`;
    }

    private getStepKey(runId: string, stepId: string): string {
        return `${this.keyPrefix}${runId}:step:${stepId}`;
    }

    private getStepListKey(runId: string): string {
        return `${this.keyPrefix}${runId}:steps`;
    }

    private getCheckpointKey(runId: string, checkpointId: string): string {
        return `${this.keyPrefix}${runId}:checkpoint:${checkpointId}`;
    }

    private getCheckpointListKey(runId: string): string {
        return `${this.keyPrefix}${runId}:checkpoints`;
    }

    private getActiveIndexKey(): string {
        return `${this.indexPrefix}active`;
    }

    private getStateIndexKey(state: RunState): string {
        return `${this.indexPrefix}state:${state}`;
    }

    private getUserIndexKey(userId: string): string {
        return `${this.indexPrefix}user:${userId}`;
    }
}

// Singleton instance
let runStateStore: RedisRunStateStore | null = null;

/**
 * Get the run state store instance
 */
export async function getRunStateStore(): Promise<RedisRunStateStore> {
    if (!runStateStore) {
        const redis = await CacheService.get().raw();
        runStateStore = new RedisRunStateStore(redis as Redis);
    }
    return runStateStore;
}

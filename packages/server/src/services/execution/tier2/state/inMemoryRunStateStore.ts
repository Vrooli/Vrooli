/**
 * In-memory run state store for development and testing
 */

import { 
    RunState, 
    type Location,
    type Checkpoint,
    type BranchExecution,
    type StepExecution,
} from "@vrooli/shared";
import { type IRunStateStore, type RunConfig } from "./runStateStore.js";
import { logger } from "../../../../events/logger.js";
import { InMemoryStore } from "../../shared/BaseStore.js";
// Import the modern context type
import { type RunExecutionContext } from "../orchestration/unifiedRunStateMachine.js";

/**
 * In-memory implementation of run state store - implements both legacy and modern interfaces
 * 
 * This implementation provides both IRunStateStore and IModernRunStateStore interfaces
 * with modern RunExecutionContext support for comprehensive context management.
 * 
 * Useful for development and testing without Redis dependency.
 */
export class InMemoryRunStateStore extends InMemoryStore<RunConfig> implements IRunStateStore, IModernRunStateStore {
    private states = new Map<string, RunState>();
    private runExecutionContexts = new Map<string, RunExecutionContext>(); // Modern context storage
    private locations = new Map<string, Location>();
    private locationHistory = new Map<string, Location[]>();
    private branches = new Map<string, Map<string, BranchExecution>>();
    private steps = new Map<string, Map<string, StepExecution>>();
    private stepLists = new Map<string, string[]>();
    private checkpoints = new Map<string, Map<string, Checkpoint>>();
    private checkpointLists = new Map<string, string[]>();
    private activeRuns = new Set<string>();
    private runsByState = new Map<RunState, Set<string>>();
    private runsByUser = new Map<string, Set<string>>();

    constructor() {
        super(logger);
    }

    async initialize(): Promise<void> {
        // No initialization needed for in-memory store
    }
    
    async createRun(runId: string, config: RunConfig): Promise<void> {
        await this.set(runId, config);
        
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
        
        // Add to active runs
        this.activeRuns.add(runId);
        
        // Add to user's runs
        if (!this.runsByUser.has(config.userId)) {
            this.runsByUser.set(config.userId, new Set());
        }
        this.runsByUser.get(config.userId)!.add(runId);
        
        logger.info("Run created (in-memory)", { runId });
    }
    
    async getRun(runId: string): Promise<RunConfig | null> {
        return this.get(runId);
    }
    
    async updateRun(runId: string, updates: Partial<RunConfig>): Promise<void> {
        const current = await this.get(runId);
        if (!current) {
            throw new Error(`Run ${runId} not found`);
        }
        
        const updated = {
            ...current,
            ...updates,
            updatedAt: new Date(),
        };
        
        await this.set(runId, updated);
    }
    
    async deleteRun(runId: string): Promise<void> {
        const config = await this.get(runId);
        if (!config) return;
        
        // Delete all run-related data
        await this.delete(runId);
        this.states.delete(runId);
        this.runExecutionContexts.delete(runId); // Clean up modern context
        this.locations.delete(runId);
        this.locationHistory.delete(runId);
        this.branches.delete(runId);
        this.steps.delete(runId);
        this.stepLists.delete(runId);
        this.checkpoints.delete(runId);
        this.checkpointLists.delete(runId);
        
        // Remove from indexes
        this.activeRuns.delete(runId);
        
        // Remove from state indexes
        for (const stateSet of this.runsByState.values()) {
            stateSet.delete(runId);
        }
        
        // Remove from user's runs
        const userRuns = this.runsByUser.get(config.userId);
        if (userRuns) {
            userRuns.delete(runId);
        }
        
        logger.info("Run deleted (in-memory)", { runId });
    }
    
    async getRunState(runId: string): Promise<RunState> {
        return this.states.get(runId) || RunState.PENDING;
    }
    
    async updateRunState(runId: string, state: RunState): Promise<void> {
        const oldState = this.states.get(runId);
        this.states.set(runId, state);
        
        // Update state index
        if (!this.runsByState.has(state)) {
            this.runsByState.set(state, new Set());
        }
        this.runsByState.get(state)!.add(runId);
        
        // Remove from old state index
        if (oldState && this.runsByState.has(oldState)) {
            this.runsByState.get(oldState)!.delete(runId);
        }
        
        // Remove from active if terminal state
        const terminalStates = [RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED];
        if (terminalStates.includes(state)) {
            this.activeRuns.delete(runId);
        }
    }
    
    // Modern context methods using RunExecutionContext - see getRunContext/updateRunContext below
    
    /**
     * Modern context management methods - Phase 2A implementation
     * 
     * These methods implement the IModernRunStateStore interface for in-memory storage,
     * providing RunExecutionContext-based context management that replaces the deprecated
     * ProcessRunContext methods.
     */
    
    /**
     * Retrieves the complete RunExecutionContext for a run (in-memory implementation)
     * 
     * This method implements the modern context retrieval pattern for in-memory storage,
     * with the same fallback and migration logic as the Redis implementation.
     */
    async getRunContext(runId: string): Promise<RunExecutionContext> {
        // First check for modern context
        const modernContext = this.runExecutionContexts.get(runId);
        if (modernContext) {
            return modernContext;
        }
        
        // No legacy context to migrate in Phase 2B
        const legacyContext = null;
        const runConfig = await this.get(runId);
        
        if (legacyContext && runConfig) {
            logger.info("[InMemoryRunStateStore] Migrating legacy context to RunExecutionContext", { runId });
            
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
            
            return migratedContext;
        }
        
        // Final fallback: Create minimal default context
        logger.warn("[InMemoryRunStateStore] Creating default RunExecutionContext", { runId });
        
        const defaultContext: RunExecutionContext = {
            runId,
            routineId: runConfig?.routineId || runId, // Fallback to runId if no routine info available
            
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
        
        return defaultContext;
    }
    
    /**
     * Updates the complete RunExecutionContext for a run (in-memory implementation)
     * 
     * This method implements the modern context storage pattern for in-memory storage,
     * storing the comprehensive execution state directly in the Map.
     */
    async updateRunContext(runId: string, context: RunExecutionContext): Promise<void> {
        // Deep clone the context to prevent external mutations
        const clonedContext: RunExecutionContext = {
            ...context,
            variables: { ...context.variables },
            outputs: { ...context.outputs },
            blackboard: { ...context.blackboard },
            scopes: context.scopes.map(scope => ({ ...scope, variables: { ...scope.variables } })),
            completedSteps: [...context.completedSteps],
            visitedLocations: [...context.visitedLocations],
            parallelBranches: [...context.parallelBranches],
            resourceLimits: { ...context.resourceLimits },
            resourceUsage: { 
                ...context.resourceUsage,
                startTime: new Date(context.resourceUsage.startTime), // Ensure proper Date object
            },
            progress: { 
                ...context.progress,
                completedSteps: [...context.progress.completedSteps],
            },
        };
        
        this.runExecutionContexts.set(runId, clonedContext);
        
        logger.debug("[InMemoryRunStateStore] Updated RunExecutionContext", {
            runId,
            currentStepId: context.progress.currentStepId,
            percentComplete: context.progress.percentComplete,
        });
    }
    
    async getCurrentLocation(runId: string): Promise<Location | null> {
        return this.locations.get(runId) || null;
    }
    
    async updateLocation(runId: string, location: Location): Promise<void> {
        // Update current location
        this.locations.set(runId, location);
        
        // Add to location history
        if (!this.locationHistory.has(runId)) {
            this.locationHistory.set(runId, []);
        }
        this.locationHistory.get(runId)!.push(location);
    }
    
    async getLocationHistory(runId: string): Promise<Location[]> {
        return this.locationHistory.get(runId) || [];
    }
    
    async createBranch(runId: string, branch: BranchExecution): Promise<void> {
        if (!this.branches.has(runId)) {
            this.branches.set(runId, new Map());
        }
        this.branches.get(runId)!.set(branch.id, branch);
    }
    
    async getBranch(runId: string, branchId: string): Promise<BranchExecution | null> {
        const runBranches = this.branches.get(runId);
        return runBranches?.get(branchId) || null;
    }
    
    async updateBranch(runId: string, branchId: string, updates: Partial<BranchExecution>): Promise<void> {
        const runBranches = this.branches.get(runId);
        if (!runBranches) {
            throw new Error(`Run ${runId} has no branches`);
        }
        
        const current = runBranches.get(branchId);
        if (!current) {
            throw new Error(`Branch ${branchId} not found`);
        }
        
        runBranches.set(branchId, { ...current, ...updates });
    }
    
    async listBranches(runId: string): Promise<BranchExecution[]> {
        const runBranches = this.branches.get(runId);
        return runBranches ? Array.from(runBranches.values()) : [];
    }
    
    async recordStepExecution(runId: string, step: StepExecution): Promise<void> {
        if (!this.steps.has(runId)) {
            this.steps.set(runId, new Map());
        }
        this.steps.get(runId)!.set(step.stepId, step);
        
        // Add to step list
        if (!this.stepLists.has(runId)) {
            this.stepLists.set(runId, []);
        }
        this.stepLists.get(runId)!.push(step.stepId);
    }
    
    async getStepExecution(runId: string, stepId: string): Promise<StepExecution | null> {
        const runSteps = this.steps.get(runId);
        return runSteps?.get(stepId) || null;
    }
    
    async listStepExecutions(runId: string): Promise<StepExecution[]> {
        const stepIds = this.stepLists.get(runId) || [];
        const runSteps = this.steps.get(runId);
        if (!runSteps) return [];
        
        return stepIds
            .map(id => runSteps.get(id))
            .filter((step): step is StepExecution => !!step);
    }
    
    async createCheckpoint(runId: string, checkpoint: Checkpoint): Promise<void> {
        if (!this.checkpoints.has(runId)) {
            this.checkpoints.set(runId, new Map());
        }
        this.checkpoints.get(runId)!.set(checkpoint.id, checkpoint);
        
        // Add to checkpoint list
        if (!this.checkpointLists.has(runId)) {
            this.checkpointLists.set(runId, []);
        }
        this.checkpointLists.get(runId)!.push(checkpoint.id);
    }
    
    async getCheckpoint(runId: string, checkpointId: string): Promise<Checkpoint | null> {
        const runCheckpoints = this.checkpoints.get(runId);
        return runCheckpoints?.get(checkpointId) || null;
    }
    
    async listCheckpoints(runId: string): Promise<Checkpoint[]> {
        const checkpointIds = this.checkpointLists.get(runId) || [];
        const runCheckpoints = this.checkpoints.get(runId);
        if (!runCheckpoints) return [];
        
        return checkpointIds
            .map(id => runCheckpoints.get(id))
            .filter((checkpoint): checkpoint is Checkpoint => !!checkpoint);
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
        
        logger.info("Restored from checkpoint (in-memory)", { runId, checkpointId });
    }
    
    async listActiveRuns(): Promise<string[]> {
        return Array.from(this.activeRuns);
    }
    
    async getRunsByState(state: RunState): Promise<string[]> {
        const runs = this.runsByState.get(state);
        return runs ? Array.from(runs) : [];
    }
    
    async getRunsByUser(userId: string): Promise<string[]> {
        const runs = this.runsByUser.get(userId);
        return runs ? Array.from(runs) : [];
    }
}

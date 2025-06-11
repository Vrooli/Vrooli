/**
 * Run state store
 * Manages persistent state for routine execution
 */

import { type Redis } from "ioredis";
import { 
    RunState, 
    type Location,
    type Checkpoint,
    type BranchExecution,
    type StepExecution,
} from "@vrooli/shared";
import { type ProcessRunContext } from "../context/contextManager.js";
import { redis } from "../../../../redisConn.js";
import { logger } from "../../../../events/logger.js";
import { RedisStore } from "../../shared/BaseStore.js";
import { RedisIndexManager } from "../../shared/RedisIndexManager.js";

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
 * Run state store interface
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
    
    // Context management
    getContext(runId: string): Promise<ProcessRunContext>;
    updateContext(runId: string, context: ProcessRunContext): Promise<void>;
    setVariable(runId: string, name: string, value: unknown): Promise<void>;
    getVariable(runId: string, name: string): Promise<unknown>;
    
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
    restoreFromCheckpoint(runId: string, checkpointId: string): Promise<void>;
    
    // Query operations
    listActiveRuns(): Promise<string[]>;
    getRunsByState(state: RunState): Promise<string[]>;
    getRunsByUser(userId: string): Promise<string[]>;
}

/**
 * Redis-based run state store
 */
export class RedisRunStateStore extends RedisStore<RunConfig> implements IRunStateStore {
    private readonly indexPrefix = "run_index:";
    private readonly subsidiaryTtl = 86400; // 24 hours
    private readonly indexManager: RedisIndexManager;
    
    constructor() {
        super(logger, redis, "run", 86400); // 24 hours TTL
        this.indexManager = new RedisIndexManager(redis, logger, 86400);
    }
    
    async initialize(): Promise<void> {
        // Base store doesn't need explicit initialization
    }
    
    async createRun(runId: string, config: RunConfig): Promise<void> {
        await this.set(runId, config);
        
        // Initialize state
        await this.updateRunState(runId, RunState.PENDING);
        
        // Initialize empty context
        await this.updateContext(runId, {
            variables: config.inputs,
            blackboard: {},
            scopes: [{
                id: "global",
                name: "Global Scope",
                variables: config.inputs,
            }],
        });
        
        // Update indexes
        await this.updateIndexes(runId, config);
        
        logger.info("Run created", { runId });
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
        
        // Update indexes if userId changed
        if (updates.userId && updates.userId !== current.userId) {
            await this.updateIndexes(runId, updated);
        }
    }
    
    async deleteRun(runId: string): Promise<void> {
        const config = await this.get(runId);
        if (!config) return;
        
        // Remove from indexes first
        await this.removeFromIndexes(runId, config);
        
        // Clean up subsidiary data
        await this.cleanupRunData(runId);
        
        // Delete main run
        await this.delete(runId);
        
        logger.info("Run deleted", { runId });
    }
    
    async getRunState(runId: string): Promise<RunState> {
        const key = this.getStateKey(runId);
        const state = await redis.get(key);
        
        return (state as RunState) || RunState.PENDING;
    }
    
    async updateRunState(runId: string, state: RunState): Promise<void> {
        const oldState = await this.getRunState(runId);
        const key = this.getStateKey(runId);
        await redis.setex(key, this.subsidiaryTtl, state);
        
        // Update state index using IndexManager
        await this.indexManager.updateStateIndex(
            runId,
            oldState === RunState.PENDING ? null : oldState,
            state,
            (s) => this.getStateIndexKey(s),
            Object.values(RunState)
        );
        
        // Remove from active if terminal state
        const terminalStates = [RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED];
        if (terminalStates.includes(state)) {
            await this.indexManager.removeFromSet(this.getActiveIndexKey(), runId);
        }
    }
    
    async getContext(runId: string): Promise<ProcessRunContext> {
        const key = this.getContextKey(runId);
        const data = await redis.get(key);
        
        if (!data) {
            return {
                variables: {},
                blackboard: {},
                scopes: [{
                    id: "global",
                    name: "Global Scope",
                    variables: {},
                }],
            };
        }
        
        return JSON.parse(data);
    }
    
    async updateContext(runId: string, context: ProcessRunContext): Promise<void> {
        const key = this.getContextKey(runId);
        await redis.setex(key, this.subsidiaryTtl, JSON.stringify(context));
    }
    
    async setVariable(runId: string, name: string, value: unknown): Promise<void> {
        const context = await this.getContext(runId);
        context.variables[name] = value;
        await this.updateContext(runId, context);
    }
    
    async getVariable(runId: string, name: string): Promise<unknown> {
        const context = await this.getContext(runId);
        return context.variables[name];
    }
    
    async getCurrentLocation(runId: string): Promise<Location | null> {
        const key = this.getCurrentLocationKey(runId);
        const data = await redis.get(key);
        
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
            'tail',
            this.subsidiaryTtl
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
            'tail',
            this.subsidiaryTtl
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
    
    async restoreFromCheckpoint(runId: string, checkpointId: string): Promise<void> {
        const checkpoint = await this.getCheckpoint(runId, checkpointId);
        if (!checkpoint) {
            throw new Error(`Checkpoint ${checkpointId} not found`);
        }
        
        // Restore context
        await this.updateContext(runId, checkpoint.context);
        
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

    // Key generation helpers
    private getStateKey(runId: string): string {
        return `${this.keyPrefix}${runId}:state`;
    }

    private getContextKey(runId: string): string {
        return `${this.keyPrefix}${runId}:context`;
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
export function getRunStateStore(): RedisRunStateStore {
    if (!runStateStore) {
        runStateStore = new RedisRunStateStore();
    }
    return runStateStore;
}

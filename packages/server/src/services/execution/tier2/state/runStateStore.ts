/**
 * Run state store
 * Manages persistent state for routine execution
 */

import { type Redis } from "ioredis";
import { 
    RunState, 
    type RunContext,
    type Location,
    type Checkpoint,
    type BranchExecution,
    type StepExecution,
} from "@vrooli/shared";
import { getRedisConnection } from "../../../../redisConn.js";
import { logger } from "../../../../events/logger.js";

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
    // Run lifecycle
    createRun(runId: string, config: RunConfig): Promise<void>;
    getRun(runId: string): Promise<RunConfig | null>;
    updateRun(runId: string, updates: Partial<RunConfig>): Promise<void>;
    deleteRun(runId: string): Promise<void>;
    
    // State management
    getRunState(runId: string): Promise<RunState>;
    updateRunState(runId: string, state: RunState): Promise<void>;
    
    // Context management
    getContext(runId: string): Promise<RunContext>;
    updateContext(runId: string, context: RunContext): Promise<void>;
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
export class RedisRunStateStore implements IRunStateStore {
    private client: Redis | null = null;
    private readonly prefix = "run:state:";
    private readonly ttl = 86400; // 24 hours
    
    async initialize(): Promise<void> {
        this.client = await getRedisConnection();
    }
    
    private getKey(runId: string, ...parts: string[]): string {
        return [this.prefix, runId, ...parts].join(":");
    }
    
    async createRun(runId: string, config: RunConfig): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "config");
        await this.client.setex(key, this.ttl, JSON.stringify(config));
        
        // Initialize state
        await this.updateRunState(runId, RunState.PENDING);
        
        // Initialize empty context
        await this.updateContext(runId, {
            variables: {},
            inputs: config.inputs,
            outputs: {},
            parentContext: null,
        });
        
        // Add to active runs set
        await this.client.sadd(this.prefix + "active", runId);
        
        // Add to user's runs
        await this.client.sadd(this.prefix + "by-user:" + config.userId, runId);
        
        logger.info("Run created", { runId });
    }
    
    async getRun(runId: string): Promise<RunConfig | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "config");
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateRun(runId: string, updates: Partial<RunConfig>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const current = await this.getRun(runId);
        if (!current) {
            throw new Error(`Run ${runId} not found`);
        }
        
        const updated = {
            ...current,
            ...updates,
            updatedAt: new Date(),
        };
        
        const key = this.getKey(runId, "config");
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async deleteRun(runId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const config = await this.getRun(runId);
        if (!config) return;
        
        // Delete all run-related keys
        const pattern = this.getKey(runId, "*");
        const keys = await this.client.keys(pattern);
        
        if (keys.length > 0) {
            await this.client.del(...keys);
        }
        
        // Remove from active runs
        await this.client.srem(this.prefix + "active", runId);
        
        // Remove from user's runs
        await this.client.srem(this.prefix + "by-user:" + config.userId, runId);
        
        logger.info("Run deleted", { runId });
    }
    
    async getRunState(runId: string): Promise<RunState> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "state");
        const state = await this.client.get(key);
        
        return (state as RunState) || RunState.PENDING;
    }
    
    async updateRunState(runId: string, state: RunState): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "state");
        await this.client.setex(key, this.ttl, state);
        
        // Update state index
        const stateKey = this.prefix + "by-state:" + state;
        await this.client.sadd(stateKey, runId);
        
        // Remove from other state indexes
        for (const s of Object.values(RunState)) {
            if (s !== state) {
                const oldStateKey = this.prefix + "by-state:" + s;
                await this.client.srem(oldStateKey, runId);
            }
        }
        
        // Remove from active if terminal state
        const terminalStates = [RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED];
        if (terminalStates.includes(state)) {
            await this.client.srem(this.prefix + "active", runId);
        }
    }
    
    async getContext(runId: string): Promise<RunContext> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "context");
        const data = await this.client.get(key);
        
        if (!data) {
            return {
                variables: {},
                inputs: {},
                outputs: {},
                parentContext: null,
            };
        }
        
        return JSON.parse(data);
    }
    
    async updateContext(runId: string, context: RunContext): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "context");
        await this.client.setex(key, this.ttl, JSON.stringify(context));
    }
    
    async setVariable(runId: string, name: string, value: unknown): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const context = await this.getContext(runId);
        context.variables[name] = value;
        await this.updateContext(runId, context);
    }
    
    async getVariable(runId: string, name: string): Promise<unknown> {
        if (!this.client) throw new Error("Store not initialized");
        
        const context = await this.getContext(runId);
        return context.variables[name];
    }
    
    async getCurrentLocation(runId: string): Promise<Location | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "current-location");
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateLocation(runId: string, location: Location): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        // Update current location
        const key = this.getKey(runId, "current-location");
        await this.client.setex(key, this.ttl, JSON.stringify(location));
        
        // Add to location history
        const historyKey = this.getKey(runId, "location-history");
        await this.client.rpush(historyKey, JSON.stringify(location));
        await this.client.expire(historyKey, this.ttl);
    }
    
    async getLocationHistory(runId: string): Promise<Location[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "location-history");
        const history = await this.client.lrange(key, 0, -1);
        
        return history.map(item => JSON.parse(item));
    }
    
    async createBranch(runId: string, branch: BranchExecution): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "branches", branch.id);
        await this.client.setex(key, this.ttl, JSON.stringify(branch));
        
        // Add to branches set
        await this.client.sadd(this.getKey(runId, "branch-ids"), branch.id);
    }
    
    async getBranch(runId: string, branchId: string): Promise<BranchExecution | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "branches", branchId);
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateBranch(runId: string, branchId: string, updates: Partial<BranchExecution>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const current = await this.getBranch(runId, branchId);
        if (!current) {
            throw new Error(`Branch ${branchId} not found`);
        }
        
        const updated = { ...current, ...updates };
        const key = this.getKey(runId, "branches", branchId);
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async listBranches(runId: string): Promise<BranchExecution[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const branchIds = await this.client.smembers(this.getKey(runId, "branch-ids"));
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
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "steps", step.stepId);
        await this.client.setex(key, this.ttl, JSON.stringify(step));
        
        // Add to steps list
        const listKey = this.getKey(runId, "step-list");
        await this.client.rpush(listKey, step.stepId);
        await this.client.expire(listKey, this.ttl);
    }
    
    async getStepExecution(runId: string, stepId: string): Promise<StepExecution | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "steps", stepId);
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async listStepExecutions(runId: string): Promise<StepExecution[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const listKey = this.getKey(runId, "step-list");
        const stepIds = await this.client.lrange(listKey, 0, -1);
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
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "checkpoints", checkpoint.id);
        await this.client.setex(key, this.ttl, JSON.stringify(checkpoint));
        
        // Add to checkpoints list
        const listKey = this.getKey(runId, "checkpoint-list");
        await this.client.rpush(listKey, checkpoint.id);
        await this.client.expire(listKey, this.ttl);
    }
    
    async getCheckpoint(runId: string, checkpointId: string): Promise<Checkpoint | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(runId, "checkpoints", checkpointId);
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async listCheckpoints(runId: string): Promise<Checkpoint[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const listKey = this.getKey(runId, "checkpoint-list");
        const checkpointIds = await this.client.lrange(listKey, 0, -1);
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
        if (!this.client) throw new Error("Store not initialized");
        
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
        if (!this.client) throw new Error("Store not initialized");
        
        return await this.client.smembers(this.prefix + "active");
    }
    
    async getRunsByState(state: RunState): Promise<string[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.prefix + "by-state:" + state;
        return await this.client.smembers(key);
    }
    
    async getRunsByUser(userId: string): Promise<string[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.prefix + "by-user:" + userId;
        return await this.client.smembers(key);
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

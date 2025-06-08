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
import { type ProcessRunContext } from "../context/contextManager.js";
import { logger } from "../../../../events/logger.js";

/**
 * In-memory implementation of run state store
 * Useful for development and testing without Redis dependency
 */
export class InMemoryRunStateStore implements IRunStateStore {
    private runs = new Map<string, RunConfig>();
    private states = new Map<string, RunState>();
    private contexts = new Map<string, ProcessRunContext>();
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
    
    async createRun(runId: string, config: RunConfig): Promise<void> {
        this.runs.set(runId, config);
        
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
        return this.runs.get(runId) || null;
    }
    
    async updateRun(runId: string, updates: Partial<RunConfig>): Promise<void> {
        const current = this.runs.get(runId);
        if (!current) {
            throw new Error(`Run ${runId} not found`);
        }
        
        const updated = {
            ...current,
            ...updates,
            updatedAt: new Date(),
        };
        
        this.runs.set(runId, updated);
    }
    
    async deleteRun(runId: string): Promise<void> {
        const config = this.runs.get(runId);
        if (!config) return;
        
        // Delete all run-related data
        this.runs.delete(runId);
        this.states.delete(runId);
        this.contexts.delete(runId);
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
    
    async getContext(runId: string): Promise<ProcessRunContext> {
        return this.contexts.get(runId) || {
            variables: {},
            blackboard: {},
            scopes: [{
                id: "global",
                name: "Global Scope",
                variables: {},
            }],
        };
    }
    
    async updateContext(runId: string, context: ProcessRunContext): Promise<void> {
        this.contexts.set(runId, context);
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
        
        // Restore context
        await this.updateContext(runId, checkpoint.context);
        
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
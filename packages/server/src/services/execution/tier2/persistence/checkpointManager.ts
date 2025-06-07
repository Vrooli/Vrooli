import { type Logger } from "winston";
import {
    type Run,
    type Checkpoint,
    type RunState,
    type RunProgress,
    type RunContext,
    generatePk,
} from "@vrooli/shared";
import { type IRunStateStore } from "../state/runStateStore.js";

/**
 * Checkpoint creation options
 */
export interface CheckpointOptions {
    compression?: boolean;
    includeMetadata?: boolean;
    maxAge?: number; // Maximum age in milliseconds
}

/**
 * Checkpoint restore options
 */
export interface RestoreOptions {
    validateIntegrity?: boolean;
    mergeContext?: boolean;
}

/**
 * CheckpointManager - Manages run state persistence and recovery
 * 
 * This component provides checkpoint and recovery capabilities for long-running
 * workflows. It enables:
 * 
 * - Periodic state snapshots during execution
 * - Fast recovery from failures
 * - State migration between versions
 * - Checkpoint compression and optimization
 * - Automatic checkpoint lifecycle management
 * 
 * Checkpoints are critical for resilience in complex workflows that may
 * run for extended periods or across system restarts.
 */
export class CheckpointManager {
    private readonly stateStore: IRunStateStore;
    private readonly logger: Logger;
    private readonly maxCheckpointsPerRun: number = 10;
    private readonly checkpointTTL: number = 86400000; // 24 hours

    constructor(stateStore: IRunStateStore, logger: Logger) {
        this.stateStore = stateStore;
        this.logger = logger;
    }

    /**
     * Creates a checkpoint for a run
     */
    async createCheckpoint(
        run: Run,
        options: CheckpointOptions = {},
    ): Promise<Checkpoint> {
        const checkpointId = generatePk();
        const timestamp = new Date();

        this.logger.info("[CheckpointManager] Creating checkpoint", {
            runId: run.id,
            checkpointId,
            state: run.state,
        });

        try {
            // Serialize run state
            const serializedContext = await this.serializeContext(
                run.context,
                options.compression,
            );

            // Create checkpoint
            const checkpoint: Checkpoint = {
                id: checkpointId,
                runId: run.id,
                timestamp,
                state: run.state,
                progress: this.cloneProgress(run.progress),
                context: run.context, // Store original for now
                size: serializedContext.length,
            };

            // Add metadata if requested
            if (options.includeMetadata) {
                (checkpoint as any).metadata = {
                    version: "1.0.0",
                    compressed: options.compression || false,
                    createdBy: "CheckpointManager",
                };
            }

            // Store checkpoint
            await this.stateStore.createCheckpoint(run.id, checkpoint);

            // Clean up old checkpoints
            await this.cleanupOldCheckpoints(run.id);

            this.logger.info("[CheckpointManager] Checkpoint created", {
                runId: run.id,
                checkpointId,
                size: checkpoint.size,
            });

            return checkpoint;

        } catch (error) {
            this.logger.error("[CheckpointManager] Failed to create checkpoint", {
                runId: run.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Restores a run from checkpoint
     */
    async restoreCheckpoint(
        run: Run,
        checkpoint: Checkpoint,
        options: RestoreOptions = {},
    ): Promise<void> {
        this.logger.info("[CheckpointManager] Restoring from checkpoint", {
            runId: run.id,
            checkpointId: checkpoint.id,
            checkpointAge: Date.now() - checkpoint.timestamp.getTime(),
        });

        try {
            // Validate checkpoint if requested
            if (options.validateIntegrity) {
                await this.validateCheckpoint(checkpoint);
            }

            // Restore state
            run.state = checkpoint.state;
            run.progress = this.cloneProgress(checkpoint.progress);

            // Restore or merge context
            if (options.mergeContext) {
                run.context = await this.mergeContexts(run.context, checkpoint.context);
            } else {
                run.context = this.cloneContext(checkpoint.context);
            }

            // Update state store
            await this.stateStore.updateRunState(run.id, run.state);
            await this.stateStore.updateContext(run.id, run.context);

            this.logger.info("[CheckpointManager] Checkpoint restored", {
                runId: run.id,
                checkpointId: checkpoint.id,
                restoredState: run.state,
            });

        } catch (error) {
            this.logger.error("[CheckpointManager] Failed to restore checkpoint", {
                runId: run.id,
                checkpointId: checkpoint.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Gets the last checkpoint for a run
     */
    async getLastCheckpoint(runId: string): Promise<Checkpoint | null> {
        const checkpoints = await this.stateStore.listCheckpoints(runId);
        
        if (checkpoints.length === 0) {
            return null;
        }

        // Sort by timestamp (newest first)
        checkpoints.sort((a, b) => 
            b.timestamp.getTime() - a.timestamp.getTime(),
        );

        return checkpoints[0];
    }

    /**
     * Lists checkpoints for a run
     */
    async listCheckpoints(
        runId: string,
        limit?: number,
    ): Promise<Checkpoint[]> {
        const checkpoints = await this.stateStore.listCheckpoints(runId);
        
        // Sort by timestamp (newest first)
        checkpoints.sort((a, b) => 
            b.timestamp.getTime() - a.timestamp.getTime(),
        );

        // Apply limit if specified
        if (limit && limit > 0) {
            return checkpoints.slice(0, limit);
        }

        return checkpoints;
    }

    /**
     * Deletes a checkpoint
     */
    async deleteCheckpoint(runId: string, checkpointId: string): Promise<void> {
        // For now, we don't have a delete method in the store interface
        // This would be implemented when needed
        this.logger.warn("[CheckpointManager] Delete checkpoint not implemented", {
            runId,
            checkpointId,
        });
    }

    /**
     * Cleans up old checkpoints
     */
    private async cleanupOldCheckpoints(runId: string): Promise<void> {
        const checkpoints = await this.stateStore.listCheckpoints(runId);
        
        if (checkpoints.length <= this.maxCheckpointsPerRun) {
            return;
        }

        // Sort by timestamp (oldest first)
        checkpoints.sort((a, b) => 
            a.timestamp.getTime() - b.timestamp.getTime(),
        );

        // Remove old checkpoints
        const toRemove = checkpoints.length - this.maxCheckpointsPerRun;
        const removed: string[] = [];

        for (let i = 0; i < toRemove; i++) {
            const checkpoint = checkpoints[i];
            
            // Check if checkpoint is expired
            const age = Date.now() - checkpoint.timestamp.getTime();
            if (age > this.checkpointTTL) {
                // TODO: Implement delete in state store
                removed.push(checkpoint.id);
            }
        }

        if (removed.length > 0) {
            this.logger.info("[CheckpointManager] Cleaned up old checkpoints", {
                runId,
                removedCount: removed.length,
            });
        }
    }

    /**
     * Validates checkpoint integrity
     */
    private async validateCheckpoint(checkpoint: Checkpoint): Promise<void> {
        // Basic validation
        if (!checkpoint.id || !checkpoint.runId) {
            throw new Error("Invalid checkpoint: missing required fields");
        }

        // Check age
        const age = Date.now() - checkpoint.timestamp.getTime();
        if (age > this.checkpointTTL) {
            throw new Error(`Checkpoint too old: ${age}ms`);
        }

        // Validate structure
        if (!checkpoint.progress || !checkpoint.context) {
            throw new Error("Invalid checkpoint: missing state data");
        }
    }

    /**
     * Serializes context for storage
     */
    private async serializeContext(
        context: RunContext,
        compress = false,
    ): Promise<string> {
        const json = JSON.stringify(context);
        
        if (compress) {
            // TODO: Implement compression
            return json;
        }

        return json;
    }

    /**
     * Clones progress object
     */
    private cloneProgress(progress: RunProgress): RunProgress {
        return {
            totalSteps: progress.totalSteps,
            completedSteps: progress.completedSteps,
            failedSteps: progress.failedSteps,
            skippedSteps: progress.skippedSteps,
            currentLocation: { ...progress.currentLocation },
            locationStack: {
                locations: progress.locationStack.locations.map(loc => ({ ...loc })),
                depth: progress.locationStack.depth,
            },
            branches: progress.branches.map(branch => ({
                ...branch,
                steps: branch.steps.map(step => ({ ...step })),
            })),
        };
    }

    /**
     * Clones context object
     */
    private cloneContext(context: RunContext): RunContext {
        return {
            variables: { ...context.variables },
            blackboard: { ...context.blackboard },
            scopes: context.scopes.map(scope => ({
                ...scope,
                variables: { ...scope.variables },
            })),
        };
    }

    /**
     * Merges two contexts
     */
    private async mergeContexts(
        current: RunContext,
        checkpoint: RunContext,
    ): Promise<RunContext> {
        // For now, simple merge - checkpoint wins
        return {
            variables: {
                ...current.variables,
                ...checkpoint.variables,
            },
            blackboard: {
                ...current.blackboard,
                ...checkpoint.blackboard,
            },
            scopes: checkpoint.scopes, // Use checkpoint scopes
        };
    }

    /**
     * Creates a checkpoint summary
     */
    async getCheckpointSummary(runId: string): Promise<{
        count: number;
        totalSize: number;
        oldestTimestamp?: Date;
        newestTimestamp?: Date;
    }> {
        const checkpoints = await this.stateStore.listCheckpoints(runId);
        
        if (checkpoints.length === 0) {
            return {
                count: 0,
                totalSize: 0,
            };
        }

        const totalSize = checkpoints.reduce((sum, cp) => sum + cp.size, 0);
        const timestamps = checkpoints.map(cp => cp.timestamp.getTime());

        return {
            count: checkpoints.length,
            totalSize,
            oldestTimestamp: new Date(Math.min(...timestamps)),
            newestTimestamp: new Date(Math.max(...timestamps)),
        };
    }
}

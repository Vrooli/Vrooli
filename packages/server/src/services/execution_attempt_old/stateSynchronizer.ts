import { type BlackboardItem, type SwarmResource, type SwarmSubTask, type ToolCallRecord } from "@local/shared";
import { type Logger } from "winston";
import { logger } from "../../events/logger.js";
import { type ConversationStateStore } from "../conversation/chatStore.js";
import { type ExecutionContext } from "./executionContext.js";
import { type SubroutineExecutionResult } from "./unifiedExecutionEngine.js";

/**
 * Configuration for the state synchronizer.
 */
export interface StateSynchronizerConfig {
    /** Logger instance */
    logger?: Logger;
    /** Whether to auto-sync on every update */
    autoSync?: boolean;
    /** Debounce time for auto-sync in milliseconds */
    syncDebounceMs?: number;
}

/**
 * Synchronizes state between different execution tiers (swarms, routines, subroutines).
 * 
 * Responsibilities:
 * - Propagate routine results back to parent swarms
 * - Update shared blackboard/resources
 * - Manage subtask status updates
 * - Handle state conflicts and merging
 */
export class StateSynchronizer {
    private readonly logger: Logger;
    private readonly config: StateSynchronizerConfig;
    private readonly pendingSyncs = new Map<string, NodeJS.Timeout>();

    constructor(
        private readonly conversationStore: ConversationStateStore,
        config?: StateSynchronizerConfig,
    ) {
        this.config = {
            autoSync: true,
            syncDebounceMs: 1000, // 1 second default debounce
            ...config,
        };
        this.logger = config?.logger || logger;
    }

    /**
     * Synchronizes routine execution results back to the parent swarm.
     */
    async syncRoutineResultsToSwarm(
        swarmId: string,
        routineResult: SubroutineExecutionResult,
        context: ExecutionContext,
    ): Promise<void> {
        if (!context.parentSwarmContext) {
            this.logger.debug("No parent swarm context to sync to");
            return;
        }

        this.logger.info(`Syncing routine results to swarm ${swarmId}`, {
            routine: context.routine.id,
            success: routineResult.success,
        });

        try {
            // Get current swarm state
            const swarmState = await this.conversationStore.get(swarmId);
            if (!swarmState) {
                throw new Error(`Swarm state not found for ${swarmId}`);
            }

            // Create a mutable copy of the config
            const updatedConfig = { ...swarmState.config };

            // Update blackboard with routine outputs
            if (routineResult.ioMapping?.outputs) {
                updatedConfig.blackboard = await this.mergeBlackboard(
                    updatedConfig.blackboard || [],
                    routineResult.ioMapping.outputs,
                );
            }

            // Add created resources
            if (routineResult.createdResources && routineResult.createdResources.length > 0) {
                updatedConfig.resources = await this.mergeResources(
                    updatedConfig.resources || [],
                    routineResult.createdResources,
                    context,
                );
            }

            // Add tool call record
            if (context.routine.id) {
                const toolRecord: ToolCallRecord = {
                    id: `routine_${Date.now()}_${context.subroutineInstanceId}`,
                    routine_id: context.routine.id,
                    routine_name: context.routine.translations?.[0]?.name || "Unknown Routine",
                    params: this.extractRoutineParams(context),
                    output_resource_ids: routineResult.createdResources || [],
                    caller_bot_id: context.executingBotId || "system",
                    created_at: new Date().toISOString(),
                };
                updatedConfig.records = [...(updatedConfig.records || []), toolRecord];
            }

            // Update subtask status if this routine was for a subtask
            if (context.parentSwarmContext && routineResult.metadata && "subtaskId" in routineResult.metadata) {
                const subtaskId = (routineResult.metadata as any).subtaskId;
                if (typeof subtaskId === "string") {
                    updatedConfig.subtasks = this.updateSubtaskStatus(
                        updatedConfig.subtasks || [],
                        subtaskId,
                        routineResult.success ? "done" : "failed",
                    );
                }
            }

            // Update credits used
            if (updatedConfig.stats) {
                updatedConfig.stats.totalCredits = (
                    BigInt(updatedConfig.stats.totalCredits) + routineResult.creditsUsed
                ).toString();
                updatedConfig.stats.totalToolCalls++;
            }

            // Save updated config
            await this.conversationStore.updateConfig(swarmId, updatedConfig);

            this.logger.info(`Successfully synced routine results to swarm ${swarmId}`);

        } catch (error) {
            this.logger.error(`Failed to sync routine results to swarm ${swarmId}`, error);
            throw error;
        }
    }

    /**
     * Synchronizes subroutine results to parent routine context.
     */
    async syncSubroutineToRoutine(
        parentRoutineId: string,
        subroutineResult: SubroutineExecutionResult,
        context: ExecutionContext,
    ): Promise<void> {
        if (!context.parentRoutineContext) {
            this.logger.debug("No parent routine context to sync to");
            return;
        }

        this.logger.info(`Syncing subroutine results to routine ${parentRoutineId}`, {
            subroutine: context.routine.id,
            success: subroutineResult.success,
        });

        // Update parent routine context data
        const updatedContextData = {
            ...context.parentRoutineContext.contextData,
            [`subroutine_${context.subroutineInstanceId}_result`]: {
                success: subroutineResult.success,
                outputs: subroutineResult.ioMapping.outputs,
                error: subroutineResult.error,
                creditsUsed: subroutineResult.creditsUsed.toString(),
                timeElapsed: subroutineResult.timeElapsed,
            },
        };

        // In a real implementation, this would update the parent routine's state
        // For now, we'll just log it
        this.logger.info("Updated parent routine context", { updatedContextData });
    }

    /**
     * Schedules a debounced sync operation.
     */
    scheduleDebouncedSync(
        syncKey: string,
        syncFn: () => Promise<void>,
    ): void {
        // Cancel any pending sync for this key
        const existingTimeout = this.pendingSyncs.get(syncKey);
        if (existingTimeout) {
            clearTimeout(existingTimeout);
        }

        // Schedule new sync
        const timeout = setTimeout(async () => {
            this.pendingSyncs.delete(syncKey);
            try {
                await syncFn();
            } catch (error) {
                this.logger.error(`Debounced sync failed for ${syncKey}`, error);
            }
        }, this.config.syncDebounceMs);

        this.pendingSyncs.set(syncKey, timeout);
    }

    /**
     * Merges blackboard items from routine outputs.
     */
    private async mergeBlackboard(
        currentBlackboard: BlackboardItem[],
        routineOutputs: Record<string, { value?: unknown }>,
    ): Promise<BlackboardItem[]> {
        const newItems: BlackboardItem[] = [];
        const now = new Date().toISOString();

        for (const [key, output] of Object.entries(routineOutputs)) {
            if (output.value !== undefined) {
                newItems.push({
                    id: `output_${key}_${Date.now()}`,
                    value: output.value,
                    created_at: now,
                });
            }
        }

        return [...currentBlackboard, ...newItems];
    }

    /**
     * Merges new resources with existing ones.
     */
    private async mergeResources(
        currentResources: SwarmResource[],
        newResourceIds: string[],
        context: ExecutionContext,
    ): Promise<SwarmResource[]> {
        const newResources: SwarmResource[] = [];
        const now = new Date().toISOString();

        for (const resourceId of newResourceIds) {
            // In a real implementation, we would fetch resource details
            // For now, create a placeholder
            newResources.push({
                id: resourceId,
                kind: "Other",
                creator_bot_id: context.executingBotId || "system",
                created_at: now,
                meta: {
                    fromRoutine: context.routine.id,
                },
            });
        }

        return [...currentResources, ...newResources];
    }

    /**
     * Updates the status of a subtask.
     */
    private updateSubtaskStatus(
        subtasks: SwarmSubTask[],
        subtaskId: string,
        newStatus: SwarmSubTask["status"],
    ): SwarmSubTask[] {
        return subtasks.map(task => {
            if (task.id === subtaskId) {
                return { ...task, status: newStatus };
            }
            return task;
        });
    }

    /**
     * Extracts routine parameters from execution context.
     */
    private extractRoutineParams(context: ExecutionContext): Record<string, any> {
        const params: Record<string, any> = {};

        // Extract input values
        if (context.ioMapping?.inputs) {
            for (const [key, input] of Object.entries(context.ioMapping.inputs)) {
                if (input.value !== undefined) {
                    params[key] = input.value;
                }
            }
        }

        return params;
    }

    /**
     * Cancels all pending syncs.
     */
    cancelAllPendingSyncs(): void {
        for (const timeout of this.pendingSyncs.values()) {
            clearTimeout(timeout);
        }
        this.pendingSyncs.clear();
        this.logger.debug("Cancelled all pending syncs");
    }

    /**
     * Gets the number of pending sync operations.
     */
    getPendingSyncCount(): number {
        return this.pendingSyncs.size;
    }
} 

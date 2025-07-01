// AI_CHECK: STARTUP_ERRORS=1 | LAST: 2025-06-25
import { ExecutionStates } from "@vrooli/shared";
import { type ChatConfigObject, type SwarmSubTask } from "@vrooli/shared";
import { type Swarm, type SwarmConfiguration } from "@vrooli/shared";

/**
 * Service to map between execution tier swarm states and UI-friendly chat config states
 */
export class SwarmStateMapper {
    /**
     * Maps ExecutionState enum to UI-friendly status strings
     */
    static mapExecutionStateToSwarmStatus(state: keyof typeof ExecutionStates): string {
        const stateMap: Record<keyof typeof ExecutionStates, string> = {
            [ExecutionStates.UNINITIALIZED]: "initializing",
            [ExecutionStates.STARTING]: "planning",
            [ExecutionStates.RUNNING]: "active",
            [ExecutionStates.IDLE]: "idle",
            [ExecutionStates.PAUSED]: "paused",
            [ExecutionStates.STOPPED]: "stopped",
            [ExecutionStates.FAILED]: "failed",
            [ExecutionStates.TERMINATED]: "completed",
        };
        return stateMap[state] || "unknown";
    }

    /**
     * Maps UI status strings back to ExecutionState enum
     */
    static mapSwarmStatusToExecutionState(status: string): keyof typeof ExecutionStates {
        const statusMap: Record<string, keyof typeof ExecutionStates> = {
            "initializing": ExecutionStates.UNINITIALIZED,
            "planning": ExecutionStates.STARTING,
            "active": ExecutionStates.RUNNING,
            "idle": ExecutionStates.IDLE,
            "paused": ExecutionStates.PAUSED,
            "stopped": ExecutionStates.STOPPED,
            "completed": ExecutionStates.TERMINATED,
            "failed": ExecutionStates.FAILED,
            "terminated": ExecutionStates.TERMINATED,
        };
        return statusMap[status] || ExecutionStates.UNINITIALIZED;
    }

    /**
     * Converts execution tier Swarm resources to ChatConfigObject stats format
     */
    static mapSwarmResourcestoStats(swarm: Swarm): ChatConfigObject["stats"] {
        return {
            totalToolCalls: swarm.metrics?.tasksCompleted || 0,
            totalCredits: swarm.resources.consumed.toString(),
            startedAt: swarm.createdAt ? new Date(swarm.createdAt).getTime() : null,
            lastProcessingCycleEndedAt: swarm.updatedAt ? new Date(swarm.updatedAt).getTime() : null,
        };
    }

    /**
     * Updates ChatConfigObject with latest swarm state information
     */
    static updateChatConfigFromSwarm(
        chatConfig: ChatConfigObject,
        swarmConfig: SwarmConfiguration,
        swarm: Swarm,
    ): ChatConfigObject {
        return {
            ...chatConfig,
            // Update stats from swarm resources
            stats: this.mapSwarmResourcestoStats(swarm),
            // Update team information if available
            teamId: swarmConfig.teams?.[0]?.id || chatConfig.teamId,
            // Preserve existing swarm leader and subtask leaders
            swarmLeader: chatConfig.swarmLeader,
            subtaskLeaders: chatConfig.subtaskLeaders,
        };
    }

    /**
     * Creates a partial ChatConfigObject update from swarm state changes
     */
    static createConfigUpdateFromSwarmState(
        swarm: Swarm,
        additionalUpdates?: Partial<ChatConfigObject>,
    ): Partial<ChatConfigObject> {
        const update: Partial<ChatConfigObject> = {
            stats: this.mapSwarmResourcestoStats(swarm),
        };

        if (additionalUpdates) {
            return { ...update, ...additionalUpdates };
        }

        return update;
    }

    /**
     * Maps swarm task completion metrics to subtask statuses
     */
    static inferSubtaskStatuses(
        subtasks: SwarmSubTask[],
        swarm: Swarm,
    ): SwarmSubTask[] {
        // This is a simplified implementation - in practice, you'd need more
        // detailed task tracking to properly map individual subtask statuses
        const completionRatio = swarm.metrics?.tasksCompleted 
            ? swarm.metrics.tasksCompleted / Math.max(subtasks.length, 1)
            : 0;

        return subtasks.map((task, index) => {
            // Simple heuristic: mark tasks as done based on completion ratio
            const taskIndex = index + 1;
            const isComplete = taskIndex <= Math.floor(completionRatio * subtasks.length);
            
            return {
                ...task,
                status: isComplete ? "done" : task.status,
            };
        });
    }
}

import { ExecutionState, type Swarm, type SwarmConfiguration } from "@vrooli/shared";
import { SwarmSocketEmitter } from "../../swarmSocketEmitter.js";
import { SwarmStateMapper } from "../../swarmStateMapper.js";

/**
 * Example showing how to emit swarm updates from the execution tiers
 */
export class SwarmSocketExample {
    private swarmEmitter = SwarmSocketEmitter.get();

    /**
     * Example: Emit when swarm state changes
     */
    emitSwarmStateChange(chatId: string, swarm: Swarm, oldState: ExecutionState): void {
        // Emit the state change
        this.swarmEmitter.emitSwarmStateChange(chatId, swarm.id, oldState, swarm.state);

        // Also emit resource updates if they changed
        this.swarmEmitter.emitSwarmResourceUpdate(chatId, swarm.id, swarm);
    }

    /**
     * Example: Emit when swarm configuration is updated
     */
    emitSwarmConfigUpdate(chatId: string, swarmConfig: SwarmConfiguration, swarm: Swarm): void {
        // Create a config update from the swarm state
        const configUpdate = SwarmStateMapper.createConfigUpdateFromSwarmState(swarm, {
            // Add any additional updates, like new subtasks
            subtasks: swarmConfig.chatConfig.subtasks,
            swarmLeader: swarmConfig.chatConfig.swarmLeader,
            subtaskLeaders: swarmConfig.chatConfig.subtaskLeaders,
        });

        // Emit the config update
        this.swarmEmitter.emitSwarmConfigUpdate(chatId, configUpdate);
    }

    /**
     * Example: Emit full swarm update (all aspects)
     */
    emitFullSwarmUpdate(chatId: string, swarmConfig: SwarmConfiguration, swarm: Swarm): void {
        this.swarmEmitter.emitFullSwarmUpdate(chatId, swarmConfig, swarm);
    }

    /**
     * Example: Emit when a new bot joins the swarm
     */
    emitBotJoinedSwarm(chatId: string, swarmId: string, botId: string, role: "leader" | "subtask"): void {
        if (role === "leader") {
            this.swarmEmitter.emitSwarmTeamUpdate(chatId, swarmId, undefined, botId);
        } else {
            // For subtask leaders, you'd need to know which subtask
            // This is just an example
            this.swarmEmitter.emitSwarmTeamUpdate(chatId, swarmId, undefined, undefined, {
                "subtask-123": botId,
            });
        }
    }

    /**
     * Example: Integration point in TierOneCoordinator
     */
    exampleTierOneIntegration(chatId: string, swarmConfig: SwarmConfiguration): void {
        // When swarm execution starts
        const swarm: Swarm = {
            id: swarmConfig.id,
            name: "Swarm for " + chatId,
            state: ExecutionState.STARTING,
            resources: {
                allocated: 1000,
                consumed: 0,
                remaining: 1000,
                reservedByChildren: 0,
            },
            metrics: {
                tasksCompleted: 0,
                tasksFailed: 0,
                avgTaskDuration: 0,
            },
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        };

        // Emit initial state
        this.emitFullSwarmUpdate(chatId, swarmConfig, swarm);

        // Later, when state changes to RUNNING
        const oldState = swarm.state;
        swarm.state = ExecutionState.RUNNING;
        swarm.updatedAt = new Date().toISOString();
        this.emitSwarmStateChange(chatId, swarm, oldState);

        // As resources are consumed
        swarm.resources.consumed = 150;
        swarm.resources.remaining = 850;
        this.swarmEmitter.emitSwarmResourceUpdate(chatId, swarm.id, swarm);

        // When tasks complete
        swarm.metrics.tasksCompleted = 1;
        const configUpdate = SwarmStateMapper.createConfigUpdateFromSwarmState(swarm);
        this.swarmEmitter.emitSwarmConfigUpdate(chatId, configUpdate);
    }
}
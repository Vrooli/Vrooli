import { type Job } from "bullmq";
import { nanoid } from "@vrooli/shared";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
// Note: completionService import removed - swarm execution now uses tier1 architecture
import { type SwarmStateMachine } from "../../services/execution/tier1/coordination/swarmStateMachine.js";
import { SwarmExecutionService } from "../../services/execution/swarmExecutionService.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord, type ManagedTaskStateMachine } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask, type SwarmExecutionTask } from "../taskTypes.js";

/**
 * Adapter to make the new SwarmExecutionService work with the existing ActiveSwarmRegistry
 */
export class NewSwarmStateMachineAdapter implements ManagedTaskStateMachine {
    constructor(
        private readonly swarmId: string,
        private readonly swarmExecutionService: SwarmExecutionService,
        private readonly userId?: string,
    ) {}

    getTaskId(): string {
        return this.swarmId;
    }

    getCurrentSagaStatus(): string {
        // This is synchronous but we need to fetch status async
        // Return a default status for now - the registry will call this periodically
        return "RUNNING";
    }

    async requestPause(): Promise<boolean> {
        // For now, swarm execution doesn't support pause
        // This could be implemented in the future
        return false;
    }

    async requestStop(reason: string): Promise<boolean> {
        try {
            const result = await this.swarmExecutionService.cancelSwarm(
                this.swarmId,
                this.userId || "system",
                reason,
            );
            return result.success;
        } catch (error) {
            logger.error("[NewSwarmStateMachineAdapter] Failed to stop swarm", {
                swarmId: this.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    getAssociatedUserId?(): string | undefined {
        return this.userId;
    }
}

export type ActiveSwarmRecord = BaseActiveTaskRecord;
export class ActiveSwarmRegistry extends BaseActiveTaskRegistry<ActiveSwarmRecord, SwarmStateMachine | NewSwarmStateMachineAdapter> {
    // Add swarm-specific registry setup here
}
export const activeSwarmRegistry = new ActiveSwarmRegistry();

// Initialize the new execution service
const swarmExecutionService = new SwarmExecutionService(logger);

export async function llmProcessBotMessage(payload: LLMCompletionTask) {
    // TODO: Route through swarm execution service instead of conversation/responseEngine
    throw new Error("LLM completion through conversation service deprecated - use swarm execution");
}

/**
 * Process new three-tier swarm execution
 */
async function processNewSwarmExecution(payload: SwarmExecutionTask) {
    logger.info("[processNewSwarmExecution] Starting new swarm execution", {
        swarmId: payload.swarmId,
        name: payload.config.name,
        userId: payload.userData.id,
    });

    try {
        // Extract swarm ID or generate new one if not provided
        const swarmId = payload.swarmId || nanoid();

        // Start swarm through new three-tier architecture
        const result = await swarmExecutionService.startSwarm({
            swarmId,
            name: payload.config.name,
            description: payload.config.description,
            goal: payload.config.goal,
            resources: payload.config.resources,
            config: payload.config.config,
            userId: payload.userData.id,
            organizationId: payload.config.organizationId,
        });

        // Create adapter for registry management
        const adapter = new NewSwarmStateMachineAdapter(
            swarmId,
            swarmExecutionService,
            payload.userData.id,
        );

        // Add to active swarm registry
        const record: ActiveSwarmRecord = {
            id: swarmId,
            userId: payload.userData.id,
            hasPremium: payload.userData.hasPremium || false,
            startTime: Date.now(),
        };

        activeSwarmRegistry.add(record, adapter);

        logger.info("[processNewSwarmExecution] Successfully started swarm", {
            swarmId: result.swarmId,
        });

        return result;

    } catch (error) {
        logger.error("[processNewSwarmExecution] Failed to start swarm execution", {
            swarmId: payload.swarmId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

export async function llmProcess({ data }: Job<LLMCompletionTask | SwarmExecutionTask>) {
    switch (data.type) {
        case QueueTaskType.LLM_COMPLETION:
            return llmProcessBotMessage(data as LLMCompletionTask);
        
        case QueueTaskType.SWARM_EXECUTION:
            return processNewSwarmExecution(data as SwarmExecutionTask);
            
        default:
            throw new CustomError("0330", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}

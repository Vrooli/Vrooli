import { generatePK } from "@vrooli/shared";
import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { conversationEngine, responseService, swarmContextManager } from "../../services/conversation/responseEngine.js";
import { SwarmStateMachine } from "../../services/execution/tier1/swarmStateMachine.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask, type SwarmExecutionTask } from "../taskTypes.js";


export type ActiveSwarmRecord = BaseActiveTaskRecord;
class SwarmRegistry extends BaseActiveTaskRegistry<ActiveSwarmRecord, SwarmStateMachine> {
    // Add any swarm-specific methods here
}
export const activeSwarmRegistry = new SwarmRegistry();


export async function llmProcessBotMessage(payload: LLMCompletionTask) {
    // TODO: Route through swarm execution service instead of conversation/responseEngine
    throw new Error("LLM completion through conversation service deprecated - use swarm execution");
}

/**
 * Process swarm execution directly through SwarmStateMachine
 * 
 * This uses SwarmStateMachine directly with SwarmExecutionTask structure
 * for clean integration with the three-tier architecture.
 */
async function processSwarmExecution(payload: SwarmExecutionTask) {
    logger.info("[processSwarmExecution] Starting swarm execution", {
        swarmId: payload.context.swarmId,
        userId: payload.context.userData.id,
    });

    try {
        // Extract swarm ID or generate new one if not provided
        const swarmId = payload.context.swarmId || generatePK().toString();

        // Use the existing service instances from responseEngine
        const coordinator = new SwarmStateMachine(
            swarmContextManager,
            conversationEngine,
            responseService,
        );

        // Start swarm with SwarmExecutionTask directly
        const result = await coordinator.start(payload);
        if (!result.success) {
            throw new Error(result.error?.message || "Failed to start swarm");
        }

        // Create registry record
        const record: ActiveSwarmRecord = {
            id: swarmId,
            userId: payload.context.userData.id,
            hasPremium: payload.context.userData.hasPremium || false,
            startTime: Date.now(),
        };

        // Add coordinator to registry
        activeSwarmRegistry.add(record, coordinator);

        logger.info("[processSwarmExecution] Successfully started swarm", {
            swarmId,
        });

        return { swarmId };

    } catch (error) {
        logger.error("[processSwarmExecution] Failed to start swarm execution", {
            swarmId: payload.context.swarmId,
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
            return processSwarmExecution(data as SwarmExecutionTask);

        default:
            throw new CustomError("0330", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}

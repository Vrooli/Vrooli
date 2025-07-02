import { generatePK, type SwarmCoordinationInput } from "@vrooli/shared";
import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { SwarmCoordinator } from "../../services/execution/tier1/swarmCoordinator.js";
import type { SwarmStateMachine } from "../../services/execution/tier1/swarmStateMachine.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask, type SwarmExecutionTask } from "../taskTypes.js";
import { conversationEngine, responseService, swarmContextManager } from "../../services/conversation/responseEngine.js";
import { getEventBus } from "../../services/events/eventBus.js";


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
 * Process swarm execution directly through SwarmCoordinator
 * 
 * This eliminates the SwarmExecutionService + NewSwarmStateMachineAdapter overhead
 * by using SwarmCoordinator directly, which already implements ManagedTaskStateMachine.
 */
async function processSwarmExecution(payload: SwarmExecutionTask) {
    logger.info("[processSwarmExecution] Starting swarm execution", {
        swarmId: payload.context.swarmId,
        userId: payload.context.userData.id,
    });

    try {
        // Extract swarm ID or generate new one if not provided
        const swarmId = payload.context.swarmId || generatePK().toString();

        const request: SwarmCoordinationInput = payload;
        
        // Use the existing service instances from responseEngine
        const coordinator = new SwarmCoordinator(
            swarmContextManager,
            conversationEngine,
            responseService,
            getEventBus(),
        );
        const result = await coordinator.execute(request);
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

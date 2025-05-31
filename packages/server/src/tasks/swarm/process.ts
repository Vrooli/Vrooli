import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { completionService, type SwarmStateMachine } from "../../services/conversation/responseEngine.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type LLMCompletionTask } from "../taskTypes.js";

export type ActiveSwarmRecord = BaseActiveTaskRecord;
export class ActiveSwarmRegistry extends BaseActiveTaskRegistry<ActiveSwarmRecord, SwarmStateMachine> {
    // Add swarm-specific registry setup here
}
export const activeSwarmRegistry = new ActiveSwarmRegistry();

export async function llmProcessBotMessage(payload: LLMCompletionTask) {
    return completionService.respond(payload);
}

export async function llmProcess({ data }: Job<LLMCompletionTask>) {
    switch (data.type) {
        case QueueTaskType.LLM_COMPLETION:
            return llmProcessBotMessage(data);
        default:
            throw new CustomError("0330", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}

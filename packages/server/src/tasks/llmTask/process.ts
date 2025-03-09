import { AITaskInfo, LlmTask, ServerLlmTaskInfo } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger.js";
import { emitSocketEvent } from "../../sockets/events.js";
import { generateTaskExec } from "../llm/converter.js";
import { LlmTaskProcessPayload, changeLlmTaskStatus } from "./queue.js";

type ExecuteLlmTaskParams = {
    data: Omit<LlmTaskProcessPayload, "status">;
}
export type ExecuteLlmTaskResult = ServerLlmTaskInfo & Pick<AITaskInfo, "resultLabel" | "resultLink"> & {
    status: "Completed" | "Failed"
};

export async function executeLlmTask({
    data,
}: ExecuteLlmTaskParams): Promise<ExecuteLlmTaskResult> {
    const { chatId, language, taskInfo, userData } = data;
    let success = false;
    let resultLabel: string | undefined;
    let resultLink: string | undefined;
    try {
        // Notify UI that command is being processed
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, { updates: [{ taskId: taskInfo.taskId, status: "Running" }] });
        }

        // Disallow "Start" task
        const task = taskInfo.task;
        if (task === "Start") {
            throw new Error("Cannot execute Start task");
        }
        // Convert command to task and execute
        const taskExec = await generateTaskExec(task as Exclude<LlmTask, "Start">, language, userData);
        const taskResult = await taskExec(taskInfo.properties ?? {});
        resultLabel = taskResult.label ?? undefined;
        resultLink = taskResult.link ?? undefined;
        success = true;
    } catch (error) {
        logger.error("Caught error in executeLlmTask", { trace: "0498", error, taskInfo, language, userId: (userData as unknown as LlmTaskProcessPayload["userData"])?.id });
        await changeLlmTaskStatus(taskInfo.taskId, "Failed", userData.id);
    }
    const result = {
        ...(taskInfo as LlmTaskProcessPayload["taskInfo"]),
        lastUpdated: new Date().toISOString(),
        resultLabel,
        resultLink,
        status: success ? "Completed" as const : "Failed" as const,
    };
    if (chatId && taskInfo !== null) {
        emitSocketEvent("llmTasks", chatId, { tasks: [result] });
    }
    await changeLlmTaskStatus(taskInfo.taskId, result.status, userData.id);
    return result;
}

export async function llmTaskProcess({ data }: Job<LlmTaskProcessPayload>) {
    await changeLlmTaskStatus(data.taskInfo.taskId, "Running", data.userData.id);
    await executeLlmTask({ data });
}

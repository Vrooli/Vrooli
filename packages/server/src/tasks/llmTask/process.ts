import { LlmTaskInfo, ServerLlmTaskInfo } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { generateTaskExec } from "../llm/converter";
import { LlmTaskProcessPayload, changeLlmTaskStatus } from "./queue";

type ExecuteLlmTaskParams = {
    data: Omit<LlmTaskProcessPayload, "status">;
}
export type ExecuteLlmTaskResult = ServerLlmTaskInfo & Pick<LlmTaskInfo, "resultLabel" | "resultLink"> & {
    status: "Completed" | "Failed"
};

export const executeLlmTask = async ({
    data,
}: ExecuteLlmTaskParams): Promise<ExecuteLlmTaskResult> => {
    const { chatId, language, taskInfo, userData } = data;
    let success = false;
    let resultLabel: string | undefined;
    let resultLink: string | undefined;
    try {
        // Notify UI that command is being processed
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, { updates: [{ id: taskInfo.id, status: "Running" }] });
        }

        // Convert command to task and execute
        const taskExec = await generateTaskExec(taskInfo.task, language, userData);
        const taskResult = await taskExec(taskInfo.properties ?? {});
        resultLabel = taskResult.label ?? undefined;
        resultLink = taskResult.link ?? undefined;
        success = true;
    } catch (error) {
        logger.error("Caught error in executeLlmTask", { trace: "0498", error, taskInfo, language, userId: (userData as unknown as LlmTaskProcessPayload["userData"])?.id });
        await changeLlmTaskStatus(taskInfo.id, "Failed", userData.id);
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
    await changeLlmTaskStatus(taskInfo.id, result.status, userData.id);
    return result;
};

export const llmTaskProcess = async ({ data }: Job<LlmTaskProcessPayload>) => {
    await changeLlmTaskStatus(data.taskInfo.id, "Running", data.userData.id);
    await executeLlmTask({ data });
};

import { ServerLlmTaskInfo } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { generateTaskExec } from "../llm/converter";
import { LlmTaskProcessPayload } from "./queue";

type ExecuteLlmTaskParams = {
    data: LlmTaskProcessPayload;
    update: (data: LlmTaskProcessPayload) => Promise<unknown>;
}
export type ExecuteLlmTaskResult = ServerLlmTaskInfo & { status: "completed" | "failed" };

export const executeLlmTask = async ({
    data,
    update,
}: ExecuteLlmTaskParams): Promise<ExecuteLlmTaskResult> => {
    const { chatId, language, taskInfo, userData } = data;
    let success = false;
    try {
        // Notify UI that command is being processed
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, {
                tasks: [{
                    ...taskInfo,
                    lastUpdated: new Date().toISOString(),
                    status: "running",
                }],
            });
        }

        // Convert command to task and execute
        const taskExec = await generateTaskExec(taskInfo.task, language, userData);
        taskExec(taskInfo.properties ?? {});
        success = true;
    } catch (error) {
        logger.error("Caught error in executeLlmTask", { trace: "0498", error, taskInfo, language, userId: (userData as unknown as LlmTaskProcessPayload["userData"])?.id });
        await update({ ...data, status: "failed" });

    }
    const result = {
        ...(taskInfo as LlmTaskProcessPayload["taskInfo"]),
        lastUpdated: new Date().toISOString(),
        status: success ? "completed" as const : "failed" as const,
    };
    if (chatId && taskInfo !== null) {
        emitSocketEvent("llmTasks", chatId, {
            tasks: [result],
        });
    }
    await update({ ...data, status: "completed" });
    return result;
};

export const llmTaskProcess = async ({ data, update }: Job<LlmTaskProcessPayload>) => {
    await update({ ...data, status: "running" });
    await executeLlmTask({ data, update });
};

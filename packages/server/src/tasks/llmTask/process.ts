import { LlmTaskInfo } from "@local/shared";
import { Job } from "bull";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { generateTaskExec } from "../llm/config";
import { LlmTaskProcessPayload } from "./queue";

export type ExecuteLlmTaskResult = Omit<LlmTaskInfo, "status"> & { status: "completed" | "failed" };

export const executeLlmTask = async ({
    chatId,
    language,
    taskInfo,
    userData,
}: LlmTaskProcessPayload): Promise<ExecuteLlmTaskResult> => {
    let success = false;
    try {
        // Notify UI that command is being processed
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, {
                tasks: [{
                    ...taskInfo,
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
    }
    const result = {
        ...(taskInfo as LlmTaskProcessPayload["taskInfo"]),
        status: success ? "completed" as const : "failed" as const,
    };
    if (chatId && taskInfo !== null) {
        emitSocketEvent("llmTasks", chatId, {
            tasks: [result],
        });
    }
    return result;
};

export const llmTaskProcess = async ({ data }: Job<LlmTaskProcessPayload>) => {
    await executeLlmTask(data);
};

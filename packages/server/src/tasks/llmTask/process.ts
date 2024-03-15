import { Job } from "bull";
import { emitSocketEvent } from "../../sockets/events";
import { withPrisma } from "../../utils/withPrisma";
import { generateTaskExec } from "../llm/config";
import { LlmTaskProcessPayload } from "./queue";

export const llmTaskProcess = async (job: Job<LlmTaskProcessPayload>) => {
    let taskInfo: LlmTaskProcessPayload["taskInfo"] | null = null;
    let chatId: string | null = null;
    let language: LlmTaskProcessPayload["language"] | null = null;
    let userData: LlmTaskProcessPayload["userData"] | null = null;

    const success = await withPrisma({
        process: async (prisma) => {
            // Parse data from job
            taskInfo = job.data.taskInfo;
            chatId = job.data.chatId;
            language = job.data.language;
            userData = job.data.userData;

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
            const taskExec = await generateTaskExec(taskInfo.task, language, prisma, userData);
            taskExec(taskInfo.properties ?? {});
        },
        trace: "0498",
        traceObject: { taskInfo, language, userId: (userData as unknown as LlmTaskProcessPayload["userData"])?.id },
    });
    if (chatId && taskInfo !== null) {
        emitSocketEvent("llmTasks", chatId, {
            tasks: [{
                ...(taskInfo as LlmTaskProcessPayload["taskInfo"]),
                status: success ? "completed" : "failed",
            }],
        });
    }
};

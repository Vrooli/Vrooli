import { Job } from "bull";
import { emitSocketEvent } from "../../sockets/events";
import { withPrisma } from "../../utils/withPrisma";
import { generateTaskExec } from "../llm/config";
import { CommandProcessPayload } from "./queue";

export const commandProcess = async (job: Job<CommandProcessPayload>) => {
    let command: CommandProcessPayload["command"] | null = null;
    let chatId: string | null = null;
    let language: CommandProcessPayload["language"] | null = null;
    let userData: CommandProcessPayload["userData"] | null = null;

    const success = await withPrisma({
        process: async (prisma) => {
            // Parse data from job
            command = job.data.command;
            chatId = job.data.chatId;
            language = job.data.language;
            userData = job.data.userData;

            // Notify UI that command is being processed
            if (chatId) {
                emitSocketEvent("llmTasks", chatId, [{
                    command,
                    status: "Pending",
                }]);
            }

            // Convert command to task and execute
            const taskExec = await generateTaskExec(command.task, language, prisma, userData);
            taskExec(command.properties ?? {});
        },
        trace: "0498",
        traceObject: { command, language, userId: (userData as unknown as CommandProcessPayload["userData"])?.id },
    });
    if (chatId) {
        emitSocketEvent("llmTasks", chatId, [{
            command,
            status: success ? "Success" : "Failed",
        }]);
    }
};

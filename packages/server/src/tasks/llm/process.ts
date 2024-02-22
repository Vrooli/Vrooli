import { ChatMessage, toBotSettings } from "@local/shared";
import { Job } from "bull";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { chatMessage_findOne } from "../../endpoints/generated/chatMessage_findOne";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { emitSocketEvent } from "../../sockets/events";
import { withPrisma } from "../../utils/withPrisma";
import { extractCommands, filterInvalidCommands, removeCommands } from "./commands";
import { generateTaskExec, getUnstructuredTaskConfig, importCommandToTask } from "./config";
import { RequestBotResponsePayload } from "./queue";
import { getLanguageModelService } from "./service";

export async function llmProcess({ data }: Job<RequestBotResponsePayload>) {
    await withPrisma({
        process: async (prisma) => {
            // Extract data from payload
            const { chatId, messageId, message, respondingBotId, participantsData, userData } = data;
            const language = userData.languages[0] ?? "en";
            // Parse bot information
            const bot = participantsData[respondingBotId];
            if (!bot) throw new Error("Bot data not found in participants data");
            const botSettings = toBotSettings(bot, logger);
            const service = getLanguageModelService(botSettings);
            // Check for commands
            const commandToTask = await importCommandToTask(language);
            const maybeCommands = extractCommands(message.content, commandToTask);
            const commands = await filterInvalidCommands(maybeCommands, await getUnstructuredTaskConfig("Start", botSettings)); //TODO need task
            const messageWithoutCommands = removeCommands(message.content, commands);
            console.log('got commands', commands);
            // TODO should probably have these in redis queue, with status updates sent via websocket
            const taskExecProcesses = commands.map(({ task, properties }) =>
                generateTaskExec(task, language, prisma, userData)
                    .then(execFunc => execFunc(properties ?? {}, botSettings)));
            // Execute all task exec functions asynchronously
            try {
                const taskResults = await Promise.all(taskExecProcesses);
            } catch (error) {
                logger.error("Error executing task exec functions", { trace: "0083", error });
            }
            // If there is text besides commands, also generate a response
            if (messageWithoutCommands.trim() === "") return;
            // Send typing message while bot is responding
            emitSocketEvent("typing", message.chatId as string, { starting: [respondingBotId] });
            const text = await service.generateResponse(chatId, messageId, message.content, respondingBotId, botSettings, userData);
            const select = selectHelper(chatMessage_findOne);
            const createdData = await prisma.chat_message.create({
                data: {
                    chat: { connect: { id: message.chatId as string } },
                    parent: { connect: { id: messageId } },
                    user: { connect: { id: respondingBotId } },
                    translations: {
                        create: {
                            language: message.language,
                            text,
                        },
                    },
                },
                ...select,
            });
            const formatted = modelToGql(createdData, chatMessage_findOne);
            const fullMessageData = (await addSupplementalFields(prisma, userData, [formatted], chatMessage_findOne))[0] as ChatMessage;
            await Trigger(prisma, [message.language]).objectCreated({
                createdById: respondingBotId,
                hasCompleteAndPublic: true, // N/A
                hasParent: true, // N/A
                owner: { id: respondingBotId, __typename: "User" },
                objectId: fullMessageData.id,
                objectType: "ChatMessage",
            });
            await Trigger(prisma, [message.language]).chatMessageCreated({
                createdById: respondingBotId,
                data: message,
                message: fullMessageData,
            });
            // Reduce user's credits by 1
            await prisma.user.update({
                where: { id: userData.id },
                data: { premium: { update: { credits: { decrement: 1 } } } },
            });
        },
        trace: "0081",
    });
    if (data.message?.chatId && data?.respondingBotId) {
        emitSocketEvent("typing", data.message.chatId, { stopping: [data.respondingBotId] });
    }
}

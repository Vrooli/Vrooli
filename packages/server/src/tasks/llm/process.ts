import { BotSettings, ChatMessage, toBotSettings } from "@local/shared";
import { Job } from "bull";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { chatMessage_findOne } from "../../endpoints/generated/chatMessage_findOne";
import { CustomError } from "../../events";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { emitSocketEvent } from "../../sockets/events";
import { processCommand } from "../../tasks/command";
import { withPrisma } from "../../utils/withPrisma";
import { LlmCommand, extractCommands, filterInvalidCommands, removeCommands } from "./commands";
import { CommandToTask, LlmTask, getUnstructuredTaskConfig, importCommandToTask } from "./config";
import { RequestBotResponsePayload } from "./queue";
import { LanguageModelService, getLanguageModelService } from "./service";

const parseBotInformation = (
    participants: Record<string, PreMapUserData>,
    respondingBotId: string,
    logger: { error: (message: string, data?: Record<string, any>) => unknown },
    language: string,
): { botSettings: BotSettings, service: LanguageModelService<any, any> } => {
    const bot = participants[respondingBotId];
    if (!bot) {
        throw new CustomError("0176", "InternalError", [language]);
    }
    const botSettings = toBotSettings(bot, logger);
    const service = getLanguageModelService(botSettings);
    return { botSettings, service };
};

const getCommands = async (
    message: string,
    task: LlmTask,
    botSettings: BotSettings,
    commandToTask: CommandToTask,
): Promise<{ commands: LlmCommand[], messageWithoutCommands: string }> => {
    const maybeCommands = extractCommands(message, commandToTask);
    const commands = await filterInvalidCommands(maybeCommands, await getUnstructuredTaskConfig(task, botSettings));
    const messageWithoutCommands = removeCommands(message, commands);
    return { commands, messageWithoutCommands };
};

export async function llmProcess({ data }: Job<RequestBotResponsePayload>) {
    await withPrisma({
        process: async (prisma) => {
            // Extract data from payload
            const { chatId, messageId, message, respondingBotId, task, participantsData, userData } = data;
            const language = userData.languages[0] ?? "en";
            // Parse bot information
            const { botSettings, service } = parseBotInformation(participantsData, respondingBotId, logger, language);
            // Check for commands in user's message
            const commandToTask = await importCommandToTask(language);
            const { commands, messageWithoutCommands } = await getCommands(message.content, task, botSettings, commandToTask);
            for (const command of commands) {
                processCommand({
                    command,
                    chatId,
                    language,
                    userData,
                });
            }
            // If there is no text left after removing commands, don't create a response
            if (messageWithoutCommands.trim() === "") return;
            // Send typing message while bot is responding
            emitSocketEvent("typing", message.chatId as string, { starting: [respondingBotId] });
            const botResponse = await service.generateResponse(chatId, messageId, messageWithoutCommands, respondingBotId, botSettings, task, userData);
            // Check for commands in bot's response
            const { commands: botCommands, messageWithoutCommands: botResponseWithoutCommands } = await getCommands(botResponse, task, botSettings, commandToTask);
            for (const command of botCommands) {
                // If we're in a start command, generate a new response with the config to get the command properties
                if (task === "Start") {
                    //TODO
                } else {
                    processCommand({
                        command,
                        chatId,
                        language,
                        userData,
                    });
                }
            }
            // If there is no text left after removing commands, don't send a response
            if (botResponseWithoutCommands.trim() === "") return;
            // TODO should instead sent some sort of confirmation message
            const select = selectHelper(chatMessage_findOne);
            const createdData = await prisma.chat_message.create({
                data: {
                    chat: { connect: { id: message.chatId as string } },
                    parent: { connect: { id: messageId } },
                    user: { connect: { id: respondingBotId } },
                    translations: {
                        create: {
                            language: message.language,
                            text: botResponseWithoutCommands,
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

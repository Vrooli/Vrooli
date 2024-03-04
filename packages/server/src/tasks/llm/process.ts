import { SessionUserToken } from "@local/server";
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
import { CommandToTask, LlmTask, importCommandToTask } from "./config";
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
    language: string,
    commandToTask: CommandToTask,
): Promise<{ commands: LlmCommand[], messageWithoutCommands: string }> => {
    const maybeCommands = extractCommands(message, commandToTask);
    const commands = await filterInvalidCommands(maybeCommands, task, language);
    const messageWithoutCommands = removeCommands(message, commands);
    return { commands, messageWithoutCommands };
};

const forceGetCommand = async (
    command: LlmCommand,
    chatId: string,
    respondingToMessage: {
        id: string,
        text: string,
    } | null,
    respondingBotId: string,
    respondingBotConfig: BotSettings,
    userData: SessionUserToken,
    language: string,
    service: LanguageModelService<any, any>,
    commandToTask: CommandToTask,
): Promise<{ command: LlmCommand, messageWithoutCommands: string } | null> => {
    let retryCount = 0;
    const MAX_RETRIES = 3; // Set a maximum number of retries to avoid infinite loops
    const commandFound = false;

    while (!commandFound && retryCount < MAX_RETRIES) {
        const startResponse = await service.generateResponse({
            chatId,
            respondingToMessage,
            respondingBotId,
            respondingBotConfig,
            task: command.task, // Change to the command's task
            force: true, // Force the bot to respond with a command
            userData,
        });

        // Check for commands in the start response
        const { commands: startCommands, messageWithoutCommands } = await getCommands(startResponse, command.task, language, commandToTask);

        if (startCommands.length > 0) {
            // Only use the first command found
            return { command: startCommands[0], messageWithoutCommands };
        } else {
            // Increment the retry count if no command is found
            retryCount++;
            logger.warning(`No command found in start response. Retrying... (${retryCount}/${MAX_RETRIES})`, { trace: "0349", chatId, respondingBotId, task: command.task });
        }
    }

    logger.error("Failed to find a command in start response after maximum retries.", { trace: "0350", chatId, respondingBotId, task: command.task });
    return null;
};

export async function llmProcess({ data }: Job<RequestBotResponsePayload>) {
    let chatId: string | undefined = undefined;
    let respondingBotId: string | undefined = undefined;
    await withPrisma({
        process: async (prisma) => {
            // Extract data from payload
            const { parent, task, participantsData, userData, ...rest } = data;
            chatId = rest.chatId;
            respondingBotId = rest.respondingBotId;
            const language = userData.languages[0] ?? "en";
            // Parse bot information
            const { botSettings, service } = parseBotInformation(participantsData, respondingBotId, logger, language);
            const commandToTask = await importCommandToTask(language);
            // Parse previous message
            let respondingToMessage: { id: string, text: string } | null = null;
            if (parent) {
                // Check for commands in user's message
                const { commands, messageWithoutCommands } = await getCommands(parent.content, task, language, commandToTask);
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
                respondingToMessage = {
                    id: parent.id,
                    text: messageWithoutCommands,
                };
            }
            // Start typing indicator
            emitSocketEvent("typing", chatId, { starting: [respondingBotId] });
            // Generate bot response
            const botResponse = await service.generateResponse({
                chatId,
                respondingToMessage,
                respondingBotId,
                respondingBotConfig: botSettings,
                task,
                force: false,
                userData,
            });
            // Check for commands in bot's response
            const { commands: botCommands, messageWithoutCommands: botResponseWithoutCommands } = await getCommands(botResponse, task, language, commandToTask);
            for (const command of botCommands) {
                // If we're in a start command, generate a new response with the config to get the command properties
                if (task === "Start") {
                    const forcedCommand = await forceGetCommand(
                        command,
                        chatId,
                        respondingToMessage,
                        respondingBotId,
                        botSettings,
                        userData,
                        language,
                        service,
                        commandToTask,
                    );
                    if (forcedCommand) {
                        processCommand({
                            command: forcedCommand.command,
                            chatId,
                            language,
                            userData,
                        });
                    }
                    // TODO forcedCommand.message should be used
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
            // TODO should instead sent some sort of confirmation message
            if (botResponseWithoutCommands.trim() === "") return;
            const select = selectHelper(chatMessage_findOne);
            const createdData = await prisma.chat_message.create({
                data: {
                    chat: { connect: { id: chatId } },
                    parent: parent ? { connect: { id: parent.id } } : undefined,
                    user: { connect: { id: respondingBotId } },
                    //TODO need to check existing number of parent's children to set versionIndex. Find somewhere to do this without having to call prisma again
                    translations: {
                        create: {
                            language,
                            text: botResponseWithoutCommands,
                        },
                    },
                },
                ...select,
            });
            const formattedResponseMessage = modelToGql(createdData, chatMessage_findOne);
            const fullResponseMessage = (await addSupplementalFields(prisma, userData, [formattedResponseMessage], chatMessage_findOne))[0] as ChatMessage;
            await Trigger(prisma, [language]).objectCreated({
                createdById: userData.id,
                hasCompleteAndPublic: true, // N/A
                hasParent: true, // N/A
                owner: { id: respondingBotId, __typename: "User" },
                objectId: fullResponseMessage.id,
                objectType: "ChatMessage",
            });
            await Trigger(prisma, [language]).chatMessageCreated({
                excludeUserId: userData.id,
                chatId,
                messageId: fullResponseMessage.id,
                senderId: respondingBotId,
                message: fullResponseMessage,
            });
            // Reduce user's credits by 1
            const updatedUser = await prisma.user.update({
                where: { id: userData.id },
                data: { premium: { update: { credits: { decrement: 1 } } } },
                select: { premium: { select: { credits: true } } },
            });
            if (updatedUser.premium) {
                emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits });
            }
        },
        trace: "0081",
    });
    // Remove typing indicator
    if (chatId && respondingBotId) {
        emitSocketEvent("typing", chatId, { stopping: [respondingBotId] });
    }
}

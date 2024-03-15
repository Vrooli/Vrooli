import { BotSettings, ChatMessage, LlmTaskInfo, toBotSettings } from "@local/shared";
import { Job } from "bull";
import i18next from "i18next";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { chatMessage_findOne } from "../../endpoints/generated/chatMessage_findOne";
import { CustomError } from "../../events";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { emitSocketEvent } from "../../sockets/events";
import { withPrisma } from "../../utils/withPrisma";
import { processLlmTask } from "../llmTask";
import { importCommandToTask } from "./config";
import { LlmRequestPayload, RequestAutoFillPayload, RequestBotMessagePayload } from "./queue";
import { LanguageModelService, getLanguageModelService } from "./service";
import { ForceGetTaskParams, forceGetTask, getValidTasksFromMessage } from "./tasks";

const parseBotInformation = (
    participants: Record<string, PreMapUserData>,
    respondingBotId: string,
    logger: { error: (message: string, data?: Record<string, any>) => unknown },
    language: string,
): { botSettings: BotSettings, service: LanguageModelService<string, string> } => {
    const bot = participants[respondingBotId];
    if (!bot) {
        throw new CustomError("0176", "InternalError", [language]);
    }
    const botSettings = toBotSettings(bot, logger);
    const service = getLanguageModelService(botSettings);
    return { botSettings, service };
};

type ForceGetAndProcessCommandParams = ForceGetTaskParams;
type ForceGetAndProcessCommandResult = {
    messageWithoutTasks: string | null,
    tasksToSuggest: Omit<LlmTaskInfo, "status">[],
    cost: number
};
const forceGetAndProcessCommand = async (params: ForceGetAndProcessCommandParams): Promise<ForceGetAndProcessCommandResult> => {
    const { taskThatWasRun, tasksToSuggest, messageWithoutTasks, cost } = await forceGetTask(params);
    if (!taskThatWasRun || !messageWithoutTasks) {
        return { messageWithoutTasks: null, tasksToSuggest: [], cost };
    }
    processLlmTask({ taskInfo: taskThatWasRun, ...params });
    return { messageWithoutTasks, tasksToSuggest, cost };
};

export const llmProcessBotMessage = async (data: RequestBotMessagePayload) => {
    // Extract data from payload
    const { parent, task, participantsData, userData, ...rest } = data;
    const chatId = rest.chatId;
    const respondingBotId = rest.respondingBotId;
    const language = userData.languages[0] ?? "en";

    let wasResponseSent = false;
    let totalCost = 0;

    await withPrisma({
        process: async (prisma) => {
            // Parse bot information
            const { botSettings, service } = parseBotInformation(participantsData, respondingBotId, logger, language);
            const commandToTask = await importCommandToTask(language);

            // Parse previous message
            let respondingToMessage: { id: string, text: string } | null = null;
            if (parent) {
                // Check for commands in user's message
                const { tasksToRun, messageWithoutTasks } = await getValidTasksFromMessage(parent.content, task, language, commandToTask);
                for (const taskInfo of tasksToRun) {
                    processLlmTask({
                        taskInfo,
                        chatId,
                        language,
                        userData,
                    });
                }
                // If there is no text left after removing commands, don't create a response
                if (messageWithoutTasks.trim() === "") return;
                respondingToMessage = {
                    id: parent.id,
                    text: messageWithoutTasks,
                };
            }

            // Start typing indicator
            emitSocketEvent("typing", chatId, { starting: [respondingBotId] });

            let responseMessage: string | null = null;
            let botCommandsToRun: Omit<LlmTaskInfo, "status">[] = [];
            let botTasksToSuggest: Omit<LlmTaskInfo, "status">[] = [];

            // If we're not in a "Start" task (the only task that allows general conversation), 
            // we'll demand a valid command immediately
            if (task !== "Start") {
                const { messageWithoutTasks, tasksToSuggest, cost } = await forceGetAndProcessCommand({
                    chatId,
                    commandToTask,
                    language,
                    participantsData,
                    respondingBotConfig: botSettings,
                    respondingBotId,
                    respondingToMessage,
                    service,
                    task,
                    userData,
                });
                responseMessage = messageWithoutTasks;
                botTasksToSuggest = tasksToSuggest;
                totalCost += cost;
            }
            // Otherwise, we'll generate a normal response and handle any commands that it contains
            else {
                // Generate bot response
                const { message: botResponse, cost } = await service.generateResponse({
                    chatId,
                    force: false,
                    participantsData,
                    respondingBotConfig: botSettings,
                    respondingBotId,
                    respondingToMessage,
                    task,
                    userData,
                });
                totalCost += cost;

                // Check for commands in bot's response
                const { tasksToRun, tasksToSuggest, messageWithoutTasks: botResponseWithoutCommands } = await getValidTasksFromMessage(botResponse, task, language, commandToTask);
                botCommandsToRun = tasksToRun;
                botTasksToSuggest = tasksToSuggest;
                if (botResponseWithoutCommands.trim() !== "") {
                    responseMessage = botResponseWithoutCommands;
                }

                // Handle any commands in the bot's response
                for (const command of botCommandsToRun) {
                    // If we're in the "Start" task, the commands won't have the full information we need to process them. 
                    // So we'll have to make a separate call to get the full command information
                    if (task === "Start") {
                        const { messageWithoutTasks, cost } = await forceGetAndProcessCommand({
                            chatId,
                            commandToTask,
                            language,
                            participantsData,
                            respondingBotConfig: botSettings,
                            respondingBotId,
                            respondingToMessage,
                            service,
                            task: command.task,
                            userData,
                        });
                        totalCost += cost;
                        // If we don't have a response message yet and this one isn't empty, use it
                        if (!responseMessage && messageWithoutTasks && messageWithoutTasks.trim() !== "") {
                            responseMessage = messageWithoutTasks;
                        }
                    }
                    // Otherwise, we can process the command as normal
                    else {
                        processLlmTask({
                            taskInfo: command,
                            chatId,
                            language,
                            userData,
                        });
                    }
                }
            }

            // If we still don't have a response message, set a generic one
            if (!responseMessage) {
                if (botCommandsToRun.length > 0) {
                    responseMessage = i18next.t("common:CommandsProcessed", { lng: language, count: botCommandsToRun.length, defaultValue: "Commands processed" }) as string;
                } else if (botTasksToSuggest.length > 0) {
                    responseMessage = i18next.t("common:CommandsSuggested", { lng: language, defaultValue: "Here are the suggested next steps..." }) as string;
                } else {
                    responseMessage = i18next.t("error:SomethingWentWrong", { lng: language, defaultValue: "Something went wrong. Please try again." }) as string;
                }
            }

            // Store response in database 
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
                            text: responseMessage,
                        },
                    },
                },
                ...select,
            });
            const formattedResponseMessage = modelToGql(createdData, chatMessage_findOne);
            const fullResponseMessage = (await addSupplementalFields(prisma, userData, [formattedResponseMessage], chatMessage_findOne))[0] as ChatMessage;

            // Perform triggers for notifications, achievements, etc.
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
            wasResponseSent = true;

            // Let the user know about commands that were run or are being suggested
            if (botCommandsToRun.length > 0 || botTasksToSuggest.length > 0) {
                emitSocketEvent("llmTasks", chatId, {
                    tasks: [
                        ...botCommandsToRun.map((command) => ({
                            ...command,
                            messageId: fullResponseMessage.id,
                            status: "running" as const,
                        })),
                        ...botTasksToSuggest.map((command) => ({
                            ...command,
                            messageId: fullResponseMessage.id,
                            status: "suggested" as const,
                        })),
                    ],
                });
            }

            // Reduce user's credits by 1
            const updatedUser = await prisma.user.update({
                where: { id: userData.id },
                data: { premium: { update: { credits: { decrement: totalCost } } } },
                select: { premium: { select: { credits: true } } },
            });
            if (updatedUser.premium) {
                emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits });
            }
        },
        trace: "0081",
    });
    // If we failed without sending a response, emit an event to retry the task
    if (!wasResponseSent) {
        //TODO
    }
    // Remove typing indicator
    if (chatId && respondingBotId) {
        emitSocketEvent("typing", chatId, { stopping: [respondingBotId] });
    }
};

export const llmProcessAutoFill = async ({ data, task }: RequestAutoFillPayload) => {
    try {
        //TODO
    } catch (error) {
        logger.error("Failed to process autofill", { trace: "0331", error });
    }
};

export const llmProcess = async ({ data }: Job<LlmRequestPayload>) => {
    switch (data.__process) {
        case "BotMessage":
            return llmProcessBotMessage(data);
        case "AutoFill":
            return llmProcessAutoFill(data);
        default:
            throw new CustomError("0330", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
};

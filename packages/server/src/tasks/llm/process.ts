import { AutoFillResult, BotSettings, ChatMessage, LlmTaskInfo, Success, VALYXA_ID, toBotSettings } from "@local/shared";
import { Job } from "bull";
import i18next from "i18next";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { prismaInstance } from "../../db/instance";
import { chatMessage_findOne } from "../../endpoints/generated/chatMessage_findOne";
import { CustomError } from "../../events";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { emitSocketEvent } from "../../sockets/events";
import { processLlmTask } from "../llmTask";
import { importCommandToTask } from "./config";
import { getBotInfo } from "./context";
import { LlmRequestPayload, RequestAutoFillPayload, RequestBotMessagePayload, StartTaskPayload } from "./queue";
import { generateResponseWithFallback } from "./service";
import { ExistingTaskData, ForceGetTaskParams, forceGetTask, getValidTasksFromMessage } from "./tasks";

const parseBotInformation = (
    participants: Record<string, PreMapUserData>,
    respondingBotId: string,
    logger: { error: (message: string, data?: Record<string, any>) => unknown },
    language: string,
): BotSettings => {
    const bot = participants[respondingBotId];
    if (!bot) {
        throw new CustomError("0176", "InternalError", [language]);
    }
    return toBotSettings(bot, logger);
};

type ForceGetAndProcessCommandParams = ForceGetTaskParams;
type ForceGetAndProcessCommandResult = {
    messageWithoutTasks: string | null,
    tasksToSuggest: Omit<LlmTaskInfo, "status">[],
    cost: number
};
const forceGetAndProcessCommand = async (params: ForceGetAndProcessCommandParams): Promise<ForceGetAndProcessCommandResult> => {
    const { taskToRun, tasksToSuggest, messageWithoutTasks, cost } = await forceGetTask(params);
    if (!taskToRun || !messageWithoutTasks) {
        return { messageWithoutTasks: null, tasksToSuggest: [], cost };
    }
    processLlmTask({ taskInfo: taskToRun, ...params });
    return { messageWithoutTasks, tasksToSuggest, cost };
};

export const llmProcessBotMessage = async ({
    chatId,
    parent,
    participantsData,
    respondingBotId,
    task,
    userData,
}: RequestBotMessagePayload) => {
    const language = userData.languages[0] ?? "en";
    let wasResponseSent = false;
    let totalCost = 0;

    try {
        // Parse bot information
        const botSettings = parseBotInformation(participantsData, respondingBotId, logger, language);
        const commandToTask = await importCommandToTask(language);

        // Parse previous message
        let respondingToMessage: { id: string, text: string } | null = null;
        if (parent) {
            // Check for commands in user's message
            const { tasksToRun, messageWithoutTasks } = await getValidTasksFromMessage({
                commandToTask,
                existingData: null,
                language,
                message: parent.content,
                taskMode: task,
            });
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
            const { message: botResponse, cost } = await generateResponseWithFallback({
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
            const { tasksToRun, tasksToSuggest, messageWithoutTasks: botResponseWithoutCommands } = await getValidTasksFromMessage({
                commandToTask,
                existingData: null,
                language,
                message: botResponse,
                taskMode: task,
            });
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
        const createdData = await prismaInstance.chat_message.create({
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
        const fullResponseMessage = (await addSupplementalFields(userData, [formattedResponseMessage], chatMessage_findOne))[0] as ChatMessage;

        // Perform triggers for notifications, achievements, etc.
        await Trigger([language]).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic: true, // N/A
            hasParent: true, // N/A
            owner: { id: respondingBotId, __typename: "User" },
            objectId: fullResponseMessage.id,
            objectType: "ChatMessage",
        });
        await Trigger([language]).chatMessageCreated({
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

        // Reduce user's credits
        const updatedUser = await prismaInstance.user.update({
            where: { id: userData.id },
            data: { premium: { update: { credits: { decrement: totalCost } } } },
            select: { premium: { select: { credits: true } } },
        });
        if (updatedUser.premium) {
            emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits + "" });
        }
    } catch (error) {
        logger.error("Caught error in llmProcessBotResponse", { trace: "0081", error });
    }
    // If we failed without sending a response, emit an event to retry the task
    if (!wasResponseSent) {
        //TODO
    }
    // Remove typing indicator
    if (chatId && respondingBotId) {
        emitSocketEvent("typing", chatId, { stopping: [respondingBotId] });
    }
};

export const llmProcessAutoFill = async ({
    data,
    task,
    userData,
}: RequestAutoFillPayload): Promise<AutoFillResult> => {
    let result: Omit<LlmTaskInfo, "status"> | null = null;
    try {
        const language = userData.languages[0] ?? "en";
        const botInfo = await getBotInfo(VALYXA_ID);
        if (!botInfo) {
            throw new CustomError("0238", "InternalError", userData.languages, { task });
        }
        const participantsData = { [VALYXA_ID]: botInfo };
        const botSettings = parseBotInformation(participantsData, VALYXA_ID, logger, language);
        const commandToTask = await importCommandToTask(language);
        const { taskToRun, cost } = await forceGetTask({
            commandToTask,
            existingData: data as ExistingTaskData,
            language,
            participantsData,
            respondingBotConfig: botSettings,
            respondingBotId: VALYXA_ID,
            respondingToMessage: {
                text: Object.keys(data).length ?
                    "Your goal is to auto-fill a form. Here is the existing data:\n" + JSON.stringify(data, null, 2) + "\nRespond with the missing information"
                    : "Your goal is to auto-fill a form. The form is currently blank. Respond with the information you'd like to fill it with.",
            },
            task,
            userData,
        });
        // Reduce user's credits
        const updatedUser = await prismaInstance.user.update({
            where: { id: userData.id },
            data: { premium: { update: { credits: { decrement: cost } } } },
            select: { premium: { select: { credits: true } } },
        });
        if (updatedUser.premium) {
            emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits + "" });
        }
        // Set result
        result = taskToRun;
    } catch (error) {
        logger.error("Caught error in llmProcessAutoFill", { trace: "0331", error });
    }
    if (result) {
        return { __typename: "AutoFillResult", data: (result as Omit<LlmTaskInfo, "status">).properties };
    } else {
        throw new CustomError("0230", "InternalError", userData.languages, { task });
    }
};

export const llmProcessStartTask = async ({
    botId,
    label,
    messageId,
    properties,
    task,
    userData,
}: StartTaskPayload): Promise<Success> => {
    const result: Omit<LlmTaskInfo, "status"> | null = null;
    try {
        const language = userData.languages[0] ?? "en";
        const botInfo = await getBotInfo(botId);
        if (!botInfo) {
            throw new CustomError("0238", "InternalError", userData.languages, { task });
        }
        //TODO context is dependent on model (because of context sizes). So we need to update generateResponseWithFallback so that it can do all of this stuff itself
        // Get context
        //TODO
        // Get participantsData for every bot that appears in the context's messages
        //TODO
        const participantsData = { [VALYXA_ID]: botInfo };
        const botSettings = parseBotInformation(participantsData, VALYXA_ID, logger, language);
        const commandToTask = await importCommandToTask(language);
        //TODO
    } catch (error) {
        logger.error("Caught error in llmProcessStartTask", { trace: "0331", error });
    }
    //TODO
    return {} as any;
};

export const llmProcess = async ({ data }: Job<LlmRequestPayload>) => {
    switch (data.__process) {
        case "BotMessage":
            return llmProcessBotMessage(data);
        case "AutoFill":
            return llmProcessAutoFill(data);
        case "StartTask":
            return llmProcessStartTask(data);
        default:
            throw new CustomError("0330", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
};

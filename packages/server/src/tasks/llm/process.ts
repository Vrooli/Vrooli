import { BotSettingsConfig, ChatMessage, GetValidTasksFromMessageParams, ServerLlmTaskInfo, SessionUser, getValidTasksFromMessage, importCommandToTask } from "@local/shared";
import { Job } from "bull";
import i18next from "i18next";
import { InfoConverter, addSupplementalFields } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { chatMessage_findOne } from "../../endpoints/generated/chatMessage_findOne.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Trigger } from "../../events/trigger.js";
import { SocketService } from "../../sockets/io.js";
import { PreMapUserData, getChatParticipantData } from "../../utils/chat.js";
import { reduceUserCredits } from "../../utils/reduceCredits.js";
import { processLlmTask } from "../llmTask/queue.js";
import { ChatContextManager, stringifyTaskContexts } from "./context.js";
import { type LlmRequestPayload, type RequestBotMessagePayload } from "./queue.js";
import { ForceGetTaskParams, forceGetTask, generateResponseWithFallback } from "./service.js";

const CONTEXT_TEMPLATE_DEFAULT = "Give me a command for the task \"<TASK>\", using the context of this chat. Here is the existing data:\n```\n<DATA>\n```\nRespond with a full command to complete the task, with properties for all missing fields. Do not include any other text.";

type ForceGetTaskHelperParams = ForceGetTaskParams & {
    shouldProcessImmediately: boolean;
}
type ForceGetTaskHelperResult = {
    messageWithoutTasks: string | null,
    taskToRun: ServerLlmTaskInfo | null,
    tasksToSuggest: ServerLlmTaskInfo[],
    cost: number
};
/**
 * Wrapper around forceGetTask that handles collecting data for and immediately processing a task
 */
async function forceGetTaskHelper(params: ForceGetTaskHelperParams): Promise<ForceGetTaskHelperResult> {
    const { taskToRun, tasksToSuggest, messageWithoutTasks, cost } = await forceGetTask(params);
    if (!taskToRun || !messageWithoutTasks) {
        return { messageWithoutTasks: null, taskToRun: null, tasksToSuggest: [], cost };
    }
    if (params.shouldProcessImmediately) {
        processLlmTask({
            __process: "LlmTask",
            taskInfo: taskToRun,
            ...params,
        } as const);
    }
    return { messageWithoutTasks, taskToRun, tasksToSuggest, cost };
}

/**
 * @throws If the bot information can't be parsed
 */
async function parseBotInformationOrThrow(
    participants: Record<string, { name: string, botSettings: string }>,
    respondingBotId: string,
    throwCode: string,
) {
    const bot = participants[respondingBotId];
    const botSettings = bot ? BotSettingsConfig.deserialize(bot, logger).export().schema : null;
    if (!botSettings) {
        throw new CustomError(throwCode, "InternalError");
    }
    return botSettings;
}

type ProcessPotentialParentCommandsParams = Omit<GetValidTasksFromMessageParams, "existingData" | "logger" | "message"> & {
    chatId: string;
    parentMessage: string;
    userData: SessionUser;
};
/**
 * Helper function to process any potential commands in the parent message
 * @returns True if the message contained only commands and nothing else, 
 * meaning that we should stop early and not generate a response
 */
async function processPotentialParentCommands({
    chatId,
    commandToTask,
    language,
    mode,
    parentMessage,
    taskMode,
    userData,
}: ProcessPotentialParentCommandsParams): Promise<boolean> {
    // Check for commands in user's message
    const { tasksToRun, messageWithoutTasks } = await getValidTasksFromMessage({
        commandToTask,
        existingData: null,
        language,
        logger,
        mode,
        message: parentMessage,
        taskMode,
    });
    // Process any commands found
    for (const taskInfo of tasksToRun) {
        processLlmTask({
            __process: "LlmTask",
            taskInfo,
            chatId,
            language,
            userData,
        } as const);
    }
    // If there is no text left after removing commands, don't create a response
    const shouldStopEarly = messageWithoutTasks.trim() === "";
    return shouldStopEarly;
}

export async function llmProcessBotMessage({
    chatId,
    mode,
    model,
    parentId,
    parentMessage,
    participantsData,
    respondingBotId,
    task,
    taskContexts,
    userData,
}: RequestBotMessagePayload) {
    const language = userData.languages[0] ?? "en";
    let wasResponseSent = false;
    let totalCost = 0;

    try {
        let preMapUserData: Record<string, PreMapUserData> = {};
        // If we're in a chat and haven't found participants data, we need to fetch it
        if (!participantsData) {
            // This will update preMapUserData with the chat's participants
            await getChatParticipantData({
                chatIds: [chatId],
                includeMessageInfo: false,
                includeMessageParentInfo: false,
                preMap: { chatData: {}, messageData: {}, userData: preMapUserData },
                userData,
            });
        } else {
            preMapUserData = participantsData;
        }

        // Parse bot information
        const botSettings = await parseBotInformationOrThrow(preMapUserData, respondingBotId, "0176");
        const commandToTask = await importCommandToTask(language, logger);
        const taskMessage = taskContexts.length > 0 ? stringifyTaskContexts(task, taskContexts, CONTEXT_TEMPLATE_DEFAULT) : null;

        // Parse previous message
        if (parentMessage) {
            const shouldStopEarly = await processPotentialParentCommands({
                chatId,
                commandToTask,
                language,
                mode,
                parentMessage,
                taskMode: task,
                userData,
            });
            if (shouldStopEarly) return;
        }

        // Start typing indicator
        SocketService.get().emitSocketEvent("typing", chatId, { starting: [respondingBotId] });

        let responseMessage: string | null = null;
        let botCommandsToRun: ServerLlmTaskInfo[] = [];
        let botTasksToSuggest: ServerLlmTaskInfo[] = [];

        // If we're not in a "Start" task (the only task that allows general conversation), 
        // we'll demand a valid command immediately
        if (task !== "Start") {
            const { messageWithoutTasks, taskToRun, tasksToSuggest, cost } = await forceGetTaskHelper({
                chatId,
                commandToTask,
                language,
                latestMessage: parentId,
                mode: "text",
                participantsData,
                shouldProcessImmediately: true,//respondAsSuggestion !== true, TODO
                respondingBotConfig: botSettings,
                respondingBotId,
                task,
                taskMessage,
                userData,
            });
            responseMessage = messageWithoutTasks;
            botTasksToSuggest = tasksToSuggest;
            totalCost += cost;
            //TODO if responding as suggestion, handle taskToRun by stringifying it and adding/replacing responseMessage
        }
        // Otherwise, we'll generate a normal response and handle any commands that it contains
        else {
            // Generate bot response
            const { message: botResponse, cost } = await generateResponseWithFallback({
                chatId,
                force: false,
                latestMessage: parentId,
                mode: "text",
                participantsData,
                respondingBotConfig: botSettings,
                respondingBotId,
                stream: true,
                task,
                taskMessage,
                userData,
            });
            totalCost += cost;

            // Check for commands in bot's response
            const { tasksToRun, tasksToSuggest, messageWithoutTasks: botResponseWithoutCommands } = await getValidTasksFromMessage({
                commandToTask,
                existingData: null,
                language,
                logger,
                message: botResponse,
                mode,
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
                    const { messageWithoutTasks, cost } = await forceGetTaskHelper({
                        chatId,
                        commandToTask,
                        language,
                        latestMessage: parentId,
                        mode: "text",
                        participantsData,
                        respondingBotConfig: botSettings,
                        respondingBotId,
                        shouldProcessImmediately: true,
                        task: command.task,
                        //TODO not sure if we should provide contexts taskMessage here
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
                        __process: "LlmTask",
                        taskInfo: command,
                        chatId,
                        language,
                        userData,
                    } as const);
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

        // Determine if the response should be stored in the database, and handle any 
        // actions only taken for stored responses
        //TODO there are other cases where we shouldn't store a response, such as for autofill
        const shouldStoreResponse = typeof chatId === "string";
        if (shouldStoreResponse) {
            // Store response in database 
            const select = InfoConverter.get().fromPartialApiToPrismaSelect(chatMessage_findOne);
            const createdData = await DbProvider.get().chat_message.create({
                data: {
                    text: responseMessage,
                    chat: { connect: { id: BigInt(chatId) } },
                    parent: parentId ? { connect: { id: BigInt(parentId) } } : undefined,
                    user: { connect: { id: BigInt(respondingBotId) } },
                    //TODO need to check existing number of parent's children to set versionIndex. Find somewhere to do this without having to call prisma again
                },
                ...select,
            });
            // Store message in cache
            await (new ChatContextManager(model)).addMessage({
                __type: "Create",
                chatId,
                messageId: createdData.id.toString(),
                parentId: parentId ?? null,
                text: responseMessage,
                userId: respondingBotId,
            });

            const formattedResponseMessage = InfoConverter.get().fromDbToApi(createdData, chatMessage_findOne);
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
        }

        wasResponseSent = true;

        // Let the user know about commands that were run or are being suggested
        if (botCommandsToRun.length > 0 || botTasksToSuggest.length > 0) {
            SocketService.get().emitSocketEvent("llmTasks", chatId, {
                tasks: [
                    ...botCommandsToRun.map((command) => ({
                        ...command,
                        lastUpdated: new Date().toISOString(),
                        status: "Running" as const,
                    })),
                    ...botTasksToSuggest.map((command) => ({
                        ...command,
                        lastUpdated: new Date().toISOString(),
                        status: "Suggested" as const,
                    })),
                ],
            });
        }

        // Reduce user's credits
        await reduceUserCredits(userData.id, totalCost);
    } catch (error) {
        logger.error("Caught error in llmProcessBotResponse", { trace: "0081", error });
    }
    // If we failed without sending a response, emit an event to retry the task
    if (!wasResponseSent) {
        //TODO
    }
    // Remove typing indicator
    if (chatId && respondingBotId) {
        SocketService.get().emitSocketEvent("typing", chatId, { stopping: [respondingBotId] });
    }
}

export async function llmProcess({ data }: Job<LlmRequestPayload>) {
    switch (data.__process) {
        case "BotMessage":
            return llmProcessBotMessage(data);
        case "Test":
            logger.info("llmProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0330", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}

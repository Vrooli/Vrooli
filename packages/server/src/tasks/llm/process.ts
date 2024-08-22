import { AutoFillResult, ChatMessage, ExistingTaskData, ServerLlmTaskInfo, VALYXA_ID, getValidTasksFromMessage, importCommandToTask, parseBotInformation } from "@local/shared";
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
import { ChatMessageModelInfo } from "../../models/base/types";
import { emitSocketEvent } from "../../sockets/events";
import { PreMapChatData, PreMapMessageData, PreMapUserData, getChatParticipantData } from "../../utils/chat";
import { getSingleTypePermissions } from "../../validators/permissions";
import { processLlmTask } from "../llmTask";
import { getBotInfo } from "./context";
import { type LlmRequestPayload, type RequestAutoFillPayload, type RequestBotMessagePayload, type StartLlmTaskPayload } from "./queue";
import { ForceGetTaskParams, forceGetTask, generateResponseWithFallback } from "./service";

type ForceGetAndProcessCommandParams = ForceGetTaskParams;
type ForceGetAndProcessCommandResult = {
    messageWithoutTasks: string | null,
    tasksToSuggest: ServerLlmTaskInfo[],
    cost: number
};
async function forceGetAndProcessCommand(params: ForceGetAndProcessCommandParams): Promise<ForceGetAndProcessCommandResult> {
    const { taskToRun, tasksToSuggest, messageWithoutTasks, cost } = await forceGetTask(params);
    if (!taskToRun || !messageWithoutTasks) {
        return { messageWithoutTasks: null, tasksToSuggest: [], cost };
    }
    processLlmTask({
        __process: "LlmTask",
        taskInfo: taskToRun,
        ...params,
    } as const);
    return { messageWithoutTasks, tasksToSuggest, cost };
}

export async function llmProcessBotMessage({
    chatId,
    mode,
    parent,
    participantsData,
    respondingBotId,
    task,
    userData,
}: RequestBotMessagePayload) {
    const language = userData.languages[0] ?? "en";
    let wasResponseSent = false;
    let totalCost = 0;

    try {
        // Parse bot information
        const botSettings = parseBotInformation(participantsData, respondingBotId, logger);
        if (!botSettings) {
            throw new CustomError("0176", "InternalError", [language]);
        }
        const commandToTask = await importCommandToTask(language, logger);
        const latestMessage = parent?.id ?? null;

        // Parse previous message
        if (parent) {
            // Check for commands in user's message
            const { tasksToRun, messageWithoutTasks } = await getValidTasksFromMessage({
                commandToTask,
                existingData: null,
                language,
                logger,
                mode,
                message: parent.content,
                taskMode: task,
            });
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
            if (messageWithoutTasks.trim() === "") return;
        }

        // Start typing indicator
        emitSocketEvent("typing", chatId, { starting: [respondingBotId] });

        let responseMessage: string | null = null;
        let botCommandsToRun: ServerLlmTaskInfo[] = [];
        let botTasksToSuggest: ServerLlmTaskInfo[] = [];

        // If we're not in a "Start" task (the only task that allows general conversation), 
        // we'll demand a valid command immediately
        if (task !== "Start") {
            const { messageWithoutTasks, tasksToSuggest, cost } = await forceGetAndProcessCommand({
                chatId,
                commandToTask,
                language,
                latestMessage,
                mode: "text",
                participantsData,
                respondingBotConfig: botSettings,
                respondingBotId,
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
                latestMessage,
                mode: "text",
                participantsData,
                respondingBotConfig: botSettings,
                respondingBotId,
                stream: true,
                task,
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
                    const { messageWithoutTasks, cost } = await forceGetAndProcessCommand({
                        chatId,
                        commandToTask,
                        language,
                        latestMessage,
                        mode: "text",
                        participantsData,
                        respondingBotConfig: botSettings,
                        respondingBotId,
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
                        lastUpdated: new Date().toISOString(),
                        messageId: fullResponseMessage.id,
                        status: "Running" as const,
                    })),
                    ...botTasksToSuggest.map((command) => ({
                        ...command,
                        lastUpdated: new Date().toISOString(),
                        messageId: fullResponseMessage.id,
                        status: "Suggested" as const,
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
}

export async function llmProcessAutoFill({
    data,
    task,
    userData,
}: RequestAutoFillPayload): Promise<AutoFillResult> {
    let result: ServerLlmTaskInfo | null = null;
    try {
        const language = userData.languages[0] ?? "en";
        const botInfo = await getBotInfo(VALYXA_ID);
        if (!botInfo) {
            throw new CustomError("0238", "InternalError", userData.languages, { task });
        }
        const participantsData = { [VALYXA_ID]: botInfo };
        const botSettings = parseBotInformation(participantsData, VALYXA_ID, logger);
        if (!botSettings) {
            throw new CustomError("0600", "InternalError", [language]);
        }
        const commandToTask = await importCommandToTask(language, logger);
        let taskMessage = "Your goal is to auto-fill a form with valid and relevant properties";
        if (Object.keys(data).length) {
            taskMessage += "\nHere are the existing properties:\n" + JSON.stringify(data, null, 2);
            taskMessage += "\nRespond with any missing properties (including at minimum the missing required properties) to complete the form.";
        }
        const { taskToRun, cost } = await forceGetTask({
            commandToTask,
            existingData: data as ExistingTaskData,
            language,
            mode: "json",
            participantsData,
            respondingBotConfig: botSettings,
            respondingBotId: VALYXA_ID,
            taskMessage,
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
        return { __typename: "AutoFillResult", data: (result as ServerLlmTaskInfo).properties };
    } else {
        throw new CustomError("0230", "InternalError", userData.languages, { task });
    }
}

/**
 * Process for starting an LLM task, not including the "Start" task itself (despite the confusing name)
 */
export async function llmProcessStartTask({
    botId,
    messageId,
    properties,
    task,
    taskId,
    userData,
}: StartLlmTaskPayload) {
    let chatId: string | null = null;
    try {
        const language = userData.languages[0] ?? "en";
        // Use delete permissions to determine if we can perform a task
        const { canDelete: canStartTask } = await getSingleTypePermissions<ChatMessageModelInfo["GqlPermission"]>("ChatMessage", [messageId], userData);
        if (!Array.isArray(canStartTask) || !canStartTask.every(Boolean)) {
            throw new CustomError("0487", "Unauthorized", userData.languages, { task });
        }
        // Initialize objects to store queried information
        const preMapChatData: Record<string, PreMapChatData> = {};
        const preMapMessageData: Record<string, PreMapMessageData> = {};
        const preMapUserData: Record<string, PreMapUserData> = {};
        // Collect chat and participant information
        const userId = userData.id;
        await getChatParticipantData({
            includeMessageInfo: true,
            includeMessageParentInfo: true,
            messageIds: [messageId],
            preMapChatData,
            preMapMessageData,
            preMapUserData,
            userData,
        });
        const messageData = preMapMessageData[messageId];
        chatId = messageData?.chatId;
        const chat: PreMapChatData | undefined = chatId ? preMapChatData[chatId] : undefined;
        const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMapUserData[id]).filter(b => b) ?? [];
        if (
            !messageData ||
            !messageData.userId ||
            !chatId ||
            !chat ||
            bots.length === 0
        ) {
            throw new CustomError("0415", "InternalError", userData.languages, { messageId, messageData, userId });
        }
        const respondingBotId = bots.find(b => b.id === botId)?.id ?? bots[0].id;
        const respondingBotConfig = parseBotInformation(preMapUserData, respondingBotId, logger);
        if (!respondingBotConfig) {
            throw new CustomError("0601", "InternalError", [language]);
        }
        // Let the UI know that the task is Running
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, { updates: [{ taskId, status: "Running" }] });
        }
        // Generate information to run the task
        const commandToTask = await importCommandToTask(language, logger);
        const { taskToRun, cost } = await forceGetTask({
            chatId,
            commandToTask,
            language,
            latestMessage: messageId,
            mode: "json",
            participantsData: preMapUserData,
            respondingBotConfig,
            respondingBotId,
            taskMessage: Object.keys(properties).length
                ? `Give me a command for the task "${task}", using the context of this chat. Here is the existing data:\n\`\`\`\n${JSON.stringify(properties, null, 2)}\n\`\`\`\nRespond with a full command to complete the task, with properties for all missing fields. Do not include any other text.`
                : `Give me a command for the task "${task}", using the context of this chat. Respond with a full command to complete the task, with properties for all missing fields. Do not include any other text.`,
            task,
            userData,
        });
        if (!taskToRun) {
            throw new CustomError("0541", "InternalError", userData.languages, { task });
        }
        // Reduce user's credits
        const updatedUser = await prismaInstance.user.update({
            where: { id: userData.id },
            data: { premium: { update: { credits: { decrement: cost } } } },
            select: { premium: { select: { credits: true } } },
        });
        if (updatedUser.premium) {
            emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits + "" });
        }
        // Run the task
        const taskData = {
            __process: "LlmTask",
            chatId,
            language,
            taskInfo: { ...taskToRun, messageId },
            userData,
        } as const;
        processLlmTask(taskData);
    } catch (error) {
        if (chatId) {
            emitSocketEvent("llmTasks", chatId, { updates: [{ taskId, status: "Failed" }] });
        }
        logger.error("Caught error in llmProcessStartTask", { trace: "0331", error });
        return { __typename: "Success" as const, success: false };
    }
}

export async function llmProcess({ data }: Job<LlmRequestPayload>) {
    switch (data.__process) {
        case "BotMessage":
            return llmProcessBotMessage(data);
        case "AutoFill":
            return llmProcessAutoFill(data);
        case "StartTask":
            return llmProcessStartTask(data);
        case "Test":
            logger.info("llmProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0330", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
}

import { BotSettings, BotSettingsTranslation, ChatSocketEventPayloads, CommandToTask, ExistingTaskData, LlmTask, ServerLlmTaskInfo, getLlmConfigLocation, getStructuredTaskConfig, getValidTasksFromMessage, toBotSettings } from "@local/shared";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { SessionUserToken } from "../../types";
import { objectToYaml } from "../../utils";
import { bestTranslation } from "../../utils/bestTranslation";
import { PreMapUserData } from "../../utils/chat";
import { ChatContextCollector, ContextInfo, MessageContextInfo } from "./context";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry, LlmServiceState } from "./registry";

type SimpleChatMessageData = {
    id: string;
    translations: { language: string, text: string }[];
}

export type LanguageModelMessage = Partial<ContextInfo> & {
    role: "user" | "assistant";
    content: string;
}
export type LanguageModelContext = {
    messages: LanguageModelMessage[];
    systemMessage: string;
}

export type EstimateTokensParams = {
    /** The requested model to base token logic on */
    model: string;
    /** The text to estimate tokens for */
    text: string;
}
export type EstimateTokensResult = {
    /** The name of the token estimation model used (if requested one was invalid/incomplete) */
    model: string;
    /** The estimated amount of tokens */
    tokens: number;
}
export type GetConfigObjectParams = {
    botSettings: BotSettings,
    includeInitMessage: boolean,
    userData: Pick<SessionUserToken, "languages">,
    task: LlmTask | `${LlmTask}`,
    force: boolean,
}
export type GenerateContextParams = {
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    contextInfo: ContextInfo[];
    model: string;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    respondingToMessage: {
        id?: string | null;
        text: string;
    } | null;
    task: LlmTask | `${LlmTask}`;
    userData: SessionUserToken;
}
export type GenerateResponseParams = {
    messages: LanguageModelMessage[];
    model: string;
    systemMessage: string;
    userData: Pick<SessionUserToken, "languages" | "name">;
}
export type GenerateResponseResult = {
    message: string;
    cost: number;
}
export type GenerateResponseStreamingResult = {
    __type: "stream" | "end" | "error";
    /**
     * The current stream of the message if __type is "stream" 
     * (so we can minimize the amount of data sent to the UI), 
     * or the full message if __type is "end"
     */
    message: string;
    /**
     * The accumulated cost of the response so far, or the total 
     * cost if __type is "end"
     */
    cost: number;
}

export type GetResponseCostParams = {
    model: string;
    usage: { input: number, output: number };
}

export interface LanguageModelService<GenerateNameType extends string, TokenNameType extends string> {
    /** Identifier for service */
    __id: LlmServiceId;
    /** Estimate the amount of tokens a string is */
    estimateTokens(params: EstimateTokensParams): EstimateTokensResult;
    /** Generates config for setting up bot persona and task instructions */
    getConfigObject(params: GetConfigObjectParams): Promise<Record<string, unknown>>;
    /** Generates context prompt from messages */
    generateContext(params: GenerateContextParams): Promise<LanguageModelContext>;
    /** Generate a message response, non-streaming */
    generateResponse(params: GenerateResponseParams): Promise<GenerateResponseResult>;
    /**  Generate a message repsonse, streaming */
    generateResponseStreaming(params: GenerateResponseParams): AsyncIterable<GenerateResponseStreamingResult>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** 
     * Calculates the api credits used by the generation of an LLM response, 
     * based on the model's input token and output token costs (in cents per 1_000_000 tokens).
     * 
     * NOTE: Instead of using decimals, the cost is in cents multiplied by 1_000_000. This 
     * can be normalized in the UI so the number isn't overwhelming. 
     * 
     * @returns the total cost of the response
     */
    getResponseCost(params: GetResponseCostParams): number;
    /** @returns the estimation method for the model */
    getEstimationMethod(model?: string | null): TokenNameType;
    /** @returns a list of token estimation types used by this service */
    getEstimationTypes(): readonly TokenNameType[];
    /** Convert a preferred model to an available one */
    getModel(model?: string | null): GenerateNameType;
    /** Converts error received by service to a standardized type */
    getErrorType(error: unknown): LlmServiceErrorType;
}

/**
 * Basic token estimation, which can be used as a placeholder 
 * until a more advanced one is implemented
 * @returns The name of the token estimation model, and the estimated amount of tokens
 */
export const tokenEstimationDefault = ({ text }: EstimateTokensParams) => {
    const words = text.split(" ");
    let tokens = 0;
    for (const word of words) {
        // Add token for each 5 characters
        tokens += Math.floor(word.length / 5);
        // Also increment for whitespace
        tokens++;
    }
    return { model: "default" as const, tokens };
};

/**
 * General configuration builder for providing context for language model services. 
 * 
 * Can be used as a fallback for services that don't have a specific implementation.
 */
export const getDefaultConfigObject = async ({
    botSettings,
    includeInitMessage,
    userData,
    task,
    force,
}: GetConfigObjectParams) => {
    const translationsList = Object.entries(botSettings?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
    let translation = (bestTranslation(translationsList, userData.languages) ?? {}) as BotSettingsTranslation & { language?: string };

    const name: string | undefined = botSettings.name;
    const initMessage = translation.startingMessage?.length
        ? translation.startingMessage
        : name
            ? `Hello👋, I'm ${name}. How can I help you today?`
            : "Hello👋, how can I help you today?";
    delete translation.startingMessage;
    delete translation.language;
    // Remove empty values from the translation object
    translation = Object.fromEntries(Object.entries(translation).filter(([_, value]) => value !== ""));

    const temp = await getLlmConfigLocation();
    const taskConfig = await getStructuredTaskConfig(task, force, userData.languages[0] ?? "en", logger);
    const configObject = {
        ai_assistant: {
            metadata: {
                name: botSettings.name ?? "Bot",
            },
            ...(includeInitMessage ? { init_message: initMessage } : {}),
            personality: Object.keys(translation).length ? { ...translation } : undefined,
            ...taskConfig,
        },
    };

    return configObject;
};

/**
 * General configuration builder for providing context for language model services, 
 * including all messages in the context, as well as information needed for the bot's 
 * persona and for completing the task.
 * 
 * Can be used as a fallback for services that don't have a specific implementation.
 */
export const generateDefaultContext = async <GenerateNameType extends string, TokenNameType extends string>({
    contextInfo,
    model,
    respondingBotId,
    respondingBotConfig,
    participantsData,
    task,
    force,
    userData,
    service,
}: GenerateContextParams & {
    service: LanguageModelService<GenerateNameType, TokenNameType>;
}): Promise<LanguageModelContext> => {
    const messages: LanguageModelContext["messages"] = [];

    // Construct the initial YAML configuration message for relevant participants
    let systemMessage = "You are a helpful assistant for an app named Vrooli. Please follow the configuration below to best suit each user's needs:\n\n";

    // Add information to set up the bot's persona and task instructions
    const config = await service.getConfigObject({
        botSettings: respondingBotConfig,
        includeInitMessage: contextInfo.length === 0,
        userData,
        task,
        force,
    });
    systemMessage += objectToYaml(config) + "\n";

    // If there are other users/bots in the chat
    const hasOtherParticipants = participantsData && Object.keys(participantsData).length > 2;
    if (hasOtherParticipants) {
        // Add info about message labeling
        systemMessage += "In the conversation, each message will be preceded by a participant identifier in the format [Role: Name], where 'Role' is either 'User' or 'Bot', and 'Name' is the name of the user or bot. For example:\n";
        systemMessage += "- [User: John]: This is a message from a user named John.\n";
        systemMessage += "- [Bot: AssistantBot]: This is a message from a bot named AssistantBot.\n";
        systemMessage += "Please use these identifiers to understand the context of each message and generate appropriate responses.\n";

        // If there are other bots besides the one responding, add their configurations 
        // and instructions for how to prompt them
        const otherBots = Object.entries(participantsData ?? {}).filter(([id, data]) => id !== respondingBotId && data.isBot);
        if (otherBots.length > 0) {
            systemMessage += "\n";
            systemMessage += "There are other bots in the conversation. You may prompt these if you feel their input is necessary for generating a response. Use this only when strictly necessary. Here are the configurations for the other bots:\n\n";

            for (const [_, data] of otherBots) {
                // Find bot personality
                const botSettings = toBotSettings(data, logger);
                const translationsList = Object.entries(botSettings?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
                const translation = (bestTranslation(translationsList, userData.languages) ?? {}) as BotSettingsTranslation & { language?: string };
                delete translation.language;
                // Construct an object with the bot's configuration
                const botConfig = {
                    // In the future, we can wrap this with the bot's role in the organization (when relevant)
                    metadata: {
                        name: data.name,
                    },
                    personality: { ...translation },
                };

                // Add the bot's configuration to the system message
                systemMessage += `Configuration for bot named ${data.name}:\n`;
                systemMessage += objectToYaml(botConfig) + "\n\n";
            }

            systemMessage += "To prompt a bot, use the following format: @botName prompt. For example, to prompt a bot named AssistantBot, use @AssistantBot prompt.\n";
        }
    }

    // Calculate token size for the YAML configuration
    const { tokens: systemMessageSize } = service.estimateTokens({ text: systemMessage, model });
    const maxContentSize = service.getContextSize(model);

    // Fetch messages from the database
    const messagesFromDB = await fetchMessagesFromDatabase(contextInfo.filter(info => info.__type === "message").map(info => (info as MessageContextInfo).messageId));
    let currentTokenCount = systemMessageSize;

    // If the YAML configuration exceeds the context size, omit context entirely
    if (currentTokenCount > maxContentSize) {
        return { messages: [], systemMessage: "" };
    }

    // Convert every message into a role/content pair
    for (const context of contextInfo) {
        const messageData = context.__type === "message"
            ? messagesFromDB.find(message => message.id === context.messageId)
            : { translations: [{ language: context.language, text: context.text }] };
        if (!messageData) continue;

        const messageTranslation = messageData.translations.find(translation => translation.language === context.language);
        if (!messageTranslation) continue;

        const userName = (context.userId && participantsData) ? participantsData[context.userId]?.name : undefined;
        const isBot = (context.userId && participantsData) ? participantsData[context.userId]?.isBot : true;
        const role: LanguageModelMessage["role"] = isBot ? "assistant" : "user";

        const content = hasOtherParticipants ? `[${role === "assistant" ? "Bot" : "User"}: ${userName ?? "Unknown"}]: ${messageTranslation.text}` : messageTranslation.text;
        const tokenSize = context.tokenSize + service.estimateTokens({ text: content, model }).tokens;

        // Stop if adding this message would exceed the context size
        if (currentTokenCount + tokenSize > maxContentSize) {
            break;
        }

        // Otherwise, increment the token count and add the message
        currentTokenCount += tokenSize;
        messages.push({ role, content });
    }

    return { messages, systemMessage };
};

/**
 * Fetches messages from the database for given message IDs.
 * 
 * @param messageIds Array of message IDs
 * @returns An array of message objects
 */
export const fetchMessagesFromDatabase = async (messageIds: string[]): Promise<SimpleChatMessageData[]> => {
    const messages = await prismaInstance.chat_message.findMany({
        where: {
            id: { in: messageIds },
        },
        select: {
            id: true,
            translations: {
                select: {
                    language: true,
                    text: true,
                },
            },
        },
    }) ?? [];

    return messages;
};

type GenerateResponseWithFallbackParams = {
    chatId?: string | null;
    /** 
    * Determines if context should be written to force the response to be a command 
    */
    force: boolean;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    respondingToMessage: {
        id?: string | null;
        text: string;
    } | null;
    /** If we should use a stream to show the response as its being generated */
    stream: boolean;
    task: LlmTask | `${LlmTask}`;
    userData: SessionUserToken;
}

/**
 * Attempts to generate a response using a preferred LLM service, falling back to 
 * other services if the preferred one is unavailable.
 */
export const generateResponseWithFallback = async ({
    chatId,
    force,
    participantsData,
    respondingBotConfig,
    respondingBotId,
    respondingToMessage,
    stream,
    task,
    userData,
}: GenerateResponseWithFallbackParams): Promise<GenerateResponseResult> => {
    const retryLimit = 3; // Set a limit to prevent infinite loops
    let attempts = 0;

    while (attempts < retryLimit) {
        const serviceId = LlmServiceRegistry.get().getBestService(respondingBotConfig.model);
        if (!serviceId) {
            throw new CustomError("0252", "InternalError", userData.languages, { respondingBotConfig });
        }

        try {
            // Try using the service to generate a response
            const serviceInstance = LlmServiceRegistry.get().getService(serviceId) as LanguageModelService<string, string>;
            const model = serviceInstance.getModel(respondingBotConfig?.model);
            const contextInfo = await (new ChatContextCollector(serviceInstance)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessage);
            const { messages, systemMessage } = await serviceInstance.generateContext({
                contextInfo,
                force,
                model,
                participantsData,
                respondingBotId,
                respondingBotConfig,
                respondingToMessage,
                task,
                userData,
            });
            let responseMessage = "";
            let finalCost = 0;
            // Stream the response if requested and we're in a chat
            if (chatId && stream) {
                const response = serviceInstance.generateResponseStreaming({
                    messages,
                    model,
                    systemMessage,
                    userData,
                });
                let messageStreamStarted = false;
                for await (const { __type, message, cost } of response) {
                    if (__type === "end") {
                        responseMessage = message;
                        finalCost = cost;
                    } else if (__type === "error") {
                        responseMessage = message;
                        finalCost = cost;
                    }
                    const payload: ChatSocketEventPayloads["responseStream"] = { __type, message };
                    // At the start of the stream, send the bot's ID so the UI can identify which bot is responding
                    if (!messageStreamStarted) {
                        payload.botId = respondingBotId;
                        messageStreamStarted = true;
                    }
                    emitSocketEvent("responseStream", chatId, { __type, message });
                }
            } else {
                const response = await serviceInstance.generateResponse({
                    messages,
                    model,
                    systemMessage,
                    userData,
                });
                responseMessage = response.message;
                finalCost = response.cost;
            }
            return { message: responseMessage, cost: finalCost };
        } catch (error) {
            const serviceState = LlmServiceRegistry.get().getServiceState(serviceId);
            // If the service is still active, then the error was likely due 
            // to the request we made. So we'll throw the error instead of retrying.
            if (serviceState === LlmServiceState.Active) {
                throw error;
            }
            // Otherwise, we'll retry with the next best service
        }

        attempts++;
    }

    throw new CustomError("0253", "InternalError", userData.languages, { respondingBotConfig });
};

export type ForceGetTaskParams = {
    chatId?: string | null,
    commandToTask: CommandToTask,
    existingData?: ExistingTaskData | null,
    language: string,
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings,
    respondingBotId: string,
    respondingToMessage: {
        id?: string | null,
        text: string,
    } | null,
    task: LlmTask | `${LlmTask}`,
    userData: SessionUserToken,
}

/**
 * Repeatedly generates a bot response until it contains a valid task.
 * @returns The valid task and the rest of the message without the tasks
 */
export const forceGetTask = async ({
    chatId,
    commandToTask,
    existingData,
    language,
    participantsData,
    respondingBotConfig,
    respondingBotId,
    respondingToMessage,
    task,
    userData,
}: ForceGetTaskParams): Promise<{
    taskToRun: ServerLlmTaskInfo | null,
    tasksToSuggest: ServerLlmTaskInfo[],
    messageWithoutTasks: string | null,
    cost: number
}> => {
    let retryCount = 0;
    const MAX_RETRIES = 3; // Set a maximum number of retries to avoid infinite loops
    let totalCost = 0;

    while (retryCount < MAX_RETRIES) {
        const { message, cost } = await generateResponseWithFallback({
            chatId,
            force: true, // Force the bot to respond with a task
            participantsData,
            respondingBotConfig,
            respondingBotId,
            respondingToMessage,
            stream: false,
            task,
            userData,
        });
        totalCost += cost;

        // Check for tasks in the start response
        const { tasksToRun, tasksToSuggest, messageWithoutTasks } = await getValidTasksFromMessage({
            commandToTask,
            existingData: existingData ?? null,
            language,
            logger,
            message,
            taskMode: task,
        });

        // If a valid task is found, return it
        if (tasksToRun.length > 0) {
            // Only use the first task found
            return { taskToRun: tasksToRun[0], tasksToSuggest, messageWithoutTasks, cost };
        } else {
            // Increment the retry count if no task is found
            retryCount++;
            logger.warning(`No task found in start response. Retrying... (${retryCount}/${MAX_RETRIES})`, { trace: "0349", chatId, respondingBotId, task });
        }
    }

    logger.error("Failed to find a task in start response after maximum retries.", { trace: "0350", chatId, respondingBotId, task });
    return { taskToRun: null, tasksToSuggest: [], messageWithoutTasks: null, cost: totalCost };
};

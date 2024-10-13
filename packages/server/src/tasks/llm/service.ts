import { API_CREDITS_MULTIPLIER, BotSettings, BotSettingsTranslation, ChatSocketEventPayloads, CommandToTask, ExistingTaskData, LanguageModelResponseMode, LlmTask, LlmTaskStructuredConfig, ModelInfo, ServerLlmTaskInfo, getStructuredTaskConfig, getTranslation, getValidTasksFromMessage, toBotSettings } from "@local/shared";
import { cudHelper } from "../../actions/cuds";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { SessionUserToken } from "../../types";
import { objectToYaml } from "../../utils";
import { PreMapUserData } from "../../utils/chat";
import { ChatContextCollector, CollectMessageContextInfoParams, ContextInfo, MessageContextInfo } from "./context";
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
    mode: LanguageModelResponseMode,
    userData: Pick<SessionUserToken, "languages">,
    task: LlmTask | `${LlmTask}` | undefined,
    force: boolean,
}
export type GenerateContextParams = {
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    contextInfo: ContextInfo[];
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    /**
     * The model to use for generating the response.
     */
    model: string;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    task: LlmTask | `${LlmTask}` | undefined;
    taskMessage?: string | null;
    userData: SessionUserToken;
}
export type GenerateResponseParams = {
    /**
     * Messages to include as context for the response. 
     * Typically the whole chat history tree, or as many as you can fit 
     * within the current token limit.
     */
    messages: LanguageModelMessage[];
    /**
     * The maximum number of tokens to output (meaning the input cost is not included). 
     * The model may (and most likely will) stop before reaching this limit.
     * If null, the service will decide what the maximum output should be.
     */
    maxTokens: number | null;
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    /**
     * The model to use for generating the response.
     */
    model: string;
    /**
     * The system message to include in the context. This is typically
     * the configuration for the bot and task instructions.
     */
    systemMessage: string;
    /**
     * Information about the user requesting the response.
     */
    userData: Pick<SessionUserToken, "languages" | "name">;
}
export type GenerateResponseResult = {
    /** How many attempts it took to generate a reponse */
    attempts: number;
    /** The generated message */
    message: string;
    /** How many API credits were used to generate the response */
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

export type GetOutputTokenLimitParams = {
    maxCredits: bigint;
    model: string;
    inputTokens: number;
}

export type GetOutputTokenLimitResult = {
    cost: number;
    isSafe: boolean;
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
    /** @returns Information about the available models */
    getModelInfo(): Record<GenerateNameType, ModelInfo>;
    /** @returns the max output window for the model */
    getMaxOutputTokens(requestedModel?: string | null): number;
    /**
     * Calculates the maximum number of tokens that can be output by the model 
     * to stay under the cost limit.
     * @param maxCredits The maximum cost (in cents * 1_000_000) that the response can have
     * @param model The model to use for the calculation
     * @param inputTokens The number of tokens in the input
     * @returns The maximum number of tokens that can be output, or 0 if the cost is too low 
     * (i.e. the input already exceeds the cost limit)
     */
    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number;
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
    /** 
     * Checks if the input contains potentially harmful content 
     * @returns true if the input is safe, false otherwise
     */
    safeInputCheck(input: string): Promise<GetOutputTokenLimitResult>;
}

/**
 * Basic token estimation, which can be used as a placeholder 
 * until a more advanced one is implemented
 * @returns The name of the token estimation model, and the estimated amount of tokens
 */
export function tokenEstimationDefault({ text }: EstimateTokensParams) {
    const words = text.split(" ");
    let tokens = 0;
    for (const word of words) {
        // Add token for each 5 characters
        // eslint-disable-next-line no-magic-numbers
        tokens += Math.floor(word.length / 5);
        // Also increment for whitespace
        tokens++;
    }
    return { model: "default" as const, tokens };
}

/**
 * General configuration builder for providing context for language model services. 
 * 
 * Can be used as a fallback for services that don't have a specific implementation.
 */
export async function getDefaultConfigObject({
    botSettings,
    includeInitMessage,
    mode,
    userData,
    task,
    force,
}: GetConfigObjectParams) {
    const translationsList = Object.entries(botSettings?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
    let translation = getTranslation({ translations: translationsList }, userData.languages) as BotSettingsTranslation & { language?: string };

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

    let taskConfig: LlmTaskStructuredConfig | undefined;
    if (task) {
        taskConfig = await getStructuredTaskConfig({
            force,
            language: userData.languages[0] ?? "en",
            logger,
            mode,
            task,
        });
    }
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
}

/**
 * General configuration builder for providing context for language model services.
 * 
 * NOTE: Currently, this does not add the system message to the context. Services 
 * must handle this themselves.
 * 
 * Can be used as a fallback for services that don't have a specific implementation.
 */
export async function generateDefaultContext<GenerateNameType extends string, TokenNameType extends string>({
    contextInfo,
    mode,
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
}): Promise<LanguageModelContext> {
    const messages: LanguageModelContext["messages"] = [];

    // Construct the initial YAML configuration message for relevant participants
    let systemMessage = "You are a helpful assistant for an app named Vrooli. Please follow the configuration below to best suit each user's needs:\n\n";

    // Add the current timestamp
    const currentTimestamp = new Date().toISOString();
    systemMessage += `current_timestamp: ${currentTimestamp}\n\n`;

    // Add information to set up the bot's persona and task instructions
    const config = await service.getConfigObject({
        botSettings: respondingBotConfig,
        includeInitMessage: contextInfo.length === 0,
        mode,
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
                const translation = getTranslation({ translations: translationsList }, userData.languages) as BotSettingsTranslation & { language?: string };
                delete translation.language;
                // Construct an object with the bot's configuration
                const botConfig = {
                    // In the future, we can wrap this with the bot's role in the team (when relevant)
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
}

/**
 * Default output token calculator to determine available output tokens based on cost limit.
 * 
 * NOTE: input and output costs should be in cents per API_CREDITS_MULTIPLIER tokens. 
 * Since credits are stored with this multiplier already, it should cancel out (meaning 
 * we won't be using API_CREDITS_MULTIPLIER in the calculation).
 */
export function getDefaultMaxOutputTokensRestrained<GenerateNameType extends string, TokenNameType extends string>(
    { maxCredits, model, inputTokens }: GetOutputTokenLimitParams,
    service: LanguageModelService<GenerateNameType, TokenNameType>,
) {
    const modelToUse = service.getModel(model);
    const modelInfo = service.getModelInfo()[modelToUse];

    if (!modelInfo || !modelInfo.inputCost || !modelInfo.outputCost) {
        throw new Error(`Model "${model}" (converted to ${modelToUse}) not found in cost records`);
    }

    const inputCost = BigInt(modelInfo.inputCost * inputTokens);
    const remainingCredits = BigInt(maxCredits) - BigInt(inputCost);

    if (remainingCredits <= 0) {
        return 0;
    }

    const maxOutputTokensRestrained = remainingCredits / BigInt(Math.floor(modelInfo.outputCost));
    return Math.min(Math.max(0, Number(maxOutputTokensRestrained)), service.getMaxOutputTokens(model) - inputTokens);
}

/**
 * Default cost calculator for language model input and output tokens.
 * 
 * NOTE: input and output costs should be in cents per API_CREDITS_MULTIPLIER tokens. 
 * Since credits are stored with this multiplier already, it should cancel out (meaning 
 * we won't be using API_CREDITS_MULTIPLIER in the calculation).
 */
export function getDefaultResponseCost<GenerateNameType extends string, TokenNameType extends string>(
    { model, usage }: GetResponseCostParams,
    service: LanguageModelService<GenerateNameType, TokenNameType>,
) {
    const { input, output } = usage;
    const modelToUse = service.getModel(model);
    const modelInfo = service.getModelInfo()[modelToUse];

    if (!modelInfo || !modelInfo.inputCost || !modelInfo.outputCost) {
        throw new Error(`Model "${model}" (converted to ${modelToUse}) not found in cost records`);
    }

    return Math.max((modelInfo.inputCost * input), 0) + Math.max((modelInfo.outputCost * output), 0);
}

/**
 * Fetches messages from the database for given message IDs.
 * 
 * @param messageIds Array of message IDs
 * @returns An array of message objects
 */
export async function fetchMessagesFromDatabase(messageIds: string[]): Promise<SimpleChatMessageData[]> {
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
}

export type CreditValue = number | string | bigint;

/**
 * Calculates the maximum credits allowed for a task. 
 * Ensures that we don't exceed the user's remaining credits or the task's maximum credits 
 * (factoring in credits already spent on this task).
 * 
 * @param userRemainingCredits The user's remaining credits.
 * @param taskMaxCredits The maximum credits defined for the task.
 * @param creditsSpent Credits already spent on this task.
 * @returns The maximum credits allowed for the task, as a BigInt.
 * 
 * @example
 * calculateMaxCredits(500000, 800000, 100000) // Returns 700000n
 * calculateMaxCredits("750000", 800000, "50000") // Returns 750000n
 * calculateMaxCredits(1500000n, "1000000", 200000) // Returns 800000n
 * calculateMaxCredits("2000000", undefined, 500000n) // Returns 1000000n (using DEFAULT_MAX_CREDITS)
 */
export function calculateMaxCredits(
    userRemainingCredits: CreditValue,
    taskMaxCredits: CreditValue,
    creditsSpent: CreditValue | undefined,
): bigint {
    // Convert all inputs to BigInt
    const userRemainingCreditsBigInt = BigInt(userRemainingCredits || 0);
    const creditsSpentBigInt = BigInt(creditsSpent || 0);
    const taskMaxCreditsBigInt = BigInt(taskMaxCredits || 0);

    // eslint-disable-next-line no-magic-numbers
    if (userRemainingCreditsBigInt <= 0n || taskMaxCreditsBigInt <= 0n || creditsSpentBigInt < 0n) {
        // eslint-disable-next-line no-magic-numbers
        return 0n;
    }

    // Calculate the effective task max credits (task max minus credits already spent on this task
    const effectiveTaskMax = taskMaxCreditsBigInt > creditsSpentBigInt
        ? BigInt(taskMaxCreditsBigInt) - BigInt(creditsSpentBigInt)
        : BigInt(0);

    // Return the smaller of the userRemaining credits and the effective task max
    return userRemainingCreditsBigInt < effectiveTaskMax
        ? userRemainingCreditsBigInt
        : effectiveTaskMax;
}

type GenerateResponseWithFallbackParams = Pick<CollectMessageContextInfoParams, "chatId" | "latestMessage" | "taskMessage"> & {
    /** 
    * Determines if context should be written to force the response to be a command 
    */
    force: boolean;
    /** Maximum number of credits that can be spent */
    maxCredits?: bigint;
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    /** If we should use a stream to show the response as its being generated */
    stream: boolean;
    task: LlmTask | `${LlmTask}` | undefined;
    userData: SessionUserToken;
}

const UNSAFE_CONTENT_CODE = "0605";

/** Limit chat responses to $0.50 for now */
// eslint-disable-next-line no-magic-numbers
export const DEFAULT_MAX_RESPONSE_CREDITS = BigInt(50) * API_CREDITS_MULTIPLIER;

/**
 * Attempts to generate a response using a preferred LLM service, falling back to 
 * other services if the preferred one is unavailable.
 */
export async function generateResponseWithFallback({
    chatId,
    force,
    latestMessage,
    maxCredits,
    mode,
    participantsData,
    respondingBotConfig,
    respondingBotId,
    stream,
    task,
    taskMessage,
    userData,
}: GenerateResponseWithFallbackParams): Promise<GenerateResponseResult> {
    const retryLimit = 3; // Set a limit to prevent infinite loops
    let attempts = 0;
    let accumulatedCost = 0;

    while (attempts < retryLimit) {
        attempts++;

        const serviceId = LlmServiceRegistry.get().getBestService(respondingBotConfig.model);
        if (!serviceId) {
            throw new CustomError("0252", "InternalError", userData.languages, { respondingBotConfig });
        }

        try {
            // Try using the service to generate a response
            const serviceInstance = LlmServiceRegistry.get().getService(serviceId) as LanguageModelService<string, string>;
            const model = serviceInstance.getModel(respondingBotConfig?.model);
            const languages = userData.languages;

            const contextInfo = await (new ChatContextCollector(serviceInstance)).collectMessageContextInfo({
                chatId,
                languages,
                latestMessage,
                model,
                taskMessage,
            });
            const { messages, systemMessage } = await serviceInstance.generateContext({
                contextInfo,
                force,
                mode,
                model,
                participantsData,
                respondingBotId,
                respondingBotConfig,
                task,
                taskMessage,
                userData,
            });

            // Check input safety
            const stringifiedInput = JSON.stringify(messages) + systemMessage;
            const { cost: safetyCheckCost, isSafe: isInputSafe } = await serviceInstance.safeInputCheck(stringifiedInput);
            accumulatedCost += safetyCheckCost;
            if (!isInputSafe) {
                // Delete the message
                try {
                    if (latestMessage) {
                        await cudHelper({
                            inputData: [{
                                action: "Delete",
                                input: latestMessage,
                                objectType: "ChatMessage",
                            }],
                            partialInfo: {},
                            userData,
                        });
                    }
                } catch (error) {
                    logger.error("Failed to delete unsafe message", { trace: "0607", chatId, respondingBotId, error });
                }
                const MAX_LOGGED_INPUT_LENGTH = 1000;
                throw new CustomError(UNSAFE_CONTENT_CODE, "UnsafeContent", userData.languages, {
                    input: stringifiedInput.length > MAX_LOGGED_INPUT_LENGTH ? stringifiedInput.slice(0, MAX_LOGGED_INPUT_LENGTH) + "..." : stringifiedInput,
                    latestMessage,
                });
            }

            // Calculate input tokens and determine max output tokens based on maxCredits
            const inputTokens = serviceInstance.estimateTokens({ text: stringifiedInput, model }).tokens;
            const maxOutputCredits = calculateMaxCredits(
                userData.credits,
                maxCredits || DEFAULT_MAX_RESPONSE_CREDITS,
                accumulatedCost,
            );
            const maxOutputTokens = serviceInstance.getMaxOutputTokensRestrained({
                maxCredits: maxOutputCredits,
                model,
                inputTokens,
            });
            if (maxOutputTokens !== null && maxOutputTokens <= 0) {
                throw new CustomError("0604", "CostLimitExceeded", userData.languages, { maxCredits, accumulatedCost });
            }

            let responseMessage = "";
            let finalCost = 0;
            // Stream the response if requested and we're in a chat
            if (chatId && stream) {
                const response = serviceInstance.generateResponseStreaming({
                    maxTokens: maxOutputTokens,
                    messages,
                    mode,
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
                    maxTokens: maxOutputTokens,
                    messages,
                    mode,
                    model,
                    systemMessage,
                    userData,
                });
                responseMessage = response.message;
                finalCost = response.cost;
                accumulatedCost += finalCost;
            }
            return { attempts, cost: finalCost, message: responseMessage };
        } catch (error) {
            // If the error is due to unsafe content, immediately throw it without retrying
            if (error instanceof CustomError && error.code === "UnsafeContent") {
                throw error;
            }
            const serviceState = LlmServiceRegistry.get().getServiceState(serviceId);
            // If the service is still active, then the error was likely due 
            // to the request we made. So we'll throw the error instead of retrying.
            if (serviceState === LlmServiceState.Active) {
                throw error;
            }
            // Otherwise, we'll retry with the next best service
        }
    }

    throw new CustomError("0253", "InternalError", userData.languages, { respondingBotConfig });
}

export type ForceGetTaskParams = Pick<CollectMessageContextInfoParams, "chatId" | "latestMessage" | "taskMessage"> & {
    commandToTask: CommandToTask,
    existingData?: ExistingTaskData | null,
    language: string,
    /**
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings,
    respondingBotId: string,
    task: LlmTask | `${LlmTask}`,
    userData: SessionUserToken,
}

/**
 * Repeatedly generates a bot response until it contains a valid task.
 * @returns The valid task and the rest of the message without the tasks
 */
export async function forceGetTask({
    chatId,
    commandToTask,
    existingData,
    language,
    latestMessage,
    mode,
    participantsData,
    respondingBotConfig,
    respondingBotId,
    task,
    taskMessage,
    userData,
}: ForceGetTaskParams): Promise<{
    taskToRun: ServerLlmTaskInfo | null,
    tasksToSuggest: ServerLlmTaskInfo[],
    messageWithoutTasks: string | null,
    cost: number
}> {
    const MAX_ATTEMPTS = 3; // Set a maximum number of LLM calls to avoid infinite loops and excessive costs
    let totalAttempts = 0;
    let totalCost = 0;

    while (totalAttempts < MAX_ATTEMPTS) {
        const { attempts, cost, message } = await generateResponseWithFallback({
            chatId,
            force: true, // Force the bot to respond with a task
            latestMessage,
            mode,
            participantsData,
            respondingBotConfig,
            respondingBotId,
            stream: false,
            task,
            taskMessage,
            userData,
        });
        totalAttempts += Math.max(attempts, 1); // Ensure we always increment by at least 1
        totalCost += Math.max(cost, 0); // Ensure we don't give negative costs

        // Check for tasks in the start response
        const { tasksToRun, tasksToSuggest, messageWithoutTasks } = await getValidTasksFromMessage({
            commandToTask,
            existingData: existingData ?? null,
            language,
            logger,
            message,
            mode,
            taskMode: task,
        });

        // If a valid task is found, return it
        if (tasksToRun.length > 0) {
            // Only use the first task found
            return { taskToRun: tasksToRun[0], tasksToSuggest, messageWithoutTasks, cost };
        } else {
            logger.warning(`No task found in start response. Retrying... (${totalAttempts}/${MAX_ATTEMPTS})`, { trace: "0349", chatId, respondingBotId, task });
        }
    }

    logger.error("Failed to find a task in start response after maximum retries.", { trace: "0350", chatId, respondingBotId, task });
    return { taskToRun: null, tasksToSuggest: [], messageWithoutTasks: null, cost: totalCost };
}

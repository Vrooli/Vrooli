import { BotSettings, BotSettingsTranslation, LlmTask, toBotSettings } from "@local/shared";
import { logger } from "../../events/logger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { SessionUserToken } from "../../types";
import { objectToYaml } from "../../utils";
import { bestTranslation } from "../../utils/bestTranslation";
import { withPrisma } from "../../utils/withPrisma";
import { getStructuredTaskConfig } from "./config";
import { MessageContextInfo } from "./context";
import { AnthropicService } from "./services/anthropic";
import { OpenAIService } from "./services/openai";

type SimpleChatMessageData = {
    id: string;
    translations: { language: string, text: string }[];
}

export type LanguageModelMessage = Partial<MessageContextInfo> & {
    role: "user" | "assistant";
    content: string;
}
export type LanguageModelContext = {
    messages: LanguageModelMessage[];
    systemMessage: string;
}

export type EstimateTokensParams = {
    text: string;
    requestedModel: string;
}
export type GetConfigObjectParams = {
    botSettings: BotSettings,
    includeInitMessage: boolean,
    userData: Pick<SessionUserToken, "languages">,
    task: LlmTask | `${LlmTask}`,
    force: boolean,
}
export type GenerateContextParams = {
    respondingBotId: string;
    respondingBotConfig: BotSettings;
    messageContextInfo: MessageContextInfo[];
    participantsData: Record<string, PreMapUserData>;
    task: LlmTask | `${LlmTask}`;
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    userData: SessionUserToken;
    requestedModel: string;
}
export type GenerateResponseParams = {
    chatId: string;
    /** 
    * Determines if context should be written to force the response to be a command 
    */
    force: boolean; participantsData: Record<string, PreMapUserData>;
    respondingBotConfig: BotSettings;
    respondingBotId: string;
    respondingToMessage: {
        id: string;
        text: string;
    } | null;
    task: LlmTask | `${LlmTask}`;
    userData: SessionUserToken;
}

export interface LanguageModelService<GenerateNameType extends string, TokenNameType extends string> {
    /** Estimate the amount of tokens a string is */
    estimateTokens(params: EstimateTokensParams): readonly [TokenNameType, number];
    /** Generates config for setting up bot persona and task instructions */
    getConfigObject(params: GetConfigObjectParams): Promise<Record<string, unknown>>;
    /** Generates context prompt from messages */
    generateContext(params: GenerateContextParams): Promise<LanguageModelContext>;
    /** Generate a message response */
    generateResponse(params: GenerateResponseParams): Promise<string>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** @returns the estimation method for the model */
    getEstimationMethod(requestedModel?: string | null): TokenNameType;
    /** @returns a list of token estimation types used by this service */
    getEstimationTypes(): readonly TokenNameType[];
    /** Convert a preferred model to an available one */
    getModel(requestedModel?: string | null): GenerateNameType;
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
    return ["default", tokens] as const;
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
    const translation = (bestTranslation(translationsList, userData.languages) ?? {}) as BotSettingsTranslation & { language?: string };

    const name: string | undefined = botSettings.name;
    const initMessage = translation.startingMessage?.length
        ? translation.startingMessage
        : name
            ? `Hello👋, I'm ${name}. How can I help you today?`
            : "Hello👋, how can I help you today?";
    delete translation.language;

    const taskConfig = await getStructuredTaskConfig(task, force, userData.languages[0] ?? "en");
    const configObject = {
        ai_assistant: {
            metadata: {
                name: botSettings.name ?? "Bot",
            },
            ...(includeInitMessage ? { init_message: initMessage } : {}),
            personality: { ...translation },
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
    respondingBotId,
    respondingBotConfig,
    messageContextInfo,
    participantsData,
    task,
    force,
    userData,
    requestedModel,
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
        includeInitMessage: messageContextInfo.length === 0,
        userData,
        task,
        force,
    });
    systemMessage += objectToYaml(config) + "\n";

    // If there are other users/bots in the chat
    const hasOtherParticipants = Object.keys(participantsData).length > 2;
    if (hasOtherParticipants) {
        // Add info about message labeling
        systemMessage += "In the conversation, each message will be preceded by a participant identifier in the format [Role: Name], where 'Role' is either 'User' or 'Bot', and 'Name' is the name of the user or bot. For example:\n";
        systemMessage += "- [User: John]: This is a message from a user named John.\n";
        systemMessage += "- [Bot: AssistantBot]: This is a message from a bot named AssistantBot.\n";
        systemMessage += "Please use these identifiers to understand the context of each message and generate appropriate responses.\n";

        // If there are other bots besides the one responding, add their configurations 
        // and instructions for how to prompt them
        const otherBots = Object.entries(participantsData).filter(([id, data]) => id !== respondingBotId && data.isBot);
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
    const systemMessageSize = service.estimateTokens({ text: systemMessage, requestedModel })[1];
    const maxContentSize = service.getContextSize(requestedModel);

    // Fetch messages from the database
    const messagesFromDB = await fetchMessagesFromDatabase(messageContextInfo.map(info => info.messageId));
    let currentTokenCount = systemMessageSize;

    // If the YAML configuration exceeds the context size, omit context entirely
    if (currentTokenCount > maxContentSize) {
        return { messages: [], systemMessage: "" };
    }

    // Convert every message into a role/content pair
    for (const contextInfo of messageContextInfo) {
        const messageData = messagesFromDB.find(message => message.id === contextInfo.messageId);
        if (!messageData) continue;

        const messageTranslation = messageData.translations.find(translation => translation.language === contextInfo.language);
        if (!messageTranslation) continue;

        const userName = contextInfo.userId ? participantsData[contextInfo.userId]?.name : undefined;
        const isBot = contextInfo.userId ? participantsData[contextInfo.userId]?.isBot : true;
        const role: LanguageModelMessage["role"] = isBot ? "assistant" : "user";

        const content = hasOtherParticipants ? `[${role === "assistant" ? "Bot" : "User"}: ${userName ?? "Unknown"}]: ${messageTranslation.text}` : messageTranslation.text;
        const tokenSize = contextInfo.tokenSize + service.estimateTokens({ text: content, requestedModel })[1];

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
    let messages: SimpleChatMessageData[] = [];

    await withPrisma({
        process: async (prisma) => {
            messages = await prisma.chat_message.findMany({
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
            });
        },
        trace: "0080",
    });

    return messages;
};

export const getLanguageModelService = (botSettings: BotSettings): LanguageModelService<string, string> => {
    const { model } = botSettings;
    if (!model) return new OpenAIService();
    if (model.includes("claude")) return new AnthropicService();
    return new OpenAIService();
};

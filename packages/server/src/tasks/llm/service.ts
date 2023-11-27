import OpenAI from "openai";
import { logger } from "../../events/logger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { SessionUserToken } from "../../types";
import { bestTranslation } from "../../utils/bestTranslation";
import { withPrisma } from "../../utils/withPrisma";
import { ChatContextCollector, MessageContextInfo } from "./context";

export type BotSettings = {
    model?: string;
    maxTokens?: number;
    name: string;
    translations?: Record<string, {
        bias?: string;
        creativity?: number;
        domainKnowledge?: string;
        keyPhrases?: string;
        occupation?: string;
        persona?: string;
        startingMessage?: string;
        tone?: string;
        verbosity?: number;
    }>
};

type SimpleChatMessageData = {
    id: string;
    translations: { language: string, text: string }[];
}

/**
 * Basic token estimation, which can be used as a placeholder 
 * until a more advanced one is implemented
 * @param text The text to estimate the amount of tokens for
 * @returns The name of the token estimation model, and the estimated amount of tokens
 */
export const tokenEstimationDefault = (text: string) => {
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
 * Fetches messages from the database for given message IDs.
 * 
 * @param messageIds Array of message IDs
 * @returns An array of message objects
 */
const fetchMessagesFromDatabase = async (messageIds: string[]): Promise<SimpleChatMessageData[]> => {
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

type LanguageModelMessage = Partial<MessageContextInfo> & {
    role?: string;
    text: string;
}
type LanguageModelContext = {
    messages: LanguageModelMessage[];
    systemMessage: string;
}

export interface LanguageModelService<GenerateNameType extends string, TokenNameType> {
    /** Estimate the amount of tokens a string is */
    estimateTokens(text: string, requestedModel?: string | null): readonly [TokenNameType, number];
    /** Generates context prompt from messages */
    generateContext(
        respondingBotId: string,
        respondingBotConfig: BotSettings,
        messageContextInfo: MessageContextInfo[],
        participantsData: Record<string, PreMapUserData>,
        userData: SessionUserToken,
        requestedModel?: string | null
    ): Promise<LanguageModelContext>;
    /** Generate a message response */
    generateResponse(
        chatId: string,
        respondingToMessageId: string,
        respondingToMessageContent: string,
        respondingBotId: string,
        respondingBotConfig: BotSettings,
        userData: SessionUserToken
    ): Promise<string>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** @returns a list of token estimation types used by this service */
    /** @returns the estimation method for the model */
    getEstimationMethod(requestedModel?: string | null): TokenNameType;
    getEstimationTypes(): readonly TokenNameType[];
    /** Convert a preferred model to an available one */
    getModel(requestedModel?: string | null): GenerateNameType;
}

type OpenAIGenerateModel = "gpt-3.5-turbo" | "gpt-4";
type OpenAITokenModel = "default";
export class OpenAIService implements LanguageModelService<OpenAIGenerateModel, OpenAITokenModel> {
    private openai: OpenAI;
    private defaultModel: OpenAIGenerateModel = "gpt-3.5-turbo";

    constructor() {
        this.openai = new OpenAI({
            apiKey: process.env.OPENAI_API_KEY,
        });
    }

    estimateTokens(text: string, _requestedModel?: string | null) {
        return tokenEstimationDefault(text);
    }

    private getConfigYaml(config: BotSettings, userData: SessionUserToken) {
        const translationsList = Object.entries(config?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
        const translation = bestTranslation(translationsList, userData.languages) ?? {};

        let result = `
        ai_assistant:
          metadata:
            name: ${config.name ?? "Bot"}
          personality:
        `;
        for (const [key, value] of Object.entries(translation)) {
            result += `    ${key}: ${value}
            `;
        }
        return result;
    }

    async generateContext(
        _respondingBotId: string,
        respondingBotConfig: BotSettings,
        messageContextInfo: MessageContextInfo[],
        participantsData: Record<string, PreMapUserData>,
        userData: SessionUserToken,
        requestedModel?: string | null,
    ): Promise<LanguageModelContext> {
        const messages: LanguageModelContext["messages"] = [];

        // Construct the initial YAML configuration message for relevant participants
        let systemMessage = "You are a helpful assistant for an app named Vrooli. Please follow the configuration below to best suit each user's needs:\n\n";
        // Add yml for bot responding
        systemMessage += this.getConfigYaml(respondingBotConfig, userData) + "\n";
        // We'll see if we need the other bot configs after testing
        // // Identify bots present in the message context, minus the one responding
        // const participantIdsInContext = new Set(messageContextInfo.filter(info => info.userId && info.userId !== respondingBotId).map(info => info.userId));
        // // Add yml for each bot in the context
        // if (participantIdsInContext.size > 0) {
        //     systemMessage += `There are other bots in this chat. Here are their configurations:\n\n`;
        // }

        // Calculate token size for the YAML configuration
        const systemMessageSize = this.estimateTokens(systemMessage, requestedModel)[1];
        const maxContentSize = this.getContextSize(requestedModel);

        // Fetch messages from the database
        const messagesFromDB = await fetchMessagesFromDatabase(messageContextInfo.map(info => info.messageId));
        let currentTokenCount = systemMessageSize;

        // Add the YAML configuration as the first message if it doesn't exceed the context size
        if (currentTokenCount <= maxContentSize) {
            messages.push({ role: "system", text: systemMessage });
        }
        // Otherwise, omit context entirely
        else {
            return { messages: [], systemMessage: "" };
        }

        for (const contextInfo of messageContextInfo) {
            const messageData = messagesFromDB.find(message => message.id === contextInfo.messageId);
            if (!messageData) continue;

            const messageTranslation = messageData.translations.find(translation => translation.language === contextInfo.language);
            if (!messageTranslation) continue;

            const userName = contextInfo.userId ? participantsData[contextInfo.userId]?.name : undefined;
            const tokenSize = contextInfo.tokenSize + (userName?.length ? (userName.length / 2) : 0); // For now, add a rough buffer for displaying the user's name

            // Stop if adding this message would exceed the context size
            if (currentTokenCount + tokenSize > maxContentSize) {
                break;
            }

            // Otherwise, increment the token count and add the message
            currentTokenCount += tokenSize;
            messages.push({ role: "user", text: `${userName ? `${userName}: ` : ""}${messageTranslation.text}` });
        }

        return { messages, systemMessage };
    }

    async generateResponse(
        chatId: string,
        respondingToMessageId: string,
        respondingToMessageContent: string,
        respondingBotId: string,
        respondingBotConfig: BotSettings,
        userData: SessionUserToken,
    ) {
        const model = this.getModel(respondingBotConfig?.model);
        const messageContextInfo = await (new ChatContextCollector(this)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessageId);
        const context = await this.generateContext(respondingBotId, respondingBotConfig, messageContextInfo, {}, userData);

        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: [
                ...(context.messages.map(({ role, text }) => ({ role: role ?? "assistant", content: text })) as { role: "user" | "assistant", content: string }[]),
                { role: "user", content: respondingToMessageContent },
            ],
            model,
            user: userData.name ?? undefined,
        };
        const chatCompletion: OpenAI.Chat.ChatCompletion = await this.openai.chat.completions
            .create(params)
            .catch((error) => {
                const message = "Failed to call OpenAI";
                logger.error(message, { trace: "0009", error });
                throw new Error(message);
            });
        return chatCompletion.choices[0].message.content ?? "";
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        switch (model) {
            case "gpt-3.5-turbo":
                return 4096;
            case "gpt-4":
                return 8192;
            default:
                return 4096;
        }
    }

    getEstimationMethod(_requestedModel?: string | null | undefined): "default" {
        return "default";
    }
    getEstimationTypes() {
        return ["default"] as const;
    }

    getModel(requestedModel?: string | null) {
        if (typeof requestedModel !== "string") return this.defaultModel;
        if (requestedModel.startsWith("gpt-4")) return "gpt-4";
        return this.defaultModel;
    }
}

export const getLanguageModelService = (botSettings: BotSettings): LanguageModelService<any, any> => {
    switch (botSettings.model) {
        default:
            return new OpenAIService();
    }
};
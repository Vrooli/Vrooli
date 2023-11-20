import { ChatMessage } from "@local/shared";
import OpenAI from "openai";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { modelToGql } from "../builders/modelToGql";
import { selectHelper } from "../builders/selectHelper";
import { chatMessage_findOne } from "../endpoints";
import { logger } from "../events/logger";
import { Trigger } from "../events/trigger";
import { io } from "../io";
import { PreMapMessageData, PreMapUserData } from "../models/base/chatMessage";
import { PrismaType, SessionUserToken } from "../types";
import { bestTranslation } from "./bestTranslation";
import { ChatContextCollector, MessageContextInfo } from "./llmContext";
import { withPrisma } from "./withPrisma";

type BotSettings = {
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

/**
 * Converts db bot info to BotSettings type
 */
const toBotSettings = (bot: { name: string, botSettings: string | undefined }): BotSettings => {
    let result: BotSettings = { name: bot.name };
    if (typeof bot.botSettings !== "string") return result;
    try {
        const botSettings = JSON.parse(bot.botSettings);
        if (typeof botSettings !== "object") return result;
        result = { ...botSettings, ...result };
        // eslint-disable-next-line no-empty
    } catch { }
    return result;
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

// TODO should query messages all at once if there are less than k in the chat, otherwise batch. Should select 
// fork data, fork's fork data, etc. a few times, as a way to locate relevant messages
// TODO to handle race conditions where multiple messages set the same forkId, we should batch both upwards (see the previous TODO), 
// and downwards. To query downwards, get all forkIds from upwards query and query for all messages with those forkIds. Then,
// combine upwards and downwards.
// TODO create cron job to fix messages with duplicate forkIds, and to fix messages with forkIds that don't exist
/**
 * Responds to a chat message, handling response generation and processing, 
 * websocket events, and any other logic
 */
export const respondToMessage = async (chatId: string, messageId: string, message: PreMapMessageData, respondingBotId: string, participantsData: Record<string, PreMapUserData>, prisma: PrismaType, userData: SessionUserToken): Promise<void> => {
    try {
        const bot = participantsData[respondingBotId];
        if (!bot) throw new Error("Bot data not found in participants data");
        const botSettings = toBotSettings(bot);
        const service = getLanguageModelService(botSettings);
        // Send typing message while bot is responding
        io.to(message.chatId as string).emit("typing", { starting: [respondingBotId] });
        const text = await service.generateResponse(chatId, messageId, message.content, respondingBotId, botSettings, userData);
        const select = selectHelper(chatMessage_findOne);
        const createdData = await prisma.chat_message.create({
            data: {
                chat: { connect: { id: message.chatId as string } },
                parent: { connect: { id: messageId } },
                user: { connect: { id: respondingBotId } },
                translations: {
                    create: {
                        language: message.language,
                        text,
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
    } catch (error) {
        logger.error("Error generating response or saving to database:", { trace: "0010", error });
    } finally {
        io.to(message.chatId as string).emit("typing", { stopping: [respondingBotId] });
    }
};

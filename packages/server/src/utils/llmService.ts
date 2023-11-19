import { ChatMessage } from "@local/shared";
import OpenAI from "openai";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { modelToGql } from "../builders/modelToGql";
import { selectHelper } from "../builders/selectHelper";
import { chatMessage_findOne } from "../endpoints";
import { logger } from "../events/logger";
import { Trigger } from "../events/trigger";
import { io } from "../io";
import { PreMapBotData, PreMapMessageData } from "../models/base/chatMessage";
import { PrismaType, SessionUserToken } from "../types";
import { bestTranslation } from "./bestTranslation";

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

type OpenAIGenerateModel = "gpt-3.5-turbo" | "gpt-4";
type OpenAITokenModel = "default";

export interface LanguageModelService<GenerateNameType extends string, TokenNameType> {
    /** Estimate the amount of tokens a string is */
    estimateTokens(text: string, requestedModel?: string | null): readonly [TokenNameType, number];
    /** Generate a message response */
    generateResponse(prompt: string, config: BotSettings, userData: SessionUserToken): Promise<string>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** @returns a list of token estimation types used by this service */
    getEstimationTypes(): readonly TokenNameType[];
    /** Convert a preferred model to an available one */
    getModel(requestedModel?: string | null): GenerateNameType;
}

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

    async generateResponse(prompt: string, config: BotSettings, userData: SessionUserToken) {
        const translationsList = Object.entries(config?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
        const translation = bestTranslation(translationsList, userData.languages) ?? {};

        let instructions = `You are a helpful assistant for an app named Vrooli. Please follow the configuration below to best suit the user's needs:
        
        ai_assistant:
          metadata:
            name: ${config.name ?? "Bot"}
          personality:
        `;
        for (const [key, value] of Object.entries(translation)) {
            instructions += `    ${key}: ${value}
            `;
        }

        const model = this.getModel(config?.model);
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: [{ role: "user", content: instructions }, { role: "user", content: prompt }],
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

//TODO need to pass all chat's botData to this function. Then if one of the bots is in the previous messages, we 
// can also include its persona in the system message.
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
export const respondToMessage = async (messageId: string, message: PreMapMessageData, bot: PreMapBotData, prisma: PrismaType, userData: SessionUserToken): Promise<void> => {
    try {
        const botSettings: BotSettings = typeof bot.botSettings === "string" ? JSON.parse(bot.botSettings) : {};
        const service = getLanguageModelService(botSettings);
        // Send typing message while bot is responding
        io.to(message.chatId as string).emit("typing", { starting: [bot.id] });
        const text = await service.generateResponse(message.content, { ...botSettings, name: bot.name }, userData);
        const select = selectHelper(chatMessage_findOne);
        const createdData = await prisma.chat_message.create({
            data: {
                chat: { connect: { id: message.chatId as string } },
                parent: { connect: { id: messageId } },
                user: { connect: { id: bot.id } },
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
            createdById: bot.id,
            hasCompleteAndPublic: true, // N/A
            hasParent: true, // N/A
            owner: { id: bot.id, __typename: "User" },
            objectId: fullMessageData.id,
            objectType: "ChatMessage",
        });
        await Trigger(prisma, [message.language]).chatMessageCreated({
            createdById: bot.id,
            data: message,
            message: fullMessageData,
        });
    } catch (error) {
        logger.error("Error generating response or saving to database:", { trace: "0010", error });
    } finally {
        io.to(message.chatId as string).emit("typing", { stopping: [bot.id] });
    }
};

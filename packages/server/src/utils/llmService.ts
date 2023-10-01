import { ChatMessage } from "@local/shared";
import OpenAI from "openai";
import { addSupplementalFields, modelToGql, selectHelper } from "../builders";
import { chatMessage_findOne } from "../endpoints";
import { logger, Trigger } from "../events";
import { io } from "../io";
import { PreMapBotData, PreMapMessageData } from "../models/base";
import { PrismaType, SessionUserToken } from "../types";

// Define an interface for a language model service
interface LanguageModelService<Response> {
    /** Generate a message response */
    generateResponse(prompt: string, config?: any): Promise<Response>;
    /** Convert a preferred model to an available one */
    getModel(requestedModel: string): string;
    /** Store and process a message response (i.e. insert into database and call triggers) */
    processResponse(response: Response, message: PreMapMessageData, bot: PreMapBotData, prisma: PrismaType, userData: SessionUserToken): Promise<void>;
}

export class OpenAIService implements LanguageModelService<string> {
    private openai: OpenAI;

    constructor() {
        this.openai = new OpenAI({
            apiKey: process.env.OPENAI_API_KEY,
        });
    }

    async generateResponse(prompt: string, config?: { model?: string, maxTokens?: number }): Promise<string> {
        const model = this.getModel(config?.model);
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: [{ role: "user", content: prompt }],
            model,
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

    getModel(requestedModel: string | null | undefined): string {
        const defaultModel = "gpt-3.5-turbo";
        if (typeof requestedModel !== "string") return defaultModel;
        if (requestedModel.startsWith("gpt-4")) return "gpt-4";
        return defaultModel;
    }

    async processResponse(response: string, message: PreMapMessageData, bot: PreMapBotData, prisma: PrismaType, userData: SessionUserToken): Promise<void> {
        const select = selectHelper(chatMessage_findOne);
        const createdData = await prisma.chat_message.create({
            data: {
                chat: { connect: { id: message.chatId as string } },
                isFork: false,
                user: { connect: { id: bot.id } },
                translations: {
                    create: {
                        language: message.language,
                        text: response,
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
    }
}

export const getLanguageModelService = (_botSettings: Record<string, unknown>): LanguageModelService<unknown> => {
    // For now, always return OpenAIService, but you could add logic to return different services based on botSettings
    return new OpenAIService();
};

/**
 * Responds to a chat message, handling response generation and processing, 
 * websocket events, and any other logic
 */
export const respondToMessage = async (message: PreMapMessageData, bot: PreMapBotData, prisma: PrismaType, userData: SessionUserToken): Promise<void> => {
    try {
        const botSettings = typeof bot.botSettings === "string" ? JSON.parse(bot.botSettings) : {};
        const service = getLanguageModelService(botSettings);
        // Send typing message while bot is responding
        io.to(message.chatId as string).emit("typing", { starting: [bot.id] });
        const responseText = await service.generateResponse(message.content, botSettings);
        await service.processResponse(responseText, message, bot, prisma, userData);

    } catch (error) {
        logger.error("Error generating response or saving to database:", { trace: "0010", error });
    } finally {
        io.to(message.chatId as string).emit("typing", { stopping: [bot.id] });
    }
};

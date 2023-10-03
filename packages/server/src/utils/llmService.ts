import { ChatMessage } from "@local/shared";
import OpenAI from "openai";
import { addSupplementalFields, modelToGql, selectHelper } from "../builders";
import { chatMessage_findOne } from "../endpoints";
import { logger, Trigger } from "../events";
import { io } from "../io";
import { PreMapBotData, PreMapMessageData } from "../models/base";
import { PrismaType, SessionUserToken } from "../types";

// Define an interface for a language model service
interface LanguageModelService {
    /** Generate a message response */
    generateResponse(prompt: string, config?: any): Promise<string>;
    /** Convert a preferred model to an available one */
    getModel(requestedModel: string): string;
}

export class OpenAIService implements LanguageModelService {
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
}

export const getLanguageModelService = (_botSettings: Record<string, unknown>): LanguageModelService => {
    // For now, always return OpenAIService, but you could add logic to return different services based on botSettings
    return new OpenAIService();
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
        const botSettings = typeof bot.botSettings === "string" ? JSON.parse(bot.botSettings) : {};
        const service = getLanguageModelService(botSettings);
        // Send typing message while bot is responding
        io.to(message.chatId as string).emit("typing", { starting: [bot.id] });
        const text = await service.generateResponse(message.content, botSettings);
        const select = selectHelper(chatMessage_findOne);
        const createdData = await prisma.chat_message.create({
            data: {
                chat: { connect: { id: message.chatId as string } },
                isFork: false,
                fork: { connect: { id: messageId } },
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

import { RedisClientType } from "redis";
import { logger } from "../events/logger";
import { PreMapMessageData } from "../models/base/chatMessage";
import { withRedis } from "../redisConn";
import { LanguageModelService, OpenAIService } from "./llmService";
import { withPrisma } from "./withPrisma";

type CachedChatMessage = {
    id: string;
    userId?: string;
    parentId?: string;
    translatedTokenCounts: string;
}
export type MessageContextInfo = {
    messageId: string;
    tokenSize: number;
    userId: string | null;
    language: string;
};

/**
 * ChatContextManager manages the writing and updating of chat message data in Redis.
 * This class is responsible for maintaining the integrity and structure of chat messages
 * within the cache, ensuring that new messages, edits, and branching are properly handled.
 */
export class ChatContextManager {
    private languageModelService: LanguageModelService<string, string>;

    constructor(languageModelService?: LanguageModelService<string, string>) {
        this.languageModelService = languageModelService ?? new OpenAIService();
    }

    async addMessage(chatId: string, message: PreMapMessageData): Promise<void> {
        await withRedis({
            process: async (redisClient: RedisClientType) => {
                const messageData: CachedChatMessage = {
                    id: message.id,
                    translatedTokenCounts: JSON.stringify(this.calculateTokenCounts(message.translations, ...this.languageModelService.getEstimationTypes())),
                };
                if (message.parentId) {
                    messageData.parentId = message.parentId;
                }
                if (message.userId) {
                    messageData.userId = message.userId;
                }

                await redisClient.hSet(`message:${message.id}`, messageData);
                await redisClient.zAdd(`chat:${chatId}`, { score: Date.now(), value: message.id });
                if (message.parentId) {
                    await redisClient.sAdd(`children:${message.parentId}`, message.id);
                }
            },
            trace: "0076",
        });
    }

    async editMessage(message: PreMapMessageData): Promise<void> {
        await withRedis({
            process: async (redisClient: RedisClientType) => {
                // Find existing message data
                const existingMessageData = await redisClient.hGetAll(`message:${message.id}`);
                let existingTranslations: Record<string, Record<string, number>> = {};
                try {
                    if (existingMessageData) {
                        existingTranslations = JSON.parse(existingMessageData.translatedTokenCounts);
                    }
                }
                // eslint-disable-next-line no-empty
                catch (error) { }
                const messageData: CachedChatMessage = {
                    id: message.id,
                    translatedTokenCounts: JSON.stringify({
                        ...existingTranslations,
                        ...this.calculateTokenCounts(message.translations, ...this.languageModelService.getEstimationTypes()),
                    }),
                };
                if (message.parentId && !existingMessageData.parentId) {
                    messageData.parentId = message.parentId;
                }
                if (message.userId && !existingMessageData.userId) {
                    messageData.userId = message.userId;
                }
                await redisClient.hSet(`message:${message.id}`, messageData);
                if (!existingMessageData.parentId && message.parentId) {
                    await redisClient.sAdd(`children:${message.parentId}`, message.id);
                }
            },
            trace: "0077",
        });
    }

    async deleteMessage(chatId: string, messageId: string): Promise<void> {
        await withRedis({
            process: async (redisClient: RedisClientType) => {
                const messageToDelete = await redisClient.hGetAll(`message:${messageId}`);
                const children = await redisClient.sMembers(`children:${messageId}`);

                for (const childId of children) {
                    await redisClient.hSet(`message:${childId}`, "parent", messageToDelete.parent);
                    if (messageToDelete.parent) {
                        await redisClient.sAdd(`children:${messageToDelete.parent}`, childId);
                    }
                }

                await redisClient.del([`message:${messageId}`, `children:${messageId}`]);
                await redisClient.zRem(`chat:${chatId}`, messageId);
            },
            trace: "0078",
        });
    }

    async deleteChat(chatId: string): Promise<void> {
        await withRedis({
            process: async (redisClient: RedisClientType) => {
                // Retrieve all message IDs associated with the chat
                const messageIds = await redisClient.zRange(`chat:${chatId}`, 0, -1);

                // Prepare keys for deletion (messages and their child references)
                const keysToDelete = messageIds.map(messageId => [`message:${messageId}`, `children:${messageId}`]).flat();

                // Delete all messages and their child references
                if (keysToDelete.length > 0) {
                    await redisClient.del(keysToDelete);
                }

                // Remove the sorted set for the chat
                await redisClient.del(`chat:${chatId}`);
            },
            trace: "0079",
        });
    }

    /**
     * Calculates token counts for each translation of a message for multiple estimation methods.
     * 
     * @param translations Array of message translations
     * @param estimationMethods Each token estimation method to use
     * @returns An object with languages as keys and token counts for each estimation method as values
     */
    public calculateTokenCounts(translations: { language: string, text: string }[], ...estimationMethods: string[]): Record<string, Record<string, number>> {
        const translatedTokenCounts = {};

        translations.forEach(translation => {
            if (!translatedTokenCounts[translation.language]) {
                translatedTokenCounts[translation.language] = {};
            }

            estimationMethods.forEach(method => {
                translatedTokenCounts[translation.language][method] = this.languageModelService.estimateTokens(translation.text, method)[1];
            });
        });

        return translatedTokenCounts;
    }
}

/**
 * ChatContextCollector manages chat message contexts for generating chatbot responses.
 * It uses Redis to retrieve chat messages and a LanguageModelService to handle token estimation.
 *
 * This class is essential for ensuring the chatbot's response generation is based on
 * an appropriate context window, considering the token limits of different language models.
 * It assumes a specific Redis schema with messages stored in hashes and ordered in sorted sets.
 */
export class ChatContextCollector {
    private languageModelService: LanguageModelService<string, string>;

    constructor(languageModelService: LanguageModelService<string, string>) {
        this.languageModelService = languageModelService;
    }

    async collectMessageContextInfo(chatId: string, model: string, languages: string[], latestMessageId?: string): Promise<MessageContextInfo[]> {
        const contextSize = this.languageModelService.getContextSize(model);
        let currentTokenCount = 0;
        const messageContextInfo: MessageContextInfo[] = [];

        await withRedis({
            process: async (redisClient: RedisClientType) => {
                let currentMessageId = latestMessageId ?? await this.getLatestMessageId(redisClient, chatId);

                while (currentMessageId && currentTokenCount < contextSize) {
                    const messageDetails = await this.getMessageDetails(redisClient, currentMessageId);
                    const estimationMethod = this.languageModelService.getEstimationMethod(model);
                    const [estimatedTokens, language] = await this.getTokenCountForLanguages(redisClient, currentMessageId, estimationMethod, languages);

                    if (estimatedTokens >= 0) {
                        messageContextInfo.push({
                            messageId: currentMessageId,
                            tokenSize: estimatedTokens,
                            userId: messageDetails.userId ?? null,
                            language,
                        });

                        currentTokenCount += estimatedTokens;
                        currentMessageId = messageDetails.parentId ?? null;
                    } else {
                        logger.warn("Failed to estimate tokens for message", { trace: "0075", messageDetails });
                        break; // Break the loop if token estimation fails
                    }
                }
            },
            trace: "0072",
        });

        return messageContextInfo;
    }

    private async getLatestMessageId(redisClient: RedisClientType, chatId: string): Promise<string | null> {
        // Retrieve the last element in the sorted set (the most recent message)
        const latestMessages = await redisClient.zRange(`chat:${chatId}`, -1, -1);
        return latestMessages.length > 0 ? latestMessages[0] : null;
    }

    private async getMessageDetails(redisClient: RedisClientType, messageId: string): Promise<CachedChatMessage> {
        let messageData = await redisClient.hGetAll(`message:${messageId}`);

        if (!messageData || Object.keys(messageData).length === 0) {
            // Query from Prisma if not found in Redis
            let success = false;
            await withPrisma({
                process: async (prisma) => {
                    const message = await prisma.chat_message.findUnique({
                        where: { id: messageId },
                        select: {
                            id: true,
                            parent: {
                                select: {
                                    id: true,
                                },
                            },
                            translations: {
                                select: {
                                    language: true,
                                    text: true,
                                },
                            },
                            user: {
                                select: {
                                    id: true,
                                },
                            },
                        },
                    });

                    if (message) {
                        messageData = {
                            id: message.id,
                            translatedTokenCounts: JSON.stringify((new ChatContextManager(this.languageModelService)).calculateTokenCounts(message.translations, ...this.languageModelService.getEstimationTypes())),
                        };
                        if (message.parent) {
                            messageData.parent = message.parent.id;
                        }
                        if (message.user) {
                            messageData.userId = message.user.id;
                        }

                        // Store the fetched data in Redis for future use
                        await redisClient.hSet(`message:${messageId}`, messageData);

                        success = true;
                    }
                },
                trace: "0073",
            });

            if (!success) {
                throw new Error(`Failed to fetch message details for ID: ${messageId}`);
            }
        }

        return {
            id: messageId,
            parentId: messageData.parentId,
            translatedTokenCounts: JSON.parse(messageData.translatedTokenCounts ?? "{}"),
            userId: messageData.userId,
        };
    }

    /**
     * Tries to find the token count for the given estimation method and preferred languages.
     * If not found, uses the first available language. If none exist, returns -1.
     * Calculates and updates the token count if necessary.
     *
     * @param redisClient Redis client instance
     * @param messageId The ID of the message
     * @param estimationMethod The token estimation method
     * @param languages User's preferred languages
     * @returns The token count for the message and the language used
     */
    private async getTokenCountForLanguages(redisClient: RedisClientType, messageId: string, estimationMethod: string, languages: string[]): Promise<[number, string]> {
        const messageData = await redisClient.hGetAll(`message:${messageId}`);
        let translatedTokenCounts = messageData ? JSON.parse(messageData.translatedTokenCounts ?? "{}") : {};

        // Check if token counts for preferred languages are available in cache
        for (const language of languages) {
            if (translatedTokenCounts[language] && translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        // If token counts are not in cache, query database and update cache
        if (Object.keys(translatedTokenCounts).length === 0) {
            let success = false;
            await withPrisma({
                process: async (prisma) => {
                    const message = await prisma.chat_message.findUnique({
                        where: { id: messageId },
                        include: { translations: true },
                    });
                    if (message?.translations) {
                        translatedTokenCounts = (new ChatContextManager(this.languageModelService)).calculateTokenCounts(message.translations, estimationMethod);
                        await redisClient.hSet(`message:${messageId}`, "translatedTokenCounts", JSON.stringify(translatedTokenCounts));
                        success = true;
                    }
                },
                trace: "0074",
            });

            if (!success) {
                return [-1, ""]; // Return -1 if no translations are found or on failure
            }
        }

        // After updating the cache, try again to find the token count
        for (const language of languages) {
            if (translatedTokenCounts[language] && translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        // Fallback to the first available language if preferred languages are not available
        for (const language in translatedTokenCounts) {
            if (translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        return [-1, ""]; // Return -1 if no valid token count is found
    }
}

import { RedisClientType } from "redis";
import { logger } from "../../events/logger";
import { PreMapMessageData } from "../../models/base/chatMessage";
import { withRedis } from "../../redisConn";
import { withPrisma } from "../../utils/withPrisma";
import { LanguageModelService } from "./service";
import { OpenAIService } from "./services/openai";

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

type TokenCounts = Record<string, Record<string, number>>;

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
                // Make sure the message is actually new
                if (!message.isNew) {
                    logger.error("Tried to add an message not marked as new", { trace: "0071", message });
                    return;
                }

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
                // Make sure the message is actually existing
                if (message.isNew) {
                    logger.error("Tried to edit a message marked as new", { trace: "0070", message });
                    return;
                }

                // Find existing message data
                let existingData = await redisClient.hGetAll(`message:${message.id}`);
                let shouldDeleteOldData = false;
                if (typeof existingData !== "object" || Object.keys(existingData).length === 0) {
                    logger.error("Failed to find existing message data", { trace: "0068", message });
                    existingData = {};
                    shouldDeleteOldData = true;
                }
                const existingMessageData = { ...existingData };
                let existingTranslations: Record<string, Record<string, number>> = {};
                try {
                    if (existingMessageData?.translatedTokenCounts) {
                        existingTranslations = JSON.parse(existingMessageData.translatedTokenCounts);
                    }
                } catch (error) {
                    logger.error("Failed to parse existing translations", { trace: "0069", error, existingMessageData });
                }
                const messageData: CachedChatMessage = {
                    id: message.id,
                    translatedTokenCounts: JSON.stringify({
                        ...existingTranslations,
                        ...this.calculateTokenCounts(message.translations, ...this.languageModelService.getEstimationTypes()),
                    }),
                };
                // hSet keeps existing fields if not provided, so we only need to update the fields that have changed
                if (message.parentId && message.parentId !== existingMessageData.parentId) {
                    messageData.parentId = message.parentId;
                }
                if (message.userId && message.userId !== existingMessageData.userId) {
                    messageData.userId = message.userId;
                }
                // Delete invalid data if necessary and update the message data
                if (shouldDeleteOldData) {
                    await redisClient.del(`message:${message.id}`);
                }
                await redisClient.hSet(`message:${message.id}`, messageData);
                // Check if parentId has changed and handle accordingly
                if (messageData.parentId) {
                    if (existingMessageData.parentId) {
                        await redisClient.sRem(`children:${existingMessageData.parentId}`, messageData.id); // Remove from old parent's children set
                    }
                    await redisClient.sAdd(`children:${messageData.parentId}`, messageData.id); // Add to new parent's children set
                }
            },
            trace: "0077",
        });
    }

    async deleteMessage(chatId: string, messageId: string): Promise<void> {
        await withRedis({
            process: async (redisClient: RedisClientType) => {
                // Find existing data
                let messageToDelete = await redisClient.hGetAll(`message:${messageId}`);
                if (typeof messageToDelete !== "object" || Object.keys(messageToDelete).length === 0) {
                    logger.error("Failed to find message data to delete", { trace: "0066", messageId });
                    messageToDelete = {};
                }
                // Find children IDs of the message to delete
                let children = await redisClient.sMembers(`children:${messageId}`);
                if (!Array.isArray(children)) {
                    logger.error("Failed to find children of message to delete", { trace: "0065", messageId });
                    children = [];
                }

                // Update parent reference for each child message
                for (const childId of children) {
                    // Update parent reference to the parent of the message being deleted
                    await redisClient.hSet(`message:${childId}`, "parentId", messageToDelete.parentId);

                    // If the parent exists, add the child to its children set
                    if (messageToDelete.parentId) {
                        await redisClient.sAdd(`children:${messageToDelete.parentId}`, childId);
                    }
                }

                // Delete the message and its reference in the children set
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
    public calculateTokenCounts(translations: { language: string, text: string }[], ...estimationMethods: string[]): TokenCounts {
        const translatedTokenCounts = {};

        translations.forEach(translation => {
            if (!translatedTokenCounts[translation.language]) {
                translatedTokenCounts[translation.language] = {};
            }

            estimationMethods.forEach(method => {
                // Check if the method is valid; if not, fall back to a default method
                const validMethod = method ?? "default";

                // Ensure we only proceed if there's a method to use
                if (validMethod) {
                    // Safely handle the case where estimateTokens might return undefined
                    const estimate = this.languageModelService.estimateTokens(translation.text, validMethod);
                    const tokenCount = estimate ? estimate[1] : 0; // Fallback to 0 if estimate is undefined

                    translatedTokenCounts[translation.language][validMethod] = tokenCount;
                }
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
                        currentTokenCount += estimatedTokens;
                        if (currentTokenCount > contextSize) {
                            break; // Break the loop if the context size is exceeded
                        }

                        messageContextInfo.push({
                            messageId: currentMessageId,
                            tokenSize: estimatedTokens,
                            userId: messageDetails.userId ?? null,
                            language,
                        });

                        currentMessageId = messageDetails.parentId ?? null;
                    } else {
                        logger.warning("Failed to estimate tokens for message", { trace: "0075", messageDetails });
                        break; // Break the loop if token estimation fails
                    }
                }
            },
            trace: "0072",
        });

        // Reverse the array to get the messages in chronological order
        messageContextInfo.reverse();
        return messageContextInfo;
    }

    async getLatestMessageId(redisClient: RedisClientType, chatId: string): Promise<string | null> {
        // Retrieve the last element in the sorted set (the most recent message)
        const latestMessages = await redisClient.zRange(`chat:${chatId}`, -1, -1);
        return latestMessages.length > 0 ? latestMessages[0] : null;
    }

    async getMessageDetails(redisClient: RedisClientType, messageId: string): Promise<CachedChatMessage> {
        let messageData: CachedChatMessage = await redisClient.hGetAll(`message:${messageId}`) as CachedChatMessage;

        if (!messageData || typeof messageData !== "object" || Object.keys(messageData).length === 0) {
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
                            messageData.parentId = message.parent.id;
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
    async getTokenCountForLanguages(redisClient: RedisClientType, messageId: string, estimationMethod: string, languages: string[]): Promise<[number, string]> {
        const messageData = await redisClient.hGetAll(`message:${messageId}`);
        let translatedTokenCounts = messageData ? JSON.parse(messageData.translatedTokenCounts ?? "{}") : {};
        const languagesWithDefault = languages.length === 0 ? ["en"] : languages;

        // Check if token counts for preferred languages are available in cache
        for (const language of languagesWithDefault) {
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
        for (const language of languagesWithDefault) {
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

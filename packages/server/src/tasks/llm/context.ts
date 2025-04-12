import { DAYS_1_S, DEFAULT_LANGUAGE, TaskContextInfo } from "@local/shared";
import { RedisClientType } from "redis";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { withRedis } from "../../redisConn.js";
import { UI_URL } from "../../server.js";
import { PreMapMessageDataCreate, PreMapMessageDataDelete, PreMapMessageDataUpdate, PreMapUserData } from "../../utils/chat.js";
import { LlmServiceRegistry } from "./registry.js";
import { type LanguageModelService } from "./types.js";

type CachedChatMessage = {
    id: string;
    userId?: string;
    parentId?: string;
    translatedTokenCounts: string;
}
type ContextInfoBase = {
    language: string;
    tokenSize: number;
    userId: string | null;
}
export type MessageContextInfo = ContextInfoBase & {
    __type: "message";
    messageId: string;
}
export type TextContextInfo = ContextInfoBase & {
    __type: "text";
    text: string;
}
export type ContextInfo = MessageContextInfo | TextContextInfo;

/**
 * Calculated token counts for a message, grouped by language and estimation method/encoding pair
 */
type StoredTokenCounts = { [language: string]: { [estimationMethodEncodingPair: string]: number } };

export type CollectMessageContextInfoParams = {
    /** The chat ID where the message was sent */
    chatId?: string | null;
    /** 
     * The message ID to start fetching context from. 
     * The context info will be all messages leading up and including this message, 
     * which fits within the token limits of the language model. 
     * 
     * NOTE 1: If you include `taskMessage`, the context info will also include this message.
     */
    latestMessage?: string | null;
    /**
     * An additional task message to include in the context info, if needed. 
     * This is useful for things like autoFill and running routines, among other things.
     */
    taskMessage?: string | null;
};

/**
 * Handles in-memory caching for chat contexts.
 */
export class ChatContextCache {
    private static instance: ChatContextCache;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static get(): ChatContextCache {
        if (!this.instance) {
            this.instance = new ChatContextCache();
        }
        return this.instance;
    }

    /**
     * Constructs a key for a message in Redis.
     * 
     * @param messageId The ID of the message
     * @returns A key string.
     */
    public messageKey(messageId: string): string {
        return `message:${messageId}`;
    }

    /**
     * Constructs a key for the children of a message in Redis.
     * 
     * @param parentId The ID of the parent message
     * @returns A key string.
     */
    public childrenKey(parentId: string): string {
        return `children:${parentId}`;
    }

    /**     
     * Constructs a key for a chat in Redis.
     * 
     * @param chatId The ID of the chat
     * @returns A key string.
     */
    public chatKey(chatId: string): string {
        return `chat:${chatId}`;
    }

    /**
     * Constructs a key for a bot in Redis.
     * 
     * @param botId The ID of the bot
     * @returns A key string.
     */
    public botKey(botId: string): string {
        return `bot:${botId}`;
    }
    /**
     * Deletes a list of keys from Redis.
     * 
     * @param redisClient The Redis client
     * @param keys The keys to delete
     */
    public async deleteKeys(redisClient: RedisClientType | null, keys: string[]): Promise<void> {
        if (!redisClient) return;
        await redisClient.del(keys);
    }

    /**
     * Gets a message from the cache.
     * 
     * @param redisClient The Redis client
     * @param messageId The ID of the message
     * @returns The message data, or null if not found
     */
    public async getMessage(redisClient: RedisClientType | null, messageId: string): Promise<CachedChatMessage | null> {
        if (!redisClient) return null;
        const key = this.messageKey(messageId);
        const data = await redisClient.hGetAll(key);
        if (!data) {
            return null;
        }
        // Delete malformed data
        if (typeof data !== "object" || Object.keys(data).length === 0) {
            await redisClient.del(key);
            return null;
        }
        return data as CachedChatMessage;
    }

    /**
     * Sets a message in the cache.
     * 
     * @param redisClient The Redis client
     * @param messageId The ID of the message
     * @param messageData The message data to set
     */
    public async setMessage(redisClient: RedisClientType | null, messageId: string, messageData: CachedChatMessage): Promise<void> {
        if (!redisClient) return;
        const key = this.messageKey(messageId);
        await redisClient.hSet(key, messageData);
    }

    /**
     * Sets a property on a message in the cache.
     * 
     * @param redisClient The Redis client
     * @param messageId The ID of the message
     * @param property The property to set
     * @param value The value to set the property to
     */
    public async setMessageProperty(redisClient: RedisClientType | null, messageId: string, property: string, value: string): Promise<void> {
        if (!redisClient) return;
        const key = this.messageKey(messageId);
        await redisClient.hSet(key, property, value);
    }

    /**
     * Adds a message to a sorted set.
     * This is used to get the latest messages in a chat.
     * 
     * NOTE: It assumes that the message is the newest message in the chat. 
     * Do not use this to add messages that were already in the database!
     * 
     * @param redisClient The Redis client
     * @param chatId The ID of the chat
     * @param messageId The ID of the message
     */
    public async addMessageToChat(redisClient: RedisClientType | null, chatId: string, messageId: string): Promise<void> {
        if (!redisClient) return;
        const key = this.chatKey(chatId);
        await redisClient.zAdd(key, { score: Date.now(), value: messageId });
    }

    /**
     * Removes a message from the chat.
     * 
     * @param redisClient The Redis client
     * @param chatId The ID of the chat
     * @param messageId The ID of the message
     */
    public async removeMessageFromChat(redisClient: RedisClientType | null, chatId: string, messageId: string): Promise<void> {
        if (!redisClient) return;
        const key = this.chatKey(chatId);
        await redisClient.zRem(key, messageId);
    }

    /**
     * Gets all message IDs in a chat.
     * 
     * @param redisClient The Redis client
     * @param chatId The ID of the chat
     * @returns The message IDs in the chat
     */
    public async getAllMessagesInChat(redisClient: RedisClientType | null, chatId: string): Promise<string[]> {
        if (!redisClient) return [];
        const key = this.chatKey(chatId);
        return redisClient.zRange(key, 0, -1);
    }

    /**
     * Gets the latest message ID in a chat.
     * 
     * @param redisClient The Redis client
     * @param chatId The ID of the chat
     * @returns The latest message ID in the chat, or null if the chat is empty
     */
    public async getLatestMessage(redisClient: RedisClientType | null, chatId: string): Promise<string | null> {
        if (!redisClient) return null;
        // Retrieve the last element in the sorted set (the most recent message)
        const key = this.chatKey(chatId);
        const latestMessages = await redisClient.zRange(key, -1, -1);
        return latestMessages.length > 0 ? latestMessages[0] : null;
    }

    /**
     * Adds a child message to the set of children for a given parent message.
     * This is used to construct a conversation tree where messages can branch off a parent.
     *
     * @param redisClient The Redis client.
     * @param parentId The ID of the parent message.
     * @param messageId The ID of the child message.
     */
    public async addChildMessage(
        redisClient: RedisClientType | null,
        parentId: string | null,
        messageId: string,
    ): Promise<void> {
        if (!redisClient || !parentId) return;
        const key = this.childrenKey(parentId);
        await redisClient.sAdd(key, messageId);
    }

    /**
     * Removes a child message from the set of children for a given parent message. 
     * 
     * @param redisClient The Redis client.
     * @param parentId The ID of the parent message.
     * @param messageId The ID of the child message.
     */
    public async removeChildMessage(
        redisClient: RedisClientType | null,
        parentId: string | null | undefined,
        messageId: string,
    ): Promise<void> {
        if (!redisClient || !parentId) return;
        const key = this.childrenKey(parentId);
        await redisClient.sRem(key, messageId);
    }

    /**
     * Finds all children of a message.
     * 
     * @param redisClient The Redis client.
     * @param messageId The ID of the message.
     * @returns The IDs of the children of the message.
     */
    public async getChildren(redisClient: RedisClientType | null, messageId: string): Promise<string[]> {
        if (!redisClient) return [];
        const key = this.childrenKey(messageId);
        const data = await redisClient.sMembers(key);
        if (!Array.isArray(data)) {
            return [];
        }
        return data;
    }
}

export class ChatContextDb {
    private static instance: ChatContextDb;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static get(): ChatContextDb {
        if (!this.instance) {
            this.instance = new ChatContextDb();
        }
        return this.instance;
    }

    public async getMessage(messageId: string) {
        const message = await DbProvider.get().chat_message.findUnique({
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
        return message;
    }
}

export class TokenCountManager {
    private static instance: TokenCountManager;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static get(): TokenCountManager {
        if (!this.instance) {
            this.instance = new TokenCountManager();
        }
        return this.instance;
    }

    /**
     * Safely attempts to get the token counts for a message.
     * 
     * @param messageCache The cached message data
     * @returns The token counts for the message, or null if not found
     */
    public getTokenCountsFromMessageCache(messageCache: CachedChatMessage | null): StoredTokenCounts {
        if (!messageCache) return {};
        try {
            return JSON.parse(messageCache.translatedTokenCounts);
        } catch (error) {
            logger.error("Failed to parse existing translations", { trace: "0069", error, messageCache });
            return {};
        }
    }

    /**
     * Determines the key to use for the token count's language object.
     * 
     * @param modelService The language model service
     * @param aiModel The AI model to use
     * @returns The key to use for the token count's language object
     */
    public getEstimationKey(modelService: LanguageModelService<string>, aiModel: string): string {
        const { estimationModel, encoding } = modelService.getEstimationInfo(aiModel);
        return `${estimationModel}-${encoding}`;
    }

    /**
     * Creates or updates `StoredTokenCounts` for a message.
     * 
     * @param messageObject The message to calculate token counts for
     * @param messageCache The existing cached information for the message
     * @param userLanguages The languages the user speaks
     * @param languageModelService The language model service
     * @param aiModel The AI model to use
     * @returns The calculated token counts
     */
    public calculateTokenCounts(
        messageObject: { translations?: readonly { language: string, text: string }[] | undefined },
        messageCache: CachedChatMessage | null,
        userLanguages: string[],
        languageModelService: LanguageModelService<string>,
        aiModel: string,
    ): StoredTokenCounts {
        const { translations } = messageObject;

        // Start with the existing token counts
        const result = this.getTokenCountsFromMessageCache(messageCache);

        if (!translations || !Array.isArray(translations) || translations.length === 0) {
            return result;
        }

        // Loop through the translations
        for (const translation of translations) {
            // Initialize the object for the language if it doesn't exist
            if (!result[translation.language]) {
                result[translation.language] = {};
            }

            // Don't bother translating if the user doesn't speak the language
            if (!userLanguages.includes(translation.language)) {
                continue;
            }

            // Calculate the token counts for the translation if they don't exist
            const estimationKey = this.getEstimationKey(languageModelService, aiModel);
            if (!result[translation.language][estimationKey]) {
                const tokenEstimation = languageModelService.estimateTokens({
                    aiModel,
                    text: translation.text,
                });
                result[translation.language][estimationKey] = tokenEstimation.tokens;
            }
        }

        // Return the calculated token counts
        return result;
    }
}

/**
 * ChatContextManager manages:
 *  - the writing and updating of chat message data in Redis.
 *  - the reading of chat message data from the database.
 *  - the calculation of token counts for chat messages.
 *  - the collection of chat messages for context.
 * 
 * This class is essential for ensuring the chatbot's response generation is based on
 * an appropriate context window, considering the token limits of different language models.
 */
export class ChatContextManager {
    private aiModel: string;
    private userLanguages: string[];
    private languageModelService: LanguageModelService<string>;

    constructor(aiModel: string, userLanguages: string[]) {
        this.aiModel = aiModel;
        this.userLanguages = userLanguages;
        this.languageModelService = LlmServiceRegistry.get().getService(LlmServiceRegistry.get().getServiceId(aiModel));
    }

    async addMessage(data: PreMapMessageDataCreate): Promise<void> {
        const { chatId, messageId, parentId, userId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;

                const tokenCounts = TokenCountManager.get().calculateTokenCounts(
                    data,
                    null,
                    this.userLanguages,
                    this.languageModelService,
                    this.aiModel,
                );
                const messageData: CachedChatMessage = {
                    id: messageId,
                    translatedTokenCounts: JSON.stringify(tokenCounts),
                };
                if (parentId) {
                    messageData.parentId = parentId;
                }
                if (userId) {
                    messageData.userId = userId;
                }

                await ChatContextCache.get().setMessage(redisClient, messageId, messageData);
                await ChatContextCache.get().addMessageToChat(redisClient, chatId, messageId);
                if (parentId) {
                    await ChatContextCache.get().addChildMessage(redisClient, parentId, messageId);
                }
            },
            trace: "0076",
        });
    }

    async editMessage(data: PreMapMessageDataUpdate): Promise<void> {
        const { messageId, parentId, userId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;

                // Find existing cache data
                const existingData = await ChatContextCache.get().getMessage(redisClient, messageId);
                if (existingData) {
                    logger.error("Failed to find existing message data", { trace: "0068", messageId });
                }
                // Create new cache data
                const tokenCounts = TokenCountManager.get().calculateTokenCounts(
                    data,
                    null, // We'll pass in null in case the message text has changed (which would invalidate the existing token counts)
                    this.userLanguages,
                    this.languageModelService,
                    this.aiModel,
                );
                const messageData: CachedChatMessage = {
                    id: messageId,
                    translatedTokenCounts: JSON.stringify(tokenCounts),
                };
                // hSet keeps existing fields if not provided, so we only need to update the fields that have changed
                if (parentId && parentId !== existingData?.parentId) {
                    messageData.parentId = parentId;
                }
                if (userId && userId !== existingData?.userId) {
                    messageData.userId = userId;
                }
                // Add/Update the message data
                await ChatContextCache.get().setMessage(redisClient, messageId, messageData);
                // Check if parentId has changed and handle accordingly
                if (messageData.parentId) {
                    await ChatContextCache.get().removeChildMessage(redisClient, existingData?.parentId, messageId);
                    await ChatContextCache.get().addChildMessage(redisClient, messageData.parentId, messageData.id);
                }
            },
            trace: "0077",
        });
    }

    async deleteMessage(data: PreMapMessageDataDelete): Promise<void> {
        const { chatId, messageId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Find existing data
                const messageToDelete = await ChatContextCache.get().getMessage(redisClient, messageId);
                if (!messageToDelete) {
                    logger.error("Failed to find message data to delete", { trace: "0066", messageId });
                }
                // Find children IDs of the message to delete
                const children = await ChatContextCache.get().getChildren(redisClient, messageId);

                // Update parent reference for each child message
                const newParentId = messageToDelete?.parentId;
                if (newParentId) {
                    for (const childMessageId of children) {
                        await ChatContextCache.get().setMessageProperty(redisClient, childMessageId, "parentId", newParentId);
                        if (newParentId) {
                            await ChatContextCache.get().addChildMessage(redisClient, newParentId, childMessageId);
                        }
                    }
                }

                // Delete the message and its children set
                await ChatContextCache.get().deleteKeys(redisClient, [ChatContextCache.get().messageKey(messageId), ChatContextCache.get().childrenKey(messageId), ChatContextCache.get().childrenKey(chatId)]);
                // Remove the message from the chat
                await ChatContextCache.get().removeMessageFromChat(redisClient, chatId, messageId);
            },
            trace: "0078",
        });
    }

    async deleteChat(chatId: string): Promise<void> {
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Retrieve all message IDs associated with the chat
                const messageIds = await ChatContextCache.get().getAllMessagesInChat(redisClient, chatId);

                // Prepare keys for deletion (messages and their child references)
                const keysToDelete = messageIds.map(messageId => [ChatContextCache.get().messageKey(messageId), ChatContextCache.get().childrenKey(messageId)]).flat();

                // Delete all messages and their child references
                if (keysToDelete.length > 0) {
                    await ChatContextCache.get().deleteKeys(redisClient, keysToDelete);
                }

                // Finally, remove the chat
                await ChatContextCache.get().deleteKeys(redisClient, [ChatContextCache.get().chatKey(chatId)]);
            },
            trace: "0079",
        });
    }

    /**
     * Collects messages to include in the context info for generating a response. 
     * Messages are collected starting from the latest message and going back, 
     * until the context size is reached or no more messages are found.
     * 
     * NOTE: This also adds the taskMessage to the context info if provided, which 
     * basically simulates a message in the chat history to tell the bot how to respond.
     */
    async collectMessageContextInfo({
        chatId,
        latestMessage,
        taskMessage,
    }: CollectMessageContextInfoParams): Promise<ContextInfo[]> {
        const contextSize = this.languageModelService.getContextSize(this.aiModel);
        let currentTokenCount = 0;
        const contextInfo: ContextInfo[] = [];
        const language = this.userLanguages.length ? this.userLanguages[0] : DEFAULT_LANGUAGE;

        // Add task message to context info if provided
        if (taskMessage) {
            const estimation = this.languageModelService.estimateTokens({
                aiModel: this.aiModel,
                text: taskMessage,
            });

            if (estimation.tokens >= 0 && estimation.tokens <= contextSize) {
                contextInfo.push({
                    __type: "text",
                    tokenSize: estimation.tokens,
                    userId: null,
                    language,
                    text: taskMessage,
                });
            } else {
                logger.warning("Failed to estimate tokens for provided text", { trace: "0075", text: taskMessage });
            }
        }

        // Collect chat messages for context info, starting at `latestMessage` and going back
        if (chatId) {
            await withRedis({
                process: async (redisClient) => {
                    // Get provided message ID, or fetch if not provided
                    let currentMessageId: string | null = latestMessage ?? await ChatContextCache.get().getLatestMessage(redisClient, chatId);

                    // If message ID is found, loop up message tree to collect context, 
                    // until context size is reached or no more messages are found
                    while (currentMessageId && currentTokenCount < contextSize) {
                        const { messageDetails, tokenSize } = await this.getMessageDetails(redisClient, currentMessageId);
                        if (!messageDetails) {
                            logger.warning("Failed to find message details", { trace: "0080", currentMessageId });
                            break; // Break the loop if message details are not found
                        }

                        if (tokenSize >= 0) {
                            currentTokenCount += tokenSize;
                            if (currentTokenCount > contextSize) {
                                break; // Break the loop if the context size is exceeded
                            }

                            contextInfo.push({
                                __type: "message",
                                messageId: currentMessageId,
                                tokenSize,
                                userId: messageDetails.userId ?? null,
                                language,
                            });

                            currentMessageId = messageDetails.parentId ?? null;
                        } else {
                            logger.warning("Failed to estimate tokens for message", { trace: "0083", messageDetails });
                            break; // Break the loop if token estimation fails
                        }
                    }
                },
                trace: "0072",
            });
        }

        // Reverse the array to get the messages in chronological order
        contextInfo.reverse();

        // If there are no messages, log warning. This may indicate an problem, 
        // and may break some llm services
        if (contextInfo.length === 0) {
            logger.warning("No messages found in context. This may cause an error with response generation", { trace: "0239" });
        }

        return contextInfo;
    }

    /**
     * Gets the details of a message from Redis, or queries the database if not found.
     * 
     * @param redisClient The Redis client instance
     * @param messageId The ID of the message to get details for
     * @returns The details of the message, and how many tokens it is
     */
    async getMessageDetails(redisClient: RedisClientType | null, messageId: string): Promise<{ messageDetails: CachedChatMessage | null, tokenSize: number }> {
        // Try to get message details from in-memory cache first
        const messageCache = await ChatContextCache.get().getMessage(redisClient, messageId);
        let tokenSize = 0;

        // Check if the message data includes a token count for the current language and estimation method
        const currentLanguage = this.userLanguages.length ? this.userLanguages[0] : DEFAULT_LANGUAGE;
        const tokenCounts = TokenCountManager.get().getTokenCountsFromMessageCache(messageCache);
        if (!tokenCounts[currentLanguage] || typeof tokenCounts[currentLanguage] !== "object") {
            tokenCounts[currentLanguage] = {};
        }
        const estimationKey = TokenCountManager.get().getEstimationKey(this.languageModelService, this.aiModel);
        // If the token count is not found, calculate it
        if (tokenCounts[currentLanguage][estimationKey] === undefined) {
            const messageData = await ChatContextDb.get().getMessage(messageId);
            const translation = messageData?.translations?.find((t) => t.language === currentLanguage);
            if (translation) {
                const tokenEstimation = this.languageModelService.estimateTokens({
                    aiModel: this.aiModel,
                    text: translation.text,
                });
                tokenCounts[currentLanguage][estimationKey] = tokenEstimation.tokens;
                tokenSize = tokenEstimation.tokens;
            }
            // Update the cache with the new token counts and any other message data it might be missing
            const updatedMessageCache: CachedChatMessage = {
                ...messageCache,
                translatedTokenCounts: JSON.stringify(tokenCounts),
                id: messageId,
                userId: messageData?.user?.id ?? undefined,
                parentId: messageData?.parent?.id ?? undefined,
            };
            await ChatContextCache.get().setMessage(redisClient, messageId, updatedMessageCache);
            return { messageDetails: updatedMessageCache, tokenSize };
        } else {
            tokenSize = tokenCounts[currentLanguage][estimationKey];
            return { messageDetails: messageCache, tokenSize };
        }
    }
}

/**
 * Finds bot information for a given bot ID. 
 * First checks the redis cache, then falls back to the database.
 */
export async function getBotInfo(botId: string): Promise<PreMapUserData | null> {
    let botInfo: PreMapUserData | null = null;

    // Check Redis cache first
    await withRedis({
        process: async (redisClient) => {
            if (!redisClient) return;
            const key = ChatContextCache.get().botKey(botId);
            const rawData = await redisClient.hGetAll(key);
            if (rawData && Object.keys(rawData).length >= 4) {
                // Convert isBot back to boolean
                botInfo = {
                    ...rawData,
                    isBot: rawData.isBot === "true",
                } as unknown as PreMapUserData;
            }
        },
        trace: "0233",
    });

    // If not found in cache, query the database
    if (!botInfo || Object.keys(botInfo).length < 4) { // There are 4 fields in PreMapUserData. Can add better check if needed
        botInfo = await DbProvider.get().user.findUnique({
            where: { id: botId },
            select: {
                id: true,
                name: true,
                isBot: true,
                botSettings: true,
            },
        }) as PreMapUserData;
        // Store the fetched data in Redis for future use
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Convert isBot to string, since Redis only accepts string values
                const botInfoForRedis = {
                    ...botInfo,
                    isBot: botInfo!.isBot.toString(),
                };
                const key = ChatContextCache.get().botKey(botId);
                await redisClient.hSet(key, botInfoForRedis);
                await redisClient.expire(key, DAYS_1_S);
            },
            trace: "0235",
        });
    }

    return botInfo;
}

/**
 * @returns Valid mentions in a message
 */
export function processMentions(
    messageContent: string,
    chat: { botParticipants?: string[] },
    bots: Pick<PreMapUserData, "id" | "name">[],
): string[] {
    // Find markdown links in the message
    const linkStrings = messageContent.match(/\[([^\]]+)\]\(([^)]+)\)/g);
    // Get the label and link for each link
    let links: { label: string, link: string }[] = linkStrings?.map(s => {
        const [label, link] = s.slice(1, -1).split("](");
        return { label, link };
    }) ?? [];

    // Filter out links where the that aren't a mention. Rules:
    // 1. Label must start with @
    // 2. Link must be to this site
    links = links.filter(l => {
        if (!l.label.startsWith("@")) return false;
        try {
            const url = new URL(l.link);
            return url.origin === UI_URL;
        } catch (e) {
            return false;
        }
    });

    let botsToRespond: string[] = [];
    // If one of the links is "@Everyone", all bots should respond
    if (links.some(l => l.label === "@Everyone")) {
        botsToRespond = chat?.botParticipants ?? [];
    }
    // Otherwise, find the bots that were mentioned by name (e.g. "@BotName")
    else {
        botsToRespond = links.map(l => {
            const botId = bots.find(b => b.name === l.label.slice(1))?.id;
            if (!botId) return null;
            return botId;
        }).filter(id => id !== null) as string[];

        botsToRespond = [...new Set(botsToRespond)];
    }
    return botsToRespond;
}

/**
 * Determines which bots should respond based on the message content and chat context.
 * 
 * Conditions:
 * 1. If the message content is blank (likely meaning the message was updated but not its actual content), then no bots should respond.
 * 2. If the message is not associated with your user ID, no bots should respond.
 * 3. If there are no bots in the chat, no bots should respond.
 * 4. If there is one bot in the chat and two participants (i.e., just you and the bot), the bot should respond.
 * 5. Otherwise, check the message to see if any bots were mentioned.
 * 
 * @param message - The message that might trigger bot responses.
 * @param messageFromUserId - The ID of the user who sent the message.
 * @param chat - Information about the chat where the message was sent.
 * @param bots - Information about the bots in the chat.
 * @param userId - The ID of the user sending the message.
 * @returns An array of botIds that should respond to the message.
 */
export function determineRespondingBots(
    message: string | null,
    messageFromUserId: string,
    chat: { botParticipants?: string[], participantsCount?: number },
    bots: Pick<PreMapUserData, "id" | "name">[],
    userId: string,
): string[] {
    if (
        !chat ||
        !message ||
        message.trim() === "" ||
        !messageFromUserId ||
        messageFromUserId !== userId ||
        bots.length === 0
    ) {
        return [];
    }

    if (bots.length === 1 && chat.participantsCount === 2) {
        return [bots[0].id];
    } else {
        return processMentions(message, chat, bots);
    }
}

/**
 * Stringifies a list of task context objects. These are used to provide context to the LLM,
 * typically when referencing a form or other client-side data.
 * @param taskLabel The label for the current task type, which may be used in the template 
 * @param taskContexts The list of task context objects to stringify
 * @param contextTemplateDefault The default template to use for stringifying the data
 * @returns A stringified list of task context objects
 */
export function stringifyTaskContexts(
    taskLabel: string,
    taskContexts: TaskContextInfo[],
    contextTemplateDefault?: string,
): string {
    let result = "";

    for (let i = 0; i < taskContexts.length; i++) {
        const { template, templateVariables, data } = taskContexts[i];

        let contextString = "";
        const stringifiedData = typeof data === "string" ? data : JSON.stringify(data, null, 2);

        // If the template is defined, replace template variables with actual data
        if (template || contextTemplateDefault) {
            contextString = template || contextTemplateDefault || "";

            // Define template variables
            const variables = {
                [templateVariables?.task || "<TASK>"]: taskLabel,
                [templateVariables?.data || "<DATA>"]: stringifiedData,
            };
            // Sort variables from longest to shortest, to avoid issues with overlapping variable names
            const sortedVariables = Object.entries(variables).sort(
                ([varNameA], [varNameB]) => varNameB.length - varNameA.length,
            );
            // Replace each variable in the template
            for (const [varName, value] of sortedVariables) {
                // Escape regex special characters in variable names
                const escapedVarName = varName.replace(/[-\/\\^$*+?.()|[\]{}]/g, "\\$&");
                const regex = new RegExp(escapedVarName, "g");
                contextString = contextString.replace(regex, value);
            }
        }
        // Otherwise, default to displaying the data
        else {
            contextString = stringifiedData;
        }

        result += contextString;

        // Add spacing between contexts
        if (i < taskContexts.length - 1) {
            result += "\n\n";
        }
    }

    return result;
}

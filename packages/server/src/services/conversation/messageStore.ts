import { DAYS_1_S, DEFAULT_LANGUAGE, MessageConfig, MessageConfigObject, WEEKS_1_DAYS, generatePK } from "@local/shared";
import { Prisma, chat, chat_message, user } from "@prisma/client";
import { type Redis as IoRedis, type Cluster as IoRedisCluster } from "ioredis";
import { DbProvider } from "../../db/provider.js";
import { CacheService } from "../../redisConn.js";
import { AIServiceRegistry } from "./registry.js";
import { AIService } from "./services.js";
import { MessageState } from "./types.js";

/**
 * The `MessageStore` interface defines a contract for managing messages in a conversation system.
 * It provides methods for adding, updating, deleting, and retrieving messages, ensuring that all
 * operations are consistent across different storage layers, such as in-memory caches, Redis, and
 * the database.
 *
 * This interface is designed to abstract the underlying storage mechanisms, allowing for a unified
 * way to interact with message data while maintaining performance and data integrity.
 *
 * Implementations of this interface should handle synchronization between different storage layers
 * and ensure that changes are propagated correctly.
 */
export interface MessageStore {
    /**
     * Adds a new message to the specified conversation.
     *
     * @param conversationId - The ID of the conversation to which the message belongs.
     * @param message - The message data to add, excluding `id` and `createdAt` fields.
     * @param skipDb - If true, we assume the message is already in the database and we only update the cache.
     * @returns A promise that resolves to the persisted `MessageState` object.
     *
     * This method creates a new message in the database, generates a unique ID and timestamp,
     * and updates any necessary caches (e.g., Redis) to ensure consistency.
     */
    addMessage(
        conversationId: string,
        message: Omit<MessageState, "id" | "createdAt">,
        skipDb?: boolean
    ): Promise<MessageState>;

    /**
    * Updates an existing message with the provided partial updates.
    *
    * @param messageId - The ID of the message to update.
    * @param updates - The partial updates to apply to the message.
    * @param skipDb - If true, we assume the database has already been updated and we only update the cache.
    * @returns A promise that resolves to the updated `MessageState` object.
    *
    * This method applies the updates to the message in the database and ensures that any
    * caches (e.g., Redis) are updated or invalidated as necessary.
    */
    updateMessage(
        messageId: string,
        updates: Partial<Omit<MessageState, "id" | "createdAt">>,
        skipDb?: boolean
    ): Promise<MessageState>;

    /**
     * Deletes a message by its ID.
     *
     * @param messageId - The ID of the message to delete.
     * @param skipDb - If true, we assume the database has already been updated and we only update the cache.
     * @returns A promise that resolves when the message has been deleted.
     *
     * This method removes the message from the database and ensures that any caches (e.g., Redis)
     * are updated to reflect the deletion.
     */
    deleteMessage(messageId: string, skipDb?: boolean): Promise<void>;

    /**
     * Retrieves a message by its ID.
     *
     * @param messageId - The ID of the message to retrieve.
     * @returns A promise that resolves to the `MessageState` object or `null` if not found.
     *
     * This method fetches the message from the database and may use caching mechanisms
     * to improve performance.
     */
    getMessage(messageId: string): Promise<MessageState | null>;

    /**
     * Retrieves a list of messages for a specific conversation, with optional filtering.
     *
     * @param conversationId - The ID of the conversation to retrieve messages from.
     * @param options - Optional parameters to filter the messages (e.g., limit, before, after).
     * @returns A promise that resolves to an array of `MessageState` objects.
     *
     * This method retrieves messages from the database and ensures that any caches (e.g., Redis)
     * are synchronized with the latest data.
     */
    getConversationMessages(
        conversationId: string,
        options?: { limit?: number; before?: string; after?: string }
    ): Promise<MessageState[]>;

    /**
     * Retrieves a message along with its token count for a specific AI model.
     *
     * @param messageId - The ID of the message to retrieve.
     * @param aiModel - The AI model for which to compute the token count.
     * @returns A promise that resolves to an object containing the `MessageState` and its token size,
     *          or `null` if the message is not found.
     *
     * This method fetches the message and computes its token count based on the specified AI model.
     * The token count is cached for future requests to improve performance.
     */
    getMessageWithTokenCount(
        messageId: string,
        aiModel: string
    ): Promise<{ message: MessageState; tokenSize: number } | null>;
}

/**
 * The shape of a message from the database.
 */
type MessageDbData = Pick<chat_message, "id" | "text" | "createdAt" | "config" | "language"> & {
    chat: Pick<chat, "id">;
    user?: Pick<user, "id" | "name" | "handle" | "isBot"> | null
    parent?: Pick<chat_message, "id"> | null
};

const chatMessageSelect = {
    id: true,
    text: true,
    createdAt: true,
    config: true,
    language: true,
    chat: {
        select: {
            id: true,
        },
    },
    parent: {
        select: {
            id: true,
        },
    },
    user: {
        select: {
            id: true,
            handle: true,
            isBot: true,
            name: true,
        },
    },
} satisfies Prisma.chat_messageSelect;

/**
 * Converts a message from the database to the shape used by ConversationLoop and related logic.
 * @param message - The message from the database
 * @returns A MessageState object
 */
function messageDbToMessageState(message: MessageDbData): MessageState {
    const config = (message.config ?? MessageConfig.default()) as MessageConfigObject;
    const language = message.language ?? DEFAULT_LANGUAGE;
    return {
        id: message.id.toString(),
        config,
        language,
        text: message.text,
        parent: message.parent ? { id: message.parent.id.toString() } : null,
        user: message.user ? { id: message.user.id.toString() } : null,
    };
}

/**
 * Represents a mapping of token count keys to their numeric values.
 * Used to store token counts for different models and encodings.
 * Keys are typically in the format `${estimationModel}-${encoding}`.
 */
interface StoredTokenCounts { [key: string]: number; }

/**
 * Represents a cached chat message in Redis.
 * This interface defines the structure of a message stored in Redis,
 * including its ID, token counts (of content + metadata), parent message ID, and user ID.
 */
interface CachedChatMessage {
    id: string;
    tokenCounts: string; // stringified StoredTokenCounts
    parentId?: string;
    userId?: string;
    chatId: string;
}

/**
 * The `TokenCounter` class is responsible for estimating and managing token counts for text inputs
 * using a specified AI service and model. It provides methods to generate a unique key based on the
 * model's estimation configuration, parse stored token counts, and ensure that token counts are
 * computed and cached for given texts.
 *
 * This class is useful for optimizing token count estimations by avoiding redundant computations
 * and ensuring that token counts are consistent for the same text and model configuration.
 */
class TokenCounter {
    /**
     * Creates a new `TokenCounter` instance with the specified AI service and model.
     *
     * @param service - The AI service used for token estimation.
     * @param model - The specific model for which token counts are estimated.
     */
    constructor(private service: AIService<string>, private model: string) { }

    /**
     * Generates a unique key based on the estimation model and encoding of the specified model.
     * This key is used to store and retrieve token counts for different model configurations.
     *
     * @returns A string key in the format `${estimationModel}-${encoding}`.
     */
    private _key(): string {
        const { estimationModel, encoding } = this.service.getEstimationInfo(this.model);
        return `${estimationModel}-${encoding}`;
    }

    /**
     * Returns the unique key for the current model configuration.
     * This method provides public access to the key generated by the private `_key()` method.
     *
     * @returns The unique key as a string.
     */
    getKey(): string {
        return this._key();
    }

    /**
     * Parses a raw string into a `StoredTokenCounts` object.
     * If the input is not provided or cannot be parsed, an empty object is returned.
     *
     * @param raw - An optional string representing serialized token counts.
     * @returns A `StoredTokenCounts` object, or an empty object if parsing fails.
     */
    parse(raw?: string): StoredTokenCounts {
        if (!raw) return {};
        try {
            return JSON.parse(raw) as StoredTokenCounts;
        } catch {
            return {};
        }
    }

    /**
     * Ensures that the token count for the given message is computed and stored in the provided `counts` object.
     * If the token count for the current model configuration is not already present in `counts`, it is estimated
     * using the AI service and stored under the unique key.
     * 
     * NOTE: We include the message's text AND metadata in the token count estimation, since metadata is also 
     * typically included in the context window (e.g. tool use)
     *
     * @param message - The message for which to estimate the token count.
     * @param counts - The `StoredTokenCounts` object where token counts are stored.
     * @returns An object containing the updated `counts` and the `size` (token count) for the text.
     */
    ensure(message: MessageState, counts: StoredTokenCounts): { counts: StoredTokenCounts; size: number } {
        const dataToEstimate = `${message.text}\n${JSON.stringify(message.config)}`;
        const k = this._key();
        if (counts[k] === undefined) {
            counts[k] = this.service.estimateTokens({ aiModel: this.model, text: dataToEstimate }).tokens;
        }
        return { counts, size: counts[k] };
    }
}

/**
 * A singleton class for caching chat context data using Redis.
 * This cache manages messages, their relationships, and conversation histories
 * to optimize access and reduce database queries.
 *
 * The cache uses the following Redis data structures:
 * - Hashes for storing message data (`msg:<id>`).
 * - Sorted sets for maintaining ordered lists of messages in a chat (`chat:<id>`).
 * - Sets for tracking child messages of a parent message (`children:<id>`).
 *
 * All cache entries have a TTL (Time To Live) set to ensure data freshness.
 */
class ChatContextCache {
    private static _instance: ChatContextCache | null = null;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor(
        private readonly ttl_s: number = DAYS_1_S * WEEKS_1_DAYS, // Default cache expiry to 1 week
    ) { }

    /**
     * Retrieves the singleton instance of the ChatContextCache.
     * If no instance exists, a new one is created.
     *
     * @returns The singleton instance of ChatContextCache.
     */
    static get(): ChatContextCache {
        return this._instance ?? (this._instance = new ChatContextCache());
    }

    // ===== Key helpers =====
    /**
     * Generates the Redis key for a message hash.
     *
     * @param id - The message ID.
     * @returns The Redis key for the message hash (e.g., "msg:123").
     */
    public messageKey = (id: string) => `msg:${id}`;

    /**
     * Generates the Redis key for the set of child messages.
     *
     * @param id - The parent message ID.
     * @returns The Redis key for the children set (e.g., "children:123").
     */
    public childrenKey = (id: string) => `children:${id}`;

    /**
     * Generates the Redis key for the sorted set of messages in a chat.
     *
     * @param id - The chat ID.
     * @returns The Redis key for the chat's message list (e.g., "chat:456").
     */
    public chatKey = (id: string) => `chat:${id}`;

    // ===== Utility – write + TTL in one helper =====
    /**
     * Sets an expiration time (TTL) on multiple Redis keys.
     *
     * @param rc - The Redis client.
     * @param keys - The keys to set expiration on.
     */
    private async _expire(rc: IoRedis | IoRedisCluster, ...keys: string[]) {
        const pipe = rc.multi();
        for (const k of keys) pipe.expire(k, this.ttl_s);
        await pipe.exec();
    }

    // ===== Hash helpers =====
    /**
     * Retrieves a message from the Redis cache.
     *
     * @param id - The message ID.
     * @returns The cached message data or null if not found or Redis is unavailable.
     */
    async getMessage(id: string) {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return null; // Should not happen if CacheService.ensure works
        const data = await rawClient.hgetall(this.messageKey(id));
        return Object.keys(data).length ? (data as unknown as CachedChatMessage) : null;
    }

    /**
    * Stores a message in the Redis cache.
    *
    * @param id - The message ID.
    * @param data - The message data to cache.
    */
    async setMessage(id: string, data: CachedChatMessage) {
        const rawClient = await CacheService.get().raw();
        await rawClient.hset(this.messageKey(id), data as any);
        await this._expire(rawClient, this.messageKey(id));
    }

    /**
     * Updates a specific field of a message in the Redis cache.
     *
     * @param id - The message ID.
     * @param field - The field to update.
     * @param val - The new value for the field.
     */
    async setMessageField(id: string, field: string, val: string) {
        const rawClient = await CacheService.get().raw();
        await rawClient.hset(this.messageKey(id), field, val);
        await this._expire(rawClient, this.messageKey(id));
    }

    // ===== ZSET helpers =====
    /**
     * Extracts a score from a Snowflake ID for use in sorted sets.
     * The score is derived from the timestamp portion of the Snowflake ID (41 bits).
     *
     * @param id - The Snowflake ID.
     * @returns The extracted score as a number.
     */
    public scoreFromSnowflake(id: string): number {
        // eslint-disable-next-line no-magic-numbers
        return Number((BigInt(id) >> 22n) & 0x1fffffffffffn); // 41‑bit mask
    }

    /**
     * Adds a message to a chat's sorted set in Redis.
     *
     * @param chatId - The chat ID.
     * @param messageId - The message ID to add.
     */
    async addMessageToChat(chatId: string, messageId: string) {
        const rawClient = await CacheService.get().raw();
        const score = this.scoreFromSnowflake(messageId);
        await rawClient.zadd(this.chatKey(chatId), score, messageId);
        await this._expire(rawClient, this.chatKey(chatId));
    }

    /**
    * Retrieves the latest message ID from a chat's sorted set.
    *
    * @param chatId - The chat ID.
    * @returns The latest message ID or null if not found or Redis is unavailable.
    */
    async getLatestMessage(chatId: string) {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return null;
        const res = await rawClient.zrange(this.chatKey(chatId), -1, -1);
        return res[0] ?? null;
    }

    /**
     * Retrieves all message IDs from a chat's sorted set, with an optional limit.
     * 
     * @param chatId - The chat ID.
     * @param limit - Optional maximum number of message IDs to retrieve (default: 100).
     * @returns An array of message IDs or an empty array if Redis is unavailable.
     */
    async getAllMessages(chatId: string, limit = 100) {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return [] as string[];
        return rawClient.zrange(this.chatKey(chatId), 0, limit - 1);
    }

    /**
     * Removes a message from a chat's sorted set.
     *
     * @param chatId - The chat ID.
     * @param messageId - The message ID to remove.
     */
    async removeMessageFromChat(chatId: string, messageId: string) {
        const rawClient = await CacheService.get().raw();
        await rawClient.zrem(this.chatKey(chatId), messageId);
    }

    // ===== Children helpers =====
    /**
     * Adds a child message to a parent's set in Redis.
     *
     * @param parent - The parent message ID.
     * @param child - The child message ID.
     */
    async addChild(parent: string, child: string) {
        const rawClient = await CacheService.get().raw();
        await rawClient.sadd(this.childrenKey(parent), child);
        await this._expire(rawClient, this.childrenKey(parent));
    }

    /**
     * Removes a child message from a parent's set in Redis.
     *
     * @param parent - The parent message ID.
     * @param child - The child message ID.
     */
    async removeChild(parent: string, child: string) {
        const rawClient = await CacheService.get().raw();
        await rawClient.srem(this.childrenKey(parent), child);
    }

    /**
     * Retrieves the child message IDs of a parent message.
     *
     * @param id - The parent message ID.
     * @returns An array of child message IDs or an empty array if not found or Redis is unavailable.
     */
    async getChildren(id: string) {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return [] as string[];
        return rawClient.smembers(this.childrenKey(id));
    }

    // ===== Bulk helpers =====
    /**
     * Deletes multiple keys from Redis.
     *
     * @param keys - The keys to delete.
     */
    async deleteKeys(keys: string[]) {
        if (keys.length) {
            const rawClient = await CacheService.get().raw();
            await rawClient.del(keys);
        }
    }

    /**
     * Clears all cached data for a chat, including the chat's message list and all message data.
     *
     * @param chatId - The chat ID.
     */
    async clearChat(chatId: string) {
        const messageIds = await this.getAllMessages(chatId);
        const keys: string[] = [this.chatKey(chatId)];
        for (const m of messageIds) keys.push(this.messageKey(m), this.childrenKey(m));
        await this.deleteKeys(keys);
    }
}

/**
 * The `PrismaRedisMessageStore` class implements the `MessageStore` interface, providing a concrete
 * implementation that integrates with Prisma for database persistence and Redis for caching.
 *
 * This class ensures that message data is consistently stored and retrieved across different layers:
 * - **Database (Prisma)**: Handles permanent storage and retrieval of messages.
 * - **Redis**: Caches message data, including token counts, for fast access and context building.
 *
 * Key Features:
 * - **Consistency**: Ensures that changes (add, update, delete) are reflected in both the database and Redis cache.
 * - **Token Counts**: Computes and caches token counts for messages on demand, specific to AI models.
 * - **Efficient Access**: Provides quick access to message data through Redis caching while maintaining data integrity.
 *
 * This implementation is designed to work seamlessly with the conversation system, ensuring that message
 * operations are efficient and synchronized across all storage layers.
 */
export class PrismaRedisMessageStore implements MessageStore {
    private readonly prisma = DbProvider.get();
    private readonly cache = ChatContextCache.get();
    private readonly tokenService = AIServiceRegistry.get();

    async addMessage(
        conversationId: string,
        message: MessageState,
        skipDb?: boolean,
    ): Promise<MessageState> {
        const messageId = generatePK();
        const parentId = message.parent?.id.toString(); // Cast in case it's still a bigint
        const userId = message.user?.id.toString(); // Cast in case it's still a bigint
        const createdAt = new Date();
        let messageState: MessageState = message;
        if (!skipDb) {
            const row = await this.prisma.chat_message.create({
                data: {
                    id: BigInt(messageId),
                    config: message.config as unknown as Prisma.InputJsonValue,
                    createdAt,
                    text: message.text,
                    language: message.language ?? DEFAULT_LANGUAGE,
                    chat: { connect: { id: BigInt(conversationId) } },
                    ...(parentId ? { parent: { connect: { id: BigInt(parentId) } } } : {}),
                    ...(userId ? { user: { connect: { id: BigInt(userId) } } } : {}),
                },
                select: chatMessageSelect,
            });
            messageState = messageDbToMessageState(row);
        }

        // Update Redis
        const cachedMessage: CachedChatMessage = {
            id: messageId.toString(),
            tokenCounts: "{}", // Initially empty, computed on demand
            parentId,
            userId,
            chatId: conversationId,
        };
        await this.cache.setMessage(cachedMessage.id, cachedMessage);
        await this.cache.addMessageToChat(conversationId, cachedMessage.id);
        if (parentId) {
            await this.cache.addChild(parentId, cachedMessage.id);
        }

        return messageState;
    }

    async updateMessage(
        messageId: string,
        updates: Partial<MessageState>,
        skipDb?: boolean,
    ): Promise<MessageState> {
        const parentId = updates.parent?.id.toString(); // Cast in case it's still a bigint
        const userId = updates.user?.id.toString(); // Cast in case it's still a bigint
        let updated: MessageState;
        // If we're skipping the database, we'll assume the `updates` object is a full MessageState object
        if (skipDb) {
            updated = updates as MessageState;
        } else {
            const row = await this.prisma.chat_message.update({
                where: { id: BigInt(messageId) },
                data: {
                    text: updates.text,
                    config: updates.config as unknown as Prisma.InputJsonValue,
                    language: updates.language ?? DEFAULT_LANGUAGE,
                    ...(parentId ? { parent: { connect: { id: BigInt(parentId) } } } : {}),
                    ...(userId ? { user: { connect: { id: BigInt(userId) } } } : {}),
                },
                select: chatMessageSelect,
            });
            updated = messageDbToMessageState(row);
        }

        // Invalidate the cache
        const existing = await this.cache.getMessage(messageId);
        if (existing) {
            const msg: CachedChatMessage = {
                ...existing,
                tokenCounts: "{}", // Invalidate token counts
                parentId: parentId ?? existing.parentId,
                userId: userId ?? existing.userId,
                chatId: existing.chatId, // Preserve chatId from the existing cached message
            };
            await this.cache.setMessage(messageId, msg);
            if (parentId !== undefined) {
                if (existing.parentId && (parentId !== existing.parentId || parentId === null)) {
                    await this.cache.removeChild(existing.parentId, messageId);
                }
                if (typeof parentId === "string") {
                    await this.cache.addChild(parentId, messageId);
                }
            }
        }
        return updated;
    }

    async deleteMessage(messageId: string, skipDb?: boolean): Promise<void> {
        if (!skipDb) {
            await this.prisma.chat_message.delete({ where: { id: BigInt(messageId) } });
        }

        // Update Redis
        const msgCacheEntry = await this.cache.getMessage(messageId);
        if (msgCacheEntry) {
            if (msgCacheEntry.parentId) {
                await this.cache.removeChild(msgCacheEntry.parentId, messageId);
            }

            if (msgCacheEntry.chatId) {
                await this.cache.removeMessageFromChat(msgCacheEntry.chatId, messageId);
            }
        }
        await this.cache.deleteKeys([this.cache.messageKey(messageId), this.cache.childrenKey(messageId)]);
    }

    async getMessage(messageId: string): Promise<MessageState | null> {
        let message: MessageState | null = null;
        const row = await this.prisma.chat_message.findUnique({
            where: { id: BigInt(messageId) },
            select: chatMessageSelect,
        });
        if (row) message = messageDbToMessageState(row);

        // Sync Redis if not present
        if (message && row) {
            const cached = await this.cache.getMessage(messageId);
            if (!cached) {
                const parentId = message.parent?.id.toString();
                const userId = message.user?.id.toString();
                const msgToCache: CachedChatMessage = {
                    id: messageId,
                    tokenCounts: "{}",
                    parentId,
                    userId,
                    chatId: row.chat.id.toString(),
                };
                await this.cache.setMessage(messageId, msgToCache);
            }
        }

        return message;
    }

    async getConversationMessages(
        conversationId: string,
        options: { limit?: number; before?: string; after?: string } = {},
    ): Promise<MessageState[]> {
        const { limit, before, after } = options;
        const rows = await this.prisma.chat_message.findMany({
            where: {
                chatId: BigInt(conversationId),
                ...(before ? { id: { lt: BigInt(before) } } : {}),
                ...(after ? { id: { gt: BigInt(after) } } : {}),
            },
            orderBy: { createdAt: "desc" },
            take: limit,
            select: chatMessageSelect,
        });
        const messages = rows.map(messageDbToMessageState);

        // Sync Redis and remove stale entries
        const cachedIds = await this.cache.getAllMessages(conversationId);
        const fetchedIds = messages.map(m => m.id);
        const staleIds = cachedIds.filter(id => !fetchedIds.includes(id));
        for (const staleId of staleIds) {
            await this.cache.removeMessageFromChat(conversationId, staleId);
            await this.cache.deleteKeys([this.cache.messageKey(staleId), this.cache.childrenKey(staleId)]);
        }
        const missing = messages.filter(missingMessage => missingMessage.id && !cachedIds.includes(missingMessage.id));
        for (const missingMessage of missing) {
            const currentMessageId = missingMessage.id;
            if (!currentMessageId) continue; // Should not happen
            const parentId = missingMessage.parent?.id.toString();
            const userId = missingMessage.user?.id.toString();
            const msgToCache: CachedChatMessage = {
                id: currentMessageId,
                tokenCounts: "{}",
                parentId: parentId ?? undefined,
                userId: userId ?? undefined,
                chatId: conversationId,
            };
            await this.cache.setMessage(currentMessageId, msgToCache);
            await this.cache.addMessageToChat(conversationId, currentMessageId);
            if (parentId) {
                await this.cache.addChild(parentId, currentMessageId);
            }
        }

        return messages;
    }

    async getMessageWithTokenCount(
        messageId: string,
        aiModel: string,
    ): Promise<{ message: MessageState; tokenSize: number } | null> {
        const message = await this.getMessage(messageId);
        if (!message) return null;

        let tokenSize: number | undefined;
        const cached = await this.cache.getMessage(messageId);
        const service = this.tokenService.getService(this.tokenService.getServiceId(aiModel));
        const tokenCounter = new TokenCounter(service, aiModel);
        const key = tokenCounter.getKey();
        const counts = tokenCounter.parse(cached?.tokenCounts);

        if (counts[key] === undefined) {
            const { counts: updatedCounts, size } = tokenCounter.ensure(message, counts);
            tokenSize = size;
            await this.cache.setMessageField(messageId, "tokenCounts", JSON.stringify(updatedCounts));
        } else {
            tokenSize = counts[key];
        }

        if (tokenSize === undefined) {
            // This case should ideally not be reached if logic is correct
            // but as a fallback, compute and store if still undefined.
            const { counts: finalCounts, size } = tokenCounter.ensure(message, counts);
            tokenSize = size;
            await this.cache.setMessageField(messageId, "tokenCounts", JSON.stringify(finalCounts));
        }

        return { message, tokenSize };
    }
}


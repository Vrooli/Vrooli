import { DAYS_1_S, type MessageConfigObject, WEEKS_1_DAYS } from "@vrooli/shared";
import { type Redis as IoRedis, type Cluster as IoRedisCluster } from "ioredis";
import { CacheService } from "../../redisConn.js";
import { type MessageState } from "../conversation/types.js";
import { AIServiceRegistry } from "./registry.js";
import { type AIService } from "./services.js";

/**
 * The `MessageStore` interface defines a contract for managing messages in a conversation system,
 * focusing solely on caching mechanisms (e.g., Redis). It provides methods for adding, updating,
 * deleting, and retrieving messages from the cache.
 *
 * This interface abstracts the underlying caching mechanisms, allowing for a unified way to
 * interact with cached message data. Database persistence and tree integrity are assumed to be
 * handled by other layers of the application (see {@link ../../utils/messageTree.ts}, {@link ../../models/base/chat.ts} and {@link ../../models/base/chatMessage.ts}).
 *
 * Messages within this system can form a tree structure. Each message can have an optional `parent`
 * message. Implementations must manage these relationships carefully within the cache.
 */
export interface MessageStore {
    /**
     * Adds a new message to the cache for the specified conversation.
     * The message is assumed to have been persisted in the database already.
     *
     * @param conversationId - The ID of the conversation to which the message belongs.
     * @param message - The complete `MessageState` object (including DB-generated `id` and `createdAt`).
     * @returns A promise that resolves to the cached `MessageState` object.
     */
    addMessage(
        conversationId: string,
        message: MessageState,
    ): Promise<MessageState>;

    /**
    * Updates an existing message in the cache with the provided full updates.
    * The message is assumed to have been updated in the database already.
    *
    * @param messageId - The ID of the message to update in the cache.
    * @param updates - The complete `MessageState` object reflecting the updated state.
    * @returns A promise that resolves to the updated `MessageState` object from the cache.
    */
    updateMessage(
        messageId: string,
        updates: MessageState,
    ): Promise<MessageState>;

    /**
     * Deletes a message by its ID from the cache.
     * Database deletion is assumed to be handled by another layer.
     *
     * @param messageId - The ID of the message to delete from the cache.
     * @returns A promise that resolves when the message has been deleted from the cache.
     */
    deleteMessage(messageId: string): Promise<void>;

    /**
     * Retrieves a message by its ID from the cache.
     *
     * @param messageId - The ID of the message to retrieve.
     * @returns A promise that resolves to the `MessageState` object from the cache or `null` if not found.
     */
    getMessage(messageId: string): Promise<MessageState | null>;

    /**
     * Retrieves a list of messages for a specific conversation from the cache.
     *
     * @param conversationId - The ID of the conversation to retrieve messages from.
     * @param options - Optional parameters to filter/paginate the messages from the cache (e.g., limit).
     * @returns A promise that resolves to an array of `MessageState` objects from the cache.
     */
    getConversationMessages(
        conversationId: string,
        options?: { limit?: number; before?: string; after?: string } // `before` and `after` might need score-based fetching for ZSETs
    ): Promise<MessageState[]>;

    /**
     * Retrieves a message from the cache along with its token count for a specific AI model.
     *
     * @param messageId - The ID of the message to retrieve from the cache.
     * @param aiModel - The AI model for which to compute the token count.
     * @returns A promise that resolves to an object containing the `MessageState` (from cache) and its token size,
     *          or `null` if the message is not found in the cache.
     */
    getMessageWithTokenCount(
        messageId: string,
        aiModel: string
    ): Promise<{ message: MessageState; tokenSize: number } | null>;
}

/**
 * Represents a mapping of token count keys to their numeric values.
 * Used to store token counts for different models and encodings.
 * Keys are typically in the format `${estimationModel}-${encoding}`.
 */
interface StoredTokenCounts { [key: string]: number; }

/**
 * Represents a cached chat message in Redis.
 * This interface defines the structure of a message stored in Redis.
 * It extends MessageState to include all its fields directly, plus cache-specific fields.
 */
interface CachedChatMessage extends MessageState {
    // Inherits: id, createdAt, config, language, text, parent, user from MessageState
    tokenCounts: string; // stringified StoredTokenCounts
    chatId: string;      // ID of the chat this message belongs to, for context.
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
export class TokenCounter {
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
 * Maximum number of history entries to retain per chat.
 */
const MAX_HISTORY_ENTRIES = 1000;

/**
 * Represents a single message entry in the chat history map.
 * Stored as a JSON object keyed by messageId.
 */
interface HistoryEntry {
    id: string;
    parentId: string | null;
    userId: string | null;
    text: string;
    config: unknown;
    language: string;
    createdAt: string;  // ISO timestamp
    tokenSize: number;
}

/** The history map for a chat, keyed by messageId. */
type HistoryMap = Record<string, HistoryEntry>;

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
export class ChatContextCache {
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

    /**
     * Generates the Redis key for a chat's history map.
     * @param id - The chat ID.
     * @returns The Redis key for the history map.
     */
    public historyKey(id: string): string {
        return `chat:${id}:history`;
    }

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
    async getMessage(id: string): Promise<CachedChatMessage | null> {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return null;
        const data = await rawClient.hgetall(this.messageKey(id));
        // Ensure all required fields for CachedChatMessage are present before casting
        if (Object.keys(data).length > 0 &&
            data.id && data.createdAt && data.text && data.chatId && data.tokenCounts && data.config && data.language) {
            // Attempt to parse date string back to Date object
            // And parse config back to object
            try {
                return {
                    ...data,
                    createdAt: new Date(data.createdAt),
                    config: JSON.parse(data.config as string) as MessageConfigObject,
                    // parent and user might be stringified null/undefined or objects
                    parent: data.parent ? JSON.parse(data.parent as string) : null,
                    user: data.user ? JSON.parse(data.user as string) : null,
                } as unknown as CachedChatMessage;
            } catch (error) {
                console.error("Error parsing cached message data from Redis:", error, data);
                return null;
            }
        }
        return null;
    }

    /**
    * Stores a message in the Redis cache.
    *
    * @param id - The message ID.
    * @param data - The message data to cache.
    */
    async setMessage(id: string, data: CachedChatMessage) {
        const rawClient = await CacheService.get().raw();
        // Ensure complex objects like config, parent, user are stringified for hset
        const dataToStore = {
            ...data,
            createdAt: data.createdAt.toISOString(), // Store dates as ISO strings
            config: JSON.stringify(data.config),
            parent: data.parent ? JSON.stringify(data.parent) : null,
            user: data.user ? JSON.stringify(data.user) : null,
        };
        await rawClient.hset(this.messageKey(id), dataToStore as any);
        await this._expire(rawClient, this.messageKey(id));
    }

    /**
     * Updates a specific field of a message in the Redis cache.
     * Note: This is a low-level update. For updating structured data like `MessageState`,
     * prefer `setMessage` with the full `CachedChatMessage` object.
     *
     * @param id - The message ID.
     * @param field - The field to update.
     * @param val - The new value for the field (should be string or number for hset).
     */
    async setMessageField(id: string, field: string, val: string | number) {
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

    /**
     * Retrieves all message IDs from a chat's sorted set in Redis.
     *
     * @param chatId - The chat ID.
     * @returns An array of all message IDs for the chat.
     */
    async getAllMessageIdsForChat(chatId: string): Promise<string[]> {
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return [] as string[];
        return rawClient.zrange(this.chatKey(chatId), 0, -1); // Fetch all IDs
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
     * @param chatId - The chat ID
     */
    async clearChat(chatId: string) {
        // Use getAllMessageIdsForChat to ensure all messages are targeted for deletion from cache
        const messageIds = await this.getAllMessageIdsForChat(chatId);
        const keys: string[] = [this.chatKey(chatId)]; // Key for the sorted set of chat messages
        for (const m of messageIds) {
            keys.push(this.messageKey(m)); // Key for the message hash
            keys.push(this.childrenKey(m)); // Key for the set of children of this message
        }
        // Delete all collected keys from Redis
        await this.deleteKeys(keys);
    }

    /**
     * Fetches the full history map for a chat.
     * @param chatId - The chat ID.
     * @returns The HistoryMap parsed from Redis, or null if missing or on parse error.
     */
    async getHistoryMap(chatId: string): Promise<HistoryMap | null> {
        const rc = await CacheService.get().raw();
        if (!rc) return null;
        const raw = await rc.get(this.historyKey(chatId));
        if (!raw) return null;
        try {
            return JSON.parse(raw) as HistoryMap;
        } catch (err) {
            console.error(`Error parsing history map for chat ${chatId}`, err);
            return null;
        }
    }

    /**
     * Overwrites the history map for a chat and resets its TTL.
     * @param chatId - The chat ID.
     * @param map - The full history map to store.
     */
    async setHistoryMap(chatId: string, map: HistoryMap): Promise<void> {
        const rc = await CacheService.get().raw();
        if (!rc) return;
        const key = this.historyKey(chatId);
        await rc.set(key, JSON.stringify(map));
        await this._expire(rc, key);
    }

    /**
     * Adds or updates a single entry in the history map for a chat.
     * Performs a get-modify-set inside a Redis transaction to avoid races.
     * Trims the map to the most recent MAX_HISTORY_ENTRIES.
     * @param chatId - The chat ID.
     * @param entry - The HistoryEntry to upsert.
     */
    async upsertHistoryEntry(chatId: string, entry: HistoryEntry): Promise<void> {
        const rc = await CacheService.get().raw();
        if (!rc) return;
        const key = this.historyKey(chatId);
        await rc.watch(key);
        const raw = await rc.get(key);
        const map: HistoryMap = raw ? JSON.parse(raw) : {};
        map[entry.id] = entry;
        // Trim oldest entries if over limit
        const ids = Object.keys(map);
        if (ids.length > MAX_HISTORY_ENTRIES) {
            const sorted = Object.values(map).sort((a, b) =>
                new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
            );
            for (let i = 0; i < ids.length - MAX_HISTORY_ENTRIES; i++) {
                delete map[sorted[i].id];
            }
        }
        const tx = rc.multi();
        tx.set(key, JSON.stringify(map));
        tx.expire(key, this.ttl_s);
        await tx.exec();
    }

    /**
     * Deletes an entry from the history map, reparenting its children to the deleted entry's parent.
     * @param chatId - The chat ID.
     * @param messageId - The ID of the message to delete.
     */
    async deleteHistoryEntry(chatId: string, messageId: string): Promise<void> {
        const rc = await CacheService.get().raw();
        if (!rc) return;
        const key = this.historyKey(chatId);
        await rc.watch(key);
        const raw = await rc.get(key);
        if (!raw) return;
        const map: HistoryMap = JSON.parse(raw);
        const deleted = map[messageId];
        if (!deleted) {
            await rc.unwatch();
            return;
        }
        const parent = deleted.parentId;
        delete map[messageId];
        // Reparent children
        Object.values(map).forEach(e => {
            if (e.parentId === messageId) {
                e.parentId = parent;
            }
        });
        const tx = rc.multi();
        tx.set(key, JSON.stringify(map));
        tx.expire(key, this.ttl_s);
        await tx.exec();
    }
}

/**
 * The `RedisMessageStore` class implements the `MessageStore` interface, providing a concrete
 * implementation that interacts solely with Redis for caching.
 *
 * This class ensures that message data is consistently managed within the Redis cache.
 * Database persistence and complex tree integrity (beyond cache representation) are assumed
 * to be handled by other application layers (e.g., utilizing {@link ../../utils/messageTree.ts}, {@link ../../models/base/chat.ts}, and {@link ../../models/base/chatMessage.ts}).
 *
 * Key Features:
 * - **Cache-centric**: All operations are performed against Redis.
 * - **Token Counts**: Computes and caches token counts for messages on demand.
 * - **Efficient Access**: Provides quick access to message data through Redis.
 */
export class RedisMessageStore implements MessageStore {
    private readonly cache = ChatContextCache.get();
    private readonly tokenService = AIServiceRegistry.get();

    async addMessage(
        conversationId: string,
        message: MessageState,
    ): Promise<MessageState> {
        // message.id and message.createdAt are assumed to be set by the DB layer

        const cachedMessage: CachedChatMessage = {
            ...message, // Spread all fields from MessageState
            tokenCounts: "{}", // Initially empty, computed on demand
            chatId: conversationId,
        };

        await this.cache.setMessage(cachedMessage.id, cachedMessage);
        await this.cache.addMessageToChat(conversationId, cachedMessage.id);
        if (message.parent?.id) {
            await this.cache.addChild(message.parent.id, cachedMessage.id);
        }
        // Upsert history map entry (tokenSize will be computed on next context build)
        await this.cache.upsertHistoryEntry(conversationId, {
            id: message.id,
            parentId: message.parent?.id ?? null,
            userId: message.user?.id ?? null,
            text: message.text,
            config: message.config,
            language: message.language,
            createdAt: message.createdAt.toISOString(),
            tokenSize: 0,
        });
        return message; // Return the original MessageState object
    }

    async updateMessage(
        messageId: string,
        updates: MessageState,
    ): Promise<MessageState> {
        const existingCached = await this.cache.getMessage(messageId);

        if (!existingCached) {
            // Or throw an error, depending on desired behavior for updating a non-existent cached message
            console.warn(`Attempted to update non-existent message in cache: ${messageId}`);
            return updates; // Or handle as an error
        }

        const updatedCachedMessage: CachedChatMessage = {
            ...updates, // Use all fields from the updates
            chatId: existingCached.chatId, // Preserve original chatId
            tokenCounts: "{}", // Invalidate token counts, to be recomputed on next request
        };

        await this.cache.setMessage(messageId, updatedCachedMessage);
        // Update history map entry
        await this.cache.upsertHistoryEntry(existingCached.chatId, {
            id: messageId,
            parentId: updates.parent?.id ?? null,
            userId: updates.user?.id ?? null,
            text: updates.text,
            config: updates.config,
            language: updates.language,
            createdAt: updates.createdAt.toISOString(),
            tokenSize: 0,
        });
        return updates;
    }

    async deleteMessage(messageId: string): Promise<void> {
        // This logic is largely from the previous refinement for cache cleanup
        const msgCacheEntry = await this.cache.getMessage(messageId);

        if (msgCacheEntry) {
            // 1. Remove the message from its parent's children list (if parent exists)
            if (msgCacheEntry.parent?.id) {
                await this.cache.removeChild(msgCacheEntry.parent.id, messageId);
            }

            // 2. Remove the message from its chat's message list
            // msgCacheEntry.chatId should always exist for a valid cached message
            await this.cache.removeMessageFromChat(msgCacheEntry.chatId, messageId);

            // 3. Handle children of the deleted message in the cache
            const childrenIds = await this.cache.getChildren(messageId);
            const keysToDeleteForChildren: string[] = [];
            for (const childId of childrenIds) {
                keysToDeleteForChildren.push(this.cache.messageKey(childId));
            }
            if (keysToDeleteForChildren.length > 0) {
                await this.cache.deleteKeys(keysToDeleteForChildren);
            }

            // 4. Delete the message's own cache entry and children list
            await this.cache.deleteKeys([
                this.cache.messageKey(messageId),
                this.cache.childrenKey(messageId),
            ]);
            // Remove from history map
            await this.cache.deleteHistoryEntry(msgCacheEntry.chatId, messageId);
        }
    }

    async getMessage(messageId: string): Promise<MessageState | null> {
        const cachedMessage = await this.cache.getMessage(messageId);
        if (cachedMessage) {
            // CachedChatMessage is now compatible with MessageState
            // No specific conversion needed if CachedChatMessage directly includes all MessageState fields
            // and ChatContextCache.getMessage correctly reconstructs objects like Date, config, parent, user.
            return cachedMessage as MessageState;
        }
        return null;
    }

    async getConversationMessages(
        conversationId: string,
        options: { limit?: number; before?: string; after?: string } = {},
    ): Promise<MessageState[]> {
        // Note: 'before' and 'after' for pagination would require score-based fetching from Redis ZSETs.
        // This simplified version primarily uses limit.
        const messageIds = await this.cache.getAllMessages(conversationId, options.limit ?? 100);

        const messages: MessageState[] = [];
        for (const id of messageIds) {
            const message = await this.getMessage(id); // Uses the cache-only getMessage
            if (message) {
                messages.push(message);
            }
        }
        // Ensure messages are sorted by createdAt if Redis zrange doesn't guarantee it
        // or if that's the desired final order. Redis zrange by score (timestamp based) should keep order.
        return messages.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
    }

    async getMessageWithTokenCount(
        messageId: string,
        aiModel: string,
    ): Promise<{ message: MessageState; tokenSize: number } | null> {
        const message = await this.getMessage(messageId); // Uses cache-only getMessage
        if (!message) return null;

        let tokenSize: number | undefined;
        // Fetch the raw string for tokenCounts as it's stored directly in the main hash
        const rawClient = await CacheService.get().raw();
        if (!rawClient) return { message, tokenSize: 0 }; // Or handle error

        const currentTokenCountsStr = await rawClient.hget(this.cache.messageKey(messageId), "tokenCounts");

        const service = this.tokenService.getService(this.tokenService.getServiceId(aiModel));
        const tokenCounter = new TokenCounter(service, aiModel);
        const key = tokenCounter.getKey();
        const counts = tokenCounter.parse(currentTokenCountsStr ?? undefined);

        if (counts[key] === undefined) {
            const { counts: updatedCounts, size } = tokenCounter.ensure(message, counts);
            tokenSize = size;
            // Update only the tokenCounts field in Redis
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


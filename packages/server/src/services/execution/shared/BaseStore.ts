/**
 * Base Store
 * 
 * Generic store pattern for state management across all tiers.
 * Provides a consistent interface with Redis and in-memory implementations.
 * 
 * This reduces duplication while maintaining flexibility for tier-specific needs.
 */

import { type Logger } from "winston";
import { type Redis as RedisClient } from "ioredis";

/**
 * Generic store interface
 */
export interface IStore<T> {
    /**
     * Get an item by ID
     */
    get(id: string): Promise<T | null>;
    
    /**
     * Create or update an item
     */
    set(id: string, data: T): Promise<void>;
    
    /**
     * Delete an item
     */
    delete(id: string): Promise<void>;
    
    /**
     * Check if an item exists
     */
    exists(id: string): Promise<boolean>;
    
    /**
     * Get all items (use with caution for large datasets)
     */
    getAll(): Promise<Map<string, T>>;
    
    /**
     * Clear all items
     */
    clear(): Promise<void>;
}

/**
 * Base in-memory store implementation
 */
export class InMemoryStore<T> implements IStore<T> {
    protected readonly store: Map<string, T> = new Map();
    
    constructor(protected readonly logger: Logger) {}

    async get(id: string): Promise<T | null> {
        const item = this.store.get(id);
        if (!item) {
            this.logger.debug(`[InMemoryStore] Item not found: ${id}`);
            return null;
        }
        // Return a deep copy to prevent external modifications
        return this.deepCopy(item);
    }

    async set(id: string, data: T): Promise<void> {
        // Store a deep copy to prevent external modifications
        this.store.set(id, this.deepCopy(data));
        this.logger.debug(`[InMemoryStore] Item saved: ${id}`);
    }

    async delete(id: string): Promise<void> {
        const deleted = this.store.delete(id);
        if (deleted) {
            this.logger.debug(`[InMemoryStore] Item deleted: ${id}`);
        } else {
            this.logger.debug(`[InMemoryStore] Item not found for deletion: ${id}`);
        }
    }

    async exists(id: string): Promise<boolean> {
        return this.store.has(id);
    }

    async getAll(): Promise<Map<string, T>> {
        // Return a deep copy of the entire map
        const copy = new Map<string, T>();
        for (const [key, value] of this.store.entries()) {
            copy.set(key, this.deepCopy(value));
        }
        return copy;
    }

    async clear(): Promise<void> {
        const size = this.store.size;
        this.store.clear();
        this.logger.info(`[InMemoryStore] Cleared ${size} items`);
    }

    /**
     * Create a deep copy of an object
     */
    protected deepCopy(obj: T): T {
        return JSON.parse(JSON.stringify(obj));
    }
}

/**
 * Base Redis store implementation
 */
export class RedisStore<T> implements IStore<T> {
    constructor(
        protected readonly logger: Logger,
        protected readonly redis: RedisClient,
        protected readonly keyPrefix: string,
        protected readonly ttlSeconds?: number,
    ) {}

    async get(id: string): Promise<T | null> {
        const key = this.makeKey(id);
        try {
            const data = await this.redis.get(key);
            if (!data) {
                this.logger.debug(`[RedisStore] Item not found: ${key}`);
                return null;
            }
            return JSON.parse(data) as T;
        } catch (error) {
            this.logger.error(`[RedisStore] Error getting item: ${key}`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async set(id: string, data: T): Promise<void> {
        const key = this.makeKey(id);
        try {
            const json = JSON.stringify(data);
            if (this.ttlSeconds) {
                await this.redis.setex(key, this.ttlSeconds, json);
            } else {
                await this.redis.set(key, json);
            }
            this.logger.debug(`[RedisStore] Item saved: ${key}`);
        } catch (error) {
            this.logger.error(`[RedisStore] Error saving item: ${key}`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async delete(id: string): Promise<void> {
        const key = this.makeKey(id);
        try {
            const deleted = await this.redis.del(key);
            if (deleted > 0) {
                this.logger.debug(`[RedisStore] Item deleted: ${key}`);
            } else {
                this.logger.debug(`[RedisStore] Item not found for deletion: ${key}`);
            }
        } catch (error) {
            this.logger.error(`[RedisStore] Error deleting item: ${key}`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async exists(id: string): Promise<boolean> {
        const key = this.makeKey(id);
        try {
            const exists = await this.redis.exists(key);
            return exists === 1;
        } catch (error) {
            this.logger.error(`[RedisStore] Error checking existence: ${key}`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getAll(): Promise<Map<string, T>> {
        try {
            const pattern = `${this.keyPrefix}:*`;
            const keys = await this.redis.keys(pattern);
            
            if (keys.length === 0) {
                return new Map();
            }

            // Use pipeline for efficient multi-get
            const pipeline = this.redis.pipeline();
            for (const key of keys) {
                pipeline.get(key);
            }
            
            const results = await pipeline.exec();
            const map = new Map<string, T>();
            
            if (!results) {
                return map;
            }

            for (let i = 0; i < keys.length; i++) {
                const [err, data] = results[i];
                if (!err && data) {
                    const id = this.extractId(keys[i]);
                    try {
                        map.set(id, JSON.parse(data as string) as T);
                    } catch (parseError) {
                        this.logger.error(`[RedisStore] Error parsing item: ${keys[i]}`, {
                            error: parseError instanceof Error ? parseError.message : String(parseError),
                        });
                    }
                }
            }

            return map;
        } catch (error) {
            this.logger.error(`[RedisStore] Error getting all items`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async clear(): Promise<void> {
        try {
            const pattern = `${this.keyPrefix}:*`;
            const keys = await this.redis.keys(pattern);
            
            if (keys.length === 0) {
                this.logger.info(`[RedisStore] No items to clear`);
                return;
            }

            // Use pipeline for efficient multi-delete
            const pipeline = this.redis.pipeline();
            for (const key of keys) {
                pipeline.del(key);
            }
            
            await pipeline.exec();
            this.logger.info(`[RedisStore] Cleared ${keys.length} items`);
        } catch (error) {
            this.logger.error(`[RedisStore] Error clearing items`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Create a Redis key from an ID
     */
    protected makeKey(id: string): string {
        return `${this.keyPrefix}:${id}`;
    }

    /**
     * Extract ID from a Redis key
     */
    protected extractId(key: string): string {
        return key.substring(this.keyPrefix.length + 1);
    }
}

/**
 * Store with write-behind caching
 * Useful for high-write scenarios where eventual consistency is acceptable
 */
export class CachedStore<T> implements IStore<T> {
    private readonly cache: Map<string, T> = new Map();
    private readonly dirtyKeys: Set<string> = new Set();
    private flushTimer: NodeJS.Timeout | null = null;

    constructor(
        protected readonly logger: Logger,
        protected readonly backingStore: IStore<T>,
        protected readonly flushIntervalMs: number = 5000,
        protected readonly maxBatchSize: number = 100,
    ) {
        this.startFlushTimer();
    }

    async get(id: string): Promise<T | null> {
        // Check cache first
        if (this.cache.has(id)) {
            return this.deepCopy(this.cache.get(id)!);
        }

        // Load from backing store
        const item = await this.backingStore.get(id);
        if (item) {
            this.cache.set(id, item);
        }
        
        return item;
    }

    async set(id: string, data: T): Promise<void> {
        this.cache.set(id, this.deepCopy(data));
        this.dirtyKeys.add(id);
        
        // Flush if batch size exceeded
        if (this.dirtyKeys.size >= this.maxBatchSize) {
            await this.flush();
        }
    }

    async delete(id: string): Promise<void> {
        this.cache.delete(id);
        // Mark for deletion in backing store
        this.dirtyKeys.add(id);
        
        // Immediate delete from backing store
        await this.backingStore.delete(id);
        this.dirtyKeys.delete(id);
    }

    async exists(id: string): Promise<boolean> {
        return this.cache.has(id) || await this.backingStore.exists(id);
    }

    async getAll(): Promise<Map<string, T>> {
        // Flush any pending changes first
        await this.flush();
        
        // Return from backing store to ensure consistency
        return this.backingStore.getAll();
    }

    async clear(): Promise<void> {
        this.cache.clear();
        this.dirtyKeys.clear();
        await this.backingStore.clear();
    }

    /**
     * Flush dirty items to backing store
     */
    async flush(): Promise<void> {
        if (this.dirtyKeys.size === 0) {
            return;
        }

        const keysToFlush = Array.from(this.dirtyKeys);
        this.dirtyKeys.clear();

        const promises: Promise<void>[] = [];
        
        for (const id of keysToFlush) {
            const item = this.cache.get(id);
            if (item) {
                promises.push(this.backingStore.set(id, item));
            }
        }

        try {
            await Promise.all(promises);
            this.logger.debug(`[CachedStore] Flushed ${keysToFlush.length} items`);
        } catch (error) {
            // Re-add failed keys to dirty set
            keysToFlush.forEach(key => this.dirtyKeys.add(key));
            this.logger.error(`[CachedStore] Error flushing items`, {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Stop the flush timer and flush remaining items
     */
    async shutdown(): Promise<void> {
        this.stopFlushTimer();
        await this.flush();
    }

    private startFlushTimer(): void {
        this.flushTimer = setInterval(() => {
            this.flush().catch(error => 
                this.logger.error(`[CachedStore] Error in periodic flush`, {
                    error: error instanceof Error ? error.message : String(error),
                })
            );
        }, this.flushIntervalMs);
    }

    private stopFlushTimer(): void {
        if (this.flushTimer) {
            clearInterval(this.flushTimer);
            this.flushTimer = null;
        }
    }

    private deepCopy(obj: T): T {
        return JSON.parse(JSON.stringify(obj));
    }
}
import { Redis } from "ioredis";
import { Logger } from "../../../common/logger.js";
import { ErrorHandler, Result } from "./ErrorHandler.js";
import { EventPublisher } from "./EventPublisher.js";

export interface StoreConfig<T> {
    /** Redis key prefix for this store */
    keyPrefix: string;
    /** Default TTL in seconds (0 = no expiration) */
    defaultTTL?: number;
    /** Custom serialization function */
    serialize?: (data: T) => string;
    /** Custom deserialization function */
    deserialize?: (data: string) => T;
    /** Validation function to ensure data integrity */
    validate?: (data: unknown) => data is T;
    /** Whether to publish events on store operations */
    publishEvents?: boolean;
    /** Event channel prefix for store events */
    eventChannelPrefix?: string;
}

export interface BatchOperation<T> {
    key: string;
    value: T;
    ttl?: number;
}

export interface QueryOptions {
    /** Pattern to match keys (e.g., "user:*") */
    pattern?: string;
    /** Maximum number of results */
    limit?: number;
    /** Cursor for pagination */
    cursor?: string;
}

/**
 * Generic Redis store implementation that provides:
 * - Type-safe storage and retrieval
 * - Consistent serialization/deserialization
 * - Batch operations
 * - TTL management
 * - Event publishing on operations
 * - Error handling and logging
 */
export class GenericStore<T> {
    private readonly config: Required<StoreConfig<T>>;
    private readonly errorHandler: ErrorHandler;
    private readonly eventPublisher?: EventPublisher;

    constructor(
        protected readonly logger: Logger,
        protected readonly redis: Redis,
        config: StoreConfig<T>,
        eventPublisher?: EventPublisher,
    ) {
        this.config = {
            keyPrefix: config.keyPrefix,
            defaultTTL: config.defaultTTL ?? 0,
            serialize: config.serialize ?? ((data: T) => JSON.stringify(data)),
            deserialize: config.deserialize ?? ((data: string) => JSON.parse(data) as T),
            validate: config.validate ?? (() => true),
            publishEvents: config.publishEvents ?? false,
            eventChannelPrefix: config.eventChannelPrefix ?? config.keyPrefix,
        };

        this.errorHandler = new ErrorHandler(logger, eventPublisher);
        
        if (this.config.publishEvents && eventPublisher) {
            this.eventPublisher = eventPublisher.createChild(`store.${this.config.keyPrefix}`);
        }
    }

    /**
     * Get a value by key
     */
    async get(key: string): Promise<Result<T | null>> {
        return this.errorHandler.wrap(async () => {
            const fullKey = this.makeKey(key);
            const data = await this.redis.get(fullKey);

            if (!data) {
                return null;
            }

            const value = this.config.deserialize(data);

            if (!this.config.validate(value)) {
                this.logger.warn(`[GenericStore] Invalid data format in key`, {
                    store: this.config.keyPrefix,
                    key: fullKey,
                });
                return null;
            }

            await this.publishEvent("get", key, value);
            return value;
        }, {
            operation: "get",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key },
        });
    }

    /**
     * Get a required value by key (throws if not found)
     */
    async getRequired(key: string): Promise<Result<T>> {
        const result = await this.get(key);
        
        if (!result.success) {
            return result;
        }

        if (result.data === null) {
            return {
                success: false,
                error: new Error(`Key not found: ${this.makeKey(key)}`),
            };
        }

        return { success: true, data: result.data };
    }

    /**
     * Set a value with optional TTL
     */
    async set(key: string, value: T, ttl?: number): Promise<Result<void>> {
        return this.errorHandler.wrap(async () => {
            if (!this.config.validate(value)) {
                throw new Error("Invalid value format");
            }

            const fullKey = this.makeKey(key);
            const serialized = this.config.serialize(value);
            const effectiveTTL = ttl ?? this.config.defaultTTL;

            if (effectiveTTL > 0) {
                await this.redis.setex(fullKey, effectiveTTL, serialized);
            } else {
                await this.redis.set(fullKey, serialized);
            }

            await this.publishEvent("set", key, value);
        }, {
            operation: "set",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key, ttl },
        });
    }

    /**
     * Delete a value by key
     */
    async delete(key: string): Promise<Result<boolean>> {
        return this.errorHandler.wrap(async () => {
            const fullKey = this.makeKey(key);
            const deleted = await this.redis.del(fullKey);

            await this.publishEvent("delete", key, null);
            return deleted > 0;
        }, {
            operation: "delete",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key },
        });
    }

    /**
     * Check if a key exists
     */
    async exists(key: string): Promise<Result<boolean>> {
        return this.errorHandler.wrap(async () => {
            const fullKey = this.makeKey(key);
            const exists = await this.redis.exists(fullKey);
            return exists > 0;
        }, {
            operation: "exists",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key },
        });
    }

    /**
     * Get multiple values by keys
     */
    async getMany(keys: string[]): Promise<Result<Map<string, T>>> {
        return this.errorHandler.wrap(async () => {
            if (keys.length === 0) {
                return new Map();
            }

            const fullKeys = keys.map(key => this.makeKey(key));
            const values = await this.redis.mget(...fullKeys);

            const result = new Map<string, T>();

            for (let i = 0; i < keys.length; i++) {
                const data = values[i];
                if (data) {
                    try {
                        const value = this.config.deserialize(data);
                        if (this.config.validate(value)) {
                            result.set(keys[i], value);
                        }
                    } catch (error) {
                        this.logger.warn(`[GenericStore] Failed to deserialize value`, {
                            store: this.config.keyPrefix,
                            key: keys[i],
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            return result;
        }, {
            operation: "getMany",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { count: keys.length },
        });
    }

    /**
     * Set multiple values in a batch
     */
    async setMany(operations: BatchOperation<T>[]): Promise<Result<void>> {
        return this.errorHandler.wrap(async () => {
            if (operations.length === 0) {
                return;
            }

            const pipeline = this.redis.pipeline();

            for (const { key, value, ttl } of operations) {
                if (!this.config.validate(value)) {
                    throw new Error(`Invalid value format for key: ${key}`);
                }

                const fullKey = this.makeKey(key);
                const serialized = this.config.serialize(value);
                const effectiveTTL = ttl ?? this.config.defaultTTL;

                if (effectiveTTL > 0) {
                    pipeline.setex(fullKey, effectiveTTL, serialized);
                } else {
                    pipeline.set(fullKey, serialized);
                }
            }

            await pipeline.exec();

            // Publish batch event
            await this.publishEvent("setMany", "batch", operations);
        }, {
            operation: "setMany",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { count: operations.length },
        });
    }

    /**
     * Delete multiple values by keys
     */
    async deleteMany(keys: string[]): Promise<Result<number>> {
        return this.errorHandler.wrap(async () => {
            if (keys.length === 0) {
                return 0;
            }

            const fullKeys = keys.map(key => this.makeKey(key));
            const deleted = await this.redis.del(...fullKeys);

            await this.publishEvent("deleteMany", "batch", keys);
            return deleted;
        }, {
            operation: "deleteMany",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { count: keys.length },
        });
    }

    /**
     * Query keys by pattern
     */
    async queryKeys(options: QueryOptions = {}): Promise<Result<string[]>> {
        return this.errorHandler.wrap(async () => {
            const pattern = options.pattern 
                ? this.makeKey(options.pattern)
                : this.makeKey("*");

            const keys: string[] = [];
            const stream = this.redis.scanStream({
                match: pattern,
                count: 100,
            });

            return new Promise((resolve, reject) => {
                stream.on("data", (chunk: string[]) => {
                    // Remove key prefix from results
                    const cleanKeys = chunk.map(key => 
                        key.replace(`${this.config.keyPrefix}:`, "")
                    );
                    keys.push(...cleanKeys);

                    // Apply limit if specified
                    if (options.limit && keys.length >= options.limit) {
                        stream.destroy();
                        resolve(keys.slice(0, options.limit));
                    }
                });

                stream.on("end", () => resolve(keys));
                stream.on("error", reject);
            });
        }, {
            operation: "queryKeys",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { pattern: options.pattern, limit: options.limit },
        });
    }

    /**
     * Get all values matching a pattern
     */
    async queryValues(options: QueryOptions = {}): Promise<Result<T[]>> {
        const keysResult = await this.queryKeys(options);
        
        if (!keysResult.success) {
            return keysResult;
        }

        const valuesResult = await this.getMany(keysResult.data);
        
        if (!valuesResult.success) {
            return valuesResult;
        }

        return {
            success: true,
            data: Array.from(valuesResult.data.values()),
        };
    }

    /**
     * Clear all values in this store
     */
    async clear(): Promise<Result<number>> {
        return this.errorHandler.wrap(async () => {
            const keysResult = await this.queryKeys();
            
            if (!keysResult.success || keysResult.data.length === 0) {
                return 0;
            }

            const deleteResult = await this.deleteMany(keysResult.data);
            return ErrorHandler.unwrap(deleteResult);
        }, {
            operation: "clear",
            component: `GenericStore<${this.config.keyPrefix}>`,
        });
    }

    /**
     * Update TTL for a key
     */
    async updateTTL(key: string, ttl: number): Promise<Result<boolean>> {
        return this.errorHandler.wrap(async () => {
            const fullKey = this.makeKey(key);
            
            if (ttl > 0) {
                const success = await this.redis.expire(fullKey, ttl);
                return success === 1;
            } else {
                const success = await this.redis.persist(fullKey);
                return success === 1;
            }
        }, {
            operation: "updateTTL",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key, ttl },
        });
    }

    /**
     * Get remaining TTL for a key
     */
    async getTTL(key: string): Promise<Result<number>> {
        return this.errorHandler.wrap(async () => {
            const fullKey = this.makeKey(key);
            const ttl = await this.redis.ttl(fullKey);
            return ttl; // -2 if key doesn't exist, -1 if no TTL
        }, {
            operation: "getTTL",
            component: `GenericStore<${this.config.keyPrefix}>`,
            metadata: { key },
        });
    }

    /**
     * Create a full Redis key with prefix
     */
    private makeKey(key: string): string {
        return `${this.config.keyPrefix}:${key}`;
    }

    /**
     * Publish store event if enabled
     */
    private async publishEvent(
        operation: string,
        key: string,
        value: T | T[] | string[] | null,
    ): Promise<void> {
        if (!this.config.publishEvents || !this.eventPublisher) {
            return;
        }

        await this.eventPublisher.publish(
            `${this.config.eventChannelPrefix}.${operation}`,
            `store.${operation}`,
            {
                key,
                value: value !== null && operation !== "delete" ? "***" : null, // Don't publish actual values
                timestamp: new Date(),
            },
            { throwOnError: false },
        );
    }

    /**
     * Create a specialized store with additional functionality
     */
    static withExtensions<T, S extends GenericStore<T>>(
        BaseStore: new (...args: any[]) => S,
    ): new (...args: any[]) => S {
        return BaseStore;
    }
}
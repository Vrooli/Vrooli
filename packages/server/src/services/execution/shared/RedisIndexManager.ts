/**
 * Redis Index Manager
 * 
 * @deprecated This 200+ line Redis indexing utility will be replaced by automated 
 * indexing in SwarmContextManager as outlined in swarm-state-management-redesign.md.
 * 
 * ## DEPRECATION DETAILS:
 * 
 * **Why Deprecated:**
 * 1. **Manual Index Management**: 200+ lines of manual Redis index operations
 * 2. **Complex State Transitions**: Manual state-based indexing prone to race conditions
 * 3. **No Atomic Operations**: Index updates separate from data updates (consistency issues)
 * 4. **Memory Overhead**: Multiple index structures maintained manually
 * 5. **Cleanup Complexity**: Manual TTL and cleanup management across multiple indexes
 * 
 * **Current Manual Operations (To Be Automated):**
 * - addToSet() + removeFromSet() + getSetMembers() - 60+ lines of set management
 * - addToList() + removeFromList() + getListMembers() - 40+ lines of list management  
 * - updateStateIndex() - 50+ lines of complex state transition logic
 * - Manual TTL management and cleanup - 30+ lines
 * - Batch operations and pipeline management - 20+ lines
 * 
 * **Replacement Strategy:**
 * SwarmContextManager will provide automatic indexing through:
 * - Built-in state-based indexing (no manual management)
 * - Atomic context updates with automatic index consistency
 * - Redis pub/sub for automatic index invalidation
 * - Versioned context updates with rollback capability
 * 
 * **Migration Timeline:**
 * - Phase 1: SwarmContextManager handles indexing automatically (weeks 1-2)
 * - Phase 2: Remove manual index operations (weeks 3-4)
 * - Phase 3: Delete RedisIndexManager entirely (weeks 5-6)
 * 
 * **Benefits After Removal:**
 * - 200+ lines removed (100% reduction)
 * - No manual index management or cleanup
 * - Atomic state + index updates (no consistency issues)
 * - Automatic index invalidation through pub/sub
 * - Simplified state operations without index concerns
 * 
 * Centralized utility for managing Redis indexes across all stores.
 * Eliminates duplicate index management code and provides consistent patterns.
 * 
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete replacement plan
 * @see SwarmContextManager - Automated indexing replacement
 */

import { type ChainableCommander, type Redis as RedisClient } from "ioredis";
import { logger } from "../../../events/logger.js";

/**
 * Redis Index Manager interface
 */
export interface IRedisIndexManager {
    // Set-based indexes (unordered collections)
    addToSet(indexKey: string, itemId: string, ttl?: number): Promise<void>;
    removeFromSet(indexKey: string, itemId: string): Promise<void>;
    getSetMembers(indexKey: string): Promise<string[]>;
    setExists(indexKey: string, itemId: string): Promise<boolean>;

    // List-based indexes (ordered collections)
    addToList(indexKey: string, itemId: string, position?: "head" | "tail", ttl?: number): Promise<void>;
    removeFromList(indexKey: string, itemId: string): Promise<void>;
    getListMembers(indexKey: string, start?: number, end?: number): Promise<string[]>;

    // State transition management
    updateStateIndex<T extends string>(
        itemId: string,
        oldState: T | null,
        newState: T,
        stateKeyGenerator: (state: T) => string,
        allStates: T[]
    ): Promise<void>;

    // Bulk operations for cleanup
    cleanupIndexesByPattern(pattern: string): Promise<void>;
    cleanupItemFromAllIndexes(itemId: string, indexPatterns: string[]): Promise<void>;

    // TTL management
    refreshTTL(indexKey: string, ttl: number): Promise<void>;

    // Validation and consistency
    validateIndexConsistency(indexKey: string, expectedItems: string[]): Promise<string[]>;
}

/**
 * Redis Index Manager implementation
 */
export class RedisIndexManager implements IRedisIndexManager {
    constructor(
        private readonly redis: RedisClient,
        private readonly defaultTtl?: number,
    ) { }

    /**
     * Execute multiple Redis operations in a pipeline for efficiency
     */
    private async executePipeline(operations: Array<(pipeline: ChainableCommander) => void>): Promise<void> {
        if (operations.length === 0) return;

        try {
            const pipeline = this.redis.pipeline();
            operations.forEach(op => op(pipeline));
            await pipeline.exec();
        } catch (error) {
            logger.error("[RedisIndexManager] Pipeline execution failed", {
                operationCount: operations.length,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Add item to a set-based index
     */
    async addToSet(indexKey: string, itemId: string, ttl?: number): Promise<void> {
        try {
            const operations: Array<(pipeline: ChainableCommander) => void> = [
                (pipeline) => pipeline.sadd(indexKey, itemId),
            ];

            if (ttl || this.defaultTtl) {
                operations.push((pipeline) => pipeline.expire(indexKey, ttl || this.defaultTtl!));
            }

            await this.executePipeline(operations);

            logger.debug("[RedisIndexManager] Added to set", {
                indexKey,
                itemId,
                ttl: ttl || this.defaultTtl,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to add to set", {
                indexKey,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Remove item from a set-based index
     */
    async removeFromSet(indexKey: string, itemId: string): Promise<void> {
        try {
            await this.redis.srem(indexKey, itemId);

            logger.debug("[RedisIndexManager] Removed from set", {
                indexKey,
                itemId,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to remove from set", {
                indexKey,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get all members of a set-based index
     */
    async getSetMembers(indexKey: string): Promise<string[]> {
        try {
            const members = await this.redis.smembers(indexKey);

            logger.debug("[RedisIndexManager] Retrieved set members", {
                indexKey,
                memberCount: members.length,
            });

            return members;
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to get set members", {
                indexKey,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Check if item exists in a set-based index
     */
    async setExists(indexKey: string, itemId: string): Promise<boolean> {
        try {
            const exists = await this.redis.sismember(indexKey, itemId);
            return exists === 1;
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to check set existence", {
                indexKey,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Add item to a list-based index
     */
    async addToList(indexKey: string, itemId: string, position: "head" | "tail" = "tail", ttl?: number): Promise<void> {
        try {
            const operations: Array<(pipeline: ChainableCommander) => void> = [
                (pipeline) => {
                    if (position === "head") {
                        pipeline.lpush(indexKey, itemId);
                    } else {
                        pipeline.rpush(indexKey, itemId);
                    }
                },
            ];

            if (ttl || this.defaultTtl) {
                operations.push((pipeline) => pipeline.expire(indexKey, ttl || this.defaultTtl!));
            }

            await this.executePipeline(operations);

            logger.debug("[RedisIndexManager] Added to list", {
                indexKey,
                itemId,
                position,
                ttl: ttl || this.defaultTtl,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to add to list", {
                indexKey,
                itemId,
                position,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Remove item from a list-based index
     */
    async removeFromList(indexKey: string, itemId: string): Promise<void> {
        try {
            // Remove all occurrences of the item from the list
            await this.redis.lrem(indexKey, 0, itemId);

            logger.debug("[RedisIndexManager] Removed from list", {
                indexKey,
                itemId,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to remove from list", {
                indexKey,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get members of a list-based index
     */
    async getListMembers(indexKey: string, start = 0, end = -1): Promise<string[]> {
        try {
            const members = await this.redis.lrange(indexKey, start, end);

            logger.debug("[RedisIndexManager] Retrieved list members", {
                indexKey,
                start,
                end,
                memberCount: members.length,
            });

            return members;
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to get list members", {
                indexKey,
                start,
                end,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Update state-based indexes atomically
     */
    async updateStateIndex<T extends string>(
        itemId: string,
        oldState: T | null,
        newState: T,
        stateKeyGenerator: (state: T) => string,
        allStates: T[],
    ): Promise<void> {
        try {
            const operations: Array<(pipeline: ChainableCommander) => void> = [];

            // Remove from old state if it exists
            if (oldState) {
                operations.push((pipeline) => pipeline.srem(stateKeyGenerator(oldState), itemId));
            } else {
                // If no old state specified, remove from all possible states to ensure consistency
                allStates.forEach(state => {
                    if (state !== newState) {
                        operations.push((pipeline) => pipeline.srem(stateKeyGenerator(state), itemId));
                    }
                });
            }

            // Add to new state
            operations.push((pipeline) => pipeline.sadd(stateKeyGenerator(newState), itemId));

            // Apply TTL to new state index
            if (this.defaultTtl) {
                operations.push((pipeline) => pipeline.expire(stateKeyGenerator(newState), this.defaultTtl));
            }

            await this.executePipeline(operations);

            logger.debug("[RedisIndexManager] Updated state index", {
                itemId,
                oldState,
                newState,
                statesProcessed: allStates.length,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to update state index", {
                itemId,
                oldState,
                newState,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Clean up indexes matching a pattern
     */
    async cleanupIndexesByPattern(pattern: string): Promise<void> {
        try {
            const keys = await this.redis.keys(pattern);

            if (keys.length === 0) {
                logger.debug("[RedisIndexManager] No indexes found for cleanup", { pattern });
                return;
            }

            const operations: Array<(pipeline: ChainableCommander) => void> = keys.map(key =>
                (pipeline) => pipeline.del(key),
            );

            await this.executePipeline(operations);

            logger.info("[RedisIndexManager] Cleaned up indexes", {
                pattern,
                keysDeleted: keys.length,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to cleanup indexes", {
                pattern,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Remove an item from all indexes matching given patterns
     */
    async cleanupItemFromAllIndexes(itemId: string, indexPatterns: string[]): Promise<void> {
        try {
            const operations: Array<(pipeline: ChainableCommander) => void> = [];

            for (const pattern of indexPatterns) {
                const keys = await this.redis.keys(pattern);

                keys.forEach(key => {
                    // Remove from sets
                    operations.push((pipeline) => pipeline.srem(key, itemId));
                    // Remove from lists
                    operations.push((pipeline) => pipeline.lrem(key, 0, itemId));
                });
            }

            await this.executePipeline(operations);

            logger.debug("[RedisIndexManager] Cleaned up item from indexes", {
                itemId,
                patternCount: indexPatterns.length,
                operationCount: operations.length,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to cleanup item from indexes", {
                itemId,
                indexPatterns,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Refresh TTL on an index
     */
    async refreshTTL(indexKey: string, ttl: number): Promise<void> {
        try {
            await this.redis.expire(indexKey, ttl);

            logger.debug("[RedisIndexManager] Refreshed TTL", {
                indexKey,
                ttl,
            });
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to refresh TTL", {
                indexKey,
                ttl,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Validate index consistency and return inconsistent items
     */
    async validateIndexConsistency(indexKey: string, expectedItems: string[]): Promise<string[]> {
        try {
            const actualItems = await this.getSetMembers(indexKey);
            const actualSet = new Set(actualItems);
            const expectedSet = new Set(expectedItems);

            const inconsistentItems: string[] = [];

            // Items in index but not expected
            for (const item of actualItems) {
                if (!expectedSet.has(item)) {
                    inconsistentItems.push(`+${item}`); // + means extra
                }
            }

            // Items expected but not in index
            for (const item of expectedItems) {
                if (!actualSet.has(item)) {
                    inconsistentItems.push(`-${item}`); // - means missing
                }
            }

            if (inconsistentItems.length > 0) {
                logger.warn("[RedisIndexManager] Index inconsistency detected", {
                    indexKey,
                    inconsistentItems,
                    actualCount: actualItems.length,
                    expectedCount: expectedItems.length,
                });
            } else {
                logger.debug("[RedisIndexManager] Index consistency validated", {
                    indexKey,
                    itemCount: expectedItems.length,
                });
            }

            return inconsistentItems;
        } catch (error) {
            logger.error("[RedisIndexManager] Failed to validate index consistency", {
                indexKey,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }
}

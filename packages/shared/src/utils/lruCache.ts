import { SECONDS_1_MS } from "../consts/numbers.js";

/** Configuration options for the LRUCache */
export interface LRUCacheOptions {
    /** Maximum number of items allowed in the cache */
    limit: number;
    /** Optional max size in bytes for each value (null = no size limit) */
    maxSizeBytes?: number | null;
    /** Optional default time-to-live for every entry (in ms; <=0 means no expiration) */
    defaultTTLMs?: number | null;
    /** Optional cleanup cooldown in milliseconds (default: 1 second) */
    cleanupCooldownMs?: number;
    /** Optional maximum expired entries before forcing cleanup (default: 100) */
    maxExpiredEntries?: number;
}

/**
 * A Least Recently Used (LRU) cache implementation with optional per-entry TTL.
 * Stores key-value pairs with a capacity limit and evicts the least recently used entries
 * when the cache reaches its limit. Each entry can optionally expire after a TTL.
 */
export class LRUCache<KeyType, ValueType> {
    private limit: number;
    private maxSizeBytes: number | null;   // max size per entry in bytes
    private defaultTTLMs: number | null;      // default TTL for entries in ms
    private cache: Map<KeyType, CacheEntry<ValueType>>;
    private textEncoder = new TextEncoder();
    private lastCleanupTime = 0;
    private cleanupCooldownMs = SECONDS_1_MS;
    private expirationQueue: [number, KeyType][] = []; // [expiresAt, key] sorted by expiration time
    private expirationMap: Map<KeyType, number> = new Map(); // Maps keys to their expiration time
    private maxExpiredEntries = 100;

    /**
     * Creates a new LRUCache instance with the specified options.
     * @param options - Configuration options for the cache
     */
    constructor(options: LRUCacheOptions) {
        if (options.limit <= 0) {
            throw new Error("LRUCache limit must be > 0");
        }
        this.limit = options.limit;
        this.maxSizeBytes = options.maxSizeBytes ?? null;
        this.defaultTTLMs = options.defaultTTLMs != null && options.defaultTTLMs > 0
            ? options.defaultTTLMs
            : null;
        this.cache = new Map();
        this.cleanupCooldownMs = options.cleanupCooldownMs ?? SECONDS_1_MS;
        this.maxExpiredEntries = options.maxExpiredEntries ?? 100;
    }

    /**
     * Checks if a key exists in the cache without affecting its LRU position.
     * If the entry has expired, it is removed and false is returned.
     * @param key - The key to check for.
     * @returns true if the key exists and is not expired, false otherwise.
     */
    has(key: KeyType): boolean {
        this.cleanupExpired();

        const entry = this.cache.get(key);
        if (!entry) return false;

        // Check if expired
        if (entry.expiresAt !== null && Date.now() > entry.expiresAt) {
            this.cache.delete(key);
            this.removeFromExpirationQueue(key);
            return false;
        }

        return true;
    }

    /**
     * Retrieves the value for the given key, renewing its LRU position.
     * If the entry has expired, it is removed and undefined is returned.
     * Amortized O(1), but does a full expired-entry cleanup.
     * @param key - The key to look up.
     */
    get(key: KeyType): ValueType | undefined {
        this.cleanupExpired();

        const entry = this.cache.get(key);
        if (!entry) return undefined;

        // Check if expired
        if (entry.expiresAt !== null && Date.now() > entry.expiresAt) {
            this.cache.delete(key);
            this.removeFromExpirationQueue(key);
            return undefined;
        }

        // Renew LRU position
        this.cache.delete(key);
        this.cache.set(key, entry);

        return entry.value;
    }

    /**
     * Sets a value for the given key, with an optional TTL (in ms).
     * If the cache is full, evicts the least recently used entry.
     * If maxSizeBytes is set and the value is too large, insertion is skipped.
     * Amortized O(1), but may do a full expired-entry cleanup (O(n)).
     * @param key   - The cache key.
     * @param value - The value to store.
     * @param ttl   - Optional TTL in milliseconds (<=0 means no expiration; overrides defaultTTLMs).
     */
    set(key: KeyType, value: ValueType, ttl?: number): void {
        const now = Date.now();

        // Determine actual TTL
        let expiresAt: number | null = null;
        const effectiveTTL = ttl != null
            ? (ttl > 0 ? ttl : null)
            : this.defaultTTLMs;
        if (effectiveTTL != null) {
            expiresAt = now + effectiveTTL;
        }

        // Compute size and enforce maxSizeBytes
        let entrySize = 0;
        if (this.maxSizeBytes != null) {
            entrySize = this.calculateValueSize(value);
            if (entrySize > this.maxSizeBytes) {
                console.warn(
                    `Skipping cache set for key ${String(key)}: ` +
                    `value size ${entrySize} bytes exceeds maxSizeBytes ${this.maxSizeBytes}`,
                );
                return;
            }
        }

        // Update expiration queue if previously existed
        if (this.cache.has(key)) {
            this.removeFromExpirationQueue(key);
            this.cache.delete(key);
        } else {
            // If we are at the limit (or over, though not expected), we need to evict.
            if (this.cache.size >= this.limit) {
                const lruKey = this.cache.keys().next().value;
                if (lruKey !== undefined) {
                    this.cache.delete(lruKey); // Delete from cache first
                    this.removeFromExpirationQueue(lruKey); // Then update expiration structures
                }
            }
            // After potentially making space, call cleanupExpired.
            // It will use its internal cooldowns/thresholds unless a previous operation
            // (like the one above if it decided to force) already did a recent cleanup.
            // Passing `this.cache.size >= this.limit` as force argument here might be too aggressive
            // if we just evicted. Let cleanupExpired decide based on its own state.
            this.cleanupExpired();
        }

        // Add to expiration queue if it has an expiration
        if (expiresAt !== null) {
            this.addToExpirationQueue(key, expiresAt);
        }

        this.cache.set(key, { value, size: entrySize, expiresAt });
    }

    /**
     * Returns an iterator over [key, value] pairs in LRU order (oldest first),
     * skipping expired entries.
     */
    *entries(): IterableIterator<[KeyType, ValueType]> {
        this.cleanupExpired();
        const now = Date.now();
        for (const [key, entry] of this.cache.entries()) {
            // Skip expired entries that might have expired during iteration
            if (entry.expiresAt !== null && entry.expiresAt <= now) {
                // Remove expired entry found during iteration
                this.cache.delete(key);
                this.removeFromExpirationQueue(key);
                continue;
            }
            yield [key, entry.value];
        }
    }

    /**
     * Returns the current number of non-expired items.
     */
    size(): number {
        this.cleanupExpired();
        return this.cache.size;
    }

    /**
     * Deletes a single key.
     * @returns true if the key was present and deleted.
     */
    delete(key: KeyType): boolean {
        if (this.cache.has(key)) {
            this.removeFromExpirationQueue(key);
            return this.cache.delete(key);
        }
        return false;
    }

    /**
     * Clears all entries.
     */
    clear(): void {
        this.cache.clear();
        this.expirationQueue = [];
        this.expirationMap.clear();
    }

    /**
     * Force a complete cleanup of expired entries and maintain queue consistency
     */
    forceCleanup(): void {
        this.cleanupExpired(true);
    }

    /**
     * Remove all expired entries from the cache.
     * Uses lazy cleaning of the expiration queue for more efficient operations.
     * @param force Force cleanup regardless of cooldown
     */
    private cleanupExpired(force = false): void {
        const now = Date.now();
        const expiredCount = this.countExpiredEntries(now);

        // Skip cleanup if we've done it recently and not forcing cleanup
        // and haven't reached max expired entries threshold
        if (!force &&
            now - this.lastCleanupTime < this.cleanupCooldownMs &&
            expiredCount < this.maxExpiredEntries) {
            return;
        }

        this.lastCleanupTime = now;

        if (expiredCount === 0) {
            return; // No expired entries to clean up
        }

        // Fast path: if all entries in the expiration queue are expired, clear them efficiently.
        // This condition implies expiredCount > 0 because if expiredCount was 0, we would have returned above.
        if (expiredCount === this.expirationQueue.length) {
            // All items currently in expirationQueue are confirmed expired.
            // Remove them from the main cache.
            for (let i = 0; i < expiredCount; i++) {
                const [_expiresAt, keyToClear] = this.expirationQueue[i];
                this.cache.delete(keyToClear);
                // No need to individually delete from expirationMap as it's about to be cleared.
            }
            this.expirationQueue = []; // Clear the queue
            this.expirationMap.clear(); // Clear the map
            return;
        }

        // Create a new array for the cleaned queue, only taking the unexpired entries
        const newQueue: [number, KeyType][] = [];

        // Since the queue is sorted by expiration time, we can optimize by
        // processing only what we need
        let i = 0;
        // Process expired entries first (we know exactly how many from countExpiredEntries)
        for (; i < expiredCount; i++) {
            const [expiresAt, key] = this.expirationQueue[i];
            // Delete from cache if it exists and has this expiration time
            if (this.cache.has(key) &&
                this.expirationMap.has(key) &&
                this.expirationMap.get(key) === expiresAt) {
                this.cache.delete(key);
            }
            this.expirationMap.delete(key);
        }

        // Keep remaining unexpired entries, but filter out any stale references
        for (; i < this.expirationQueue.length; i++) {
            const [expiresAt, key] = this.expirationQueue[i];
            // Keep valid entries where the expiration time matches
            if (this.expirationMap.has(key) && this.expirationMap.get(key) === expiresAt) {
                newQueue.push([expiresAt, key]);
            }
        }

        // Replace the old queue with the cleaned queue
        this.expirationQueue = newQueue;
    }

    /**
     * Count the number of expired entries (approximation)
     */
    private countExpiredEntries(now = Date.now()): number {
        // Count expired entries at the beginning of the queue
        let count = 0;
        for (let i = 0; i < this.expirationQueue.length; i++) {
            if (this.expirationQueue[i][0] <= now) {
                count++;
            } else {
                break; // Queue is sorted, so we can stop when we hit a non-expired entry
            }
        }
        return count;
    }

    /**
     * Adds a key to the expiration queue in sorted order.
     */
    private addToExpirationQueue(key: KeyType, expiresAt: number): void {
        // Store the expiration time in the map
        this.expirationMap.set(key, expiresAt);

        // Use binary search to find the correct insertion point
        let low = 0;
        let high = this.expirationQueue.length;

        while (low < high) {
            const mid = Math.floor((low + high) / 2);
            if (this.expirationQueue[mid][0] < expiresAt) {
                low = mid + 1;
            } else {
                high = mid;
            }
        }

        // Insert at the correct position to maintain sorted order
        this.expirationQueue.splice(low, 0, [expiresAt, key]);
    }

    /**
     * Removes a key from the expiration queue.
     */
    private removeFromExpirationQueue(key: KeyType): void {
        const oldExpiresAt = this.expirationMap.get(key); // Get expiration time before deleting from map

        const successfullyRemovedFromMap = this.expirationMap.delete(key);

        if (successfullyRemovedFromMap && oldExpiresAt !== undefined) {
            // Key was in expirationMap and had a defined expiration time.
            // Attempt to remove its corresponding entry from the expirationQueue array.
            // This is O(N) for findIndex + O(N) for splice in worst case.
            const index = this.expirationQueue.findIndex(
                (item) => item[1] === key && item[0] === oldExpiresAt,
            );
            if (index !== -1) {
                this.expirationQueue.splice(index, 1);
            }
        }

        // Original heuristic conditions for triggering a full forceCleanup.
        // These can be kept as a safeguard or for scenarios where the queue might become
        // generally unhealthy or bloated for reasons beyond single-item removal,
        // especially if direct removal via splice proves too slow under certain loads
        // and is conditionally disabled.

        let needsForceCleanup = false;
        // Check if a key was indeed part of the expiration mechanism and if the queue might be bloated.
        if (successfullyRemovedFromMap &&
            this.expirationQueue.length > Math.max(this.limit, this.cache.size + this.maxExpiredEntries)) {
            // Condition 1: A key involved in expiration was removed, and queue is much larger 
            // than cache size or limit, suggesting potential bloat.
            needsForceCleanup = true;
        } else if (this.expirationQueue.length > this.limit * 2) {
            // Condition 2: Queue is simply too large compared to the limit, regardless of recent removal.
            needsForceCleanup = true;
        }

        if (needsForceCleanup) {
            // It's possible the queue contains stale entries not caught by direct removal,
            // or has grown disproportionately. A full cleanup will resync it based on expirationMap.
            this.forceCleanup();
        }
    }

    /**
     * Calculates the approximate byte size of a value (UTF-8).
     */
    private calculateValueSize(value: ValueType): number {
        let str: string;
        if (typeof value === "string") {
            str = value;
        } else {
            try {
                str = JSON.stringify(value);
            } catch (error) {
                console.warn(`Error calculating size for cache value: ${error instanceof Error ? error.message : String(error)}`);
                // Fallback to a string representation that won't throw
                str = String(value);
            }
        }
        // UTF-8 byte length
        return this.textEncoder.encode(str).length;
    }
}

/** Internal cache entry type */
interface CacheEntry<ValueType> {
    value: ValueType;
    size: number;
    expiresAt: number | null;
}

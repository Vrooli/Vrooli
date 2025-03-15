/**
 * A Least Recently Used (LRU) cache implementation that stores key-value pairs with a size limit.
 * When the cache reaches the specified limit, it evicts the least recently used entries.
 */
export class LRUCache<KeyType, ValueType> {
    private limit: number;
    private maxSizeBytes: number | null; // maxSizeBytes in bytes, null means no limit
    private cache: Map<KeyType, ValueType>;

    /**
     * Constructs a new LRUCache.
     * @param limit - Maximum number of items allowed in the cache.
     * @param maxSizeBytes - Optional maximum size in bytes for each value, or null for no size limit.
     */
    constructor(limit: number, maxSizeBytes: number | null = null) {
        this.limit = limit;
        this.maxSizeBytes = maxSizeBytes;
        this.cache = new Map<KeyType, ValueType>();
    }

    /**
     * Retrieves the value associated with the given key, if present.
     * If the key is found, this method renews its position as most recently used.
     * @param key - The key to retrieve from the cache.
     * @returns The value associated with the key, or undefined if not found.
     */
    get(key: KeyType): ValueType | undefined {
        const value = this.cache.get(key);
        if (value !== undefined) {
            this.cache.delete(key); // Remove the key and then set it to renew its position (most recently used)
            this.cache.set(key, value);
            return value;
        }
        return undefined;
    }

    /**
     * Sets a value for a given key in the cache.
     * If the cache exceeds the limit, the least recently used item is evicted.
     * If maxSizeBytes is set and the value's size exceeds this limit, the value is not stored.
     * @param key - The key associated with the value.
     * @param value - The value to store in the cache.
     */
    set(key: KeyType, value: ValueType): void {
        if (this.maxSizeBytes != null) {
            const size = this.calculateValueSize(value);
            if (size > this.maxSizeBytes) {
                // Skip setting the value if it's larger than the maxSizeBytes
                console.warn(`Skipping cache set for key ${key}: value size ${size} exceeds maxSizeBytes ${this.maxSizeBytes}`);
                return;
            }
        }
        if (this.cache.has(key)) {
            this.cache.delete(key); // Remove the key if it already exists so that we can reset its position
        } else if (this.cache.size === this.limit) {
            // Remove the least recently used (first) item in the cache
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        this.cache.set(key, value);
    }

    /**
     * Calculates the size of a value in bytes.
     * For strings, it uses Blob size. For other types, it uses the length of the JSON string representation.
     * @param value - The value whose size is to be calculated.
     * @returns The size of the value in bytes.
     */
    private calculateValueSize(value: ValueType): number {
        if (typeof value === "string") {
            return new Blob([value]).size;
        } else {
            // Might need a more complex calculation in the future
            return JSON.stringify(value).length;
        }
    }

    /**
     * Returns the current number of items in the cache.
     * @returns The number of items currently stored in the cache.
     */
    size(): number {
        return this.cache.size;
    }

    /**
     * @returns All key-value pairs in the cache.
     */
    entries(): IterableIterator<[KeyType, ValueType]> {
        return this.cache.entries();
    }

    /**
     * Clears the cache of all key-value pairs.
     */
    clear(): void {
        this.cache.clear();
    }
}

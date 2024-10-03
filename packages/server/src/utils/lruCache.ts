export class LRUCache<KeyType, ValueType> {
    private limit: number;
    private maxSizeBytes: number | null; // maxSizeBytes in bytes, null means no limit
    private cache: Map<KeyType, ValueType>;

    constructor(limit: number, maxSizeBytes: number | null = null) {
        this.limit = limit;
        this.maxSizeBytes = maxSizeBytes;
        this.cache = new Map<KeyType, ValueType>();
    }

    get(key: KeyType): ValueType | undefined {
        if (this.cache.has(key)) {
            const value = this.cache.get(key)!;
            this.cache.delete(key); // Remove the key and then set it to renew its position (most recently used)
            this.cache.set(key, value);
            return value;
        }
        return undefined;
    }

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

    private calculateValueSize(value: ValueType): number {
        if (typeof value === "string") {
            return new Blob([value]).size;
        } else {
            // Might need a more complex calculation in the future
            return JSON.stringify(value).length;
        }
    }

    size(): number {
        return this.cache.size;
    }
}

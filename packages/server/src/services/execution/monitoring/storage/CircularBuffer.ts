/**
 * Efficient circular buffer implementation for metrics storage.
 * Provides O(1) insertion and automatic eviction of oldest entries.
 */

export class CircularBuffer<T> {
    private buffer: (T | undefined)[];
    private head = 0;
    private tail = 0;
    private count = 0;
    private _totalAdded = 0;
    
    constructor(private readonly capacity: number) {
        if (capacity <= 0) {
            throw new Error("Buffer capacity must be positive");
        }
        this.buffer = new Array(capacity);
    }
    
    /**
     * Add an item to the buffer
     */
    add(item: T): void {
        this.buffer[this.tail] = item;
        this.tail = (this.tail + 1) % this.capacity;
        
        if (this.count < this.capacity) {
            this.count++;
        } else {
            // Buffer is full, move head forward
            this.head = (this.head + 1) % this.capacity;
        }
        
        this._totalAdded++;
    }
    
    /**
     * Add multiple items efficiently
     */
    addBatch(items: T[]): void {
        for (const item of items) {
            this.add(item);
        }
    }
    
    /**
     * Get the most recent N items
     */
    getRecent(n: number): T[] {
        const result: T[] = [];
        const itemsToGet = Math.min(n, this.count);
        
        // Start from the most recent item and work backwards
        let index = (this.tail - 1 + this.capacity) % this.capacity;
        
        for (let i = 0; i < itemsToGet; i++) {
            const item = this.buffer[index];
            if (item !== undefined) {
                result.push(item);
            }
            index = (index - 1 + this.capacity) % this.capacity;
        }
        
        return result;
    }
    
    /**
     * Get all items in the buffer
     */
    getAll(): T[] {
        const result: T[] = [];
        
        if (this.count === 0) return result;
        
        if (this.count < this.capacity) {
            // Buffer not full yet
            for (let i = 0; i < this.count; i++) {
                const item = this.buffer[i];
                if (item !== undefined) {
                    result.push(item);
                }
            }
        } else {
            // Buffer is full, read in circular order
            let index = this.head;
            for (let i = 0; i < this.capacity; i++) {
                const item = this.buffer[index];
                if (item !== undefined) {
                    result.push(item);
                }
                index = (index + 1) % this.capacity;
            }
        }
        
        return result;
    }
    
    /**
     * Find items matching a predicate
     */
    find(predicate: (item: T) => boolean): T[] {
        const result: T[] = [];
        const all = this.getAll();
        
        for (const item of all) {
            if (predicate(item)) {
                result.push(item);
            }
        }
        
        return result;
    }
    
    /**
     * Get items within a time range (assumes items have timestamp property)
     */
    getInTimeRange(
        startTime: Date,
        endTime: Date,
        timestampGetter: (item: T) => Date
    ): T[] {
        return this.find(item => {
            const timestamp = timestampGetter(item);
            return timestamp >= startTime && timestamp <= endTime;
        });
    }
    
    /**
     * Clear the buffer
     */
    clear(): void {
        this.buffer = new Array(this.capacity);
        this.head = 0;
        this.tail = 0;
        this.count = 0;
        // Note: we don't reset _totalAdded as it's a lifetime counter
    }
    
    /**
     * Get current size
     */
    size(): number {
        return this.count;
    }
    
    /**
     * Check if buffer is empty
     */
    isEmpty(): boolean {
        return this.count === 0;
    }
    
    /**
     * Check if buffer is full
     */
    isFull(): boolean {
        return this.count === this.capacity;
    }
    
    /**
     * Get total number of items ever added
     */
    totalAdded(): number {
        return this._totalAdded;
    }
    
    /**
     * Get buffer statistics
     */
    getStats(): {
        capacity: number;
        size: number;
        utilized: number;
        totalAdded: number;
        evicted: number;
    } {
        const evicted = Math.max(0, this._totalAdded - this.capacity);
        
        return {
            capacity: this.capacity,
            size: this.count,
            utilized: this.count / this.capacity,
            totalAdded: this._totalAdded,
            evicted,
        };
    }
}

/**
 * Time-based circular buffer that automatically evicts old entries
 */
export class TimeBasedCircularBuffer<T> extends CircularBuffer<T> {
    private evictionTimer?: NodeJS.Timeout;
    
    constructor(
        capacity: number,
        private readonly maxAgeMs: number,
        private readonly timestampGetter: (item: T) => Date
    ) {
        super(capacity);
        this.startEvictionTimer();
    }
    
    /**
     * Start automatic eviction of old entries
     */
    private startEvictionTimer(): void {
        // Run eviction every 10% of max age
        const interval = Math.max(1000, this.maxAgeMs / 10);
        
        this.evictionTimer = setInterval(() => {
            this.evictOldEntries();
        }, interval);
    }
    
    /**
     * Evict entries older than maxAge
     */
    private evictOldEntries(): void {
        const now = new Date();
        const cutoffTime = new Date(now.getTime() - this.maxAgeMs);
        
        // Get all items and filter out old ones
        const validItems = this.getAll().filter(item => {
            const timestamp = this.timestampGetter(item);
            return timestamp > cutoffTime;
        });
        
        // If we removed items, rebuild the buffer
        if (validItems.length < this.size()) {
            this.clear();
            this.addBatch(validItems);
        }
    }
    
    /**
     * Stop the eviction timer
     */
    stop(): void {
        if (this.evictionTimer) {
            clearInterval(this.evictionTimer);
            this.evictionTimer = undefined;
        }
    }
    
    /**
     * Get items within the valid time window
     */
    getValid(): T[] {
        const now = new Date();
        const cutoffTime = new Date(now.getTime() - this.maxAgeMs);
        
        return this.find(item => {
            const timestamp = this.timestampGetter(item);
            return timestamp > cutoffTime;
        });
    }
}
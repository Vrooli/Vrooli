export class LocalStorageLruCache<ValueType> {
    private limit: number;
    private maxSize: number | null; // maxSize in bytes, null means no limit
    private namespace: string; // Unique identifier for this cache
    private cacheKeys: Array<string>; // Store keys to manage the order for LRU

    constructor(namespace: string, limit: number, maxSize: number | null = null) {
        this.namespace = namespace;
        this.limit = limit;
        this.maxSize = maxSize;
        this.cacheKeys = this.loadKeys();
    }

    private loadKeys(): Array<KeyType> {
        try {
            const keys = localStorage.getItem(this.getNamespacedKey("cacheKeys"));
            if (keys) {
                const parsedKeys = JSON.parse(keys);
                if (Array.isArray(parsedKeys)) {
                    return parsedKeys;
                }
            }
        } catch (error) {
            console.error("Error loading keys from localStorage:", error);
        }
        return [];
    }

    private saveKeys(): void {
        localStorage.setItem(this.getNamespacedKey("cacheKeys"), JSON.stringify(this.cacheKeys));
    }

    private getNamespacedKey(key: string): string {
        return `${this.namespace}:${key}`;
    }

    get(key: string): ValueType | undefined {
        try {
            const serializedValue = localStorage.getItem(this.getNamespacedKey(key));
            if (serializedValue) {
                this.touchKey(key);
                return JSON.parse(serializedValue);
            }
        } catch (error) {
            console.error(`Error parsing value for key ${key} from localStorage:`, error);
        }
        return undefined;
    }

    set(key: string, value: ValueType): void {
        const serializedValue = JSON.stringify(value);
        if (this.maxSize != null && serializedValue.length > this.maxSize) {
            console.warn(`Skipping cache set for key ${key}: value size ${serializedValue.length} exceeds maxSize ${this.maxSize}`);
            return;
        }

        this.touchKey(key);
        localStorage.setItem(this.getNamespacedKey(key), serializedValue);
    }

    remove(key: string): void {
        // Remove the key from the cacheKeys array
        this.removeKey(key);
        // Remove the item from localStorage
        localStorage.removeItem(this.getNamespacedKey(key));
        // Save the updated keys to localStorage
        this.saveKeys();
    }

    removeKeysWithValue(predicate: (key: string, value: ValueType) => boolean): void {
        console.log("in removeKeysWithValue", this.cacheKeys);
        this.cacheKeys.forEach((key) => {
            const value = this.get(key);
            console.log("key and value", key, value);
            if (value !== undefined && predicate(key, value)) {
                console.log("removing key:", key);
                this.remove(key);
            }
        });
    }

    private touchKey(key: string): void {
        this.removeKey(key);
        this.cacheKeys.push(key);
        while (this.cacheKeys.length > this.limit) {
            const oldKey = this.cacheKeys.shift();
            if (oldKey !== undefined) {
                localStorage.removeItem(this.getNamespacedKey(oldKey));
            }
        }
        this.saveKeys();
    }

    private removeKey(key: string): void {
        const index = this.cacheKeys.indexOf(key);
        if (index > -1) {
            this.cacheKeys.splice(index, 1);
        }
    }

    size(): number {
        return this.cacheKeys.length;
    }
}

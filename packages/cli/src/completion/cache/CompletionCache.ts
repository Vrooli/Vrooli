import { promises as fs } from "fs";
import * as path from "path";
import { type ConfigManager } from "../../utils/config.js";
import type { CompletionResult, CompletionCache as ICompletionCache } from "../types.js";
import { CACHE_CONSTANTS, SECONDS_1_MS } from "../../utils/constants.js";

interface CacheEntry {
    results: CompletionResult[];
    timestamp: number;
    ttl: number;
}

export class CompletionCache implements ICompletionCache {
    private cache: Map<string, CacheEntry> = new Map();
    private cacheFile: string;
    private readonly maxCacheSize = CACHE_CONSTANTS.MAX_SIZE;
    private readonly defaultTtl = CACHE_CONSTANTS.DEFAULT_TTL_SECONDS;
    
    constructor(private config: ConfigManager) {
        this.cacheFile = path.join(config.getConfigDir(), "completion-cache.json");
        this.loadCache();
    }
    
    async get(key: string): Promise<CompletionResult[] | null> {
        const entry = this.cache.get(key);
        if (!entry) return null;
        
        if (this.isExpired(entry)) {
            this.cache.delete(key);
            return null;
        }
        
        return entry.results;
    }
    
    async set(key: string, value: CompletionResult[], ttl: number = this.defaultTtl): Promise<void> {
        // Implement LRU eviction if cache is full
        if (this.cache.size >= this.maxCacheSize) {
            this.evictOldest();
        }
        
        const entry: CacheEntry = {
            results: value,
            timestamp: Date.now(),
            ttl: ttl * SECONDS_1_MS, // Convert to milliseconds
        };
        
        this.cache.set(key, entry);
        
        // Save to disk asynchronously
        this.saveCache().catch(() => {
            // Ignore save errors
        });
    }
    
    isExpired(entry: CacheEntry): boolean {
        return Date.now() - entry.timestamp > entry.ttl;
    }
    
    async clear(): Promise<void> {
        this.cache.clear();
        try {
            await fs.unlink(this.cacheFile);
        } catch (error) {
            // Ignore if file doesn't exist
        }
    }
    
    private evictOldest(): void {
        let oldestKey: string | null = null;
        let oldestTimestamp = Number.MAX_SAFE_INTEGER;
        
        for (const [key, entry] of this.cache) {
            if (entry.timestamp < oldestTimestamp) {
                oldestTimestamp = entry.timestamp;
                oldestKey = key;
            }
        }
        
        if (oldestKey) {
            this.cache.delete(oldestKey);
        }
    }
    
    private async loadCache(): Promise<void> {
        try {
            const data = await fs.readFile(this.cacheFile, "utf-8");
            const parsed = JSON.parse(data);
            
            // Validate and load cache entries
            for (const [key, entry] of Object.entries(parsed)) {
                if (this.isValidCacheEntry(entry)) {
                    this.cache.set(key, entry as CacheEntry);
                }
            }
            
            // Clean up expired entries
            this.cleanupExpired();
        } catch (error) {
            // Cache file doesn't exist or is corrupted, start with empty cache
            this.cache.clear();
        }
    }
    
    private async saveCache(): Promise<void> {
        try {
            // Ensure config directory exists
            const configDir = path.dirname(this.cacheFile);
            await fs.mkdir(configDir, { recursive: true });
            
            // Convert Map to object and save
            const cacheObject: Record<string, CacheEntry> = {};
            for (const [key, entry] of this.cache) {
                cacheObject[key] = entry;
            }
            
            await fs.writeFile(this.cacheFile, JSON.stringify(cacheObject, null, 2));
        } catch (error) {
            // Ignore save errors to avoid breaking completions
        }
    }
    
    private isValidCacheEntry(entry: unknown): entry is CacheEntry {
        if (typeof entry !== "object" || entry === null) return false;
        
        const cacheEntry = entry as Record<string, unknown>;
        
        return (
            Array.isArray(cacheEntry.results) &&
            typeof cacheEntry.timestamp === "number" &&
            typeof cacheEntry.ttl === "number"
        );
    }
    
    private cleanupExpired(): void {
        const now = Date.now();
        const expiredKeys: string[] = [];
        
        for (const [key, entry] of this.cache) {
            if (now - entry.timestamp > entry.ttl) {
                expiredKeys.push(key);
            }
        }
        
        for (const key of expiredKeys) {
            this.cache.delete(key);
        }
    }
    
    /**
     * Get cache statistics
     */
    getStats(): { size: number; maxSize: number; hitRate?: number } {
        return {
            size: this.cache.size,
            maxSize: this.maxCacheSize,
        };
    }
    
    /**
     * Cleanup old cache entries periodically
     */
    startCleanupTimer(): void {
        setInterval(() => {
            this.cleanupExpired();
        }, CACHE_CONSTANTS.FLUSH_INTERVAL_MS); // Clean up every minute
    }
}

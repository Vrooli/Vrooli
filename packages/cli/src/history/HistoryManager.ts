import { type ConfigManager } from "../utils/config.js";
import { StorageFactory, type ExtendedHistoryStorage } from "./storage/StorageFactory.js";
import { generatePK } from "@vrooli/shared";
import type { HistoryEntry, HistorySearchQuery, HistoryStats } from "./types.js";
import { CLI_LIMITS, DAYS_90_MS } from "../utils/constants.js";

export class HistoryManager {
    private storage: ExtendedHistoryStorage;
    private currentEntry?: Partial<HistoryEntry>;
    private readonly sensitiveKeys = ["password", "token", "secret", "key", "auth"];
    private storagePromise: Promise<ExtendedHistoryStorage>;
    
    constructor(
        private config: ConfigManager,
        _storageType: "sqlite" | "json" = "sqlite",
    ) {
        // Create storage asynchronously
        this.storagePromise = StorageFactory.create(config).then(storage => {
            this.storage = storage;
            return storage;
        });
        
        // For immediate use in sync contexts (e.g., exit handlers), create a sync fallback
        this.storage = StorageFactory.createSync(config);
    }
    
    /**
     * Ensures storage is initialized before use
     */
    private async ensureStorage(): Promise<ExtendedHistoryStorage> {
        if (this.storagePromise) {
            this.storage = await this.storagePromise;
            this.storagePromise = null as unknown as Promise<ExtendedHistoryStorage>; // Clear to avoid repeated awaits
        }
        return this.storage;
    }
    
    async startCommand(command: string, args: string[], options: Record<string, unknown>): Promise<void> {
        // Check if history tracking is disabled
        if (this.isHistoryDisabled(options)) {
            return;
        }
        
        // Filter sensitive data
        const sanitizedOptions = this.sanitizeOptions(options);
        
        this.currentEntry = {
            id: generatePK().toString(),
            command,
            args,
            options: sanitizedOptions,
            timestamp: new Date(),
            profile: this.config.getActiveProfileName(),
            userId: this.config.getActiveProfile().userId,
        };
    }
    
    async endCommand(exitCode: number, error?: Error): Promise<void> {
        if (!this.currentEntry) return;
        
        const now = Date.now();
        const startTime = this.currentEntry.timestamp?.getTime() ?? now;
        
        this.currentEntry.duration = now - startTime;
        this.currentEntry.exitCode = exitCode;
        this.currentEntry.success = exitCode === 0;
        this.currentEntry.error = error?.message;
        
        // Add completion metadata
        this.addCompletionMetadata();
        
        const storage = await this.ensureStorage();
        await storage.add(this.currentEntry as HistoryEntry);
        this.currentEntry = undefined;
    }

    /**
     * Synchronous version of endCommand for process exit handlers
     */
    endCommandSync(exitCode: number, error?: Error): void {
        if (!this.currentEntry) return;
        
        const now = Date.now();
        const startTime = this.currentEntry.timestamp?.getTime() ?? now;
        
        this.currentEntry.duration = now - startTime;
        this.currentEntry.exitCode = exitCode;
        this.currentEntry.success = exitCode === 0;
        this.currentEntry.error = error?.message;
        
        // Add completion metadata
        this.addCompletionMetadata();
        
        try {
            // Use synchronous call for exit handlers
            if (this.storage.addSync) {
                this.storage.addSync(this.currentEntry as HistoryEntry);
            }
        } catch (error) {
            // Ignore errors during exit
        }
        
        this.currentEntry = undefined;
    }
    
    async addMetadata(key: string, value: unknown): Promise<void> {
        if (!this.currentEntry) return;
        
        if (!this.currentEntry.metadata) {
            this.currentEntry.metadata = {};
        }
        
        this.currentEntry.metadata[key] = value;
    }
    
    async get(id: string): Promise<HistoryEntry | null> {
        const storage = await this.ensureStorage();
        return storage.get(id);
    }
    
    async search(query: HistorySearchQuery = {}): Promise<HistoryEntry[]> {
        const storage = await this.ensureStorage();
        return storage.search(query);
    }
    
    async getStats(): Promise<HistoryStats> {
        const storage = await this.ensureStorage();
        return storage.getStats();
    }
    
    async clear(): Promise<void> {
        const storage = await this.ensureStorage();
        return storage.clear();
    }
    
    async delete(id: string): Promise<void> {
        const storage = await this.ensureStorage();
        return storage.delete(id);
    }
    
    async export(format: "json" | "csv" | "script"): Promise<string> {
        const storage = await this.ensureStorage();
        return storage.export(format);
    }
    
    private isHistoryDisabled(options: Record<string, unknown>): boolean {
        // Check for history disable flags
        if (options.noHistory || options["no-history"] || options.disableHistory || options["disable-history"]) {
            return true;
        }
        
        // Check for sensitive commands that should not be tracked
        const sensitiveCommands = ["auth login"];
        const commandString = `${this.currentEntry?.command} ${this.currentEntry?.args?.join(" ")}`.trim();
        
        return sensitiveCommands.some(cmd => commandString.startsWith(cmd));
    }
    
    private sanitizeOptions(options: Record<string, unknown>): Record<string, unknown> {
        const sanitized = { ...options };
        
        for (const key of Object.keys(sanitized)) {
            if (this.isSensitiveKey(key)) {
                sanitized[key] = "***";
            } else if (typeof sanitized[key] === "object" && sanitized[key] !== null && !Array.isArray(sanitized[key])) {
                // Recursively sanitize nested objects
                sanitized[key] = this.sanitizeOptions(sanitized[key] as Record<string, unknown>);
            }
        }
        
        return sanitized;
    }
    
    private isSensitiveKey(key: string): boolean {
        const lowerKey = key.toLowerCase();
        return this.sensitiveKeys.some(sensitive => lowerKey.includes(sensitive));
    }
    
    private addCompletionMetadata(): void {
        if (!this.currentEntry) return;
        
        // Add basic completion metadata
        if (!this.currentEntry.metadata) {
            this.currentEntry.metadata = {};
        }
        
        // Record command completion time
        this.currentEntry.metadata.completedAt = new Date().toISOString();
        
        // Add command pattern for analysis
        const pattern = `${this.currentEntry.command} ${this.currentEntry.args?.join(" ") || ""}`.trim();
        this.currentEntry.metadata.commandPattern = pattern;
    }
    
    /**
     * Get recent commands for quick access
     */
    async getRecentCommands(limit = CLI_LIMITS.DEFAULT_HISTORY_LIMIT): Promise<HistoryEntry[]> {
        return this.search({
            limit,
            successOnly: true,
        });
    }
    
    /**
     * Get command suggestions based on current input
     */
    async getSuggestions(partial: string, limit = CLI_LIMITS.DEFAULT_SUGGESTIONS_LIMIT): Promise<HistoryEntry[]> {
        const results = await this.search({
            text: partial,
            limit: limit * CLI_LIMITS.SUGGESTION_MULTIPLIER, // Get more to filter
            successOnly: true,
        });
        
        // Filter and deduplicate
        const seen = new Set<string>();
        const unique: HistoryEntry[] = [];
        
        for (const entry of results) {
            const commandStr = `${entry.command} ${entry.args.join(" ")}`.trim();
            if (!seen.has(commandStr) && commandStr.includes(partial)) {
                seen.add(commandStr);
                unique.push(entry);
                if (unique.length >= limit) break;
            }
        }
        
        return unique;
    }
    
    /**
     * Get frequently used commands
     */
    async getFrequentCommands(limit = CLI_LIMITS.DEFAULT_HISTORY_LIMIT): Promise<Array<{ command: string; count: number; lastUsed: Date }>> {
        const stats = await this.getStats();
        const topCommands = stats.topCommands.slice(0, limit);
        
        const result: Array<{ command: string; count: number; lastUsed: Date }> = [];
        
        for (const cmd of topCommands) {
            const recent = await this.search({
                command: cmd.command,
                limit: 1,
            });
            
            if (recent.length > 0) {
                result.push({
                    command: cmd.command,
                    count: cmd.count,
                    lastUsed: recent[0].timestamp,
                });
            }
        }
        
        return result;
    }
    
    /**
     * Get command completion patterns for analysis
     */
    async getCommandPatterns(): Promise<Record<string, { count: number; avgDuration: number; successRate: number }>> {
        const entries = await this.search({ limit: CLI_LIMITS.EXPORT_PREVIEW_LENGTH });
        const patterns: Record<string, { total: number; successful: number; totalDuration: number }> = {};
        
        for (const entry of entries) {
            const pattern = `${entry.command} ${entry.args.join(" ")}`.trim();
            
            if (!patterns[pattern]) {
                patterns[pattern] = { total: 0, successful: 0, totalDuration: 0 };
            }
            
            patterns[pattern].total++;
            if (entry.success) {
                patterns[pattern].successful++;
            }
            if (entry.duration) {
                patterns[pattern].totalDuration += entry.duration;
            }
        }
        
        const result: Record<string, { count: number; avgDuration: number; successRate: number }> = {};
        
        for (const [pattern, stats] of Object.entries(patterns)) {
            result[pattern] = {
                count: stats.total,
                avgDuration: stats.totalDuration / stats.total,
                successRate: stats.successful / stats.total,
            };
        }
        
        return result;
    }
    
    /**
     * Clean up old history entries
     */
    async cleanup(options: { olderThan?: Date; keepCount?: number } = {}): Promise<void> {
        const { olderThan = new Date(Date.now() - DAYS_90_MS), keepCount = CLI_LIMITS.EXPORT_PREVIEW_LENGTH } = options;
        
        // This would need to be implemented in the storage layer
        // For now, we'll just note that cleanup is available
        console.log(`Cleanup would remove entries older than ${olderThan.toISOString()}, keeping at least ${keepCount} entries`);
    }
    
    /**
     * Get history for a specific time period
     */
    async getHistoryForPeriod(start: Date, end: Date): Promise<HistoryEntry[]> {
        return this.search({
            startDate: start,
            endDate: end,
            limit: CLI_LIMITS.EXPORT_PREVIEW_LENGTH * CLI_LIMITS.DEFAULT_HISTORY_LIMIT, // Large limit for period queries
        });
    }
}

import * as fs from "fs";
import * as path from "path";
import type { ConfigManager } from "../../utils/config.js";
import type { HistoryEntry, HistorySearchQuery, HistoryStats, HistoryStorage } from "../types.js";
import { DAYS_30_MS, HISTORY_CONSTANTS } from "../../utils/constants.js";

const MAX_ENTRIES = 10000; // Limit entries to prevent file from growing too large
const WRITE_RETRY_COUNT = 3;
const WRITE_RETRY_DELAY = 100;
const TOP_COMMANDS_LIMIT = 10;

interface JsonHistoryData {
    version: number;
    entries: HistoryEntry[];
}

export class JsonStorage implements HistoryStorage {
    private readonly filePath: string;
    private data: JsonHistoryData;
    private writeQueue: Promise<void> = Promise.resolve();
    
    constructor(private config: ConfigManager) {
        this.filePath = path.join(config.getConfigDir(), "history.json");
        this.data = this.loadData();
    }
    
    private loadData(): JsonHistoryData {
        try {
            if (fs.existsSync(this.filePath)) {
                const content = fs.readFileSync(this.filePath, "utf-8");
                const data = JSON.parse(content) as JsonHistoryData;
                
                // Convert date strings back to Date objects
                data.entries = data.entries.map(entry => ({
                    ...entry,
                    timestamp: new Date(entry.timestamp),
                }));
                
                return data;
            }
        } catch (error) {
            console.warn("Failed to load history file, starting with empty history:", error);
        }
        
        return {
            version: 1,
            entries: [],
        };
    }
    
    private async saveData(): Promise<void> {
        const tempPath = `${this.filePath}.tmp`;
        
        // Limit entries to MAX_ENTRIES most recent
        if (this.data.entries.length > MAX_ENTRIES) {
            this.data.entries = this.data.entries
                .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
                .slice(0, MAX_ENTRIES);
        }
        
        const dataToSave = JSON.stringify(this.data, null, 2);
        
        // Atomic write with retries
        for (let attempt = 0; attempt < WRITE_RETRY_COUNT; attempt++) {
            try {
                // Ensure directory exists
                const dir = path.dirname(this.filePath);
                if (!fs.existsSync(dir)) {
                    fs.mkdirSync(dir, { recursive: true });
                }
                
                // Write to temp file
                fs.writeFileSync(tempPath, dataToSave, "utf-8");
                
                // Atomic rename
                fs.renameSync(tempPath, this.filePath);
                
                return;
            } catch (error) {
                if (attempt === WRITE_RETRY_COUNT - 1) {
                    throw error;
                }
                await new Promise(resolve => setTimeout(resolve, WRITE_RETRY_DELAY));
            }
        }
    }
    
    private queueWrite(): void {
        this.writeQueue = this.writeQueue.then(() => this.saveData()).catch(error => {
            console.error("Failed to save history:", error);
        });
    }
    
    async add(entry: HistoryEntry): Promise<void> {
        this.data.entries.push(entry);
        this.queueWrite();
    }
    
    /**
     * Synchronous version of add for process exit handlers
     */
    addSync(entry: HistoryEntry): void {
        this.data.entries.push(entry);
        try {
            // Synchronous save for exit handlers
            const tempPath = `${this.filePath}.tmp`;
            const dir = path.dirname(this.filePath);
            
            if (!fs.existsSync(dir)) {
                fs.mkdirSync(dir, { recursive: true });
            }
            
            fs.writeFileSync(tempPath, JSON.stringify(this.data, null, 2), "utf-8");
            fs.renameSync(tempPath, this.filePath);
        } catch (error) {
            // Best effort - don't throw on exit
            console.error("Failed to save history on exit:", error);
        }
    }
    
    async get(id: string): Promise<HistoryEntry | null> {
        const entry = this.data.entries.find(e => e.id === id);
        return entry || null;
    }
    
    async search(query: HistorySearchQuery): Promise<HistoryEntry[]> {
        let results = [...this.data.entries];
        
        // Apply filters
        if (query.text) {
            const searchText = query.text.toLowerCase();
            results = results.filter(entry => 
                entry.command.toLowerCase().includes(searchText) ||
                entry.args.some(arg => arg.toLowerCase().includes(searchText)),
            );
        }
        
        if (query.command) {
            results = results.filter(entry => entry.command === query.command);
        }
        
        if (query.profile) {
            results = results.filter(entry => entry.profile === query.profile);
        }
        
        if (query.userId) {
            results = results.filter(entry => entry.userId === query.userId);
        }
        
        if (query.successOnly) {
            results = results.filter(entry => entry.success);
        }
        
        if (query.failedOnly) {
            results = results.filter(entry => !entry.success);
        }
        
        if (query.startDate) {
            const startTime = query.startDate.getTime();
            results = results.filter(entry => entry.timestamp.getTime() >= startTime);
        }
        
        if (query.endDate) {
            const endTime = query.endDate.getTime();
            results = results.filter(entry => entry.timestamp.getTime() <= endTime);
        }
        
        if (query.before) {
            const beforeTime = query.before.getTime();
            results = results.filter(entry => entry.timestamp.getTime() < beforeTime);
        }
        
        if (query.after) {
            const afterTime = query.after.getTime();
            results = results.filter(entry => entry.timestamp.getTime() > afterTime);
        }
        
        // Sort by timestamp descending
        results.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
        
        // Apply pagination
        const offset = query.offset || 0;
        const limit = query.limit || results.length;
        results = results.slice(offset, offset + limit);
        
        return results;
    }
    
    async getStats(): Promise<HistoryStats> {
        const entries = this.data.entries;
        const successful = entries.filter(e => e.success);
        
        // Calculate top commands
        const commandCounts = new Map<string, number>();
        entries.forEach(entry => {
            commandCounts.set(entry.command, (commandCounts.get(entry.command) || 0) + 1);
        });
        
        const topCommands = Array.from(commandCounts.entries())
            .map(([command, count]) => ({ command, count }))
            .sort((a, b) => b.count - a.count)
            .slice(0, TOP_COMMANDS_LIMIT);
        
        // Calculate commands by profile
        const profileCounts: Record<string, number> = {};
        entries.forEach(entry => {
            profileCounts[entry.profile] = (profileCounts[entry.profile] || 0) + 1;
        });
        
        // Calculate recent activity (last 30 days)
        const thirtyDaysAgo = Date.now() - DAYS_30_MS;
        const recentEntries = entries.filter(e => e.timestamp.getTime() > thirtyDaysAgo);
        
        const activityByDate = new Map<string, number>();
        recentEntries.forEach(entry => {
            const dateStr = entry.timestamp.toISOString().split("T")[0];
            activityByDate.set(dateStr, (activityByDate.get(dateStr) || 0) + 1);
        });
        
        const recentActivity = Array.from(activityByDate.entries())
            .map(([date, count]) => ({ date, count }))
            .sort((a, b) => b.date.localeCompare(a.date))
            .slice(0, HISTORY_CONSTANTS.RECENT_ACTIVITY_LIMIT);
        
        // Calculate average duration
        const durationsMs = entries
            .filter(e => e.duration !== undefined)
            .map(e => e.duration as number);
        const avgDuration = durationsMs.length > 0
            ? durationsMs.reduce((sum, d) => sum + d, 0) / durationsMs.length
            : 0;
        
        // Find last command
        const sortedByTime = [...entries].sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
        const lastCommand = sortedByTime[0]?.timestamp || new Date(0);
        
        return {
            totalCommands: entries.length,
            successfulCommands: successful.length,
            failedCommands: entries.length - successful.length,
            uniqueCommands: commandCounts.size,
            avgDuration,
            lastCommand,
            topCommands,
            commandsByProfile: profileCounts,
            recentActivity,
        };
    }
    
    async clear(): Promise<void> {
        this.data.entries = [];
        this.queueWrite();
    }
    
    async delete(id: string): Promise<void> {
        const index = this.data.entries.findIndex(e => e.id === id);
        if (index !== -1) {
            this.data.entries.splice(index, 1);
            this.queueWrite();
        }
    }
    
    async export(format: "json" | "csv" | "script"): Promise<string> {
        const entries = await this.search({ limit: 10000 });
        
        switch (format) {
            case "json":
                return JSON.stringify(entries, null, 2);
                
            case "csv":
                return this.exportToCsv(entries);
                
            case "script":
                return this.exportToScript(entries);
                
            default:
                throw new Error(`Unsupported export format: ${format}`);
        }
    }
    
    private exportToCsv(entries: HistoryEntry[]): string {
        const headers = ["id", "command", "args", "timestamp", "duration", "exitCode", "profile", "success", "error"];
        const rows = entries.map(entry => [
            entry.id,
            entry.command,
            entry.args.join(" "),
            entry.timestamp.toISOString(),
            entry.duration || "",
            entry.exitCode || "",
            entry.profile,
            entry.success ? "true" : "false",
            entry.error || "",
        ]);
        
        return [headers, ...rows].map(row => 
            row.map(cell => `"${String(cell).replace(/"/g, "\"\"")}"`).join(","),
        ).join("\\n");
    }
    
    private exportToScript(entries: HistoryEntry[]): string {
        const successfulEntries = entries.filter(e => e.success);
        const commands = successfulEntries.map(entry => 
            `${entry.command} ${entry.args.join(" ")}`.trim(),
        );
        
        return [
            "#!/bin/bash",
            "# Generated by Vrooli CLI history export",
            `# Exported on ${new Date().toISOString()}`,
            "# Contains successful commands only",
            "",
            ...commands,
        ].join("\\n");
    }
    
    /**
     * Close any resources (no-op for JSON storage)
     */
    close(): void {
        // Ensure any pending writes complete synchronously
        try {
            this.saveData();
        } catch (error) {
            console.error("Failed to save on close:", error);
        }
    }
    
    /**
     * Get storage info
     */
    getDbInfo(): { path: string; size: number; pageCount: number } {
        try {
            const stats = fs.statSync(this.filePath);
            return {
                path: this.filePath,
                size: stats.size,
                pageCount: 1, // Single file
            };
        } catch {
            return {
                path: this.filePath,
                size: 0,
                pageCount: 0,
            };
        }
    }
    
    /**
     * No-op for JSON storage
     */
    vacuum(): void {
        // Already compact - we limit to MAX_ENTRIES
    }
    
    /**
     * No-op for JSON storage
     */
    analyze(): void {
        // No query optimization needed
    }
}

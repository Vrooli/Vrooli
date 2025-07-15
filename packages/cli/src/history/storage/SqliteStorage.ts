import Database from "better-sqlite3";
import * as path from "path";
import type { ConfigManager } from "../../utils/config.js";
import type { HistoryEntry, HistorySearchQuery, HistoryStats, HistoryStorage } from "../types.js";
import { DAYS_30_MS, HISTORY_CONSTANTS } from "../../utils/constants.js";

interface DatabaseRow {
    id: string;
    command: string;
    args: string;
    options: string;
    timestamp: number;
    duration: number | null;
    exit_code: number | null;
    profile: string;
    user_id: string | null;
    success: number;
    error: string | null;
    metadata: string | null;
}

interface StatsRow {
    total: number;
    successful: number;
    unique_commands: number;
    avg_duration: number;
    last_command: number;
}

interface CommandCountRow {
    command: string;
    count: number;
}

interface ProfileCountRow {
    profile: string;
    count: number;
}

interface ActivityRow {
    date: string;
    count: number;
}

export class SqliteStorage implements HistoryStorage {
    private db: Database.Database;
    
    constructor(private config: ConfigManager) {
        const dbPath = path.join(config.getConfigDir(), "history.db");
        this.db = new Database(dbPath);
        this.initialize();
    }
    
    private initialize(): void {
        this.db.exec(`
            CREATE TABLE IF NOT EXISTS history (
                id TEXT PRIMARY KEY,
                command TEXT NOT NULL,
                args TEXT NOT NULL,
                options TEXT NOT NULL,
                timestamp INTEGER NOT NULL,
                duration INTEGER,
                exit_code INTEGER,
                profile TEXT NOT NULL,
                user_id TEXT,
                success INTEGER NOT NULL,
                error TEXT,
                metadata TEXT,
                created_at INTEGER DEFAULT (strftime('%s', 'now'))
            );
            
            CREATE INDEX IF NOT EXISTS idx_timestamp ON history(timestamp DESC);
            CREATE INDEX IF NOT EXISTS idx_command ON history(command);
            CREATE INDEX IF NOT EXISTS idx_success ON history(success);
            CREATE INDEX IF NOT EXISTS idx_profile ON history(profile);
            CREATE INDEX IF NOT EXISTS idx_user_id ON history(user_id);
            CREATE INDEX IF NOT EXISTS idx_created_at ON history(created_at);
            
            -- Full text search
            CREATE VIRTUAL TABLE IF NOT EXISTS history_fts USING fts5(
                command, args, content=history, content_rowid=rowid
            );
            
            -- Triggers to keep FTS in sync
            CREATE TRIGGER IF NOT EXISTS history_ai AFTER INSERT ON history BEGIN
                INSERT INTO history_fts(rowid, command, args) 
                VALUES (new.rowid, new.command, new.args);
            END;
            
            CREATE TRIGGER IF NOT EXISTS history_au AFTER UPDATE ON history BEGIN
                UPDATE history_fts SET command = new.command, args = new.args
                WHERE rowid = new.rowid;
            END;
            
            CREATE TRIGGER IF NOT EXISTS history_ad AFTER DELETE ON history BEGIN
                DELETE FROM history_fts WHERE rowid = old.rowid;
            END;
        `);
    }
    
    async add(entry: HistoryEntry): Promise<void> {
        this.addSync(entry);
    }

    /**
     * Synchronous version of add for process exit handlers
     */
    addSync(entry: HistoryEntry): void {
        const stmt = this.db.prepare(`
            INSERT INTO history (
                id, command, args, options, timestamp, duration,
                exit_code, profile, user_id, success, error, metadata
            ) VALUES (
                ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
            )
        `);
        
        stmt.run(
            entry.id,
            entry.command,
            JSON.stringify(entry.args),
            JSON.stringify(entry.options),
            entry.timestamp.getTime(),
            entry.duration || null,
            entry.exitCode || null,
            entry.profile,
            entry.userId || null,
            entry.success ? 1 : 0,
            entry.error || null,
            entry.metadata ? JSON.stringify(entry.metadata) : null,
        );
    }
    
    async get(id: string): Promise<HistoryEntry | null> {
        const stmt = this.db.prepare(`
            SELECT * FROM history WHERE id = ?
        `);
        
        const row = stmt.get(id);
        return row ? this.rowToEntry(row) : null;
    }
    
    async search(query: HistorySearchQuery): Promise<HistoryEntry[]> {
        let sql = "SELECT * FROM history WHERE 1=1";
        const params: unknown[] = [];
        
        if (query.text) {
            sql += " AND rowid IN (SELECT rowid FROM history_fts WHERE history_fts MATCH ?)";
            params.push(query.text);
        }
        
        if (query.command) {
            sql += " AND command = ?";
            params.push(query.command);
        }
        
        if (query.profile) {
            sql += " AND profile = ?";
            params.push(query.profile);
        }
        
        if (query.userId) {
            sql += " AND user_id = ?";
            params.push(query.userId);
        }
        
        if (query.successOnly) {
            sql += " AND success = 1";
        }
        
        if (query.failedOnly) {
            sql += " AND success = 0";
        }
        
        if (query.startDate) {
            sql += " AND timestamp >= ?";
            params.push(query.startDate.getTime());
        }
        
        if (query.endDate) {
            sql += " AND timestamp <= ?";
            params.push(query.endDate.getTime());
        }
        
        if (query.before) {
            sql += " AND timestamp < ?";
            params.push(query.before.getTime());
        }
        
        if (query.after) {
            sql += " AND timestamp > ?";
            params.push(query.after.getTime());
        }
        
        sql += " ORDER BY timestamp DESC";
        
        if (query.limit) {
            sql += " LIMIT ?";
            params.push(query.limit);
        }
        
        if (query.offset) {
            sql += " OFFSET ?";
            params.push(query.offset);
        }
        
        const stmt = this.db.prepare(sql);
        const rows = stmt.all(...params);
        return rows.map((row: unknown) => this.rowToEntry(row));
    }
    
    async getStats(): Promise<HistoryStats> {
        const basicStats = this.db.prepare(`
            SELECT 
                COUNT(*) as total,
                SUM(success) as successful,
                COUNT(DISTINCT command) as unique_commands,
                AVG(duration) as avg_duration,
                MAX(timestamp) as last_command
            FROM history
        `).get() as StatsRow;
        
        const topCommands = this.db.prepare(`
            SELECT command, COUNT(*) as count
            FROM history
            GROUP BY command
            ORDER BY count DESC
            LIMIT 10
        `).all() as CommandCountRow[];
        
        const commandsByProfile = this.db.prepare(`
            SELECT profile, COUNT(*) as count
            FROM history
            GROUP BY profile
        `).all() as ProfileCountRow[];
        
        const recentActivity = this.db.prepare(`
            SELECT 
                date(timestamp/1000, 'unixepoch') as date,
                COUNT(*) as count
            FROM history
            WHERE timestamp > ?
            GROUP BY date(timestamp/1000, 'unixepoch')
            ORDER BY date DESC
            LIMIT ${HISTORY_CONSTANTS.RECENT_ACTIVITY_LIMIT}
        `).all(Date.now() - DAYS_30_MS) as ActivityRow[];
        
        const profileStats: Record<string, number> = {};
        commandsByProfile.forEach(row => {
            profileStats[row.profile] = row.count;
        });
        
        return {
            totalCommands: basicStats.total || 0,
            successfulCommands: basicStats.successful || 0,
            failedCommands: (basicStats.total || 0) - (basicStats.successful || 0),
            uniqueCommands: basicStats.unique_commands || 0,
            avgDuration: basicStats.avg_duration || 0,
            lastCommand: basicStats.last_command ? new Date(basicStats.last_command) : new Date(0),
            topCommands: topCommands.map(row => ({
                command: row.command,
                count: row.count,
            })),
            commandsByProfile: profileStats,
            recentActivity: recentActivity.map(row => ({
                date: row.date,
                count: row.count,
            })),
        };
    }
    
    async clear(): Promise<void> {
        this.db.exec("DELETE FROM history");
    }
    
    async delete(id: string): Promise<void> {
        const stmt = this.db.prepare("DELETE FROM history WHERE id = ?");
        stmt.run(id);
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
    
    private rowToEntry(row: unknown): HistoryEntry {
        const dbRow = row as DatabaseRow;
        return {
            id: dbRow.id,
            command: dbRow.command,
            args: JSON.parse(dbRow.args),
            options: JSON.parse(dbRow.options),
            timestamp: new Date(dbRow.timestamp),
            duration: dbRow.duration ?? undefined,
            exitCode: dbRow.exit_code ?? undefined,
            profile: dbRow.profile,
            userId: dbRow.user_id ?? undefined,
            success: dbRow.success === 1,
            error: dbRow.error ?? undefined,
            metadata: dbRow.metadata ? JSON.parse(dbRow.metadata) : undefined,
        };
    }
    
    /**
     * Close the database connection
     */
    close(): void {
        this.db.close();
    }
    
    /**
     * Get database size information
     */
    getDbInfo(): { path: string; size: number; pageCount: number } {
        const info = this.db.pragma("database_list") as Array<{ name: string; file: string }>;
        const mainDb = info.find(db => db.name === "main");
        
        const pageCount = this.db.pragma("page_count", { simple: true }) as number;
        const pageSize = this.db.pragma("page_size", { simple: true }) as number;
        
        return {
            path: mainDb?.file || "",
            size: pageCount * pageSize,
            pageCount,
        };
    }
    
    /**
     * Vacuum the database to reclaim space
     */
    vacuum(): void {
        this.db.exec("VACUUM");
    }
    
    /**
     * Analyze the database for query optimization
     */
    analyze(): void {
        this.db.exec("ANALYZE");
    }
}

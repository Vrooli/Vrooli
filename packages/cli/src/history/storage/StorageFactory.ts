import chalk from "chalk";
import type { ConfigManager } from "../../utils/config.js";
import type { HistoryEntry, HistoryStorage } from "../types.js";
import { JsonStorage } from "./JsonStorage.js";

export interface ExtendedHistoryStorage extends HistoryStorage {
    addSync?(entry: HistoryEntry): void;
    close?(): void;
    getDbInfo?(): { path: string; size: number; pageCount: number };
    vacuum?(): void;
    analyze?(): void;
}

export class StorageFactory {
    /**
     * Creates a history storage instance, attempting SQLite first, falling back to JSON
     */
    static async create(config: ConfigManager): Promise<ExtendedHistoryStorage> {
        // Try SQLite storage first
        try {
            // Dynamic import to avoid loading better-sqlite3 if it's not available
            const { SqliteStorage } = await import("./SqliteStorage.js");
            const storage = new SqliteStorage(config);
            
            // Test that it works by calling getStats
            await storage.getStats();
            
            console.log(chalk.green("✓") + " Using SQLite storage for command history");
            return storage;
        } catch (error) {
            // Check if it's a bindings error or other SQLite-specific error
            const errorMessage = error instanceof Error ? error.message : String(error);
            const errorCode = (error as NodeJS.ErrnoException)?.code;
            
            const isBindingsError = errorMessage.includes("Could not locate the bindings file") ||
                                  errorMessage.includes("better_sqlite3.node") ||
                                  errorCode === "MODULE_NOT_FOUND";
            
            if (isBindingsError) {
                console.log(chalk.yellow("⚠") + " SQLite bindings not found, falling back to JSON storage");
                console.log(chalk.dim("  To use SQLite storage, run: pnpm rebuild better-sqlite3"));
            } else {
                console.log(chalk.yellow("⚠") + " SQLite initialization failed, falling back to JSON storage");
                console.log(chalk.dim(`  Error: ${errorMessage}`));
            }
        }
        
        // Fall back to JSON storage
        const storage = new JsonStorage(config);
        console.log(chalk.blue("ℹ") + " Using JSON storage for command history");
        console.log(chalk.dim("  History limited to 10,000 most recent entries"));
        
        return storage;
    }
    
    /**
     * Creates a storage instance synchronously (used in exit handlers)
     * This will only work if a storage instance was already created asynchronously
     */
    static createSync(config: ConfigManager): ExtendedHistoryStorage {
        // For sync creation, we can only use JsonStorage since SQLite requires async init
        return new JsonStorage(config);
    }
}

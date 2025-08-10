import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import { HistoryManager } from "../history/HistoryManager.js";
import { HistoryUI } from "../history/HistoryUI.js";
import { type HistorySearchQuery, type HistoryEntry } from "../history/types.js";
import chalk from "chalk";
import { promises as fs } from "fs";
import * as path from "path";
import { SECONDS_1_MS, CLI_LIMITS } from "../utils/constants.js";

export class HistoryCommands {
    private manager: HistoryManager;
    
    constructor(
        private program: Command,
        private client: ApiClient,
        private config: ConfigManager,
    ) {
        this.manager = new HistoryManager(config);
        this.registerCommands();
    }
    
    private registerCommands(): void {
        const historyCmd = this.program
            .command("history")
            .description("View and manage command history");
        
        // List recent commands
        historyCmd
            .command("list")
            .description("List recent commands")
            .option("-n, --count <number>", "Number of entries to show", "50")
            .option("-c, --command <cmd>", "Filter by command")
            .option("--success", "Show only successful commands")
            .option("--failed", "Show only failed commands")
            .option("-f, --format <format>", "Output format (table|json|script)", "table")
            .option("-p, --profile <profile>", "Filter by profile")
            .action(async (options) => {
                await this.listHistory(options);
            });
        
        // Search history
        historyCmd
            .command("search <query>")
            .description("Search command history")
            .option("-n, --count <number>", "Number of results", "20")
            .option("-c, --command <cmd>", "Filter by command")
            .option("--success", "Show only successful commands")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .action(async (query, options) => {
                await this.searchHistory(query, options);
            });
        
        // Replay a command
        historyCmd
            .command("replay <id>")
            .description("Replay a command from history")
            .option("--edit", "Edit before executing")
            .option("--dry-run", "Show command without executing")
            .action(async (id, options) => {
                await this.replayCommand(id, options);
            });
        
        // Show history statistics
        historyCmd
            .command("stats")
            .description("Show history statistics")
            .action(async () => {
                await this.showStats();
            });
        
        // Browse history interactively
        historyCmd
            .command("browse")
            .description("Browse history interactively")
            .action(async () => {
                await this.browseHistory();
            });
        
        // Create alias from history entry
        historyCmd
            .command("alias <name> <id>")
            .description("Create an alias from a history entry")
            .action(async (name, id) => {
                await this.createAlias(name, id);
            });
        
        // Export history
        historyCmd
            .command("export")
            .description("Export command history")
            .option("-o, --output <file>", "Output file")
            .option("--format <format>", "Export format (json|csv|script)", "json")
            .option("--since <date>", "Export commands since date")
            .option("--until <date>", "Export commands until date")
            .option("--success", "Export only successful commands")
            .option("--failed", "Export only failed commands")
            .action(async (options) => {
                await this.exportHistory(options);
            });
        
        // Clear history
        historyCmd
            .command("clear")
            .description("Clear command history")
            .option("--force", "Skip confirmation")
            .option("--older-than <date>", "Clear entries older than date")
            .action(async (options) => {
                await this.clearHistory(options);
            });
        
        // Delete specific entry
        historyCmd
            .command("delete <id>")
            .description("Delete a specific history entry")
            .option("--force", "Skip confirmation")
            .action(async (id, options) => {
                await this.deleteEntry(id, options);
            });
        
        // Show database info
        historyCmd
            .command("info")
            .description("Show history database information")
            .action(async () => {
                await this.showDbInfo();
            });
        
        // Suggest commands based on partial input
        historyCmd
            .command("suggest <partial>")
            .description("Get command suggestions based on partial input")
            .option("-n, --count <number>", "Number of suggestions", "5")
            .action(async (partial, options) => {
                await this.suggestCommands(partial, options);
            });
        
        // Show frequent commands
        historyCmd
            .command("frequent")
            .description("Show frequently used commands")
            .option("-n, --count <number>", "Number of commands", "10")
            .action(async (options) => {
                await this.showFrequentCommands(options);
            });
    }
    
    private async listHistory(options: {
        count?: string;
        command?: string;
        success?: boolean;
        failed?: boolean;
        format?: string;
        profile?: string;
    }): Promise<void> {
        try {
            const entries = await this.manager.search({
                limit: parseInt(options.count || "50"),
                command: options.command,
                successOnly: options.success,
                failedOnly: options.failed,
                profile: options.profile,
            });
            
            if (this.config.isJsonOutput() || options.format === "json") {
                output.json(entries);
                return;
            }
            
            if (options.format === "script") {
                const successfulEntries = entries.filter(e => e.success);
                successfulEntries.forEach(entry => {
                    output.info(`${entry.command} ${entry.args.join(" ")}`.trim());
                });
                return;
            }
            
            if (entries.length === 0) {
                output.info(chalk.yellow("No history entries found"));
                return;
            }
            
            this.displayHistoryTable(entries);
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to list history: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async searchHistory(query: string, options: {
        count?: string;
        command?: string;
        success?: boolean;
        format?: string;
    }): Promise<void> {
        try {
            const entries = await this.manager.search({
                text: query,
                limit: parseInt(options.count || "20"),
                command: options.command,
                successOnly: options.success,
            });
            
            if (this.config.isJsonOutput() || options.format === "json") {
                output.json(entries);
                return;
            }
            
            if (entries.length === 0) {
                output.info(chalk.yellow(`No history entries found for "${query}"`));
                return;
            }
            
            output.info(chalk.bold(`\\nSearch Results for "${query}":`));
            this.displayHistoryTable(entries);
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to search history: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async replayCommand(id: string, options: {
        edit?: boolean;
        dryRun?: boolean;
    }): Promise<void> {
        try {
            const entry = await this.manager.get(id);
            
            if (!entry) {
                output.error(`Command not found in history: ${id}`);
                process.exit(1);
            }
            
            const command = `${entry.command} ${entry.args.join(" ")}`.trim();
            
            if (options.dryRun) {
                output.info(chalk.gray(`Would execute: ${command}`));
                return;
            }
            
            if (options.edit) {
                // In a real implementation, this would open an editor
                output.info(chalk.yellow("Edit mode not implemented yet"));
                output.info(chalk.gray(`Command: ${command}`));
                return;
            }
            
            output.info(chalk.gray(`Replaying: ${command}`));
            
            // In a real implementation, this would execute the command
            // For now, we'll just show what would be executed
            output.info(chalk.yellow("Command replay not implemented yet"));
            output.info(chalk.gray(`Would execute: ${command}`));
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to replay command: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async showStats(): Promise<void> {
        try {
            const stats = await this.manager.getStats();
            
            if (this.config.isJsonOutput()) {
                output.json(stats);
                return;
            }
            
            output.info(chalk.bold("\\nCommand History Statistics:"));
            output.info(`  Total Commands: ${stats.totalCommands}`);
            output.info(`  Successful: ${chalk.green(stats.successfulCommands)} (${(stats.successfulCommands / stats.totalCommands * 100).toFixed(1)}%)`);
            output.info(`  Failed: ${chalk.red(stats.failedCommands)} (${(stats.failedCommands / stats.totalCommands * 100).toFixed(1)}%)`);
            output.info(`  Unique Commands: ${stats.uniqueCommands}`);
            output.info(`  Average Duration: ${(stats.avgDuration / SECONDS_1_MS).toFixed(2)}s`);
            output.info(`  Last Command: ${stats.lastCommand.toLocaleString()}`);
            
            if (stats.topCommands.length > 0) {
                output.info(chalk.bold("\\nTop Commands:"));
                stats.topCommands.forEach((cmd, i) => {
                    output.info(`  ${i + 1}. ${cmd.command} (${cmd.count} times)`);
                });
            }
            
            if (Object.keys(stats.commandsByProfile).length > 0) {
                output.info(chalk.bold("\\nCommands by Profile:"));
                Object.entries(stats.commandsByProfile).forEach(([profile, count]) => {
                    output.info(`  ${profile}: ${count} commands`);
                });
            }
            
            if (stats.recentActivity.length > 0) {
                output.info(chalk.bold("\\nRecent Activity (last 30 days):"));
                stats.recentActivity.slice(0, CLI_LIMITS.DISPLAY_RECENT_DAYS).forEach(({ date, count }) => {
                    output.info(`  ${date}: ${count} commands`);
                });
            }
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to show stats: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async browseHistory(): Promise<void> {
        try {
            const ui = new HistoryUI(this.manager, this.config);
            await ui.browse();
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to browse history: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async createAlias(name: string, id: string): Promise<void> {
        try {
            const entry = await this.manager.get(id);
            
            if (!entry) {
                output.error(`Command not found in history: ${id}`);
                process.exit(1);
            }
            
            const command = `${entry.command} ${entry.args.join(" ")}`.trim();
            
            // For now, just show what the alias would be
            output.success(`Would create alias '${name}' for command:`);
            output.info(chalk.gray(`  ${command}`));
            output.info(chalk.yellow("Alias creation not implemented yet"));
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to create alias: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async exportHistory(options: {
        output?: string;
        format?: string;
        since?: string;
        until?: string;
        success?: boolean;
        failed?: boolean;
    }): Promise<void> {
        try {
            const query: HistorySearchQuery = {};
            
            if (options.since) {
                query.startDate = new Date(options.since);
            }
            
            if (options.until) {
                query.endDate = new Date(options.until);
            }
            
            if (options.success) {
                query.successOnly = true;
            }
            
            if (options.failed) {
                query.failedOnly = true;
            }
            
            const entries = await this.manager.search(query);
            
            if (entries.length === 0) {
                output.info(chalk.yellow("No history entries found to export"));
                return;
            }
            
            const format = options.format || "json";
            let exportData: string;
            
            switch (format) {
                case "json":
                    exportData = JSON.stringify(entries, null, 2);
                    break;
                case "csv":
                    exportData = this.exportToCsv(entries);
                    break;
                case "script":
                    exportData = this.exportToScript(entries);
                    break;
                default:
                    throw new Error(`Unsupported export format: ${format}`);
            }
            
            if (options.output) {
                const outputPath = path.resolve(options.output);
                await fs.writeFile(outputPath, exportData);
                output.success(`Exported ${entries.length} entries to ${outputPath}`);
            } else {
                output.info(exportData);
            }
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to export history: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async clearHistory(options: {
        force?: boolean;
        olderThan?: string;
    }): Promise<void> {
        try {
            if (!options.force) {
                // In a real implementation, this would prompt for confirmation
                output.info(chalk.yellow("History clearing requires --force flag"));
                output.info(chalk.gray("Use: vrooli history clear --force"));
                return;
            }
            
            await this.manager.clear();
            output.success("History cleared");
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to clear history: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async deleteEntry(id: string, options: {
        force?: boolean;
    }): Promise<void> {
        try {
            const entry = await this.manager.get(id);
            
            if (!entry) {
                output.error(`Command not found in history: ${id}`);
                process.exit(1);
            }
            
            if (!options.force) {
                output.info(chalk.yellow("Entry deletion requires --force flag"));
                output.info(chalk.gray(`Entry: ${entry.command} ${entry.args.join(" ")}`));
                return;
            }
            
            await this.manager.delete(id);
            output.success("History entry deleted");
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to delete entry: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async showDbInfo(): Promise<void> {
        try {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const storage = (this.manager as any).storage;
            
            if (storage && typeof storage.getDbInfo === "function") {
                const dbInfo = storage.getDbInfo();
                
                output.info(chalk.bold("\\nHistory Database Information:"));
                output.info(`  Database Path: ${dbInfo.path}`);
                output.info(`  Database Size: ${this.formatFileSize(dbInfo.size)}`);
                output.info(`  Page Count: ${dbInfo.pageCount}`);
                
                const stats = await this.manager.getStats();
                output.info(`  Total Entries: ${stats.totalCommands}`);
                output.info(`  Oldest Entry: ${stats.recentActivity[stats.recentActivity.length - 1]?.date || "N/A"}`);
                output.info(`  Newest Entry: ${stats.lastCommand.toLocaleString()}`);
            } else {
                output.info(chalk.yellow("Database info not available"));
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to show database info: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async suggestCommands(partial: string, options: {
        count?: string;
    }): Promise<void> {
        try {
            const DEFAULT_SUGGESTION_COUNT = 5;
            const suggestions = await this.manager.getSuggestions(partial, parseInt(options.count || "5") as typeof DEFAULT_SUGGESTION_COUNT);
            
            if (suggestions.length === 0) {
                output.info(chalk.yellow(`No suggestions found for "${partial}"`));
                return;
            }
            
            output.info(chalk.bold(`\\nSuggestions for "${partial}":`));
            suggestions.forEach((entry, i) => {
                const command = `${entry.command} ${entry.args.join(" ")}`.trim();
                output.info(`  ${i + 1}. ${command}`);
                output.info(`     ${chalk.gray(`Used ${entry.timestamp.toLocaleDateString()}`)}`);
            });
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to get suggestions: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private async showFrequentCommands(options: {
        count?: string;
    }): Promise<void> {
        try {
            const DEFAULT_FREQUENT_COUNT = 10;
            const frequent = await this.manager.getFrequentCommands(parseInt(options.count || "10") as typeof DEFAULT_FREQUENT_COUNT);
            
            if (frequent.length === 0) {
                output.info(chalk.yellow("No frequent commands found"));
                return;
            }
            
            output.info(chalk.bold("\\nFrequently Used Commands:"));
            frequent.forEach((cmd, i) => {
                output.info(`  ${i + 1}. ${cmd.command} (${cmd.count} times)`);
                output.info(`     ${chalk.gray(`Last used: ${cmd.lastUsed.toLocaleString()}`)}`);
            });
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`Failed to show frequent commands: ${errorMessage}`);
            process.exit(1);
        }
    }
    
    private displayHistoryTable(entries: HistoryEntry[]): void {
        output.info(chalk.bold("\\nCommand History:"));
        output.newline();
        
        entries.forEach((entry, index) => {
            const status = entry.success ? chalk.green("✓") : chalk.red("✗");
            const command = `${entry.command} ${entry.args.join(" ")}`.trim();
            const time = entry.timestamp.toLocaleString();
            const duration = entry.duration ? `${(entry.duration / SECONDS_1_MS).toFixed(2)}s` : "N/A";
            
            output.info(`${status} ${chalk.cyan(entry.id.slice(0, CLI_LIMITS.ID_DISPLAY_LENGTH))} ${command}`);
            output.info(`   ${chalk.gray(`${time} | ${duration} | ${entry.profile}`)}`);
            
            if (entry.error) {
                output.info(`   ${chalk.red(`Error: ${entry.error}`)}`);
            }
            
            if (index < entries.length - 1) {
                output.newline();
            }
        });
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
    
    private formatFileSize(bytes: number): string {
        if (bytes === 0) return "0 B";
        
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB"];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
    }
}

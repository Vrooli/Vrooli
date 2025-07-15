import inquirer from "inquirer";
import chalk from "chalk";
import { type ConfigManager } from "../utils/config.js";
import { type HistoryManager } from "./HistoryManager.js";
import { type HistoryEntry } from "./types.js";
import { CLI_LIMITS, UI, SECONDS_1_MS } from "../utils/constants.js";
import type { HistorySearchQuery } from "./types.js";

interface HistoryUIState {
    entries: HistoryEntry[];
    currentPage: number;
    pageSize: number;
    totalEntries: number;
    searchQuery?: string;
    filterCommand?: string;
    filterProfile?: string;
    showSuccessOnly: boolean;
    showFailedOnly: boolean;
}

export class HistoryUI {
    private state: HistoryUIState = {
        entries: [],
        currentPage: 0,
        pageSize: CLI_LIMITS.DEFAULT_HISTORY_LIMIT,
        totalEntries: 0,
        showSuccessOnly: false,
        showFailedOnly: false,
    };

    constructor(
        private manager: HistoryManager,
        private config: ConfigManager,
    ) {}

    async browse(): Promise<void> {
        console.log(chalk.bold("\n📚 Interactive History Browser"));
        console.log(chalk.gray("Navigate through your command history with filters and actions\n"));

        await this.refreshEntries();
        await this.showMainMenu();
    }

    private async refreshEntries(): Promise<void> {
        try {
            const query: HistorySearchQuery = {
                limit: this.state.pageSize,
                offset: this.state.currentPage * this.state.pageSize,
            };

            if (this.state.searchQuery) {
                query.text = this.state.searchQuery;
            }

            if (this.state.filterCommand) {
                query.command = this.state.filterCommand;
            }

            if (this.state.filterProfile) {
                query.profile = this.state.filterProfile;
            }

            if (this.state.showSuccessOnly) {
                query.successOnly = true;
            }

            if (this.state.showFailedOnly) {
                query.failedOnly = true;
            }

            this.state.entries = await this.manager.search(query);

            // Get total count for pagination (approximate)
            const totalQuery = { ...query };
            delete totalQuery.limit;
            delete totalQuery.offset;
            const allEntries = await this.manager.search(totalQuery);
            this.state.totalEntries = allEntries.length;

        } catch (error) {
            console.error(chalk.red("Failed to load history entries"));
            throw error;
        }
    }

    private async showMainMenu(): Promise<void> {
        let shouldExit = false;
        while (!shouldExit) {
            console.clear();
            this.displayHeader();
            this.displayEntries();
            this.displayPagination();

            const action = await this.promptForAction();
            
            if (action === "exit") {
                console.log(chalk.green("👋 Goodbye!"));
                shouldExit = true;
            } else {
                await this.handleAction(action);
            }
        }
    }

    private displayHeader(): void {
        console.log(chalk.bold("📚 History Browser"));
        
        // Show active filters
        const filters: string[] = [];
        if (this.state.searchQuery) filters.push(`Search: "${this.state.searchQuery}"`);
        if (this.state.filterCommand) filters.push(`Command: ${this.state.filterCommand}`);
        if (this.state.filterProfile) filters.push(`Profile: ${this.state.filterProfile}`);
        if (this.state.showSuccessOnly) filters.push("Success only");
        if (this.state.showFailedOnly) filters.push("Failed only");
        
        if (filters.length > 0) {
            console.log(chalk.yellow(`Active filters: ${filters.join(", ")}`));
        }
        
        console.log(chalk.gray(`Total entries: ${this.state.totalEntries}`));
        console.log("");
    }

    private displayEntries(): void {
        if (this.state.entries.length === 0) {
            console.log(chalk.yellow("No history entries found matching your criteria."));
            return;
        }

        this.state.entries.forEach((entry, index) => {
            const globalIndex = this.state.currentPage * this.state.pageSize + index;
            const status = entry.success ? chalk.green("✓") : chalk.red("✗");
            const command = `${entry.command} ${entry.args.join(" ")}`.trim();
            const time = entry.timestamp.toLocaleString();
            const duration = entry.duration ? `${(entry.duration / SECONDS_1_MS).toFixed(2)}s` : "N/A";
            
            console.log(`${chalk.cyan((globalIndex + 1).toString().padStart(3))}. ${status} ${command}`);
            console.log(`     ${chalk.gray(`${time} | ${duration} | ${entry.profile}`)}`);
            
            if (entry.error) {
                console.log(`     ${chalk.red(`Error: ${entry.error}`)}`);
            }
            
            console.log("");
        });
    }

    private displayPagination(): void {
        const totalPages = Math.ceil(this.state.totalEntries / this.state.pageSize);
        const currentPage = this.state.currentPage + 1;
        
        if (totalPages > 1) {
            console.log(chalk.gray(`Page ${currentPage} of ${totalPages}`));
        }
        
        console.log("");
    }

    private async promptForAction(): Promise<string> {
        const choices = [
            { name: "🔍 Search history", value: "search" },
            { name: "🎛️  Filter by command", value: "filter_command" },
            { name: "👤 Filter by profile", value: "filter_profile" },
            { name: "✅ Show successful only", value: "filter_success" },
            { name: "❌ Show failed only", value: "filter_failed" },
            { name: "🚫 Clear all filters", value: "clear_filters" },
            new inquirer.Separator(),
            { name: "▶️  Execute entry", value: "execute" },
            { name: "📋 Copy to clipboard", value: "copy" },
            { name: "🔗 Create alias", value: "alias" },
            { name: "🗑️  Delete entry", value: "delete" },
            new inquirer.Separator(),
            { name: "⬅️  Previous page", value: "prev_page" },
            { name: "➡️  Next page", value: "next_page" },
            { name: "📊 Show statistics", value: "stats" },
            { name: "📤 Export history", value: "export" },
            new inquirer.Separator(),
            { name: "🚪 Exit", value: "exit" },
        ];

        // Filter choices based on current state
        const filteredChoices = choices.filter(choice => {
            if (typeof choice === "object" && choice && "value" in choice) {
                if (choice.value === "prev_page" && this.state.currentPage === 0) return false;
                if (choice.value === "next_page" && (this.state.currentPage + 1) * this.state.pageSize >= this.state.totalEntries) return false;
                if (choice.value === "filter_success" && this.state.showSuccessOnly) return false;
                if (choice.value === "filter_failed" && this.state.showFailedOnly) return false;
                if (choice.value === "clear_filters" && !this.hasActiveFilters()) return false;
            }
            return true;
        });

        const { action } = await inquirer.prompt([
            {
                type: "list",
                name: "action",
                message: "What would you like to do?",
                choices: filteredChoices,
                pageSize: 15,
            },
        ]);

        return action;
    }

    private async handleAction(action: string): Promise<void> {
        switch (action) {
            case "search":
                await this.handleSearch();
                break;
            case "filter_command":
                await this.handleFilterCommand();
                break;
            case "filter_profile":
                await this.handleFilterProfile();
                break;
            case "filter_success":
                this.state.showSuccessOnly = true;
                this.state.showFailedOnly = false;
                this.state.currentPage = 0;
                await this.refreshEntries();
                break;
            case "filter_failed":
                this.state.showFailedOnly = true;
                this.state.showSuccessOnly = false;
                this.state.currentPage = 0;
                await this.refreshEntries();
                break;
            case "clear_filters":
                this.clearFilters();
                await this.refreshEntries();
                break;
            case "execute":
                await this.handleExecute();
                break;
            case "copy":
                await this.handleCopy();
                break;
            case "alias":
                await this.handleCreateAlias();
                break;
            case "delete":
                await this.handleDelete();
                break;
            case "prev_page":
                this.state.currentPage--;
                await this.refreshEntries();
                break;
            case "next_page":
                this.state.currentPage++;
                await this.refreshEntries();
                break;
            case "stats":
                await this.handleStats();
                break;
            case "export":
                await this.handleExport();
                break;
        }
    }

    private async handleSearch(): Promise<void> {
        const { query } = await inquirer.prompt([
            {
                type: "input",
                name: "query",
                message: "Enter search query:",
                default: this.state.searchQuery,
            },
        ]);

        this.state.searchQuery = query || undefined;
        this.state.currentPage = 0;
        await this.refreshEntries();
    }

    private async handleFilterCommand(): Promise<void> {
        const { command } = await inquirer.prompt([
            {
                type: "input",
                name: "command",
                message: "Filter by command:",
                default: this.state.filterCommand,
            },
        ]);

        this.state.filterCommand = command || undefined;
        this.state.currentPage = 0;
        await this.refreshEntries();
    }

    private async handleFilterProfile(): Promise<void> {
        const { profile } = await inquirer.prompt([
            {
                type: "input",
                name: "profile",
                message: "Filter by profile:",
                default: this.state.filterProfile,
            },
        ]);

        this.state.filterProfile = profile || undefined;
        this.state.currentPage = 0;
        await this.refreshEntries();
    }

    private clearFilters(): void {
        this.state.searchQuery = undefined;
        this.state.filterCommand = undefined;
        this.state.filterProfile = undefined;
        this.state.showSuccessOnly = false;
        this.state.showFailedOnly = false;
        this.state.currentPage = 0;
    }

    private hasActiveFilters(): boolean {
        return !!(
            this.state.searchQuery ||
            this.state.filterCommand ||
            this.state.filterProfile ||
            this.state.showSuccessOnly ||
            this.state.showFailedOnly
        );
    }

    private async handleExecute(): Promise<void> {
        const entry = await this.selectEntry("Select entry to execute:");
        if (!entry) return;

        const command = `${entry.command} ${entry.args.join(" ")}`.trim();
        
        console.log(chalk.yellow(`\nCommand: ${command}`));
        
        const { confirm } = await inquirer.prompt([
            {
                type: "confirm",
                name: "confirm",
                message: "Execute this command?",
                default: false,
            },
        ]);

        if (confirm) {
            console.log(chalk.gray("\nExecuting command..."));
            console.log(chalk.yellow("Command execution not implemented yet"));
            console.log(chalk.gray(`Would execute: ${command}`));
            await this.waitForKeypress();
        }
    }

    private async handleCopy(): Promise<void> {
        const entry = await this.selectEntry("Select entry to copy:");
        if (!entry) return;

        const command = `${entry.command} ${entry.args.join(" ")}`.trim();
        
        // In a real implementation, this would copy to clipboard
        console.log(chalk.green(`\n✓ Copied to clipboard: ${command}`));
        console.log(chalk.yellow("Clipboard functionality not implemented yet"));
        await this.waitForKeypress();
    }

    private async handleCreateAlias(): Promise<void> {
        const entry = await this.selectEntry("Select entry to create alias for:");
        if (!entry) return;

        const command = `${entry.command} ${entry.args.join(" ")}`.trim();
        
        const { aliasName } = await inquirer.prompt([
            {
                type: "input",
                name: "aliasName",
                message: "Enter alias name:",
                validate: (input) => {
                    if (!input.trim()) return "Alias name cannot be empty";
                    if (input.includes(" ")) return "Alias name cannot contain spaces";
                    return true;
                },
            },
        ]);

        console.log(chalk.green(`\n✓ Would create alias '${aliasName}' for:`));
        console.log(chalk.gray(`  ${command}`));
        console.log(chalk.yellow("Alias creation not implemented yet"));
        await this.waitForKeypress();
    }

    private async handleDelete(): Promise<void> {
        const entry = await this.selectEntry("Select entry to delete:");
        if (!entry) return;

        const command = `${entry.command} ${entry.args.join(" ")}`.trim();
        
        console.log(chalk.yellow(`\nEntry: ${command}`));
        
        const { confirm } = await inquirer.prompt([
            {
                type: "confirm",
                name: "confirm",
                message: "Are you sure you want to delete this entry?",
                default: false,
            },
        ]);

        if (confirm) {
            try {
                await this.manager.delete(entry.id);
                console.log(chalk.green("\n✓ Entry deleted successfully"));
                await this.refreshEntries();
            } catch (error) {
                console.error(chalk.red(`\n✗ Failed to delete entry: ${error instanceof Error ? error.message : String(error)}`));
            }
            await this.waitForKeypress();
        }
    }

    private async handleStats(): Promise<void> {
        try {
            const stats = await this.manager.getStats();
            
            console.clear();
            console.log(chalk.bold("📊 History Statistics\n"));
            console.log(`Total Commands: ${stats.totalCommands}`);
            console.log(`Successful: ${chalk.green(stats.successfulCommands)} (${(stats.successfulCommands / stats.totalCommands * 100).toFixed(1)}%)`);
            console.log(`Failed: ${chalk.red(stats.failedCommands)} (${(stats.failedCommands / stats.totalCommands * 100).toFixed(1)}%)`);
            console.log(`Unique Commands: ${stats.uniqueCommands}`);
            console.log(`Average Duration: ${(stats.avgDuration / SECONDS_1_MS).toFixed(2)}s`);
            console.log(`Last Command: ${stats.lastCommand.toLocaleString()}`);
            
            if (stats.topCommands.length > 0) {
                console.log(chalk.bold("\nTop Commands:"));
                stats.topCommands.slice(0, CLI_LIMITS.STATS_TOP_COMMANDS_LIMIT).forEach((cmd, i) => {
                    console.log(`  ${i + 1}. ${cmd.command} (${cmd.count} times)`);
                });
            }
            
            if (Object.keys(stats.commandsByProfile).length > 0) {
                console.log(chalk.bold("\nCommands by Profile:"));
                Object.entries(stats.commandsByProfile).forEach(([profile, count]) => {
                    console.log(`  ${profile}: ${count} commands`);
                });
            }
            
            await this.waitForKeypress();
        } catch (error) {
            console.error(chalk.red(`Failed to load statistics: ${error instanceof Error ? error.message : String(error)}`));
            await this.waitForKeypress();
        }
    }

    private async handleExport(): Promise<void> {
        const { format } = await inquirer.prompt([
            {
                type: "list",
                name: "format",
                message: "Select export format:",
                choices: [
                    { name: "JSON", value: "json" },
                    { name: "CSV", value: "csv" },
                    { name: "Shell Script", value: "script" },
                ],
            },
        ]);

        try {
            const data = await this.manager.export(format);
            
            console.log(chalk.bold("\n📤 Export Preview:"));
            console.log(chalk.gray("─".repeat(UI.SEPARATOR_LENGTH)));
            console.log(data.slice(0, CLI_LIMITS.EXPORT_PREVIEW_LENGTH) + (data.length > CLI_LIMITS.EXPORT_PREVIEW_LENGTH ? "\n... (truncated)" : ""));
            console.log(chalk.gray("─".repeat(UI.SEPARATOR_LENGTH)));
            
            console.log(chalk.green(`\n✓ Export completed (${data.length} characters)`));
            console.log(chalk.yellow("File saving not implemented yet"));
            
            await this.waitForKeypress();
        } catch (error) {
            console.error(chalk.red(`Export failed: ${error instanceof Error ? error.message : String(error)}`));
            await this.waitForKeypress();
        }
    }

    private async selectEntry(message: string): Promise<HistoryEntry | null> {
        if (this.state.entries.length === 0) {
            console.log(chalk.yellow("No entries available"));
            await this.waitForKeypress();
            return null;
        }

        const choices = this.state.entries.map((entry, index) => {
            const globalIndex = this.state.currentPage * this.state.pageSize + index;
            const status = entry.success ? chalk.green("✓") : chalk.red("✗");
            const command = `${entry.command} ${entry.args.join(" ")}`.trim();
            const time = entry.timestamp.toLocaleString();
            
            return {
                name: `${status} ${command} (${time})`,
                value: entry,
                short: `${globalIndex + 1}. ${command}`,
            };
        });

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (choices as any).push(new inquirer.Separator());
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (choices as any).push({ name: "← Back to main menu", value: null, short: "Back" });

        const { selectedEntry } = await inquirer.prompt([
            {
                type: "list",
                name: "selectedEntry",
                message,
                choices,
                pageSize: 12,
            },
        ]);

        return selectedEntry;
    }

    private async waitForKeypress(): Promise<void> {
        await inquirer.prompt([
            {
                type: "input",
                name: "continue",
                message: "Press Enter to continue...",
            },
        ]);
    }
}

import chalk from "chalk";
import { type ConfigManager } from "./config.js";
import { type ContextManager } from "./contextManager.js";
import { ConversationExporter, type ExportFormat, type ExportOptions } from "./conversationExporter.js";
import { type ChatMessage, type User } from "@vrooli/shared";
import { LIMITS } from "./constants.js";

export interface SlashCommand {
    name: string;
    args: string[];
    rawInput: string;
}

export interface SlashCommandResult {
    shouldExit?: boolean;
    shouldClear?: boolean;
    shouldSend?: boolean;
    shouldContinue?: boolean;
    message?: string;
    error?: string;
}

export class SlashCommandParser {
    constructor(
        private config: ConfigManager,
        private chatId: string,
        private contextManager?: ContextManager,
        private getMessages?: () => ChatMessage[],
        private getParticipants?: () => User[],
    ) {}

    /**
     * Parse input and determine if it's a slash command
     */
    parseCommand(input: string): SlashCommand | null {
        const trimmed = input.trim();
        
        if (!trimmed.startsWith("/")) {
            return null;
        }

        // Remove leading slash and split by spaces
        const parts = trimmed.slice(1).split(/\s+/).filter(Boolean);
        
        if (parts.length === 0) {
            return null;
        }

        return {
            name: parts[0].toLowerCase(),
            args: parts.slice(1),
            rawInput: trimmed,
        };
    }

    /**
     * Execute a slash command and return the result
     */
    async executeCommand(command: SlashCommand): Promise<SlashCommandResult> {
        switch (command.name) {
            case "help":
            case "?":
                return this.handleHelp();
            
            case "exit":
            case "quit":
            case "q":
                return this.handleExit();
            
            case "clear":
            case "cls":
                return this.handleClear();
            
            case "history":
            case "hist":
                return this.handleHistory(command.args);
            
            case "status":
                return this.handleStatus();
            
            case "context":
                return await this.handleContext(command.args);
            
            case "ctx":
                return await this.handleContext(command.args);
            
            case "save":
                return await this.handleSave(command.args);
            
            case "bots":
                return this.handleBots();
            
            case "settings":
                return this.handleSettings(command.args);
            
            default:
                return {
                    error: `Unknown command: ${command.name}. Type /help for available commands.`,
                };
        }
    }

    /**
     * Get help text for slash commands
     */
    private handleHelp(): SlashCommandResult {
        const helpText = `
${chalk.bold("Available Slash Commands:")}

${chalk.cyan("/help, /?")}           Show this help message
${chalk.cyan("/exit, /quit, /q")}     Exit interactive mode
${chalk.cyan("/clear, /cls")}         Clear conversation display
${chalk.cyan("/history [n]")}         Show last n messages (default: ${LIMITS.MAX_MESSAGE_DISPLAY})
${chalk.cyan("/status")}              Show bot and connection status

${chalk.bold("Context Management:")}
${chalk.cyan("/context file <path>")} Add file as context
${chalk.cyan("/context url <url>")}   Add web content as context
${chalk.cyan("/context list")}        Show all context items
${chalk.cyan("/context remove <id>")} Remove context item
${chalk.cyan("/context clear")}       Clear all context
${chalk.cyan("/ctx")}                 Alias for /context

${chalk.bold("Other Commands:")}
${chalk.cyan("/save <file> [format]")} Save conversation (markdown, json, txt, html, csv)
${chalk.cyan("/bots")}                 Switch to different bot (future feature)
${chalk.cyan("/settings")}             Show/modify chat settings (future feature)

${chalk.gray("Type any message without a slash to send it to the bot.")}
`;

        return {
            message: helpText,
        };
    }

    /**
     * Handle exit command
     */
    private handleExit(): SlashCommandResult {
        return {
            shouldExit: true,
            message: chalk.yellow("Exiting interactive chat mode..."),
        };
    }

    /**
     * Handle clear command
     */
    private handleClear(): SlashCommandResult {
        return {
            shouldClear: true,
            message: chalk.gray("Chat display cleared"),
        };
    }

    /**
     * Handle history command
     */
    private handleHistory(args: string[]): SlashCommandResult {
        const count = args.length > 0 ? parseInt(args[0]) : LIMITS.MAX_MESSAGE_DISPLAY;
        
        if (isNaN(count) || count <= 0) {
            return {
                error: "Invalid number for history command. Use: /history [number]",
            };
        }

        // This will be implemented by the chat engine
        return {
            message: chalk.gray(`Showing last ${count} messages (implemented by chat engine)`),
        };
    }

    /**
     * Handle status command
     */
    private handleStatus(): SlashCommandResult {
        const profile = this.config.getActiveProfileName();
        const server = this.config.getServerUrl();
        const isAuthenticated = !!this.config.getAuthToken();

        const statusText = `
${chalk.bold("Chat Status:")}
  Chat ID: ${chalk.cyan(this.chatId)}
  Profile: ${chalk.cyan(profile)}
  Server: ${chalk.cyan(server)}
  Authentication: ${isAuthenticated ? chalk.green("✓ Connected") : chalk.red("✗ Not authenticated")}
  WebSocket: ${chalk.gray("Status checked by chat engine")}
`;

        return {
            message: statusText,
        };
    }

    /**
     * Handle context command
     */
    private async handleContext(args: string[]): Promise<SlashCommandResult> {
        if (!this.contextManager) {
            return {
                error: "Context management is not available in this session.",
            };
        }

        if (args.length === 0) {
            return {
                error: "Context command requires arguments. Use: /context list, /context file <path>, /context url <url>, /context remove <id>, or /context clear",
            };
        }

        const subcommand = args[0].toLowerCase();

        try {
            switch (subcommand) {
                case "file":
                    return await this.handleContextFile(args.slice(1));
                
                case "url":
                    return await this.handleContextUrl(args.slice(1));
                
                case "list":
                case "ls":
                    return this.handleContextList();
                
                case "remove":
                case "rm":
                    return this.handleContextRemove(args.slice(1));
                
                case "clear":
                    return this.handleContextClear();
                
                default:
                    return {
                        error: `Unknown context subcommand: ${subcommand}. Use: file, url, list, remove, or clear`,
                    };
            }
        } catch (error) {
            return {
                error: `Context command failed: ${error instanceof Error ? error.message : String(error)}`,
            };
        }
    }

    /**
     * Handle adding file context
     */
    private async handleContextFile(args: string[]): Promise<SlashCommandResult> {
        if (args.length === 0) {
            return {
                error: "File path required. Use: /context file <path> [alias]",
            };
        }

        const filePath = args[0];
        const alias = args[1];

        if (!this.contextManager) {
            return {
                error: "Context manager not available",
            };
        }

        try {
            const contextId = await this.contextManager.addFile(filePath, alias);
            return {
                message: chalk.green(`✓ Added file context: ${chalk.cyan(contextId)} (${filePath})`),
            };
        } catch (error) {
            return {
                error: `Failed to add file: ${error instanceof Error ? error.message : String(error)}`,
            };
        }
    }

    /**
     * Handle adding URL context
     */
    private async handleContextUrl(args: string[]): Promise<SlashCommandResult> {
        if (args.length === 0) {
            return {
                error: "URL required. Use: /context url <url> [alias]",
            };
        }

        const url = args[0];
        const alias = args[1];

        if (!this.contextManager) {
            return {
                error: "Context manager not available",
            };
        }

        try {
            const contextId = await this.contextManager.addUrl(url, alias);
            return {
                message: chalk.green(`✓ Added URL context: ${chalk.cyan(contextId)} (${url})`),
            };
        } catch (error) {
            return {
                error: `Failed to add URL: ${error instanceof Error ? error.message : String(error)}`,
            };
        }
    }

    /**
     * Handle listing context items
     */
    private handleContextList(): SlashCommandResult {
        if (!this.contextManager) {
            return {
                error: "Context manager not available",
            };
        }

        const contexts = this.contextManager.getAllContexts();
        
        if (contexts.size === 0) {
            return {
                message: chalk.gray("No context items added"),
            };
        }

        let output = chalk.bold("Context Items:\n");
        
        for (const [id, context] of contexts) {
            if ("path" in context) {
                output += `📄 ${chalk.cyan(id)} - ${context.path} (${this.formatFileSize(context.size)})\n`;
            } else if ("url" in context) {
                output += `🌐 ${chalk.cyan(id)} - ${context.url} (${this.formatFileSize(context.size)})\n`;
            } else {
                output += `⚙️ ${chalk.cyan(id)} - Routine ${context.id}\n`;
            }
        }

        if (!this.contextManager) {
            return {
                error: "Context manager not initialized",
                shouldContinue: true,
            };
        }
        const stats = this.contextManager.getStats();
        output += `\n${chalk.gray(`Total: ${stats.totalContexts}/${stats.maxContexts} items, ${this.formatFileSize(stats.totalSize)}`)}`;

        return {
            message: output,
        };
    }

    /**
     * Handle removing context item
     */
    private handleContextRemove(args: string[]): SlashCommandResult {
        if (args.length === 0) {
            return {
                error: "Context ID required. Use: /context remove <id>",
            };
        }

        const contextId = args[0];
        if (!this.contextManager) {
            return {
                error: "Context manager not available",
            };
        }

        const removed = this.contextManager.removeContext(contextId);

        if (removed) {
            return {
                message: chalk.green(`✓ Removed context: ${chalk.cyan(contextId)}`),
            };
        } else {
            return {
                error: `Context not found: ${contextId}`,
            };
        }
    }

    /**
     * Handle clearing all context
     */
    private handleContextClear(): SlashCommandResult {
        if (!this.contextManager) {
            return {
                error: "Context manager not available",
            };
        }

        const stats = this.contextManager.getStats();
        this.contextManager.clearAll();

        return {
            message: chalk.green(`✓ Cleared all context (${stats.totalContexts} items removed)`),
        };
    }

    /**
     * Format file size for display
     */
    private formatFileSize(bytes: number): string {
        if (bytes === 0) return "0 B";
        
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB"];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
    }

    /**
     * Handle save command
     */
    private async handleSave(args: string[]): Promise<SlashCommandResult> {
        if (!this.getMessages || !this.getParticipants) {
            return {
                error: "Conversation export is not available in this session.",
            };
        }

        if (args.length === 0) {
            return {
                error: "Save command requires a filename. Use: /save <filename> [format] [options]",
            };
        }

        const filename = args[0];
        const format = args[1] as ExportFormat;
        
        try {
            const messages = this.getMessages();
            const participants = this.getParticipants();
            
            if (messages.length === 0) {
                return {
                    error: "No messages to export.",
                };
            }

            // Validate format if provided
            if (format && !ConversationExporter.isValidFormat(format)) {
                const availableFormats = ConversationExporter.getAvailableFormats().join(", ");
                return {
                    error: `Invalid format '${format}'. Available formats: ${availableFormats}`,
                };
            }

            // Prepare export data
            const exportData = {
                chatId: this.chatId,
                title: `Chat ${this.chatId}`,
                participants,
                messages,
                exportedAt: new Date(),
                totalMessages: messages.length,
            };

            // Export options
            const exportOptions: Partial<ExportOptions> = {
                format,
                includeTimestamps: true,
                includeUserIds: false,
                includeMessageIds: false,
            };

            // Parse additional options from args
            for (let i = 2; i < args.length; i++) {
                const arg = args[i];
                switch (arg) {
                    case "--include-ids":
                        exportOptions.includeUserIds = true;
                        exportOptions.includeMessageIds = true;
                        break;
                    case "--no-timestamps":
                        exportOptions.includeTimestamps = false;
                        break;
                    case "--max-messages": {
                        const maxMessages = parseInt(args[++i]);
                        if (!isNaN(maxMessages) && maxMessages > 0) {
                            exportOptions.maxMessages = maxMessages;
                        }
                        break;
                    }
                }
            }

            // Export conversation
            const exporter = new ConversationExporter();
            await exporter.exportToFile(exportData, filename, exportOptions);

            const formatUsed = exportOptions.format || "auto-detected";
            const messageCount = exportOptions.maxMessages || messages.length;
            
            return {
                message: chalk.green(`✓ Conversation exported to ${chalk.cyan(filename)}\n`) +
                         chalk.gray(`  Format: ${formatUsed} | Messages: ${messageCount} | Size: ${this.getFileSize(filename)}`),
            };

        } catch (error) {
            return {
                error: `Failed to export conversation: ${error instanceof Error ? error.message : String(error)}`,
            };
        }
    }

    /**
     * Get file size (placeholder - would need fs.stat in real implementation)
     */
    private getFileSize(_filename: string): string {
        // This is a placeholder - in real implementation would use fs.stat
        return "~KB";
    }

    /**
     * Handle bots command (future feature)
     */
    private handleBots(): SlashCommandResult {
        return {
            message: chalk.yellow("Bot switching is not yet implemented. Use 'vrooli chat create' to start a new chat."),
        };
    }

    /**
     * Handle settings command (future feature)
     */
    private handleSettings(args: string[]): SlashCommandResult {
        if (args.length === 0) {
            const settingsText = `
${chalk.bold("Current Settings:")}
  Output Format: ${this.config.isJsonOutput() ? "JSON" : "Human-readable"}
  Debug Mode: ${this.config.isDebug() ? "Enabled" : "Disabled"}
  Profile: ${this.config.getActiveProfileName()}

${chalk.gray("Setting modification is not yet implemented.")}
`;
            return {
                message: settingsText,
            };
        }

        return {
            message: chalk.yellow("Setting modification is not yet implemented."),
        };
    }

    /**
     * Check if input looks like a command but is malformed
     */
    isLikelyCommand(input: string): boolean {
        const trimmed = input.trim();
        return trimmed.startsWith("/") && trimmed.length > 1;
    }

    /**
     * Get available command names for autocomplete (future feature)
     */
    getAvailableCommands(): string[] {
        return [
            "help", "?",
            "exit", "quit", "q",
            "clear", "cls",
            "history", "hist",
            "status",
            "context",
            "save",
            "bots",
            "settings",
        ];
    }
}

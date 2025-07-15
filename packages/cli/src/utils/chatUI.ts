import chalk from "chalk";
import { type ChatMessage, type User } from "@vrooli/shared";
import { TERMINAL_DIMENSIONS, UI } from "./constants.js";

export interface ChatDisplayOptions {
    showTimestamps?: boolean;
    showUserIds?: boolean;
    maxMessageLength?: number;
    enableColors?: boolean;
}

export interface BotStatus {
    status: "idle" | "thinking" | "tool_calling" | "responding" | "error";
    message?: string;
    timestamp?: Date;
}

export class InteractiveChatUI {
    private messages: ChatMessage[] = [];
    private currentBotResponse = "";
    private botStatus: BotStatus = { status: "idle" };
    private options: ChatDisplayOptions;

    constructor(options: ChatDisplayOptions = {}) {
        this.options = {
            showTimestamps: true,
            showUserIds: false,
            maxMessageLength: 2000,
            enableColors: true,
            ...options,
        };
    }

    /**
     * Clear the terminal screen
     */
    clearScreen(): void {
        console.clear();
        this.displayHeader();
    }

    /**
     * Display chat header with connection info
     */
    displayHeader(): void {
        const headerLine = "─".repeat(process.stdout.columns || TERMINAL_DIMENSIONS.FALLBACK_WIDTH);
        console.log(chalk.bold.cyan(headerLine));
        console.log(chalk.bold.cyan("🤖 Vrooli Interactive Chat"));
        console.log(chalk.gray("Type your message and press Enter. Use /help for commands, /exit to quit."));
        console.log(chalk.bold.cyan(headerLine));
        console.log();
    }

    /**
     * Display a chat message
     */
    displayMessage(message: ChatMessage): void {
        const isBot = message.user.isBot;
        const userName = this.formatUserName(message.user);
        const timestamp = this.options.showTimestamps ? 
            this.formatTimestamp(new Date(message.createdAt)) : "";
        
        const icon = isBot ? "🤖" : "👤";
        const color = isBot ? chalk.blue : chalk.green;
        
        // Display user line
        const userLine = `${icon} ${color.bold(userName)}${timestamp}`;
        console.log(userLine);
        
        // Display message content with proper formatting
        const formattedText = this.formatMessageText(message.text);
        console.log(formattedText);
        console.log(); // Empty line for spacing
    }

    /**
     * Start streaming a bot response
     */
    startStreamingResponse(botUser: User): void {
        const userName = this.formatUserName(botUser);
        const timestamp = this.options.showTimestamps ? 
            this.formatTimestamp(new Date()) : "";
        
        const userLine = `🤖 ${chalk.blue.bold(userName)}${timestamp}`;
        console.log(userLine);
        
        this.currentBotResponse = "";
    }

    /**
     * Append text to the current streaming response
     */
    appendToStreamingResponse(chunk: string): void {
        this.currentBotResponse += chunk;
        
        // Clear current line and rewrite with updated content
        process.stdout.clearLine(0);
        process.stdout.cursorTo(0);
        
        // Display the accumulated response
        const formattedText = this.formatMessageText(this.currentBotResponse);
        process.stdout.write(formattedText);
    }

    /**
     * Finish streaming response
     */
    finishStreamingResponse(): void {
        console.log(); // Move to new line
        console.log(); // Add spacing
        this.currentBotResponse = "";
    }

    /**
     * Display bot status indicator
     */
    displayBotStatus(status: BotStatus): void {
        this.botStatus = status;
        
        let statusText = "";
        let color = chalk.gray;
        
        switch (status.status) {
            case "thinking":
                statusText = "💭 Bot is thinking...";
                color = chalk.yellow;
                break;
            case "tool_calling":
                statusText = `🔧 ${status.message || "Executing tools..."}`;
                color = chalk.cyan;
                break;
            case "responding":
                statusText = "💬 Bot is responding...";
                color = chalk.blue;
                break;
            case "error":
                statusText = `❌ ${status.message || "Error occurred"}`;
                color = chalk.red;
                break;
            case "idle":
                // Don't display idle status
                return;
        }
        
        console.log(color(statusText));
    }

    /**
     * Clear bot status indicator
     */
    clearBotStatus(): void {
        this.botStatus = { status: "idle" };
    }

    /**
     * Display typing indicator
     */
    displayTypingIndicator(users: string[]): void {
        if (users.length === 0) return;
        
        const userNames = users.join(", ");
        console.log(chalk.gray(`💬 ${userNames} ${users.length === 1 ? "is" : "are"} typing...`));
    }

    /**
     * Display connection status
     */
    displayConnectionStatus(connected: boolean, chatId: string): void {
        const status = connected ? 
            chalk.green("⚡ Connected") : 
            chalk.red("❌ Disconnected");
        
        console.log(`Status: ${status} | 💬 Chat: ${chalk.cyan(chatId)}`);
    }

    /**
     * Display error message
     */
    displayError(error: string): void {
        console.log(chalk.red(`❌ Error: ${error}`));
    }

    /**
     * Display info message
     */
    displayInfo(message: string): void {
        console.log(chalk.gray(`ℹ️  ${message}`));
    }

    /**
     * Display success message
     */
    displaySuccess(message: string): void {
        console.log(chalk.green(`✅ ${message}`));
    }

    /**
     * Display warning message
     */
    displayWarning(message: string): void {
        console.log(chalk.yellow(`⚠️  ${message}`));
    }

    /**
     * Display the input prompt
     */
    displayPrompt(): void {
        process.stdout.write(chalk.bold.cyan("> "));
    }

    /**
     * Display tool execution information
     */
    displayToolExecution(toolName: string, args: Record<string, unknown>): void {
        console.log(chalk.cyan(`🔧 Executing: ${toolName}`));
        if (args && Object.keys(args).length > 0) {
            const argsStr = JSON.stringify(args, null, 2);
            console.log(chalk.gray(`   Args: ${argsStr}`));
        }
    }

    /**
     * Display conversation history
     */
    displayHistory(messages: ChatMessage[], count?: number): void {
        const messagesToShow = count ? messages.slice(-count) : messages;
        
        if (messagesToShow.length === 0) {
            this.displayInfo("No messages in chat history");
            return;
        }

        console.log(chalk.bold("\n📜 Chat History:"));
        console.log("─".repeat(UI.SEPARATOR_LENGTH));
        
        messagesToShow.forEach((message) => {
            this.displayMessage(message);
        });
        
        console.log("─".repeat(UI.SEPARATOR_LENGTH));
    }

    /**
     * Format user name for display
     */
    private formatUserName(user: User): string {
        const name = user.name || user.handle || "Unknown User";
        
        if (this.options.showUserIds) {
            return `${name} (${user.id})`;
        }
        
        return name;
    }

    /**
     * Format timestamp for display
     */
    private formatTimestamp(date: Date): string {
        if (!this.options.showTimestamps) return "";
        
        const time = date.toLocaleTimeString([], { 
            hour: "2-digit", 
            minute: "2-digit",
            second: "2-digit",
        });
        
        return ` ${chalk.gray(`(${time})`)}`;
    }

    /**
     * Format message text with proper wrapping and styling
     */
    private formatMessageText(text: string): string {
        if (!this.options.enableColors) {
            return text;
        }

        // Truncate if too long
        const maxLength = this.options.maxMessageLength || TERMINAL_DIMENSIONS.MAX_MESSAGE_LENGTH;
        const displayText = text.length > maxLength ? 
            text.substring(0, maxLength) + "..." : text;

        // Handle code blocks
        return this.formatCodeBlocks(displayText);
    }

    /**
     * Format code blocks in message text
     */
    private formatCodeBlocks(text: string): string {
        // Simple code block formatting - can be enhanced
        return text
            .replace(/```(\w+)?\n([\s\S]*?)```/g, (match, lang, code) => {
                return chalk.gray("```") + (lang ? chalk.blue(lang) : "") + "\n" +
                       chalk.white(code) + "\n" + chalk.gray("```");
            })
            .replace(/`([^`]+)`/g, (match, code) => {
                return chalk.bgGray.white(` ${code} `);
            });
    }

    /**
     * Get current terminal dimensions for layout
     */
    getTerminalDimensions(): { width: number; height: number } {
        return {
            width: process.stdout.columns || TERMINAL_DIMENSIONS.DEFAULT_WIDTH,
            height: process.stdout.rows || TERMINAL_DIMENSIONS.DEFAULT_HEIGHT,
        };
    }

    /**
     * Check if terminal supports colors
     */
    supportsColor(): boolean {
        return process.stdout.isTTY && (process.env.TERM !== "dumb");
    }
}

import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";
import ora from "ora";
import {
    type Chat,
    type ChatCreateInput,
    type ChatMessage,
    type ChatMessageSearchInput,
    type ChatSearchInput,
    type User,
    type UserSearchInput,
    type UserSearchResult,
    type ChatSearchResult,
    type ChatMessageSearchResult,
    generatePK,
    type MessageConfigObject,
    type SwarmTask,
    type TaskContextInfoInput,
    type ChatMessageCreateWithTaskInfoInput,
    type VisibilityType,
    endpointsUser,
    endpointsChat,
    endpointsChatMessage,
} from "@vrooli/shared";
import { InteractiveChatEngine, type InteractiveChatOptions } from "../utils/chatEngine.js";
import { TIMEOUTS, UI } from "../utils/constants.js";

// Interface definitions for chat operations
interface SocketData {
    chatId?: string;
    chunk?: string;
    message?: {
        text?: string;
        [key: string]: unknown;
    };
    [key: string]: unknown;
}

export class ChatCommands {
    constructor(
        private program: Command,
        private client: ApiClient,
        private config: ConfigManager,
    ) {
        this.registerCommands();
    }

    private registerCommands(): void {
        const chatCmd = this.program
            .command("chat")
            .description("Chat commands for interacting with bots");

        // List available bots
        chatCmd
            .command("list-bots")
            .description("List available bots")
            .option("-s, --search <term>", "Search bots by name or description")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .option("-l, --limit <number>", "Limit number of results", "20")
            .action(async (options) => {
                await this.listBots(options);
            });

        // Create a new chat
        chatCmd
            .command("create <bot-id>")
            .description("Start a new chat with a bot")
            .option("-n, --name <name>", "Name for the chat")
            .option("-t, --task <task>", "Task or goal for the chat")
            .option("--public", "Make chat public (default: private)")
            .action(async (botId, options) => {
                await this.createChat(botId, options);
            });

        // List user's chats
        chatCmd
            .command("list")
            .description("List your chats")
            .option("-f, --format <format>", "Output format (table|json)", "table")
            .option("-l, --limit <number>", "Limit number of results", "20")
            .option("--mine", "Show only your own chats", true)
            .option("-s, --search <term>", "Search chats by name")
            .action(async (options) => {
                await this.listChats(options);
            });

        // Show chat history
        chatCmd
            .command("show <chat-id>")
            .description("Display chat history")
            .option("-l, --limit <number>", "Limit number of messages", "50")
            .option("-f, --format <format>", "Output format (conversation|json)", "conversation")
            .action(async (chatId, options) => {
                await this.showChat(chatId, options);
            });

        // Send a message
        chatCmd
            .command("send <chat-id> <message>")
            .description("Send a message to a chat")
            .option("-m, --model <model>", "AI model to use", "claude-3-5-sonnet-20241022")
            .option("--wait", "Wait for bot response", true)
            .option("--timeout <seconds>", "Response timeout in seconds", "30")
            .action(async (chatId, message, options) => {
                await this.sendMessage(chatId, message, options);
            });

        // Interactive chat mode
        chatCmd
            .command("interactive [chat-id]")
            .description("Enter interactive chat mode (creates new chat if ID not provided)")
            .option("-b, --bot <bot-id>", "Bot ID for new chat (required if no chat-id)")
            .option("-n, --name <name>", "Name for new chat")
            .option("-t, --task <task>", "Task or goal for new chat")
            .option("-m, --model <model>", "AI model to use", "claude-3-5-sonnet-20241022")
            .option("--show-tools", "Show bot tool executions", false)
            .option("--approve-tools", "Enable interactive tool approval", false)
            .option("--context-file <files...>", "Add files as context")
            .option("--timeout <seconds>", "Response timeout in seconds", TIMEOUTS.MAX_WAIT_SECONDS.toString())
            .action(async (chatId, options) => {
                await this.interactiveChat(chatId, options);
            });
    }

    private async listBots(options: {
        search?: string;
        format: string;
        limit: string;
    }): Promise<void> {
        try {
            const spinner = ora("Searching for bots...").start();

            // Search for users with isBot = true
            const searchInput: UserSearchInput = {
                take: parseInt(options.limit),
                searchString: options.search,
                isBot: true,
            };

            try {
                const result = await this.client.requestWithEndpoint<UserSearchResult>(
                    endpointsUser.findMany,
                    searchInput as Record<string, unknown>,
                );
                spinner.succeed(`Found ${result.edges.length} bots`);

                if (this.config.isJsonOutput() || options.format === "json") {
                    output.json(result);
                } else {
                    this.displayBotsTable(result.edges.map(edge => edge.node));
                }
            } catch (error) {
                spinner.fail("Failed to search bots");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Failed to list bots: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async createChat(botId: string, options: {
        name?: string;
        task?: string;
        public?: boolean;
    }): Promise<void> {
        try {
            const spinner = ora("Creating chat...").start();

            const chatInput: ChatCreateInput = {
                id: generatePK().toString(),
                openToAnyoneWithInvite: options.public || false,
                task: options.task,
                translationsCreate: options.name ? [{
                    id: generatePK().toString(),
                    language: "en",
                    name: options.name,
                }] : undefined,
                // Create initial invite for the bot
                invitesCreate: [{
                    id: generatePK().toString(),
                    chatConnect: "", // Will be set automatically
                    userConnect: botId,
                    message: options.task ? `Join chat: ${options.task}` : "Join chat",
                }],
            };

            try {
                const result = await this.client.requestWithEndpoint<Chat>(
                    endpointsChat.createOne,
                    chatInput as unknown as Record<string, unknown>,
                );
                spinner.succeed("Chat created successfully");

                if (this.config.isJsonOutput()) {
                    output.json(result);
                } else {
                    output.success("Chat created");
                    output.info(`  Chat ID: ${result.id}`);
                    output.info(`  Public ID: ${result.publicId}`);
                    if (result.translations?.[0]?.name) {
                        output.info(`  Name: ${result.translations[0].name}`);
                    }
                    if (options.task) {
                        output.info(`  Task: ${options.task}`);
                    }
                    output.info(chalk.gray(`\nUse 'vrooli chat send ${result.id} "your message"' to start chatting`));
                }
            } catch (error) {
                spinner.fail("Failed to create chat");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Failed to create chat: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async listChats(options: {
        format: string;
        limit: string;
        mine: boolean;
        search?: string;
    }): Promise<void> {
        try {
            const spinner = ora("Fetching chats...").start();

            const searchInput: ChatSearchInput = {
                take: parseInt(options.limit),
                searchString: options.search,
                visibility: options.mine ? ("Own" as VisibilityType) : ("OwnOrPublic" as VisibilityType),
            };

            try {
                const result = await this.client.requestWithEndpoint<ChatSearchResult>(
                    endpointsChat.findMany,
                    searchInput as Record<string, unknown>,
                );
                spinner.succeed(`Found ${result.edges.length} chats`);

                if (this.config.isJsonOutput() || options.format === "json") {
                    output.json(result);
                } else {
                    this.displayChatsTable(result.edges.map(edge => edge.node));
                }
            } catch (error) {
                spinner.fail("Failed to fetch chats");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Failed to list chats: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async showChat(chatId: string, options: {
        limit: string;
        format: string;
    }): Promise<void> {
        try {
            const spinner = ora("Fetching chat history...").start();

            const searchInput: ChatMessageSearchInput = {
                chatId,
                take: parseInt(options.limit),
            };

            try {
                const result = await this.client.requestWithEndpoint<ChatMessageSearchResult>(
                    endpointsChatMessage.findMany,
                    searchInput,
                );
                spinner.succeed(`Found ${result.edges.length} messages`);

                if (this.config.isJsonOutput() || options.format === "json") {
                    output.json(result);
                } else {
                    this.displayChatHistory(result.edges.map(edge => edge.node));
                }
            } catch (error) {
                spinner.fail("Failed to fetch chat history");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Failed to show chat: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async sendMessage(chatId: string, message: string, options: {
        model: string;
        wait: boolean;
        timeout: string;
    }): Promise<void> {
        try {
            const spinner = ora("Sending message...").start();

            // Basic message configuration
            const messageConfig: MessageConfigObject = {
                __version: "1.0",
                resources: [],
                role: "user",
            };

            // Create task context for the swarm
            const swarmTask: SwarmTask = {
                goal: message,
            };

            const taskContexts: TaskContextInfoInput[] = [];

            const messageInput: ChatMessageCreateWithTaskInfoInput = {
                message: {
                    id: generatePK().toString(),
                    chatConnect: chatId,
                    userConnect: "", // Will be determined by auth
                    text: message,
                    language: "en",
                    config: messageConfig,
                    versionIndex: 1,
                },
                model: options.model,
                task: swarmTask,
                taskContexts,
            };

            try {
                const result = await this.client.requestWithEndpoint<ChatMessage>(
                    endpointsChatMessage.createOne,
                    messageInput,
                );
                spinner.succeed("Message sent");

                if (this.config.isJsonOutput()) {
                    output.json(result);
                } else {
                    output.success("Message sent");
                    output.info(`  Message ID: ${result.id}`);
                    output.info(`  Text: ${result.text}`);
                    
                    if (options.wait) {
                        output.info(chalk.gray("\nWaiting for bot response..."));
                        await this.waitForResponse(chatId, parseInt(options.timeout));
                    }
                }
            } catch (error) {
                spinner.fail("Failed to send message");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Failed to send message: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async waitForResponse(chatId: string, timeoutSeconds: number): Promise<void> {
        return new Promise((resolve, reject) => {
            const socket = this.client.connectWebSocket();
            const MILLISECONDS_PER_SECOND = 1000;
            const timeout = setTimeout(() => {
                socket.disconnect();
                reject(new Error("Response timeout"));
            }, timeoutSeconds * MILLISECONDS_PER_SECOND); // Convert seconds to milliseconds

            // Listen for new messages in this chat
            socket.on("messages", (data: SocketData) => {
                if (data.chatId === chatId) {
                    clearTimeout(timeout);
                    socket.disconnect();
                    
                    // Display the bot's response
                    output.info(chalk.blue("\n🤖 Bot response:"));
                    output.info(`${data.message?.text || ""}\n`);
                    resolve();
                }
            });

            // Also listen for response streaming
            socket.on("responseStream", (data: SocketData) => {
                if (data.chatId === chatId) {
                    process.stdout.write(chalk.blue(data.chunk));
                }
            });

            socket.on("error", (error: unknown) => {
                clearTimeout(timeout);
                socket.disconnect();
                reject(error);
            });
        });
    }

    private async interactiveChat(chatId: string | undefined, options: {
        bot?: string;
        name?: string;
        task?: string;
        model: string;
        showTools: boolean;
        approveTools: boolean;
        contextFile?: string[];
        timeout: string;
    }): Promise<void> {
        try {
            // Check authentication first
            const token = this.config.getAuthToken();
            if (!token) {
                if (this.config.isJsonOutput()) {
                    output.json({
                        success: false,
                        error: "Not authenticated. Run 'vrooli auth login' first.",
                    });
                } else {
                    output.error("Not authenticated. Run 'vrooli auth login' first.");
                }
                process.exit(1);
            }

            // If no chat ID provided, create a new chat
            if (!chatId) {
                // Bot ID is required for new chat
                if (!options.bot) {
                    if (this.config.isJsonOutput()) {
                        output.json({
                            success: false,
                            error: "Bot ID required when creating new chat. Use -b <bot-id> or provide a chat ID.",
                        });
                    } else {
                        output.error("Bot ID required when creating new chat. Use -b <bot-id> or provide a chat ID.");
                        output.info(chalk.gray("\nTip: Run 'vrooli chat list-bots' to see available bots"));
                    }
                    process.exit(1);
                }

                // Create new chat
                output.info(chalk.cyan("Creating new chat..."));
                const chatPk = generatePK().toString();
                const createInput: ChatCreateInput = {
                    id: chatPk,
                    invitesCreate: [{
                        id: generatePK().toString(),
                        chatConnect: chatPk,
                        userConnect: options.bot,
                        message: "Bot invited to chat",
                    }],
                    translationsCreate: options.name ? [{
                        id: generatePK().toString(),
                        language: "en",
                        name: options.name,
                    }] : undefined,
                };

                try {
                    const result = await this.client.requestWithEndpoint<Chat>(
                        endpointsChat.createOne,
                        createInput as unknown as Record<string, unknown>,
                    );
                    chatId = result.id;
                    output.success("Created new chat: " + chatId);
                } catch (error) {
                    const errorMessage = error instanceof Error ? error.message : String(error);
                    output.error(`Failed to create chat: ${errorMessage}`);
                    process.exit(1);
                }
            }

            // Create interactive chat options
            const chatOptions: InteractiveChatOptions = {
                model: options.model,
                showTools: options.showTools,
                approveTools: options.approveTools,
                contextFiles: options.contextFile,
                timeout: parseInt(options.timeout),
            };

            // Create and start the interactive chat engine
            const chatEngine = new InteractiveChatEngine(
                this.client,
                this.config,
                chatId,
                chatOptions,
            );

            // Start the interactive session
            await chatEngine.startSession();

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`Interactive chat failed: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    // Display helpers
    private displayBotsTable(bots: User[]): void {
        if (bots.length === 0) {
            output.info(chalk.yellow("No bots found"));
            return;
        }

        output.info(chalk.bold("\nAvailable Bots:"));
        output.info("");
        
        bots.forEach((bot, index) => {
            const name = bot.name || bot.handle || "Unnamed Bot";
            const description = bot.translations?.[0]?.bio || "No description";
            
            output.info(`${chalk.cyan((index + 1).toString().padStart(3))}. ${chalk.bold(name)}`);
            output.info(`     ID: ${chalk.gray(bot.id)}`);
            output.info(`     ${description.length > UI.TERMINAL_WIDTH ? description.substring(0, UI.TERMINAL_WIDTH) + "..." : description}`);
            output.info("");
        });
    }

    private displayChatsTable(chats: Chat[]): void {
        if (chats.length === 0) {
            output.info(chalk.yellow("No chats found"));
            return;
        }

        output.info(chalk.bold("\nYour Chats:"));
        output.info("");
        
        chats.forEach((chat, index) => {
            const name = chat.translations?.[0]?.name || "Unnamed Chat";
            const participants = chat.participantsCount || 0;
            const messages = chat.messages?.length || 0;
            const updated = new Date(chat.updatedAt).toLocaleDateString();
            
            output.info(`${chalk.cyan((index + 1).toString().padStart(3))}. ${chalk.bold(name)}`);
            output.info(`     ID: ${chalk.gray(chat.id)}`);
            output.info(`     Participants: ${participants} | Messages: ${messages} | Updated: ${updated}`);
            output.info("");
        });
    }

    private displayChatHistory(messages: ChatMessage[]): void {
        if (messages.length === 0) {
            output.info(chalk.yellow("No messages found"));
            return;
        }

        output.info(chalk.bold("\nChat History:"));
        output.info("─".repeat(TIMEOUTS.MAX_WAIT_SECONDS));
        
        // Sort messages by creation date
        const sortedMessages = messages.sort((a, b) => 
            new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );
        
        sortedMessages.forEach((message) => {
            const isBot = message.user.isBot;
            const userName = message.user.name || message.user.handle || "Unknown";
            const timestamp = new Date(message.createdAt).toLocaleString();
            const icon = isBot ? "🤖" : "👤";
            const color = isBot ? chalk.blue : chalk.green;
            
            output.info("");
            output.info(color(`${icon} ${userName} (${timestamp})`));
            output.info(message.text);
        });
        
        output.info("");
        output.info("─".repeat(TIMEOUTS.MAX_WAIT_SECONDS));
    }
}

import { createInterface, type Interface } from "readline";
import { type Socket } from "socket.io-client";
import {
    type ChatMessage,
    type ChatMessageCreateWithTaskInfoInput,
    type ChatMessageSearchInput,
    type User,
    generatePK,
    type MessageConfigObject,
    type SwarmTask,
    type TaskContextInfoInput,
} from "@vrooli/shared";
import { type ApiClient } from "./client.js";
import { type ConfigManager } from "./config.js";
import { InteractiveChatUI, type BotStatus } from "./chatUI.js";
import { TIMEOUTS, CHAT_CONFIG } from "./constants.js";

// Socket event data interfaces
interface SocketEventData {
    chatId: string;
}

interface ResponseStreamData extends SocketEventData {
    chunk?: string;
    isComplete?: boolean;
    messageId?: string;
    finished?: boolean;
}

interface BotStatusData extends SocketEventData {
    status?: string;
    message?: string;
}

interface TypingIndicatorData extends SocketEventData {
    isTyping: boolean;
    userId: string;
    users?: string[];
}

interface LlmTasksData extends SocketEventData {
    tasks: Array<{
        id: string;
        type: string;
        status: string;
        description?: string;
        name?: string;
        args?: Record<string, unknown>;
    }>;
}

interface NewMessageData extends SocketEventData {
    message: ChatMessage;
}

interface ToolApprovalData extends SocketEventData {
    pendingId: string;
    toolCallId: string;
    toolName: string;
    toolArguments: Record<string, unknown>;
    callerBotId: string;
    callerBotName?: string;
    approvalTimeoutAt?: number;
    estimatedCost?: string;
    conversationId: string;
    reason?: string;
}


interface ChatSearchEdge {
    node: ChatMessage;
}
import { SlashCommandParser, type SlashCommand } from "./slashCommands.js";
import { ToolApprovalHandler, type ToolExecutionInfo } from "./toolApproval.js";
import { ContextManager } from "./contextManager.js";

export interface InteractiveChatOptions {
    model?: string;
    showTools?: boolean;
    approveTools?: boolean;
    contextFiles?: string[];
    autoScroll?: boolean;
    timeout?: number;
}

export interface ChatSessionInfo {
    chatId: string;
    botUser?: User;
    currentUser?: User;
    messageCount: number;
    startTime: Date;
}

export class InteractiveChatEngine {
    private ui: InteractiveChatUI;
    private slashParser: SlashCommandParser;
    private toolApprovalHandler: ToolApprovalHandler | null = null;
    private contextManager: ContextManager;
    private readline: Interface | null = null;
    private socket: Socket | null = null;
    private isRunning = false;
    private session: ChatSessionInfo;
    private messages: ChatMessage[] = [];
    private currentStreamingMessage = "";

    constructor(
        private client: ApiClient,
        private config: ConfigManager,
        private chatId: string,
        private options: InteractiveChatOptions = {},
    ) {
        this.ui = new InteractiveChatUI({
            showTimestamps: true,
            enableColors: true, // Default to true, will be checked later
        });
        
        // Initialize context manager
        this.contextManager = new ContextManager();
        
        // Initialize slash parser with context manager and data accessors
        this.slashParser = new SlashCommandParser(
            config, 
            chatId, 
            this.contextManager,
            () => this.messages,
            () => this.getParticipants(),
        );
        
        // Initialize tool approval handler if enabled
        if (this.options.approveTools) {
            this.toolApprovalHandler = new ToolApprovalHandler(
                client,
                config,
                (toolInfo: ToolExecutionInfo) => {
                    this.handleToolStatusUpdate(toolInfo);
                },
            );
        }
        
        this.session = {
            chatId,
            messageCount: 0,
            startTime: new Date(),
        };
    }

    /**
     * Start the interactive chat session
     */
    async startSession(): Promise<void> {
        try {
            this.isRunning = true;
            
            // Clear screen and show header
            this.ui.clearScreen();
            
            // Load initial chat history
            await this.loadChatHistory();
            
            // Initialize context files if provided
            await this.initializeContextFiles();
            
            // Connect WebSocket
            await this.connectWebSocket();
            
            // Setup readline interface
            this.setupReadline();
            
            // Display connection status
            this.ui.displayConnectionStatus(true, this.chatId);
            console.log();
            
            // Show initial help
            this.ui.displayInfo("Interactive chat mode started. Type /help for commands or start chatting!");
            console.log();
            
            // Start input loop
            this.startInputLoop();
            
        } catch (error) {
            this.ui.displayError(`Failed to start chat session: ${error instanceof Error ? error.message : String(error)}`);
            await this.cleanup();
            throw error;
        }
    }

    /**
     * Send a message to the chat
     */
    async sendMessage(messageText: string): Promise<void> {
        try {
            const messageConfig: MessageConfigObject = {
                __version: "1.0",
                resources: [],
                role: "user",
            };

            const swarmTask: SwarmTask = {
                goal: messageText,
            };

            const taskContexts: TaskContextInfoInput[] = this.contextManager.getTaskContexts();

            const messageInput: ChatMessageCreateWithTaskInfoInput = {
                message: {
                    id: generatePK().toString(),
                    chatConnect: this.chatId,
                    userConnect: "", // Will be determined by auth
                    text: messageText,
                    language: "en",
                    config: messageConfig,
                    versionIndex: 1,
                },
                model: this.options.model || "claude-3-5-sonnet-20241022",
                task: swarmTask,
                taskContexts,
            };

            // Send message via API
            await this.client.post("/chatMessage", messageInput);
            
            // Update message count
            this.session.messageCount++;
            
            // The message will be displayed via WebSocket events
            
        } catch (error) {
            this.ui.displayError(`Failed to send message: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Handle user input
     */
    private async handleUserInput(input: string): Promise<void> {
        const trimmedInput = input.trim();
        
        if (!trimmedInput) {
            this.promptForInput();
            return;
        }

        // Check for slash commands
        const slashCommand = this.slashParser.parseCommand(trimmedInput);
        
        if (slashCommand) {
            await this.handleSlashCommand(slashCommand);
            return;
        }

        // Check if it looks like a command but is malformed
        if (this.slashParser.isLikelyCommand(trimmedInput)) {
            this.ui.displayError("Invalid command format. Type /help for available commands.");
            this.promptForInput();
            return;
        }

        // Send as regular message
        await this.sendMessage(trimmedInput);
        this.promptForInput();
    }

    /**
     * Handle slash commands
     */
    private async handleSlashCommand(command: SlashCommand): Promise<void> {
        const result = await this.slashParser.executeCommand(command);
        
        if (result.error) {
            this.ui.displayError(result.error);
        }
        
        if (result.message) {
            console.log(result.message);
        }
        
        if (result.shouldClear) {
            this.ui.clearScreen();
            this.ui.displayHistory(this.messages);
        }
        
        if (result.shouldExit) {
            await this.cleanup();
            return;
        }
        
        // Handle special commands
        if (command.name === "history") {
            const count = command.args.length > 0 ? parseInt(command.args[0]) : CHAT_CONFIG.DEFAULT_LIMIT;
            this.ui.displayHistory(this.messages, count);
        }
        
        this.promptForInput();
    }

    /**
     * Initialize context files from command line options
     */
    private async initializeContextFiles(): Promise<void> {
        if (!this.options.contextFiles || this.options.contextFiles.length === 0) {
            return;
        }

        for (const filePath of this.options.contextFiles) {
            try {
                const contextId = await this.contextManager.addFile(filePath);
                this.ui.displaySuccess(`Added context file: ${contextId}`);
            } catch (error) {
                this.ui.displayWarning(`Failed to add context file ${filePath}: ${error instanceof Error ? error.message : String(error)}`);
            }
        }

        // Show context summary if any files were added
        const stats = this.contextManager.getStats();
        if (stats.totalContexts > 0) {
            this.ui.displayInfo(`Loaded ${stats.totalContexts} context items (${this.formatFileSize(stats.totalSize)})`);
        }
    }

    /**
     * Load initial chat history
     */
    private async loadChatHistory(): Promise<void> {
        try {
            const searchInput: ChatMessageSearchInput = {
                chatId: this.chatId,
                take: 50, // Load last 50 messages
            };

            const result = await this.client.post<{ edges: ChatSearchEdge[] }>("/chatMessages", searchInput);
            this.messages = result.edges.map((edge: ChatSearchEdge) => edge.node);
            
            // Sort messages by creation date
            this.messages.sort((a, b) => 
                new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
            );
            
            // Display recent history
            if (this.messages.length > 0) {
                this.ui.displayHistory(this.messages.slice(-CHAT_CONFIG.DEFAULT_LIMIT)); // Show last 10
                
                // Extract bot user info if available
                const botMessage = this.messages.find(m => m.user.isBot);
                if (botMessage) {
                    this.session.botUser = botMessage.user;
                }
            }
            
        } catch (error) {
            this.ui.displayWarning("Could not load chat history");
        }
    }

    /**
     * Connect to WebSocket for real-time updates
     */
    private async connectWebSocket(): Promise<void> {
        this.socket = this.client.connectWebSocket();
        
        // Set up event listeners
        this.socket.on("connect", () => {
            this.ui.displaySuccess("Connected to real-time chat");
        });
        
        this.socket.on("disconnect", (reason) => {
            this.ui.displayWarning(`Disconnected from chat: ${reason}`);
        });
        
        this.socket.on("error", (error) => {
            this.ui.displayError(`WebSocket error: ${error}`);
        });
        
        // Chat-specific events
        this.socket.on("responseStream", (data) => {
            this.handleResponseStream(data);
        });
        
        this.socket.on("modelReasoningStream", (data) => {
            this.handleModelReasoningStream(data);
        });
        
        this.socket.on("botStatusUpdate", (data) => {
            this.handleBotStatusUpdate(data);
        });
        
        this.socket.on("typing", (data) => {
            this.handleTypingIndicator(data);
        });
        
        this.socket.on("llmTasks", (data) => {
            this.handleLlmTasks(data);
        });
        
        this.socket.on("messages", (data) => {
            this.handleNewMessage(data);
        });
        
        // Tool approval and execution events
        if (this.toolApprovalHandler) {
            this.socket.on("toolApprovalRequired", (data) => {
                this.handleToolApprovalRequired(data);
            });
            
            this.socket.on("toolApprovalGranted", (data) => {
                this.handleToolApprovalGranted(data);
            });
            
            this.socket.on("toolApprovalRejected", (data) => {
                this.handleToolApprovalRejected(data);
            });
            
            this.socket.on("toolApprovalTimeout", (data) => {
                this.handleToolApprovalTimeout(data);
            });
        }
        
        // Tool execution events (for display even without approval)
        if (this.options.showTools || this.toolApprovalHandler) {
            this.socket.on("toolCalled", (data) => {
                this.handleToolExecution("TOOL_CALLED", data);
            });
            
            this.socket.on("toolCompleted", (data) => {
                this.handleToolExecution("TOOL_COMPLETED", data);
            });
            
            this.socket.on("toolFailed", (data) => {
                this.handleToolExecution("TOOL_FAILED", data);
            });
        }
    }

    /**
     * Handle streaming response chunks
     */
    private handleResponseStream(data: ResponseStreamData): void {
        if (data.chatId !== this.chatId) return;
        
        if (data.chunk) {
            // First chunk - start streaming display
            if (!this.currentStreamingMessage) {
                this.ui.startStreamingResponse(this.session.botUser || { 
                    name: "Bot", 
                    isBot: true, 
                } as User);
            }
            
            this.currentStreamingMessage += data.chunk;
            this.ui.appendToStreamingResponse(data.chunk);
        }
        
        // Check if this is the end of the stream
        if (data.finished) {
            this.ui.finishStreamingResponse();
            this.currentStreamingMessage = "";
        }
    }

    /**
     * Handle model reasoning stream (thinking process)
     */
    private handleModelReasoningStream(data: SocketEventData): void {
        if (data.chatId !== this.chatId) return;
        
        // Display thinking indicator
        this.ui.displayBotStatus({
            status: "thinking",
            message: "Processing your request...",
        });
    }

    /**
     * Handle bot status updates
     */
    private handleBotStatusUpdate(data: BotStatusData): void {
        if (data.chatId !== this.chatId) return;
        
        const statusValue = data.status || "idle";
        const status: BotStatus = {
            status: statusValue as "idle" | "thinking" | "tool_calling" | "responding" | "error",
            message: data.message,
            timestamp: new Date(),
        };
        
        this.ui.displayBotStatus(status);
        
        // Clear status after a delay for some statuses
        if (status.status === "idle" || status.status === "error") {
            setTimeout(() => {
                this.ui.clearBotStatus();
            }, TIMEOUTS.CHAT_ENGINE_DEFAULT);
        }
    }

    /**
     * Handle typing indicators
     */
    private handleTypingIndicator(data: TypingIndicatorData): void {
        if (data.chatId !== this.chatId) return;
        
        if (data.isTyping && data.users && data.users.length > 0) {
            this.ui.displayTypingIndicator(data.users);
        }
    }

    /**
     * Handle LLM task updates
     */
    private handleLlmTasks(data: LlmTasksData): void {
        if (!this.options.showTools) return;
        if (data.chatId !== this.chatId) return;
        
        if (data.tasks && data.tasks.length > 0) {
            data.tasks.forEach((task) => {
                if (task.type === "tool_call" && task.name) {
                    this.ui.displayToolExecution(task.name, task.args || {});
                }
            });
        }
    }

    /**
     * Handle new message events
     */
    private handleNewMessage(data: NewMessageData): void {
        if (data.chatId !== this.chatId) return;
        
        const message = data.message;
        if (message && message.id) {
            // Add to local message history
            this.messages.push(message);
            
            // Only display if it's not currently streaming
            if (!this.currentStreamingMessage && !message.user.isBot) {
                this.ui.displayMessage(message);
            }
        }
    }

    /**
     * Setup readline interface for input
     */
    private setupReadline(): void {
        this.readline = createInterface({
            input: process.stdin,
            output: process.stdout,
            prompt: "",
        });
        
        this.readline.on("line", (input) => {
            this.handleUserInput(input);
        });
        
        this.readline.on("close", () => {
            this.cleanup();
        });
        
        // Handle Ctrl+C gracefully
        this.readline.on("SIGINT", () => {
            this.ui.displayInfo("Use /exit to quit or Ctrl+C again to force quit");
            this.promptForInput();
        });
    }

    /**
     * Start the input loop
     */
    private startInputLoop(): void {
        this.promptForInput();
    }

    /**
     * Prompt for user input
     */
    private promptForInput(): void {
        if (this.isRunning && this.readline) {
            this.ui.displayPrompt();
        }
    }

    /**
     * Handle tool approval required event
     */
    private async handleToolApprovalRequired(data: ToolApprovalData): Promise<void> {
        if (data.chatId !== this.chatId || !this.toolApprovalHandler) return;
        
        // Pause current input to handle approval
        if (this.readline) {
            console.log(); // New line before approval prompt
        }
        
        await this.toolApprovalHandler.handleApprovalRequest(data, this.chatId);
        
        // Resume input prompt after approval
        this.promptForInput();
    }

    /**
     * Handle tool approval granted event
     */
    private handleToolApprovalGranted(data: ToolApprovalData): void {
        if (data.chatId !== this.chatId) return;
        
        this.ui.displaySuccess(`Tool approved: ${data.toolName}`);
    }

    /**
     * Handle tool approval rejected event
     */
    private handleToolApprovalRejected(data: ToolApprovalData): void {
        if (data.chatId !== this.chatId) return;
        
        this.ui.displayWarning(`Tool rejected: ${data.toolName}`);
        if (data.reason) {
            this.ui.displayInfo(`Reason: ${data.reason}`);
        }
    }

    /**
     * Handle tool approval timeout event
     */
    private handleToolApprovalTimeout(data: ToolApprovalData): void {
        if (data.chatId !== this.chatId) return;
        
        this.ui.displayError(`Tool approval timed out: ${data.toolName}`);
    }

    /**
     * Handle tool execution events
     */
    private handleToolExecution(event: string, data: Record<string, unknown>): void {
        if (data.chatId !== this.chatId) return;
        
        // Delegate to tool approval handler if available
        if (this.toolApprovalHandler) {
            this.toolApprovalHandler.handleToolExecution(event, data);
        } else if (this.options.showTools) {
            // Display basic tool execution info even without approval handler
            const toolData = data as Record<string, unknown>;
            switch (event) {
                case "TOOL_CALLED":
                    if (toolData.toolName && typeof toolData.toolName === "string") {
                        const args = (typeof toolData.toolArguments === "object" && toolData.toolArguments !== null) 
                            ? toolData.toolArguments as Record<string, unknown>
                            : {};
                        this.ui.displayToolExecution(toolData.toolName, args);
                    }
                    break;
                case "TOOL_COMPLETED":
                    if (toolData.toolName) {
                        this.ui.displaySuccess(`Tool completed: ${toolData.toolName}`);
                    }
                    break;
                case "TOOL_FAILED":
                    if (toolData.toolName) {
                        this.ui.displayError(`Tool failed: ${toolData.toolName}`);
                    }
                    break;
            }
        }
    }

    /**
     * Handle tool status updates from approval handler
     */
    private handleToolStatusUpdate(toolInfo: ToolExecutionInfo): void {
        // Update session info or display additional information if needed
        // This can be extended for more sophisticated tool monitoring
        if (this.config.isDebug()) {
            console.log(`Tool ${toolInfo.toolName} status: ${toolInfo.status}`);
        }
    }

    /**
     * Cleanup resources and exit
     */
    async cleanup(): Promise<void> {
        this.isRunning = false;
        
        if (this.readline) {
            this.readline.close();
            this.readline = null;
        }
        
        if (this.socket) {
            this.socket.disconnect();
            this.socket = null;
        }
        
        if (this.toolApprovalHandler) {
            this.toolApprovalHandler.cleanup();
            this.toolApprovalHandler = null;
        }
        
        const duration = Date.now() - this.session.startTime.getTime();
        const durationMinutes = Math.round(duration / TIMEOUTS.CHAT_ENGINE_MINUTES_TO_MS);
        
        console.log();
        this.ui.displaySuccess(`Chat session ended. Duration: ${durationMinutes} minutes, Messages: ${this.session.messageCount}`);
        console.log();
        
        process.exit(0);
    }

    /**
     * Get participants from messages
     */
    private getParticipants(): User[] {
        const participantsMap = new Map<string, User>();
        
        for (const message of this.messages) {
            participantsMap.set(message.user.id, message.user);
        }
        
        return Array.from(participantsMap.values());
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
     * Get session information
     */
    getSessionInfo(): ChatSessionInfo {
        return { ...this.session };
    }
}

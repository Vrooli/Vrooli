import { spawn, type ChildProcess } from "child_process";
import { LlmServiceId, type MessageState, type ThirdPartyModelInfo, ModelFeature, type Tool, ClaudeCodeModel } from "@vrooli/shared";
import type { ChatCompletionMessageParam } from "openai/resources/chat/completions.js";
import { logger } from "../../../events/logger.js";
import { AIServiceErrorType } from "../registry.js";
import { TokenEstimatorType, type EstimateTokensResult } from "../tokenTypes.js";
import { AIService, type ResponseStreamOptions, type ServiceStreamEvent, type GetOutputTokenLimitParams, type GetOutputTokenLimitResult, type GetResponseCostParams } from "../services.js";
import { generateContextFromMessages } from "../contextGeneration.js";
import { hasHarmfulContent, hasPromptInjection } from "../messageValidation.js";
import { withStreamTimeout, StreamTimeoutError } from "../streamTimeout.js";

/**
 * COST CALCULATION DOCUMENTATION
 * 
 * The Claude Code service uses a monthly subscription model:
 * - All operations return 0 cost (monthly subscription already paid)
 * - No per-token charges for Claude Code usage
 * - Cost tracking is for resource monitoring only
 */

// Tool profile types
export type ToolProfile = "readonly" | "analysis" | "safe" | "full";

/**
 * Tool profiles for common use cases
 */
export const TOOL_PROFILES: Record<ToolProfile, string[]> = {
    readonly: ["Read", "Glob", "Grep", "LS"],
    analysis: ["Read", "Glob", "Grep", "WebFetch", "WebSearch", "LS"],
    safe: ["Read", "Edit", "Write", "Glob", "Grep", "LS", "MultiEdit"],
    full: ["Bash(*)", "Edit", "Write", "Read", "Glob", "Grep", "LS", "MultiEdit", "WebFetch", "WebSearch"],
};

/**
 * Claude Code service-specific configuration
 */
export interface ClaudeCodeServiceConfig {
    /** Specific tools to allow for this request */
    allowedTools?: string[];
    /** Named tool profile to use */
    toolProfile?: ToolProfile;
    /** Working directory for command execution */
    workingDirectory?: string;
    /** Session timeout in milliseconds */
    sessionTimeout?: number;
}

/**
 * Type guard for ClaudeCodeServiceConfig
 */
export function isClaudeCodeConfig(config: unknown): config is ClaudeCodeServiceConfig {
    if (!config || typeof config !== "object") return false;
    const c = config as Record<string, unknown>;
    return (!c.allowedTools || Array.isArray(c.allowedTools)) &&
           (!c.toolProfile || typeof c.toolProfile === "string") &&
           (!c.workingDirectory || typeof c.workingDirectory === "string") &&
           (!c.sessionTimeout || typeof c.sessionTimeout === "number");
}

// Constants
const DEFAULT_CONTEXT_WINDOW = 200_000; // 200K tokens - Claude's typical context window
const DEFAULT_MAX_OUTPUT_TOKENS = 8_192; // 8K tokens output limit
const MAX_INPUT_LENGTH = 200_000; // Large limit for Claude Code
const ZERO_COST = 0; // Monthly subscription model - no per-request cost

// Session management constants
const SESSION_TIMEOUT_MINUTES = 30;
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
const MINUTES_TO_MS = SECONDS_PER_MINUTE * MS_PER_SECOND;
const CLEANUP_INTERVAL_MINUTES = 5;
const PROCESS_KILL_DELAY_MS = 5000;
const MAX_ACTIVE_SESSIONS = 100; // Prevent unbounded growth

// Session management
const activeSessions = new Map<string, { lastUsed: number; sessionId: string }>();
const SESSION_TIMEOUT_MS = SESSION_TIMEOUT_MINUTES * MINUTES_TO_MS;

// Claude CLI event types based on observed output format
interface ClaudeSystemEvent {
    type: "system";
    subtype: "init";
    session_id: string;
    model: string;
    tools: string[];
    cwd?: string;
}

interface ClaudeAssistantEvent {
    type: "assistant";
    message: {
        id: string;
        content: Array<{ type: "text"; text: string }>;
        role: "assistant";
        model: string;
    };
    session_id: string;
}

interface ClaudeResultEvent {
    type: "result";
    subtype: "success" | "error_max_turns" | "error";
    is_error: boolean;
    duration_ms: number;
    num_turns: number;
    result: string;
    session_id: string;
    total_cost_usd?: number;
    usage?: {
        input_tokens: number;
        output_tokens: number;
        cache_creation_input_tokens?: number;
        cache_read_input_tokens?: number;
    };
}

interface ClaudeToolCallEvent {
    type: "tool_use";
    tool_name: string;
    tool_input: Record<string, unknown>;
    tool_use_id: string;
    session_id: string;
}

type ClaudeEvent = ClaudeSystemEvent | ClaudeAssistantEvent | ClaudeResultEvent | ClaudeToolCallEvent;

/**
 * ClaudeCodeService - Service for communicating with Claude Code CLI
 * 
 * This service spawns Claude CLI processes and parses their streaming JSON output.
 * It supports session management, tool calling, and full Claude Code functionality.
 */
export class ClaudeCodeService extends AIService<string, ThirdPartyModelInfo> {
    private readonly cliCommand: string;
    private readonly workingDirectory: string;
    private readonly allowedTools: string[];
    private cleanupIntervalId: NodeJS.Timeout | null = null;

    __id: LlmServiceId = LlmServiceId.ClaudeCode;
    featureFlags = { supportsStatefulConversations: true }; // Claude Code supports session resumption
    defaultModel = ClaudeCodeModel.Sonnet;

    constructor(options?: { 
        cliCommand?: string; 
        workingDirectory?: string; 
        allowedTools?: string[];
        defaultModel?: ClaudeCodeModel;
    }) {
        super();
        this.cliCommand = options?.cliCommand ?? "claude";
        this.workingDirectory = options?.workingDirectory ?? process.cwd();
        this.defaultModel = options?.defaultModel ?? ClaudeCodeModel.Sonnet;
        
        // Default tools that Claude Code supports
        this.allowedTools = options?.allowedTools ?? [
            "Bash(*)",           // Full bash access
            "Edit",              // File editing
            "Write",             // File writing  
            "Glob",              // File globbing
            "Grep",              // File searching
            "LS",                // Directory listing
            "Read",              // File reading
            "MultiEdit",         // Multiple file edits
            "WebFetch",          // Web content fetching
            "WebSearch",         // Web searching
        ];
        
        logger.info(`[ClaudeCodeService] Initialized with CLI: ${this.cliCommand}`);
        
        // Clean up old sessions periodically
        this.cleanupSessions();
        this.cleanupIntervalId = setInterval(() => this.cleanupSessions(), CLEANUP_INTERVAL_MINUTES * MINUTES_TO_MS);
    }

    /**
     * Clean up expired sessions and enforce max sessions limit
     */
    private cleanupSessions(): void {
        const now = Date.now();
        
        // Remove expired sessions
        for (const [key, session] of Array.from(activeSessions.entries())) {
            if (now - session.lastUsed > SESSION_TIMEOUT_MS) {
                activeSessions.delete(key);
            }
        }
        
        // If still over limit, remove oldest sessions
        if (activeSessions.size > MAX_ACTIVE_SESSIONS) {
            const sortedSessions = Array.from(activeSessions.entries())
                .sort((a, b) => a[1].lastUsed - b[1].lastUsed);
            
            const toRemove = sortedSessions.slice(0, activeSessions.size - MAX_ACTIVE_SESSIONS);
            for (const [key] of toRemove) {
                activeSessions.delete(key);
                logger.warn(`[ClaudeCodeService] Evicted old session due to max limit: ${key}`);
            }
        }
    }

    /**
     * Check if we support a specific model
     */
    async supportsModel(model: string): Promise<boolean> {
        // Support all Claude model aliases and full names
        const claudeModelPatterns = [
            /^(sonnet|opus|haiku)$/i,
            /^claude-/i,
        ];
        
        return claudeModelPatterns.some(pattern => pattern.test(model));
    }

    generateContext(messages: MessageState[], systemMessage?: string): ChatCompletionMessageParam[] {
        return generateContextFromMessages(messages, systemMessage, "ClaudeCode");
    }

    async *generateResponseStreaming(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        // Create inner generator for timeout wrapping
        const innerGenerator = this.generateResponseStreamingInternal(options);
        
        // Wrap with timeout protection
        yield* withStreamTimeout(innerGenerator, {
            serviceName: "ClaudeCode",
            modelName: options.model,
            signal: options.signal,
            timeoutMs: CLEANUP_INTERVAL_MINUTES * 2 * MINUTES_TO_MS, // 10 minute timeout for Claude Code
        });
    }
    
    private async *generateResponseStreamingInternal(options: ResponseStreamOptions): AsyncGenerator<ServiceStreamEvent> {
        if (!(await this.supportsModel(options.model))) {
            throw new Error(`Model ${options.model} not supported by Claude Code`);
        }

        // Build the command arguments
        const args = this.buildCommandArgs(options);
        
        // Determine working directory
        let workingDir = this.workingDirectory;
        const config = options.serviceConfig;
        if (isClaudeCodeConfig(config) && config.workingDirectory) {
            workingDir = config.workingDirectory;
        }
        
        // Spawn the Claude CLI process
        let process: ChildProcess;
        try {
            process = spawn(this.cliCommand, args, {
                cwd: workingDir,
                stdio: ["pipe", "pipe", "pipe"],
                env: globalThis.process.env,
            });
        } catch (error) {
            throw new Error(`Failed to spawn Claude CLI: ${error}`);
        }

        if (!process.stdout || !process.stderr) {
            throw new Error("Failed to get process streams");
        }

        // Handle process errors
        process.on("error", (error) => {
            logger.error("[ClaudeCodeService] Process error:", error);
        });

        let sessionId: string | null = null;
        const totalCost = ZERO_COST; // Always zero for Claude Code

        // Helper function for handling abort signals
        const abortHandler = () => {
            if (process && !process.killed) {
                process.kill("SIGTERM");
                // Schedule force kill as backup
                const killTimer = setTimeout(() => {
                    if (process && !process.killed) {
                        process.kill("SIGKILL");
                    }
                }, PROCESS_KILL_DELAY_MS);
                
                // Clear timer if process exits
                process.once("exit", () => clearTimeout(killTimer));
            }
        };

        // Track cleanup state
        let cleanupDone = false;
        const cleanup = () => {
            if (cleanupDone) return;
            cleanupDone = true;
            
            if (options.signal) {
                options.signal.removeEventListener("abort", abortHandler);
            }
            
            if (process && !process.killed) {
                process.kill("SIGTERM");
            }
        };
        
        try {
            // Handle abort signal
            if (options.signal) {
                // Check if already aborted
                if (options.signal.aborted) {
                    cleanup();
                    throw new Error("Request aborted");
                }
                
                options.signal.addEventListener("abort", abortHandler);
                
                // Clean up listener when process ends
                process.once("exit", () => {
                    options.signal?.removeEventListener("abort", abortHandler);
                });
            }

            // Parse streaming JSON output
            let buffer = "";
            
            for await (const chunk of process.stdout) {
                buffer += chunk.toString();
                
                // Process complete lines
                const lines = buffer.split("\n");
                buffer = lines.pop() || ""; // Keep incomplete line in buffer
                
                for (const line of lines) {
                    if (!line.trim()) continue;
                    
                    try {
                        const event: ClaudeEvent = JSON.parse(line);
                        
                        // Handle different event types
                        if (event.type === "system" && event.subtype === "init") {
                            sessionId = event.session_id;
                            logger.debug(`[ClaudeCodeService] Session started: ${sessionId}`);
                            
                        } else if (event.type === "assistant" && event.message?.content) {
                            // Extract text content from assistant message
                            for (const contentItem of event.message.content) {
                                if (contentItem.type === "text" && contentItem.text) {
                                    yield { type: "text", content: contentItem.text };
                                }
                            }
                            
                        } else if (event.type === "tool_use") {
                            // Handle tool calls
                            yield {
                                type: "function_call",
                                name: event.tool_name,
                                arguments: event.tool_input,
                                callId: event.tool_use_id,
                            };
                            
                        } else if (event.type === "result") {
                            // Handle completion
                            if (event.usage) {
                                // Store session for reuse if successful
                                if (sessionId && !event.is_error) {
                                    // Ensure we don't exceed max sessions
                                    if (activeSessions.size >= MAX_ACTIVE_SESSIONS) {
                                        this.cleanupSessions();
                                    }
                                    
                                    activeSessions.set(sessionId, {
                                        sessionId,
                                        lastUsed: Date.now(),
                                    });
                                }
                            }
                            
                            yield { type: "done", cost: totalCost };
                            return;
                        }
                        
                    } catch (parseError) {
                        // Skip malformed JSON lines
                        logger.debug("[ClaudeCodeService] Skipping malformed JSON:", line);
                    }
                }
            }
            
            // Wait for process to complete
            await new Promise((resolve, reject) => {
                process.on("exit", (code) => {
                    if (code === 0) {
                        resolve(code);
                    } else {
                        reject(new Error(`Claude CLI exited with code ${code}`));
                    }
                });
            });
            
        } catch (error) {
            cleanup();
            
            // If it's a timeout error, ensure process is killed
            if (error instanceof StreamTimeoutError) {
                if (process && !process.killed) {
                    process.kill("SIGKILL");
                }
            }
            
            throw error;
            
        } finally {
            // Ensure process cleanup with timeout
            cleanup();
            
            // Wait for process to actually terminate (with timeout)
            if (process && !process.killed) {
                await new Promise<void>((resolve) => {
                    const timeout = setTimeout(() => {
                        if (!process.killed) {
                            process.kill("SIGKILL");
                        }
                        resolve();
                    }, PROCESS_KILL_DELAY_MS);
                    
                    process.once("exit", () => {
                        clearTimeout(timeout);
                        resolve();
                    });
                });
            }
        }
    }

    /**
     * Get allowed tools based on service config
     */
    private getAllowedTools(options: ResponseStreamOptions): string[] {
        const config = options.serviceConfig;
        
        if (!isClaudeCodeConfig(config)) {
            return this.allowedTools; // Default tools
        }
        
        // TypeScript now knows config is ClaudeCodeServiceConfig
        
        // Priority 1: Explicit tool list
        if (config.allowedTools && config.allowedTools.length > 0) {
            return config.allowedTools;
        }
        
        // Priority 2: Tool profile
        if (config.toolProfile && TOOL_PROFILES[config.toolProfile]) {
            return TOOL_PROFILES[config.toolProfile];
        }
        
        // Priority 3: Constructor defaults
        return this.allowedTools;
    }

    /**
     * Build command line arguments for Claude CLI
     */
    private buildCommandArgs(options: ResponseStreamOptions): string[] {
        const args = [
            "-p",                              // Print mode (non-interactive)
            "--output-format", "stream-json",  // Streaming JSON output
            "--verbose",                       // Verbose output for stream-json
            "--model", this.getModel(options.model),
        ];

        // Add allowed tools based on service config
        const allowedTools = this.getAllowedTools(options);
        for (const tool of allowedTools) {
            args.push("--allowedTools", tool);
        }

        // Add max tokens if specified
        if (options.maxTokens) {
            args.push("--max-turns", "1"); // Single turn with token limit
        }

        // Add session resumption if available
        const sessionKey = this.buildSessionKey(options);
        const existingSession = activeSessions.get(sessionKey);
        if (existingSession) {
            args.push("--resume", existingSession.sessionId);
        }

        // Build prompt from messages and system message
        const prompt = this.buildPrompt(options.input, options.systemMessage);
        args.push(prompt);

        return args;
    }

    /**
     * Build a session key for session management
     */
    private buildSessionKey(options: ResponseStreamOptions): string {
        // Create a session key based on model and first few messages
        const messageHashes = options.input
            .slice(0, 3)
            .map(msg => msg.id)
            .join("-");
        return `${options.model}-${messageHashes}`;
    }

    /**
     * Build a prompt from messages and system message
     */
    private buildPrompt(messages: MessageState[], systemMessage?: string): string {
        const context = this.generateContext(messages, systemMessage);
        
        // Convert to a single prompt string
        const parts: string[] = [];
        
        if (systemMessage) {
            parts.push(`System: ${systemMessage}`);
        }
        
        for (const msg of context) {
            if (msg.role === "system") continue; // Already handled above
            parts.push(`${msg.role}: ${msg.content}`);
        }
        
        return parts.join("\n\n");
    }

    getContextSize(requestedModel?: string | null): number {
        const model = this.getModel(requestedModel);
        const modelInfo = this.getModelInfo()[model];
        return modelInfo?.contextWindow || DEFAULT_CONTEXT_WINDOW;
    }

    getModelInfo(): Record<string, ThirdPartyModelInfo> {
        // Return model info for Claude Code models
        return {
            [ClaudeCodeModel.Sonnet]: {
                enabled: true,
                name: "Claude Code Sonnet",
                descriptionShort: "Claude Sonnet via Claude Code CLI",
                inputCost: 0,  // Monthly subscription
                outputCost: 0, // Monthly subscription
                contextWindow: 200_000,
                maxOutputTokens: 8_192,
                features: {
                    [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution" },
                    [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access" },
                    [ModelFeature.FileSearch]: { type: "generic", notes: "File system access" },
                },
                supportsReasoning: true,
            },
            [ClaudeCodeModel.Opus]: {
                enabled: true,
                name: "Claude Code Opus",
                descriptionShort: "Claude Opus via Claude Code CLI",
                inputCost: 0,
                outputCost: 0,
                contextWindow: 200_000,
                maxOutputTokens: 4_096,
                features: {
                    [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution" },
                    [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access" },
                    [ModelFeature.FileSearch]: { type: "generic", notes: "File system access" },
                    [ModelFeature.Vision]: { type: "vision", notes: "Image understanding" },
                },
                supportsReasoning: false,
            },
            [ClaudeCodeModel.Haiku]: {
                enabled: true,
                name: "Claude Code Haiku",
                descriptionShort: "Claude Haiku via Claude Code CLI",
                inputCost: 0,
                outputCost: 0,
                contextWindow: 200_000,
                maxOutputTokens: 4_096,
                features: {
                    [ModelFeature.CodeInterpreter]: { type: "generic", notes: "Built-in code execution" },
                    [ModelFeature.FunctionCalling]: { type: "generic", notes: "Full tool access" },
                    [ModelFeature.FileSearch]: { type: "generic", notes: "File system access" },
                },
                supportsReasoning: false,
            },
        };
    }

    getMaxOutputTokens(requestedModel?: string | null): number {
        const model = this.getModel(requestedModel);
        const modelInfo = this.getModelInfo()[model];
        return modelInfo?.maxOutputTokens || DEFAULT_MAX_OUTPUT_TOKENS;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        // Claude Code has no cost constraints - return max possible
        return this.getMaxOutputTokens(params.model);
    }

    getResponseCost(_params: GetResponseCostParams): number {
        // Claude Code is subscription-based - always return 0 cost
        return ZERO_COST;
    }

    getEstimationInfo(): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return {
            estimationModel: TokenEstimatorType.Default,
            encoding: "cl100k_base", // Claude uses similar tokenization to GPT
        };
    }

    getModel(model?: string | null): string {
        if (!model) return this.defaultModel;
        
        // Map common model names to Claude Code equivalents
        const modelMapping: Record<string, string> = {
            "sonnet": ClaudeCodeModel.Sonnet,
            "opus": ClaudeCodeModel.Opus,
            "haiku": ClaudeCodeModel.Haiku,
            "claude-3.5-sonnet": ClaudeCodeModel.Claude_3_5_Sonnet,
            "claude-3-opus": ClaudeCodeModel.Claude_3_Opus,
            "claude-3-haiku": ClaudeCodeModel.Claude_3_Haiku,
            "claude-sonnet-4": ClaudeCodeModel.Claude_Sonnet_4,
        };
        
        return modelMapping[model.toLowerCase()] || model || this.defaultModel;
    }

    getErrorType(error: unknown): AIServiceErrorType {
        if (error instanceof Error) {
            const message = error.message.toLowerCase();
            
            if (message.includes("spawn") || message.includes("command not found")) {
                return AIServiceErrorType.Authentication; // CLI not available
            }
            
            if (message.includes("rate limit") || message.includes("too many requests")) {
                return AIServiceErrorType.RateLimit;
            }
            
            if (message.includes("timeout") || message.includes("killed")) {
                return AIServiceErrorType.Overloaded;
            }
            
            if (message.includes("invalid") || message.includes("unsupported")) {
                return AIServiceErrorType.InvalidRequest;
            }
        }
        
        return AIServiceErrorType.ApiError;
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        // Check for harmful content
        const harmfulCheck = hasHarmfulContent(input);
        if (harmfulCheck.isHarmful) {
            logger.warn("Claude Code safety check flagged harmful content", { 
                pattern: harmfulCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for prompt injection attempts
        const injectionCheck = hasPromptInjection(input);
        if (injectionCheck.isInjection) {
            logger.warn("Claude Code safety check flagged potential prompt injection", { 
                pattern: injectionCheck.pattern,
                inputLength: input.length, 
            });
            return { cost: 0, isSafe: false };
        }

        // Check for excessive length
        if (input.length > MAX_INPUT_LENGTH) {
            logger.warn("Claude Code safety check flagged excessive input length", { 
                inputLength: input.length,
                maxLength: MAX_INPUT_LENGTH,
            });
            return { cost: 0, isSafe: false };
        }

        // Claude Code has built-in safety measures
        return { cost: 0, isSafe: true };
    }

    getNativeToolCapabilities(): Array<Pick<Tool, "name" | "description">> {
        // Claude Code supports extensive native tool capabilities
        return [
            { name: "bash", description: "Execute bash commands with full system access" },
            { name: "edit", description: "Edit files with precise line-by-line modifications" },
            { name: "write", description: "Create and write new files" },
            { name: "read", description: "Read file contents with syntax highlighting" },
            { name: "glob", description: "Find files using glob patterns" },
            { name: "grep", description: "Search file contents with regex patterns" },
            { name: "web_fetch", description: "Fetch and analyze web content" },
            { name: "web_search", description: "Search the web for information" },
        ];
    }

    /**
     * Checks if the ClaudeCode service is healthy and available.
     * @returns true if the service is healthy and can accept requests, false otherwise
     */
    async isHealthy(): Promise<boolean> {
        try {
            // Check if the CLI tool is installed by attempting to run it with --version
            const process = spawn("claude", ["--version"], {
                timeout: 5000, // 5 second timeout
            });

            return new Promise<boolean>((resolve) => {
                let output = "";
                
                process.stdout?.on("data", (data) => {
                    output += data.toString();
                });

                process.on("error", (error) => {
                    logger.error(`[ClaudeCodeService] Health check error: ${error}`);
                    resolve(false);
                });

                process.on("close", (code) => {
                    if (code === 0 && output.includes("claude")) {
                        // CLI is installed and working
                        return resolve(true);
                    }
                    
                    logger.warn(`[ClaudeCodeService] Health check failed: CLI not available or returned code ${code}`);
                    resolve(false);
                });

                // Fallback timeout
                setTimeout(() => {
                    process.kill();
                    logger.warn("[ClaudeCodeService] Health check timeout");
                    resolve(false);
                }, 5000);
            });
        } catch (error) {
            logger.error(`[ClaudeCodeService] Health check error: ${error}`);
            return false;
        }
    }

    /**
     * Clean up resources when service is destroyed
     */
    destroy(): void {
        if (this.cleanupIntervalId) {
            clearInterval(this.cleanupIntervalId);
            this.cleanupIntervalId = null;
        }
        
        // Clear all active sessions
        activeSessions.clear();
        
        logger.info("[ClaudeCodeService] Service destroyed and resources cleaned up");
    }
}

import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { StaticProvider } from "./providers/StaticProvider.js";
import { DynamicProvider } from "./providers/DynamicProvider.js";
import { FileProvider } from "./providers/FileProvider.js";
import { CompletionCache } from "./cache/CompletionCache.js";
import type { CompletionResult, CompletionContext, CompletionProvider } from "./types.js";

export class CompletionEngine {
    private staticProvider: StaticProvider;
    private dynamicProvider: DynamicProvider;
    private fileProvider: FileProvider;
    private cache: CompletionCache;
    private providers: CompletionProvider[];
    
    constructor(
        private config: ConfigManager,
        private client: ApiClient,
    ) {
        this.staticProvider = new StaticProvider();
        this.dynamicProvider = new DynamicProvider(client, config);
        this.fileProvider = new FileProvider();
        this.cache = new CompletionCache(config);
        
        // Register providers in priority order
        this.providers = [
            this.staticProvider,
            this.fileProvider,
            this.dynamicProvider,
        ];
    }
    
    async getCompletions(args: string[]): Promise<CompletionResult[]> {
        const context = this.parseContext(args);
        
        // Find the appropriate provider
        for (const provider of this.providers) {
            if (provider.canHandle(context)) {
                // Check cache for dynamic completions
                if (provider === this.dynamicProvider) {
                    const cacheKey = this.getCacheKey(context);
                    const cached = await this.cache.get(cacheKey);
                    if (cached) {
                        return cached;
                    }
                    
                    // Fetch and cache
                    const results = await provider.getCompletions(context);
                    const COMPLETION_CACHE_TTL = 300; // 5 minutes
                    await this.cache.set(cacheKey, results, COMPLETION_CACHE_TTL);
                    return results;
                }
                
                return provider.getCompletions(context);
            }
        }
        
        return [];
    }
    
    private parseContext(args: string[]): CompletionContext {
        // Remove the program name and any global options
        const cleanArgs = this.removeGlobalOptions(args.slice(1));
        
        // The current partial word being completed
        const partial = cleanArgs[cleanArgs.length - 1] || "";
        const previousArgs = cleanArgs.slice(0, -1);
        
        // Determine context type
        if (previousArgs.length === 0) {
            // Completing a top-level command
            return {
                type: "command",
                partial,
                args: cleanArgs,
                options: {},
            };
        }
        
        const command = previousArgs[0];
        
        // Check if we're completing an option
        if (partial.startsWith("-")) {
            return {
                type: "option",
                command,
                subcommand: previousArgs[1],
                partial,
                args: cleanArgs,
                options: this.parseOptions(previousArgs),
            };
        }
        
        // Check if we're completing a subcommand
        if (previousArgs.length === 1) {
            return {
                type: "subcommand",
                command,
                partial,
                args: cleanArgs,
                options: {},
            };
        }
        
        const subcommand = previousArgs[1];
        
        // Check if this position expects a file
        if (this.isFileArgument(command, subcommand, previousArgs.length - 2)) {
            return {
                type: "file",
                command,
                subcommand,
                partial,
                args: cleanArgs,
                options: this.parseOptions(previousArgs),
            };
        }
        
        // Check if this position expects a resource ID
        if (this.isResourceArgument(command, subcommand, previousArgs.length - 2)) {
            return {
                type: "resource",
                command,
                subcommand,
                partial,
                args: cleanArgs,
                options: this.parseOptions(previousArgs),
                resourceType: this.getResourceType(command, subcommand),
            };
        }
        
        // Default to option completion
        return {
            type: "option",
            command,
            subcommand,
            partial,
            args: cleanArgs,
            options: this.parseOptions(previousArgs),
        };
    }
    
    private removeGlobalOptions(args: string[]): string[] {
        const globalOptions = ["-p", "--profile", "-d", "--debug", "--json", "-h", "--help", "-V", "--version"];
        const cleaned: string[] = [];
        
        for (let i = 0; i < args.length; i++) {
            const arg = args[i];
            
            if (globalOptions.includes(arg)) {
                // Skip this option and its value if it has one
                if ((arg === "-p" || arg === "--profile") && i + 1 < args.length) {
                    i++; // Skip the next argument (profile name)
                }
            } else {
                cleaned.push(arg);
            }
        }
        
        return cleaned;
    }
    
    private parseOptions(args: string[]): Record<string, unknown> {
        const options: Record<string, unknown> = {};
        
        for (let i = 0; i < args.length; i++) {
            const arg = args[i];
            
            if (arg.startsWith("-")) {
                const key = arg.replace(/^-+/, "");
                
                // Check if next arg is a value or another option
                if (i + 1 < args.length && !args[i + 1].startsWith("-")) {
                    options[key] = args[i + 1];
                    i++; // Skip the value
                } else {
                    options[key] = true;
                }
            }
        }
        
        return options;
    }
    
    private isFileArgument(command: string, subcommand: string, position: number): boolean {
        const fileCommands: Record<string, Record<string, number[]>> = {
            "routine": {
                "import": [0],
                "import-dir": [0],
                "export": [1], // Second argument after routine ID
                "validate": [0],
            },
            "agent": {
                "import": [0],
                "import-dir": [0],
                "export": [1],
                "validate": [0],
            },
            "team": {
                "import": [0],
                "export": [1],
            },
            "chat": {
                "interactive": [1], // Context files
            },
        };
        
        return fileCommands[command]?.[subcommand]?.includes(position) || false;
    }
    
    private isResourceArgument(command: string, subcommand: string, position: number): boolean {
        const resourceCommands: Record<string, Record<string, number[]>> = {
            "routine": {
                "export": [0],
                "get": [0],
                "run": [0],
            },
            "agent": {
                "export": [0],
                "get": [0],
            },
            "team": {
                "get": [0],
                "monitor": [0],
                "spawn": [0],
                "update": [0],
                "export": [0],
                "insights": [0],
            },
            "chat": {
                "create": [0], // Bot ID
                "show": [0],
                "send": [0],
                "interactive": [0], // Optional chat ID
            },
            "history": {
                "replay": [0],
                "alias": [1], // History entry ID
            },
        };
        
        return resourceCommands[command]?.[subcommand]?.includes(position) || false;
    }
    
    private getResourceType(command: string, subcommand: string): string {
        const resourceTypes: Record<string, Record<string, string>> = {
            "routine": {
                "export": "routine",
                "get": "routine",
                "run": "routine",
            },
            "agent": {
                "export": "agent",
                "get": "agent",
            },
            "team": {
                "get": "team",
                "monitor": "team",
                "spawn": "team",
                "update": "team",
                "export": "team",
                "insights": "team",
            },
            "chat": {
                "create": "bot",
                "show": "chat",
                "send": "chat",
                "interactive": "chat",
            },
            "history": {
                "replay": "history",
                "alias": "history",
            },
        };
        
        return resourceTypes[command]?.[subcommand] || "unknown";
    }
    
    private getCacheKey(context: CompletionContext): string {
        return `completion:${context.type}:${context.command}:${context.subcommand}:${context.resourceType}:${context.partial}`;
    }
}

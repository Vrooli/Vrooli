import type { CompletionProvider, CompletionResult, CompletionContext } from "../types.js";

interface CommandTree {
    [command: string]: {
        description: string;
        subcommands: {
            [subcommand: string]: {
                description: string;
                options: {
                    [option: string]: string;
                };
            };
        };
        globalOptions?: {
            [option: string]: string;
        };
    };
}

export class StaticProvider implements CompletionProvider {
    private commandTree: CommandTree = {
        auth: {
            description: "Authentication commands",
            subcommands: {
                login: {
                    description: "Login to your Vrooli account",
                    options: {
                        "-e": "Email address",
                        "--email": "Email address",
                        "-p": "Password (not recommended)",
                        "--password": "Password (not recommended)",
                        "--no-save": "Don't save credentials",
                    },
                },
                logout: {
                    description: "Logout and clear stored credentials",
                    options: {},
                },
                status: {
                    description: "Check authentication status",
                    options: {},
                },
                whoami: {
                    description: "Display current user information",
                    options: {},
                },
            },
        },
        routine: {
            description: "Manage routines",
            subcommands: {
                import: {
                    description: "Import a routine from a JSON file",
                    options: {
                        "--dry-run": "Validate without importing",
                        "--validate": "Perform extensive validation",
                    },
                },
                "import-dir": {
                    description: "Import all routines from a directory",
                    options: {
                        "--dry-run": "Validate without importing",
                        "--fail-fast": "Stop on first error",
                        "--pattern": "File pattern to match",
                    },
                },
                export: {
                    description: "Export a routine to a JSON file",
                    options: {
                        "-o": "Output file path",
                        "--output": "Output file path",
                    },
                },
                list: {
                    description: "List routines",
                    options: {
                        "-l": "Number of routines to show",
                        "--limit": "Number of routines to show",
                        "-s": "Search routines",
                        "--search": "Search routines",
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                        "--mine": "Show only my routines",
                    },
                },
                get: {
                    description: "Get routine details",
                    options: {},
                },
                validate: {
                    description: "Validate a routine JSON file",
                    options: {},
                },
                run: {
                    description: "Execute a routine",
                    options: {
                        "-i": "Input data as JSON",
                        "--input": "Input data as JSON",
                        "--watch": "Watch execution progress",
                    },
                },
                search: {
                    description: "Search for routines using semantic similarity",
                    options: {
                        "-l": "Number of results to return",
                        "--limit": "Number of results to return",
                        "-t": "Filter by routine type",
                        "--type": "Filter by routine type",
                        "-f": "Output format (table|json|ids)",
                        "--format": "Output format (table|json|ids)",
                        "--min-score": "Minimum similarity score (0-1)",
                    },
                },
                discover: {
                    description: "List all available routines for use as subroutines",
                    options: {
                        "--type": "Filter by routine type",
                        "--format": "Output format (table|json|mapping)",
                    },
                },
            },
        },
        agent: {
            description: "Manage AI agents for swarm orchestration",
            subcommands: {
                import: {
                    description: "Import an agent from a JSON file",
                    options: {
                        "--dry-run": "Validate without importing",
                        "--validate": "Perform extensive validation including routine checks",
                    },
                },
                "import-dir": {
                    description: "Import all agents from a directory",
                    options: {
                        "--dry-run": "Validate without importing",
                        "--fail-fast": "Stop on first error",
                        "--pattern": "File pattern to match",
                    },
                },
                export: {
                    description: "Export an agent to a JSON file",
                    options: {
                        "-o": "Output file path",
                        "--output": "Output file path",
                    },
                },
                list: {
                    description: "List agents",
                    options: {
                        "-l": "Number of agents to show",
                        "--limit": "Number of agents to show",
                        "-s": "Search agents by name or goal",
                        "--search": "Search agents by name or goal",
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                        "--mine": "Show only my agents",
                        "--role": "Filter by role (coordinator|specialist|monitor|bridge)",
                    },
                },
                get: {
                    description: "Get agent details",
                    options: {},
                },
                validate: {
                    description: "Validate an agent JSON file",
                    options: {
                        "--check-routines": "Validate referenced routine existence",
                    },
                },
                search: {
                    description: "Search for agents by goal or capability",
                    options: {
                        "-l": "Number of results to return",
                        "--limit": "Number of results to return",
                        "-r": "Filter by role",
                        "--role": "Filter by role",
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                    },
                },
            },
        },
        team: {
            description: "Manage teams for swarm orchestration",
            subcommands: {
                create: {
                    description: "Create a new team from configuration",
                    options: {
                        "-c": "Team configuration JSON file",
                        "--config": "Team configuration JSON file",
                        "-n": "Team name",
                        "--name": "Team name",
                        "-g": "Team goal",
                        "--goal": "Team goal",
                        "-t": "Deployment type (development|saas|appliance)",
                        "--type": "Deployment type (development|saas|appliance)",
                        "--gpu": "GPU percentage allocation",
                        "--ram": "RAM allocation in GB",
                        "--target-profit": "Target monthly profit in USD",
                    },
                },
                list: {
                    description: "List teams",
                    options: {
                        "-l": "Number of teams to show",
                        "--limit": "Number of teams to show",
                        "-s": "Search teams by name or goal",
                        "--search": "Search teams by name or goal",
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                        "--mine": "Show only my teams",
                        "-t": "Filter by deployment type",
                        "--type": "Filter by deployment type",
                    },
                },
                get: {
                    description: "Get team details",
                    options: {
                        "--show-blackboard": "Include blackboard insights",
                        "--show-stats": "Include detailed statistics",
                    },
                },
                monitor: {
                    description: "Monitor team performance and resource usage",
                    options: {
                        "--interval": "Update interval in seconds",
                        "--duration": "Monitoring duration in minutes",
                        "--show-blackboard": "Show blackboard insights",
                        "--show-stats": "Show detailed statistics",
                    },
                },
                spawn: {
                    description: "Spawn a new chat instance from team template",
                    options: {
                        "-n": "Name for the chat instance",
                        "--name": "Name for the chat instance",
                        "-c": "Client context as JSON",
                        "--context": "Client context as JSON",
                        "-t": "Specific task for this instance",
                        "--task": "Specific task for this instance",
                        "--auto-start": "Automatically start the chat",
                    },
                },
                update: {
                    description: "Update team configuration",
                    options: {
                        "--goal": "Update team goal",
                        "--prompt": "Update business prompt",
                        "--target-profit": "Update target profit in USD",
                        "--cost-limit": "Update cost limit in USD",
                    },
                },
                import: {
                    description: "Import team configuration from JSON file",
                    options: {
                        "--dry-run": "Validate without creating",
                    },
                },
                export: {
                    description: "Export team configuration to JSON file",
                    options: {
                        "-o": "Output file path",
                        "--output": "Output file path",
                    },
                },
                insights: {
                    description: "View team blackboard insights",
                    options: {
                        "-t": "Filter by insight type",
                        "--type": "Filter by insight type",
                        "-l": "Number of insights to show",
                        "--limit": "Number of insights to show",
                        "--min-confidence": "Minimum confidence score",
                    },
                },
            },
        },
        chat: {
            description: "Chat commands for interacting with bots",
            subcommands: {
                "list-bots": {
                    description: "List available bots",
                    options: {
                        "-s": "Search bots by name or description",
                        "--search": "Search bots by name or description",
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                        "-l": "Limit number of results",
                        "--limit": "Limit number of results",
                    },
                },
                create: {
                    description: "Start a new chat with a bot",
                    options: {
                        "-n": "Name for the chat",
                        "--name": "Name for the chat",
                        "-t": "Task or goal for the chat",
                        "--task": "Task or goal for the chat",
                        "--public": "Make chat public (default: private)",
                    },
                },
                list: {
                    description: "List your chats",
                    options: {
                        "-f": "Output format (table|json)",
                        "--format": "Output format (table|json)",
                        "-l": "Limit number of results",
                        "--limit": "Limit number of results",
                        "--mine": "Show only your own chats",
                        "-s": "Search chats by name",
                        "--search": "Search chats by name",
                    },
                },
                show: {
                    description: "Display chat history",
                    options: {
                        "-l": "Limit number of messages",
                        "--limit": "Limit number of messages",
                        "-f": "Output format (conversation|json)",
                        "--format": "Output format (conversation|json)",
                    },
                },
                send: {
                    description: "Send a message to a chat",
                    options: {
                        "-m": "AI model to use",
                        "--model": "AI model to use",
                        "--wait": "Wait for bot response",
                        "--timeout": "Response timeout in seconds",
                    },
                },
                interactive: {
                    description: "Enter interactive chat mode",
                    options: {
                        "-b": "Bot ID for new chat (required if no chat-id)",
                        "--bot": "Bot ID for new chat (required if no chat-id)",
                        "-n": "Name for new chat",
                        "--name": "Name for new chat",
                        "-t": "Task or goal for new chat",
                        "--task": "Task or goal for new chat",
                        "-m": "AI model to use",
                        "--model": "AI model to use",
                        "--show-tools": "Show bot tool executions",
                        "--approve-tools": "Enable interactive tool approval",
                        "--context-file": "Add files as context",
                        "--timeout": "Response timeout in seconds",
                    },
                },
            },
        },
        profile: {
            description: "Manage CLI profiles",
            subcommands: {
                list: {
                    description: "List all profiles",
                    options: {},
                },
                use: {
                    description: "Switch to a different profile",
                    options: {},
                },
                create: {
                    description: "Create a new profile",
                    options: {
                        "-u": "Server URL",
                        "--url": "Server URL",
                    },
                },
            },
        },
        completion: {
            description: "Manage shell completions",
            subcommands: {
                install: {
                    description: "Install shell completions",
                    options: {
                        "--shell": "Shell type (bash, zsh, fish)",
                    },
                },
                uninstall: {
                    description: "Uninstall shell completions",
                    options: {},
                },
            },
        },
        history: {
            description: "View and manage command history",
            subcommands: {
                list: {
                    description: "List recent commands",
                    options: {
                        "-n": "Number of entries to show",
                        "--count": "Number of entries to show",
                        "-c": "Filter by command",
                        "--command": "Filter by command",
                        "--success": "Show only successful commands",
                        "--failed": "Show only failed commands",
                        "-f": "Output format (table|json|script)",
                        "--format": "Output format (table|json|script)",
                    },
                },
                search: {
                    description: "Search command history",
                    options: {
                        "-n": "Number of results",
                        "--count": "Number of results",
                    },
                },
                replay: {
                    description: "Replay a command from history",
                    options: {
                        "--edit": "Edit before executing",
                    },
                },
                stats: {
                    description: "Show history statistics",
                    options: {},
                },
                browse: {
                    description: "Browse history interactively",
                    options: {},
                },
                alias: {
                    description: "Create an alias from a history entry",
                    options: {},
                },
                export: {
                    description: "Export command history",
                    options: {
                        "-o": "Output file",
                        "--output": "Output file",
                        "--format": "Export format (json|csv|script)",
                        "--since": "Export commands since date",
                    },
                },
                sync: {
                    description: "Sync history with server",
                    options: {
                        "--push": "Push local history to server",
                        "--pull": "Pull history from server",
                    },
                },
                clear: {
                    description: "Clear command history",
                    options: {
                        "--force": "Skip confirmation",
                    },
                },
            },
        },
    };
    
    canHandle(context: CompletionContext): boolean {
        return context.type === "command" || context.type === "subcommand" || context.type === "option";
    }
    
    async getCompletions(context: CompletionContext): Promise<CompletionResult[]> {
        switch (context.type) {
            case "command":
                return this.getCommands(context.partial);
            case "subcommand":
                return context.command ? this.getSubcommands(context.command, context.partial) : [];
            case "option":
                return context.command ? this.getOptions(context.command, context.subcommand, context.partial) : [];
            default:
                return [];
        }
    }
    
    private getCommands(partial: string): CompletionResult[] {
        return Object.entries(this.commandTree)
            .filter(([cmd]) => cmd.startsWith(partial))
            .map(([cmd, info]) => ({
                value: cmd,
                description: info.description,
                type: "command" as const,
            }));
    }
    
    private getSubcommands(command: string, partial: string): CompletionResult[] {
        const cmd = this.commandTree[command];
        if (!cmd) return [];
        
        return Object.entries(cmd.subcommands)
            .filter(([sub]) => sub.startsWith(partial))
            .map(([sub, info]) => ({
                value: sub,
                description: info.description,
                type: "subcommand" as const,
            }));
    }
    
    private getOptions(command: string, subcommand: string | undefined, partial: string): CompletionResult[] {
        const cmd = this.commandTree[command];
        
        let options: Record<string, string> = {};
        
        // Get subcommand-specific options (only if command is known)
        if (cmd && subcommand && cmd.subcommands[subcommand]) {
            options = { ...cmd.subcommands[subcommand].options };
        }
        
        // Add global command options if they exist (only if command is known)
        if (cmd && cmd.globalOptions) {
            options = { ...options, ...cmd.globalOptions };
        }
        
        // Always add global CLI options (even for unknown commands)
        const globalOptions: Record<string, string> = {
            "-p": "Use a specific profile",
            "--profile": "Use a specific profile",
            "-d": "Enable debug output",
            "--debug": "Enable debug output",
            "--json": "Output in JSON format",
            "-h": "Show help",
            "--help": "Show help",
            "-V": "Show version",
            "--version": "Show version",
        };
        
        options = { ...options, ...globalOptions };
        
        return Object.entries(options)
            .filter(([opt]) => opt.startsWith(partial))
            .map(([opt, desc]) => ({
                value: opt,
                description: desc,
                type: "option" as const,
            }));
    }
}

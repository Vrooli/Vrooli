export interface CompletionResult {
    value: string;
    description?: string;
    type: "command" | "subcommand" | "option" | "resource" | "file";
    metadata?: Record<string, unknown>;
}

export interface CompletionContext {
    type: "command" | "subcommand" | "option" | "resource" | "file";
    command?: string;
    subcommand?: string;
    partial: string;
    args: string[];
    options: Record<string, unknown>;
    resourceType?: string;
}

export interface CompletionProvider {
    canHandle(context: CompletionContext): boolean;
    getCompletions(context: CompletionContext): Promise<CompletionResult[]>;
}

export interface CompletionCache {
    get(key: string): Promise<CompletionResult[] | null>;
    set(key: string, value: CompletionResult[], ttl?: number): Promise<void>;
    isExpired(entry: { results: CompletionResult[]; timestamp: number; ttl: number }): boolean;
    clear(): Promise<void>;
}

export interface CommandDescription {
    name: string;
    description: string;
    subcommands?: CommandDescription[];
    options?: OptionDescription[];
}

export interface OptionDescription {
    name: string;
    description: string;
    type?: string;
    required?: boolean;
}

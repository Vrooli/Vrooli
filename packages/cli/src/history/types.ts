export interface HistoryEntry {
    id: string;
    command: string;
    args: string[];
    options: Record<string, unknown>;
    timestamp: Date;
    duration?: number;
    exitCode?: number;
    profile: string;
    userId?: string;
    success: boolean;
    error?: string;
    metadata?: {
        resourcesCreated?: string[];
        resourcesModified?: string[];
        apiCallsCount?: number;
        outputSize?: number;
        completedAt?: string;
        commandPattern?: string;
        [key: string]: unknown;
    };
}

export interface HistorySearchQuery {
    text?: string;
    command?: string;
    profile?: string;
    userId?: string;
    successOnly?: boolean;
    failedOnly?: boolean;
    startDate?: Date;
    endDate?: Date;
    limit?: number;
    offset?: number;
    before?: Date;
    after?: Date;
}

export interface HistoryStats {
    totalCommands: number;
    successfulCommands: number;
    failedCommands: number;
    uniqueCommands: number;
    avgDuration: number;
    lastCommand: Date;
    topCommands: Array<{
        command: string;
        count: number;
    }>;
    commandsByProfile: Record<string, number>;
    recentActivity: Array<{
        date: string;
        count: number;
    }>;
}

export interface HistoryStorage {
    add(entry: HistoryEntry): Promise<void>;
    get(id: string): Promise<HistoryEntry | null>;
    search(query: HistorySearchQuery): Promise<HistoryEntry[]>;
    getStats(): Promise<HistoryStats>;
    clear(): Promise<void>;
    delete(id: string): Promise<void>;
    export(format: "json" | "csv" | "script"): Promise<string>;
}

export interface HistoryExportOptions {
    format: "json" | "csv" | "script";
    output?: string;
    since?: Date;
    until?: Date;
    includeSuccessful?: boolean;
    includeFailed?: boolean;
    commands?: string[];
    profiles?: string[];
}

export interface HistoryImportOptions {
    format: "json" | "csv";
    merge?: boolean;
    overwrite?: boolean;
    validateEntries?: boolean;
}

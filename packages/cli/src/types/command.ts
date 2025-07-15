// Command-related types for the CLI
import type { AxiosRequestConfig } from "axios";

// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-14

export interface CommandOptions {
    profile?: string;
    [key: string]: unknown;
}

export interface AuthLoginOptions extends CommandOptions {
    username?: string;
    password?: string;
}

export interface ChatStartOptions extends CommandOptions {
    interactive?: boolean;
    context?: string[];
    bot?: string;
    timeout?: string;
}

export interface ChatListOptions extends CommandOptions {
    limit?: string;
    page?: string;
    sort?: string;
}

export interface RoutineRunOptions extends CommandOptions {
    inputs?: string[];
    timeout?: string;
    watch?: boolean;
}

export interface RoutineListOptions extends CommandOptions {
    limit?: string;
    page?: string;
    type?: string;
    owned?: boolean;
}

export interface RoutineUploadOptions extends CommandOptions {
    name?: string;
    description?: string;
    type?: string;
    force?: boolean;
}

export interface RoutineSearchOptions extends CommandOptions {
    limit?: string;
    type?: string;
    sort?: string;
}

// Error handling types
export interface CLIError extends Error {
    code?: string;
    statusCode?: number;
    details?: unknown;
}

// API response types
export interface PaginatedResponse<T> {
    edges: Array<{ node: T }>;
    pageInfo: {
        hasNextPage: boolean;
        hasPreviousPage: boolean;
        startCursor?: string;
        endCursor?: string;
        count: number;
    };
}

export interface RoutineSearchResult {
    publicId: string;
    name: string;
    description: string;
    type: string;
    score: number;
}

export interface ChatSearchResult {
    id: string;
    publicId: string;
    name?: string;
    created_at: string;
    messages?: number;
}

// Request configuration types
export interface AxiosRequestConfigWithRetry extends AxiosRequestConfig {
    _retry?: boolean;
}


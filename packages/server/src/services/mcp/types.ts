/**
 * Tool annotations providing metadata about a tool's behavior.
 */
export interface ToolAnnotations {
    /** Human-readable title for the tool */
    title?: string;
    /** If true, indicates the tool does not modify its environment */
    readOnlyHint?: boolean;
    /** If true, the tool may perform destructive updates */
    destructiveHint?: boolean;
    /** If true, calling the tool repeatedly with the same arguments has no additional effect */
    idempotentHint?: boolean;
    /** If true, the tool may interact with an "open world" of external entities */
    openWorldHint?: boolean;
}

export interface ToolInputSchema {
    type: string;
    properties: Record<string, any>;
    required?: string[];
}

/**
 * Interface representing a Tool in the MCP protocol.
 */
export interface Tool {
    /** Unique identifier for the tool */
    name: string;
    /** Human-readable description */
    description?: string;
    /** JSON Schema for the tool's parameters */
    inputSchema: ToolInputSchema | { oneOf: ToolInputSchema[] };
    /** Optional hints about tool behavior */
    annotations?: ToolAnnotations;
}

/**
 * Content item for tool response
 */
export interface ContentItem {
    type: string;
    text: string;
}

/**
 * Response structure for tool execution as expected by MCP SDK
 */
export interface ToolResponse {
    content: ContentItem[];
    isError?: boolean;
}

/**
 * Type for logger instance
 */
export interface Logger {
    info: (message: string, ...meta: any[]) => void;
    warn: (message: string, ...meta: any[]) => void;
    error: (message: string, ...meta: any[]) => void;
    debug: (message: string, ...meta: any[]) => void;
} 
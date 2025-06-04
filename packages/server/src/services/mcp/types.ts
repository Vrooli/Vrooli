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

/** Vrooli-specific tool definition (compatible with OpenAI function tool schema) */
export interface Tool {
    /** Name of the tool (must be a-z, A-Z, 0-9, _, or - and must be 64 chars or less) */
    name: string;
    /** Optional description of the tool */
    description?: string;
    /** Optional JSON schema for the input parameters */
    inputSchema?: Record<string, unknown>;
    /** Optional: Estimated cost of running the tool, as a string (e.g., for BigInt compatibility). */
    estimatedCost?: string;
    /** Optional annotations for how the tool should be used/displayed */
    annotations?: {
        /** Optional display name for the tool */
        title?: string;
        /** Optional hint that the tool does not modify state */
        readOnlyHint?: boolean;
        /** Optional hint that the tool interacts with the outside world */
        openWorldHint?: boolean;
        /** Custom Vrooli flag to indicate if it's a native AI tool (vs. an MCP/Swarm function) */
        isNativeAiTool?: boolean;
    };
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
    isError: boolean;
    content: Array<{ type: "text"; text: string } | { type: "image_url"; image_url: string }>;
    runId?: string;
    creditsUsed?: string;
}

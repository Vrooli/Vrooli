export enum MCPEndpoint {
    Execute = "/execute",
    ListTools = "/tools",
    SearchTools = "/search",
    RegisterTool = "/register",
}

export interface MCPTool {
    name: string;
    description: string;
    parameters: Record<string, unknown>;
}

export interface ExecuteToolParams {
    toolName: string;
    arguments: Record<string, unknown>;
}

export interface SearchToolsParams {
    query: string;
}

export type RegisterToolParams = MCPTool

export type ExecuteToolResponse = {
    success: boolean;
    result: unknown;
}

export type ListToolsResponse = {
    success: boolean;
    tools: MCPTool[];
}

export type SearchToolsResponse = {
    success: boolean;
    results: MCPTool[];
}

export type RegisterToolResponse = {
    success: boolean;
    tool: MCPTool;
} 

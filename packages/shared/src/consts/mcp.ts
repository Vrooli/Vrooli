export interface MCPTool {
    /** Unique identifier used to look up the tool. */
    id: string;
    /** Human-readable name of the tool. */
    name: string;
    /** Description of what the tool does. */
    description: string;
    /** Parameters/schema definition for the tool. */
    parameters: Record<string, unknown>;
    /** The function that gets executed when the tool is called. */
    execute: (args: Record<string, unknown>) => Promise<unknown>;
}

export interface ExecuteToolParams {
    /** The arguments to pass to the tool. */
    arguments: Record<string, unknown>;
    /** 
     * The unique identifier of the tool to execute.
     * Should be unique but meaningful, such as `$(sanitized-routine-name)-$(random-string)`.
     */
    name: string;
}

/**
 * Parameters for tool-specific JSON-RPC style requests.
 * This follows the MCP standard for method-based tool execution.
 */
export interface ToolSpecificRequest {
    /** The method to invoke on the tool. */
    method: string;
    /** The arguments to pass to the tool. */
    arguments: Record<string, unknown>;
    /** Optional query for search operations */
    query?: string;
}

export interface SearchToolsParams {
    query: string;
}

export type RegisterToolParams = MCPTool;

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

/**
 * Available root-level tools, ordered by (likely) frequency of use.
 */
export enum McpToolName {
    /** Get detailed parameters for other tools based on a variant */
    DefineTool = "define_tool",
    /** Sends a message to a new or existing chat. */
    SendMessage = "send_message",
    /** CRUD for *any* resource */
    ResourceManage = "resource_manage",
    /** Run a routine (dynamic tool) inline (synchronous, no run object created) or as a job (asynchronous, run object created) */
    RunRoutine = "run_routine",
    /** Starts a swarm session with a bot or team of bots */
    SpawnSwarm = "spawn_swarm",
}

/**
 * Available swarm tools (in addition to the standard MCP tools), ordered by (likely) frequency of use.
 */
export enum McpSwarmToolName {
    /** Update swarm's shared state */
    UpdateSwarmSharedState = "update_swarm_shared_state",
    /** End the swarm. Called when the goal is complete or limits are reached. */
    EndSwarm = "end_swarm",
}

/**
 * MCP Integration Module
 * 
 * Provides integration between the Vrooli execution architecture and
 * the Model Context Protocol (MCP) tool system.
 */

export { 
    IntegratedToolRegistry,
    convertToolResourceToTool,
    type IntegratedToolContext,
    type ToolApprovalStatus,
} from "./toolRegistry.js";
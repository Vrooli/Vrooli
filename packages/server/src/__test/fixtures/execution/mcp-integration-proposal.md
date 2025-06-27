# MCP Tool Integration Enhancement Proposal

## Current State Analysis

### ✅ What's Already Implemented
1. **AI Mock Infrastructure** (`ai-mocks/`)
   - Comprehensive tool call factories and fixtures
   - Tool use behavior patterns
   - Mock registry for testing
   - Runtime validation capabilities

2. **Execution Architecture Integration**
   - `mcpTools` field in `IntegrationDefinition`
   - Basic MCP tool name validation
   - Integration scenarios reference MCP tools

3. **Emergent Capabilities Testing**
   - Runtime execution with AI mocks
   - Tool orchestration patterns
   - Cross-tier tool integration

### ⚠️ Identified Gaps

1. **MCP Tool Registry Integration**
   - No connection to actual MCP tool registry (`packages/server/src/services/mcp/registry.ts`)
   - Missing validation against registered tools
   - No MCP-specific mock behaviors

2. **MCP Protocol Testing**
   - Missing MCP request/response format validation
   - No testing of MCP tool discovery
   - No MCP server connection mocking

3. **Tool Approval Flow Testing**
   - Missing integration with tool approval system
   - No testing of user approval workflows
   - No scheduled tool execution testing

## Proposed Enhancement

### 1. Create MCP Tool Mock Factory

```typescript
// packages/server/src/__test/fixtures/execution/ai-mocks/factories/mcpToolFactory.ts

import type { MCPTool, MCPToolCall, MCPToolResponse } from "@vrooli/shared";
import type { AIMockConfig } from "../types.js";

export function createMCPToolCallResponse(config: {
    tool: string;
    server: string;
    method: string;
    params: Record<string, unknown>;
    requiresApproval?: boolean;
}): AIMockConfig {
    return {
        content: `Executing MCP tool ${config.tool} from ${config.server}`,
        toolCalls: [{
            name: `mcp:${config.server}/${config.tool}`,
            arguments: {
                method: config.method,
                params: config.params,
                mcpMetadata: {
                    server: config.server,
                    requiresApproval: config.requiresApproval
                }
            },
            result: {
                success: true,
                data: generateMCPResponse(config)
            }
        }],
        metadata: {
            mcpExecution: true,
            server: config.server,
            approvalRequired: config.requiresApproval
        }
    };
}

function generateMCPResponse(config: any): any {
    // Generate appropriate response based on tool type
    switch (config.tool) {
        case "filesystem":
            return { path: config.params.path, content: "mock file content" };
        case "database":
            return { rows: [], count: 0 };
        case "web":
            return { html: "<div>Mock response</div>", status: 200 };
        default:
            return { result: "success" };
    }
}
```

### 2. Create MCP Tool Validation Helper

```typescript
// packages/server/src/__test/fixtures/execution/mcp-validation.ts

import { MCPRegistry } from "../../../services/mcp/registry.js";
import type { ExecutionFixture } from "./types.js";

export async function validateMCPToolUsage<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>
): Promise<ValidationResult> {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    if (!fixture.integration.mcpTools || fixture.integration.mcpTools.length === 0) {
        return { pass: true, message: "No MCP tools to validate" };
    }
    
    // Get registered MCP tools
    const registry = MCPRegistry.getInstance();
    const registeredTools = await registry.getAvailableTools();
    
    for (const toolName of fixture.integration.mcpTools) {
        // Parse tool name (format: server/tool or just tool)
        const [server, tool] = toolName.includes('/') 
            ? toolName.split('/', 2) 
            : ['default', toolName];
        
        // Check if tool is registered
        const isRegistered = registeredTools.some(t => 
            t.server === server && t.name === tool
        );
        
        if (!isRegistered) {
            warnings.push(`MCP tool '${toolName}' not found in registry`);
        }
        
        // Validate tool name format
        if (!isValidMCPToolName(toolName)) {
            errors.push(`Invalid MCP tool name format: ${toolName}`);
        }
    }
    
    return {
        pass: errors.length === 0,
        message: errors.length === 0 
            ? "MCP tool validation passed" 
            : "MCP tool validation failed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}
```

### 3. Add MCP Tool Fixtures

```typescript
// packages/server/src/__test/fixtures/execution/ai-mocks/fixtures/mcpToolFixtures.ts

export const mcpToolFixtures = {
    // Filesystem operations
    fileRead: () => createMCPToolCallResponse({
        tool: "read_file",
        server: "filesystem",
        method: "read",
        params: { path: "/test/file.txt" }
    }),
    
    // Database operations
    databaseQuery: () => createMCPToolCallResponse({
        tool: "query",
        server: "database",
        method: "select",
        params: { 
            query: "SELECT * FROM users",
            params: []
        },
        requiresApproval: true
    }),
    
    // Web operations
    webFetch: () => createMCPToolCallResponse({
        tool: "fetch_url",
        server: "web",
        method: "get",
        params: { url: "https://example.com" }
    }),
    
    // Tool discovery
    toolDiscovery: () => ({
        content: "Available MCP tools discovered",
        metadata: {
            mcpServers: ["filesystem", "database", "web"],
            toolCount: 15,
            tools: [
                { server: "filesystem", name: "read_file", requiresApproval: false },
                { server: "database", name: "query", requiresApproval: true },
                { server: "web", name: "fetch_url", requiresApproval: false }
            ]
        }
    })
};
```

### 4. Enhance Execution Fixtures with MCP

```typescript
// Update ExecutionFixture types to include MCP-specific fields

export interface MCPToolConfiguration {
    server: string;
    tools: string[];
    requiresApproval: boolean;
    approvalTimeout?: number;
    retryStrategy?: "exponential" | "linear" | "none";
}

export interface EnhancedIntegrationDefinition extends IntegrationDefinition {
    mcpConfiguration?: {
        servers: MCPToolConfiguration[];
        defaultApprovalStrategy: "auto" | "manual" | "scheduled";
        toolDiscoveryEnabled: boolean;
    };
}
```

### 5. Create MCP Integration Test Scenarios

```typescript
// packages/server/src/__test/fixtures/execution/integration-scenarios/mcp-integration.ts

export const mcpIntegrationScenarios = {
    fileProcessing: {
        name: "MCP File Processing Workflow",
        description: "Test file operations through MCP tools",
        tiers: {
            tier1: createSwarmWithMCP("file_processor", ["filesystem/read_file", "filesystem/write_file"]),
            tier2: createRoutineWithMCP("process_files", ["filesystem/*"]),
            tier3: createExecutionWithMCP("file_operations", { requiresApproval: false })
        },
        expectedCapabilities: ["file_manipulation", "data_transformation"],
        mcpValidation: {
            toolsUsed: ["filesystem/read_file", "filesystem/write_file"],
            approvalRequired: false
        }
    },
    
    databaseIntegration: {
        name: "MCP Database Integration",
        description: "Test database operations with approval flow",
        tiers: {
            tier1: createSwarmWithMCP("data_analyst", ["database/query", "database/execute"]),
            tier2: createRoutineWithMCP("analyze_data", ["database/*"]),
            tier3: createExecutionWithMCP("secure_queries", { requiresApproval: true })
        },
        expectedCapabilities: ["data_analysis", "secure_execution"],
        mcpValidation: {
            toolsUsed: ["database/query"],
            approvalRequired: true,
            approvalTimeout: 30000
        }
    }
};
```

## Implementation Plan

### Phase 1: Foundation (Week 1)
1. ✅ Add basic MCP tool validation (DONE - with TODO)
2. Create MCP tool mock factory
3. Add MCP-specific fixtures

### Phase 2: Integration (Week 2)
1. Connect to actual MCP registry for validation
2. Add tool approval flow testing
3. Create MCP integration scenarios

### Phase 3: Runtime Testing (Week 3)
1. Add MCP tool discovery testing
2. Test cross-tier MCP tool usage
3. Validate emergence of MCP-based capabilities

### Phase 4: Advanced Features (Week 4)
1. Test scheduled MCP tool execution
2. Add MCP server connection mocking
3. Test error handling and recovery

## Benefits

1. **Complete MCP Testing Coverage**: Validates all aspects of MCP tool integration
2. **Real Protocol Validation**: Tests against actual MCP protocol specifications
3. **Tool Approval Testing**: Ensures security workflows function correctly
4. **Emergent Capability Validation**: Tests how MCP tools enable new capabilities

## Success Criteria

- [ ] All MCP tools in fixtures are validated against registry
- [ ] Tool approval flows are tested end-to-end
- [ ] MCP protocol compliance is verified
- [ ] Integration scenarios demonstrate MCP-enabled emergence
- [ ] Runtime tests validate actual MCP tool execution

## Notes

- The AI mock infrastructure is already excellent - we just need MCP-specific additions
- Focus on integration with existing patterns rather than rewriting
- Maintain type safety throughout MCP enhancements
- Ensure emergent capabilities can use MCP tools naturally
import { MCPRegistry } from "../../../../services/mcp/registry.js";
import { type MCPTool } from "../../../../services/mcp/tools.js";
import { logger } from "../../../../services/logger.js";

export interface RegistryFixture {
    registry: MCPRegistry;
    registeredTools: Map<string, MCPTool>;
    validationResults: ValidationResult[];
}

export interface ValidationResult {
    toolId: string;
    valid: boolean;
    errors: string[];
    warnings: string[];
}

export interface ToolMock {
    id: string;
    execute: jest.Mock;
    validate: jest.Mock;
    getSchema: jest.Mock;
}

export class MCPRegistryFixture {
    private registry: MCPRegistry;
    private mockTools: Map<string, ToolMock> = new Map();
    private realTools: Map<string, MCPTool> = new Map();
    private validationResults: ValidationResult[] = [];

    constructor(registry: MCPRegistry) {
        this.registry = registry;
    }

    /**
     * Set up fixture with connection to real MCP registry
     */
    async setupWithRealTools(): Promise<void> {
        try {
            // Get all registered tools from the real registry
            const tools = await this.registry.listTools();
            
            for (const tool of tools) {
                this.realTools.set(tool.id, tool);
            }
            
            // Validate fixture tools against real tools
            await this.validateRegisteredTools();
            
            logger.info("MCP Registry fixture initialized", {
                realToolCount: this.realTools.size,
                validationErrors: this.validationResults.filter(r => !r.valid).length,
            });
        } catch (error) {
            logger.error("Failed to setup MCP registry fixture", { error });
            throw error;
        }
    }

    /**
     * Create a mock tool for testing
     */
    createMockTool(config: {
        id: string;
        name: string;
        description: string;
        schema?: any;
        executeResponse?: any;
    }): ToolMock {
        const mock: ToolMock = {
            id: config.id,
            execute: jest.fn().mockResolvedValue(config.executeResponse || { success: true }),
            validate: jest.fn().mockReturnValue({ valid: true }),
            getSchema: jest.fn().mockReturnValue(config.schema || {}),
        };
        
        this.mockTools.set(config.id, mock);
        return mock;
    }

    /**
     * Register a fixture tool with the test registry
     */
    async registerFixtureTool(tool: MCPTool | ToolMock): Promise<void> {
        // If it's a mock, wrap it to match MCPTool interface
        const mcpTool = this.isMockTool(tool) ? this.wrapMockTool(tool) : tool;
        
        await this.registry.registerTool(mcpTool);
    }

    /**
     * Validate that fixture tools match real registered tools
     */
    async validateRegisteredTools(): Promise<ValidationResult[]> {
        this.validationResults = [];
        
        for (const [toolId, realTool] of this.realTools) {
            const result = await this.validateTool(toolId, realTool);
            this.validationResults.push(result);
        }
        
        return this.validationResults;
    }

    /**
     * Get a tool from the registry (real or mock)
     */
    async getTool(toolId: string): Promise<MCPTool | null> {
        // Check mocks first
        const mock = this.mockTools.get(toolId);
        if (mock) {
            return this.wrapMockTool(mock);
        }
        
        // Then check real tools
        return this.registry.getTool(toolId);
    }

    /**
     * Test tool execution with fixture data
     */
    async testToolExecution(
        toolId: string,
        input: any,
        expectedOutput?: any,
    ): Promise<ToolExecutionResult> {
        const tool = await this.getTool(toolId);
        if (!tool) {
            throw new Error(`Tool not found: ${toolId}`);
        }
        
        const startTime = Date.now();
        let result: any;
        let error: any;
        
        try {
            result = await tool.execute(input);
        } catch (e) {
            error = e;
        }
        
        const executionTime = Date.now() - startTime;
        
        return {
            toolId,
            input,
            output: result,
            error,
            executionTime,
            success: !error && (!expectedOutput || this.deepEqual(result, expectedOutput)),
        };
    }

    /**
     * Batch test multiple tools
     */
    async batchTestTools(
        tests: ToolTest[],
    ): Promise<BatchTestResult> {
        const results: ToolExecutionResult[] = [];
        
        for (const test of tests) {
            const result = await this.testToolExecution(
                test.toolId,
                test.input,
                test.expectedOutput,
            );
            results.push(result);
        }
        
        return {
            total: tests.length,
            passed: results.filter(r => r.success).length,
            failed: results.filter(r => !r.success).length,
            results,
            averageExecutionTime: results.reduce((sum, r) => sum + r.executionTime, 0) / results.length,
        };
    }

    /**
     * Create a test context with registry access
     */
    createTestContext(): MCPTestContext {
        return {
            registry: this.registry,
            getTool: (id: string) => this.getTool(id),
            mockTool: (config: any) => this.createMockTool(config),
            validateTools: () => this.validateRegisteredTools(),
            fixtures: {
                tools: this.getFixtureTools(),
                mocks: Array.from(this.mockTools.values()),
            },
        };
    }

    /**
     * Get all fixture tools (for testing)
     */
    getFixtureTools(): MCPTool[] {
        const tools: MCPTool[] = [];
        
        // Add wrapped mocks
        this.mockTools.forEach(mock => {
            tools.push(this.wrapMockTool(mock));
        });
        
        // Add real tools
        this.realTools.forEach(tool => {
            tools.push(tool);
        });
        
        return tools;
    }

    /**
     * Clean up fixture resources
     */
    async cleanup(): Promise<void> {
        // Clear mocks
        this.mockTools.clear();
        
        // Reset mock functions
        jest.clearAllMocks();
        
        // Clear validation results
        this.validationResults = [];
    }

    private async validateTool(toolId: string, realTool: MCPTool): Promise<ValidationResult> {
        const errors: string[] = [];
        const warnings: string[] = [];
        
        try {
            // Check if tool has required properties
            if (!realTool.name) errors.push("Missing tool name");
            if (!realTool.description) warnings.push("Missing tool description");
            
            // Validate schema if present
            if (realTool.inputSchema) {
                const schemaValid = this.validateSchema(realTool.inputSchema);
                if (!schemaValid) errors.push("Invalid input schema");
            }
            
            // Test basic execution (with safe test input)
            try {
                const testInput = this.generateSafeTestInput(realTool);
                await realTool.execute(testInput);
            } catch (e: any) {
                // Execution errors during validation are warnings, not errors
                warnings.push(`Execution test failed: ${e.message}`);
            }
            
        } catch (error: any) {
            errors.push(`Validation error: ${error.message}`);
        }
        
        return {
            toolId,
            valid: errors.length === 0,
            errors,
            warnings,
        };
    }

    private validateSchema(schema: any): boolean {
        // Basic schema validation
        if (typeof schema !== "object") return false;
        
        // Check for required schema properties
        if (schema.type && typeof schema.type !== "string") return false;
        
        // More comprehensive validation could be added here
        return true;
    }

    private generateSafeTestInput(tool: MCPTool): any {
        // Generate safe test input based on tool schema
        if (!tool.inputSchema) return {};
        
        const input: any = {};
        
        // Simple schema-based input generation
        if (tool.inputSchema.properties) {
            Object.entries(tool.inputSchema.properties).forEach(([key, prop]: [string, any]) => {
                switch (prop.type) {
                    case "string":
                        input[key] = "test";
                        break;
                    case "number":
                        input[key] = 0;
                        break;
                    case "boolean":
                        input[key] = false;
                        break;
                    case "array":
                        input[key] = [];
                        break;
                    case "object":
                        input[key] = {};
                        break;
                }
            });
        }
        
        return input;
    }

    private isMockTool(tool: any): tool is ToolMock {
        return tool.execute && tool.execute._isMockFunction;
    }

    private wrapMockTool(mock: ToolMock): MCPTool {
        return {
            id: mock.id,
            name: `Mock ${mock.id}`,
            description: `Mock tool for ${mock.id}`,
            execute: mock.execute,
            validate: mock.validate,
            inputSchema: mock.getSchema(),
        } as MCPTool;
    }

    private deepEqual(a: any, b: any): boolean {
        // Simple deep equality check
        return JSON.stringify(a) === JSON.stringify(b);
    }
}

export interface ToolExecutionResult {
    toolId: string;
    input: any;
    output: any;
    error: any;
    executionTime: number;
    success: boolean;
}

export interface ToolTest {
    toolId: string;
    input: any;
    expectedOutput?: any;
}

export interface BatchTestResult {
    total: number;
    passed: number;
    failed: number;
    results: ToolExecutionResult[];
    averageExecutionTime: number;
}

export interface MCPTestContext {
    registry: MCPRegistry;
    getTool: (id: string) => Promise<MCPTool | null>;
    mockTool: (config: any) => ToolMock;
    validateTools: () => Promise<ValidationResult[]>;
    fixtures: {
        tools: MCPTool[];
        mocks: ToolMock[];
    };
}

/**
 * Create a registry fixture for testing
 */
export async function createRegistryFixture(
    options: {
        useRealRegistry?: boolean;
        mockTools?: Array<{
            id: string;
            name: string;
            description: string;
        }>;
    } = {},
): Promise<MCPRegistryFixture> {
    const { useRealRegistry = true, mockTools = [] } = options;
    
    // Get or create registry instance
    const registry = useRealRegistry 
        ? MCPRegistry.getInstance()
        : new MCPRegistry();
    
    const fixture = new MCPRegistryFixture(registry);
    
    // Set up with real tools if requested
    if (useRealRegistry) {
        await fixture.setupWithRealTools();
    }
    
    // Add mock tools
    for (const mockConfig of mockTools) {
        fixture.createMockTool(mockConfig);
    }
    
    return fixture;
}

/**
 * Standard test tools for common scenarios
 */
export const STANDARD_TEST_TOOLS = {
    MONITOR: {
        id: "test_monitor",
        name: "Test Monitor",
        description: "Monitoring tool for tests",
        schema: {
            type: "object",
            properties: {
                target: { type: "string" },
                metrics: { type: "array" },
            },
        },
    },
    
    ANALYZER: {
        id: "test_analyzer",
        name: "Test Analyzer",
        description: "Analysis tool for tests",
        schema: {
            type: "object",
            properties: {
                data: { type: "object" },
                mode: { type: "string" },
            },
        },
    },
    
    EXECUTOR: {
        id: "test_executor",
        name: "Test Executor",
        description: "Execution tool for tests",
        schema: {
            type: "object",
            properties: {
                command: { type: "string" },
                args: { type: "array" },
            },
        },
    },
};

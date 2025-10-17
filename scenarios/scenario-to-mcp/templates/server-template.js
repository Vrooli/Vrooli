#!/usr/bin/env node
/**
 * MCP Server Template
 * This is a template for generating MCP servers for Vrooli scenarios
 * Replace {{VARIABLES}} with actual values during generation
 */

const { Server } = require('@modelcontextprotocol/sdk/server/stdio');
const { exec } = require('child_process');
const { promisify } = require('util');
const axios = require('axios');

const execAsync = promisify(exec);

// Initialize MCP server
const server = new Server({
    name: '{{SCENARIO_NAME}}',
    version: '1.0.0',
    description: '{{SCENARIO_DESCRIPTION}}'
});

// Configuration
const config = {
    apiBase: process.env.API_BASE || 'http://localhost:{{API_PORT}}',
    mcpPort: process.env.MCP_PORT || {{MCP_PORT}},
    scenarioPath: '{{SCENARIO_PATH}}'
};

// Tool definitions
const tools = {{TOOLS_JSON}};

// Register tool list handler
server.setRequestHandler('tools/list', async () => ({
    tools: tools
}));

// Register tool call handler
server.setRequestHandler('tools/call', async (request) => {
    const { name, arguments: args } = request.params;
    
    try {
        const handler = toolHandlers[name];
        if (!handler) {
            throw new Error(`Unknown tool: ${name}`);
        }
        
        const result = await handler(args);
        
        return {
            content: [
                {
                    type: 'text',
                    text: typeof result === 'string' ? result : JSON.stringify(result, null, 2)
                }
            ]
        };
    } catch (error) {
        return {
            content: [
                {
                    type: 'text',
                    text: `Error: ${error.message}`
                }
            ],
            isError: true
        };
    }
});

// Tool handlers
const toolHandlers = {
    {{TOOL_HANDLERS}}
};

// Health check handler
server.setRequestHandler('health', async () => ({
    status: 'healthy',
    scenario: '{{SCENARIO_NAME}}',
    timestamp: new Date().toISOString(),
    config: config
}));

// Resources handler - list available resources
server.setRequestHandler('resources/list', async () => ({
    resources: [
        {
            uri: `scenario://${config.scenarioPath}`,
            name: '{{SCENARIO_NAME}}',
            mimeType: 'application/json'
        }
    ]
}));

// Start the server
async function main() {
    try {
        console.log(`Starting MCP server for {{SCENARIO_NAME}}`);
        console.log(`Configuration:`, config);
        
        await server.start();
        
        console.log('MCP server started successfully');
        console.log(`Available tools: ${tools.map(t => t.name).join(', ')}`);
        
        // Register with MCP registry if available
        try {
            await axios.post('http://localhost:{{REGISTRY_PORT}}/register', {
                scenario: '{{SCENARIO_NAME}}',
                port: config.mcpPort,
                tools: tools.map(t => t.name)
            });
            console.log('Registered with MCP registry');
        } catch (err) {
            console.log('Registry registration skipped (registry may not be running)');
        }
    } catch (error) {
        console.error('Failed to start MCP server:', error);
        process.exit(1);
    }
}

// Graceful shutdown handlers
process.on('SIGINT', async () => {
    console.log('Shutting down MCP server...');
    await server.stop();
    process.exit(0);
});

process.on('SIGTERM', async () => {
    console.log('Shutting down MCP server...');
    await server.stop();
    process.exit(0);
});

// Handle uncaught errors
process.on('uncaughtException', (error) => {
    console.error('Uncaught exception:', error);
    process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
    console.error('Unhandled rejection at:', promise, 'reason:', reason);
    process.exit(1);
});

// Start the server
main();
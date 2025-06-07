/**
 * Example: Using ExecutionArchitecture with Integrated Tool Registry
 * 
 * This example demonstrates how the ExecutionArchitecture factory properly
 * initializes and wires the tool infrastructure into the three-tier system.
 */

import { ExecutionArchitecture } from "./executionArchitecture.js";
import { generatePK } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";

async function demonstrateToolIntegration() {
    // 1. Create the execution architecture with tool integration
    const architecture = await ExecutionArchitecture.create({
        useRedis: false, // Use in-memory for demo
        telemetryEnabled: true,
        logger,
    });

    // 2. Get the integrated tool registry
    const toolRegistry = architecture.getToolRegistry();
    logger.info("Tool registry initialized and available", {
        registryReady: !!toolRegistry,
    });

    // 3. Get Tier 3 (Execution Intelligence) which uses the tool registry
    const tier3 = architecture.getTier3();
    
    // 4. Example: Execute a step that uses tools
    const stepId = generatePK();
    const request = {
        executionId: stepId,
        request: {
            stepId,
            stepType: "tool",
            inputs: {
                message: "Search the web for information about TypeScript decorators",
            },
            resources: {
                tools: [{
                    name: "web_search",
                    description: "Search the web for information",
                    parameters: {},
                    required: [],
                }],
                tokens: 10000,
                apiCalls: 10,
                computeTime: 60000,
                memory: 512,
                cost: 10,
            },
            config: {
                strategyPreference: "conversational",
                outputSchema: {
                    type: "object",
                    properties: {
                        searchResults: { type: "array" },
                        summary: { type: "string" },
                    },
                },
            },
            constraints: {
                timeout: 30000,
                maxTokens: 5000,
            },
        },
    };

    // Execute through Tier 3
    const result = await tier3.execute(request);
    
    logger.info("Step execution result", {
        success: result.status === "completed",
        hasOutput: !!result.result?.output,
        toolsUsed: result.metadata?.toolsUsed,
    });

    // 5. Check tool usage metrics
    const metrics = toolRegistry.getUsageMetrics("web_search");
    if (metrics) {
        logger.info("Tool usage metrics", {
            toolName: "web_search",
            totalCalls: metrics.calls,
            averageDuration: metrics.totalDuration / metrics.calls,
            errorRate: metrics.errors / metrics.calls,
        });
    }

    // 6. Clean up
    await architecture.stop();
}

// Example of registering a custom tool dynamically
async function registerCustomTool() {
    const architecture = await ExecutionArchitecture.create();
    const toolRegistry = architecture.getToolRegistry();
    
    // Register a custom tool for a specific run
    toolRegistry.registerDynamicTool({
        name: "custom_calculator",
        description: "Performs custom calculations",
        inputSchema: {
            type: "object",
            properties: {
                expression: { type: "string", description: "Math expression to evaluate" },
            },
            required: ["expression"],
        },
        annotations: {
            title: "Custom Calculator",
            readOnlyHint: false,
        },
    }, {
        runId: "run_123",
        scope: "run",
    });
    
    logger.info("Custom tool registered successfully");
    
    await architecture.stop();
}

// Run the examples
if (require.main === module) {
    demonstrateToolIntegration().catch(console.error);
    // registerCustomTool().catch(console.error);
}
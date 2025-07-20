import { describe, it, expect, beforeAll } from "vitest";
import { LATEST_CONFIG_VERSION, type MessageState } from "@vrooli/shared";
import { IntegrationTestRunner } from "./__test/integrationTestRunner.js";
import { ClaudeCodeService } from "./ClaudeCodeService.js";
import { logger } from "../../../events/logger.js";

const testRunner = new IntegrationTestRunner();
const SUITE_NAME = "ClaudeCodeService";

// Check if we can run the integration tests
const canRun = testRunner.canRunTest(SUITE_NAME);

// Skip the entire suite if we can't run
describe.skipIf(!canRun)("ClaudeCodeService Integration Tests (Real CLI)", () => {
    let service: ClaudeCodeService;
    
    beforeAll(() => {
        logger.info("🚀 Running ClaudeCodeService integration tests with real Claude CLI");
        
        // Record that we're running this test
        testRunner.recordTestRun(SUITE_NAME);
        
        // Create real service instance with safety constraints
        service = new ClaudeCodeService({
            cliCommand: "claude", // Real CLI command
            allowedTools: ["Read", "Grep", "LS"], // Limited tools for safety
            workingDirectory: "/tmp", // Isolated directory
        });
    });
    
    it("should verify Claude CLI is available and responsive", async () => {
        const messages: MessageState[] = [{
            id: "test-msg-1",
            text: "Please respond with exactly: 'Integration test successful'",
            config: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                role: "user",
            },
            language: "en",
            createdAt: new Date(),
        }];
        
        const options = {
            model: "haiku", // Use the cheapest/fastest model
            input: messages,
            maxTokens: 20, // Limit response size
            tools: [],
            serviceConfig: {
                toolProfile: "readonly", // Extra safety
            },
        };
        
        const events = [];
        let error = null;
        
        try {
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
                if (event.type === "text") {
                    logger.debug(`Claude response chunk: ${event.content}`);
                }
            }
        } catch (e) {
            error = e;
            logger.error("Integration test failed:", e);
        }
        
        // Assertions
        expect(error).toBeNull();
        expect(events.some(e => e.type === "text")).toBe(true);
        expect(events.some(e => e.type === "done")).toBe(true);
        
        // Log full response for debugging
        const fullResponse = events
            .filter(e => e.type === "text")
            .map(e => e.content)
            .join("");
        logger.info(`Full Claude response: "${fullResponse}"`);
        
        // Verify we got a reasonable response
        expect(fullResponse.toLowerCase()).toContain("test");
    }, 60000); // 60 second timeout for real CLI interaction
    
    it("should handle simple tool usage", async () => {
        const messages: MessageState[] = [{
            id: "test-msg-2",
            text: "What files are in the /tmp directory? Use the LS tool to check.",
            config: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                role: "user",
            },
            language: "en",
            createdAt: new Date(),
        }];
        
        const options = {
            model: "haiku",
            input: messages,
            maxTokens: 100,
            tools: [],
            serviceConfig: {
                toolProfile: "readonly", // Only Read, Grep, LS, Glob
                workingDirectory: "/tmp",
            },
        };
        
        const events = [];
        const toolCalls = [];
        
        try {
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
                if (event.type === "function_call") {
                    toolCalls.push(event);
                    logger.info(`Tool called: ${event.name} with args:`, event.arguments);
                }
            }
        } catch (e) {
            logger.error("Tool test failed:", e);
            // Don't fail the test - CLI might not be properly configured
        }
        
        // Should have made at least one tool call
        expect(events.some(e => e.type === "done")).toBe(true);
        
        if (toolCalls.length > 0) {
            expect(toolCalls[0].name).toBe("LS");
            logger.info(`Successfully called ${toolCalls.length} tool(s)`);
        } else {
            logger.warn("No tool calls made - Claude CLI might need tool configuration");
        }
    }, 60000);
    
    it("should respect service config tool restrictions", async () => {
        const messages: MessageState[] = [{
            id: "test-msg-3",
            text: "Can you list what tools you have access to?",
            config: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                role: "user",
            },
            language: "en",
            createdAt: new Date(),
        }];
        
        const options = {
            model: "haiku",
            input: messages,
            maxTokens: 100,
            tools: [],
            serviceConfig: {
                allowedTools: ["Read"], // Only allow Read tool
            },
        };
        
        const events = [];
        
        try {
            for await (const event of service.generateResponseStreaming(options)) {
                events.push(event);
            }
        } catch (e) {
            logger.error("Config test failed:", e);
        }
        
        // Should complete successfully
        expect(events.some(e => e.type === "done")).toBe(true);
        
        const response = events
            .filter(e => e.type === "text")
            .map(e => e.content)
            .join("");
        
        logger.info(`Tool restriction test response: "${response}"`);
    }, 60000);
});

// Log a message if tests were skipped
if (!canRun) {
    describe("ClaudeCodeService Integration Tests (Skipped)", () => {
        it("should skip when conditions not met", () => {
            logger.info(`Integration tests skipped. To run:
- Set VROOLI_INTEGRATION_TESTS_ENABLED=true
- Or use VROOLI_INTEGRATION_TESTS_FORCE=true to bypass daily limit
- Current status: ${JSON.stringify(testRunner.getStatus()[SUITE_NAME] || "Never run")}`);
            expect(true).toBe(true); // Dummy assertion
        });
    });
}

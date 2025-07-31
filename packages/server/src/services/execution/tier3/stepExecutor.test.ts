import {
    CallDataApiConfig,
    generatePK,
    type CodeLanguage,
    type RoutineVersionConfigObject,
    type StepExecutionInput,
    type SubroutineIOMapping,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, test, vi, type MockedFunction } from "vitest";
import { logger } from "../../../events/logger.js";
import { runUserCode } from "../../../tasks/sandbox/process.js";
import { APIKeyService } from "../../http/apiKeyService.js";
import { HTTPClient } from "../../http/httpClient.js";
import type { SwarmTools } from "../../mcp/tools.js";
import { McpToolRunner } from "../../response/toolRunner.js";
import { StepExecutor, type StepDefinition } from "./stepExecutor.js";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../../tasks/sandbox/process.js", () => ({
    runUserCode: vi.fn(),
}));

vi.mock("../../response/toolRunner.js", () => ({
    McpToolRunner: vi.fn().mockImplementation(() => ({
        run: vi.fn(),
    })),
}));

vi.mock("../../http/httpClient.js", () => ({
    HTTPClient: vi.fn().mockImplementation(() => ({
        makeRequest: vi.fn(),
    })),
}));

vi.mock("../../http/apiKeyService.js", () => ({
    APIKeyService: vi.fn().mockImplementation(() => ({
        getCredentials: vi.fn(),
    })),
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        nanoid: vi.fn(() => "test-nanoid-123"),
        CallDataApiConfig: vi.fn().mockImplementation((config) => ({
            schema: config,
        })),
    };
});

describe("StepExecutor", () => {
    let stepExecutor: StepExecutor;
    let mockMcpToolRunner: any;
    let mockHttpClient: any;
    let mockApiKeyService: any;
    let mockSwarmTools: SwarmTools;
    let mockRunUserCode: MockedFunction<typeof runUserCode>;

    beforeEach(() => {
        vi.clearAllMocks();

        // Setup mocks
        mockMcpToolRunner = {
            run: vi.fn(),
        };
        mockHttpClient = {
            makeRequest: vi.fn(),
        };
        mockApiKeyService = {
            getCredentials: vi.fn(),
        };
        mockSwarmTools = {} as SwarmTools;
        mockRunUserCode = runUserCode as MockedFunction<typeof runUserCode>;

        // Mock constructors to return our mocks
        (McpToolRunner as any).mockImplementation(() => mockMcpToolRunner);
        (HTTPClient as any).mockImplementation(() => mockHttpClient);
        (APIKeyService as any).mockImplementation(() => mockApiKeyService);

        stepExecutor = new StepExecutor(mockSwarmTools);
    });

    afterEach(() => {
        vi.resetAllMocks();
    });

    describe("constructor", () => {
        test("should initialize with swarm tools", () => {
            const executor = new StepExecutor(mockSwarmTools);
            expect(executor).toBeInstanceOf(StepExecutor);
            expect(McpToolRunner).toHaveBeenCalledWith(mockSwarmTools);
        });

        test("should initialize without swarm tools", () => {
            const executor = new StepExecutor();
            expect(executor).toBeInstanceOf(StepExecutor);
            expect(McpToolRunner).toHaveBeenCalledWith(undefined);
        });
    });

    describe("execute", () => {
        function createBasicRequest(overrides: Partial<TierExecutionRequest<StepExecutionInput>> = {}): TierExecutionRequest<StepExecutionInput> {
            return {
                input: {
                    stepId: "test-step-123",
                    type: "llm_call",
                    parameters: { prompt: "Hello, world!" },
                    strategy: "reasoning",
                    ...overrides.input,
                },
                context: {
                    userData: { languages: ["en"] },
                    ...overrides.context,
                },
                options: {
                    timeout: 30000,
                    ...overrides.options,
                },
            };
        }

        test("should execute LLM call successfully", async () => {
            const request = createBasicRequest({
                input: {
                    stepId: "llm-step",
                    type: "llm_call",
                    parameters: { prompt: "Test prompt" },
                    strategy: "conversational",
                },
            });

            const result = await stepExecutor.execute(request);

            expect(result.success).toBe(true);
            expect(result.result).toEqual({
                message: "Conversational response for step llm-step. I understand your request and would engage in natural dialogue.",
                finishReason: "stop",
                strategy: "conversational",
                model: "gpt-4",
            });
            expect(result.resourcesUsed.creditsUsed).toBe("100");
            expect(result.resourcesUsed.stepsExecuted).toBe(1);
            expect(result.duration).toBeGreaterThanOrEqual(0);
        });

        test("should execute tool call successfully", async () => {
            const request = createBasicRequest({
                input: {
                    stepId: "tool-step",
                    type: "tool_call",
                    parameters: {
                        tool: "test_tool",
                        arguments: { param1: "value1" },
                        conversationId: "conv-123",
                        sessionUser: { id: generatePK().toString() },
                    },
                },
            });

            mockMcpToolRunner.run.mockResolvedValue({
                ok: true,
                data: {
                    output: { result: "Tool executed successfully" },
                    creditsUsed: 25,
                },
            });

            const result = await stepExecutor.execute(request);

            expect(result.success).toBe(true);
            expect(result.result).toEqual({ result: "Tool executed successfully" });
            expect(result.resourcesUsed.creditsUsed).toBe("25");
            expect(result.resourcesUsed.toolCalls).toBe(1);
            expect(mockMcpToolRunner.run).toHaveBeenCalledWith(
                "test_tool",
                { param1: "value1" },
                {
                    conversationId: "conv-123",
                    sessionUser: { id: expect.any(String) },
                    callerBotId: undefined,
                },
            );
        });

        test("should execute code successfully", async () => {
            const request = createBasicRequest({
                input: {
                    stepId: "code-step",
                    type: "code_execution",
                    parameters: {
                        code: "console.log('Hello, world!');",
                        codeLanguage: "Javascript" as CodeLanguage,
                        input: { data: "test" },
                    },
                },
            });

            mockRunUserCode.mockResolvedValue({
                __type: "success",
                output: "Hello, world!",
            });

            const result = await stepExecutor.execute(request);

            expect(result.success).toBe(true);
            expect(result.result).toEqual({
                result: "Hello, world!",
                type: "success",
            });
            expect(mockRunUserCode).toHaveBeenCalledWith({
                code: "console.log('Hello, world!');",
                codeLanguage: "Javascript",
                input: { data: "test" },
            });
        });

        test("should execute API call successfully", async () => {
            const routineConfig: RoutineVersionConfigObject = {
                callDataApi: {
                    endpoint: "https://api.example.com/data/test-123",
                    method: "GET",
                    headers: {
                        "Authorization": "Bearer secret-token",
                        "Content-Type": "application/json",
                    },
                },
            };

            const ioMapping: SubroutineIOMapping = {
                inputs: {},
                outputs: {},
            };

            const request = createBasicRequest({
                input: {
                    stepId: "api-step",
                    type: "api_call",
                    parameters: {},
                    routineConfig,
                    ioMapping,
                },
            });

            // Mock CallDataApiConfig constructor properly
            (CallDataApiConfig as any).mockImplementation((config) => ({
                schema: config,
            }));

            mockHttpClient.makeRequest.mockResolvedValue({
                success: true,
                status: 200,
                statusText: "OK",
                headers: { "content-type": "application/json" },
                data: { result: "API call successful" },
                metadata: {
                    executionTime: 500,
                    url: "https://api.example.com/data/test-123",
                    method: "GET",
                    retries: 0,
                },
            });

            const result = await stepExecutor.execute(request);

            // Test that the API call was attempted and the http client was called
            expect(mockHttpClient.makeRequest).toHaveBeenCalledWith(
                expect.objectContaining({
                    url: "https://api.example.com/data/test-123",
                    method: "GET",
                    headers: expect.objectContaining({
                        "Authorization": "Bearer secret-token",
                        "Content-Type": "application/json",
                    }),
                }),
            );

            // Test that result has expected structure
            expect(result).toHaveProperty("success");
            expect(result).toHaveProperty("outputs");
            expect(result).toHaveProperty("metadata");
            expect(result.success).toBe(true);
        });

        test("should handle execution errors", async () => {
            const request = createBasicRequest({
                input: {
                    stepId: "error-step",
                    type: "code_execution",
                    parameters: { code: "throw new Error('Test error');" },
                },
            });

            mockRunUserCode.mockRejectedValue(new Error("Sandbox error"));

            const result = await stepExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error).toEqual({
                code: "EXECUTION_ERROR",
                message: "Sandbox error",
                tier: "tier3",
                type: "StepExecutionError",
            });
        });

        test("should handle tool call errors", async () => {
            const request = createBasicRequest({
                input: {
                    stepId: "tool-error-step",
                    type: "tool_call",
                    parameters: { tool: "failing_tool" },
                },
            });

            mockMcpToolRunner.run.mockResolvedValue({
                ok: false,
                error: {
                    message: "Tool execution failed",
                    creditsUsed: 5,
                },
            });

            const result = await stepExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error).toEqual({
                code: "EXECUTION_ERROR",
                message: "Tool execution failed",
                tier: "tier3",
                type: "StepExecutionError",
            });
        });
    });

    describe("inferStepType", () => {
        test("should infer llm_call from messages", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {
                    messages: [{ role: "user", content: "Hello" }],
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("llm_call");
        });

        test("should infer llm_call from prompt", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {
                    prompt: "Test prompt",
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("llm_call");
        });

        test("should infer tool_call from toolName", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {
                    tool: "test_tool",
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("tool_call");
        });

        test("should infer code_execution from code", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {
                    code: "console.log('test');",
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("code_execution");
        });

        test("should infer api_call from routineConfig.callDataApi", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {},
                routineConfig: {
                    callDataApi: {
                        endpoint: "https://api.example.com",
                        method: "GET",
                    },
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("api_call");
        });

        test("should default to llm_call for unclear input", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                parameters: {
                    someOtherParam: "value",
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("llm_call");
        });

        test("should respect explicit type", () => {
            const input: StepExecutionInput = {
                stepId: "test",
                type: "code_execution",
                parameters: {
                    prompt: "This looks like LLM but type says code",
                },
            };

            const stepDefinition = {
                id: "test",
                type: (stepExecutor as any).inferStepType(input),
                inputs: input.parameters,
            };

            expect(stepDefinition.type).toBe("code_execution");
        });
    });

    describe("buildConversationHistory", () => {
        test("should build history from messages array", () => {
            const step: StepDefinition = {
                id: "test-step",
                type: "llm_call",
                inputs: {
                    messages: [
                        { role: "user", content: "Hello", language: "en" },
                        { role: "assistant", content: "Hi there!" },
                    ],
                },
            };

            const history = (stepExecutor as any).buildConversationHistory(step);

            expect(history).toHaveLength(2);
            expect(history[0]).toEqual({
                id: expect.stringContaining("test-step"),
                text: "Hello",
                role: "user",
                language: "en",
            });
            expect(history[1]).toEqual({
                id: expect.stringContaining("test-step"),
                text: "Hi there!",
                role: "assistant",
                language: "en",
            });
        });

        test("should build history from single prompt", () => {
            const step: StepDefinition = {
                id: "test-step",
                type: "llm_call",
                inputs: {
                    prompt: "Generate a summary",
                },
            };

            const history = (stepExecutor as any).buildConversationHistory(step);

            expect(history).toHaveLength(1);
            expect(history[0]).toEqual({
                id: "test-step-prompt",
                text: "Generate a summary",
                role: "user",
                language: "en",
            });
        });

        test("should build generic history from other inputs", () => {
            const step: StepDefinition = {
                id: "test-step",
                type: "llm_call",
                inputs: {
                    topic: "AI development",
                    style: "formal",
                },
            };

            const history = (stepExecutor as any).buildConversationHistory(step);

            expect(history).toHaveLength(1);
            expect(history[0]).toEqual({
                id: "test-step-generic",
                text: "topic: \"AI development\"\nstyle: \"formal\"",
                role: "user",
                language: "en",
            });
        });

        test("should handle parameters.messages", () => {
            const step: StepDefinition = {
                id: "test-step",
                type: "llm_call",
                inputs: {
                    parameters: {
                        messages: [
                            { content: "Nested message", role: "user" },
                        ],
                    },
                },
            };

            const history = (stepExecutor as any).buildConversationHistory(step);

            expect(history).toHaveLength(1);
            expect(history[0].text).toBe("Nested message");
        });
    });

    describe("executeCode", () => {
        test("should execute Python code", async () => {
            const step: StepDefinition = {
                id: "python-step",
                type: "code_execution",
                inputs: {
                    code: "print('Hello from Python')",
                    codeLanguage: "Python",
                    input: { name: "test" },
                },
            };

            mockRunUserCode.mockResolvedValue({
                __type: "success",
                output: "Hello from Python\n",
            });

            const result = await (stepExecutor as any).executeCode(step);

            expect(result.success).toBe(true);
            expect(result.outputs).toEqual({
                result: "Hello from Python\n",
                type: "success",
            });
            expect(mockRunUserCode).toHaveBeenCalledWith({
                code: "print('Hello from Python')",
                codeLanguage: "Python",
                input: { name: "test" },
            });
        });

        test("should handle code execution errors", async () => {
            const step: StepDefinition = {
                id: "error-step",
                type: "code_execution",
                inputs: {
                    code: "raise ValueError('Test error')",
                    codeLanguage: "Python",
                },
            };

            mockRunUserCode.mockResolvedValue({
                __type: "error",
                error: "ValueError: Test error",
            });

            const result = await (stepExecutor as any).executeCode(step);

            expect(result.success).toBe(false);
            expect(result.error).toBe("ValueError: Test error");
        });

        test("should handle missing code", async () => {
            const step: StepDefinition = {
                id: "no-code-step",
                type: "code_execution",
                inputs: {},
            };

            const result = await (stepExecutor as any).executeCode(step);

            expect(result.success).toBe(false);
            expect(result.error).toBe("No code provided for execution");
        });

        test("should default language to Javascript", async () => {
            const step: StepDefinition = {
                id: "js-step",
                type: "code_execution",
                inputs: {
                    code: "console.log('test');",
                },
            };

            mockRunUserCode.mockResolvedValue({
                __type: "success",
                output: "test",
            });

            await (stepExecutor as any).executeCode(step);

            expect(mockRunUserCode).toHaveBeenCalledWith({
                code: "console.log('test');",
                codeLanguage: "Javascript",
                input: {},
            });
        });
    });

    describe("template processing", () => {
        test("should process simple placeholders", () => {
            const template = "Hello {{input.name}}, today is {{userLanguage}}";
            const config = {
                inputs: {
                    name: { value: "Alice" },
                },
                userLanguages: ["fr", "en"],
            };

            const result = (stepExecutor as any).processTemplate(template, config);

            expect(result).toBe("Hello Alice, today is fr");
        });

        test("should process object templates", () => {
            const template = {
                greeting: "Hello {{input.name}}",
                timestamp: "{{now()}}",
                id: "{{nanoid()}}",
            };
            const config = {
                inputs: {
                    name: { value: "Bob" },
                },
                userLanguages: ["en"],
                seededIds: {},
            };

            const result = (stepExecutor as any).processTemplateObject(template, config);

            expect(result.greeting).toBe("Hello Bob");
            expect(result.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T/); // ISO date format
            expect(result.id).toBe("test-nanoid-123");
        });

        test("should handle nested object templates", () => {
            const template = {
                user: {
                    name: "{{input.userName}}",
                    id: "{{nanoid(user)}}",
                },
                metadata: ["{{userLanguage}}", "{{now()}}"],
            };
            const config = {
                inputs: {
                    userName: { value: "Charlie" },
                },
                userLanguages: ["es"],
                seededIds: {},
            };

            const result = (stepExecutor as any).processTemplateValue(template, config);

            expect(result.user.name).toBe("Charlie");
            expect(result.user.id).toBe("test-nanoid-123");
            expect(result.metadata[0]).toBe("es");
            expect(result.metadata[1]).toMatch(/^\d{4}-\d{2}-\d{2}T/);
        });

        test("should handle missing input gracefully", () => {
            const template = "Value: {{input.missing}}";
            const config = {
                inputs: {},
                userLanguages: ["en"],
            };

            expect(() => {
                (stepExecutor as any).processTemplate(template, config);
            }).toThrow("Input \"missing\" not found in ioMapping");
        });

        test("should handle seeded nanoid", () => {
            const config = {
                inputs: {},
                userLanguages: ["en"],
                seededIds: { "testSeed": "existing-id" },
            };

            const result1 = (stepExecutor as any).getPlaceholderValue("nanoid(testSeed)", config);
            const result2 = (stepExecutor as any).getPlaceholderValue("nanoid(testSeed)", config);

            expect(result1).toBe("existing-id");
            expect(result2).toBe("existing-id");
        });

        test("should handle special functions", () => {
            const config = {
                inputs: {},
                userLanguages: ["de", "en"],
                seededIds: {},
            };

            const userLanguage = (stepExecutor as any).getPlaceholderValue("userLanguage", config);
            const userLanguages = (stepExecutor as any).getPlaceholderValue("userLanguages", config);
            const now = (stepExecutor as any).getPlaceholderValue("now()", config);
            const random = (stepExecutor as any).getPlaceholderValue("random()", config);

            expect(userLanguage).toBe("de");
            expect(userLanguages).toEqual(["de", "en"]);
            expect(now).toMatch(/^\d{4}-\d{2}-\d{2}T/);
            expect(typeof random).toBe("number");
            expect(random).toBeGreaterThanOrEqual(0);
            expect(random).toBeLessThan(1);
        });
    });

    describe("getValueFromPath", () => {
        test("should get nested values", () => {
            const obj = {
                data: {
                    user: {
                        name: "Alice",
                        profile: {
                            age: 30,
                        },
                    },
                    items: ["item1", "item2", "item3"],
                },
            };

            expect((stepExecutor as any).getValueFromPath(obj, "data.user.name")).toBe("Alice");
            expect((stepExecutor as any).getValueFromPath(obj, "data.user.profile.age")).toBe(30);
            expect((stepExecutor as any).getValueFromPath(obj, "data.items[1]")).toBe("item2");
        });

        test("should handle missing paths", () => {
            const obj = { data: { user: null } };

            expect((stepExecutor as any).getValueFromPath(obj, "data.user.name")).toBeUndefined();
            expect((stepExecutor as any).getValueFromPath(obj, "missing.path")).toBeUndefined();
        });

        test("should handle array access", () => {
            const obj = {
                users: [
                    { name: "Alice", scores: [10, 20, 30] },
                    { name: "Bob", scores: [15, 25, 35] },
                ],
            };

            expect((stepExecutor as any).getValueFromPath(obj, "users[0].name")).toBe("Alice");
            expect((stepExecutor as any).getValueFromPath(obj, "users[1].scores[2]")).toBe(35);
        });
    });

    describe("getStatus", () => {
        test("should return healthy status", async () => {
            const status = await stepExecutor.getStatus();

            expect(status).toEqual({
                healthy: true,
                tier: "tier3",
                activeExecutions: 0,
            });
        });
    });

    describe("LLM execution strategies", () => {
        test("should handle conversational strategy", async () => {
            const step: StepDefinition = {
                id: "conv-step",
                type: "llm_call",
                strategy: "conversational",
                inputs: { prompt: "Hello" },
            };

            const result = await (stepExecutor as any).executeLLMCall(step);

            expect(result.success).toBe(true);
            expect(result.outputs.message).toContain("Conversational response");
            expect(result.outputs.strategy).toBe("conversational");
        });

        test("should handle reasoning strategy", async () => {
            const step: StepDefinition = {
                id: "reasoning-step",
                type: "llm_call",
                strategy: "reasoning",
                inputs: { prompt: "Solve this problem" },
            };

            const result = await (stepExecutor as any).executeLLMCall(step);

            expect(result.success).toBe(true);
            expect(result.outputs.message).toContain("Reasoning response");
            expect(result.outputs.strategy).toBe("reasoning");
        });

        test("should handle deterministic strategy", async () => {
            const step: StepDefinition = {
                id: "det-step",
                type: "llm_call",
                strategy: "deterministic",
                inputs: { prompt: "Generate output" },
            };

            const result = await (stepExecutor as any).executeLLMCall(step);

            expect(result.success).toBe(true);
            expect(result.outputs.message).toContain("Deterministic response");
            expect(result.outputs.strategy).toBe("deterministic");
        });

        test("should default to reasoning strategy", async () => {
            const step: StepDefinition = {
                id: "default-step",
                type: "llm_call",
                inputs: { prompt: "Test" },
            };

            const result = await (stepExecutor as any).executeLLMCall(step);

            expect(result.success).toBe(true);
            expect(result.outputs.strategy).toBe("reasoning");
        });
    });

    describe("API call edge cases", () => {
        test("should require routineConfig and ioMapping for API calls", async () => {
            const step: StepDefinition = {
                id: "api-step",
                type: "api_call",
                inputs: {},
            };

            const result = await (stepExecutor as any).executeAPICall(step);

            expect(result.success).toBe(false);
            expect(result.error).toContain("API calls require routineConfig.callDataApi and ioMapping");
        });

        test("should handle API request failures", async () => {
            const step: StepDefinition = {
                id: "api-step",
                type: "api_call",
                inputs: {},
                routineConfig: {
                    callDataApi: {
                        endpoint: "https://api.example.com/test",
                        method: "GET",
                    },
                },
                ioMapping: { inputs: {}, outputs: {} },
            };

            // Mock CallDataApiConfig constructor properly
            (CallDataApiConfig as any).mockImplementation((config) => ({
                schema: config,
            }));

            mockHttpClient.makeRequest.mockRejectedValue(new Error("Network error"));

            const result = await (stepExecutor as any).executeAPICall(step);

            expect(result.success).toBe(false);
            expect(result.error).toBe("Network error");
        });

        test("should handle API response transformation", async () => {
            const step: StepDefinition = {
                id: "api-step",
                type: "api_call",
                inputs: {},
                routineConfig: {
                    callDataApi: {
                        endpoint: "https://api.example.com/test",
                        method: "GET",
                    },
                    callDataAction: {
                        schema: {
                            outputMapping: {
                                result: "data.value",
                                status: "status",
                            },
                        },
                    },
                },
                ioMapping: {
                    inputs: {},
                    outputs: {
                        result: { value: undefined },
                        status: { value: undefined },
                    },
                },
            };

            // Mock CallDataApiConfig constructor properly
            (CallDataApiConfig as any).mockImplementation((config) => ({
                schema: config,
            }));

            mockHttpClient.makeRequest.mockResolvedValue({
                success: true,
                status: 200,
                data: { value: "extracted_value" },
                metadata: {
                    executionTime: 300,
                    url: "https://api.example.com/test",
                    method: "GET",
                    retries: 0,
                },
            });

            const result = await (stepExecutor as any).executeAPICall(step);

            expect(result.success).toBe(true);
            expect(result.outputs.result).toBe("extracted_value");
            expect(result.outputs.status).toBe(200);
            expect(step.ioMapping.outputs.result.value).toBe("extracted_value");
            expect(step.ioMapping.outputs.status.value).toBe(200);
        });
    });

    describe("error handling", () => {
        test("should handle unknown step type", async () => {
            const step: StepDefinition = {
                id: "unknown-step",
                type: "unknown_type" as any,
                inputs: {},
            };

            const result = await (stepExecutor as any).executeStep(step);

            expect(result.success).toBe(false);
            expect(result.error).toContain("Unknown step type: unknown_type");
        });

        test("should log errors appropriately", async () => {
            const step: StepDefinition = {
                id: "error-step",
                type: "code_execution",
                inputs: {
                    code: "invalid code",
                },
            };

            // Make runUserCode throw an error
            mockRunUserCode.mockRejectedValue(new Error("Sandbox execution failed"));

            await (stepExecutor as any).executeStep(step);

            expect(logger.error).toHaveBeenCalledWith(
                "[StepExecutor] Code execution failed",
                expect.objectContaining({
                    stepId: "error-step",
                    error: "Sandbox execution failed",
                }),
            );
        });
    });
});

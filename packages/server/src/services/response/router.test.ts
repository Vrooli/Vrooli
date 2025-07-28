import {
    API_CREDITS_MULTIPLIER,
    OpenAIModel,
    type MessageState,
    type SessionUser,
} from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CustomError } from "../../events/error.js";
import { AIServiceErrorType, AIServiceRegistry } from "./registry.js";
import {
    FallbackRouter,
    type StreamEvent,
    type StreamOptions,
} from "./router.js";
import type { AIService } from "./services.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
    },
}));

vi.mock("./registry.js", () => ({
    AIServiceRegistry: {
        get: vi.fn(),
    },
    AIServiceErrorType: {
        ApiError: "ApiError",
        Authentication: "Authentication",
        RateLimit: "RateLimit",
        InvalidRequest: "InvalidRequest",
    },
}));

vi.mock("./credits.js", () => ({
    calculateMaxCredits: vi.fn().mockReturnValue(BigInt(1000) * API_CREDITS_MULTIPLIER),
}));

// Helper to create async generator from events
function createMockServiceStream(events: any[]): AsyncGenerator<any> {
    return (async function* () {
        for (const event of events) {
            yield event;
        }
    })();
}

// Mock AI service with streaming support
const createMockService = (serviceId: string, streamEvents?: any[]): AIService => {
    const defaultEvents = [
        { type: "text", content: `Hello from ${serviceId}` },
        { type: "done", cost: 100 },
    ];

    return {
        __id: serviceId,
        estimateTokens: vi.fn().mockReturnValue({ tokens: 100 }),
        getMaxOutputTokensRestrained: vi.fn().mockReturnValue(1000),
        generateResponseStreaming: vi.fn().mockReturnValue(
            createMockServiceStream(streamEvents || defaultEvents),
        ),
        getErrorType: vi.fn().mockReturnValue(AIServiceErrorType.ApiError),
    } as unknown as AIService;
};

describe("FallbackRouter", () => {
    let router: FallbackRouter;
    let mockRegistry: any;
    let mockService: AIService;

    beforeEach(() => {
        vi.clearAllMocks();

        // Create mock service
        mockService = createMockService("test-service");

        // Mock registry
        mockRegistry = {
            getBestService: vi.fn().mockReturnValue("test-service"),
            getService: vi.fn().mockReturnValue(mockService),
            updateServiceState: vi.fn(),
        };

        (AIServiceRegistry.get as any).mockReturnValue(mockRegistry);

        router = new FallbackRouter();
    });

    const createStreamOptions = (model = OpenAIModel.Gpt4o): StreamOptions => ({
        model,
        input: [
            {
                id: "1",
                text: "Hello",
                config: { role: "user" },
                userId: "user1",
                language: "en",
            } as MessageState,
        ],
        tools: [],
        userData: {
            id: "user1",
            credits: "10000",
        } as SessionUser,
        maxCredits: BigInt(1000) * API_CREDITS_MULTIPLIER,
    });

    describe("defaultTools", () => {
        it("should return empty array for FallbackRouter", () => {
            expect(router.defaultTools()).toEqual([]);
        });
    });

    describe("stream", () => {
        describe("basic streaming", () => {
            it("should stream simple text response", async () => {
                const events = [
                    { type: "text", content: "Hello " },
                    { type: "text", content: "world!" },
                    { type: "done", cost: 150 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(events),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents).toEqual([
                    { type: "message", content: "Hello ", responseId: expect.stringContaining("test-service:") },
                    { type: "message", content: "world!", responseId: expect.stringContaining("test-service:") },
                    { type: "done", cost: 150, responseId: expect.stringContaining("test-service:") },
                ]);
            });

            it("should handle function calls in stream", async () => {
                const events = [
                    { type: "text", content: "Let me search for that" },
                    {
                        type: "function_call",
                        name: "search",
                        arguments: { query: "test" },
                        callId: "call_123",
                    },
                    { type: "done", cost: 200 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(events),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents).toHaveLength(3);
                expect(collectedEvents[1]).toEqual({
                    type: "function_call",
                    name: "search",
                    arguments: { query: "test" },
                    callId: "call_123",
                    responseId: expect.stringContaining("test-service:"),
                });
            });

            it("should handle reasoning events", async () => {
                const events = [
                    { type: "reasoning", content: "Let me think about this..." },
                    { type: "text", content: "Based on my reasoning, the answer is:" },
                    { type: "done", cost: 120 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(events),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents[0]).toEqual({
                    type: "reasoning",
                    content: "Let me think about this...",
                    responseId: expect.stringContaining("test-service:"),
                });
            });
        });

        describe("service failover", () => {
            it("should retry with next service on first service failure", async () => {
                const failingService = createMockService("failing-service");
                const workingService = createMockService("working-service", [
                    { type: "text", content: "Fallback response" },
                    { type: "done", cost: 100 },
                ]);

                // First service throws error
                failingService.generateResponseStreaming = vi.fn().mockImplementation(async function* () {
                    throw new Error("Service unavailable");
                });

                mockRegistry.getBestService
                    .mockReturnValueOnce("failing-service")
                    .mockReturnValueOnce("working-service");

                mockRegistry.getService
                    .mockImplementation((id: string) => {
                        if (id === "failing-service") return failingService;
                        if (id === "working-service") return workingService;
                        return null;
                    });

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents).toEqual([
                    { type: "message", content: "Fallback response", responseId: expect.stringContaining("working-service:") },
                    { type: "done", cost: 100, responseId: expect.stringContaining("working-service:") },
                ]);

                expect(mockRegistry.updateServiceState).toHaveBeenCalledWith(
                    "failing-service",
                    AIServiceErrorType.ApiError,
                );
            });

            it("should throw error when all services fail", async () => {
                mockRegistry.getBestService
                    .mockReturnValueOnce("service1")
                    .mockReturnValueOnce("service2")
                    .mockReturnValueOnce("service3")
                    .mockReturnValueOnce(null);

                const failingService = createMockService("failing");
                failingService.generateResponseStreaming = vi.fn().mockImplementation(async function* () {
                    throw new Error("All services down");
                });

                mockRegistry.getService.mockReturnValue(failingService);

                const options = createStreamOptions();

                await expect(async () => {
                    const stream = router.stream(options);
                    for await (const event of stream) {
                        // Should not reach here
                    }
                }).rejects.toThrow("All services down");
            });

            it("should respect retry limit", async () => {
                const failingService = createMockService("failing");
                failingService.generateResponseStreaming = vi.fn().mockImplementation(async function* () {
                    throw new Error("Persistent failure");
                });

                // Return same service ID repeatedly to test retry limit
                mockRegistry.getBestService.mockReturnValue("failing");
                mockRegistry.getService.mockReturnValue(failingService);

                const options = createStreamOptions();

                await expect(async () => {
                    const stream = router.stream(options);
                    for await (const event of stream) {
                        // Should not reach here
                    }
                }).rejects.toThrow("Persistent failure");

                // Should have tried 3 times (retry limit)
                expect(mockRegistry.getBestService).toHaveBeenCalledTimes(3);
            });
        });

        describe("concurrent streaming", () => {
            it("should handle multiple concurrent streams", async () => {
                const service1Events = [
                    { type: "text", content: "Response 1" },
                    { type: "done", cost: 100 },
                ];

                const service2Events = [
                    { type: "text", content: "Response 2" },
                    { type: "done", cost: 120 },
                ];

                const service3Events = [
                    { type: "text", content: "Response 3" },
                    { type: "done", cost: 90 },
                ];

                // Create separate services with different responses
                const services = [
                    createMockService("service1", service1Events),
                    createMockService("service2", service2Events),
                    createMockService("service3", service3Events),
                ];

                mockRegistry.getService
                    .mockReturnValueOnce(services[0])
                    .mockReturnValueOnce(services[1])
                    .mockReturnValueOnce(services[2]);

                const options1 = createStreamOptions();
                const options2 = createStreamOptions();
                const options3 = createStreamOptions();

                // Start all streams concurrently
                const [events1, events2, events3] = await Promise.all([
                    (async () => {
                        const events = [];
                        for await (const event of router.stream(options1)) {
                            events.push(event);
                        }
                        return events;
                    })(),
                    (async () => {
                        const events = [];
                        for await (const event of router.stream(options2)) {
                            events.push(event);
                        }
                        return events;
                    })(),
                    (async () => {
                        const events = [];
                        for await (const event of router.stream(options3)) {
                            events.push(event);
                        }
                        return events;
                    })(),
                ]);

                // All should complete successfully
                expect(events1).toHaveLength(2);
                expect(events2).toHaveLength(2);
                expect(events3).toHaveLength(2);

                expect(events1[0]).toMatchObject({ type: "message", content: "Response 1" });
                expect(events2[0]).toMatchObject({ type: "message", content: "Response 2" });
                expect(events3[0]).toMatchObject({ type: "message", content: "Response 3" });
            });

            it("should handle mixed success/failure in concurrent streams", async () => {
                const workingService = createMockService("working", [
                    { type: "text", content: "Success" },
                    { type: "done", cost: 100 },
                ]);

                const failingService = createMockService("failing");
                failingService.generateResponseStreaming = vi.fn().mockImplementation(async function* () {
                    throw new Error("Stream failed");
                });

                mockRegistry.getService
                    .mockReturnValueOnce(workingService)
                    .mockReturnValueOnce(failingService);

                const options1 = createStreamOptions();
                const options2 = createStreamOptions();

                const results = await Promise.allSettled([
                    (async () => {
                        const events = [];
                        for await (const event of router.stream(options1)) {
                            events.push(event);
                        }
                        return events;
                    })(),
                    (async () => {
                        const events = [];
                        for await (const event of router.stream(options2)) {
                            events.push(event);
                        }
                        return events;
                    })(),
                ]);

                expect(results[0].status).toBe("fulfilled");
                expect(results[1].status).toBe("rejected");

                if (results[0].status === "fulfilled") {
                    expect(results[0].value).toHaveLength(2);
                    expect(results[0].value[0]).toMatchObject({ content: "Success" });
                }
            });
        });

        describe("credit management", () => {
            it("should use calculated max credits for token budgeting", async () => {
                const options = createStreamOptions();
                options.maxCredits = BigInt(500) * API_CREDITS_MULTIPLIER;

                await router.stream(options).next();

                expect(mockService.getMaxOutputTokensRestrained).toHaveBeenCalledWith({
                    model: options.model,
                    maxCredits: BigInt(1000) * API_CREDITS_MULTIPLIER, // From mocked calculateMaxCredits
                    inputTokens: 100, // From mocked estimateTokens
                });
            });

            it("should use default credit limit when not specified", async () => {
                const options = createStreamOptions();
                delete options.maxCredits;

                await router.stream(options).next();

                expect(mockService.getMaxOutputTokensRestrained).toHaveBeenCalledWith({
                    model: options.model,
                    maxCredits: BigInt(1000) * API_CREDITS_MULTIPLIER,
                    inputTokens: 100,
                });
            });
        });

        describe("error handling", () => {
            it("should throw CustomError when no service available", async () => {
                mockRegistry.getBestService.mockReturnValue(null);

                const options = createStreamOptions();

                await expect(async () => {
                    const stream = router.stream(options);
                    await stream.next();
                }).rejects.toThrow(CustomError);
            });

            it("should handle service errors during stream iteration", async () => {
                const erroringService = createMockService("erroring");
                erroringService.generateResponseStreaming = vi.fn().mockImplementation(async function* () {
                    yield { type: "text", content: "Starting..." };
                    throw new Error("Mid-stream error");
                });

                mockRegistry.getService.mockReturnValue(erroringService);

                const options = createStreamOptions();
                const stream = router.stream(options);

                const events = [];
                await expect(async () => {
                    for await (const event of stream) {
                        events.push(event);
                    }
                }).rejects.toThrow("Mid-stream error");

                // Should have received the first event before the error
                expect(events).toHaveLength(1);
                expect(events[0]).toMatchObject({ content: "Starting..." });
            });
        });

        describe("event ordering and integrity", () => {
            it("should maintain event order in complex stream", async () => {
                const complexEvents = [
                    { type: "text", content: "Starting " },
                    { type: "reasoning", content: "Let me think..." },
                    { type: "text", content: "analysis. " },
                    {
                        type: "function_call",
                        name: "search",
                        arguments: { query: "test" },
                        callId: "call_1",
                    },
                    { type: "text", content: "Based on search: " },
                    { type: "text", content: "conclusion" },
                    { type: "done", cost: 250 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(complexEvents),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents).toHaveLength(complexEvents.length);

                // Check that order is preserved
                expect(collectedEvents[0]).toMatchObject({ type: "message", content: "Starting " });
                expect(collectedEvents[1]).toMatchObject({ type: "reasoning", content: "Let me think..." });
                expect(collectedEvents[2]).toMatchObject({ type: "message", content: "analysis. " });
                expect(collectedEvents[3]).toMatchObject({ type: "function_call", name: "search" });
                expect(collectedEvents[4]).toMatchObject({ type: "message", content: "Based on search: " });
                expect(collectedEvents[5]).toMatchObject({ type: "message", content: "conclusion" });
                expect(collectedEvents[6]).toMatchObject({ type: "done", cost: 250 });

                // All events should have same responseId
                const responseId = collectedEvents[0].responseId;
                expect(collectedEvents.every(event => event.responseId === responseId)).toBe(true);
            });

            it("should generate unique response IDs for different streams", async () => {
                const events = [
                    { type: "text", content: "Response" },
                    { type: "done", cost: 100 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(events),
                );

                const options1 = createStreamOptions();
                const options2 = createStreamOptions();

                const stream1 = router.stream(options1);
                const stream2 = router.stream(options2);

                const event1 = await stream1.next();
                const event2 = await stream2.next();

                expect(event1.value.responseId).not.toBe(event2.value.responseId);
                expect(event1.value.responseId).toMatch(/^test-service:/);
                expect(event2.value.responseId).toMatch(/^test-service:/);
            });
        });

        describe("performance considerations", () => {
            it("should handle rapid event succession", async () => {
                const rapidEvents = Array(1000).fill(0).map((_, i) => ({
                    type: "text",
                    content: `chunk${i} `,
                }));
                rapidEvents.push({ type: "done", cost: 500 });

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(rapidEvents),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const startTime = Date.now();

                let eventCount = 0;
                for await (const event of stream) {
                    eventCount++;
                }

                const duration = Date.now() - startTime;

                expect(eventCount).toBe(1001); // 1000 text events + 1 done event
                expect(duration).toBeLessThan(1000); // Should complete quickly
            });

            it("should handle large event payloads", async () => {
                const largeContent = "x".repeat(100000); // 100KB content
                const events = [
                    { type: "text", content: largeContent },
                    { type: "done", cost: 200 },
                ];

                mockService.generateResponseStreaming = vi.fn().mockReturnValue(
                    createMockServiceStream(events),
                );

                const options = createStreamOptions();
                const stream = router.stream(options);
                const collectedEvents: StreamEvent[] = [];

                for await (const event of stream) {
                    collectedEvents.push(event);
                }

                expect(collectedEvents).toHaveLength(2);
                expect(collectedEvents[0]).toMatchObject({
                    type: "message",
                    content: largeContent,
                });
            });
        });
    });
});

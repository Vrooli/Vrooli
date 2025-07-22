import { type LlmServiceId } from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AIServiceErrorType } from "./registry.js";
import {
    AIService,
} from "./services.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
    },
}));

// Create mock instances that can be spied on
const mockAIServiceRegistry = {
    updateServiceState: vi.fn(),
};

vi.mock("./registry.js", () => ({
    AIServiceRegistry: {
        get: vi.fn(() => mockAIServiceRegistry),
    },
    AIServiceErrorType: {
        ApiError: "ApiError",
        Authentication: "Authentication",
        RateLimit: "RateLimit",
        InvalidRequest: "InvalidRequest",
    },
}));

// Create mock instances that can be spied on
const mockTokenEstimationRegistry = {
    estimateTokens: vi.fn().mockReturnValue({
        tokens: 100,
        estimationModel: "gpt-4",
        encoding: "cl100k_base",
    }),
    getEstimationInfo: vi.fn().mockReturnValue({
        estimationModel: "gpt-4",
        encoding: "cl100k_base",
    }),
};

vi.mock("./tokens.js", () => ({
    TokenEstimationRegistry: {
        get: vi.fn(() => mockTokenEstimationRegistry),
    },
}));

vi.mock("openai", () => {
    const mockStream = {
        [Symbol.asyncIterator]: vi.fn(),
    };

    return {
        default: vi.fn().mockImplementation(() => ({
            responses: {
                create: vi.fn().mockResolvedValue(mockStream),
            },
            moderations: {
                create: vi.fn(),
            },
        })),
    };
});

// Test implementation of abstract AIService for testing base functionality
class TestAIService extends AIService<"test-model"> {
    __id = "test-service" as LlmServiceId;
    featureFlags = { supportsStatefulConversations: true };
    defaultModel = "test-model" as const;

    estimateTokens = vi.fn().mockReturnValue({ tokens: 100 });
    generateContext = vi.fn().mockReturnValue([]);
    generateResponseStreaming = vi.fn();
    getContextSize = vi.fn().mockReturnValue(4096);
    getModelInfo = vi.fn().mockReturnValue({
        "test-model": {
            name: "test-model",
            contextWindow: 4096,
            maxOutputTokens: 2048,
            inputCost: 0.5,
            outputCost: 1.5,
        },
    });
    getMaxOutputTokens = vi.fn().mockReturnValue(2048);
    getMaxOutputTokensRestrained = vi.fn().mockReturnValue(1000);
    getResponseCost = vi.fn().mockReturnValue(150);
    getEstimationInfo = vi.fn().mockReturnValue({ estimationModel: "test", encoding: "test" });
    getModel = vi.fn().mockReturnValue("test-model" as const);
    getErrorType = vi.fn().mockReturnValue(AIServiceErrorType.ApiError);
    safeInputCheck = vi.fn().mockResolvedValue({ cost: 0, isSafe: true });
    getNativeToolCapabilities = vi.fn().mockReturnValue([]);
}

describe("AIService (Abstract Base Class)", () => {
    let service: TestAIService;

    beforeEach(() => {
        vi.clearAllMocks();
        service = new TestAIService();
    });

    it("should have required abstract properties", () => {
        expect(service.__id).toBeDefined();
        expect(service.featureFlags).toBeDefined();
        expect(service.defaultModel).toBeDefined();
    });

    it("should define all required abstract methods", () => {
        expect(typeof service.estimateTokens).toBe("function");
        expect(typeof service.generateContext).toBe("function");
        expect(typeof service.generateResponseStreaming).toBe("function");
        expect(typeof service.getContextSize).toBe("function");
        expect(typeof service.getModelInfo).toBe("function");
        expect(typeof service.getMaxOutputTokens).toBe("function");
        expect(typeof service.getMaxOutputTokensRestrained).toBe("function");
        expect(typeof service.getResponseCost).toBe("function");
        expect(typeof service.getEstimationInfo).toBe("function");
        expect(typeof service.getModel).toBe("function");
        expect(typeof service.getErrorType).toBe("function");
        expect(typeof service.safeInputCheck).toBe("function");
        expect(typeof service.getNativeToolCapabilities).toBe("function");
    });
});
